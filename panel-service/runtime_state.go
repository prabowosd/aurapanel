package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const defaultRuntimeStatePath = "/var/lib/aurapanel/panel-service-state.json"

type persistedRuntimeState struct {
	State   appState    `json:"state"`
	Modules moduleState `json:"modules"`
}

func runtimeStatePath() string {
	return envOr("AURAPANEL_STATE_FILE", defaultRuntimeStatePath)
}

func (s *service) loadRuntimeState() error {
	path := runtimeStatePath()
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var persisted persistedRuntimeState
	if err := json.Unmarshal(raw, &persisted); err != nil {
		return fmt.Errorf("decode runtime state: %w", err)
	}

	s.state = persisted.State
	s.modules = persisted.Modules
	return nil
}

func (s *service) saveRuntimeState() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.saveRuntimeStateLocked()
}

func (s *service) saveRuntimeStateLocked() error {
	path := runtimeStatePath()
	payload := persistedRuntimeState{
		State:   s.state,
		Modules: s.modules,
	}
	raw, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("encode runtime state: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tempPath := path + ".tmp"
	if err := os.WriteFile(tempPath, raw, 0o600); err != nil {
		return err
	}
	return os.Rename(tempPath, path)
}
