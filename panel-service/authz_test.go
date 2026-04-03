package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleVhostListFiltersByPrincipalOwnership(t *testing.T) {
	svc := &service{
		startedAt: seedTime(),
		state:     seedState(),
		modules:   seedModuleState(),
	}
	svc.bootstrapModules()
	svc.state.Users = append(svc.state.Users, PanelUser{
		ID:       10,
		Username: "user1",
		Email:    "user1@example.com",
		Name:     "User One",
		Role:     "user",
		Active:   true,
	})
	svc.state.Websites = []Website{
		{Domain: "owned.example.com", Owner: "user1", User: "user1", Email: "webmaster@owned.example.com", Status: "active"},
		{Domain: "other.example.com", Owner: "user2", User: "user2", Email: "webmaster@other.example.com", Status: "active"},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/vhost/list", nil)
	req = req.WithContext(context.WithValue(req.Context(), servicePrincipalContextKey, servicePrincipal{
		Email:    "user1@example.com",
		Role:     "user",
		Username: "user1",
		Name:     "User One",
	}))
	rec := httptest.NewRecorder()
	svc.handleVhostList(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var payload struct {
		Data []Website `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload.Data) != 1 || payload.Data[0].Domain != "owned.example.com" {
		t.Fatalf("expected only owned domain, got %+v", payload.Data)
	}
}

func TestRequireDomainAccessBlocksNonOwner(t *testing.T) {
	svc := &service{
		startedAt: seedTime(),
		state:     seedState(),
		modules:   seedModuleState(),
	}
	svc.bootstrapModules()
	svc.state.Websites = []Website{
		{Domain: "owner.example.com", Owner: "owner", User: "owner", Email: "owner@example.com", Status: "active"},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/monitor/logs/site?domain=owner.example.com", nil)
	req = req.WithContext(context.WithValue(req.Context(), servicePrincipalContextKey, servicePrincipal{
		Email:    "intruder@example.com",
		Role:     "user",
		Username: "intruder",
		Name:     "Intruder",
	}))
	rec := httptest.NewRecorder()
	svc.handleSiteLogs(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for non-owner access, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestNonAdminRoutePolicyBlocksAdminOnlyRoutes(t *testing.T) {
	svc := &service{
		startedAt: seedTime(),
		state:     seedState(),
		modules:   seedModuleState(),
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/list", nil)
	req = req.WithContext(context.WithValue(req.Context(), servicePrincipalContextKey, servicePrincipal{
		Email:    "user@example.com",
		Role:     "user",
		Username: "user",
		Name:     "User",
	}))
	rec := httptest.NewRecorder()

	if ok := svc.nonAdminRoutePolicy(rec, req); ok {
		t.Fatalf("expected policy to block admin-only route")
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestNonAdminRoutePolicyAllowsProvisionForMatchingOwner(t *testing.T) {
	svc := &service{
		startedAt: seedTime(),
		state:     seedState(),
		modules:   seedModuleState(),
	}
	body := `{"domain":"newtenant.example.com","owner":"user","user":"user"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/vhost/create", strings.NewReader(body))
	req = req.WithContext(context.WithValue(req.Context(), servicePrincipalContextKey, servicePrincipal{
		Email:    "user@example.com",
		Role:     "user",
		Username: "user",
		Name:     "User",
	}))
	rec := httptest.NewRecorder()

	if ok := svc.nonAdminRoutePolicy(rec, req); !ok {
		t.Fatalf("expected policy to allow self-owned provisioning, got status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestNonAdminRoutePolicyBlocksAIToolsRoutes(t *testing.T) {
	svc := &service{
		startedAt: seedTime(),
		state:     seedState(),
		modules:   seedModuleState(),
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/ai/tools/status", nil)
	req = req.WithContext(context.WithValue(req.Context(), servicePrincipalContextKey, servicePrincipal{
		Email:    "user@example.com",
		Role:     "user",
		Username: "user",
		Name:     "User",
	}))
	rec := httptest.NewRecorder()

	if ok := svc.nonAdminRoutePolicy(rec, req); ok {
		t.Fatalf("expected policy to block AI tools route")
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestServiceAuthMiddlewareRejectsMissingProxyTokenInProduction(t *testing.T) {
	t.Setenv("AURAPANEL_DEV_SIMULATION", "")
	t.Setenv("AURAPANEL_INTERNAL_PROXY_TOKEN", "0123456789abcdef0123456789abcdef")

	protected := serviceAuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/status/metrics", nil)
	rec := httptest.NewRecorder()
	protected.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestServiceAuthMiddlewareAcceptsValidInternalHeaders(t *testing.T) {
	t.Setenv("AURAPANEL_DEV_SIMULATION", "")
	t.Setenv("AURAPANEL_INTERNAL_PROXY_TOKEN", "0123456789abcdef0123456789abcdef")

	protected := serviceAuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := principalFromContext(r.Context()); !ok {
			t.Fatalf("expected principal in context")
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/status/metrics", nil)
	req.Header.Set("X-Aura-Proxy-Token", "0123456789abcdef0123456789abcdef")
	req.Header.Set("X-Aura-Auth-Email", "user@example.com")
	req.Header.Set("X-Aura-Auth-Role", "user")
	req.Header.Set("X-Aura-Auth-Name", "User")
	req.Header.Set("X-Aura-Auth-Username", "user")
	rec := httptest.NewRecorder()
	protected.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d body=%s", rec.Code, rec.Body.String())
	}
}
