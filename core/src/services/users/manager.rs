use crate::services::cgroups::CGroupManager;
use crate::services::packages::PackageManager;
use bcrypt::{hash, verify, DEFAULT_COST};
use serde::{Deserialize, Serialize};
use std::fs;
#[cfg(unix)]
use std::os::unix::fs::PermissionsExt;
use std::path::{Path, PathBuf};
use std::process::Command;

#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct PanelUser {
    pub id: u64,
    pub username: String,
    pub email: String,
    pub role: String,
    pub package: String,
    pub sites: u32,
    pub active: bool,
    #[serde(default)]
    pub password_hash: String,
    #[serde(default)]
    pub totp_enabled: bool,
    #[serde(default)]
    pub totp_secret: Option<String>,
}

#[derive(Serialize, Deserialize, Debug)]
pub struct CreateUserRequest {
    pub username: String,
    pub email: String,
    pub password: String,
    pub role: String,
    pub package: String,
}

#[derive(Serialize, Deserialize, Debug)]
pub struct ChangePasswordRequest {
    pub username: String,
    pub current_password: Option<String>, // None when admin resets
    pub new_password: String,
}

pub struct UserManager;

impl UserManager {
    pub fn list_users() -> Result<Vec<PanelUser>, String> {
        let path = users_db_path();
        if !path.exists() {
            return Ok(Vec::new());
        }

        let json_str =
            fs::read_to_string(path).map_err(|e| format!("Kullanici listesi okunamadi: {}", e))?;
        serde_json::from_str(&json_str)
            .map_err(|e| format!("Kullanici listesi parse edilemedi: {}", e))
    }

    pub fn create_user(req: &CreateUserRequest) -> Result<String, String> {
        let username = sanitize_username(&req.username)
            .ok_or_else(|| "Gecerli username zorunludur".to_string())?;
        let email = req.email.trim().to_ascii_lowercase();
        if email.is_empty() || !email.contains('@') {
            return Err("Gecerli email zorunludur".to_string());
        }
        if req.password.len() < 8 {
            return Err("Sifre en az 8 karakter olmalidir.".to_string());
        }

        let role = normalize_role(&req.role);
        let package = if req.package.trim().is_empty() {
            "default".to_string()
        } else {
            req.package.trim().to_string()
        };

        let mut users = Self::list_users()?;
        if users.iter().any(|u| u.username == username) {
            return Err(format!("Kullanici '{}' zaten mevcut.", username));
        }

        let password_hash = hash(&req.password, DEFAULT_COST)
            .map_err(|e| format!("Sifre hashleme basarisiz: {}", e))?;

        let new_id = users.iter().map(|u| u.id).max().unwrap_or(0) + 1;
        users.push(PanelUser {
            id: new_id,
            username: username.clone(),
            email,
            role,
            package: package.clone(),
            sites: 0,
            active: true,
            password_hash,
            totp_enabled: false,
            totp_secret: None,
        });
        save_users(&users)?;

        // Create the system user on Linux
        if !cfg!(windows) {
            let _ = Command::new("useradd")
                .args(["-m", "-s", "/bin/bash", &username])
                .output();

            // OLS worker'in /home/<user>/public_html altina ulasabilmesi icin
            // home dizininde traverse (execute) biti acik olmali.
            let _ = ensure_home_traversable(&username);

            if let Ok(Some(pkg)) = PackageManager::get_package_by_name(&package) {
                let _ =
                    CGroupManager::apply_limits(&username, pkg.cpu_limit, pkg.ram_mb, pkg.io_limit);
            }
        }

        Ok(format!("Kullanici '{}' basariyla olusturuldu.", username))
    }

    /// Deletes a panel user and cascades: removes their vhosts, home directory,
    /// and system user account.
    pub fn delete_user(username: &str) -> Result<String, String> {
        let username =
            sanitize_username(username).ok_or_else(|| "Gecerli username zorunludur".to_string())?;

        let mut users = Self::list_users()?;
        let before = users.len();
        users.retain(|u| u.username != username);
        if users.len() == before {
            return Err(format!("Kullanici '{}' bulunamadi.", username));
        }
        save_users(&users)?;

        if !cfg!(windows) {
            // Remove OLS vhost configs for all domains owned by this user
            let vhost_conf_base = "/usr/local/lsws/conf/vhosts";
            let home_dir = format!("/home/{}/public_html", username);
            if let Ok(entries) = fs::read_dir(&home_dir) {
                for entry in entries.flatten() {
                    let domain = entry.file_name().to_string_lossy().to_string();
                    let conf_dir = format!("{}/{}", vhost_conf_base, domain);
                    if Path::new(&conf_dir).exists() {
                        let _ = fs::remove_dir_all(&conf_dir);
                    }
                }
            }

            // Reload OLS so removed vhosts take effect
            let _ = Command::new("/usr/local/lsws/bin/lswsctrl")
                .arg("restart")
                .output();

            // Remove system user and their home directory
            let _ = Command::new("userdel").args(["-r", &username]).output();
        }

        Ok(format!(
            "Kullanici '{}' ve ilgili tum veriler basariyla silindi.",
            username
        ))
    }

    /// Verifies a user's password. Used by the auth layer.
    pub fn verify_password(username: &str, password: &str) -> Result<bool, String> {
        let user =
            Self::find_by_identity(username)?.ok_or_else(|| "Kullanici bulunamadi.".to_string())?;

        if user.password_hash.is_empty() {
            return Err("Bu kullanicinin sifresi ayarlanmamis.".to_string());
        }

        verify(password, &user.password_hash).map_err(|e| format!("Sifre dogrulama hatasi: {}", e))
    }

    /// Changes a user's panel password. When `current_password` is None the
    /// caller must be an admin performing a forced reset.
    pub fn change_password(req: &ChangePasswordRequest) -> Result<String, String> {
        let user = Self::find_by_identity(&req.username)?
            .ok_or_else(|| format!("Kullanici '{}' bulunamadi.", req.username))?;

        if req.new_password.len() < 8 {
            return Err("Yeni sifre en az 8 karakter olmalidir.".to_string());
        }

        let mut users = Self::list_users()?;
        let pos = users
            .iter()
            .position(|u| u.username == user.username)
            .ok_or_else(|| format!("Kullanici '{}' bulunamadi.", req.username))?;

        // Validate current password if provided
        if let Some(current) = &req.current_password {
            let current_hash = &users[pos].password_hash;
            let valid = verify(current, current_hash)
                .map_err(|e| format!("Sifre dogrulama hatasi: {}", e))?;
            if !valid {
                return Err("Mevcut sifre yanlis.".to_string());
            }
        }

        let new_hash = hash(&req.new_password, DEFAULT_COST)
            .map_err(|e| format!("Sifre hashleme basarisiz: {}", e))?;

        users[pos].password_hash = new_hash;
        save_users(&users)?;

        Ok(format!(
            "'{}' kullanicisinin sifresi basariyla degistirildi.",
            user.username
        ))
    }

    pub fn find_by_identity(identity: &str) -> Result<Option<PanelUser>, String> {
        let identity = identity.trim().to_ascii_lowercase();
        if identity.is_empty() {
            return Ok(None);
        }

        let users = Self::list_users()?;
        Ok(users.into_iter().find(|u| {
            u.username.eq_ignore_ascii_case(&identity) || u.email.eq_ignore_ascii_case(&identity)
        }))
    }

    pub fn get_user(username: &str) -> Result<Option<PanelUser>, String> {
        Self::find_by_identity(username)
    }

    pub fn set_totp_secret(username: &str, secret: &str) -> Result<(), String> {
        let user = Self::find_by_identity(username)?
            .ok_or_else(|| format!("Kullanici '{}' bulunamadi.", username))?;
        let mut users = Self::list_users()?;
        let pos = users
            .iter()
            .position(|u| u.username == user.username)
            .ok_or_else(|| format!("Kullanici '{}' bulunamadi.", username))?;

        users[pos].totp_secret = Some(secret.trim().to_string());
        users[pos].totp_enabled = false;
        save_users(&users)
    }

    pub fn enable_totp(username: &str) -> Result<(), String> {
        let user = Self::find_by_identity(username)?
            .ok_or_else(|| format!("Kullanici '{}' bulunamadi.", username))?;
        let mut users = Self::list_users()?;
        let pos = users
            .iter()
            .position(|u| u.username == user.username)
            .ok_or_else(|| format!("Kullanici '{}' bulunamadi.", username))?;

        if users[pos]
            .totp_secret
            .as_deref()
            .unwrap_or("")
            .trim()
            .is_empty()
        {
            return Err("2FA secret bulunamadi.".to_string());
        }

        users[pos].totp_enabled = true;
        save_users(&users)
    }

    pub fn seed_default_admin() -> Result<(), String> {
        let users = Self::list_users().unwrap_or_default();
        if users.is_empty() {
            let email = std::env::var("AURAPANEL_ADMIN_EMAIL")
                .unwrap_or_else(|_| "admin@server.com".to_string());
            let password = std::env::var("AURAPANEL_ADMIN_PASSWORD").unwrap_or_else(|_| {
                let password_file = "/opt/aurapanel/logs/initial_password.txt";
                if let Ok(raw) = std::fs::read_to_string(password_file) {
                    raw.trim().to_string()
                } else {
                    "password123".to_string()
                }
            });

            let req = CreateUserRequest {
                username: "admin".to_string(),
                email,
                password,
                role: "admin".to_string(),
                package: "default".to_string(),
            };

            let _ = Self::create_user(&req);
            tracing::info!("Default admin user 'admin' seeded into users.json.");
        }
        Ok(())
    }
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

fn users_db_path() -> PathBuf {
    state_root().join("users.json")
}

#[cfg(unix)]
fn ensure_home_traversable(username: &str) -> Result<(), String> {
    let home = format!("/home/{}", username);
    if !Path::new(&home).exists() {
        return Ok(());
    }

    fs::set_permissions(&home, fs::Permissions::from_mode(0o711))
        .map_err(|e| format!("Home izinleri ayarlanamadi ({}): {}", home, e))
}

#[cfg(not(unix))]
fn ensure_home_traversable(_username: &str) -> Result<(), String> {
    Ok(())
}

fn save_users(users: &[PanelUser]) -> Result<(), String> {
    let path = users_db_path();
    if let Some(parent) = path.parent() {
        fs::create_dir_all(parent).map_err(|e| format!("Dizin olusturulamadi: {}", e))?;
    }

    let json = serde_json::to_string_pretty(users).map_err(|e| format!("JSON hatasi: {}", e))?;
    fs::write(path, json).map_err(|e| format!("Dosya yazilamadi: {}", e))
}

fn sanitize_username(input: &str) -> Option<String> {
    let cleaned = input
        .trim()
        .to_ascii_lowercase()
        .chars()
        .filter(|c| c.is_ascii_alphanumeric() || *c == '_' || *c == '-')
        .collect::<String>();

    if cleaned.is_empty() || cleaned.len() > 64 {
        None
    } else {
        Some(cleaned)
    }
}

fn normalize_role(role: &str) -> String {
    let role = role.trim().to_ascii_lowercase();
    match role.as_str() {
        "admin" | "reseller" => role,
        _ => "user".to_string(),
    }
}
