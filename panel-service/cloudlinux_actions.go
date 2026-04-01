package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os/exec"
	"runtime"
	"sort"
	"strings"
	"time"
)

const (
	cloudLinuxActionHistoryLimit = 100
	cloudLinuxActionOutputLimit  = 4096
)

type cloudLinuxActionDefinition struct {
	ID               string
	Label            string
	Description      string
	Command          []string
	RequiredCommands []string
	RequiredFeatures []string
	TimeoutSeconds   int
}

type cloudLinuxActionCatalogItem struct {
	ID               string   `json:"id"`
	Label            string   `json:"label"`
	Description      string   `json:"description"`
	Command          string   `json:"command"`
	RequiredCommands []string `json:"required_commands"`
	RequiredFeatures []string `json:"required_features"`
	TimeoutSeconds   int      `json:"timeout_seconds"`
	Available        bool     `json:"available"`
	Reason           string   `json:"reason,omitempty"`
}

type cloudLinuxActionRunResponse struct {
	ID            string `json:"id"`
	Action        string `json:"action"`
	Status        string `json:"status"`
	DryRun        bool   `json:"dry_run"`
	RequestedBy   string `json:"requested_by"`
	RequestedAt   int64  `json:"requested_at"`
	FinishedAt    int64  `json:"finished_at"`
	DurationMS    int64  `json:"duration_ms"`
	Command       string `json:"command"`
	Message       string `json:"message"`
	Output        string `json:"output,omitempty"`
	ExitCode      int    `json:"exit_code,omitempty"`
	Availability  bool   `json:"availability"`
	BlockedReason string `json:"blocked_reason,omitempty"`
}

type cloudLinuxActionAuditEntry struct {
	ID          string `json:"id"`
	Action      string `json:"action"`
	Status      string `json:"status"`
	DryRun      bool   `json:"dry_run"`
	RequestedBy string `json:"requested_by"`
	RequestedAt int64  `json:"requested_at"`
	FinishedAt  int64  `json:"finished_at"`
	DurationMS  int64  `json:"duration_ms"`
	Command     string `json:"command"`
	Message     string `json:"message"`
	Output      string `json:"output,omitempty"`
}

var cloudLinuxActionDefinitions = []cloudLinuxActionDefinition{
	{
		ID:               "cagefs_force_update",
		Label:            "CageFS Force Update",
		Description:      "Rebuild CageFS skeleton and refresh mounts.",
		Command:          []string{"cagefsctl", "--force-update"},
		RequiredCommands: []string{"cagefsctl"},
		RequiredFeatures: []string{"cagefs"},
		TimeoutSeconds:   120,
	},
	{
		ID:               "lvestats_restart",
		Label:            "LVE Stats Restart",
		Description:      "Restart lvestats service to refresh runtime metrics collectors.",
		Command:          []string{"systemctl", "restart", "lvestats"},
		RequiredCommands: []string{"systemctl"},
		RequiredFeatures: []string{"lve_manager"},
		TimeoutSeconds:   45,
	},
	{
		ID:               "mysql_governor_reload",
		Label:            "MySQL Governor Reload",
		Description:      "Reload MySQL Governor runtime policy from current configuration.",
		Command:          []string{"dbctl", "--reload"},
		RequiredCommands: []string{"dbctl"},
		RequiredFeatures: []string{"mysql_governor"},
		TimeoutSeconds:   60,
	},
}

func cloudLinuxActionByID(actionID string) (cloudLinuxActionDefinition, bool) {
	key := strings.ToLower(strings.TrimSpace(actionID))
	for _, item := range cloudLinuxActionDefinitions {
		if item.ID == key {
			return item, true
		}
	}
	return cloudLinuxActionDefinition{}, false
}

func cloudLinuxActionReadiness(item cloudLinuxActionDefinition, status cloudLinuxStatus) (bool, string) {
	if runtime.GOOS != "linux" {
		return false, "CloudLinux actions are supported only on Linux hosts."
	}
	if !status.Available {
		return false, "CloudLinux was not detected on this node."
	}
	for _, command := range item.RequiredCommands {
		exists := status.Commands[command]
		if !exists {
			exists = commandExists(command)
		}
		if !exists {
			return false, fmt.Sprintf("Required command is missing: %s", command)
		}
	}
	for _, feature := range item.RequiredFeatures {
		if !status.Features[feature] {
			return false, fmt.Sprintf("Required feature is not available: %s", feature)
		}
	}
	return true, ""
}

func buildCloudLinuxActionCatalog(status cloudLinuxStatus) []cloudLinuxActionCatalogItem {
	items := make([]cloudLinuxActionCatalogItem, 0, len(cloudLinuxActionDefinitions))
	for _, action := range cloudLinuxActionDefinitions {
		available, reason := cloudLinuxActionReadiness(action, status)
		items = append(items, cloudLinuxActionCatalogItem{
			ID:               action.ID,
			Label:            action.Label,
			Description:      action.Description,
			Command:          strings.Join(action.Command, " "),
			RequiredCommands: append([]string(nil), action.RequiredCommands...),
			RequiredFeatures: append([]string(nil), action.RequiredFeatures...),
			TimeoutSeconds:   action.TimeoutSeconds,
			Available:        available,
			Reason:           reason,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Label < items[j].Label
	})
	return items
}

func (s *service) appendCloudLinuxActionLocked(entry cloudLinuxActionAuditEntry) {
	s.modules.CloudLinuxActions = append([]cloudLinuxActionAuditEntry{entry}, s.modules.CloudLinuxActions...)
	if len(s.modules.CloudLinuxActions) > cloudLinuxActionHistoryLimit {
		s.modules.CloudLinuxActions = s.modules.CloudLinuxActions[:cloudLinuxActionHistoryLimit]
	}
}

func trimCloudLinuxOutput(value string, limit int) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	if limit <= 0 || len(trimmed) <= limit {
		return trimmed
	}
	return trimmed[:limit] + "...(truncated)"
}

func runCloudLinuxAction(action cloudLinuxActionDefinition) (output string, exitCode int, err error) {
	if len(action.Command) == 0 {
		return "", 1, fmt.Errorf("action command is not configured")
	}

	timeoutSeconds := action.TimeoutSeconds
	if timeoutSeconds <= 0 {
		timeoutSeconds = 30
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSeconds)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, action.Command[0], action.Command[1:]...)
	raw, runErr := cmd.CombinedOutput()
	output = strings.TrimSpace(string(raw))

	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return output, 124, fmt.Errorf("command timed out after %d seconds", timeoutSeconds)
	}
	if runErr != nil {
		exitCode = 1
		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) {
			exitCode = exitErr.ExitCode()
		}
		return output, exitCode, runErr
	}

	return output, 0, nil
}

func (s *service) handleCloudLinuxActions(w http.ResponseWriter) {
	status := detectCloudLinuxStatus()
	catalog := buildCloudLinuxActionCatalog(status)

	s.mu.RLock()
	history := append([]cloudLinuxActionAuditEntry(nil), s.modules.CloudLinuxActions...)
	s.mu.RUnlock()

	writeJSON(w, http.StatusOK, apiResponse{
		Status: "success",
		Data: map[string]interface{}{
			"actions": catalog,
			"history": history,
		},
	})
}

func (s *service) handleCloudLinuxActionRun(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Action string `json:"action"`
		DryRun bool   `json:"dry_run"`
	}
	if err := decodeJSON(r, &payload); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid CloudLinux action payload.")
		return
	}

	action, ok := cloudLinuxActionByID(payload.Action)
	if !ok {
		writeError(w, http.StatusNotFound, "CloudLinux action is not supported.")
		return
	}

	principal, principalOK := principalFromContext(r.Context())
	requestedBy := "system"
	if principalOK {
		requestedBy = firstNonEmpty(strings.TrimSpace(principal.Username), strings.TrimSpace(principal.Email), "system")
	}

	now := time.Now().UTC()
	status := detectCloudLinuxStatus()
	available, blockedReason := cloudLinuxActionReadiness(action, status)
	commandText := strings.Join(action.Command, " ")

	result := cloudLinuxActionRunResponse{
		ID:           generateSecret(6),
		Action:       action.ID,
		DryRun:       payload.DryRun,
		RequestedBy:  requestedBy,
		RequestedAt:  now.Unix(),
		Command:      commandText,
		Availability: available,
	}

	if !available {
		result.Status = "blocked"
		result.BlockedReason = blockedReason
		result.Message = blockedReason
		result.FinishedAt = time.Now().UTC().Unix()
		result.DurationMS = 0

		s.mu.Lock()
		s.appendCloudLinuxActionLocked(cloudLinuxActionAuditEntry{
			ID:          result.ID,
			Action:      result.Action,
			Status:      result.Status,
			DryRun:      result.DryRun,
			RequestedBy: result.RequestedBy,
			RequestedAt: result.RequestedAt,
			FinishedAt:  result.FinishedAt,
			DurationMS:  result.DurationMS,
			Command:     result.Command,
			Message:     result.Message,
		})
		s.appendActivityLocked(result.RequestedBy, "cloudlinux_action_blocked", fmt.Sprintf("%s blocked: %s", action.ID, blockedReason), r.RemoteAddr)
		s.mu.Unlock()

		writeJSON(w, http.StatusOK, apiResponse{
			Status:  "success",
			Message: blockedReason,
			Data:    result,
		})
		return
	}

	if payload.DryRun {
		result.Status = "dry_run"
		result.Message = "Dry-run completed. No system changes were applied."
		result.FinishedAt = time.Now().UTC().Unix()
		result.DurationMS = 0

		s.mu.Lock()
		s.appendCloudLinuxActionLocked(cloudLinuxActionAuditEntry{
			ID:          result.ID,
			Action:      result.Action,
			Status:      result.Status,
			DryRun:      result.DryRun,
			RequestedBy: result.RequestedBy,
			RequestedAt: result.RequestedAt,
			FinishedAt:  result.FinishedAt,
			DurationMS:  result.DurationMS,
			Command:     result.Command,
			Message:     result.Message,
		})
		s.appendActivityLocked(result.RequestedBy, "cloudlinux_action_dry_run", fmt.Sprintf("%s dry-run prepared.", action.ID), r.RemoteAddr)
		s.mu.Unlock()

		writeJSON(w, http.StatusOK, apiResponse{
			Status:  "success",
			Message: result.Message,
			Data:    result,
		})
		return
	}

	start := time.Now().UTC()
	output, exitCode, runErr := runCloudLinuxAction(action)
	finished := time.Now().UTC()
	durationMS := finished.Sub(start).Milliseconds()

	result.FinishedAt = finished.Unix()
	result.DurationMS = durationMS
	result.Output = trimCloudLinuxOutput(output, cloudLinuxActionOutputLimit)
	result.ExitCode = exitCode

	if runErr != nil {
		result.Status = "failed"
		result.Message = fmt.Sprintf("Action failed (exit code %d).", exitCode)
		if result.Output == "" {
			result.Output = trimCloudLinuxOutput(runErr.Error(), cloudLinuxActionOutputLimit)
		}
	} else {
		result.Status = "success"
		result.Message = "Action completed successfully."
	}

	s.mu.Lock()
	s.appendCloudLinuxActionLocked(cloudLinuxActionAuditEntry{
		ID:          result.ID,
		Action:      result.Action,
		Status:      result.Status,
		DryRun:      result.DryRun,
		RequestedBy: result.RequestedBy,
		RequestedAt: result.RequestedAt,
		FinishedAt:  result.FinishedAt,
		DurationMS:  result.DurationMS,
		Command:     result.Command,
		Message:     result.Message,
		Output:      trimCloudLinuxOutput(result.Output, 512),
	})
	s.appendActivityLocked(
		result.RequestedBy,
		"cloudlinux_action_run",
		fmt.Sprintf("%s status=%s exit=%d dry_run=%t", action.ID, result.Status, result.ExitCode, result.DryRun),
		r.RemoteAddr,
	)
	s.mu.Unlock()

	writeJSON(w, http.StatusOK, apiResponse{
		Status:  "success",
		Message: result.Message,
		Data:    result,
	})
}
