package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type hijackableResponseWriter struct {
	header http.Header
	conn   net.Conn
	rw     *bufio.ReadWriter
}

func (w *hijackableResponseWriter) Header() http.Header {
	if w.header == nil {
		w.header = http.Header{}
	}
	return w.header
}

func (w *hijackableResponseWriter) Write(data []byte) (int, error) {
	return len(data), nil
}

func (w *hijackableResponseWriter) WriteHeader(statusCode int) {}

func (w *hijackableResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return w.conn, w.rw, nil
}

func TestStatusCapturingResponseWriterSupportsHijack(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	writer := &hijackableResponseWriter{
		conn: serverConn,
		rw:   bufio.NewReadWriter(bufio.NewReader(strings.NewReader("")), bufio.NewWriter(&bytes.Buffer{})),
	}
	wrapped := &statusCapturingResponseWriter{ResponseWriter: writer, status: http.StatusOK}

	conn, _, err := wrapped.Hijack()
	if err != nil {
		t.Fatalf("Hijack returned error: %v", err)
	}
	if conn != serverConn {
		t.Fatalf("Hijack returned unexpected conn")
	}
}

func TestHandleWebsiteAdvancedConfigGetDoesNotMutateState(t *testing.T) {
	svc := &service{
		startedAt: seedTime(),
		state:     seedState(),
		modules:   seedModuleState(),
	}
	svc.bootstrapModules()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/websites/advanced-config?domain=example.com", nil)
	rec := httptest.NewRecorder()
	svc.handleWebsiteAdvancedConfigGet(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status code: %d", rec.Code)
	}
	if got := len(svc.state.AdvancedConfig); got != 0 {
		t.Fatalf("advanced config map mutated during GET: %d entries", got)
	}

	var payload struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Data["open_basedir"] != true {
		t.Fatalf("expected default open_basedir=true, got %#v", payload.Data["open_basedir"])
	}
}

func TestHandleOLSTuningSetStagesWithoutApply(t *testing.T) {
	svc := &service{
		startedAt: seedTime(),
		state:     seedState(),
		modules:   seedModuleState(),
	}
	svc.bootstrapModules()

	body := strings.NewReader(`{"max_connections":22000,"max_ssl_connections":21000,"conn_timeout_secs":120,"keep_alive_timeout_secs":10,"max_keep_alive_requests":5000,"gzip_compression":true,"static_cache_enabled":false,"static_cache_max_age_secs":120}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ols/tuning", body)
	rec := httptest.NewRecorder()
	svc.handleOLSTuningSet(rec, req, false)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status code: %d body=%s", rec.Code, rec.Body.String())
	}
	if !svc.modules.OLSTuningPending {
		t.Fatalf("expected staged OLS config to remain pending")
	}
	if svc.modules.OLSConfig.MaxConnections != 22000 {
		t.Fatalf("expected staged max_connections to be persisted, got %d", svc.modules.OLSConfig.MaxConnections)
	}

	var payload struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Data["pending"] != true {
		t.Fatalf("expected pending=true in response, got %#v", payload.Data["pending"])
	}
}

func TestHandleVhostCreateRollsBackStateOnProvisionFailure(t *testing.T) {
	if fileExists(olsHTTPDConfigPath) {
		t.Skip("OpenLiteSpeed runtime detected on host; rollback failure path is not deterministic here")
	}

	svc := &service{
		startedAt: seedTime(),
		state:     seedState(),
		modules:   seedModuleState(),
	}
	svc.bootstrapModules()

	initialWebsites := len(svc.state.Websites)
	initialUsers := len(svc.state.Users)

	body := strings.NewReader(`{"domain":"rollback-check.example","owner":"ghostuser","php_version":"8.3"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/vhost/create", body)
	rec := httptest.NewRecorder()
	svc.handleVhostCreate(rec, req)

	if rec.Code == http.StatusOK {
		t.Skip("website provisioning succeeded on this host; rollback path was not exercised")
	}
	if len(svc.state.Websites) != initialWebsites {
		t.Fatalf("website state leaked after failed provisioning")
	}
	if len(svc.state.Users) != initialUsers {
		t.Fatalf("user state leaked after failed provisioning")
	}
	if user := svc.findUserLocked("ghostuser"); user != nil {
		t.Fatalf("ghostuser still exists after rollback")
	}
}

func TestRuntimeStatePersistsUserPasswordHashes(t *testing.T) {
	t.Setenv("AURAPANEL_STATE_FILE", filepath.Join(t.TempDir(), "panel-service-state.json"))

	svc := &service{
		startedAt: seedTime(),
		state:     seedState(),
		modules:   seedModuleState(),
	}
	svc.bootstrapModules()

	expectedHash := mustHashPassword("super-secret")
	svc.state.Users = append(svc.state.Users, PanelUser{
		ID:           99,
		Username:     "aura",
		Name:         "Aura",
		Email:        "aura@example.com",
		Role:         "user",
		Package:      "default",
		Sites:        0,
		Active:       true,
		TwoFAEnabled: false,
		PasswordHash: expectedHash,
	})

	if err := svc.saveRuntimeStateLocked(); err != nil {
		t.Fatalf("saveRuntimeStateLocked: %v", err)
	}

	restored := &service{
		startedAt: seedTime(),
		state:     seedState(),
		modules:   seedModuleState(),
	}
	if err := restored.loadRuntimeState(); err != nil {
		t.Fatalf("loadRuntimeState: %v", err)
	}

	user := restored.findUserLocked("aura")
	if user == nil {
		t.Fatalf("expected aura user after reload")
	}
	if user.PasswordHash != expectedHash {
		t.Fatalf("expected persisted password hash to survive reload")
	}
}

func TestLoadRuntimeStateRehydratesSeedAdminPasswordHash(t *testing.T) {
	t.Setenv("AURAPANEL_STATE_FILE", filepath.Join(t.TempDir(), "panel-service-state.json"))
	t.Setenv("AURAPANEL_ADMIN_EMAIL", "admin@server.com")
	t.Setenv("AURAPANEL_ADMIN_PASSWORD", "rehydrated-secret")

	state := seedState()
	state.Users[0].PasswordHash = ""

	raw, err := json.Marshal(persistedRuntimeState{
		State:   state,
		Modules: seedModuleState(),
	})
	if err != nil {
		t.Fatalf("marshal runtime state: %v", err)
	}
	if err := os.WriteFile(runtimeStatePath(), raw, 0o600); err != nil {
		t.Fatalf("write runtime state: %v", err)
	}

	svc := &service{
		startedAt: seedTime(),
		state:     seedState(),
		modules:   seedModuleState(),
	}
	if err := svc.loadRuntimeState(); err != nil {
		t.Fatalf("loadRuntimeState: %v", err)
	}

	admin := svc.findUserLocked("admin")
	if admin == nil {
		t.Fatalf("expected admin user after reload")
	}
	if strings.TrimSpace(admin.PasswordHash) == "" {
		t.Fatalf("expected admin password hash to be rehydrated")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte("rehydrated-secret")); err != nil {
		t.Fatalf("expected rehydrated admin password hash to match env password: %v", err)
	}
}

func seedTime() time.Time {
	return time.Unix(0, 0).UTC()
}
