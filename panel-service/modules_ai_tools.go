package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	aiProviderDeepSeek = "deepseek"
	aiProviderGemini   = "gemini"

	aiToolSystemScan    = "system_scan"
	aiToolService       = "service_control"
	aiToolMalwareScan   = "malware_scan"
	aiToolShellCommand  = "shell_command"
	aiToolsHistoryLimit = 250
	aiToolsPlanLimit    = 80

	aiEnvProviderActive       = "AURAPANEL_AI_PROVIDER_ACTIVE"
	aiEnvDeepSeekEnabled      = "AURAPANEL_AI_DEEPSEEK_ENABLED"
	aiEnvDeepSeekModel        = "AURAPANEL_AI_DEEPSEEK_MODEL"
	aiEnvDeepSeekBaseURL      = "AURAPANEL_AI_DEEPSEEK_BASE_URL"
	aiEnvDeepSeekAPIKey       = "AURAPANEL_AI_DEEPSEEK_API_KEY"
	aiEnvGeminiEnabled        = "AURAPANEL_AI_GEMINI_ENABLED"
	aiEnvGeminiModel          = "AURAPANEL_AI_GEMINI_MODEL"
	aiEnvGeminiBaseURL        = "AURAPANEL_AI_GEMINI_BASE_URL"
	aiEnvGeminiAPIKey         = "AURAPANEL_AI_GEMINI_API_KEY"
	aiEnvToolsEnabled         = "AURAPANEL_AI_TOOLS_ENABLED"
	aiEnvToolsAllowShell      = "AURAPANEL_AI_TOOLS_ALLOW_SHELL"
	aiEnvToolsAllowPrivileged = "AURAPANEL_AI_TOOLS_ALLOW_PRIVILEGED_SHELL"
	aiEnvToolsAllowService    = "AURAPANEL_AI_TOOLS_ALLOW_SERVICE_CONTROL"
	aiEnvToolsAllowMalware    = "AURAPANEL_AI_TOOLS_ALLOW_MALWARE_SCAN"
	aiEnvToolsRequireConfirm  = "AURAPANEL_AI_TOOLS_REQUIRE_CONFIRM_TOKEN"
	aiEnvToolsConfirmToken    = "AURAPANEL_AI_TOOLS_CONFIRM_TOKEN"
	aiEnvToolsMaxTimeout      = "AURAPANEL_AI_TOOLS_MAX_TIMEOUT"
	aiEnvToolsMaxOutput       = "AURAPANEL_AI_TOOLS_MAX_OUTPUT"
	aiEnvToolsDefaultCWD      = "AURAPANEL_AI_TOOLS_DEFAULT_CWD"
	aiEnvToolsAllowedPrefixes = "AURAPANEL_AI_TOOLS_ALLOWED_PREFIXES"

	aiToolsDefaultConfirmToken = "APPLY_AI_TOOLS"
	aiPlannerTimeoutSeconds    = 25
	aiDefaultPlanSummary       = "Generated AI tools execution plan."
	aiMaxPlannerPromptChars    = 4000
	aiDefaultShellTimeout      = 12
	aiDefaultExecutionShellCWD = "/home"
	aiPlannerMaxCandidateSteps = 8
)

type aiToolCatalogItem struct {
	ID                  string                 `json:"id"`
	Label               string                 `json:"label"`
	Description         string                 `json:"description"`
	Risk                string                 `json:"risk"`
	RequiresConfirm     bool                   `json:"requires_confirm"`
	Enabled             bool                   `json:"enabled"`
	BlockedReason       string                 `json:"blocked_reason,omitempty"`
	PrivilegedSupported bool                   `json:"privileged_supported,omitempty"`
	DefaultArgs         map[string]interface{} `json:"default_args,omitempty"`
}

type aiProviderConfigUpdatePayload struct {
	Enabled     *bool   `json:"enabled"`
	Model       *string `json:"model"`
	BaseURL     *string `json:"base_url"`
	APIKey      *string `json:"api_key"`
	ClearAPIKey bool    `json:"clear_api_key"`
}

type aiProviderUpdatePayload struct {
	ActiveProvider *string                        `json:"active_provider"`
	DeepSeek       *aiProviderConfigUpdatePayload `json:"deepseek"`
	Gemini         *aiProviderConfigUpdatePayload `json:"gemini"`
}

type aiPolicyUpdatePayload struct {
	Enabled                  *bool     `json:"enabled"`
	AllowShell               *bool     `json:"allow_shell"`
	AllowPrivilegedShell     *bool     `json:"allow_privileged_shell"`
	AllowServiceControl      *bool     `json:"allow_service_control"`
	AllowMalwareScan         *bool     `json:"allow_malware_scan"`
	RequireConfirmToken      *bool     `json:"require_confirm_token"`
	ConfirmToken             *string   `json:"confirm_token"`
	MaxCommandTimeoutSeconds *int      `json:"max_command_timeout_seconds"`
	MaxOutputChars           *int      `json:"max_output_chars"`
	DefaultCWD               *string   `json:"default_cwd"`
	AllowedCommandPrefixes   *[]string `json:"allowed_command_prefixes"`
}

type aiPlanPayload struct {
	Prompt string `json:"prompt"`
}

type aiExecutePayload struct {
	PlanID       string                 `json:"plan_id"`
	StepID       string                 `json:"step_id"`
	Tool         string                 `json:"tool"`
	Risk         string                 `json:"risk"`
	Prompt       string                 `json:"prompt"`
	Args         map[string]interface{} `json:"args"`
	DryRun       bool                   `json:"dry_run"`
	ConfirmToken string                 `json:"confirm_token"`
	ExecuteAll   bool                   `json:"execute_all"`
}

type aiPlannerResponse struct {
	Summary string
	Steps   []aiPlannerStep
}

type aiPlannerStep struct {
	Tool            string
	Risk            string
	Reason          string
	RequiresConfirm bool
	Args            map[string]interface{}
}

type aiToolExecutionResult struct {
	Status string
	Output interface{}
	Error  string
}

func (s *service) handleAIToolsStatus(w http.ResponseWriter) {
	s.mu.RLock()
	provider := s.aiProviderRuntimeSnapshotLocked()
	policy := s.aiPolicySnapshotLocked()
	planCount := len(s.modules.AIToolsPlans)
	historyCount := len(s.modules.AIToolsHistory)
	s.mu.RUnlock()

	writeJSON(w, http.StatusOK, apiResponse{
		Status: "success",
		Data: map[string]interface{}{
			"provider": provider,
			"policy":   policy,
			"stats": map[string]int{
				"plan_count":    planCount,
				"history_count": historyCount,
			},
		},
	})
}

func (s *service) handleAIToolsCatalog(w http.ResponseWriter) {
	s.mu.RLock()
	policy := s.aiPolicySnapshotLocked()
	s.mu.RUnlock()

	writeJSON(w, http.StatusOK, apiResponse{
		Status: "success",
		Data: map[string]interface{}{
			"tools": buildAIToolCatalog(policy),
		},
	})
}

func (s *service) handleAIToolsHistory(w http.ResponseWriter, r *http.Request) {
	limit := clampInt(queryInt(r, "limit", 50), 1, aiToolsHistoryLimit)
	s.mu.RLock()
	history := append([]AIToolExecutionRecord(nil), s.modules.AIToolsHistory...)
	s.mu.RUnlock()
	if len(history) > limit {
		history = history[:limit]
	}
	writeJSON(w, http.StatusOK, apiResponse{Status: "success", Data: history})
}

func (s *service) handleAIToolsProviderSet(w http.ResponseWriter, r *http.Request) {
	var payload aiProviderUpdatePayload
	if err := decodeJSON(r, &payload); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid AI provider payload.")
		return
	}

	s.mu.RLock()
	current := s.aiProviderRuntimeSnapshotLocked()
	s.mu.RUnlock()

	deepSeekAPIKey := strings.TrimSpace(aiRuntimeEnvValue(aiEnvDeepSeekAPIKey))
	geminiAPIKey := strings.TrimSpace(aiRuntimeEnvValue(aiEnvGeminiAPIKey))

	next := current
	if payload.ActiveProvider != nil {
		next.ActiveProvider = normalizeAIToolProvider(*payload.ActiveProvider)
	}
	next.DeepSeek, deepSeekAPIKey = applyAIProviderPatch(aiProviderDeepSeek, next.DeepSeek, payload.DeepSeek, deepSeekAPIKey)
	next.Gemini, geminiAPIKey = applyAIProviderPatch(aiProviderGemini, next.Gemini, payload.Gemini, geminiAPIKey)
	next.UpdatedAt = time.Now().UTC().Unix()
	next = normalizeAIProviderRuntime(next, deepSeekAPIKey, geminiAPIKey)

	updates := map[string]string{
		aiEnvProviderActive:  next.ActiveProvider,
		aiEnvDeepSeekEnabled: boolToEnvValue(next.DeepSeek.Enabled),
		aiEnvDeepSeekModel:   next.DeepSeek.Model,
		aiEnvDeepSeekBaseURL: next.DeepSeek.BaseURL,
		aiEnvDeepSeekAPIKey:  deepSeekAPIKey,
		aiEnvGeminiEnabled:   boolToEnvValue(next.Gemini.Enabled),
		aiEnvGeminiModel:     next.Gemini.Model,
		aiEnvGeminiBaseURL:   next.Gemini.BaseURL,
		aiEnvGeminiAPIKey:    geminiAPIKey,
	}
	if err := persistAIRuntimeEnv(updates); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("AI provider settings could not be persisted: %v", err))
		return
	}

	principal, _ := principalFromContext(r.Context())
	requestedBy := principalRequestIdentity(principal)

	s.mu.Lock()
	s.modules.AIToolsProvider = next
	s.appendActivityLocked(requestedBy, "ai_provider_update", fmt.Sprintf("active=%s deepseek=%t gemini=%t", next.ActiveProvider, next.DeepSeek.Enabled, next.Gemini.Enabled), r.RemoteAddr)
	s.mu.Unlock()

	writeJSON(w, http.StatusOK, apiResponse{
		Status:  "success",
		Message: "AI provider settings saved.",
		Data:    next,
	})
}

func (s *service) handleAIToolsPolicySet(w http.ResponseWriter, r *http.Request) {
	var payload aiPolicyUpdatePayload
	if err := decodeJSON(r, &payload); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid AI policy payload.")
		return
	}

	s.mu.RLock()
	current := s.aiPolicySnapshotLocked()
	s.mu.RUnlock()

	next := current
	if payload.Enabled != nil {
		next.Enabled = *payload.Enabled
	}
	if payload.AllowShell != nil {
		next.AllowShell = *payload.AllowShell
	}
	if payload.AllowPrivilegedShell != nil {
		next.AllowPrivilegedShell = *payload.AllowPrivilegedShell
	}
	if payload.AllowServiceControl != nil {
		next.AllowServiceControl = *payload.AllowServiceControl
	}
	if payload.AllowMalwareScan != nil {
		next.AllowMalwareScan = *payload.AllowMalwareScan
	}
	if payload.RequireConfirmToken != nil {
		next.RequireConfirmToken = *payload.RequireConfirmToken
	}
	if payload.ConfirmToken != nil {
		next.ConfirmToken = strings.TrimSpace(*payload.ConfirmToken)
	}
	if payload.MaxCommandTimeoutSeconds != nil {
		next.MaxCommandTimeoutSeconds = *payload.MaxCommandTimeoutSeconds
	}
	if payload.MaxOutputChars != nil {
		next.MaxOutputChars = *payload.MaxOutputChars
	}
	if payload.DefaultCWD != nil {
		next.DefaultCWD = strings.TrimSpace(*payload.DefaultCWD)
	}
	if payload.AllowedCommandPrefixes != nil {
		next.AllowedCommandPrefixes = append([]string(nil), (*payload.AllowedCommandPrefixes)...)
	}

	next = normalizeAIToolsPolicy(next)

	updates := map[string]string{
		aiEnvToolsEnabled:         boolToEnvValue(next.Enabled),
		aiEnvToolsAllowShell:      boolToEnvValue(next.AllowShell),
		aiEnvToolsAllowPrivileged: boolToEnvValue(next.AllowPrivilegedShell),
		aiEnvToolsAllowService:    boolToEnvValue(next.AllowServiceControl),
		aiEnvToolsAllowMalware:    boolToEnvValue(next.AllowMalwareScan),
		aiEnvToolsRequireConfirm:  boolToEnvValue(next.RequireConfirmToken),
		aiEnvToolsConfirmToken:    next.ConfirmToken,
		aiEnvToolsMaxTimeout:      strconv.Itoa(next.MaxCommandTimeoutSeconds),
		aiEnvToolsMaxOutput:       strconv.Itoa(next.MaxOutputChars),
		aiEnvToolsDefaultCWD:      next.DefaultCWD,
		aiEnvToolsAllowedPrefixes: strings.Join(next.AllowedCommandPrefixes, ","),
	}
	if err := persistAIRuntimeEnv(updates); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("AI policy could not be persisted: %v", err))
		return
	}

	principal, _ := principalFromContext(r.Context())
	requestedBy := principalRequestIdentity(principal)
	s.mu.Lock()
	s.modules.AIToolsPolicy = next
	s.appendActivityLocked(requestedBy, "ai_policy_update", fmt.Sprintf("enabled=%t shell=%t privileged_shell=%t", next.Enabled, next.AllowShell, next.AllowPrivilegedShell), r.RemoteAddr)
	s.mu.Unlock()

	writeJSON(w, http.StatusOK, apiResponse{
		Status:  "success",
		Message: "AI policy saved.",
		Data:    next,
	})
}

func (s *service) handleAIToolsPlan(w http.ResponseWriter, r *http.Request) {
	var payload aiPlanPayload
	if err := decodeJSON(r, &payload); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid AI planner payload.")
		return
	}
	prompt := strings.TrimSpace(payload.Prompt)
	if prompt == "" {
		writeError(w, http.StatusBadRequest, "prompt is required.")
		return
	}
	if len(prompt) > aiMaxPlannerPromptChars {
		prompt = prompt[:aiMaxPlannerPromptChars]
	}

	s.mu.RLock()
	provider := s.aiProviderRuntimeSnapshotLocked()
	policy := s.aiPolicySnapshotLocked()
	s.mu.RUnlock()
	if !policy.Enabled {
		writeError(w, http.StatusForbidden, "AI tools are disabled by policy.")
		return
	}

	catalog := buildAIToolCatalog(policy)
	plan, fallbackReason, err := s.generateAIToolPlan(prompt, provider, policy, catalog)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	principal, _ := principalFromContext(r.Context())
	requestedBy := principalRequestIdentity(principal)

	s.mu.Lock()
	s.appendAIToolPlanLocked(plan)
	s.appendActivityLocked(requestedBy, "ai_plan_create", fmt.Sprintf("provider=%s steps=%d", plan.Provider, len(plan.Steps)), r.RemoteAddr)
	s.mu.Unlock()

	data := map[string]interface{}{
		"plan": plan,
	}
	if fallbackReason != "" {
		data["fallback_reason"] = fallbackReason
	}

	writeJSON(w, http.StatusOK, apiResponse{
		Status:  "success",
		Message: "AI plan generated.",
		Data:    data,
	})
}

func (s *service) handleAIToolsExecute(w http.ResponseWriter, r *http.Request) {
	var payload aiExecutePayload
	if err := decodeJSON(r, &payload); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid AI execution payload.")
		return
	}

	s.mu.RLock()
	policy := s.aiPolicySnapshotLocked()
	plans := append([]AIToolPlan(nil), s.modules.AIToolsPlans...)
	s.mu.RUnlock()

	if !policy.Enabled {
		writeError(w, http.StatusForbidden, "AI tools are disabled by policy.")
		return
	}

	principal, _ := principalFromContext(r.Context())
	requestedBy := principalRequestIdentity(principal)

	planID := strings.TrimSpace(payload.PlanID)
	stepID := strings.TrimSpace(payload.StepID)
	if payload.ExecuteAll && planID == "" {
		writeError(w, http.StatusBadRequest, "plan_id is required when execute_all is true.")
		return
	}

	if payload.ExecuteAll {
		plan, ok := findAIToolPlan(plans, planID)
		if !ok {
			writeError(w, http.StatusNotFound, "AI plan not found.")
			return
		}
		results := make([]AIToolExecutionRecord, 0, len(plan.Steps))
		for _, step := range plan.Steps {
			record := s.executeAIToolStep(plan, step, payload.DryRun, payload.ConfirmToken, policy, requestedBy, r.RemoteAddr)
			results = append(results, record)
		}
		sort.Slice(results, func(i, j int) bool {
			return results[i].RequestedAt < results[j].RequestedAt
		})

		successCount := 0
		failedCount := 0
		blockedCount := 0
		for _, item := range results {
			switch strings.ToLower(item.Status) {
			case "success", "dry_run":
				successCount++
			case "blocked":
				blockedCount++
			default:
				failedCount++
			}
		}

		writeJSON(w, http.StatusOK, apiResponse{
			Status:  "success",
			Message: "AI plan execution completed.",
			Data: map[string]interface{}{
				"plan_id":   plan.ID,
				"results":   results,
				"succeeded": successCount,
				"failed":    failedCount,
				"blocked":   blockedCount,
			},
		})
		return
	}

	if planID != "" {
		plan, ok := findAIToolPlan(plans, planID)
		if !ok {
			writeError(w, http.StatusNotFound, "AI plan not found.")
			return
		}
		step, ok := pickAIToolStep(plan, stepID)
		if !ok {
			writeError(w, http.StatusNotFound, "AI plan step not found.")
			return
		}
		record := s.executeAIToolStep(plan, step, payload.DryRun, payload.ConfirmToken, policy, requestedBy, r.RemoteAddr)
		writeJSON(w, http.StatusOK, apiResponse{
			Status:  "success",
			Message: aiExecutionStatusMessage(record.Status),
			Data:    record,
		})
		return
	}

	tool := normalizeAIToolName(payload.Tool)
	if tool == "" {
		writeError(w, http.StatusBadRequest, "tool is required when plan_id is not provided.")
		return
	}
	args := cloneInterfaceMap(payload.Args)
	risk := normalizeAIRisk(payload.Risk)
	if risk == "" {
		risk = defaultRiskForAITool(tool)
	}
	if tool == aiToolShellCommand {
		risk = maxAIRisk(risk, estimateShellRisk(stringValue(args["command"])))
	}
	requiresConfirm := aiToolDefaultRequiresConfirm(tool, risk)
	step := AIToolPlanStep{
		ID:              generateSecret(5),
		Tool:            tool,
		Risk:            risk,
		Reason:          firstNonEmpty(strings.TrimSpace(payload.Prompt), "Manual execution"),
		RequiresConfirm: requiresConfirm,
		Args:            args,
	}
	plan := AIToolPlan{
		ID:        "",
		Prompt:    strings.TrimSpace(payload.Prompt),
		Provider:  "manual",
		Model:     "",
		Summary:   "Manual execution",
		CreatedAt: time.Now().UTC().Unix(),
		Steps:     []AIToolPlanStep{step},
	}
	record := s.executeAIToolStep(plan, step, payload.DryRun, payload.ConfirmToken, policy, requestedBy, r.RemoteAddr)
	writeJSON(w, http.StatusOK, apiResponse{
		Status:  "success",
		Message: aiExecutionStatusMessage(record.Status),
		Data:    record,
	})
}

func (s *service) executeAIToolStep(plan AIToolPlan, step AIToolPlanStep, dryRun bool, confirmToken string, policy AIToolsPolicy, requestedBy, remoteAddr string) AIToolExecutionRecord {
	start := time.Now().UTC()
	record := AIToolExecutionRecord{
		ID:          generateSecret(8),
		PlanID:      strings.TrimSpace(plan.ID),
		Prompt:      strings.TrimSpace(plan.Prompt),
		Tool:        normalizeAIToolName(step.Tool),
		Risk:        normalizeAIRisk(step.Risk),
		Status:      "running",
		DryRun:      dryRun,
		RequestedBy: firstNonEmpty(strings.TrimSpace(requestedBy), "system"),
		RequestedAt: start.Unix(),
		Args:        cloneInterfaceMap(step.Args),
	}
	if record.Tool == "" {
		record.Tool = aiToolSystemScan
	}
	if record.Risk == "" {
		record.Risk = defaultRiskForAITool(record.Tool)
	}

	if !isAIToolEnabledByPolicy(record.Tool, policy) {
		record.Status = "blocked"
		record.Error = aiToolBlockedReason(record.Tool, policy)
	} else if !dryRun && policy.RequireConfirmToken && step.RequiresConfirm {
		if strings.TrimSpace(confirmToken) == "" || strings.TrimSpace(confirmToken) != strings.TrimSpace(policy.ConfirmToken) {
			record.Status = "blocked"
			record.Error = "Confirmation token is required for this operation."
		}
	}

	if record.Status == "running" {
		result := s.runAITool(record.Tool, record.Args, policy, dryRun)
		record.Status = firstNonEmpty(result.Status, "failed")
		record.Output = result.Output
		record.Error = result.Error
	}

	finished := time.Now().UTC()
	record.FinishedAt = finished.Unix()
	record.DurationMS = finished.Sub(start).Milliseconds()

	s.mu.Lock()
	s.appendAIToolHistoryLocked(record)
	detail := fmt.Sprintf("tool=%s status=%s dry_run=%t", record.Tool, record.Status, record.DryRun)
	if record.PlanID != "" {
		detail += fmt.Sprintf(" plan=%s", record.PlanID)
	}
	s.appendActivityLocked(record.RequestedBy, "ai_tool_execution", detail, remoteAddr)
	s.mu.Unlock()
	return record
}

func (s *service) runAITool(tool string, args map[string]interface{}, policy AIToolsPolicy, dryRun bool) aiToolExecutionResult {
	normalizedTool := normalizeAIToolName(tool)
	if normalizedTool == "" {
		return aiToolExecutionResult{Status: "failed", Error: "Unknown tool requested."}
	}

	switch normalizedTool {
	case aiToolSystemScan:
		result := map[string]interface{}{
			"metrics":       collectHostMetrics(s.startedAt),
			"security":      collectSecuritySnapshot(),
			"services":      collectHostServices(),
			"top_processes": collectHostProcesses(12),
			"generated_at":  time.Now().UTC().Format(time.RFC3339),
		}
		if dryRun {
			return aiToolExecutionResult{
				Status: "dry_run",
				Output: map[string]interface{}{
					"message": "Dry-run mode: system scan would collect host telemetry.",
					"preview": result,
				},
			}
		}
		return aiToolExecutionResult{Status: "success", Output: result}

	case aiToolService:
		if !policy.AllowServiceControl {
			return aiToolExecutionResult{Status: "blocked", Error: aiToolBlockedReason(aiToolService, policy)}
		}
		name := strings.TrimSpace(stringValue(args["name"]))
		if name == "" {
			name = "openlitespeed"
		}
		action := strings.ToLower(strings.TrimSpace(stringValue(args["action"])))
		if action == "" {
			action = "restart"
		}
		switch action {
		case "start", "restart", "stop":
		default:
			return aiToolExecutionResult{Status: "failed", Error: "service_control only supports start, restart or stop."}
		}
		unit, supported := serviceUnitName(name)
		if !supported {
			return aiToolExecutionResult{Status: "failed", Error: "Requested service is not supported by control plane."}
		}
		if dryRun {
			return aiToolExecutionResult{
				Status: "dry_run",
				Output: map[string]interface{}{
					"name":    name,
					"action":  action,
					"unit":    unit,
					"message": "Dry-run mode: service action was not executed.",
				},
			}
		}
		scheduled, err := executeServiceActionFromPanel(name, action)
		if err != nil {
			return aiToolExecutionResult{
				Status: "failed",
				Error:  err.Error(),
				Output: map[string]interface{}{"name": name, "action": action, "unit": unit},
			}
		}
		message := "Service action applied."
		if scheduled {
			message = "Service action scheduled."
		}
		return aiToolExecutionResult{
			Status: "success",
			Output: map[string]interface{}{
				"name":      name,
				"action":    action,
				"unit":      unit,
				"scheduled": scheduled,
				"message":   message,
			},
		}

	case aiToolMalwareScan:
		if !policy.AllowMalwareScan {
			return aiToolExecutionResult{Status: "blocked", Error: aiToolBlockedReason(aiToolMalwareScan, policy)}
		}
		targetPath := strings.TrimSpace(stringValue(args["path"]))
		if targetPath == "" {
			targetPath = firstNonEmpty(strings.TrimSpace(policy.DefaultCWD), aiDefaultExecutionShellCWD)
		}
		engine := strings.TrimSpace(stringValue(args["engine"]))
		if dryRun {
			return aiToolExecutionResult{
				Status: "dry_run",
				Output: map[string]interface{}{
					"path":    targetPath,
					"engine":  firstNonEmpty(engine, "auto"),
					"message": "Dry-run mode: malware scan was not executed.",
				},
			}
		}
		job, err := runRuntimeMalwareScan(targetPath, engine)
		if err != nil {
			return aiToolExecutionResult{Status: "failed", Error: err.Error()}
		}
		s.mu.Lock()
		s.state.MalwareJobs = append([]MalwareJob{job}, s.state.MalwareJobs...)
		if len(s.state.MalwareJobs) > 60 {
			s.state.MalwareJobs = s.state.MalwareJobs[:60]
		}
		s.mu.Unlock()
		return aiToolExecutionResult{Status: "success", Output: map[string]interface{}{"job": job, "message": "Malware scan completed."}}

	case aiToolShellCommand:
		if !policy.AllowShell {
			return aiToolExecutionResult{Status: "blocked", Error: aiToolBlockedReason(aiToolShellCommand, policy)}
		}
		command := strings.TrimSpace(stringValue(args["command"]))
		if command == "" {
			return aiToolExecutionResult{Status: "failed", Error: "shell_command requires a command argument."}
		}
		if !isCommandAllowedByPolicy(command, policy) {
			return aiToolExecutionResult{Status: "blocked", Error: "Command prefix is not allowed by AI policy."}
		}
		privileged := flexibleBool(args["privileged"])
		if privileged && !policy.AllowPrivilegedShell {
			return aiToolExecutionResult{Status: "blocked", Error: "Privileged shell is disabled by policy."}
		}
		cwd := strings.TrimSpace(stringValue(args["cwd"]))
		if cwd == "" {
			cwd = firstNonEmpty(strings.TrimSpace(policy.DefaultCWD), aiDefaultExecutionShellCWD)
		}
		timeoutSeconds := flexibleInt(args["timeout_seconds"], aiDefaultShellTimeout)
		timeoutSeconds = clampInt(timeoutSeconds, 2, clampInt(policy.MaxCommandTimeoutSeconds, 2, 120))

		if dryRun {
			return aiToolExecutionResult{
				Status: "dry_run",
				Output: map[string]interface{}{
					"command":         command,
					"cwd":             cwd,
					"privileged":      privileged,
					"timeout_seconds": timeoutSeconds,
					"message":         "Dry-run mode: shell command was not executed.",
				},
			}
		}

		if privileged {
			rawOutput, err := runPrivilegedShellCommandWithTimeout(command, cwd, timeoutSeconds)
			output := trimAndClampOutput(rawOutput, policy.MaxOutputChars)
			if err != nil {
				return aiToolExecutionResult{
					Status: "failed",
					Error:  err.Error(),
					Output: map[string]interface{}{"command": command, "cwd": cwd, "privileged": true, "timeout_seconds": timeoutSeconds, "output": output},
				}
			}
			return aiToolExecutionResult{
				Status: "success",
				Output: map[string]interface{}{"command": command, "cwd": cwd, "privileged": true, "timeout_seconds": timeoutSeconds, "output": output},
			}
		}

		rawOutput, nextCwd, err := runManagedShellCommandWithTimeout(command, cwd, timeoutSeconds)
		output := trimAndClampOutput(rawOutput, policy.MaxOutputChars)
		if err != nil {
			return aiToolExecutionResult{
				Status: "failed",
				Error:  err.Error(),
				Output: map[string]interface{}{"command": command, "cwd": cwd, "next_cwd": nextCwd, "privileged": false, "timeout_seconds": timeoutSeconds, "output": output},
			}
		}
		return aiToolExecutionResult{
			Status: "success",
			Output: map[string]interface{}{"command": command, "cwd": cwd, "next_cwd": nextCwd, "privileged": false, "timeout_seconds": timeoutSeconds, "output": output},
		}
	}

	return aiToolExecutionResult{Status: "failed", Error: "Unhandled tool."}
}

func (s *service) generateAIToolPlan(prompt string, provider AIToolsProviderRuntime, policy AIToolsPolicy, catalog []aiToolCatalogItem) (AIToolPlan, string, error) {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return AIToolPlan{}, "", fmt.Errorf("prompt is required")
	}

	active := normalizeAIToolProvider(provider.ActiveProvider)
	activeConfig := providerConfigByName(provider, active)
	activeKey := strings.TrimSpace(aiAPIKeyForProvider(active))

	if activeConfig.Enabled && activeKey != "" {
		plannerResponse, err := requestAIPlanner(active, activeConfig, activeKey, prompt, policy, catalog)
		if err == nil {
			plan, normalizeErr := normalizePlannerPlan(plannerResponse, prompt, active, activeConfig.Model, policy, catalog)
			if normalizeErr == nil {
				return plan, "", nil
			}
			heuristic := heuristicAIToolPlan(prompt, active, activeConfig.Model, policy, catalog)
			if len(heuristic.Steps) > 0 {
				return heuristic, fmt.Sprintf("Provider response normalization failed: %v", normalizeErr), nil
			}
			return AIToolPlan{}, "", normalizeErr
		}
		heuristic := heuristicAIToolPlan(prompt, active, activeConfig.Model, policy, catalog)
		if len(heuristic.Steps) > 0 {
			return heuristic, fmt.Sprintf("Provider plan request failed: %v", err), nil
		}
		return AIToolPlan{}, "", err
	}

	fallbackReason := "Active provider is disabled or missing API key."
	heuristic := heuristicAIToolPlan(prompt, active, activeConfig.Model, policy, catalog)
	if len(heuristic.Steps) == 0 {
		return AIToolPlan{}, fallbackReason, fmt.Errorf("no available tools can satisfy this request under current policy")
	}
	return heuristic, fallbackReason, nil
}

func requestAIPlanner(provider string, config AIToolsProviderConfig, apiKey, prompt string, policy AIToolsPolicy, catalog []aiToolCatalogItem) (aiPlannerResponse, error) {
	switch normalizeAIToolProvider(provider) {
	case aiProviderGemini:
		return requestGeminiPlanner(config, apiKey, prompt, policy, catalog)
	default:
		return requestDeepSeekPlanner(config, apiKey, prompt, policy, catalog)
	}
}

func requestDeepSeekPlanner(config AIToolsProviderConfig, apiKey, prompt string, policy AIToolsPolicy, catalog []aiToolCatalogItem) (aiPlannerResponse, error) {
	baseURL := strings.TrimSuffix(strings.TrimSpace(config.BaseURL), "/")
	if baseURL == "" {
		baseURL = "https://api.deepseek.com/v1"
	}
	model := firstNonEmpty(strings.TrimSpace(config.Model), "deepseek-chat")
	endpoint := baseURL + "/chat/completions"

	systemPrompt := aiPlannerSystemPrompt(policy, catalog)
	userPrompt := aiPlannerUserPrompt(prompt, catalog)
	body := map[string]interface{}{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userPrompt},
		},
		"temperature": 0.1,
		"response_format": map[string]string{
			"type": "json_object",
		},
	}
	rawBody, err := json.Marshal(body)
	if err != nil {
		return aiPlannerResponse{}, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), aiPlannerTimeoutSeconds*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(rawBody))
	if err != nil {
		return aiPlannerResponse{}, err
	}
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(apiKey))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: aiPlannerTimeoutSeconds * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return aiPlannerResponse{}, err
	}
	defer resp.Body.Close()
	responseBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= http.StatusBadRequest {
		return aiPlannerResponse{}, fmt.Errorf("provider status=%d body=%s", resp.StatusCode, trimAndClampOutput(string(responseBody), 500))
	}

	var decoded struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(responseBody, &decoded); err != nil {
		return aiPlannerResponse{}, err
	}
	if strings.TrimSpace(decoded.Error.Message) != "" {
		return aiPlannerResponse{}, fmt.Errorf(decoded.Error.Message)
	}
	if len(decoded.Choices) == 0 {
		return aiPlannerResponse{}, fmt.Errorf("provider returned no planner choices")
	}
	content := strings.TrimSpace(decoded.Choices[0].Message.Content)
	if content == "" {
		return aiPlannerResponse{}, fmt.Errorf("provider returned empty planner content")
	}
	return parsePlannerResponse(content)
}

func requestGeminiPlanner(config AIToolsProviderConfig, apiKey, prompt string, policy AIToolsPolicy, catalog []aiToolCatalogItem) (aiPlannerResponse, error) {
	baseURL := strings.TrimSuffix(strings.TrimSpace(config.BaseURL), "/")
	if baseURL == "" {
		baseURL = "https://generativelanguage.googleapis.com/v1beta"
	}
	model := firstNonEmpty(strings.TrimSpace(config.Model), "gemini-2.5-flash")
	endpoint := fmt.Sprintf("%s/models/%s:generateContent?key=%s", baseURL, url.PathEscape(model), url.QueryEscape(strings.TrimSpace(apiKey)))

	systemPrompt := aiPlannerSystemPrompt(policy, catalog)
	userPrompt := aiPlannerUserPrompt(prompt, catalog)
	body := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]string{
					{"text": systemPrompt + "\n\n" + userPrompt},
				},
			},
		},
		"generationConfig": map[string]interface{}{
			"temperature": 0.1,
		},
	}

	rawBody, err := json.Marshal(body)
	if err != nil {
		return aiPlannerResponse{}, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), aiPlannerTimeoutSeconds*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(rawBody))
	if err != nil {
		return aiPlannerResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: aiPlannerTimeoutSeconds * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return aiPlannerResponse{}, err
	}
	defer resp.Body.Close()
	responseBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= http.StatusBadRequest {
		return aiPlannerResponse{}, fmt.Errorf("provider status=%d body=%s", resp.StatusCode, trimAndClampOutput(string(responseBody), 500))
	}

	var decoded struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(responseBody, &decoded); err != nil {
		return aiPlannerResponse{}, err
	}
	if strings.TrimSpace(decoded.Error.Message) != "" {
		return aiPlannerResponse{}, fmt.Errorf(decoded.Error.Message)
	}
	if len(decoded.Candidates) == 0 || len(decoded.Candidates[0].Content.Parts) == 0 {
		return aiPlannerResponse{}, fmt.Errorf("provider returned no planner candidates")
	}
	content := strings.TrimSpace(decoded.Candidates[0].Content.Parts[0].Text)
	if content == "" {
		return aiPlannerResponse{}, fmt.Errorf("provider returned empty planner content")
	}
	return parsePlannerResponse(content)
}

func aiPlannerSystemPrompt(policy AIToolsPolicy, catalog []aiToolCatalogItem) string {
	allowed := []string{}
	for _, item := range catalog {
		if item.Enabled {
			allowed = append(allowed, item.ID)
		}
	}
	if len(allowed) == 0 {
		allowed = []string{aiToolSystemScan}
	}
	prefixes := strings.Join(policy.AllowedCommandPrefixes, ", ")
	if prefixes == "" {
		prefixes = "(no restriction configured)"
	}
	return "You are AuraPanel AI planner. Return strict JSON only with this shape: {\"summary\":\"...\",\"steps\":[{\"tool\":\"...\",\"risk\":\"low|medium|high|critical\",\"reason\":\"...\",\"requires_confirm\":true|false,\"args\":{...}}]}. " +
		"Allowed tools: " + strings.Join(allowed, ", ") + ". " +
		"Do not include markdown fences. " +
		"Prefer safe read-only operations first. " +
		"Policy: allow_shell=" + strconv.FormatBool(policy.AllowShell) +
		", allow_privileged_shell=" + strconv.FormatBool(policy.AllowPrivilegedShell) +
		", allow_service_control=" + strconv.FormatBool(policy.AllowServiceControl) +
		", allow_malware_scan=" + strconv.FormatBool(policy.AllowMalwareScan) +
		", shell_prefix_allowlist=" + prefixes + ". " +
		"Maximum steps: " + strconv.Itoa(aiPlannerMaxCandidateSteps) + "."
}

func aiPlannerUserPrompt(prompt string, catalog []aiToolCatalogItem) string {
	type plannerCatalogRow struct {
		ID          string                 `json:"id"`
		Description string                 `json:"description"`
		Risk        string                 `json:"risk"`
		DefaultArgs map[string]interface{} `json:"default_args,omitempty"`
	}
	rows := make([]plannerCatalogRow, 0, len(catalog))
	for _, item := range catalog {
		if !item.Enabled {
			continue
		}
		rows = append(rows, plannerCatalogRow{
			ID:          item.ID,
			Description: item.Description,
			Risk:        item.Risk,
			DefaultArgs: item.DefaultArgs,
		})
	}
	rawCatalog, _ := json.Marshal(rows)
	return "Operator request:\n" + strings.TrimSpace(prompt) + "\n\nAvailable tool catalog JSON:\n" + string(rawCatalog)
}

func parsePlannerResponse(raw string) (aiPlannerResponse, error) {
	normalized := normalizePlannerPayloadText(raw)
	if normalized == "" {
		return aiPlannerResponse{}, fmt.Errorf("planner response is empty")
	}

	var doc map[string]interface{}
	if err := json.Unmarshal([]byte(normalized), &doc); err != nil {
		return aiPlannerResponse{}, err
	}
	if nestedPlan, ok := doc["plan"].(map[string]interface{}); ok {
		doc = nestedPlan
	}

	response := aiPlannerResponse{
		Summary: strings.TrimSpace(stringValue(doc["summary"])),
		Steps:   []aiPlannerStep{},
	}
	rawSteps := []interface{}{}
	switch typed := doc["steps"].(type) {
	case []interface{}:
		rawSteps = typed
	case map[string]interface{}:
		rawSteps = append(rawSteps, typed)
	}

	for _, rawStep := range rawSteps {
		stepMap, ok := rawStep.(map[string]interface{})
		if !ok {
			continue
		}
		tool := strings.TrimSpace(firstNonEmpty(stringValue(stepMap["tool"]), stringValue(stepMap["id"])))
		risk := strings.TrimSpace(firstNonEmpty(stringValue(stepMap["risk"]), stringValue(stepMap["severity"]), "medium"))
		reason := strings.TrimSpace(firstNonEmpty(stringValue(stepMap["reason"]), stringValue(stepMap["description"]), "Requested by operator prompt."))
		requiresConfirm := flexibleBool(stepMap["requires_confirm"])

		args := map[string]interface{}{}
		if argMap, ok := stepMap["args"].(map[string]interface{}); ok {
			args = cloneInterfaceMap(argMap)
		}
		if command := strings.TrimSpace(stringValue(stepMap["command"])); command != "" {
			args["command"] = command
		}
		if path := strings.TrimSpace(stringValue(stepMap["path"])); path != "" {
			args["path"] = path
		}

		response.Steps = append(response.Steps, aiPlannerStep{
			Tool:            tool,
			Risk:            risk,
			Reason:          reason,
			RequiresConfirm: requiresConfirm,
			Args:            args,
		})
	}

	return response, nil
}

func normalizePlannerPayloadText(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	if strings.HasPrefix(trimmed, "```") {
		lines := strings.Split(trimmed, "\n")
		if len(lines) >= 2 {
			lines = lines[1:]
		}
		if len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "```" {
			lines = lines[:len(lines)-1]
		}
		trimmed = strings.TrimSpace(strings.Join(lines, "\n"))
	}
	if json.Valid([]byte(trimmed)) {
		return trimmed
	}
	start := strings.Index(trimmed, "{")
	end := strings.LastIndex(trimmed, "}")
	if start >= 0 && end > start {
		candidate := strings.TrimSpace(trimmed[start : end+1])
		if json.Valid([]byte(candidate)) {
			return candidate
		}
	}
	return trimmed
}

func normalizePlannerPlan(planner aiPlannerResponse, prompt, provider, model string, policy AIToolsPolicy, catalog []aiToolCatalogItem) (AIToolPlan, error) {
	enabledByTool := map[string]bool{}
	for _, item := range catalog {
		enabledByTool[item.ID] = item.Enabled
	}

	steps := make([]AIToolPlanStep, 0, len(planner.Steps))
	for _, rawStep := range planner.Steps {
		tool := normalizeAIToolName(rawStep.Tool)
		if tool == "" || !enabledByTool[tool] {
			continue
		}
		args := normalizeAIToolArgs(tool, rawStep.Args, policy)
		if tool == aiToolShellCommand && strings.TrimSpace(stringValue(args["command"])) == "" {
			continue
		}
		risk := normalizeAIRisk(rawStep.Risk)
		if risk == "" {
			risk = defaultRiskForAITool(tool)
		}
		if tool == aiToolShellCommand {
			risk = maxAIRisk(risk, estimateShellRisk(stringValue(args["command"])))
		}
		requiresConfirm := rawStep.RequiresConfirm || aiToolDefaultRequiresConfirm(tool, risk)

		steps = append(steps, AIToolPlanStep{
			ID:              generateSecret(5),
			Tool:            tool,
			Risk:            risk,
			Reason:          firstNonEmpty(strings.TrimSpace(rawStep.Reason), "Generated by AI planner"),
			RequiresConfirm: requiresConfirm,
			Args:            args,
		})
		if len(steps) >= aiPlannerMaxCandidateSteps {
			break
		}
	}
	if len(steps) == 0 {
		return AIToolPlan{}, fmt.Errorf("planner returned no executable steps")
	}

	return AIToolPlan{
		ID:        generateSecret(8),
		Prompt:    strings.TrimSpace(prompt),
		Provider:  normalizeAIToolProvider(provider),
		Model:     strings.TrimSpace(model),
		Summary:   firstNonEmpty(strings.TrimSpace(planner.Summary), aiDefaultPlanSummary),
		CreatedAt: time.Now().UTC().Unix(),
		Steps:     steps,
	}, nil
}

func heuristicAIToolPlan(prompt, provider, model string, policy AIToolsPolicy, catalog []aiToolCatalogItem) AIToolPlan {
	enabledByTool := map[string]bool{}
	for _, item := range catalog {
		enabledByTool[item.ID] = item.Enabled
	}
	lower := strings.ToLower(strings.TrimSpace(prompt))
	steps := []AIToolPlanStep{}
	stepAdded := map[string]bool{}

	addStep := func(tool, risk, reason string, requiresConfirm bool, args map[string]interface{}) {
		tool = normalizeAIToolName(tool)
		if tool == "" || stepAdded[tool] || !enabledByTool[tool] {
			return
		}
		risk = normalizeAIRisk(risk)
		if risk == "" {
			risk = defaultRiskForAITool(tool)
		}
		steps = append(steps, AIToolPlanStep{
			ID:              generateSecret(5),
			Tool:            tool,
			Risk:            risk,
			Reason:          firstNonEmpty(strings.TrimSpace(reason), "Generated by fallback heuristic planner"),
			RequiresConfirm: requiresConfirm || aiToolDefaultRequiresConfirm(tool, risk),
			Args:            normalizeAIToolArgs(tool, args, policy),
		})
		stepAdded[tool] = true
	}

	if containsAny(lower, []string{"scan", "tarama", "health", "kontrol", "status", "diagnostic"}) {
		addStep(aiToolSystemScan, "low", "Collect host and security telemetry for diagnostics.", false, map[string]interface{}{"mode": "quick"})
	}
	if containsAny(lower, []string{"malware", "virus", "virüs", "clam", "zararlı"}) {
		addStep(aiToolMalwareScan, "medium", "Run malware scan on managed paths.", false, map[string]interface{}{"path": "/home", "engine": "auto"})
	}
	if containsAny(lower, []string{"restart", "yeniden", "service", "servis", "start", "stop", "systemctl"}) {
		addStep(aiToolService, "high", "Apply requested service control action.", true, map[string]interface{}{"name": "openlitespeed", "action": "restart"})
	}
	if containsAny(lower, []string{"shell", "komut", "command", "log", "grep", "tail", "disk", "cpu", "ram"}) {
		command := "uptime && df -h && free -m"
		if containsAny(lower, []string{"log", "tail"}) {
			command = "tail -n 120 /var/log/aurapanel/panel-service.log"
		}
		addStep(aiToolShellCommand, "medium", "Run shell diagnostics in managed paths.", false, map[string]interface{}{"command": command, "cwd": "/home", "timeout_seconds": 12})
	}
	if len(steps) == 0 {
		if enabledByTool[aiToolSystemScan] {
			addStep(aiToolSystemScan, "low", "Default diagnostics plan generated from prompt.", false, map[string]interface{}{"mode": "quick"})
		} else if enabledByTool[aiToolShellCommand] {
			addStep(aiToolShellCommand, "low", "Default shell health check.", false, map[string]interface{}{"command": "uptime", "cwd": "/home", "timeout_seconds": 8})
		} else if enabledByTool[aiToolService] {
			addStep(aiToolService, "high", "Fallback service control step.", true, map[string]interface{}{"name": "openlitespeed", "action": "restart"})
		}
	}

	return AIToolPlan{
		ID:        generateSecret(8),
		Prompt:    strings.TrimSpace(prompt),
		Provider:  firstNonEmpty(strings.TrimSpace(provider), "fallback"),
		Model:     firstNonEmpty(strings.TrimSpace(model), "heuristic"),
		Summary:   aiDefaultPlanSummary,
		CreatedAt: time.Now().UTC().Unix(),
		Steps:     steps,
	}
}

func buildAIToolCatalog(policy AIToolsPolicy) []aiToolCatalogItem {
	items := []aiToolCatalogItem{
		{
			ID:              aiToolSystemScan,
			Label:           "System Scan",
			Description:     "Collect host metrics, security snapshot, services and top processes.",
			Risk:            "low",
			RequiresConfirm: false,
			Enabled:         policy.Enabled,
			DefaultArgs: map[string]interface{}{
				"mode": "quick",
			},
		},
		{
			ID:              aiToolService,
			Label:           "Service Control",
			Description:     "Start, stop or restart managed services.",
			Risk:            "high",
			RequiresConfirm: true,
			Enabled:         policy.Enabled && policy.AllowServiceControl,
			DefaultArgs: map[string]interface{}{
				"name":   "openlitespeed",
				"action": "restart",
			},
		},
		{
			ID:              aiToolMalwareScan,
			Label:           "Malware Scan",
			Description:     "Run malware scanner across managed paths.",
			Risk:            "medium",
			RequiresConfirm: false,
			Enabled:         policy.Enabled && policy.AllowMalwareScan,
			DefaultArgs: map[string]interface{}{
				"path":   firstNonEmpty(strings.TrimSpace(policy.DefaultCWD), aiDefaultExecutionShellCWD),
				"engine": "auto",
			},
		},
		{
			ID:                  aiToolShellCommand,
			Label:               "Shell Command",
			Description:         "Execute shell commands inside managed paths or privileged mode if enabled.",
			Risk:                "medium",
			RequiresConfirm:     true,
			Enabled:             policy.Enabled && policy.AllowShell,
			PrivilegedSupported: policy.AllowPrivilegedShell,
			DefaultArgs: map[string]interface{}{
				"command":         "uptime && df -h && free -m",
				"cwd":             firstNonEmpty(strings.TrimSpace(policy.DefaultCWD), aiDefaultExecutionShellCWD),
				"timeout_seconds": minInt(policy.MaxCommandTimeoutSeconds, aiDefaultShellTimeout),
				"privileged":      false,
			},
		},
	}

	for i := range items {
		if items[i].Enabled {
			continue
		}
		switch items[i].ID {
		case aiToolService:
			items[i].BlockedReason = "Service control is disabled by policy."
		case aiToolMalwareScan:
			items[i].BlockedReason = "Malware scanning is disabled by policy."
		case aiToolShellCommand:
			items[i].BlockedReason = "Shell command execution is disabled by policy."
		default:
			items[i].BlockedReason = "AI tools are disabled by policy."
		}
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Label < items[j].Label
	})
	return items
}

func normalizeAIToolsPolicy(policy AIToolsPolicy) AIToolsPolicy {
	if strings.TrimSpace(policy.ConfirmToken) == "" {
		policy.ConfirmToken = aiToolsDefaultConfirmToken
	}
	policy.MaxCommandTimeoutSeconds = clampInt(policy.MaxCommandTimeoutSeconds, 2, 120)
	policy.MaxOutputChars = clampInt(policy.MaxOutputChars, 512, 20000)
	policy.DefaultCWD = strings.TrimSpace(policy.DefaultCWD)
	if policy.DefaultCWD == "" {
		policy.DefaultCWD = aiDefaultExecutionShellCWD
	}
	policy.AllowedCommandPrefixes = sanitizeAllowedCommandPrefixes(policy.AllowedCommandPrefixes)
	if len(policy.AllowedCommandPrefixes) == 0 {
		policy.AllowedCommandPrefixes = defaultAIToolsAllowedPrefixes()
	}
	return policy
}

func (s *service) aiPolicySnapshotLocked() AIToolsPolicy {
	policy := normalizeAIToolsPolicy(s.modules.AIToolsPolicy)

	if value := strings.TrimSpace(aiRuntimeEnvValue(aiEnvToolsEnabled)); value != "" {
		policy.Enabled = envBoolEnabled(value)
	}
	if value := strings.TrimSpace(aiRuntimeEnvValue(aiEnvToolsAllowShell)); value != "" {
		policy.AllowShell = envBoolEnabled(value)
	}
	if value := strings.TrimSpace(aiRuntimeEnvValue(aiEnvToolsAllowPrivileged)); value != "" {
		policy.AllowPrivilegedShell = envBoolEnabled(value)
	}
	if value := strings.TrimSpace(aiRuntimeEnvValue(aiEnvToolsAllowService)); value != "" {
		policy.AllowServiceControl = envBoolEnabled(value)
	}
	if value := strings.TrimSpace(aiRuntimeEnvValue(aiEnvToolsAllowMalware)); value != "" {
		policy.AllowMalwareScan = envBoolEnabled(value)
	}
	if value := strings.TrimSpace(aiRuntimeEnvValue(aiEnvToolsRequireConfirm)); value != "" {
		policy.RequireConfirmToken = envBoolEnabled(value)
	}
	if value := strings.TrimSpace(aiRuntimeEnvValue(aiEnvToolsConfirmToken)); value != "" {
		policy.ConfirmToken = value
	}
	if value := strings.TrimSpace(aiRuntimeEnvValue(aiEnvToolsMaxTimeout)); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			policy.MaxCommandTimeoutSeconds = parsed
		}
	}
	if value := strings.TrimSpace(aiRuntimeEnvValue(aiEnvToolsMaxOutput)); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			policy.MaxOutputChars = parsed
		}
	}
	if value := strings.TrimSpace(aiRuntimeEnvValue(aiEnvToolsDefaultCWD)); value != "" {
		policy.DefaultCWD = value
	}
	if value := strings.TrimSpace(aiRuntimeEnvValue(aiEnvToolsAllowedPrefixes)); value != "" {
		policy.AllowedCommandPrefixes = sanitizeAllowedCommandPrefixes([]string{value})
	}

	return normalizeAIToolsPolicy(policy)
}

func normalizeAIToolProvider(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case aiProviderGemini:
		return aiProviderGemini
	default:
		return aiProviderDeepSeek
	}
}

func normalizeAIProviderRuntime(runtimeState AIToolsProviderRuntime, deepSeekAPIKey, geminiAPIKey string) AIToolsProviderRuntime {
	runtimeState.ActiveProvider = normalizeAIToolProvider(runtimeState.ActiveProvider)
	runtimeState.DeepSeek = normalizeAIProviderConfig(aiProviderDeepSeek, runtimeState.DeepSeek)
	runtimeState.Gemini = normalizeAIProviderConfig(aiProviderGemini, runtimeState.Gemini)
	runtimeState.DeepSeek.HasAPIKey = strings.TrimSpace(deepSeekAPIKey) != ""
	runtimeState.DeepSeek.MaskedAPIKey = maskSecret(deepSeekAPIKey)
	runtimeState.Gemini.HasAPIKey = strings.TrimSpace(geminiAPIKey) != ""
	runtimeState.Gemini.MaskedAPIKey = maskSecret(geminiAPIKey)
	if runtimeState.UpdatedAt == 0 {
		runtimeState.UpdatedAt = time.Now().UTC().Unix()
	}
	return runtimeState
}

func (s *service) aiProviderRuntimeSnapshotLocked() AIToolsProviderRuntime {
	runtimeState := s.modules.AIToolsProvider

	if value := strings.TrimSpace(aiRuntimeEnvValue(aiEnvProviderActive)); value != "" {
		runtimeState.ActiveProvider = normalizeAIToolProvider(value)
	}
	if value := strings.TrimSpace(aiRuntimeEnvValue(aiEnvDeepSeekEnabled)); value != "" {
		runtimeState.DeepSeek.Enabled = envBoolEnabled(value)
	}
	if value := strings.TrimSpace(aiRuntimeEnvValue(aiEnvDeepSeekModel)); value != "" {
		runtimeState.DeepSeek.Model = value
	}
	if value := strings.TrimSpace(aiRuntimeEnvValue(aiEnvDeepSeekBaseURL)); value != "" {
		runtimeState.DeepSeek.BaseURL = value
	}
	if value := strings.TrimSpace(aiRuntimeEnvValue(aiEnvGeminiEnabled)); value != "" {
		runtimeState.Gemini.Enabled = envBoolEnabled(value)
	}
	if value := strings.TrimSpace(aiRuntimeEnvValue(aiEnvGeminiModel)); value != "" {
		runtimeState.Gemini.Model = value
	}
	if value := strings.TrimSpace(aiRuntimeEnvValue(aiEnvGeminiBaseURL)); value != "" {
		runtimeState.Gemini.BaseURL = value
	}

	deepSeekKey := strings.TrimSpace(aiRuntimeEnvValue(aiEnvDeepSeekAPIKey))
	geminiKey := strings.TrimSpace(aiRuntimeEnvValue(aiEnvGeminiAPIKey))
	return normalizeAIProviderRuntime(runtimeState, deepSeekKey, geminiKey)
}

func normalizeAIProviderConfig(provider string, cfg AIToolsProviderConfig) AIToolsProviderConfig {
	cfg.Model = strings.TrimSpace(cfg.Model)
	cfg.BaseURL = strings.TrimSuffix(strings.TrimSpace(cfg.BaseURL), "/")
	switch normalizeAIToolProvider(provider) {
	case aiProviderGemini:
		if cfg.Model == "" {
			cfg.Model = "gemini-2.5-flash"
		}
		if cfg.BaseURL == "" {
			cfg.BaseURL = "https://generativelanguage.googleapis.com/v1beta"
		}
	default:
		if cfg.Model == "" {
			cfg.Model = "deepseek-chat"
		}
		if cfg.BaseURL == "" {
			cfg.BaseURL = "https://api.deepseek.com/v1"
		}
	}
	return cfg
}

func applyAIProviderPatch(provider string, cfg AIToolsProviderConfig, patch *aiProviderConfigUpdatePayload, existingAPIKey string) (AIToolsProviderConfig, string) {
	cfg = normalizeAIProviderConfig(provider, cfg)
	key := strings.TrimSpace(existingAPIKey)
	if patch == nil {
		cfg.HasAPIKey = key != ""
		cfg.MaskedAPIKey = maskSecret(key)
		return cfg, key
	}
	if patch.Enabled != nil {
		cfg.Enabled = *patch.Enabled
	}
	if patch.Model != nil {
		cfg.Model = strings.TrimSpace(*patch.Model)
	}
	if patch.BaseURL != nil {
		cfg.BaseURL = strings.TrimSpace(*patch.BaseURL)
	}
	if patch.ClearAPIKey {
		key = ""
	}
	if patch.APIKey != nil {
		value := strings.TrimSpace(*patch.APIKey)
		if value != "" {
			key = value
		}
	}
	cfg = normalizeAIProviderConfig(provider, cfg)
	cfg.HasAPIKey = key != ""
	cfg.MaskedAPIKey = maskSecret(key)
	return cfg, key
}

func persistAIRuntimeEnv(updates map[string]string) error {
	for _, path := range []string{adminGatewayEnvPath(), adminServiceEnvPath()} {
		if err := writeEnvFileValues(path, updates); err != nil {
			return err
		}
	}
	for key, value := range updates {
		_ = os.Setenv(key, value)
	}
	return nil
}

func aiRuntimeEnvValue(key string) string {
	return firstNonEmpty(
		strings.TrimSpace(os.Getenv(key)),
		strings.TrimSpace(readEnvFileValue(adminGatewayEnvPath(), key)),
		strings.TrimSpace(readEnvFileValue(adminServiceEnvPath(), key)),
	)
}

func providerConfigByName(provider AIToolsProviderRuntime, name string) AIToolsProviderConfig {
	switch normalizeAIToolProvider(name) {
	case aiProviderGemini:
		return provider.Gemini
	default:
		return provider.DeepSeek
	}
}

func aiAPIKeyForProvider(provider string) string {
	switch normalizeAIToolProvider(provider) {
	case aiProviderGemini:
		return aiRuntimeEnvValue(aiEnvGeminiAPIKey)
	default:
		return aiRuntimeEnvValue(aiEnvDeepSeekAPIKey)
	}
}

func normalizeAIToolName(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "scan", "system_scan", "host_scan", "health_scan":
		return aiToolSystemScan
	case "service", "service_control", "restart_service", "service_restart":
		return aiToolService
	case "malware", "malware_scan", "virus_scan":
		return aiToolMalwareScan
	case "shell", "shell_command", "bash", "command":
		return aiToolShellCommand
	default:
		return ""
	}
}

func isAIToolEnabledByPolicy(tool string, policy AIToolsPolicy) bool {
	if !policy.Enabled {
		return false
	}
	switch normalizeAIToolName(tool) {
	case aiToolSystemScan:
		return true
	case aiToolService:
		return policy.AllowServiceControl
	case aiToolMalwareScan:
		return policy.AllowMalwareScan
	case aiToolShellCommand:
		return policy.AllowShell
	default:
		return false
	}
}

func aiToolBlockedReason(tool string, policy AIToolsPolicy) string {
	if !policy.Enabled {
		return "AI tools are disabled by policy."
	}
	switch normalizeAIToolName(tool) {
	case aiToolService:
		return "Service control is disabled by policy."
	case aiToolMalwareScan:
		return "Malware scan is disabled by policy."
	case aiToolShellCommand:
		return "Shell execution is disabled by policy."
	default:
		return "Tool is disabled by policy."
	}
}

func defaultAIToolsAllowedPrefixes() []string {
	return []string{
		"pwd", "ls", "cat", "tail", "grep", "find", "du", "df", "ps", "top", "free", "uptime",
		"journalctl", "systemctl", "service", "whoami", "id", "hostname", "ss", "netstat", "curl",
	}
}

func sanitizeAllowedCommandPrefixes(values []string) []string {
	seen := map[string]struct{}{}
	result := []string{}
	for _, item := range values {
		chunks := strings.Split(strings.ReplaceAll(strings.TrimSpace(item), "\n", ","), ",")
		for _, chunk := range chunks {
			cleaned := strings.ToLower(strings.TrimSpace(chunk))
			if cleaned == "" {
				continue
			}
			if _, ok := seen[cleaned]; ok {
				continue
			}
			seen[cleaned] = struct{}{}
			result = append(result, cleaned)
		}
	}
	return result
}

func isCommandAllowedByPolicy(command string, policy AIToolsPolicy) bool {
	command = strings.ToLower(strings.TrimSpace(command))
	if command == "" {
		return false
	}
	prefixes := sanitizeAllowedCommandPrefixes(policy.AllowedCommandPrefixes)
	if len(prefixes) == 0 {
		return true
	}
	for _, prefix := range prefixes {
		if prefix == "*" {
			return true
		}
		if command == prefix || strings.HasPrefix(command, prefix+" ") {
			return true
		}
	}
	return false
}

func normalizeAIToolArgs(tool string, args map[string]interface{}, policy AIToolsPolicy) map[string]interface{} {
	normalized := cloneInterfaceMap(args)
	switch normalizeAIToolName(tool) {
	case aiToolSystemScan:
		mode := strings.ToLower(strings.TrimSpace(stringValue(normalized["mode"])))
		if mode == "" {
			mode = "quick"
		}
		normalized["mode"] = mode
	case aiToolService:
		action := strings.ToLower(strings.TrimSpace(stringValue(normalized["action"])))
		if action == "" {
			action = "restart"
		}
		name := strings.TrimSpace(stringValue(normalized["name"]))
		if name == "" {
			name = "openlitespeed"
		}
		normalized["action"] = action
		normalized["name"] = name
	case aiToolMalwareScan:
		path := strings.TrimSpace(stringValue(normalized["path"]))
		if path == "" {
			path = firstNonEmpty(strings.TrimSpace(policy.DefaultCWD), aiDefaultExecutionShellCWD)
		}
		engine := strings.TrimSpace(stringValue(normalized["engine"]))
		if engine == "" {
			engine = "auto"
		}
		normalized["path"] = path
		normalized["engine"] = engine
	case aiToolShellCommand:
		command := strings.TrimSpace(stringValue(normalized["command"]))
		cwd := strings.TrimSpace(stringValue(normalized["cwd"]))
		if cwd == "" {
			cwd = firstNonEmpty(strings.TrimSpace(policy.DefaultCWD), aiDefaultExecutionShellCWD)
		}
		timeout := flexibleInt(normalized["timeout_seconds"], aiDefaultShellTimeout)
		timeout = clampInt(timeout, 2, clampInt(policy.MaxCommandTimeoutSeconds, 2, 120))
		normalized["command"] = command
		normalized["cwd"] = cwd
		normalized["timeout_seconds"] = timeout
		normalized["privileged"] = flexibleBool(normalized["privileged"])
	}
	return normalized
}

func runManagedShellCommandWithTimeout(command, cwd string, timeoutSeconds int) (string, string, error) {
	if strings.TrimSpace(command) == "" {
		return "", cwd, fmt.Errorf("command is required")
	}
	if strings.TrimSpace(cwd) == "" {
		cwd = aiDefaultExecutionShellCWD
	}
	if strings.HasPrefix(strings.TrimSpace(command), "cd ") {
		target := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(command), "cd"))
		if !filepath.IsAbs(target) {
			target = filepath.Join(cwd, target)
		}
		resolved, err := resolveManagedPath(target)
		if err != nil {
			return "", cwd, err
		}
		info, err := os.Stat(resolved)
		if err != nil || !info.IsDir() {
			return "", cwd, fmt.Errorf("directory not found")
		}
		return "", resolved, nil
	}

	resolvedCwd, err := resolveManagedPath(cwd)
	if err != nil {
		return "", cwd, err
	}
	timeoutSeconds = clampInt(timeoutSeconds, 2, 120)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSeconds)*time.Second)
	defer cancel()

	shell := "/bin/sh"
	args := []string{"-lc", command}
	if runtime.GOOS == "windows" {
		shell = "powershell"
		args = []string{"-NoProfile", "-Command", command}
	}
	cmd := exec.CommandContext(ctx, shell, args...)
	cmd.Dir = resolvedCwd
	output, runErr := cmd.CombinedOutput()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return string(output), resolvedCwd, fmt.Errorf("command timed out after %d seconds", timeoutSeconds)
	}
	if runErr != nil {
		return string(output), resolvedCwd, runErr
	}
	return string(output), resolvedCwd, nil
}

func runPrivilegedShellCommandWithTimeout(command, cwd string, timeoutSeconds int) (string, error) {
	if strings.TrimSpace(command) == "" {
		return "", fmt.Errorf("command is required")
	}
	timeoutSeconds = clampInt(timeoutSeconds, 2, 120)
	if strings.TrimSpace(cwd) == "" {
		cwd = string(os.PathSeparator)
	}
	cwd = filepath.Clean(cwd)
	if !filepath.IsAbs(cwd) {
		cwd = filepath.Clean(filepath.Join(string(os.PathSeparator), cwd))
	}
	if info, err := os.Stat(cwd); err != nil || !info.IsDir() {
		cwd = string(os.PathSeparator)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSeconds)*time.Second)
	defer cancel()

	shell := "/bin/sh"
	args := []string{"-lc", command}
	if runtime.GOOS == "windows" {
		shell = "powershell"
		args = []string{"-NoProfile", "-Command", command}
	}
	cmd := exec.CommandContext(ctx, shell, args...)
	cmd.Dir = cwd
	output, runErr := cmd.CombinedOutput()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return string(output), fmt.Errorf("command timed out after %d seconds", timeoutSeconds)
	}
	if runErr != nil {
		return string(output), runErr
	}
	return string(output), nil
}

func trimAndClampOutput(value string, maxChars int) string {
	trimmed := strings.TrimSpace(value)
	if maxChars <= 0 {
		maxChars = 4096
	}
	if len(trimmed) <= maxChars {
		return trimmed
	}
	return trimmed[:maxChars] + "...(truncated)"
}

func findAIToolPlan(plans []AIToolPlan, id string) (AIToolPlan, bool) {
	key := strings.TrimSpace(id)
	if key == "" {
		return AIToolPlan{}, false
	}
	for _, item := range plans {
		if strings.TrimSpace(item.ID) == key {
			return item, true
		}
	}
	return AIToolPlan{}, false
}

func pickAIToolStep(plan AIToolPlan, stepID string) (AIToolPlanStep, bool) {
	if len(plan.Steps) == 0 {
		return AIToolPlanStep{}, false
	}
	key := strings.TrimSpace(stepID)
	if key == "" {
		return plan.Steps[0], true
	}
	for _, step := range plan.Steps {
		if strings.TrimSpace(step.ID) == key {
			return step, true
		}
	}
	return AIToolPlanStep{}, false
}

func (s *service) appendAIToolPlanLocked(plan AIToolPlan) {
	s.modules.AIToolsPlans = append([]AIToolPlan{plan}, s.modules.AIToolsPlans...)
	if len(s.modules.AIToolsPlans) > aiToolsPlanLimit {
		s.modules.AIToolsPlans = s.modules.AIToolsPlans[:aiToolsPlanLimit]
	}
}

func (s *service) appendAIToolHistoryLocked(record AIToolExecutionRecord) {
	s.modules.AIToolsHistory = append([]AIToolExecutionRecord{record}, s.modules.AIToolsHistory...)
	if len(s.modules.AIToolsHistory) > aiToolsHistoryLimit {
		s.modules.AIToolsHistory = s.modules.AIToolsHistory[:aiToolsHistoryLimit]
	}
}

func normalizeAIRisk(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "critical":
		return "critical"
	case "high":
		return "high"
	case "low":
		return "low"
	default:
		return "medium"
	}
}

func maxAIRisk(a, b string) string {
	weights := map[string]int{"low": 1, "medium": 2, "high": 3, "critical": 4}
	a = normalizeAIRisk(a)
	b = normalizeAIRisk(b)
	if weights[b] > weights[a] {
		return b
	}
	return a
}

func defaultRiskForAITool(tool string) string {
	switch normalizeAIToolName(tool) {
	case aiToolSystemScan:
		return "low"
	case aiToolMalwareScan:
		return "medium"
	case aiToolService:
		return "high"
	case aiToolShellCommand:
		return "medium"
	default:
		return "medium"
	}
}

func aiToolDefaultRequiresConfirm(tool, risk string) bool {
	switch normalizeAIToolName(tool) {
	case aiToolService:
		return true
	case aiToolShellCommand:
		return normalizeAIRisk(risk) == "high" || normalizeAIRisk(risk) == "critical"
	default:
		return normalizeAIRisk(risk) == "critical"
	}
}

func estimateShellRisk(command string) string {
	lower := strings.ToLower(strings.TrimSpace(command))
	if lower == "" {
		return "medium"
	}
	highRiskPatterns := []string{
		"rm ", "mkfs", "shutdown", "reboot", "userdel", "passwd ", "chmod ", "chown ",
		"systemctl stop", "systemctl restart", "service ", "iptables", "ufw ", "firewall-cmd",
	}
	for _, pattern := range highRiskPatterns {
		if strings.Contains(lower, pattern) {
			return "high"
		}
	}
	return "medium"
}

func aiExecutionStatusMessage(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "success":
		return "Execution completed successfully."
	case "dry_run":
		return "Dry-run completed successfully."
	case "blocked":
		return "Execution blocked by policy."
	default:
		return "Execution failed."
	}
}

func containsAny(haystack string, needles []string) bool {
	for _, needle := range needles {
		if strings.Contains(haystack, needle) {
			return true
		}
	}
	return false
}

func principalRequestIdentity(principal servicePrincipal) string {
	return firstNonEmpty(strings.TrimSpace(principal.Username), strings.TrimSpace(principal.Email), "system")
}

func cloneInterfaceMap(input map[string]interface{}) map[string]interface{} {
	if input == nil {
		return map[string]interface{}{}
	}
	result := make(map[string]interface{}, len(input))
	for key, value := range input {
		result[key] = value
	}
	return result
}

func flexibleInt(value interface{}, fallback int) int {
	switch typed := value.(type) {
	case float64:
		return int(typed)
	case int:
		return typed
	case int64:
		return int(typed)
	case string:
		if parsed, err := strconv.Atoi(strings.TrimSpace(typed)); err == nil {
			return parsed
		}
	}
	return fallback
}

func flexibleBool(value interface{}) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case float64:
		return typed != 0
	case int:
		return typed != 0
	case string:
		switch strings.ToLower(strings.TrimSpace(typed)) {
		case "1", "true", "yes", "on":
			return true
		}
	}
	return false
}
