package main

import (
	"os"
	"testing"
)

func TestManagedWebsiteDomainFromPath(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{path: "/home/mkoerp.com.tr/public_html/core/storage", want: "mkoerp.com.tr"},
		{path: "/home/mkoerp.com.tr", want: "mkoerp.com.tr"},
		{path: "/var/www/html", want: ""},
		{path: "/home/not_a_domain/public_html", want: ""},
	}

	for _, tc := range tests {
		got := managedWebsiteDomainFromPath(tc.path)
		if got != tc.want {
			t.Fatalf("managedWebsiteDomainFromPath(%q)=%q want=%q", tc.path, got, tc.want)
		}
	}
}

func TestModeRequestsWrite(t *testing.T) {
	if !modeRequestsWrite(os.FileMode(0o775)) {
		t.Fatalf("expected 0775 to request write")
	}
	if modeRequestsWrite(os.FileMode(0o555)) {
		t.Fatalf("expected 0555 to not request write")
	}
}

func TestWebRuntimeIdentityFromPSOutput(t *testing.T) {
	output := `
root root systemd
nobody nogroup lsphp
www-data www-data php-fpm
`
	user, group := webRuntimeIdentityFromPSOutput(output)
	if user != "nobody" || group != "nogroup" {
		t.Fatalf("unexpected runtime identity user=%q group=%q", user, group)
	}
}
