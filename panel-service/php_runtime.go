package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

func phpVersionPackageToken(version string) string {
	return strings.ReplaceAll(strings.TrimSpace(version), ".", "")
}

func discoverPHPVersions() []PHPVersionInfo {
	patterns := []string{"/usr/local/lsws/lsphp*/bin/lsphp"}
	items := map[string]PHPVersionInfo{}
	for _, pattern := range patterns {
		matches, _ := filepath.Glob(pattern)
		for _, match := range matches {
			versionToken := strings.TrimPrefix(filepath.Base(filepath.Dir(filepath.Dir(match))), "lsphp")
			if len(versionToken) < 2 {
				continue
			}
			version := versionToken[:1] + "." + versionToken[1:]
			items[version] = PHPVersionInfo{
				Version:   version,
				Installed: true,
				EOL:       strings.HasPrefix(version, "7.") || version == "8.0",
			}
		}
	}
	if len(items) == 0 {
		return []PHPVersionInfo{{Version: "8.3", Installed: true, EOL: false}}
	}
	versions := make([]PHPVersionInfo, 0, len(items))
	for _, item := range items {
		versions = append(versions, item)
	}
	sort.Slice(versions, func(i, j int) bool { return versions[i].Version > versions[j].Version })
	return versions
}

func detectPHPIniPath(version string) string {
	token := phpVersionPackageToken(version)
	candidates := []string{
		fmt.Sprintf("/usr/local/lsws/lsphp%s/etc/php/%s/litespeed/php.ini", token, version),
		fmt.Sprintf("/usr/local/lsws/lsphp%s/etc/php/%s/php.ini", token, version),
		fmt.Sprintf("/usr/local/lsws/lsphp%s/etc/php.ini", token),
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return candidates[0]
}

func installPHPVersion(version string) error {
	token := phpVersionPackageToken(version)
	manager := "apt-get"
	args := []string{"install", "-y"}
	if fileExists("/usr/bin/dnf") {
		manager = "dnf"
	}
	if manager == "apt-get" {
		args = append(args,
			"lsphp"+token,
			"lsphp"+token+"-common",
			"lsphp"+token+"-mysql",
			"lsphp"+token+"-pgsql",
			"lsphp"+token+"-sqlite3",
			"lsphp"+token+"-intl",
		)
	} else {
		args = append(args,
			"lsphp"+token,
			"lsphp"+token+"-common",
			"lsphp"+token+"-mysqlnd",
			"lsphp"+token+"-pgsql",
			"lsphp"+token+"-sqlite3",
			"lsphp"+token+"-intl",
		)
	}
	cmd := exec.Command(manager, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("php install failed: %s", strings.TrimSpace(string(output)))
	}
	return nil
}

func removePHPVersion(version string) error {
	token := phpVersionPackageToken(version)
	manager := "apt-get"
	args := []string{"remove", "-y", "lsphp" + token}
	if fileExists("/usr/bin/dnf") {
		manager = "dnf"
		args = []string{"remove", "-y", "lsphp" + token}
	}
	cmd := exec.Command(manager, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("php remove failed: %s", strings.TrimSpace(string(output)))
	}
	return nil
}

func restartPHPRuntime() error {
	if fileExists("/usr/local/lsws/bin/lswsctrl") {
		cmd := exec.Command("/usr/local/lsws/bin/lswsctrl", "reload")
		if output, err := cmd.CombinedOutput(); err == nil {
			_ = output
			return nil
		}
	}
	cmd := exec.Command("systemctl", "reload", "lsws")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("php runtime reload failed: %s", strings.TrimSpace(string(output)))
	}
	return nil
}
