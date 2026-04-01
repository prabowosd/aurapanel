package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"
)

const (
	cloudLinuxRolloutHistoryLimit   = 100
	cloudLinuxRolloutCommandTimeout = 20
	cloudLinuxRolloutConfirmToken   = "APPLY_CLOUDLINUX"
)

type cloudLinuxRolloutApplyResultItem struct {
	Username        string `json:"username"`
	PackageName     string `json:"package_name"`
	Status          string `json:"status"`
	Message         string `json:"message"`
	Command         string `json:"command"`
	Readiness       string `json:"readiness,omitempty"`
	ReadinessReason string `json:"readiness_reason,omitempty"`
	ExitCode        int    `json:"exit_code,omitempty"`
	DurationMS      int64  `json:"duration_ms"`
	Output          string `json:"output,omitempty"`
}

type cloudLinuxRolloutApplyResponse struct {
	ID             string                             `json:"id"`
	DryRun         bool                               `json:"dry_run"`
	RequestedBy    string                             `json:"requested_by"`
	RequestedAt    int64                              `json:"requested_at"`
	FinishedAt     int64                              `json:"finished_at"`
	DurationMS     int64                              `json:"duration_ms"`
	PackageFilter  string                             `json:"package_filter,omitempty"`
	OnlyReady      bool                               `json:"only_ready"`
	MaxUsers       int                                `json:"max_users"`
	RequestedUsers int                                `json:"requested_users"`
	PlannedUsers   int                                `json:"planned_users"`
	AttemptedUsers int                                `json:"attempted_users"`
	Succeeded      int                                `json:"succeeded"`
	Failed         int                                `json:"failed"`
	Skipped        int                                `json:"skipped"`
	ApplyEnabled   bool                               `json:"apply_enabled"`
	Message        string                             `json:"message"`
	Results        []cloudLinuxRolloutApplyResultItem `json:"results"`
}

type cloudLinuxRolloutAuditEntry struct {
	ID             string `json:"id"`
	Status         string `json:"status"`
	DryRun         bool   `json:"dry_run"`
	RequestedBy    string `json:"requested_by"`
	RequestedAt    int64  `json:"requested_at"`
	FinishedAt     int64  `json:"finished_at"`
	DurationMS     int64  `json:"duration_ms"`
	PackageFilter  string `json:"package_filter,omitempty"`
	OnlyReady      bool   `json:"only_ready"`
	MaxUsers       int    `json:"max_users"`
	RequestedUsers int    `json:"requested_users"`
	PlannedUsers   int    `json:"planned_users"`
	AttemptedUsers int    `json:"attempted_users"`
	Succeeded      int    `json:"succeeded"`
	Failed         int    `json:"failed"`
	Skipped        int    `json:"skipped"`
	Message        string `json:"message"`
}

func cloudLinuxApplyEnabled() bool {
	value := strings.TrimSpace(os.Getenv("AURAPANEL_CLOUDLINUX_APPLY_ENABLED"))
	if value == "" {
		value = strings.TrimSpace(readEnvFileValue(adminServiceEnvPath(), "AURAPANEL_CLOUDLINUX_APPLY_ENABLED"))
	}
	return envBoolEnabled(value)
}

func (s *service) appendCloudLinuxRolloutLocked(entry cloudLinuxRolloutAuditEntry) {
	s.modules.CloudLinuxRollouts = append([]cloudLinuxRolloutAuditEntry{entry}, s.modules.CloudLinuxRollouts...)
	if len(s.modules.CloudLinuxRollouts) > cloudLinuxRolloutHistoryLimit {
		s.modules.CloudLinuxRollouts = s.modules.CloudLinuxRollouts[:cloudLinuxRolloutHistoryLimit]
	}
}

func buildCloudLinuxRolloutCommandArgs(item cloudLinuxRolloutEntry) ([]string, error) {
	username := sanitizeName(item.Username)
	if strings.TrimSpace(username) == "" {
		return nil, fmt.Errorf("username is invalid for rollout command")
	}
	args := []string{
		"set-user",
		username,
		fmt.Sprintf("--speed=%d%%", clampInt(item.CPUPercent, 1, 10000)),
		fmt.Sprintf("--pmem=%dM", clampInt(item.MemoryMB, 64, 1048576)),
		fmt.Sprintf("--io=%dMBps", clampInt(item.IOMBPS, 1, 10000)),
		fmt.Sprintf("--nproc=%d", clampInt(item.NProc, 1, 65535)),
		fmt.Sprintf("--iops=%d", clampInt(item.IOPS, 1, 65535)),
		fmt.Sprintf("--ep=%d", clampInt(item.EntryProcesses, 1, 65535)),
	}
	return args, nil
}

func runCloudLinuxRolloutCommand(args []string) (output string, exitCode int, err error) {
	if len(args) == 0 {
		return "", 1, fmt.Errorf("command arguments are not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), cloudLinuxRolloutCommandTimeout*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "lvectl", args...)
	raw, runErr := cmd.CombinedOutput()
	output = strings.TrimSpace(string(raw))

	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return output, 124, fmt.Errorf("command timed out after %d seconds", cloudLinuxRolloutCommandTimeout)
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

func normalizeRolloutUserFilter(values []string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, raw := range values {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			continue
		}
		out[cloudLinuxUserKey(trimmed)] = struct{}{}
		safe := sanitizeName(trimmed)
		if safe != "" {
			out[cloudLinuxUserKey(safe)] = struct{}{}
		}
	}
	return out
}

func filterCloudLinuxRolloutRows(rows []cloudLinuxRolloutEntry, usernames map[string]struct{}) []cloudLinuxRolloutEntry {
	if len(usernames) == 0 {
		return append([]cloudLinuxRolloutEntry(nil), rows...)
	}
	out := make([]cloudLinuxRolloutEntry, 0, len(rows))
	for _, row := range rows {
		candidates := []string{cloudLinuxUserKey(row.Username), cloudLinuxUserKey(sanitizeName(row.Username))}
		matched := false
		for _, key := range candidates {
			if key == "" {
				continue
			}
			if _, ok := usernames[key]; ok {
				matched = true
				break
			}
		}
		if matched {
			out = append(out, row)
		}
	}
	return out
}

func (s *service) handleCloudLinuxRolloutHistory(w http.ResponseWriter) {
	s.mu.RLock()
	history := append([]cloudLinuxRolloutAuditEntry(nil), s.modules.CloudLinuxRollouts...)
	s.mu.RUnlock()

	sort.Slice(history, func(i, j int) bool {
		return history[i].RequestedAt > history[j].RequestedAt
	})

	writeJSON(w, http.StatusOK, apiResponse{Status: "success", Data: history})
}

func (s *service) handleCloudLinuxRolloutApply(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Package   string   `json:"package"`
		OnlyReady *bool    `json:"only_ready"`
		DryRun    *bool    `json:"dry_run"`
		MaxUsers  int      `json:"max_users"`
		Usernames []string `json:"usernames"`
		Confirm   string   `json:"confirm"`
	}
	if err := decodeJSON(r, &payload); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid CloudLinux rollout payload.")
		return
	}

	dryRun := true
	if payload.DryRun != nil {
		dryRun = *payload.DryRun
	}
	onlyReady := true
	if payload.OnlyReady != nil {
		onlyReady = *payload.OnlyReady
	}
	maxUsers := clampInt(payload.MaxUsers, 1, 500)
	if payload.MaxUsers <= 0 {
		maxUsers = 25
	}

	applyEnabled := cloudLinuxApplyEnabled()
	if !dryRun {
		if !applyEnabled {
			writeError(w, http.StatusForbidden, "CloudLinux apply mode is disabled. Set AURAPANEL_CLOUDLINUX_APPLY_ENABLED=true.")
			return
		}
		if strings.TrimSpace(payload.Confirm) != cloudLinuxRolloutConfirmToken {
			writeError(w, http.StatusBadRequest, "Apply mode requires confirm="+cloudLinuxRolloutConfirmToken+".")
			return
		}
	}

	principal, principalOK := principalFromContext(r.Context())
	requestedBy := "system"
	if principalOK {
		requestedBy = firstNonEmpty(strings.TrimSpace(principal.Username), strings.TrimSpace(principal.Email), "system")
	}

	status := detectCloudLinuxStatus()
	s.mu.RLock()
	packages := append([]Package(nil), s.state.Packages...)
	websites := append([]Website(nil), s.state.Websites...)
	users := append([]PanelUser(nil), s.state.Users...)
	s.mu.RUnlock()

	plan := buildCloudLinuxRolloutPayload(status, packages, websites, users, payload.Package, onlyReady)
	selected := filterCloudLinuxRolloutRows(plan.Users, normalizeRolloutUserFilter(payload.Usernames))
	requestedUsers := len(selected)
	if len(selected) > maxUsers {
		selected = selected[:maxUsers]
	}
	plannedUsers := len(selected)

	startedAt := time.Now().UTC()
	response := cloudLinuxRolloutApplyResponse{
		ID:             generateSecret(6),
		DryRun:         dryRun,
		RequestedBy:    requestedBy,
		RequestedAt:    startedAt.Unix(),
		PackageFilter:  strings.TrimSpace(payload.Package),
		OnlyReady:      onlyReady,
		MaxUsers:       maxUsers,
		RequestedUsers: requestedUsers,
		PlannedUsers:   plannedUsers,
		ApplyEnabled:   applyEnabled,
		Results:        make([]cloudLinuxRolloutApplyResultItem, 0, plannedUsers),
	}

	for _, item := range selected {
		entry := cloudLinuxRolloutApplyResultItem{
			Username:        item.Username,
			PackageName:     item.PackageName,
			Readiness:       item.Readiness,
			ReadinessReason: item.ReadinessReason,
			Command:         item.CommandHint,
		}

		if !item.ReadyForApply {
			entry.Status = "blocked"
			entry.Message = firstNonEmpty(strings.TrimSpace(item.ReadinessReason), "User is not ready for rollout apply.")
			response.Skipped++
			response.Results = append(response.Results, entry)
			continue
		}

		response.AttemptedUsers++
		if dryRun {
			entry.Status = "dry_run"
			entry.Message = "Dry-run preview generated."
			response.Skipped++
			response.Results = append(response.Results, entry)
			continue
		}

		args, buildErr := buildCloudLinuxRolloutCommandArgs(item)
		if buildErr != nil {
			entry.Status = "failed"
			entry.Message = buildErr.Error()
			response.Failed++
			response.Results = append(response.Results, entry)
			continue
		}
		entry.Command = "lvectl " + strings.Join(args, " ")

		runStart := time.Now().UTC()
		output, exitCode, runErr := runCloudLinuxRolloutCommand(args)
		entry.DurationMS = time.Since(runStart).Milliseconds()
		entry.Output = trimCloudLinuxOutput(output, 1024)
		entry.ExitCode = exitCode

		if runErr != nil {
			entry.Status = "failed"
			entry.Message = fmt.Sprintf("Command failed (exit code %d).", exitCode)
			if entry.Output == "" {
				entry.Output = trimCloudLinuxOutput(runErr.Error(), 1024)
			}
			response.Failed++
		} else {
			entry.Status = "success"
			entry.Message = "Command applied successfully."
			response.Succeeded++
		}
		response.Results = append(response.Results, entry)
	}

	finishedAt := time.Now().UTC()
	response.FinishedAt = finishedAt.Unix()
	response.DurationMS = finishedAt.Sub(startedAt).Milliseconds()

	if dryRun {
		response.Message = "Dry-run rollout generated. No system changes were applied."
	} else if response.AttemptedUsers == 0 {
		response.Message = "No eligible user found for rollout apply."
	} else if response.Failed > 0 {
		response.Message = "Rollout apply finished with failures."
	} else {
		response.Message = "Rollout apply finished successfully."
	}

	auditStatus := "dry_run"
	if !dryRun {
		auditStatus = "success"
		if response.AttemptedUsers == 0 {
			auditStatus = "no_op"
		} else if response.Failed > 0 {
			auditStatus = "partial_failed"
		}
	}

	s.mu.Lock()
	s.appendCloudLinuxRolloutLocked(cloudLinuxRolloutAuditEntry{
		ID:             response.ID,
		Status:         auditStatus,
		DryRun:         response.DryRun,
		RequestedBy:    response.RequestedBy,
		RequestedAt:    response.RequestedAt,
		FinishedAt:     response.FinishedAt,
		DurationMS:     response.DurationMS,
		PackageFilter:  response.PackageFilter,
		OnlyReady:      response.OnlyReady,
		MaxUsers:       response.MaxUsers,
		RequestedUsers: response.RequestedUsers,
		PlannedUsers:   response.PlannedUsers,
		AttemptedUsers: response.AttemptedUsers,
		Succeeded:      response.Succeeded,
		Failed:         response.Failed,
		Skipped:        response.Skipped,
		Message:        response.Message,
	})
	s.appendActivityLocked(
		requestedBy,
		"cloudlinux_rollout_apply",
		fmt.Sprintf("id=%s dry_run=%t planned=%d attempted=%d success=%d failed=%d skipped=%d", response.ID, response.DryRun, response.PlannedUsers, response.AttemptedUsers, response.Succeeded, response.Failed, response.Skipped),
		r.RemoteAddr,
	)
	s.mu.Unlock()

	writeJSON(w, http.StatusOK, apiResponse{Status: "success", Message: response.Message, Data: response})
}
