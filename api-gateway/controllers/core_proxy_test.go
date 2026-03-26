package controllers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCoreProxyForwardsPathAndMethod(t *testing.T) {
	var gotPath string
	var gotMethod string

	core := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status": "success",
		})
	}))
	defer core.Close()

	t.Setenv("AURAPANEL_CORE_URL", core.URL)
	proxy, err := NewCoreProxy()
	if err != nil {
		t.Fatalf("failed to init proxy: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rec := httptest.NewRecorder()
	proxy.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if gotMethod != http.MethodGet {
		t.Fatalf("expected method GET, got %s", gotMethod)
	}
	if gotPath != "/api/v1/health" {
		t.Fatalf("expected path /api/v1/health, got %s", gotPath)
	}
}

func TestCoreProxyRejectsNonLoopbackInGatewayOnlyMode(t *testing.T) {
	t.Setenv("AURAPANEL_GATEWAY_ONLY", "1")
	t.Setenv("AURAPANEL_CORE_URL", "http://10.10.10.10:8000")

	_, err := NewCoreProxy()
	if err == nil {
		t.Fatalf("expected NewCoreProxy to reject non-loopback target in gateway-only mode")
	}
}
