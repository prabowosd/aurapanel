use serde::{Deserialize, Serialize};
use std::process::Command;

#[derive(Serialize, Deserialize, Debug)]
pub struct BackupConfig {
    pub domain: String,
    pub backup_path: String,
    pub remote_repo: String, // restic repo URL (S3, MinIO, local)
    pub password: String,
}

pub struct BackupManager;

impl BackupManager {
    /// Restic ile tam yedekleme alır (web dosyaları + veritabanı dump)
    pub async fn create_backup(config: &BackupConfig) -> Result<String, String> {
        println!("[BACKUP] Starting backup for {} -> {}", config.domain, config.remote_repo);

        if !std::path::Path::new("/usr/bin/restic").exists() {
            println!("[DEV MODE] restic not installed. Simulating backup.");
            return Ok("backup-simulated-snapshot-id".to_string());
        }

        // 1. Veritabanı dumpını al
        let db_dump = format!("/tmp/{}_db.sql", config.domain);
        let _ = Command::new("mysqldump")
            .args(["--all-databases", "--result-file", &db_dump])
            .output()
            .map_err(|e| format!("DB dump hatası: {}", e))?;

        // 2. Restic ile yedekle
        let output = Command::new("restic")
            .env("RESTIC_REPOSITORY", &config.remote_repo)
            .env("RESTIC_PASSWORD", &config.password)
            .args(["backup", &config.backup_path, &db_dump, "--tag", &config.domain])
            .output()
            .map_err(|e| format!("restic backup hatası: {}", e))?;

        if !output.status.success() {
            return Err(format!("Backup başarısız: {}", String::from_utf8_lossy(&output.stderr)));
        }

        Ok(String::from_utf8_lossy(&output.stdout).to_string())
    }

    /// Son yedeklerden geri yükler
    pub async fn restore_backup(config: &BackupConfig, snapshot_id: &str) -> Result<(), String> {
        println!("[BACKUP] Restoring snapshot {} for {}", snapshot_id, config.domain);

        if !std::path::Path::new("/usr/bin/restic").exists() {
            println!("[DEV MODE] restic not installed. Simulating restore.");
            return Ok(());
        }

        let _ = Command::new("restic")
            .env("RESTIC_REPOSITORY", &config.remote_repo)
            .env("RESTIC_PASSWORD", &config.password)
            .args(["restore", snapshot_id, "--target", &config.backup_path])
            .output()
            .map_err(|e| format!("Restore hatası: {}", e))?;

        Ok(())
    }
}
