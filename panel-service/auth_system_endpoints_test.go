package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func withPrincipal(req *http.Request, principal servicePrincipal) *http.Request {
	ctx := context.WithValue(req.Context(), servicePrincipalContextKey, principal)
	return req.WithContext(ctx)
}

func TestHandleResellerTokenLifecycle(t *testing.T) {
	t.Setenv("AURAPANEL_STATE_FILE", filepath.Join(t.TempDir(), "panel-service-state.json"))

	svc := &service{
		startedAt: seedTime(),
		state:     seedState(),
		modules:   seedModuleState(),
	}
	svc.bootstrapModules()

	admin := servicePrincipal{
		Email:    "admin@server.com",
		Role:     "admin",
		Username: "admin",
	}

	setReq := withPrincipal(
		httptest.NewRequest(http.MethodPost, "/api/v1/system/reseller-token", strings.NewReader(`{"token":"abc123TOKEN"}`)),
		admin,
	)
	setRec := httptest.NewRecorder()
	svc.handleResellerTokenSet(setRec, setReq)
	if setRec.Code != http.StatusOK {
		t.Fatalf("expected 200 on set, got %d body=%s", setRec.Code, setRec.Body.String())
	}
	if got := strings.TrimSpace(svc.state.ResellerToken); got != "abc123TOKEN" {
		t.Fatalf("expected reseller token to be saved, got %q", got)
	}

	getReq := withPrincipal(
		httptest.NewRequest(http.MethodGet, "/api/v1/system/reseller-token", nil),
		admin,
	)
	getRec := httptest.NewRecorder()
	svc.handleResellerTokenGet(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("expected 200 on get, got %d body=%s", getRec.Code, getRec.Body.String())
	}

	var getPayload struct {
		Status string `json:"status"`
		Token  string `json:"token"`
	}
	if err := json.Unmarshal(getRec.Body.Bytes(), &getPayload); err != nil {
		t.Fatalf("decode get payload: %v", err)
	}
	if getPayload.Status != "success" {
		t.Fatalf("expected success status, got %q", getPayload.Status)
	}
	if getPayload.Token != "abc123TOKEN" {
		t.Fatalf("expected token abc123TOKEN, got %q", getPayload.Token)
	}

	deleteReq := withPrincipal(
		httptest.NewRequest(http.MethodDelete, "/api/v1/system/reseller-token", nil),
		admin,
	)
	deleteRec := httptest.NewRecorder()
	svc.handleResellerTokenDelete(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusOK {
		t.Fatalf("expected 200 on delete, got %d body=%s", deleteRec.Code, deleteRec.Body.String())
	}
	if svc.state.ResellerToken != "" {
		t.Fatalf("expected reseller token to be deleted")
	}
}

func TestHandleResellerTokenRejectsNonAdmin(t *testing.T) {
	t.Setenv("AURAPANEL_STATE_FILE", filepath.Join(t.TempDir(), "panel-service-state.json"))

	svc := &service{
		startedAt: seedTime(),
		state:     seedState(),
		modules:   seedModuleState(),
	}
	svc.bootstrapModules()

	userReq := withPrincipal(
		httptest.NewRequest(http.MethodGet, "/api/v1/system/reseller-token", nil),
		servicePrincipal{Email: "user@example.com", Role: "user", Username: "user"},
	)
	userRec := httptest.NewRecorder()
	svc.handleResellerTokenGet(userRec, userReq)
	if userRec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for non-admin, got %d body=%s", userRec.Code, userRec.Body.String())
	}
}

func TestHandleAuthMeReturnsResolvedUser(t *testing.T) {
	t.Setenv("AURAPANEL_STATE_FILE", filepath.Join(t.TempDir(), "panel-service-state.json"))

	svc := &service{
		startedAt: seedTime(),
		state:     seedState(),
		modules:   seedModuleState(),
	}
	svc.bootstrapModules()

	req := withPrincipal(
		httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil),
		servicePrincipal{
			Email:    "admin@server.com",
			Role:     "admin",
			Username: "admin",
		},
	)
	rec := httptest.NewRecorder()
	svc.handleAuthMe(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Status string    `json:"status"`
		Data   PanelUser `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode auth me payload: %v", err)
	}
	if payload.Status != "success" {
		t.Fatalf("expected success status, got %q", payload.Status)
	}
	if !strings.EqualFold(payload.Data.Email, "admin@server.com") {
		t.Fatalf("expected admin email in auth me payload, got %q", payload.Data.Email)
	}
}
