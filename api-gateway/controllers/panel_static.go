package controllers

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func panelDistPath() string {
	raw := strings.TrimSpace(os.Getenv("AURAPANEL_PANEL_DIST"))
	if raw == "" {
		return "/opt/aurapanel/frontend/dist"
	}
	return raw
}

func isAPIPath(path string) bool {
	return strings.HasPrefix(path, "/api/")
}

func isStaticAssetLikePath(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".js", ".mjs", ".css", ".map", ".json", ".txt", ".xml", ".ico",
		".png", ".jpg", ".jpeg", ".gif", ".webp", ".avif", ".svg",
		".woff", ".woff2", ".ttf", ".eot", ".otf", ".wasm":
		return true
	default:
		return false
	}
}

func serveIndexNoCache(w http.ResponseWriter, r *http.Request, dist string) {
	// Prevent stale SPA shell after deployments; hashed assets remain cacheable.
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
	http.ServeFile(w, r, filepath.Join(dist, "index.html"))
}

// PanelStaticHandler serves compiled frontend assets and falls back to index.html for SPA routes.
func PanelStaticHandler() http.Handler {
	dist := panelDistPath()
	fileServer := http.FileServer(http.Dir(dist))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isAPIPath(r.URL.Path) {
			http.NotFound(w, r)
			return
		}

		cleanPath := filepath.Clean(r.URL.Path)
		if cleanPath == "." || cleanPath == "/" {
			serveIndexNoCache(w, r, dist)
			return
		}

		target := filepath.Join(dist, strings.TrimPrefix(cleanPath, "/"))
		if info, err := os.Stat(target); err == nil && !info.IsDir() {
			fileServer.ServeHTTP(w, r)
			return
		}

		// Missing file-like paths should not fall back to index.html because
		// browsers expect module/script MIME types for these URLs.
		if strings.HasPrefix(cleanPath, "/assets/") || isStaticAssetLikePath(cleanPath) {
			http.NotFound(w, r)
			return
		}

		serveIndexNoCache(w, r, dist)
	})
}
