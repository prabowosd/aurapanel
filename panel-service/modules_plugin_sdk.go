package main

import (
	"net/http"
	"sort"
	"strings"
	"time"
)

func (s *service) handlePanelPluginsList(w http.ResponseWriter) {
	s.mu.RLock()
	items := append([]PanelPlugin(nil), s.modules.PanelPlugins...)
	s.mu.RUnlock()

	sort.Slice(items, func(i, j int) bool {
		if items[i].Enabled != items[j].Enabled {
			return items[i].Enabled
		}
		return items[i].ID < items[j].ID
	})
	writeJSON(w, http.StatusOK, apiResponse{Status: "success", Data: items})
}

func (s *service) handlePanelPluginSDKInfo(w http.ResponseWriter) {
	writeJSON(w, http.StatusOK, apiResponse{
		Status: "success",
		Data: map[string]interface{}{
			"manifest_version": "v1",
			"required_fields":  []string{"id", "name", "version", "entrypoint"},
			"supported_hooks": []string{
				"ui.route",
				"activity.append",
				"website.lifecycle",
				"mail.lifecycle",
				"backup.lifecycle",
			},
			"capabilities": []string{
				"read:sites",
				"read:mail",
				"write:mail",
				"read:backup",
				"write:backup",
				"read:security",
			},
		},
	})
}

func (s *service) handlePanelPluginSave(w http.ResponseWriter, r *http.Request) {
	var payload PanelPlugin
	if err := decodeJSON(r, &payload); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid plugin payload.")
		return
	}

	payload.ID = sanitizeName(payload.ID)
	payload.Name = strings.TrimSpace(payload.Name)
	payload.Version = strings.TrimSpace(payload.Version)
	payload.Description = strings.TrimSpace(payload.Description)
	payload.Entrypoint = strings.TrimSpace(payload.Entrypoint)
	payload.Author = strings.TrimSpace(payload.Author)

	if payload.ID == "" || payload.Name == "" || payload.Entrypoint == "" {
		writeError(w, http.StatusBadRequest, "Plugin id, name and entrypoint are required.")
		return
	}
	if payload.Version == "" {
		payload.Version = "0.1.0"
	}

	now := time.Now().UTC().UnixMilli()
	payload.UpdatedAt = now

	s.mu.Lock()
	defer s.mu.Unlock()

	index := -1
	for i, item := range s.modules.PanelPlugins {
		if item.ID == payload.ID {
			index = i
			break
		}
	}

	if index >= 0 {
		payload.CreatedAt = s.modules.PanelPlugins[index].CreatedAt
		s.modules.PanelPlugins[index] = payload
		writeJSON(w, http.StatusOK, apiResponse{Status: "success", Message: "Plugin updated.", Data: payload})
		return
	}

	payload.CreatedAt = now
	if !payload.Enabled {
		payload.Enabled = true
	}
	s.modules.PanelPlugins = append(s.modules.PanelPlugins, payload)
	writeJSON(w, http.StatusOK, apiResponse{Status: "success", Message: "Plugin added.", Data: payload})
}

func (s *service) handlePanelPluginToggle(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		ID      string `json:"id"`
		Enabled bool   `json:"enabled"`
	}
	if err := decodeJSON(r, &payload); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid plugin toggle payload.")
		return
	}
	id := sanitizeName(payload.ID)
	if id == "" {
		writeError(w, http.StatusBadRequest, "Plugin id is required.")
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.modules.PanelPlugins {
		if s.modules.PanelPlugins[i].ID == id {
			s.modules.PanelPlugins[i].Enabled = payload.Enabled
			s.modules.PanelPlugins[i].UpdatedAt = time.Now().UTC().UnixMilli()
			writeJSON(w, http.StatusOK, apiResponse{
				Status:  "success",
				Message: "Plugin state updated.",
				Data:    s.modules.PanelPlugins[i],
			})
			return
		}
	}
	writeError(w, http.StatusNotFound, "Plugin not found.")
}

func (s *service) handlePanelPluginDelete(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		ID string `json:"id"`
	}
	if err := decodeJSON(r, &payload); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid plugin delete payload.")
		return
	}
	id := sanitizeName(payload.ID)
	if id == "" {
		writeError(w, http.StatusBadRequest, "Plugin id is required.")
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	filtered := s.modules.PanelPlugins[:0]
	deleted := false
	for _, item := range s.modules.PanelPlugins {
		if item.ID == id {
			deleted = true
			continue
		}
		filtered = append(filtered, item)
	}
	s.modules.PanelPlugins = filtered
	if !deleted {
		writeError(w, http.StatusNotFound, "Plugin not found.")
		return
	}
	writeJSON(w, http.StatusOK, apiResponse{Status: "success", Message: "Plugin removed."})
}
