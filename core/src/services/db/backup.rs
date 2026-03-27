use chrono::Utc;
use serde::{Deserialize, Serialize};
use std::fs;
use std::path::{Path, PathBuf};
use std::process::Command;

#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct DbBackupEntry {
    pub id: String,
    pub db_name: String,
    pub engine: String,
    pub filename: String,
    pub size_bytes: u64,
    pub created_at: String,
}

#[derive(Serialize, Deserialize, Debug)]
pub struct DbBackupRequest {
    pub db_name: String,
    pub engine: String,
}

#[derive(Serialize, Deserialize, Debug)]
pub struct DbRestoreRequest {
    pub backup_id: String,
}

pub struct DbBackupManager;

impl DbBackupManager {
    fn backup_dir() -> PathBuf {
        PathBuf::from("/var/backups/aurapanel/databases")
    }

    fn index_path() -> PathBuf {
        let state_dir = if let Ok(dir) = std::env::var("AURAPANEL_STATE_DIR") {
            PathBuf::from(dir)
        } else if Path::new("/var/lib/aurapanel").exists() {
            PathBuf::from("/var/lib/aurapanel")
        } else {
            std::env::temp_dir().join("aurapanel")
        };
        state_dir.join("db_backups.json")
    }

    fn load_index() -> Vec<DbBackupEntry> {
        if let Ok(data) = fs::read_to_string(Self::index_path()) {
            serde_json::from_str(&data).unwrap_or_default()
        } else {
            Vec::new()
        }
    }

    fn save_index(entries: &[DbBackupEntry]) -> Result<(), String> {
        let path = Self::index_path();
        if let Some(parent) = path.parent() {
            fs::create_dir_all(parent).map_err(|e| e.to_string())?;
        }
        let json = serde_json::to_string_pretty(entries).map_err(|e| e.to_string())?;
        fs::write(path, json).map_err(|e| e.to_string())
    }

    /// Runs mysqldump/pg_dump and stores the resulting .sql.gz file.
    pub fn create_backup(req: &DbBackupRequest) -> Result<DbBackupEntry, String> {
        let backup_dir = Self::backup_dir();
        fs::create_dir_all(&backup_dir)
            .map_err(|e| format!("Backup dizini olusturulamadi: {}", e))?;

        let safe_name: String = req
            .db_name
            .chars()
            .filter(|c| c.is_alphanumeric() || *c == '_' || *c == '-')
            .collect();
        if safe_name.is_empty() {
            return Err("Gecersiz veritabani adi.".to_string());
        }

        let id = format!("{}-{}", safe_name, Utc::now().format("%Y%m%d%H%M%S"));
        let filename = format!("{}.sql.gz", id);
        let filepath = backup_dir.join(&filename);

        match req.engine.as_str() {
            "mariadb" | "mysql" => {
                let dump_cmd = if Path::new("/usr/bin/mariadb-dump").exists() {
                    "mariadb-dump"
                } else if Path::new("/usr/bin/mysqldump").exists() {
                    "mysqldump"
                } else {
                    return Err("mysqldump veya mariadb-dump bulunamadi.".to_string());
                };

                let script = format!(
                    "{} --single-transaction {} | gzip > {}",
                    dump_cmd,
                    safe_name,
                    filepath.display()
                );
                let output = Command::new("sh")
                    .arg("-c")
                    .arg(&script)
                    .output()
                    .map_err(|e| format!("Backup komutu calistirilamadi: {}", e))?;

                if !output.status.success() {
                    return Err(String::from_utf8_lossy(&output.stderr).to_string());
                }
            }
            "postgres" | "postgresql" => {
                if !Path::new("/usr/bin/pg_dump").exists() {
                    return Err("pg_dump bulunamadi.".to_string());
                }
                let script = format!("pg_dump {} | gzip > {}", safe_name, filepath.display());
                let output = Command::new("sh")
                    .arg("-c")
                    .arg(&script)
                    .output()
                    .map_err(|e| format!("Backup komutu calistirilamadi: {}", e))?;

                if !output.status.success() {
                    return Err(String::from_utf8_lossy(&output.stderr).to_string());
                }
            }
            other => return Err(format!("Desteklenmeyen engine: {}", other)),
        }

        let size_bytes = fs::metadata(&filepath).map(|m| m.len()).unwrap_or(0);

        let entry = DbBackupEntry {
            id: id.clone(),
            db_name: req.db_name.clone(),
            engine: req.engine.clone(),
            filename,
            size_bytes,
            created_at: Utc::now().to_rfc3339(),
        };

        let mut index = Self::load_index();
        index.push(entry.clone());
        Self::save_index(&index)?;

        Ok(entry)
    }

    pub fn list_backups() -> Vec<DbBackupEntry> {
        let mut entries = Self::load_index();
        entries.reverse();
        entries
    }

    pub fn delete_backup(backup_id: &str) -> Result<(), String> {
        let mut entries = Self::load_index();
        let pos = entries
            .iter()
            .position(|e| e.id == backup_id)
            .ok_or_else(|| format!("Backup '{}' bulunamadi.", backup_id))?;

        let entry = entries.remove(pos);
        let filepath = Self::backup_dir().join(&entry.filename);
        if filepath.exists() {
            fs::remove_file(&filepath).map_err(|e| format!("Backup dosyasi silinemedi: {}", e))?;
        }
        Self::save_index(&entries)
    }

    pub fn restore_backup(backup_id: &str) -> Result<String, String> {
        let entries = Self::load_index();
        let entry = entries
            .iter()
            .find(|e| e.id == backup_id)
            .ok_or_else(|| format!("Backup '{}' bulunamadi.", backup_id))?;

        let filepath = Self::backup_dir().join(&entry.filename);
        if !filepath.exists() {
            return Err("Backup dosyasi mevcut degil.".to_string());
        }

        let safe_name: String = entry
            .db_name
            .chars()
            .filter(|c| c.is_alphanumeric() || *c == '_' || *c == '-')
            .collect();

        match entry.engine.as_str() {
            "mariadb" | "mysql" => {
                let script = format!("zcat {} | mysql {}", filepath.display(), safe_name);
                let output = Command::new("sh")
                    .arg("-c")
                    .arg(&script)
                    .output()
                    .map_err(|e| format!("Restore komutu calistirilamadi: {}", e))?;

                if !output.status.success() {
                    return Err(String::from_utf8_lossy(&output.stderr).to_string());
                }
            }
            "postgres" | "postgresql" => {
                let script = format!("zcat {} | psql {}", filepath.display(), safe_name);
                let output = Command::new("sh")
                    .arg("-c")
                    .arg(&script)
                    .output()
                    .map_err(|e| format!("Restore komutu calistirilamadi: {}", e))?;

                if !output.status.success() {
                    return Err(String::from_utf8_lossy(&output.stderr).to_string());
                }
            }
            other => return Err(format!("Desteklenmeyen engine: {}", other)),
        }

        Ok(format!(
            "'{}' veritabani basariyla geri yuklendi.",
            entry.db_name
        ))
    }

    pub fn backup_file_path(backup_id: &str) -> Result<PathBuf, String> {
        let entries = Self::load_index();
        let entry = entries
            .iter()
            .find(|e| e.id == backup_id)
            .ok_or_else(|| format!("Backup '{}' bulunamadi.", backup_id))?;
        Ok(Self::backup_dir().join(&entry.filename))
    }
}
