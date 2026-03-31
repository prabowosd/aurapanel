package main

import "testing"

func TestShouldMarkUpdateAvailable(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name           string
		currentVersion string
		latestVersion  string
		latestTag      string
		expected       bool
	}{
		{
			name:           "same tag",
			currentVersion: "v1.0.0",
			latestVersion:  "v1.0.0",
			latestTag:      "v1.0.0",
			expected:       false,
		},
		{
			name:           "current git describe from same release",
			currentVersion: "v1.0.0-45-g086a5e0",
			latestVersion:  "v1.0.0",
			latestTag:      "v1.0.0",
			expected:       false,
		},
		{
			name:           "latest newer than current",
			currentVersion: "v1.0.1",
			latestVersion:  "v1.0.2",
			latestTag:      "v1.0.2",
			expected:       true,
		},
		{
			name:           "current newer than latest",
			currentVersion: "v1.1.0",
			latestVersion:  "v1.0.9",
			latestTag:      "v1.0.9",
			expected:       false,
		},
		{
			name:           "missing current version",
			currentVersion: "",
			latestVersion:  "v1.0.0",
			latestTag:      "v1.0.0",
			expected:       true,
		},
		{
			name:           "missing latest version",
			currentVersion: "Aura Panel V1",
			latestVersion:  "",
			latestTag:      "",
			expected:       false,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := shouldMarkUpdateAvailable(tc.currentVersion, tc.latestVersion, tc.latestTag)
			if got != tc.expected {
				t.Fatalf("expected %v, got %v", tc.expected, got)
			}
		})
	}
}
