package main

import (
	"net"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const defaultDBToolsRuntimeAllowlistFile = "/etc/aurapanel/db-tools/runtime-allowlist.txt"

type dbToolSessionGrant struct {
	Email     string
	IP        string
	Count     int
	ExpiresAt time.Time
	UpdatedAt time.Time
}

func dbToolAccessKey(email, ip string) string {
	return strings.ToLower(strings.TrimSpace(email)) + "|" + strings.TrimSpace(ip)
}

func normalizeDBToolAccessIP(value string) string {
	raw := strings.TrimSpace(value)
	if raw == "" {
		return ""
	}
	if strings.Contains(raw, ",") {
		raw = strings.TrimSpace(strings.Split(raw, ",")[0])
	}
	if host, _, err := net.SplitHostPort(raw); err == nil {
		raw = strings.TrimSpace(host)
	}
	raw = strings.Trim(raw, "[]")
	ip := net.ParseIP(raw)
	if ip == nil {
		return ""
	}
	return ip.String()
}

func (s *service) initializeDBToolAccessRuntime() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.dbAccess = map[string]dbToolSessionGrant{}
	path := strings.TrimSpace(os.Getenv("AURAPANEL_DBTOOLS_RUNTIME_ALLOWLIST_FILE"))
	if path == "" {
		path = defaultDBToolsRuntimeAllowlistFile
	}
	s.dbACLFile = path
	_ = s.writeDBToolAllowlistFileLocked(time.Now().UTC())
}

func (s *service) registerDBToolAccess(email, rawIP string, expiresAt time.Time) {
	email = strings.ToLower(strings.TrimSpace(email))
	ip := normalizeDBToolAccessIP(rawIP)
	if email == "" || ip == "" {
		return
	}

	now := time.Now().UTC()
	if expiresAt.IsZero() || expiresAt.Before(now) {
		expiresAt = now.Add(defaultJWTSessionTTL)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.dbAccess == nil {
		s.dbAccess = map[string]dbToolSessionGrant{}
	}
	s.cleanupExpiredDBToolAccessLocked(now)

	key := dbToolAccessKey(email, ip)
	grant := s.dbAccess[key]
	grant.Email = email
	grant.IP = ip
	if grant.Count < 0 {
		grant.Count = 0
	}
	grant.Count++
	if expiresAt.After(grant.ExpiresAt) {
		grant.ExpiresAt = expiresAt
	}
	grant.UpdatedAt = now
	s.dbAccess[key] = grant

	_ = s.writeDBToolAllowlistFileLocked(now)
}

func (s *service) revokeDBToolAccess(email, rawIP string) {
	email = strings.ToLower(strings.TrimSpace(email))
	ip := normalizeDBToolAccessIP(rawIP)
	if email == "" || ip == "" {
		return
	}

	now := time.Now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.dbAccess == nil {
		return
	}
	s.cleanupExpiredDBToolAccessLocked(now)

	key := dbToolAccessKey(email, ip)
	grant, ok := s.dbAccess[key]
	if ok {
		grant.Count--
		if grant.Count <= 0 {
			delete(s.dbAccess, key)
		} else {
			grant.UpdatedAt = now
			s.dbAccess[key] = grant
		}
	}

	_ = s.writeDBToolAllowlistFileLocked(now)
}

func (s *service) cleanupExpiredDBToolAccessLocked(now time.Time) {
	if s.dbAccess == nil {
		return
	}
	for key, grant := range s.dbAccess {
		if grant.Count <= 0 || (!grant.ExpiresAt.IsZero() && !grant.ExpiresAt.After(now)) {
			delete(s.dbAccess, key)
		}
	}
}

func (s *service) writeDBToolAllowlistFileLocked(now time.Time) error {
	if s.dbACLFile == "" {
		s.dbACLFile = defaultDBToolsRuntimeAllowlistFile
	}

	s.cleanupExpiredDBToolAccessLocked(now)

	unique := map[string]struct{}{}
	for _, grant := range s.dbAccess {
		if grant.IP != "" {
			unique[grant.IP] = struct{}{}
		}
	}

	ips := make([]string, 0, len(unique))
	for ip := range unique {
		ips = append(ips, ip)
	}
	sort.Strings(ips)

	content := ""
	if len(ips) > 0 {
		content = strings.Join(ips, "\n") + "\n"
	}

	dir := filepath.Dir(s.dbACLFile)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return err
	}
	tmp := s.dbACLFile + ".tmp"
	if err := os.WriteFile(tmp, []byte(content), 0644); err != nil {
		return err
	}
	return os.Rename(tmp, s.dbACLFile)
}
