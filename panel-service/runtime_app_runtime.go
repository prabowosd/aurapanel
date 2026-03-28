package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func runtimeAppUnitName(appName string) string {
	return "aurapanel-app-" + sanitizeName(appName)
}

func runtimeAppServicePath(appName string) string {
	return filepath.Join("/etc/systemd/system", runtimeAppUnitName(appName)+".service")
}

func runtimeAppWorkDir(dir string) (string, error) {
	resolved, err := resolveManagedPath(firstNonEmpty(strings.TrimSpace(dir), "/home"))
	if err != nil {
		return "", err
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("runtime directory is not a directory")
	}
	return resolved, nil
}

func installRuntimeNodeDeps(dir string) error {
	workDir, err := runtimeAppWorkDir(dir)
	if err != nil {
		return err
	}
	lockPath := filepath.Join(workDir, "package-lock.json")
	if fileExists(lockPath) {
		_, err = commandOutputTrimmed("npm", "--prefix", workDir, "ci")
		return err
	}
	_, err = commandOutputTrimmed("npm", "--prefix", workDir, "install")
	return err
}

func createRuntimePythonVenv(dir string) error {
	workDir, err := runtimeAppWorkDir(dir)
	if err != nil {
		return err
	}
	venvDir := filepath.Join(workDir, "venv")
	if fileExists(filepath.Join(venvDir, "bin", "python")) {
		return nil
	}
	_, err = commandOutputTrimmed("python3", "-m", "venv", venvDir)
	return err
}

func installRuntimePythonRequirements(dir string) error {
	workDir, err := runtimeAppWorkDir(dir)
	if err != nil {
		return err
	}
	reqFile := filepath.Join(workDir, "requirements.txt")
	if !fileExists(reqFile) {
		return fmt.Errorf("requirements.txt not found")
	}
	if err := createRuntimePythonVenv(workDir); err != nil {
		return err
	}
	pipBin := filepath.Join(workDir, "venv", "bin", "pip")
	_, err = commandOutputTrimmed(pipBin, "install", "-r", reqFile)
	return err
}

func writeRuntimeAppService(appName, workDir, command string) error {
	unitPath := runtimeAppServicePath(appName)
	content := strings.Join([]string{
		"[Unit]",
		fmt.Sprintf("Description=AuraPanel Runtime App %s", appName),
		"After=network-online.target",
		"Wants=network-online.target",
		"",
		"[Service]",
		"Type=simple",
		fmt.Sprintf("WorkingDirectory=%s", workDir),
		"Environment=PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		fmt.Sprintf("ExecStart=/bin/bash -lc \"%s\"", escapeSystemdCommand(command)),
		"Restart=always",
		"RestartSec=3",
		"",
		"[Install]",
		"WantedBy=multi-user.target",
		"",
	}, "\n")
	if err := os.WriteFile(unitPath, []byte(content), 0o644); err != nil {
		return err
	}
	if _, err := commandOutputTrimmed("systemctl", "daemon-reload"); err != nil {
		return err
	}
	return nil
}

func escapeSystemdCommand(command string) string {
	command = strings.ReplaceAll(command, `\`, `\\`)
	command = strings.ReplaceAll(command, `"`, `\"`)
	return command
}

func startRuntimeNodeApp(dir, appName, startScript string) (RuntimeApp, error) {
	workDir, err := runtimeAppWorkDir(dir)
	if err != nil {
		return RuntimeApp{}, err
	}
	appName = firstNonEmpty(strings.TrimSpace(appName), filepath.Base(workDir))
	startScript = firstNonEmpty(strings.TrimSpace(startScript), "npm start")
	if err := writeRuntimeAppService(appName, workDir, startScript); err != nil {
		return RuntimeApp{}, err
	}
	unit := runtimeAppUnitName(appName)
	if _, err := commandOutputTrimmed("systemctl", "enable", "--now", unit); err != nil {
		return RuntimeApp{}, err
	}
	return RuntimeApp{Runtime: "nodejs", Dir: workDir, AppName: appName, Status: "running"}, nil
}

func startRuntimePythonApp(dir, appName, wsgiModule string, port int) (RuntimeApp, error) {
	workDir, err := runtimeAppWorkDir(dir)
	if err != nil {
		return RuntimeApp{}, err
	}
	if err := createRuntimePythonVenv(workDir); err != nil {
		return RuntimeApp{}, err
	}
	appName = firstNonEmpty(strings.TrimSpace(appName), filepath.Base(workDir))
	wsgiModule = firstNonEmpty(strings.TrimSpace(wsgiModule), "app:app")
	if port <= 0 {
		port = 8001
	}
	gunicornPath := filepath.Join(workDir, "venv", "bin", "gunicorn")
	if !fileExists(gunicornPath) {
		if _, err := commandOutputTrimmed(filepath.Join(workDir, "venv", "bin", "pip"), "install", "gunicorn"); err != nil {
			return RuntimeApp{}, err
		}
	}
	command := fmt.Sprintf("%s --bind 127.0.0.1:%d %s", gunicornPath, port, wsgiModule)
	if err := writeRuntimeAppService(appName, workDir, command); err != nil {
		return RuntimeApp{}, err
	}
	unit := runtimeAppUnitName(appName)
	if _, err := commandOutputTrimmed("systemctl", "enable", "--now", unit); err != nil {
		return RuntimeApp{}, err
	}
	return RuntimeApp{Runtime: "python", Dir: workDir, AppName: appName, Status: "running"}, nil
}

func stopRuntimeApp(appName string) error {
	unit := runtimeAppUnitName(appName)
	_, err := commandOutputTrimmed("systemctl", "stop", unit)
	return err
}

func runtimeAppsFromSystemd(existing []RuntimeApp) []RuntimeApp {
	items := []RuntimeApp{}
	meta := map[string]RuntimeApp{}
	for _, item := range existing {
		meta[item.AppName] = item
	}
	unitPaths, _ := filepath.Glob("/etc/systemd/system/aurapanel-app-*.service")
	for _, unitPath := range unitPaths {
		appName := strings.TrimSuffix(strings.TrimPrefix(filepath.Base(unitPath), "aurapanel-app-"), ".service")
		status := "stopped"
		if serviceActive(runtimeAppUnitName(appName)) {
			status = "running"
		}
		item := RuntimeApp{
			AppName: appName,
			Runtime: "runtime",
			Status:  status,
		}
		if stored, ok := meta[appName]; ok {
			if stored.Dir != "" {
				item.Dir = stored.Dir
			}
			if stored.Runtime != "" {
				item.Runtime = stored.Runtime
			}
		}
		items = append(items, item)
	}
	return items
}
