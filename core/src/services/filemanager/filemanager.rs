use chrono::{DateTime, Local};
use serde::{Deserialize, Serialize};
use std::fs;
use std::path::{Component, Path, PathBuf};

#[derive(Serialize, Deserialize, Debug)]
pub struct FileItem {
    pub name: String,
    pub is_dir: bool,
    pub size: u64,
    pub permissions: String,
    pub modified: String,
}

pub struct FileManager;

impl FileManager {
    fn allowed_roots() -> Vec<PathBuf> {
        let configured = std::env::var("AURAPANEL_FILEMANAGER_ROOTS").unwrap_or_else(|_| {
            "/home,/var/www,/usr/local/lsws/conf/vhosts,/var/lib/aurapanel".to_string()
        });

        configured
            .split(',')
            .map(|item| item.trim())
            .filter(|item| !item.is_empty())
            .map(PathBuf::from)
            .collect()
    }

    fn format_time(system_time: std::time::SystemTime) -> String {
        let dt: DateTime<Local> = system_time.into();
        dt.to_rfc3339()
    }

    fn normalize_absolute_path(path: &Path) -> Result<PathBuf, String> {
        if !path.is_absolute() {
            return Err("Yalnizca absolute path kabul edilir.".to_string());
        }

        let mut normalized = PathBuf::new();
        for component in path.components() {
            match component {
                Component::RootDir => normalized.push(Path::new("/")),
                Component::Prefix(prefix) => normalized.push(prefix.as_os_str()),
                Component::CurDir => {}
                Component::ParentDir => {
                    normalized.pop();
                }
                Component::Normal(value) => normalized.push(value),
            }
        }
        Ok(normalized)
    }

    fn canonical_or_normalized(path: &Path) -> Result<PathBuf, String> {
        if path.exists() {
            fs::canonicalize(path).map_err(|e| format!("Path dogrulanamadi: {}", e))
        } else {
            Self::normalize_absolute_path(path)
        }
    }

    fn ensure_allowed(path: &Path) -> Result<PathBuf, String> {
        let candidate = Self::canonical_or_normalized(path)?;
        for root in Self::allowed_roots() {
            let root_resolved = Self::canonical_or_normalized(&root)?;
            if candidate.starts_with(&root_resolved) {
                return Ok(candidate);
            }
        }
        Err("Bu path panel sandbox disinda.".to_string())
    }

    fn resolve_read_path(path: &str) -> Result<PathBuf, String> {
        let raw = PathBuf::from(path.trim());
        Self::ensure_allowed(&raw)
    }

    fn resolve_write_path(path: &str) -> Result<PathBuf, String> {
        let raw = PathBuf::from(path.trim());
        let normalized = Self::normalize_absolute_path(&raw)?;
        let parent = normalized
            .parent()
            .ok_or_else(|| "Gecersiz hedef path.".to_string())?;
        Self::ensure_allowed(parent)?;
        Ok(normalized)
    }

    #[cfg(unix)]
    fn get_permissions(meta: &fs::Metadata) -> String {
        use std::os::unix::fs::PermissionsExt;
        let mode = meta.permissions().mode();
        let user = format!(
            "{}{}{}",
            if mode & 0o400 != 0 { "r" } else { "-" },
            if mode & 0o200 != 0 { "w" } else { "-" },
            if mode & 0o100 != 0 { "x" } else { "-" }
        );
        let group = format!(
            "{}{}{}",
            if mode & 0o040 != 0 { "r" } else { "-" },
            if mode & 0o020 != 0 { "w" } else { "-" },
            if mode & 0o010 != 0 { "x" } else { "-" }
        );
        let other = format!(
            "{}{}{}",
            if mode & 0o004 != 0 { "r" } else { "-" },
            if mode & 0o002 != 0 { "w" } else { "-" },
            if mode & 0o001 != 0 { "x" } else { "-" }
        );
        let dir = if meta.is_dir() { "d" } else { "-" };
        format!("{}{}{}{}", dir, user, group, other)
    }

    #[cfg(not(unix))]
    fn get_permissions(meta: &fs::Metadata) -> String {
        let dir = if meta.is_dir() { "d" } else { "-" };
        let ro = if meta.permissions().readonly() {
            "r--r--r--"
        } else {
            "rw-r--r--"
        };
        format!("{}{}", dir, ro)
    }

    pub fn list_dir(path: &str) -> Result<Vec<FileItem>, String> {
        let target_path = Self::resolve_read_path(path)?;
        if !target_path.is_dir() {
            return Err("Hedef bir dizin degil.".to_string());
        }

        let mut items = Vec::new();
        let entries =
            fs::read_dir(&target_path).map_err(|e| format!("Dizin okunamadi: {}", e))?;

        for entry in entries.flatten() {
            if let Ok(meta) = entry.metadata() {
                let is_dir = meta.is_dir();
                let modified = meta
                    .modified()
                    .map(Self::format_time)
                    .unwrap_or_default();

                items.push(FileItem {
                    name: entry.file_name().to_string_lossy().to_string(),
                    is_dir,
                    size: if is_dir { 0 } else { meta.len() },
                    permissions: Self::get_permissions(&meta),
                    modified,
                });
            }
        }

        items.sort_by(|a, b| match b.is_dir.cmp(&a.is_dir) {
            std::cmp::Ordering::Equal => a.name.to_lowercase().cmp(&b.name.to_lowercase()),
            other => other,
        });

        Ok(items)
    }

    pub fn read_file(path: &str) -> Result<String, String> {
        let path = Self::resolve_read_path(path)?;
        fs::read_to_string(path).map_err(|e| format!("Dosya okunamadi: {}", e))
    }

    pub fn write_file(path: &str, content: &str) -> Result<(), String> {
        let path = Self::resolve_write_path(path)?;
        fs::write(path, content).map_err(|e| format!("Dosyaya yazilamadi: {}", e))
    }

    pub fn create_dir(path: &str) -> Result<(), String> {
        let path = Self::resolve_write_path(path)?;
        fs::create_dir_all(path).map_err(|e| format!("Klasor olusturulamadi: {}", e))
    }

    pub fn delete_item(path: &str) -> Result<(), String> {
        let path = Self::resolve_read_path(path)?;
        if !path.exists() {
            return Err("Dosya veya klasor bulunamadi.".to_string());
        }
        if path.is_dir() {
            fs::remove_dir_all(path).map_err(|e| format!("Klasor silinemedi: {}", e))
        } else {
            fs::remove_file(path).map_err(|e| format!("Dosya silinemedi: {}", e))
        }
    }

    pub fn trash_item(path: &str) -> Result<(), String> {
        let target_path = Self::resolve_read_path(path)?;
        if !target_path.exists() {
            return Err("Dosya veya klasor bulunamadi.".to_string());
        }

        // Find which allowed root this file belongs to
        let mut matched_root = None;
        for root in Self::allowed_roots() {
            if let Ok(root_resolved) = Self::canonical_or_normalized(&root) {
                if target_path.starts_with(&root_resolved) {
                    matched_root = Some(root_resolved);
                    break;
                }
            }
        }

        let root = matched_root.ok_or_else(|| "Dosya sandbox disinda.".to_string())?;
        let trash_dir = root.join(".trash");
        
        if !trash_dir.exists() {
            fs::create_dir_all(&trash_dir).map_err(|e| format!("Cop kutusu olusturulamadi: {}", e))?;
        }

        let file_name = target_path.file_name().unwrap_or_default().to_string_lossy();
        let timestamp = Local::now().format("%Y%m%d_%H%M%S");
        let trash_target = trash_dir.join(format!("{}_{}", file_name, timestamp));

        fs::rename(&target_path, &trash_target).map_err(|e| format!("Cop kutusuna tasinamadi: {}", e))
    }

    pub fn compress_items(format: &str, dest: &str, sources: Vec<String>) -> Result<(), String> {
        let dest_path = Self::resolve_write_path(dest)?;
        if sources.is_empty() {
            return Err("Sikistirilacak dosya secilmedi.".to_string());
        }

        let mut resolved_sources = Vec::new();
        let mut parent_dir = None;
        for src in sources {
            let p = Self::resolve_read_path(&src)?;
            if parent_dir.is_none() {
                parent_dir = p.parent().map(|p| p.to_path_buf());
            }
            resolved_sources.push(p.file_name().unwrap_or_default().to_string_lossy().to_string());
        }

        let working_dir = parent_dir.ok_or_else(|| "Gecersiz kaynak dizini.".to_string())?;

        let (cmd, args) = match format.to_lowercase().as_str() {
            "zip" => {
                let mut args = vec!["-r".to_string(), dest_path.to_string_lossy().to_string()];
                args.extend(resolved_sources);
                ("zip", args)
            }
            "tar.gz" | "tgz" => {
                let mut args = vec!["-czf".to_string(), dest_path.to_string_lossy().to_string()];
                args.extend(resolved_sources);
                ("tar", args)
            }
            _ => return Err("Desteklenmeyen format (zip veya tar.gz olmali).".to_string()),
        };

        let output = std::process::Command::new(cmd)
            .current_dir(&working_dir)
            .args(&args)
            .output()
            .map_err(|e| format!("Sikistirma araci baslatilamadi: {}", e))?;

        if !output.status.success() {
            let err_msg = String::from_utf8_lossy(&output.stderr);
            return Err(format!("Sikistirma basarisiz: {}", err_msg));
        }

        Ok(())
    }

    pub fn extract_item(archive: &str, dest_dir: &str) -> Result<(), String> {
        let archive_path = Self::resolve_read_path(archive)?;
        let dest_path = Self::resolve_write_path(dest_dir)?;

        if !dest_path.exists() {
            fs::create_dir_all(&dest_path).map_err(|e| format!("Hedef dizin olusturulamadi: {}", e))?;
        }

        let archive_str = archive_path.to_string_lossy().to_lowercase();
        let (cmd, args) = if archive_str.ends_with(".zip") {
            ("unzip", vec!["-o".to_string(), archive_path.to_string_lossy().to_string(), "-d".to_string(), dest_path.to_string_lossy().to_string()])
        } else if archive_str.ends_with(".tar.gz") || archive_str.ends_with(".tgz") {
            ("tar", vec!["-xzf".to_string(), archive_path.to_string_lossy().to_string(), "-C".to_string(), dest_path.to_string_lossy().to_string()])
        } else {
            return Err("Desteklenmeyen arsiv formati (zip veya tar.gz olmali).".to_string());
        };

        let output = std::process::Command::new(cmd)
            .args(&args)
            .output()
            .map_err(|e| format!("Cikarma araci baslatilamadi: {}", e))?;

        if !output.status.success() {
            let err_msg = String::from_utf8_lossy(&output.stderr);
            return Err(format!("Cikarma basarisiz: {}", err_msg));
        }

        Ok(())
    }

    pub fn rename_item(old_path: &str, new_path: &str) -> Result<(), String> {
        let old_path = Self::resolve_read_path(old_path)?;
        let new_path = Self::resolve_write_path(new_path)?;
        fs::rename(old_path, new_path).map_err(|e| format!("Yeniden adlandirilamadi: {}", e))
    }
}
