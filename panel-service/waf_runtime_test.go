package main

import "testing"

func TestModSecurityEnabledFromContent(t *testing.T) {
	content := `
module mod_security {
    modsecurity  on
    ls_enabled              1
}
`
	enabled, err := modSecurityEnabledFromContent(content)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !enabled {
		t.Fatalf("expected modsecurity state to be enabled")
	}
}

func TestModSecurityEnabledFromContentDisabled(t *testing.T) {
	content := `
module mod_security {
    modsecurity  off
    ls_enabled              0
}
`
	enabled, err := modSecurityEnabledFromContent(content)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if enabled {
		t.Fatalf("expected modsecurity state to be disabled")
	}
}

func TestModSecurityEnabledFromContentMissingBlock(t *testing.T) {
	content := `tuning { maxConnections 10000 }`
	_, err := modSecurityEnabledFromContent(content)
	if err == nil {
		t.Fatalf("expected error when modsecurity block is missing")
	}
}
