package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandlePanelPluginSaveAndList(t *testing.T) {
	svc := &service{
		startedAt: seedTime(),
		state:     seedState(),
		modules:   seedModuleState(),
	}
	svc.bootstrapModules()

	saveReq := httptest.NewRequest(http.MethodPost, "/api/v1/plugins/save", strings.NewReader(`{
		"id":"mail-inspector",
		"name":"Mail Inspector",
		"version":"1.0.0",
		"entrypoint":"bin/mail-inspector",
		"hooks":["mail.lifecycle"],
		"enabled":true
	}`))
	saveRec := httptest.NewRecorder()
	svc.handlePanelPluginSave(saveRec, saveReq)
	if saveRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", saveRec.Code, saveRec.Body.String())
	}

	listRec := httptest.NewRecorder()
	svc.handlePanelPluginsList(listRec)
	if listRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", listRec.Code, listRec.Body.String())
	}

	var payload struct {
		Status string        `json:"status"`
		Data   []PanelPlugin `json:"data"`
	}
	if err := json.Unmarshal(listRec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if payload.Status != "success" {
		t.Fatalf("expected success, got %q", payload.Status)
	}
	if len(payload.Data) != 1 {
		t.Fatalf("expected exactly one plugin, got %d", len(payload.Data))
	}
	if payload.Data[0].ID != "mail-inspector" {
		t.Fatalf("unexpected plugin id: %q", payload.Data[0].ID)
	}

	toggleReq := httptest.NewRequest(http.MethodPost, "/api/v1/plugins/toggle", strings.NewReader(`{"id":"mail-inspector","enabled":false}`))
	toggleRec := httptest.NewRecorder()
	svc.handlePanelPluginToggle(toggleRec, toggleReq)
	if toggleRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", toggleRec.Code, toggleRec.Body.String())
	}
	if len(svc.modules.PanelPlugins) != 1 || svc.modules.PanelPlugins[0].Enabled {
		t.Fatalf("expected plugin to be disabled")
	}
}
