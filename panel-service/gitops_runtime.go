package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func deployRuntimeGitRepo(repoURL, branch, deployPath string) (string, error) {
	repoURL = strings.TrimSpace(repoURL)
	branch = firstNonEmpty(strings.TrimSpace(branch), "main")
	deployPath = filepath.Clean(strings.TrimSpace(deployPath))
	if repoURL == "" || deployPath == "" {
		return "", fmt.Errorf("repo_url and deploy_path are required")
	}
	if err := os.MkdirAll(filepath.Dir(deployPath), 0o755); err != nil {
		return "", err
	}
	gitDir := filepath.Join(deployPath, ".git")
	if fileExists(gitDir) {
		if _, err := commandOutputTrimmed("git", "-C", deployPath, "fetch", "origin", branch); err != nil {
			return "", err
		}
		if _, err := commandOutputTrimmed("git", "-C", deployPath, "checkout", branch); err != nil {
			return "", err
		}
		if _, err := commandOutputTrimmed("git", "-C", deployPath, "pull", "--ff-only", "origin", branch); err != nil {
			return "", err
		}
	} else {
		if fileExists(deployPath) {
			entries, err := os.ReadDir(deployPath)
			if err != nil {
				return "", err
			}
			if len(entries) > 0 {
				return "", fmt.Errorf("deploy path already exists and is not a git repository")
			}
		}
		if _, err := commandOutputTrimmed("git", "clone", "--branch", branch, "--single-branch", repoURL, deployPath); err != nil {
			return "", err
		}
	}
	commit, err := commandOutputTrimmed("git", "-C", deployPath, "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(commit), nil
}
