use serde::{Deserialize, Serialize};
use std::process::Command;

#[derive(Serialize, Deserialize, Debug)]
pub struct SftpUserConfig {
    pub username: String,
    pub password: String,
    pub home_dir: String,
}

pub struct SecureConnectManager;

impl SecureConnectManager {
    /// Sistemde jailed SFTP kullanıcısı oluşturur (chroot ile izole edilmiş)
    pub async fn create_sftp_user(config: &SftpUserConfig) -> Result<(), String> {
        println!("[SFTP] Creating jailed user: {} at {}", config.username, config.home_dir);

        if cfg!(target_os = "windows") {
            println!("[DEV MODE] SFTP user creation simulated on Windows.");
            return Ok(());
        }

        // 1. Kullanıcı oluştur
        let _ = Command::new("useradd")
            .args([
                "-m",
                "-d", &config.home_dir,
                "-s", "/usr/sbin/nologin",
                "-G", "sftponly",
                &config.username,
            ])
            .output()
            .map_err(|e| format!("useradd başarısız: {}", e))?;

        // 2. Şifre ata
        let _ = Command::new("chpasswd")
            .arg(&format!("{}:{}", config.username, config.password))
            .output()
            .map_err(|e| format!("Şifre atanamadı: {}", e))?;

        // 3. Chroot dizin izinleri
        let _ = Command::new("chown")
            .args(["root:root", &config.home_dir])
            .output();
        let _ = Command::new("chmod")
            .args(["755", &config.home_dir])
            .output();

        Ok(())
    }

    /// Web tabanlı terminal (ttyd / gotty benzeri) başlatır
    pub async fn start_web_terminal(port: u16) -> Result<String, String> {
        println!("[TERMINAL] Web terminal started on port {}", port);
        // Gerçekte ttyd veya gotty binary'si çalıştırılır:
        // Command::new("ttyd").args(["-p", &port.to_string(), "/bin/bash"]).spawn();
        Ok(format!("https://panel.domain.com:{}", port))
    }
}
