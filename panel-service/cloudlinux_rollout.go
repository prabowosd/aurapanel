package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

type cloudLinuxRolloutSummary struct {
	TotalUsers         int    `json:"total_users"`
	ScopedUsers        int    `json:"scoped_users"`
	ReadyUsers         int    `json:"ready_users"`
	BlockedUsers       int    `json:"blocked_users"`
	UsersUsingDefaults int    `json:"users_using_defaults"`
	PackageFilter      string `json:"package_filter,omitempty"`
	OnlyReady          bool   `json:"only_ready"`
	ApplyEnabled       bool   `json:"apply_enabled"`
	ConfirmToken       string `json:"confirm_token"`
}

type cloudLinuxRolloutEntry struct {
	ID                string `json:"id"`
	Username          string `json:"username"`
	Role              string `json:"role"`
	PackageName       string `json:"package_name"`
	PlanType          string `json:"plan_type"`
	WebsiteCount      int    `json:"website_count"`
	CPUPercent        int    `json:"cpu_percent"`
	MemoryMB          int    `json:"memory_mb"`
	IOMBPS            int    `json:"io_mb_s"`
	EntryProcesses    int    `json:"entry_processes"`
	NProc             int    `json:"nproc"`
	IOPS              int    `json:"iops"`
	UsedCPUDefault    bool   `json:"used_cpu_default"`
	UsedMemoryDefault bool   `json:"used_memory_default"`
	UsedIODefault     bool   `json:"used_io_default"`
	UsesDefaults      bool   `json:"uses_defaults"`
	Readiness         string `json:"readiness"`
	ReadinessReason   string `json:"readiness_reason"`
	ReadyForApply     bool   `json:"ready_for_apply"`
	CommandHint       string `json:"command_hint"`
}

type cloudLinuxRolloutPayload struct {
	Summary       cloudLinuxRolloutSummary `json:"summary"`
	Users         []cloudLinuxRolloutEntry `json:"users"`
	ScriptPreview []string                 `json:"script_preview"`
	GeneratedAt   int64                    `json:"generated_at"`
}

func cloudLinuxUserKey(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func cloudLinuxRolloutBoolParam(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func buildCloudLinuxCommandHint(username string, profile cloudLinuxPackageProfile) string {
	safeUser := sanitizeName(username)
	if safeUser == "" {
		safeUser = "user"
	}
	return fmt.Sprintf(
		"lvectl set-user %s --speed=%d%% --pmem=%dM --io=%dMBps --nproc=%d --iops=%d --ep=%d",
		safeUser,
		profile.CPUPercent,
		profile.MemoryMB,
		profile.IOMBPS,
		profile.NProc,
		profile.IOPS,
		profile.EntryProcesses,
	)
}

func buildCloudLinuxRolloutPayload(status cloudLinuxStatus, packages []Package, websites []Website, users []PanelUser, packageFilter string, onlyReady bool) cloudLinuxRolloutPayload {
	profilePayload := buildCloudLinuxProfilesPayload(status, packages, websites, users)
	profileByPackage := make(map[string]cloudLinuxPackageProfile, len(profilePayload.Profiles))
	for _, row := range profilePayload.Profiles {
		profileByPackage[cloudLinuxPackageKey(row.PackageName)] = row
	}

	websitesByUser := map[string]int{}
	for _, site := range websites {
		ownerKey := cloudLinuxUserKey(firstNonEmpty(strings.TrimSpace(site.Owner), strings.TrimSpace(site.User)))
		if ownerKey == "" {
			continue
		}
		websitesByUser[ownerKey]++
	}

	targetPackageKey := cloudLinuxPackageKey(packageFilter)
	filterEnabled := strings.TrimSpace(packageFilter) != ""
	applyEnabled := cloudLinuxApplyEnabled()

	rows := make([]cloudLinuxRolloutEntry, 0, len(users))
	summary := cloudLinuxRolloutSummary{
		PackageFilter: strings.TrimSpace(packageFilter),
		OnlyReady:     onlyReady,
		ApplyEnabled:  applyEnabled,
		ConfirmToken:  cloudLinuxRolloutConfirmToken,
	}

	for _, user := range users {
		role := normalizeRole(user.Role)
		if role == "admin" {
			continue
		}
		summary.TotalUsers++

		username := firstNonEmpty(strings.TrimSpace(user.Username), strings.TrimSpace(user.Email))
		if username == "" {
			continue
		}

		packageName := firstNonEmpty(strings.TrimSpace(user.Package), "default")
		packageKey := cloudLinuxPackageKey(packageName)
		if filterEnabled && packageKey != targetPackageKey {
			continue
		}

		summary.ScopedUsers++

		profile, ok := profileByPackage[packageKey]
		if !ok {
			profile = cloudLinuxPackageProfile{
				PackageName:       packageName,
				PlanType:          "hosting",
				CPUPercent:        cloudLinuxDefaultHostingCPUPercent,
				MemoryMB:          cloudLinuxDefaultHostingMemoryMB,
				IOMBPS:            cloudLinuxDefaultHostingIOMBPS,
				EntryProcesses:    25,
				NProc:             50,
				IOPS:              2048,
				UsedCPUDefault:    true,
				UsedMemoryDefault: true,
				UsedIODefault:     true,
				Readiness:         "missing_package_profile",
				ReadinessReason:   "Package profile is not found. Re-check package catalog.",
				ReadyForApply:     false,
			}
		}

		usesDefaults := profile.UsedCPUDefault || profile.UsedMemoryDefault || profile.UsedIODefault
		readyForApply := profile.ReadyForApply

		if onlyReady && !readyForApply {
			continue
		}

		entry := cloudLinuxRolloutEntry{
			ID:                firstNonEmpty(sanitizeName(username), generateSecret(6)),
			Username:          username,
			Role:              role,
			PackageName:       packageName,
			PlanType:          profile.PlanType,
			WebsiteCount:      websitesByUser[cloudLinuxUserKey(username)],
			CPUPercent:        profile.CPUPercent,
			MemoryMB:          profile.MemoryMB,
			IOMBPS:            profile.IOMBPS,
			EntryProcesses:    profile.EntryProcesses,
			NProc:             profile.NProc,
			IOPS:              profile.IOPS,
			UsedCPUDefault:    profile.UsedCPUDefault,
			UsedMemoryDefault: profile.UsedMemoryDefault,
			UsedIODefault:     profile.UsedIODefault,
			UsesDefaults:      usesDefaults,
			Readiness:         profile.Readiness,
			ReadinessReason:   profile.ReadinessReason,
			ReadyForApply:     readyForApply,
			CommandHint:       buildCloudLinuxCommandHint(username, profile),
		}
		rows = append(rows, entry)

		if usesDefaults {
			summary.UsersUsingDefaults++
		}
		if readyForApply {
			summary.ReadyUsers++
		} else {
			summary.BlockedUsers++
		}
	}

	scriptPreview := make([]string, 0, len(rows))
	for _, item := range rows {
		if !item.ReadyForApply {
			continue
		}
		scriptPreview = append(scriptPreview, item.CommandHint)
		if len(scriptPreview) >= 25 {
			break
		}
	}

	return cloudLinuxRolloutPayload{
		Summary:       summary,
		Users:         rows,
		ScriptPreview: scriptPreview,
		GeneratedAt:   time.Now().UTC().Unix(),
	}
}

func (s *service) handleCloudLinuxRolloutPlan(w http.ResponseWriter, r *http.Request) {
	status := detectCloudLinuxStatus()
	packageFilter := strings.TrimSpace(r.URL.Query().Get("package"))
	onlyReady := cloudLinuxRolloutBoolParam(r.URL.Query().Get("only_ready"))

	s.mu.RLock()
	packages := append([]Package(nil), s.state.Packages...)
	websites := append([]Website(nil), s.state.Websites...)
	users := append([]PanelUser(nil), s.state.Users...)
	s.mu.RUnlock()

	payload := buildCloudLinuxRolloutPayload(status, packages, websites, users, packageFilter, onlyReady)
	writeJSON(w, http.StatusOK, apiResponse{Status: "success", Data: payload})
}
