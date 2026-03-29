package controllers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aurapanel/api-gateway/middleware"
	"github.com/golang-jwt/jwt/v5"
)

func TestServiceProxyForwardsPathAndMethod(t *testing.T) {
	var gotPath string
	var gotMethod string

	service := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status": "success",
		})
	}))
	defer service.Close()

	t.Setenv("AURAPANEL_SERVICE_URL", service.URL)
	proxy, err := NewServiceProxy()
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

func TestServiceProxyRejectsNonLoopbackInGatewayOnlyMode(t *testing.T) {
	t.Setenv("AURAPANEL_GATEWAY_ONLY", "1")
	t.Setenv("AURAPANEL_SERVICE_URL", "http://10.10.10.10:8081")

	_, err := NewServiceProxy()
	if err == nil {
		t.Fatalf("expected NewServiceProxy to reject non-loopback target in gateway-only mode")
	}
}

func TestServiceProxyForwardsInternalHeadersAndSanitizesClientInjectedHeaders(t *testing.T) {
	var gotProxyToken string
	var gotAuthEmail string
	var gotAuthRole string

	service := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotProxyToken = r.Header.Get("X-Aura-Proxy-Token")
		gotAuthEmail = r.Header.Get("X-Aura-Auth-Email")
		gotAuthRole = r.Header.Get("X-Aura-Auth-Role")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer service.Close()

	t.Setenv("AURAPANEL_SERVICE_URL", service.URL)
	t.Setenv("AURAPANEL_INTERNAL_PROXY_TOKEN", "internal-token")
	t.Setenv("AURAPANEL_JWT_SECRET", "0123456789abcdef0123456789abcdef")
	t.Setenv("AURAPANEL_JWT_ISSUER", "aurapanel-gateway")
	t.Setenv("AURAPANEL_JWT_AUDIENCE", "aurapanel-ui")
	proxy, err := NewServiceProxy()
	if err != nil {
		t.Fatalf("failed to init proxy: %v", err)
	}
	protected := middleware.AuthMiddleware(proxy)

	now := time.Now().UTC()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"email":    "user@example.com",
		"name":     "User",
		"role":     "user",
		"username": "user",
		"iss":      middleware.JwtIssuer(),
		"aud":      middleware.JwtAudience(),
		"iat":      now.Unix(),
		"nbf":      now.Add(-1 * time.Minute).Unix(),
		"exp":      now.Add(1 * time.Hour).Unix(),
	})
	tokenValue, err := token.SignedString([]byte(middleware.JwtSecret()))
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/status/metrics", nil)
	req.Header.Set("Authorization", "Bearer "+tokenValue)
	req.Header.Set("X-Aura-Auth-Email", "attacker@example.com")
	req.Header.Set("X-Aura-Auth-Role", "admin")
	rec := httptest.NewRecorder()

	protected.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
	if gotProxyToken != "internal-token" {
		t.Fatalf("expected proxy token to be forwarded")
	}
	if gotAuthEmail != "user@example.com" || gotAuthRole != "user" {
		t.Fatalf("expected auth headers from middleware context, got email=%q role=%q", gotAuthEmail, gotAuthRole)
	}
}
