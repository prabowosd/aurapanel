package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestIsValidDomainName(t *testing.T) {
	valid := []string{
		"example.com",
		"sub.example.com",
		"xn--bcher-kva.example",
	}
	invalid := []string{
		"",
		"localhost",
		"bad_domain.com",
		"../etc/passwd",
		"example..com",
		"-start.example",
		"end-.example",
		"bad/example.com",
	}

	for _, domain := range valid {
		if !isValidDomainName(domain) {
			t.Fatalf("expected valid domain: %q", domain)
		}
	}
	for _, domain := range invalid {
		if isValidDomainName(domain) {
			t.Fatalf("expected invalid domain: %q", domain)
		}
	}
}

func TestIsAllowedTransferHomeDir(t *testing.T) {
	cases := []struct {
		path    string
		allowed bool
	}{
		{path: "/home/customer/public_html", allowed: true},
		{path: "/home/customer", allowed: true},
		{path: "/home", allowed: false},
		{path: "/", allowed: false},
		{path: "/etc", allowed: false},
		{path: "/home/customer/../../etc", allowed: false},
	}

	for _, tc := range cases {
		if got := isAllowedTransferHomeDir(tc.path); got != tc.allowed {
			t.Fatalf("home_dir=%q expected %v got %v", tc.path, tc.allowed, got)
		}
	}
}

func TestHandleWebsiteCustomSSLGetRedactsPrivateKey(t *testing.T) {
	svc := &service{
		startedAt: seedTime(),
		state:     seedState(),
		modules:   seedModuleState(),
	}
	svc.bootstrapModules()
	svc.state.CustomSSL["example.com"] = WebsiteCustomSSL{
		CertPEM: "CERT",
		KeyPEM:  "PRIVATE_KEY",
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/websites/custom-ssl?domain=example.com", nil)
	req = req.WithContext(context.WithValue(req.Context(), servicePrincipalContextKey, servicePrincipal{
		Email:    "admin@server.com",
		Role:     "admin",
		Username: "admin",
		Name:     "Admin",
	}))
	rec := httptest.NewRecorder()
	svc.handleWebsiteCustomSSLGet(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status code: %d body=%s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Data WebsiteCustomSSL `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Data.CertPEM != "CERT" {
		t.Fatalf("expected cert to be preserved")
	}
	if payload.Data.KeyPEM != "" {
		t.Fatalf("expected private key to be redacted")
	}
}

func TestHandleBackupDestinationsGetRedactsPasswords(t *testing.T) {
	svc := &service{
		startedAt: seedTime(),
		state:     seedState(),
		modules:   seedModuleState(),
	}
	svc.bootstrapModules()
	svc.modules.BackupDestinations = []BackupDestination{
		{
			ID:         "dst1",
			Name:       "Primary",
			RemoteRepo: "s3:https://example/bucket",
			Password:   "super-secret",
			Enabled:    true,
		},
	}

	rec := httptest.NewRecorder()
	svc.handleBackupDestinationsGet(rec)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status code: %d body=%s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Data []BackupDestination `json:"data"`
	}
	if err := json.NewDecoder(strings.NewReader(rec.Body.String())).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload.Data) != 1 {
		t.Fatalf("expected 1 destination, got %d", len(payload.Data))
	}
	if payload.Data[0].Password != "" {
		t.Fatalf("expected password to be redacted in response")
	}
	if svc.modules.BackupDestinations[0].Password != "super-secret" {
		t.Fatalf("expected in-memory password to remain unchanged")
	}
}
