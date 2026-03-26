use serde::{Deserialize, Serialize};
use std::fs;
use std::path::Path;
use chrono::{DateTime, Local};

#[derive(Serialize, Deserialize, Debug)]
pub struct FileItem {
    pub name: String,
    pub r#type: String, // "dir" or "file"
    pub size: String,
    pub permissions: String,
    pub modified: String,
}

pub struct FileManager;

impl FileManager {
    fn format_size(size: u64) -> String {
        const KB: u64 = 1024;
        const MB: u64 = KB * 1024;
        const GB: u64 = MB * 1024;

        if size >= GB {
            format!("{:.2} GB", size as f64 / GB as f64)
        } else if size >= MB {
            format!("{:.2} MB", size as f64 / MB as f64)
        } else if size >= KB {
            format!("{:.2} KB", size as f64 / KB as f64)
        } else {
            format!("{} B", size)
        }
    }

    fn format_time(system_time: std::time::SystemTime) -> String {
        let dt: DateTime<Local> = system_time.into();
        dt.format("%Y-%m-%d %H:%M").to_string()
    }

    // Windows'da tam Unix permission'ları desteklenmediğinden, dummy veya temel permission çevirisi yapıyoruz
    #[cfg(unix)]
    fn get_permissions(meta: &fs::Metadata) -> String {
        use std::os::unix::fs::PermissionsExt;
        let mode = meta.permissions().mode();
        let user = format!("{}{}{}", 
            if mode & 0o400 != 0 { "r" } else { "-" },
            if mode & 0o200 != 0 { "w" } else { "-" },
            if mode & 0o100 != 0 { "x" } else { "-" });
        let group = format!("{}{}{}", 
            if mode & 0o040 != 0 { "r" } else { "-" },
            if mode & 0o020 != 0 { "w" } else { "-" },
            if mode & 0o010 != 0 { "x" } else { "-" });
        let other = format!("{}{}{}", 
            if mode & 0o004 != 0 { "r" } else { "-" },
            if mode & 0o002 != 0 { "w" } else { "-" },
            if mode & 0o001 != 0 { "x" } else { "-" });
        let dir = if meta.is_dir() { "d" } else { "-" };
        format!("{}{}{}{}", dir, user, group, other)
    }

    #[cfg(not(unix))]
    fn get_permissions(meta: &fs::Metadata) -> String {
        let dir = if meta.is_dir() { "d" } else { "-" };
        let ro = if meta.permissions().readonly() { "r--r--r--" } else { "rw-r--r--" };
        format!("{}{}", dir, ro)
    }

    /// Belirtilen dizindeki dosyaları ve klasörleri listeler
    pub fn list_dir(path: &str) -> Result<Vec<FileItem>, String> {
        // Güvenlik: Kullanıcının dışarı çıkmasını engellemek için base path denetimi yapılabilir
        let target_path = Path::new(path);
        
        let mut items = Vec::new();
        let entries = fs::read_dir(target_path).map_err(|e| format!("Dizin okunamadı: {}", e))?;

        for entry in entries {
            if let Ok(entry) = entry {
                if let Ok(meta) = entry.metadata() {
                    let file_name = entry.file_name().to_string_lossy().to_string();
                    let is_dir = meta.is_dir();
                    
                    let size_str = if is_dir {
                        "—".to_string()
                    } else {
                        Self::format_size(meta.len())
                    };

                    let modified_str = match meta.modified() {
                        Ok(sys_time) => Self::format_time(sys_time),
                        Err(_) => "Bilinmiyor".to_string(),
                    };

                    let perm_str = Self::get_permissions(&meta);

                    items.push(FileItem {
                        name: file_name,
                        r#type: if is_dir { "dir".to_string() } else { "file".to_string() },
                        size: size_str,
                        permissions: perm_str,
                        modified: modified_str,
                    });
                }
            }
        }
        
        // Klasörler başta, ardından dosyalar (alfabetik)
        items.sort_by(|a, b| {
            if a.r#type == b.r#type {
                a.name.to_lowercase().cmp(&b.name.to_lowercase())
            } else {
                b.r#type.cmp(&a.r#type) // "dir" "file" dan önde gelir
            }
        });

        Ok(items)
    }

    /// Dosya içeriğini okur
    pub fn read_file(path: &str) -> Result<String, String> {
        fs::read_to_string(path).map_err(|e| format!("Dosya okunamadı: {}", e))
    }

    /// Dosya içeriğini yazar / Dosya oluşturur
    pub fn write_file(path: &str, content: &str) -> Result<(), String> {
        fs::write(path, content).map_err(|e| format!("Dosyaya yazılamadı: {}", e))
    }

    /// Klasör oluşturur
    pub fn create_dir(path: &str) -> Result<(), String> {
        fs::create_dir_all(path).map_err(|e| format!("Klasör oluşturulamadı: {}", e))
    }

    /// Dosya veya klasör siler
    pub fn delete_item(path: &str) -> Result<(), String> {
        let p = Path::new(path);
        if !p.exists() {
            return Err("Dosya veya klasör bulunamadı.".to_string());
        }
        if p.is_dir() {
            fs::remove_dir_all(p).map_err(|e| format!("Klasör silinemedi: {}", e))
        } else {
            fs::remove_file(p).map_err(|e| format!("Dosya silinemedi: {}", e))
        }
    }

    /// Yeniden adlandırır
    pub fn rename_item(old_path: &str, new_path: &str) -> Result<(), String> {
        fs::rename(old_path, new_path).map_err(|e| format!("Yeniden adlandırılamadı: {}", e))
    }
}
