package main

import (
	"bufio"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strings"
	"time"
)

type cloudLinuxStatus struct {
	Available bool            `json:"available"`
	Enabled   bool            `json:"enabled"`
	Distro    string          `json:"distro"`
	Version   string          `json:"version"`
	Kernel    string          `json:"kernel"`
	Features  map[string]bool `json:"features"`
	Commands  map[string]bool `json:"commands"`
	Signals   []string        `json:"signals"`
	Warnings  []string        `json:"warnings"`
	CheckedAt int64           `json:"checked_at"`
}

func detectCloudLinuxStatus() cloudLinuxStatus {
	status := cloudLinuxStatus{
		Available: false,
		Enabled:   false,
		Distro:    "unknown",
		Version:   "",
		Kernel:    "",
		Features: map[string]bool{
			"lve_manager":      false,
			"cagefs":           false,
			"alt_php_selector": false,
			"mysql_governor":   false,
		},
		Commands: map[string]bool{
			"cldetect":    commandExists("cldetect"),
			"lvectl":      commandExists("lvectl"),
			"cagefsctl":   commandExists("cagefsctl"),
			"selectorctl": commandExists("selectorctl"),
			"dbctl":       commandExists("dbctl"),
		},
		Signals:   []string{},
		Warnings:  []string{},
		CheckedAt: time.Now().UTC().Unix(),
	}

	if runtime.GOOS != "linux" {
		status.Warnings = append(status.Warnings, "CloudLinux detection is only supported on Linux hosts.")
		return status
	}

	if kernel, err := runCloudLinuxCommand("uname", "-r"); err == nil {
		status.Kernel = kernel
	}

	osRelease := parseOSReleaseFile("/etc/os-release")
	prettyName := strings.TrimSpace(osRelease["PRETTY_NAME"])
	id := strings.ToLower(strings.TrimSpace(osRelease["ID"]))
	versionID := strings.TrimSpace(osRelease["VERSION_ID"])

	if prettyName != "" {
		status.Distro = prettyName
	} else if name := strings.TrimSpace(osRelease["NAME"]); name != "" {
		status.Distro = name
	}
	status.Version = versionID

	if strings.Contains(id, "cloudlinux") {
		status.Available = true
		status.Signals = append(status.Signals, "os-release-id")
	}
	if strings.Contains(strings.ToLower(prettyName), "cloudlinux") {
		status.Available = true
		status.Signals = append(status.Signals, "os-release-pretty-name")
	}

	if !status.Available && fileExists("/etc/redhat-release") {
		if raw, err := os.ReadFile("/etc/redhat-release"); err == nil {
			if strings.Contains(strings.ToLower(string(raw)), "cloudlinux") {
				status.Available = true
				status.Signals = append(status.Signals, "redhat-release")
				if status.Distro == "unknown" {
					status.Distro = strings.TrimSpace(string(raw))
				}
			}
		}
	}

	if status.Commands["cldetect"] {
		status.Available = true
		status.Signals = append(status.Signals, "cldetect-command")
	}

	if status.Commands["lvectl"] || fileExists("/proc/lve/list") {
		status.Features["lve_manager"] = true
		status.Signals = append(status.Signals, "lve")
	}
	if status.Commands["cagefsctl"] {
		status.Features["cagefs"] = true
		status.Signals = append(status.Signals, "cagefs")
	}
	if status.Commands["selectorctl"] {
		status.Features["alt_php_selector"] = true
		status.Signals = append(status.Signals, "selectorctl")
	}
	if status.Commands["dbctl"] || commandExists("dbgovernor") {
		status.Features["mysql_governor"] = true
		status.Signals = append(status.Signals, "mysql-governor")
	}

	lveRuntimeActive := fileExists("/proc/lve/list") || serviceActive("lvestats", "lvestats.service")
	status.Enabled = status.Available && (lveRuntimeActive || status.Features["lve_manager"])

	if !status.Available {
		status.Warnings = append(status.Warnings, "CloudLinux was not detected on this node.")
	} else {
		if !status.Features["lve_manager"] {
			status.Warnings = append(status.Warnings, "LVE manager was not detected (lvectl missing).")
		}
		if !status.Features["cagefs"] {
			status.Warnings = append(status.Warnings, "CageFS command was not detected (cagefsctl missing).")
		}
	}

	status.Signals = uniqueSortedStrings(status.Signals)
	return status
}

func runCloudLinuxCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func parseOSReleaseFile(path string) map[string]string {
	values := map[string]string{}
	raw, err := os.ReadFile(path)
	if err != nil {
		return values
	}

	scanner := bufio.NewScanner(strings.NewReader(string(raw)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || !strings.Contains(line, "=") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		value = strings.Trim(value, `"`)
		if key != "" {
			values[key] = value
		}
	}

	return values
}

func uniqueSortedStrings(items []string) []string {
	set := make(map[string]struct{}, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		set[trimmed] = struct{}{}
	}
	out := make([]string, 0, len(set))
	for item := range set {
		out = append(out, item)
	}
	sort.Strings(out)
	return out
}

func buildPlatformCapabilities() map[string]interface{} {
	security := collectSecuritySnapshot()
	cloudlinux := detectCloudLinuxStatus()

	return map[string]interface{}{
		"cloudlinux": map[string]interface{}{
			"available": cloudlinux.Available,
			"enabled":   cloudlinux.Enabled,
			"phase":     "p0",
			"addon":     "cloudlinux-core",
			"features":  cloudlinux.Features,
		},
		"mail": map[string]interface{}{
			"mail_domain_available": security.MailDomainAvailable,
			"detected_stack":        security.DetectedMailStack,
		},
		"web": map[string]interface{}{
			"detected_stack": security.DetectedWebStack,
		},
	}
}
