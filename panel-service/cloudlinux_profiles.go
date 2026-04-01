package main

import (
	"net/http"
	"runtime"
	"sort"
	"strings"
)

const (
	cloudLinuxDefaultHostingCPUPercent  = 100
	cloudLinuxDefaultResellerCPUPercent = 200
	cloudLinuxDefaultHostingMemoryMB    = 1024
	cloudLinuxDefaultResellerMemoryMB   = 2048
	cloudLinuxDefaultHostingIOMBPS      = 10
	cloudLinuxDefaultResellerIOMBPS     = 20
)

type cloudLinuxProfileSummary struct {
	TotalPackages        int `json:"total_packages"`
	TotalSites           int `json:"total_sites"`
	TotalUsers           int `json:"total_users"`
	ReadyProfiles        int `json:"ready_profiles"`
	ProfilesWithDefaults int `json:"profiles_with_defaults"`
}

type cloudLinuxPackageProfile struct {
	ID                string `json:"id"`
	PackageName       string `json:"package_name"`
	PlanType          string `json:"plan_type"`
	WebsiteCount      int    `json:"website_count"`
	UserCount         int    `json:"user_count"`
	CPUPercent        int    `json:"cpu_percent"`
	MemoryMB          int    `json:"memory_mb"`
	IOMBPS            int    `json:"io_mb_s"`
	EntryProcesses    int    `json:"entry_processes"`
	NProc             int    `json:"nproc"`
	IOPS              int    `json:"iops"`
	UsedCPUDefault    bool   `json:"used_cpu_default"`
	UsedMemoryDefault bool   `json:"used_memory_default"`
	UsedIODefault     bool   `json:"used_io_default"`
	Readiness         string `json:"readiness"`
	ReadinessReason   string `json:"readiness_reason"`
	ReadyForApply     bool   `json:"ready_for_apply"`
}

type cloudLinuxProfilePayload struct {
	Summary  cloudLinuxProfileSummary   `json:"summary"`
	Profiles []cloudLinuxPackageProfile `json:"profiles"`
}

type cloudLinuxDerivedLimits struct {
	CPUPercent        int
	MemoryMB          int
	IOMBPS            int
	EntryProcesses    int
	NProc             int
	IOPS              int
	UsedCPUDefault    bool
	UsedMemoryDefault bool
	UsedIODefault     bool
}

func cloudLinuxPackageKey(name string) string {
	trimmed := strings.ToLower(strings.TrimSpace(name))
	if trimmed == "" {
		return "default"
	}
	return trimmed
}

func cloudLinuxProfileReadiness(status cloudLinuxStatus) (state string, reason string, ready bool) {
	if runtime.GOOS != "linux" {
		return "unsupported_host", "CloudLinux profile application is supported only on Linux hosts.", false
	}
	if !status.Available {
		return "waiting_cloudlinux", "CloudLinux is not detected on this node.", false
	}
	if !status.Features["lve_manager"] {
		return "missing_lve_manager", "LVE manager is not detected.", false
	}
	if !status.Commands["lvectl"] {
		return "missing_lvectl", "lvectl command is not available.", false
	}
	return "ready", "Profile can be applied during controlled rollout.", true
}

func deriveCloudLinuxLimits(pkg Package, websiteCount int) cloudLinuxDerivedLimits {
	planType := normalizePlanType(pkg.PlanType)

	cpuDefault := cloudLinuxDefaultHostingCPUPercent
	memoryDefault := cloudLinuxDefaultHostingMemoryMB
	ioDefault := cloudLinuxDefaultHostingIOMBPS
	epBase := 25

	if planType == "reseller" {
		cpuDefault = cloudLinuxDefaultResellerCPUPercent
		memoryDefault = cloudLinuxDefaultResellerMemoryMB
		ioDefault = cloudLinuxDefaultResellerIOMBPS
		epBase = 60
	}

	cpu := pkg.CPULimit
	usedCPUDefault := false
	if cpu <= 0 {
		cpu = cpuDefault
		usedCPUDefault = true
	}

	memory := pkg.RamMB
	usedMemoryDefault := false
	if memory <= 0 {
		memory = memoryDefault
		usedMemoryDefault = true
	}

	io := pkg.IOLimit
	usedIODefault := false
	if io <= 0 {
		io = ioDefault
		usedIODefault = true
	}

	ep := epBase
	if pkg.Domains > 0 {
		ep = clampInt(pkg.Domains*8, epBase, 300)
	}
	if websiteCount > 0 {
		ep = clampInt(maxInt(ep, websiteCount*10), epBase, 400)
	}

	nproc := clampInt(ep*2, 40, 1024)
	iops := clampInt(io*256, 512, 8192)

	return cloudLinuxDerivedLimits{
		CPUPercent:        cpu,
		MemoryMB:          memory,
		IOMBPS:            io,
		EntryProcesses:    ep,
		NProc:             nproc,
		IOPS:              iops,
		UsedCPUDefault:    usedCPUDefault,
		UsedMemoryDefault: usedMemoryDefault,
		UsedIODefault:     usedIODefault,
	}
}

func buildCloudLinuxProfilesPayload(status cloudLinuxStatus, packages []Package, websites []Website, users []PanelUser) cloudLinuxProfilePayload {
	websiteCounts := map[string]int{}
	for _, site := range websites {
		websiteCounts[cloudLinuxPackageKey(site.Package)]++
	}

	userCounts := map[string]int{}
	for _, user := range users {
		if strings.EqualFold(strings.TrimSpace(user.Role), "admin") {
			continue
		}
		if strings.TrimSpace(user.Package) == "" {
			continue
		}
		userCounts[cloudLinuxPackageKey(user.Package)]++
	}

	rows := make([]cloudLinuxPackageProfile, 0, len(packages))
	summary := cloudLinuxProfileSummary{
		TotalPackages: len(packages),
		TotalSites:    len(websites),
		TotalUsers:    len(users),
	}

	for _, pkg := range packages {
		packageName := firstNonEmpty(strings.TrimSpace(pkg.Name), "default")
		key := cloudLinuxPackageKey(packageName)
		limits := deriveCloudLinuxLimits(pkg, websiteCounts[key])
		readinessState, readinessReason, readyForApply := cloudLinuxProfileReadiness(status)

		row := cloudLinuxPackageProfile{
			ID:                firstNonEmpty(sanitizeName(packageName), "default"),
			PackageName:       packageName,
			PlanType:          normalizePlanType(pkg.PlanType),
			WebsiteCount:      websiteCounts[key],
			UserCount:         userCounts[key],
			CPUPercent:        limits.CPUPercent,
			MemoryMB:          limits.MemoryMB,
			IOMBPS:            limits.IOMBPS,
			EntryProcesses:    limits.EntryProcesses,
			NProc:             limits.NProc,
			IOPS:              limits.IOPS,
			UsedCPUDefault:    limits.UsedCPUDefault,
			UsedMemoryDefault: limits.UsedMemoryDefault,
			UsedIODefault:     limits.UsedIODefault,
			Readiness:         readinessState,
			ReadinessReason:   readinessReason,
			ReadyForApply:     readyForApply,
		}
		rows = append(rows, row)

		if row.ReadyForApply {
			summary.ReadyProfiles++
		}
		if row.UsedCPUDefault || row.UsedMemoryDefault || row.UsedIODefault {
			summary.ProfilesWithDefaults++
		}
	}

	sort.Slice(rows, func(i, j int) bool {
		return strings.ToLower(rows[i].PackageName) < strings.ToLower(rows[j].PackageName)
	})

	return cloudLinuxProfilePayload{
		Summary:  summary,
		Profiles: rows,
	}
}

func (s *service) handleCloudLinuxProfiles(w http.ResponseWriter) {
	status := detectCloudLinuxStatus()

	s.mu.RLock()
	packages := append([]Package(nil), s.state.Packages...)
	websites := append([]Website(nil), s.state.Websites...)
	users := append([]PanelUser(nil), s.state.Users...)
	s.mu.RUnlock()

	payload := buildCloudLinuxProfilesPayload(status, packages, websites, users)
	writeJSON(w, http.StatusOK, apiResponse{Status: "success", Data: payload})
}
