package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

func normalizeDBTool(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "phpmyadmin":
		return "phpmyadmin"
	case "pgadmin", "pgadmin4":
		return "pgadmin"
	default:
		return ""
	}
}

func (s *service) handleDBToolSSO(w http.ResponseWriter, r *http.Request, tool string) {
	tool = normalizeDBTool(tool)
	if tool == "" {
		writeError(w, http.StatusBadRequest, "Unsupported database tool.")
		return
	}

	var payload struct {
		TTLSeconds int `json:"ttl_seconds"`
	}
	if err := decodeJSON(r, &payload); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid DB tool SSO payload.")
		return
	}

	ttlSeconds := clampInt(payload.TTLSeconds, 60, 900)
	token := generateSecret(12)
	expiresAt := time.Now().UTC().Add(time.Duration(ttlSeconds) * time.Second)

	issuer := "system"
	if principal, ok := principalFromContext(r.Context()); ok {
		issuer = firstNonEmpty(principal.Email, principal.Username, principal.Name, "system")
	}

	s.mu.Lock()
	if s.modules.DBToolTokens == nil {
		s.modules.DBToolTokens = map[string]DBToolToken{}
	}
	s.modules.DBToolTokens[token] = DBToolToken{
		Token:     token,
		Tool:      tool,
		IssuedBy:  issuer,
		ExpiresAt: expiresAt,
	}
	s.appendActivityLocked(issuer, "db_tool_launch", fmt.Sprintf("%s launch token issued.", tool), "")
	s.mu.Unlock()

	writeJSON(w, http.StatusOK, apiResponse{
		Status: "success",
		Data: map[string]interface{}{
			"url":        fmt.Sprintf("/api/v1/db/tools/%s/sso/consume?token=%s", tool, token),
			"tool":       tool,
			"expires_at": expiresAt.Format(time.RFC3339),
		},
	})
}

func (s *service) handleDBToolConsume(w http.ResponseWriter, r *http.Request, tool string) {
	tool = normalizeDBTool(tool)
	token := strings.TrimSpace(r.URL.Query().Get("token"))
	if tool == "" || token == "" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("<html><body><h1>Invalid DB tool token</h1></body></html>"))
		return
	}

	now := time.Now().UTC()
	s.mu.Lock()
	item, ok := s.modules.DBToolTokens[token]
	if ok {
		delete(s.modules.DBToolTokens, token)
	}
	s.mu.Unlock()
	if !ok || item.Tool != tool || item.ExpiresAt.Before(now) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusGone)
		_, _ = w.Write([]byte("<html><body><h1>DB tool token expired</h1></body></html>"))
		return
	}
	s.registerDBToolAccess(item.IssuedBy, serviceClientIP(r), item.ExpiresAt)

	targetURL := resolveDBToolBaseURL(r, tool)
	http.Redirect(w, r, targetURL, http.StatusFound)
}

func resolveDBToolBaseURL(r *http.Request, tool string) string {
	tool = normalizeDBTool(tool)
	if tool == "" {
		return "/"
	}

	baseURL := ""
	defaultPath := ""
	switch tool {
	case "phpmyadmin":
		baseURL = strings.TrimSpace(os.Getenv("AURAPANEL_PHPMYADMIN_BASE_URL"))
		defaultPath = "/phpmyadmin/index.php"
	case "pgadmin":
		baseURL = strings.TrimSpace(os.Getenv("AURAPANEL_PGADMIN_BASE_URL"))
		defaultPath = "/pgadmin4/"
	}
	if baseURL == "" {
		baseURL = defaultPath
	}

	lower := strings.ToLower(baseURL)
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		return baseURL
	}

	origin := servicePublicOrigin(r)
	if origin == "" {
		if strings.HasPrefix(baseURL, "/") {
			return baseURL
		}
		return "/" + strings.TrimLeft(baseURL, "/")
	}
	if strings.HasPrefix(baseURL, "/") {
		return origin + baseURL
	}
	return origin + "/" + strings.TrimLeft(baseURL, "/")
}

func servicePublicOrigin(r *http.Request) string {
	host := forwardedHeaderValue(r.Header.Get("X-Forwarded-Host"))
	if host == "" {
		host = strings.TrimSpace(r.Host)
	}
	if host == "" {
		return ""
	}
	scheme := forwardedHeaderValue(r.Header.Get("X-Forwarded-Proto"))
	if scheme == "" {
		if r.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}
	return fmt.Sprintf("%s://%s", scheme, host)
}
