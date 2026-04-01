package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHandleBackupSLAReportIncludesDomainSummary(t *testing.T) {
	svc := &service{
		startedAt: seedTime(),
		state:     seedState(),
		modules:   seedModuleState(),
	}
	svc.bootstrapModules()

	now := time.Now().UTC().UnixMilli()
	svc.modules.BackupSnapshots = []BackupSnapshot{
		{
			ID:            "snap-1",
			ShortID:       "s1",
			Domain:        "example.com",
			CreatedAt:     now,
			Time:          time.UnixMilli(now).UTC().Format(time.RFC3339),
			RetentionKeep: 14,
		},
	}
	svc.modules.BackupRestoreDrills = []BackupRestoreDrill{
		{
			ID:        "drill-1",
			Domain:    "example.com",
			Status:    "success",
			CheckedAt: now,
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/backup/sla/report", nil)
	rec := httptest.NewRecorder()

	svc.handleBackupSLAReport(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Status string `json:"status"`
		Data   struct {
			Domains []struct {
				Domain string `json:"domain"`
				Status string `json:"status"`
				Score  int    `json:"score"`
			} `json:"domains"`
			Summary map[string]interface{} `json:"summary"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if payload.Status != "success" {
		t.Fatalf("expected success, got %q", payload.Status)
	}
	if len(payload.Data.Domains) == 0 {
		t.Fatalf("expected at least one domain in SLA report")
	}
	if payload.Data.Domains[0].Domain != "example.com" {
		t.Fatalf("unexpected domain in report: %q", payload.Data.Domains[0].Domain)
	}
	if payload.Data.Domains[0].Score <= 0 {
		t.Fatalf("expected positive score, got %d", payload.Data.Domains[0].Score)
	}
}
