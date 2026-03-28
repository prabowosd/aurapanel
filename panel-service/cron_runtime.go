package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func cronManagedDir() string {
	return firstNonEmpty(strings.TrimSpace(os.Getenv("AURAPANEL_CRON_DIR")), "/etc/cron.d")
}

func cronManagedPath(id string) string {
	return filepath.Join(cronManagedDir(), "aurapanel-"+sanitizeName(id))
}

func parseRuntimeCronLine(id, line string) (CronJob, bool) {
	fields := strings.Fields(strings.TrimSpace(line))
	if len(fields) < 3 {
		return CronJob{}, false
	}
	if strings.HasPrefix(fields[0], "@") {
		return CronJob{
			ID:       id,
			Schedule: fields[0],
			User:     fields[1],
			Command:  strings.Join(fields[2:], " "),
		}, true
	}
	if len(fields) < 7 {
		return CronJob{}, false
	}
	return CronJob{
		ID:       id,
		Schedule: strings.Join(fields[:5], " "),
		User:     fields[5],
		Command:  strings.Join(fields[6:], " "),
	}, true
}

func runtimeCronJobs() ([]CronJob, error) {
	pattern := filepath.Join(cronManagedDir(), "aurapanel-*")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	jobs := make([]CronJob, 0, len(matches))
	for _, match := range matches {
		raw, readErr := os.ReadFile(match)
		if readErr != nil {
			return nil, readErr
		}
		id := sanitizeName(strings.TrimPrefix(filepath.Base(match), "aurapanel-"))
		for _, line := range strings.Split(string(raw), "\n") {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.Contains(trimmed, "=") {
				continue
			}
			if job, ok := parseRuntimeCronLine(id, trimmed); ok {
				jobs = append(jobs, job)
				break
			}
		}
	}
	sort.Slice(jobs, func(i, j int) bool { return jobs[i].ID < jobs[j].ID })
	return jobs, nil
}

func validateCronSchedule(schedule string) error {
	schedule = strings.TrimSpace(schedule)
	if schedule == "" {
		return fmt.Errorf("cron schedule is required")
	}
	if strings.HasPrefix(schedule, "@") {
		switch schedule {
		case "@reboot", "@yearly", "@annually", "@monthly", "@weekly", "@daily", "@midnight", "@hourly":
			return nil
		default:
			return fmt.Errorf("unsupported cron shortcut")
		}
	}
	fields := strings.Fields(schedule)
	if len(fields) != 5 {
		return fmt.Errorf("cron schedule must have 5 fields")
	}
	for _, field := range fields {
		if field == "" {
			return fmt.Errorf("invalid cron field")
		}
	}
	return nil
}

func createRuntimeCronJob(job CronJob) error {
	if err := validateCronSchedule(job.Schedule); err != nil {
		return err
	}
	if !systemUserExists(job.User) {
		return fmt.Errorf("system user not found")
	}
	if err := os.MkdirAll(cronManagedDir(), 0o755); err != nil {
		return err
	}
	content := strings.Join([]string{
		"SHELL=/bin/bash",
		"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		fmt.Sprintf("# AuraPanel Managed Job %s", job.ID),
		fmt.Sprintf("%s %s %s", strings.TrimSpace(job.Schedule), strings.TrimSpace(job.User), strings.TrimSpace(job.Command)),
		"",
	}, "\n")
	return os.WriteFile(cronManagedPath(job.ID), []byte(content), 0o644)
}

func deleteRuntimeCronJob(id string) error {
	path := cronManagedPath(id)
	if !fileExists(path) {
		return os.ErrNotExist
	}
	return os.Remove(path)
}
