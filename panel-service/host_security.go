package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

func collectEBPFStatusLines() []string {
	lines := []string{
		fmt.Sprintf("Collected at %s", time.Now().UTC().Format(time.RFC3339)),
	}
	if fileExists("/sys/fs/bpf") {
		lines = append(lines, "bpffs mount detected at /sys/fs/bpf")
	} else {
		lines = append(lines, "bpffs mount not detected")
	}
	if commandExists("bpftool") {
		cmd := exec.Command("bpftool", "prog", "show")
		output, err := cmd.CombinedOutput()
		if err == nil {
			count := 0
			for _, line := range strings.Split(string(output), "\n") {
				if strings.Contains(line, ":") && strings.Contains(line, "name") {
					count++
				}
			}
			lines = append(lines, fmt.Sprintf("bpftool detected %d loaded eBPF programs", count))
		} else {
			lines = append(lines, "bpftool present but program listing failed")
		}
	} else {
		lines = append(lines, "bpftool binary not installed")
	}
	return lines
}

func applySystemHardeningProfile(stack string) ([]string, error) {
	profile := []string{
		"net.ipv4.conf.all.rp_filter = 1",
		"net.ipv4.conf.default.rp_filter = 1",
		"net.ipv4.tcp_syncookies = 1",
		"net.ipv4.conf.all.accept_source_route = 0",
		"net.ipv4.conf.default.accept_source_route = 0",
		"net.ipv4.conf.all.accept_redirects = 0",
		"net.ipv4.conf.default.accept_redirects = 0",
		"net.ipv4.conf.all.send_redirects = 0",
		"net.ipv4.conf.default.send_redirects = 0",
	}
	switch strings.ToLower(strings.TrimSpace(stack)) {
	case "wordpress":
		profile = append(profile, "fs.protected_symlinks = 1", "fs.protected_hardlinks = 1")
	case "laravel":
		profile = append(profile, "vm.swappiness = 10")
	default:
		profile = append(profile, "kernel.kptr_restrict = 2")
	}
	path := envOr("AURAPANEL_HARDENING_SYSCTL_FILE", "/etc/sysctl.d/99-aurapanel-hardening.conf")
	content := strings.Join(profile, "\n") + "\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return nil, err
	}
	cmd := exec.Command("sysctl", "--system")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("sysctl apply failed: %s", strings.TrimSpace(string(output)))
	}
	return profile, nil
}
