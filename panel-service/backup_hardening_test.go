package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveSiteBackupTargetDirRespectsAllowedRoots(t *testing.T) {
	tempRoot := t.TempDir()
	t.Setenv("AURAPANEL_SITE_BACKUP_ALLOWED_ROOTS", tempRoot)

	inside := filepath.Join(tempRoot, "snapshots")
	resolved, err := resolveSiteBackupTargetDir("example.com", inside)
	if err != nil {
		t.Fatalf("expected inside path to be allowed: %v", err)
	}
	if resolved != filepath.Clean(inside) {
		t.Fatalf("unexpected resolved path: %s", resolved)
	}

	outside := filepath.Join(filepath.Dir(tempRoot), "outside-root")
	if _, err := resolveSiteBackupTargetDir("example.com", outside); err == nil {
		t.Fatalf("expected outside path to be rejected")
	}
}

func TestEnforceBackupRetentionLockedPrunesOldSnapshots(t *testing.T) {
	tempRoot := t.TempDir()
	t.Setenv("AURAPANEL_SITE_BACKUP_ALLOWED_ROOTS", tempRoot)

	oldPath := filepath.Join(tempRoot, "old.tar.gz")
	midPath := filepath.Join(tempRoot, "mid.tar.gz")
	newPath := filepath.Join(tempRoot, "new.tar.gz")
	otherPath := filepath.Join(tempRoot, "other.tar.gz")

	for _, path := range []string{oldPath, midPath, newPath, otherPath} {
		if err := os.WriteFile(path, []byte("backup"), 0o644); err != nil {
			t.Fatalf("seed backup file %s: %v", path, err)
		}
	}

	svc := &service{
		startedAt: seedTime(),
		state:     seedState(),
		modules:   seedModuleState(),
	}
	svc.bootstrapModules()
	svc.modules.BackupSnapshots = []BackupSnapshot{
		{ID: "new", Domain: "example.com", BackupPath: newPath, CreatedAt: 300},
		{ID: "mid", Domain: "example.com", BackupPath: midPath, CreatedAt: 200},
		{ID: "old", Domain: "example.com", BackupPath: oldPath, CreatedAt: 100},
		{ID: "other", Domain: "other.com", BackupPath: otherPath, CreatedAt: 50},
	}

	removed, err := svc.enforceBackupRetentionLocked("example.com", 2)
	if err != nil {
		t.Fatalf("unexpected retention error: %v", err)
	}
	if removed != 1 {
		t.Fatalf("expected 1 pruned snapshot, got %d", removed)
	}
	if len(svc.modules.BackupSnapshots) != 3 {
		t.Fatalf("expected 3 snapshots after retention, got %d", len(svc.modules.BackupSnapshots))
	}
	if _, statErr := os.Stat(oldPath); !os.IsNotExist(statErr) {
		t.Fatalf("expected old snapshot file to be removed")
	}
	if _, statErr := os.Stat(otherPath); statErr != nil {
		t.Fatalf("expected unrelated domain snapshot to remain: %v", statErr)
	}
}
