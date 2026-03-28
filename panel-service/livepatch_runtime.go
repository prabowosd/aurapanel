package main

import (
	"fmt"
	"os/exec"
	"strings"
)

func refreshRuntimeLivePatch(target string) (string, error) {
	target = strings.TrimSpace(strings.ToLower(target))
	switch {
	case target == "", target == "kernel", target == "canonical-livepatch":
		if _, err := exec.LookPath("canonical-livepatch"); err == nil {
			output, runErr := commandOutputTrimmed("canonical-livepatch", "status")
			if runErr == nil {
				_, _ = commandOutputTrimmed("canonical-livepatch", "refresh")
				return output, nil
			}
			return "", runErr
		}
		if _, err := exec.LookPath("kpatch"); err == nil {
			output, runErr := commandOutputTrimmed("kpatch", "list")
			if runErr != nil {
				return "", runErr
			}
			return output, nil
		}
	}
	return "", fmt.Errorf("no supported live patch runtime found")
}
