package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestSyncAdminCredentialArtifactsWritesAllTargets(t *testing.T) {
	tmp := t.TempDir()
	gatewayPath := filepath.Join(tmp, "aurapanel.env")
	servicePath := filepath.Join(tmp, "aurapanel-service.env")
	passwordPath := filepath.Join(tmp, "initial_password.txt")

	t.Setenv("AURAPANEL_GATEWAY_ENV_PATH", gatewayPath)
	t.Setenv("AURAPANEL_SERVICE_ENV_PATH", servicePath)
	t.Setenv("AURAPANEL_INITIAL_PASSWORD_FILE", passwordPath)
	t.Setenv("AURAPANEL_ADMIN_EMAIL", "")
	t.Setenv("AURAPANEL_ADMIN_PASSWORD", "")
	t.Setenv("AURAPANEL_ADMIN_PASSWORD_BCRYPT", "")

	hash := mustHashPassword("M1etin123!?,")
	if err := syncAdminCredentialArtifacts("admin@server.com", "M1etin123!?,", hash); err != nil {
		t.Fatalf("syncAdminCredentialArtifacts returned error: %v", err)
	}

	for _, path := range []string{gatewayPath, servicePath} {
		content := readTrimmedFile(path)
		if !strings.Contains(content, "AURAPANEL_ADMIN_EMAIL=admin@server.com") {
			t.Fatalf("expected admin email in %s, got %q", path, content)
		}
		if !strings.Contains(content, "AURAPANEL_ADMIN_PASSWORD=M1etin123!?,") {
			t.Fatalf("expected admin password in %s, got %q", path, content)
		}
		if !strings.Contains(content, "AURAPANEL_ADMIN_PASSWORD_BCRYPT=") {
			t.Fatalf("expected admin bcrypt in %s, got %q", path, content)
		}
	}

	if got := readTrimmedFile(passwordPath); got != "M1etin123!?," {
		t.Fatalf("expected initial password file to sync, got %q", got)
	}
}

func TestLoadAdminSeedCredentialsPrefersExplicitEnvPasswordOverGatewayFileHash(t *testing.T) {
	tmp := t.TempDir()
	gatewayPath := filepath.Join(tmp, "aurapanel.env")
	fileHash := mustHashPassword("old-secret")
	content := "AURAPANEL_ADMIN_EMAIL=admin@server.com\n" +
		"AURAPANEL_ADMIN_PASSWORD=old-secret\n" +
		"AURAPANEL_ADMIN_PASSWORD_BCRYPT=" + fileHash + "\n"
	if err := os.WriteFile(gatewayPath, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write gateway env file: %v", err)
	}

	t.Setenv("AURAPANEL_GATEWAY_ENV_PATH", gatewayPath)
	t.Setenv("AURAPANEL_ADMIN_EMAIL", "")
	t.Setenv("AURAPANEL_ADMIN_PASSWORD_BCRYPT", "")
	t.Setenv("AURAPANEL_ADMIN_PASSWORD", "new-secret")

	email, hash := loadAdminSeedCredentials()
	if email != "admin@server.com" {
		t.Fatalf("expected admin email, got %q", email)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte("new-secret")); err != nil {
		t.Fatalf("expected explicit env password to win over file hash: %v", err)
	}
}
