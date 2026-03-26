use serde::{Deserialize, Serialize};
use std::fs;
use std::path::Path;

/// Panel kullanıcı veri yapısı
#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct PanelUser {
    pub id: u64,
    pub username: String,
    pub email: String,
    pub role: String,       // "user", "reseller", "admin"
    pub package: String,
    pub sites: u32,
    pub active: bool,
}

/// Kullanıcı oluşturma isteği
#[derive(Serialize, Deserialize, Debug)]
pub struct CreateUserRequest {
    pub username: String,
    pub email: String,
    pub password: String,
    pub role: String,
    pub package: String,
}

const USERS_DB_PATH: &str = "/var/lib/aurapanel/users.json";

pub struct UserManager;

impl UserManager {
    fn dev_mode() -> bool {
        !Path::new("/var/lib/aurapanel").exists()
    }

    /// Kullanıcıları listele
    pub fn list_users() -> Result<Vec<PanelUser>, String> {
        if Self::dev_mode() {
            return Ok(vec![
                PanelUser { id: 1, username: "admin".into(), email: "admin@aurapanel.com".into(), role: "admin".into(), package: "Sınırsız".into(), sites: 0, active: true },
                PanelUser { id: 2, username: "musteri1".into(), email: "musteri@example.com".into(), role: "user".into(), package: "Başlangıç Hosting".into(), sites: 2, active: true },
                PanelUser { id: 3, username: "bayi1".into(), email: "bayi@partner.com".into(), role: "reseller".into(), package: "Bayi V1".into(), sites: 12, active: true },
            ]);
        }

        let json_str = fs::read_to_string(USERS_DB_PATH)
            .unwrap_or_else(|_| "[]".to_string());
        serde_json::from_str(&json_str)
            .map_err(|e| format!("Kullanıcı listesi okunamadı: {}", e))
    }

    /// Kullanıcı oluştur
    pub fn create_user(req: &CreateUserRequest) -> Result<String, String> {
        if Self::dev_mode() {
            return Ok(format!("[Dev Mode] Kullanıcı '{}' oluşturuldu.", req.username));
        }

        let mut users = Self::list_users()?;
        let new_id = users.iter().map(|u| u.id).max().unwrap_or(0) + 1;
        users.push(PanelUser {
            id: new_id,
            username: req.username.clone(),
            email: req.email.clone(),
            role: req.role.clone(),
            package: req.package.clone(),
            sites: 0,
            active: true,
        });
        Self::save_users(&users)?;

        // Sistem kullanıcısı oluştur (Linux)
        let _ = std::process::Command::new("useradd")
            .args(&["-m", "-s", "/bin/bash", &req.username])
            .output();

        Ok(format!("Kullanıcı '{}' başarıyla oluşturuldu.", req.username))
    }

    /// Kullanıcı sil
    pub fn delete_user(username: &str) -> Result<String, String> {
        if Self::dev_mode() {
            return Ok(format!("[Dev Mode] Kullanıcı '{}' silindi.", username));
        }

        let mut users = Self::list_users()?;
        let before = users.len();
        users.retain(|u| u.username != username);
        if users.len() == before {
            return Err(format!("Kullanıcı '{}' bulunamadı.", username));
        }
        Self::save_users(&users)?;

        // Sistem kullanıcısını sil
        let _ = std::process::Command::new("userdel")
            .args(&["-r", username])
            .output();

        Ok(format!("Kullanıcı '{}' başarıyla silindi.", username))
    }

    fn save_users(users: &[PanelUser]) -> Result<(), String> {
        fs::create_dir_all("/var/lib/aurapanel")
            .map_err(|e| format!("Dizin oluşturulamadı: {}", e))?;
        let json = serde_json::to_string_pretty(users)
            .map_err(|e| format!("JSON hatası: {}", e))?;
        fs::write(USERS_DB_PATH, json)
            .map_err(|e| format!("Dosya yazılamadı: {}", e))
    }
}
