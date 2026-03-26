use serde::{Deserialize, Serialize};
use std::fs;
use std::path::Path;
use std::process::Command;

#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct PhpVersion {
    pub version: String,
    pub installed: bool,
    pub eol: bool,
}

#[derive(Serialize, Deserialize, Debug)]
pub struct PhpExtension {
    pub name: String,
    pub enabled: bool,
}

pub struct PhpManager;

impl PhpManager {
    /// Sistemde PHP kurulu mu dev mode mu kontrolü (Windows vs Linux) 
    fn is_dev_mode() -> bool {
        if cfg!(windows) {
            true
        } else {
            // Linux'ta da apt paket yöneticisi yoksa dev mode kabul et
            !Path::new("/usr/bin/apt").exists()
        }
    }

    /// Desteklenen ve kurulu servisleri listeler
    pub fn list_versions() -> Result<Vec<PhpVersion>, String> {
        let mut versions = vec![
            PhpVersion { version: "7.4".to_string(), installed: false, eol: true },
            PhpVersion { version: "8.0".to_string(), installed: false, eol: true },
            PhpVersion { version: "8.1".to_string(), installed: false, eol: false },
            PhpVersion { version: "8.2".to_string(), installed: false, eol: false },
            PhpVersion { version: "8.3".to_string(), installed: false, eol: false },
            PhpVersion { version: "8.4".to_string(), installed: false, eol: false },
        ];

        if Self::is_dev_mode() {
            // Mock data
            if let Some(v) = versions.iter_mut().find(|v| v.version == "8.2") { v.installed = true; }
            if let Some(v) = versions.iter_mut().find(|v| v.version == "8.3") { v.installed = true; }
            return Ok(versions);
        }

        // Linux Debian/Ubuntu check
        for v in &mut versions {
            let path = format!("/usr/bin/php{}", v.version);
            if Path::new(&path).exists() {
                v.installed = true;
            }
        }

        Ok(versions)
    }

    /// PHP versiyonu kurar
    pub fn install_version(version: &str) -> Result<String, String> {
        if Self::is_dev_mode() {
            return Ok(format!("(Dev Mode) PHP {} başarıyla kuruldu gibi simüle edildi.", version));
        }

        let pkg = format!("php{}-fpm", version);
        let output = Command::new("apt")
            .args(&["install", "-y", &pkg])
            .output()
            .map_err(|e| format!("apt install çalıştırılamadı: {}", e))?;

        if output.status.success() {
            Ok(format!("PHP {} başarıyla kuruldu.", version))
        } else {
            Err(String::from_utf8_lossy(&output.stderr).to_string())
        }
    }

    /// PHP versiyonu kaldırır
    pub fn remove_version(version: &str) -> Result<String, String> {
        if Self::is_dev_mode() {
            return Ok(format!("(Dev Mode) PHP {} başarıyla kaldırıldı gibi simüle edildi.", version));
        }

        let pkg = format!("php{}-fpm", version);
        let output = Command::new("apt")
            .args(&["purge", "-y", &pkg])
            .output()
            .map_err(|e| format!("apt purge çalıştırılamadı: {}", e))?;

        if output.status.success() {
            Ok(format!("PHP {} başarıyla kaldırıldı.", version))
        } else {
            Err(String::from_utf8_lossy(&output.stderr).to_string())
        }
    }

    /// PHP servisini FPM yeniden başlatır
    pub fn restart_fpm(version: &str) -> Result<String, String> {
        if Self::is_dev_mode() {
            return Ok(format!("(Dev Mode) PHP {} FPM restart edildi.", version));
        }

        let srv = format!("php{}-fpm", version);
        let output = Command::new("systemctl")
            .args(&["restart", &srv])
            .output()
            .map_err(|e| format!("systemctl restart çalıştırılamadı: {}", e))?;

        if output.status.success() {
            Ok(format!("PHP {} FPM servisi yeniden başlatıldı.", version))
        } else {
            Err(String::from_utf8_lossy(&output.stderr).to_string())
        }
    }

    /// php.ini dosyasını okur
    pub fn get_ini(version: &str) -> Result<String, String> {
        if Self::is_dev_mode() {
            return Ok(format!("; Dev Mode php.ini for PHP {}\nmemory_limit = 256M\nupload_max_filesize = 64M\npost_max_size = 64M\nmax_execution_time = 300\n", version));
        }

        let path = format!("/etc/php/{}/fpm/php.ini", version);
        fs::read_to_string(&path)
            .map_err(|e| format!("php.ini okunamadı (Path: {}): {}", path, e))
    }

    /// php.ini dosyasını kaydeder
    pub fn save_ini(version: &str, content: &str) -> Result<String, String> {
        if Self::is_dev_mode() {
            return Ok(format!("(Dev Mode) PHP {} php.ini başarıyla kaydedildi.", version));
        }

        let path = format!("/etc/php/{}/fpm/php.ini", version);
        fs::write(&path, content)
            .map_err(|e| format!("php.ini yazılamadı (Path: {}): {}", path, e))?;
        
        // Restart fpm right after save? 
        let _ = Self::restart_fpm(version);

        Ok(format!("PHP {} php.ini başarıyla kaydedildi ve servis yeniden başlatıldı.", version))
    }
}
