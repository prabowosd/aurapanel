package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func redisInstanceName(domain string) string {
	return "aurapanel-redis-" + strings.ReplaceAll(normalizeDomain(domain), ".", "-")
}

func redisInstancePort(domain string) int {
	base := 6390
	hash := 0
	for _, r := range normalizeDomain(domain) {
		hash += int(r)
	}
	return base + (hash % 200)
}

func redisInstanceConfigPath(domain string) string {
	return filepath.Join("/etc/redis", redisInstanceName(domain)+".conf")
}

func redisInstanceDataDir(domain string) string {
	return filepath.Join("/var/lib/redis", redisInstanceName(domain))
}

func redisInstanceUnitPath(domain string) string {
	return filepath.Join("/etc/systemd/system", redisInstanceName(domain)+".service")
}

func createRuntimeRedisIsolation(domain string, maxMemoryMB int) (RedisIsolation, error) {
	domain = normalizeDomain(domain)
	if domain == "" {
		return RedisIsolation{}, fmt.Errorf("domain is required")
	}
	if maxMemoryMB <= 0 {
		maxMemoryMB = 128
	}
	configPath := redisInstanceConfigPath(domain)
	dataDir := redisInstanceDataDir(domain)
	if err := os.MkdirAll(dataDir, 0o750); err != nil {
		return RedisIsolation{}, err
	}
	port := redisInstancePort(domain)
	config := strings.Join([]string{
		fmt.Sprintf("port %d", port),
		"bind 127.0.0.1",
		"daemonize no",
		"supervised systemd",
		fmt.Sprintf("dir %s", dataDir),
		fmt.Sprintf("dbfilename %s.rdb", strings.ReplaceAll(domain, ".", "_")),
		fmt.Sprintf("maxmemory %dmb", maxMemoryMB),
		"maxmemory-policy allkeys-lru",
		"appendonly yes",
		"protected-mode yes",
		"",
	}, "\n")
	if err := os.WriteFile(configPath, []byte(config), 0o640); err != nil {
		return RedisIsolation{}, err
	}
	unitContent := strings.Join([]string{
		"[Unit]",
		fmt.Sprintf("Description=AuraPanel Redis Instance for %s", domain),
		"After=network.target",
		"",
		"[Service]",
		"Type=notify",
		"ExecStart=/usr/bin/redis-server " + configPath + " --supervised systemd --daemonize no",
		"ExecStop=/usr/bin/redis-cli -p " + strconv.Itoa(port) + " shutdown",
		"Restart=always",
		"User=redis",
		"Group=redis",
		"",
		"[Install]",
		"WantedBy=multi-user.target",
		"",
	}, "\n")
	if err := os.WriteFile(redisInstanceUnitPath(domain), []byte(unitContent), 0o644); err != nil {
		return RedisIsolation{}, err
	}
	if _, err := commandOutputTrimmed("chown", "-R", "redis:redis", dataDir); err != nil {
		_ = err
	}
	if _, err := commandOutputTrimmed("systemctl", "daemon-reload"); err != nil {
		return RedisIsolation{}, err
	}
	unit := redisInstanceName(domain)
	if _, err := commandOutputTrimmed("systemctl", "enable", "--now", unit); err != nil {
		return RedisIsolation{}, err
	}
	return RedisIsolation{
		Domain:       domain,
		Unit:         unit,
		Port:         port,
		MaxMemoryMB:  maxMemoryMB,
		ConfigPath:   configPath,
		DataDir:      dataDir,
		BindAddress:  "127.0.0.1",
		Runtime:      "host",
		Provisioning: "systemd",
	}, nil
}
