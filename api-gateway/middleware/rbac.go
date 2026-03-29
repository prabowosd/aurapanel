package middleware

import (
	"net/http"
	"strings"
)

const (
	roleAdmin    = "admin"
	roleReseller = "reseller"
	roleUser     = "user"
)

func normalizeRole(role string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case roleAdmin:
		return roleAdmin
	case roleReseller:
		return roleReseller
	default:
		return roleUser
	}
}

func pathMatchesPrefix(path, prefix string) bool {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return false
	}
	normalizedPrefix := strings.TrimSuffix(prefix, "/")
	return path == normalizedPrefix || strings.HasPrefix(path, normalizedPrefix+"/")
}

func isRestrictedNonAdminPath(path string) bool {
	return pathMatchesPrefix(path, "/api/v1/files") ||
		pathMatchesPrefix(path, "/api/v1/security/ssh-keys") ||
		pathMatchesPrefix(path, "/api/v1/websites/custom-ssl") ||
		pathMatchesPrefix(path, "/api/v1/websites/vhost-config")
}

func resellerAllowed(path string) bool {
	if isRestrictedNonAdminPath(path) {
		return false
	}

	allowedPrefixes := []string{
		"/api/v1/auth/me",
		"/api/v1/vhost",
		"/api/v1/websites",
		"/api/v1/dns",
		"/api/v1/db",
		"/api/v1/mail",
		"/api/v1/ftp",
		"/api/v1/sftp",
		"/api/v1/backup",
		"/api/v1/apps",
		"/api/v1/wordpress",
		"/api/v1/php",
		"/api/v1/ssl",
		"/api/v1/monitor/cron",
		"/api/v1/monitor/logs/site",
		"/api/v1/security/status",
		"/api/v1/security/firewall",
		"/api/v1/security/2fa",
		"/api/v1/security/immutable/status",
		"/api/v1/security/ebpf/events",
		"/api/v1/security/malware",
		"/api/v1/status/metrics",
		"/api/v1/status/services",
		"/api/v1/status/processes",
		"/api/v1/status/update",
		"/api/v1/analytics/website-traffic",
	}

	for _, prefix := range allowedPrefixes {
		if pathMatchesPrefix(path, prefix) {
			return true
		}
	}
	return false
}

func userAllowed(method, path string) bool {
	if isRestrictedNonAdminPath(path) {
		return false
	}

	if path == "/api/v1/auth/me" || path == "/api/v1/status/metrics" || path == "/api/v1/status/services" || path == "/api/v1/status/update" {
		return true
	}

	// Personal security actions.
	if pathMatchesPrefix(path, "/api/v1/security/2fa") ||
		pathMatchesPrefix(path, "/api/v1/security/status") ||
		pathMatchesPrefix(path, "/api/v1/security/immutable/status") ||
		pathMatchesPrefix(path, "/api/v1/security/ebpf/events") {
		return true
	}

	if method == http.MethodGet || method == http.MethodHead {
		return pathMatchesPrefix(path, "/api/v1/vhost/list") ||
			pathMatchesPrefix(path, "/api/v1/websites/aliases") ||
			pathMatchesPrefix(path, "/api/v1/websites/advanced-config") ||
			pathMatchesPrefix(path, "/api/v1/monitor/logs/site") ||
			pathMatchesPrefix(path, "/api/v1/analytics/website-traffic")
	}

	return false
}

func isAuthorized(role, method, path string) bool {
	switch normalizeRole(role) {
	case roleAdmin:
		return true
	case roleReseller:
		return resellerAllowed(path)
	case roleUser:
		return userAllowed(method, path)
	default:
		return false
	}
}

// RBACMiddleware enforces endpoint-level role permissions after authentication.
func RBACMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := GetAuthUser(r.Context())
		if !ok {
			WriteError(w, r, http.StatusUnauthorized, "AUTH_UNAUTHORIZED", "Unauthorized")
			return
		}

		if !isAuthorized(user.Role, r.Method, r.URL.Path) {
			WriteError(w, r, http.StatusForbidden, "AUTH_FORBIDDEN", "Role is not allowed for this endpoint")
			return
		}

		next.ServeHTTP(w, r)
	})
}
