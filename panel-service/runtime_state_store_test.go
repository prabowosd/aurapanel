package main

import "testing"

func TestRuntimeStateBackendDefaultsToFileWhenExplicitPathProvided(t *testing.T) {
	t.Setenv("AURAPANEL_STATE_BACKEND", "")
	t.Setenv("AURAPANEL_STATE_FILE", "/tmp/custom-state.json")
	if got := runtimeStateBackend(); got != "file" {
		t.Fatalf("expected file backend when explicit state file is set, got %q", got)
	}
}

func TestRuntimeStateBackendDefaultsToAutoWithoutOverrides(t *testing.T) {
	t.Setenv("AURAPANEL_STATE_BACKEND", "")
	t.Setenv("AURAPANEL_STATE_FILE", "")
	if got := runtimeStateBackend(); got != "auto" {
		t.Fatalf("expected auto backend, got %q", got)
	}
}

func TestRuntimeStateBackendRespectsMariaDBOverride(t *testing.T) {
	t.Setenv("AURAPANEL_STATE_BACKEND", "mariadb")
	t.Setenv("AURAPANEL_STATE_FILE", "/tmp/custom-state.json")
	if got := runtimeStateBackend(); got != "mariadb" {
		t.Fatalf("expected mariadb backend override, got %q", got)
	}
}

func TestIsRuntimeStateNewerPrefersVersion(t *testing.T) {
	current := persistedRuntimeState{StateVersion: 4, StateSavedAtUnix: 100}
	candidate := persistedRuntimeState{StateVersion: 5, StateSavedAtUnix: 90}
	if !isRuntimeStateNewer(candidate, current) {
		t.Fatalf("expected higher version to win")
	}
}

func TestIsRuntimeStateNewerFallsBackToTimestamp(t *testing.T) {
	current := persistedRuntimeState{StateVersion: 0, StateSavedAtUnix: 100}
	candidate := persistedRuntimeState{StateVersion: 0, StateSavedAtUnix: 101}
	if !isRuntimeStateNewer(candidate, current) {
		t.Fatalf("expected newer timestamp to win when version is unavailable")
	}
}

func TestNormalizeRuntimeStateMetadataUsesObservedTimestamp(t *testing.T) {
	payload := persistedRuntimeState{}
	payload.normalizeRuntimeStateMetadata(12345)
	if payload.StateSavedAtUnix != 12345 {
		t.Fatalf("expected observed timestamp to be injected, got %d", payload.StateSavedAtUnix)
	}
}
