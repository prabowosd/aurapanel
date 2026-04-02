package main

import (
	"strings"
	"testing"
)

func TestNormalizeSSHConfigContentReplacesAndDeduplicatesDirectives(t *testing.T) {
	input := strings.Join([]string{
		"# Example",
		"Port 22",
		"PermitRootLogin yes",
		"Port 2222",
		"PermitRootLogin no",
		"UsePAM yes",
	}, "\n")

	out := normalizeSSHConfigContent(input, 44570, "prohibit-password")

	if strings.Count(out, "Port ") != 1 {
		t.Fatalf("expected exactly one Port directive, got: %q", out)
	}
	if !strings.Contains(out, "Port 44570") {
		t.Fatalf("expected updated port directive, got: %q", out)
	}
	if strings.Count(out, "PermitRootLogin ") != 1 {
		t.Fatalf("expected exactly one PermitRootLogin directive, got: %q", out)
	}
	if !strings.Contains(out, "PermitRootLogin prohibit-password") {
		t.Fatalf("expected updated PermitRootLogin directive, got: %q", out)
	}
}

func TestNormalizeSSHConfigContentAppendsMissingDirectives(t *testing.T) {
	input := "# baseline\nUsePAM yes\n"
	out := normalizeSSHConfigContent(input, 44570, "no")

	if !strings.Contains(out, "Port 44570") {
		t.Fatalf("expected port directive to be appended, got: %q", out)
	}
	if !strings.Contains(out, "PermitRootLogin no") {
		t.Fatalf("expected PermitRootLogin directive to be appended, got: %q", out)
	}
}
