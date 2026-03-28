package main

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func dbBackupDir() string {
	return firstNonEmpty(strings.TrimSpace(os.Getenv("AURAPANEL_DB_BACKUP_DIR")), "/var/backups/aurapanel/db")
}

func siteBackupDir() string {
	return firstNonEmpty(strings.TrimSpace(os.Getenv("AURAPANEL_SITE_BACKUP_DIR")), "/var/backups/aurapanel/sites")
}

func ensureBackupDirectory(path string) error {
	return os.MkdirAll(path, 0o755)
}

func streamCommandToGzipFile(path string, command string, args ...string) error {
	if err := ensureBackupDirectory(filepath.Dir(path)); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	gz := gzip.NewWriter(file)
	defer gz.Close()

	cmd := exec.Command(command, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	if _, err := io.Copy(gz, stdout); err != nil {
		_ = cmd.Wait()
		return err
	}
	errOutput, _ := io.ReadAll(stderr)
	if err := gz.Close(); err != nil {
		_ = cmd.Wait()
		return err
	}
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("%s", strings.TrimSpace(string(errOutput)))
	}
	return nil
}

func streamGzipFileToCommand(path string, command string, args ...string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	gz, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gz.Close()

	cmd := exec.Command(command, args...)
	cmd.Stdin = gz
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s", strings.TrimSpace(string(output)))
	}
	return nil
}

func createRuntimeDBBackup(engine, dbName string) (DBBackupRecord, error) {
	engine = normalizeEngine(engine)
	dbName = sanitizeDBName(dbName)
	if dbName == "" {
		return DBBackupRecord{}, fmt.Errorf("database name is required")
	}
	if err := ensureBackupDirectory(dbBackupDir()); err != nil {
		return DBBackupRecord{}, err
	}
	filename := fmt.Sprintf("%s-%s.sql.gz", dbName, time.Now().UTC().Format("20060102-150405"))
	path := filepath.Join(dbBackupDir(), filename)
	switch engine {
	case "mariadb":
		if err := streamCommandToGzipFile(path, "mysqldump", "--single-transaction", "--routines", "--events", dbName); err != nil {
			return DBBackupRecord{}, err
		}
	default:
		if err := streamCommandToGzipFile(path, "runuser", "-u", "postgres", "--", "pg_dump", dbName); err != nil {
			return DBBackupRecord{}, err
		}
	}
	info, err := os.Stat(path)
	if err != nil {
		return DBBackupRecord{}, err
	}
	return DBBackupRecord{
		ID:        generateSecret(8),
		DBName:    dbName,
		Filename:  filename,
		Engine:    engine,
		Size:      formatBytesHuman(info.Size()),
		CreatedAt: info.ModTime().UTC().UnixMilli(),
		Path:      path,
	}, nil
}

func resolveDBBackupPath(record DBBackupRecord) string {
	if strings.TrimSpace(record.Path) != "" {
		return record.Path
	}
	return filepath.Join(dbBackupDir(), record.Filename)
}

func restoreRuntimeDBBackup(record DBBackupRecord) error {
	if record.DBName == "" {
		record.DBName = sanitizeDBName(strings.SplitN(record.Filename, "-", 2)[0])
	}
	path := resolveDBBackupPath(record)
	switch normalizeEngine(record.Engine) {
	case "mariadb":
		return streamGzipFileToCommand(path, "mysql", record.DBName)
	default:
		return streamGzipFileToCommand(path, "runuser", "-u", "postgres", "--", "psql", record.DBName)
	}
}

func listRuntimeDBBackups(existing []DBBackupRecord) ([]DBBackupRecord, error) {
	if err := ensureBackupDirectory(dbBackupDir()); err != nil {
		return nil, err
	}
	meta := make(map[string]DBBackupRecord, len(existing))
	for _, item := range existing {
		meta[item.Filename] = item
	}
	entries, err := os.ReadDir(dbBackupDir())
	if err != nil {
		return nil, err
	}
	backups := make([]DBBackupRecord, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql.gz") {
			continue
		}
		info, infoErr := entry.Info()
		if infoErr != nil {
			return nil, infoErr
		}
		record := DBBackupRecord{
			ID:        generateSecret(8),
			DBName:    sanitizeDBName(strings.SplitN(entry.Name(), "-", 2)[0]),
			Filename:  entry.Name(),
			Engine:    "mariadb",
			Size:      formatBytesHuman(info.Size()),
			CreatedAt: info.ModTime().UTC().UnixMilli(),
			Path:      filepath.Join(dbBackupDir(), entry.Name()),
		}
		if stored, ok := meta[entry.Name()]; ok {
			if stored.ID != "" {
				record.ID = stored.ID
			}
			if stored.DBName != "" {
				record.DBName = stored.DBName
			}
			if stored.Engine != "" {
				record.Engine = stored.Engine
			}
			if stored.Path != "" {
				record.Path = stored.Path
			}
		}
		if strings.HasPrefix(record.Filename, "pg_") || strings.Contains(record.Filename, ".postgres.") {
			record.Engine = "postgresql"
		}
		backups = append(backups, record)
	}
	sort.Slice(backups, func(i, j int) bool { return backups[i].CreatedAt > backups[j].CreatedAt })
	return backups, nil
}

func deleteRuntimeDBBackup(record DBBackupRecord) error {
	return os.Remove(resolveDBBackupPath(record))
}

func createRuntimeSiteBackup(domain, backupPath string) (BackupSnapshot, error) {
	domain = normalizeDomain(domain)
	if domain == "" {
		return BackupSnapshot{}, fmt.Errorf("domain is required")
	}
	docroot := domainDocroot(domain)
	if !fileExists(docroot) {
		return BackupSnapshot{}, fmt.Errorf("docroot not found")
	}
	targetDir := strings.TrimSpace(backupPath)
	if targetDir == "" {
		targetDir = siteBackupDir()
	}
	if err := ensureBackupDirectory(targetDir); err != nil {
		return BackupSnapshot{}, err
	}
	filename := fmt.Sprintf("%s-%s.tar.gz", strings.ReplaceAll(domain, ".", "_"), time.Now().UTC().Format("20060102-150405"))
	target := filepath.Join(targetDir, filename)
	parent := filepath.Dir(docroot)
	base := filepath.Base(docroot)
	if _, err := commandOutputTrimmed("tar", "-czf", target, "-C", parent, base); err != nil {
		return BackupSnapshot{}, err
	}
	hostname, _ := os.Hostname()
	return BackupSnapshot{
		ID:         generateSecret(8),
		ShortID:    generateSecret(4),
		Time:       time.Now().UTC().Format(time.RFC3339),
		Hostname:   firstNonEmpty(hostname, "aurapanel"),
		Tags:       []string{"website", domain},
		Domain:     domain,
		BackupPath: target,
	}, nil
}

func restoreRuntimeSiteBackup(snapshot BackupSnapshot, domain string) error {
	domain = normalizeDomain(firstNonEmpty(domain, snapshot.Domain))
	if domain == "" {
		return fmt.Errorf("domain is required")
	}
	if !fileExists(snapshot.BackupPath) {
		return fmt.Errorf("backup snapshot not found")
	}
	docroot := domainDocroot(domain)
	if err := ensureBackupDirectory(filepath.Dir(docroot)); err != nil {
		return err
	}
	if _, err := commandOutputTrimmed("tar", "-xzf", snapshot.BackupPath, "-C", filepath.Dir(docroot)); err != nil {
		return err
	}
	return nil
}
