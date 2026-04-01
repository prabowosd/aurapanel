package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHandleMailAuthBootstrapStoresRecord(t *testing.T) {
	svc := &service{
		startedAt: seedTime(),
		state:     seedState(),
		modules:   seedModuleState(),
	}
	svc.bootstrapModules()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/mail/auth/bootstrap", strings.NewReader(`{"domain":"example.com","policy":"reject"}`))
	rec := httptest.NewRecorder()
	svc.handleMailAuthBootstrap(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	record, ok := svc.modules.MailAuthRecords["example.com"]
	if !ok {
		t.Fatalf("expected auth record to be stored")
	}
	if record.Policy != "reject" {
		t.Fatalf("expected policy reject, got %q", record.Policy)
	}
	if !strings.Contains(record.SPFValue, "v=spf1") {
		t.Fatalf("expected SPF value to be generated, got %q", record.SPFValue)
	}

	deliverabilityReq := httptest.NewRequest(http.MethodGet, "/api/v1/mail/deliverability?domain=example.com", nil)
	deliverabilityRec := httptest.NewRecorder()
	svc.handleMailDeliverability(deliverabilityRec, deliverabilityReq)
	if deliverabilityRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", deliverabilityRec.Code, deliverabilityRec.Body.String())
	}

	var payload struct {
		Status string `json:"status"`
		Data   struct {
			Checks map[string]bool `json:"checks"`
		} `json:"data"`
	}
	if err := json.Unmarshal(deliverabilityRec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal deliverability response: %v", err)
	}
	if payload.Status != "success" {
		t.Fatalf("expected success, got %q", payload.Status)
	}
	if !payload.Data.Checks["spf_dmarc"] {
		t.Fatalf("expected spf_dmarc check to be true")
	}
}

func TestHandleMailWebmailOpsCleanupRemovesExpiredTokens(t *testing.T) {
	svc := &service{
		startedAt: seedTime(),
		state:     seedState(),
		modules:   seedModuleState(),
	}
	svc.bootstrapModules()

	svc.modules.WebmailTokens["expired-token"] = WebmailToken{
		Token:     "expired-token",
		Address:   "user@example.com",
		ExpiresAt: time.Now().UTC().Add(-1 * time.Hour),
	}
	svc.modules.WebmailTokens["active-token"] = WebmailToken{
		Token:     "active-token",
		Address:   "user@example.com",
		ExpiresAt: time.Now().UTC().Add(1 * time.Hour),
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/mail/webmail/ops/cleanup", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()
	svc.handleMailWebmailOpsCleanup(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	if _, ok := svc.modules.WebmailTokens["expired-token"]; ok {
		t.Fatalf("expired token should be removed")
	}
	if _, ok := svc.modules.WebmailTokens["active-token"]; !ok {
		t.Fatalf("active token should remain")
	}
}
