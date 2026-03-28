package main

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const defaultRuntimeStatePath = "/var/lib/aurapanel/panel-service-state.json"

type persistedRuntimeState struct {
	State   appState    `json:"state"`
	Modules moduleState `json:"modules"`
}

type runtimeSnapshot struct {
	State   appState
	Modules moduleState
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
	if rehydrateSeedCredentials(&s.state) {
		if err := s.saveRuntimeStateLocked(); err != nil {
			return fmt.Errorf("persist migrated runtime state: %w", err)
		}
	}
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

func (s *service) captureRuntimeSnapshotLocked() (runtimeSnapshot, error) {
	state, err := cloneValue(s.state)
	if err != nil {
		return runtimeSnapshot{}, fmt.Errorf("clone state: %w", err)
	}
	modules, err := cloneValue(s.modules)
	if err != nil {
		return runtimeSnapshot{}, fmt.Errorf("clone modules: %w", err)
	}
	return runtimeSnapshot{State: state, Modules: modules}, nil
}

func (s *service) restoreRuntimeSnapshotLocked(snapshot runtimeSnapshot) {
	s.state = snapshot.State
	s.modules = snapshot.Modules
}

func cloneValue[T any](input T) (T, error) {
	var zero T

	var buffer bytes.Buffer
	if err := gob.NewEncoder(&buffer).Encode(input); err != nil {
		return zero, err
	}

	var output T
	if err := gob.NewDecoder(&buffer).Decode(&output); err != nil {
		return zero, err
	}
	return output, nil
}

func rehydrateSeedCredentials(state *appState) bool {
	adminEmail, adminHash := loadAdminSeedCredentials()
	if strings.TrimSpace(adminHash) == "" {
		return false
	}

	changed := false
	for i := range state.Users {
		user := &state.Users[i]
		if !isSeedAdminUser(*user, adminEmail) {
			continue
		}
		if strings.TrimSpace(user.PasswordHash) == "" {
			user.PasswordHash = adminHash
			changed = true
		}
		return changed
	}

	seeded := seedState().Users[0]
	seeded.Email = adminEmail
	seeded.PasswordHash = adminHash
	seeded.ID = nextSeedUserID(*state)
	state.Users = append(state.Users, seeded)
	if state.NextUserID <= seeded.ID {
		state.NextUserID = seeded.ID + 1
	}
	return true
}

func isSeedAdminUser(user PanelUser, adminEmail string) bool {
	return strings.EqualFold(strings.TrimSpace(user.Email), strings.TrimSpace(adminEmail)) ||
		(strings.EqualFold(strings.TrimSpace(user.Username), "admin") && strings.EqualFold(strings.TrimSpace(user.Role), "admin"))
}

func nextSeedUserID(state appState) int {
	nextID := state.NextUserID
	if nextID < 1 {
		nextID = 1
	}
	for _, user := range state.Users {
		if user.ID >= nextID {
			nextID = user.ID + 1
		}
	}
	return nextID
}
