package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
)

func migrationUploadsDir() string {
	return "/var/lib/aurapanel/migrations/uploads"
}

func migrationJobsDir() string {
	return "/var/lib/aurapanel/migrations/jobs"
}

func normalizeMigrationSourceTypeInput(value string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch normalized {
	case "", "auto":
		return "", nil
	case "cpanel", "cyberpanel", "plesk", "generic":
		return normalized, nil
	case "openlitespeed":
		return "cyberpanel", nil
	default:
		return "", fmt.Errorf("unsupported source type: %s", value)
	}
}

func migrationSourceLabel(source string) string {
	switch strings.TrimSpace(strings.ToLower(source)) {
	case "cpanel":
		return "cPanel"
	case "cyberpanel":
		return "CyberPanel"
	case "plesk":
		return "Plesk"
	case "generic":
		return "Generic"
	default:
		return "Auto"
	}
}

func formatMigrationBytesHuman(size int64) string {
	if size <= 0 {
		return "0 B"
	}
	units := []string{"B", "KB", "MB", "GB", "TB"}
	value := float64(size)
	unit := 0
	for value >= 1024 && unit < len(units)-1 {
		value /= 1024
		unit++
	}
	if unit == 0 {
		return fmt.Sprintf("%d %s", size, units[unit])
	}
	return fmt.Sprintf("%.1f %s", value, units[unit])
}

func estimateMigrationETASeconds(analysis MigrationAnalysis) int {
	seconds := 45
	seconds += analysis.Stats.DatabaseCount * 35
	seconds += analysis.Stats.EmailCount * 8
	seconds += len(analysis.VhostCandidates) * 20
	seconds += int(math.Min(900, float64(analysis.Stats.FileCount/80)))
	seconds += int(math.Min(600, float64(analysis.ArchiveSize/(50*1024*1024))*20))
	if seconds < 60 {
		return 60
	}
	if seconds > 7200 {
		return 7200
	}
	return seconds
}

func saveMigrationUpload(fileName string, src io.Reader) (string, error) {
	if err := os.MkdirAll(migrationUploadsDir(), 0o755); err != nil {
		return "", err
	}
	base := filepath.Base(strings.TrimSpace(fileName))
	if base == "" {
		return "", fmt.Errorf("invalid upload filename")
	}
	target := filepath.Join(migrationUploadsDir(), base)
	dst, err := os.Create(target)
	if err != nil {
		return "", err
	}
	defer dst.Close()
	if _, err := io.Copy(dst, src); err != nil {
		return "", err
	}
	return target, nil
}

func migrationArchiveEntries(path string) ([]string, error) {
	lower := strings.ToLower(path)
	switch {
	case strings.HasSuffix(lower, ".zip"):
		output, err := commandOutputTrimmed("unzip", "-Z1", path)
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(output) == "" {
			return []string{}, nil
		}
		return strings.Split(output, "\n"), nil
	default:
		output, err := commandOutputTrimmed("tar", "-tf", path)
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(output) == "" {
			return []string{}, nil
		}
		return strings.Split(output, "\n"), nil
	}
}

func guessMigrationSourceType(entries []string, fallback string) string {
	if normalized, err := normalizeMigrationSourceTypeInput(fallback); err == nil && normalized != "" {
		return normalized
	}
	for _, entry := range entries {
		lower := strings.ToLower(entry)
		switch {
		case strings.Contains(lower, "cpmove-"),
			strings.Contains(lower, "homedir/"),
			strings.Contains(lower, "userdata/"):
			return "cpanel"
		case strings.Contains(lower, "domains/"),
			strings.Contains(lower, "httpdocs/"),
			strings.Contains(lower, "plesk"):
			return "plesk"
		case strings.Contains(lower, "vhosts/"),
			strings.Contains(lower, "openlitespeed"),
			strings.Contains(lower, "cyberpanel"):
			return "cyberpanel"
		}
	}
	return "generic"
}

func uniqueStrings(items []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}

func analyzeMigrationArchive(path, sourceType string) (MigrationAnalysis, error) {
	archivePath := filepath.Clean(strings.TrimSpace(path))
	if archivePath == "" {
		return MigrationAnalysis{}, fmt.Errorf("archive path is required")
	}
	info, err := os.Stat(archivePath)
	if err != nil {
		return MigrationAnalysis{}, err
	}
	if info.IsDir() {
		return MigrationAnalysis{}, fmt.Errorf("archive path must be a file")
	}

	entries, err := migrationArchiveEntries(archivePath)
	if err != nil {
		return MigrationAnalysis{}, err
	}
	mysqlDumps := []string{}
	emailAccounts := []string{}
	vhostCandidates := []string{}
	for _, entry := range entries {
		lower := strings.ToLower(strings.TrimSpace(entry))
		if lower == "" || strings.HasSuffix(lower, "/") {
			continue
		}
		if strings.HasSuffix(lower, ".sql") || strings.HasSuffix(lower, ".sql.gz") {
			mysqlDumps = append(mysqlDumps, entry)
		}
		if strings.Contains(lower, "mail/") || strings.Contains(lower, "imap/") || strings.Contains(lower, "vmail/") {
			parts := strings.Split(strings.ReplaceAll(lower, "\\", "/"), "/")
			for i := 0; i < len(parts)-1; i++ {
				if (parts[i] == "mail" || parts[i] == "vmail") && i+2 < len(parts) {
					emailAccounts = append(emailAccounts, fmt.Sprintf("%s@%s", parts[i+2], parts[i+1]))
				}
			}
		}
		if strings.Contains(lower, "public_html") || strings.Contains(lower, "httpdocs") || strings.Contains(lower, "htdocs") {
			parts := strings.Split(strings.ReplaceAll(lower, "\\", "/"), "/")
			for _, part := range parts {
				if strings.Contains(part, ".") && !strings.Contains(part, ".sql") && !strings.Contains(part, ".tar") {
					vhostCandidates = append(vhostCandidates, normalizeDomain(part))
				}
			}
		}
	}
	mysqlDumps = uniqueStrings(mysqlDumps)
	emailAccounts = uniqueStrings(emailAccounts)
	vhostCandidates = uniqueStrings(vhostCandidates)
	warnings := []string{}
	if len(mysqlDumps) == 0 {
		warnings = append(warnings, "No SQL dumps detected in archive.")
	}
	if len(vhostCandidates) == 0 {
		warnings = append(warnings, "No obvious vhost candidates detected; manual review may be required.")
	}

	detectedSource := guessMigrationSourceType(entries, strings.TrimSpace(sourceType))
	if detectedSource == "generic" {
		warnings = append(warnings, "Source panel could not be detected with high confidence; run pre-check before import.")
	}

	return MigrationAnalysis{
		SourceType:      detectedSource,
		ArchivePath:     archivePath,
		ArchiveSize:     info.Size(),
		ArchiveSizeText: formatMigrationBytesHuman(info.Size()),
		Stats:           MigrationStats{FileCount: len(entries), DatabaseCount: len(mysqlDumps), EmailCount: len(emailAccounts)},
		MySQLDumps:      mysqlDumps,
		EmailAccounts:   emailAccounts,
		VhostCandidates: vhostCandidates,
		Warnings:        warnings,
	}, nil
}

func extractMigrationArchive(path, destination string) error {
	if err := os.MkdirAll(destination, 0o755); err != nil {
		return err
	}
	lower := strings.ToLower(path)
	switch {
	case strings.HasSuffix(lower, ".zip"):
		_, err := commandOutputTrimmed("unzip", "-o", path, "-d", destination)
		return err
	default:
		_, err := commandOutputTrimmed("tar", "-xf", path, "-C", destination)
		return err
	}
}

func importMigrationArchive(analysis MigrationAnalysis, targetOwner string) (MigrationJob, error) {
	path := strings.TrimSpace(analysis.ArchivePath)
	if path == "" {
		return MigrationJob{}, fmt.Errorf("archive path is required")
	}
	jobID := "mig-" + generateSecret(6)
	workDir := filepath.Join(migrationJobsDir(), jobID)
	extractDir := filepath.Join(workDir, "extract")
	if err := extractMigrationArchive(path, extractDir); err != nil {
		return MigrationJob{}, err
	}
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		return MigrationJob{}, err
	}
	emailPlanPath := filepath.Join(workDir, "email-plan.json")
	vhostPlanPath := filepath.Join(workDir, "vhost-plan.json")
	emailPlan, _ := json.MarshalIndent(map[string]interface{}{
		"target_owner": firstNonEmpty(strings.TrimSpace(targetOwner), "admin"),
		"accounts":     analysis.EmailAccounts,
	}, "", "  ")
	vhostPlan, _ := json.MarshalIndent(map[string]interface{}{
		"target_owner": firstNonEmpty(strings.TrimSpace(targetOwner), "admin"),
		"domains":      analysis.VhostCandidates,
	}, "", "  ")
	if err := os.WriteFile(emailPlanPath, emailPlan, 0o644); err != nil {
		return MigrationJob{}, err
	}
	if err := os.WriteFile(vhostPlanPath, vhostPlan, 0o644); err != nil {
		return MigrationJob{}, err
	}

	initialProgress := 35
	status := "running"
	if analysis.Precheck.Ready {
		initialProgress = 55
	}

	return MigrationJob{
		ID:       jobID,
		Status:   status,
		Progress: initialProgress,
		Logs: []string{
			fmt.Sprintf("Source profile: %s", migrationSourceLabel(analysis.SourceType)),
			fmt.Sprintf("Archive saved from %s", path),
			fmt.Sprintf("Archive extracted to %s", extractDir),
			fmt.Sprintf("Email plan generated: %s", emailPlanPath),
			fmt.Sprintf("Vhost plan generated: %s", vhostPlanPath),
		},
		Summary: MigrationSummary{
			ConvertedDBFiles: analysis.MySQLDumps,
			EmailPlanFile:    emailPlanPath,
			VhostPlanFile:    vhostPlanPath,
			SystemApply:      false,
			PrecheckReady:    analysis.Precheck.Ready,
			ConflictCount:    len(analysis.Precheck.Conflicts),
			ETASeconds:       analysis.Precheck.ETASeconds,
		},
	}, nil
}
