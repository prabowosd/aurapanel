use serde::{Deserialize, Serialize};
use std::fs;
use std::path::{Path, PathBuf};
use std::process::Command;
use std::time::{SystemTime, UNIX_EPOCH};

#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct SftpUserConfig {
    pub username: String,
    pub password: String,
    pub home_dir: String,
}

#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct SftpUserRecord {
    pub username: String,
    pub home_dir: String,
    pub created_at: u64,
}

#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct SftpPasswordResetRequest {
    pub username: String,
    pub new_password: String,
}

fn state_root() -> PathBuf {
    if let Ok(path) = std::env::var("AURAPANEL_STATE_DIR") {
        let p = PathBuf::from(path.trim());
        if !p.as_os_str().is_empty() {
            return p;
        }
    }
    let prod = Path::new("/var/lib/aurapanel");
    if prod.exists() {
        prod.to_path_buf()
    } else {
        std::env::temp_dir().join("aurapanel")
    }
}

fn state_path() -> PathBuf {
    state_root().join("sftp_users.json")
}

fn now_ts() -> u64 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map(|d| d.as_secs())
        .unwrap_or(0)
}

fn simulation_enabled() -> bool {
    crate::runtime::simulation_enabled() || cfg!(windows)
}

fn load_users() -> Result<Vec<SftpUserRecord>, String> {
    let path = state_path();
    if !path.exists() {
        return Ok(Vec::new());
    }
    let raw = fs::read_to_string(path).map_err(|e| e.to_string())?;
    serde_json::from_str(&raw).map_err(|e| e.to_string())
}

fn save_users(users: &[SftpUserRecord]) -> Result<(), String> {
    let path = state_path();
    if let Some(parent) = path.parent() {
        fs::create_dir_all(parent).map_err(|e| e.to_string())?;
    }
    let payload = serde_json::to_string_pretty(users).map_err(|e| e.to_string())?;
    fs::write(path, payload).map_err(|e| e.to_string())
}

pub struct SecureConnectManager;

impl SecureConnectManager {
    pub async fn create_sftp_user(config: &SftpUserConfig) -> Result<(), String> {
        let username = config.username.trim().to_string();
        let password = config.password.trim().to_string();
        let home_dir = config.home_dir.trim().to_string();

        if username.is_empty() || password.is_empty() || home_dir.is_empty() {
            return Err("username, password and home_dir are required.".to_string());
        }

        if !simulation_enabled() {
            let _ = Command::new("useradd")
                .args([
                    "-m",
                    "-d", &home_dir,
                    "-s", "/usr/sbin/nologin",
                    "-G", "sftponly",
                    &username,
                ])
                .output()
                .map_err(|e| format!("useradd failed: {}", e))?;

            let _ = Command::new("sh")
                .args(["-c", &format!("echo '{}:{}' | chpasswd", username, password)])
                .output()
                .map_err(|e| format!("chpasswd failed: {}", e))?;

            let _ = Command::new("chown").args(["root:root", &home_dir]).output();
            let _ = Command::new("chmod").args(["755", &home_dir]).output();
        }

        let mut users = load_users()?;
        if !users.iter().any(|u| u.username == username) {
            users.push(SftpUserRecord {
                username,
                home_dir,
                created_at: now_ts(),
            });
            save_users(&users)?;
        }
        Ok(())
    }

    pub fn list_sftp_users() -> Result<Vec<SftpUserRecord>, String> {
        load_users()
    }

    pub fn delete_sftp_user(username: &str) -> Result<(), String> {
        let username = username.trim();
        if username.is_empty() {
            return Err("username is required.".to_string());
        }

        if !simulation_enabled() {
            let _ = Command::new("userdel").args(["-r", username]).output();
        }

        let mut users = load_users()?;
        let before = users.len();
        users.retain(|u| u.username != username);
        if before == users.len() {
            return Err("SFTP user not found.".to_string());
        }
        save_users(&users)
    }

    pub fn reset_sftp_password(req: &SftpPasswordResetRequest) -> Result<(), String> {
        let username = req.username.trim();
        let password = req.new_password.trim();
        if username.is_empty() || password.is_empty() {
            return Err("username and new_password are required.".to_string());
        }

        if !simulation_enabled() {
            let _ = Command::new("sh")
                .args(["-c", &format!("echo '{}:{}' | chpasswd", username, password)])
                .output()
                .map_err(|e| format!("chpasswd failed: {}", e))?;
        }
        Ok(())
    }

    pub async fn start_web_terminal(port: u16) -> Result<String, String> {
        Ok(format!("https://panel.domain.com:{}", port))
    }
}
