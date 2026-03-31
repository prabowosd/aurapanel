package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const defaultPanelRepoPath = "/opt/aurapanel"

type panelUpdateResult struct {
	PreviousVersion string
	CurrentVersion  string
	TargetVersion   string
	Steps           []string
	Warnings        []string
}

func panelRepoPath() string {
	return firstNonEmpty(strings.TrimSpace(os.Getenv("AURAPANEL_REPO_PATH")), defaultPanelRepoPath)
}

func resolveCurrentPanelVersion() string {
	return currentPanelVersionFromRepo(panelRepoPath())
}

func currentPanelVersionFromRepo(repoPath string) string {
	repoPath = strings.TrimSpace(repoPath)
	if repoPath == "" {
		repoPath = defaultPanelRepoPath
	}
	if value := strings.TrimSpace(os.Getenv("AURAPANEL_CURRENT_VERSION")); value != "" {
		return value
	}
	if !fileExists(filepath.Join(repoPath, ".git")) {
		return currentPanelVersion
	}
	if out, err := commandOutputTrimmed("git", "-C", repoPath, "describe", "--tags", "--always", "--dirty"); err == nil && strings.TrimSpace(out) != "" {
		return strings.TrimSpace(out)
	}
	if out, err := commandOutputTrimmed("git", "-C", repoPath, "rev-parse", "--short", "HEAD"); err == nil && strings.TrimSpace(out) != "" {
		return strings.TrimSpace(out)
	}
	return currentPanelVersion
}

func applyPanelUpdateToRelease(target string) (panelUpdateResult, error) {
	result := panelUpdateResult{
		PreviousVersion: resolveCurrentPanelVersion(),
		TargetVersion:   strings.TrimSpace(target),
		Steps:           []string{},
		Warnings:        []string{},
	}

	if runtime.GOOS != "linux" {
		return result, fmt.Errorf("panel update can only be applied on linux hosts")
	}
	if result.TargetVersion == "" {
		return result, fmt.Errorf("target release version is required")
	}

	repoPath := panelRepoPath()
	if !fileExists(filepath.Join(repoPath, ".git")) {
		return result, fmt.Errorf("repository path is not a git checkout: %s", repoPath)
	}

	if dirty, err := commandOutputTrimmed("git", "-C", repoPath, "status", "--porcelain"); err == nil && strings.TrimSpace(dirty) != "" {
		return result, fmt.Errorf("repository is dirty; commit or stash local changes before panel update")
	}

	if err := runPanelUpdateStep(&result, "Fetch latest tags", "git", "-C", repoPath, "fetch", "--tags", "origin"); err != nil {
		return result, err
	}
	if err := runPanelUpdateStep(&result, "Checkout target release", "git", "-C", repoPath, "checkout", result.TargetVersion); err != nil {
		return result, err
	}

	frontendPath := filepath.Join(repoPath, "frontend")
	if fileExists(filepath.Join(frontendPath, "package-lock.json")) {
		if err := runPanelUpdateStep(&result, "Install frontend dependencies (npm ci)", "npm", "--prefix", frontendPath, "ci"); err != nil {
			if err2 := runPanelUpdateStep(&result, "Fallback dependency install (npm install)", "npm", "--prefix", frontendPath, "install"); err2 != nil {
				return result, err2
			}
			result.Warnings = append(result.Warnings, fmt.Sprintf("npm ci failed and fallback npm install was used: %v", err))
		}
	} else {
		if err := runPanelUpdateStep(&result, "Install frontend dependencies (npm install)", "npm", "--prefix", frontendPath, "install"); err != nil {
			return result, err
		}
	}
	if err := runPanelUpdateStep(&result, "Build frontend", "npm", "--prefix", frontendPath, "run", "build"); err != nil {
		return result, err
	}

	panelServicePath := filepath.Join(repoPath, "panel-service")
	apiGatewayPath := filepath.Join(repoPath, "api-gateway")
	if err := runPanelUpdateStep(&result, "Build panel-service binary", "go", "-C", panelServicePath, "build", "-o", filepath.Join(panelServicePath, "panel-service")); err != nil {
		return result, err
	}
	if err := runPanelUpdateStep(&result, "Build api-gateway binary", "go", "-C", apiGatewayPath, "build", "-o", filepath.Join(apiGatewayPath, "apigw")); err != nil {
		return result, err
	}
	if err := runPanelUpdateStep(&result, "Restart API gateway", "systemctl", "restart", "aurapanel-api"); err != nil {
		return result, err
	}
	if err := runPanelUpdateStep(&result, "Schedule panel-service restart", "systemd-run", "--unit", "aurapanel-service-delayed-restart", "--on-active=3", "systemctl", "restart", "aurapanel-service"); err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Panel-service restart could not be scheduled automatically: %v", err))
		result.Warnings = append(result.Warnings, "Restart aurapanel-service manually to apply backend code updates.")
	} else {
		result.Warnings = append(result.Warnings, "Panel-service restart is scheduled to run in a few seconds.")
	}
	result.CurrentVersion = resolveCurrentPanelVersion()
	return result, nil
}

func runPanelUpdateStep(result *panelUpdateResult, title string, command string, args ...string) error {
	result.Steps = append(result.Steps, title)
	output, err := commandOutputTrimmed(command, args...)
	if err != nil {
		return fmt.Errorf("%s: %w", title, err)
	}
	if output != "" {
		result.Steps = append(result.Steps, output)
	}
	return nil
}
