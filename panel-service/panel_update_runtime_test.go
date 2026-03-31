package main

import "testing"

func TestParseGitAheadBehind(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		input      string
		wantAhead  int
		wantBehind int
		wantErr    bool
	}{
		{name: "tab separated", input: "0\t3", wantAhead: 0, wantBehind: 3},
		{name: "space separated", input: "2 0", wantAhead: 2, wantBehind: 0},
		{name: "invalid field count", input: "1", wantErr: true},
		{name: "invalid number", input: "x 1", wantErr: true},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ahead, behind, err := parseGitAheadBehind(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected parse error for %q", tc.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected parse error: %v", err)
			}
			if ahead != tc.wantAhead || behind != tc.wantBehind {
				t.Fatalf("expected ahead=%d behind=%d, got ahead=%d behind=%d", tc.wantAhead, tc.wantBehind, ahead, behind)
			}
		})
	}
}
