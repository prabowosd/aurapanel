package main

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestHandleUsersCreateRejectsDuplicateEmail(t *testing.T) {
	t.Setenv("AURAPANEL_STATE_FILE", filepath.Join(t.TempDir(), "panel-service-state.json"))

	svc := &service{
		startedAt: seedTime(),
		state:     seedState(),
		modules:   seedModuleState(),
	}
	svc.bootstrapModules()
	svc.state.Users = append(svc.state.Users, PanelUser{
		ID:           2,
		Username:     "alice",
		Name:         "Alice",
		Email:        "alice@example.com",
		Role:         "user",
		Package:      "default",
		Active:       true,
		PasswordHash: mustHashPassword("alicepass"),
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/create", strings.NewReader(`{"username":"bob","email":"alice@example.com","password":"Strong!123","role":"user","package":"default"}`))
	rec := httptest.NewRecorder()

	svc.handleUsersCreate(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d body=%s", rec.Code, rec.Body.String())
	}
	if user := svc.findUserLocked("bob"); user != nil {
		t.Fatalf("unexpected user creation with duplicate email")
	}
}

func TestHandleUsersCreateRejectsNonSanitizableUsername(t *testing.T) {
	t.Setenv("AURAPANEL_STATE_FILE", filepath.Join(t.TempDir(), "panel-service-state.json"))

	svc := &service{
		startedAt: seedTime(),
		state:     seedState(),
		modules:   seedModuleState(),
	}
	svc.bootstrapModules()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/create", strings.NewReader(`{"username":"***","email":"user@example.com","password":"Strong!123","role":"user","package":"default"}`))
	rec := httptest.NewRecorder()

	svc.handleUsersCreate(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}
