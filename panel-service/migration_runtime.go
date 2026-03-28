package main

import (
	"encoding/json"
	"fmt"
	"io"
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
	if fallback != "" {
		return fallback
	}
	for _, entry := range entries {
		lower := strings.ToLower(entry)
		switch {
		case strings.Contains(lower, "homedir/"), strings.Contains(lower, "userdata/"):
			return "cpanel"
		case strings.Contains(lower, "domains/"), strings.Contains(lower, "httpdocs"):
			return "plesk"
		case strings.Contains(lower, "vhosts/"), strings.Contains(lower, "openlitespeed"):
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
	entries, err := migrationArchiveEntries(path)
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
	return MigrationAnalysis{
		SourceType:      guessMigrationSourceType(entries, strings.TrimSpace(sourceType)),
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

func importMigrationArchive(path, sourceType, targetOwner string) (MigrationJob, error) {
	analysis, err := analyzeMigrationArchive(path, sourceType)
	if err != nil {
		return MigrationJob{}, err
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
		"target_owner": firstNonEmpty(strings.TrimSpace(targetOwner), "aura"),
		"accounts":     analysis.EmailAccounts,
	}, "", "  ")
	vhostPlan, _ := json.MarshalIndent(map[string]interface{}{
		"target_owner": firstNonEmpty(strings.TrimSpace(targetOwner), "aura"),
		"domains":      analysis.VhostCandidates,
	}, "", "  ")
	if err := os.WriteFile(emailPlanPath, emailPlan, 0o644); err != nil {
		return MigrationJob{}, err
	}
	if err := os.WriteFile(vhostPlanPath, vhostPlan, 0o644); err != nil {
		return MigrationJob{}, err
	}
	return MigrationJob{
		ID:       jobID,
		Status:   "completed",
		Progress: 100,
		Logs: []string{
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
		},
	}, nil
}
