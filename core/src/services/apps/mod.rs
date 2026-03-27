pub mod nodejs;
pub mod python;

use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::process::Command;
use std::sync::{Mutex, OnceLock};

use self::nodejs::NodeManager;
use self::python::PythonManager;

#[derive(Serialize, Deserialize, Debug)]
pub struct CmsInstallConfig {
    pub domain: String,
    pub app_type: String, // "wordpress", "laravel"
    pub db_name: String,
    pub db_user: String,
    pub db_pass: String,
    pub owner: Option<String>, // system user that owns the vhost
    pub admin_email: Option<String>,
    pub admin_user: Option<String>,
    pub admin_pass: Option<String>,
    pub site_title: Option<String>,
}

#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct NodeAppRequest {
    pub app_name: String,
    pub start_script: String,
    pub dir: String,
}

#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct PythonAppRequest {
    pub app_name: String,
    pub wsgi_module: String,
    pub dir: String,
    pub port: u16,
}

#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct RuntimeAppInfo {
    pub app_name: String,
    pub runtime: String,
    pub dir: String,
    pub status: String,
}

#[derive(Default)]
struct RuntimeState {
    apps: HashMap<String, RuntimeAppInfo>,
}

fn runtime_state() -> &'static Mutex<RuntimeState> {
    static STATE: OnceLock<Mutex<RuntimeState>> = OnceLock::new();
    STATE.get_or_init(|| Mutex::new(RuntimeState::default()))
}

fn command_available(cmd: &str) -> bool {
    Command::new("which")
        .arg(cmd)
        .output()
        .map(|o| o.status.success())
        .unwrap_or(false)
}

pub struct AppManager;

impl AppManager {
    /// Tek tıklamayla WordPress veya Laravel kurar.
    /// WordPress: wp core download → wp config create → wp core install
    pub async fn install_cms(config: &CmsInstallConfig) -> Result<(), String> {
        let domain = config.domain.trim().to_lowercase();
        if domain.is_empty() {
            return Err("domain zorunludur.".to_string());
        }

        // Resolve install path: /home/<owner>/public_html/<domain>
        let owner = config
            .owner
            .as_deref()
            .map(|s| s.trim().to_string())
            .filter(|s| !s.is_empty())
            .unwrap_or_else(|| "aura".to_string());
        let public_html = format!("/home/{}/public_html/{}", owner, domain);

        match config.app_type.as_str() {
            "wordpress" => {
                if !command_available("wp") {
                    return Err("wp-cli kurulu degil.".to_string());
                }

                std::fs::create_dir_all(&public_html)
                    .map_err(|e| format!("Dizin olusturulamadi: {}", e))?;

                // 1. Download WordPress core
                let dl = Command::new("wp")
                    .args(["core", "download", "--allow-root", "--path", &public_html])
                    .output()
                    .map_err(|e| format!("wp core download basarisiz: {}", e))?;
                if !dl.status.success() {
                    return Err(String::from_utf8_lossy(&dl.stderr).to_string());
                }

                // 2. Create wp-config.php
                let db_host = "localhost";
                let cfg = Command::new("wp")
                    .args([
                        "config",
                        "create",
                        "--allow-root",
                        &format!("--path={}", public_html),
                        &format!("--dbname={}", config.db_name),
                        &format!("--dbuser={}", config.db_user),
                        &format!("--dbpass={}", config.db_pass),
                        &format!("--dbhost={}", db_host),
                    ])
                    .output()
                    .map_err(|e| format!("wp config create basarisiz: {}", e))?;
                if !cfg.status.success() {
                    return Err(String::from_utf8_lossy(&cfg.stderr).to_string());
                }

                // 3. Install WordPress (creates DB tables, sets admin credentials)
                let admin_email = config.admin_email.as_deref().unwrap_or("admin@example.com");
                let admin_user = config.admin_user.as_deref().unwrap_or("admin");
                let admin_pass = config.admin_pass.as_deref().unwrap_or("changeme123!");
                let site_title = config.site_title.as_deref().unwrap_or("My Website");

                let install = Command::new("wp")
                    .args([
                        "core",
                        "install",
                        "--allow-root",
                        &format!("--path={}", public_html),
                        &format!("--url=https://{}", domain),
                        &format!("--title={}", site_title),
                        &format!("--admin_user={}", admin_user),
                        &format!("--admin_password={}", admin_pass),
                        &format!("--admin_email={}", admin_email),
                    ])
                    .output()
                    .map_err(|e| format!("wp core install basarisiz: {}", e))?;
                if !install.status.success() {
                    return Err(String::from_utf8_lossy(&install.stderr).to_string());
                }
            }
            "laravel" => {
                if !command_available("composer") {
                    return Err("composer kurulu degil.".to_string());
                }
                let output = Command::new("composer")
                    .args([
                        "create-project",
                        "--prefer-dist",
                        "--no-interaction",
                        "laravel/laravel",
                        &public_html,
                    ])
                    .output()
                    .map_err(|e| format!("composer basarisiz: {}", e))?;
                if !output.status.success() {
                    return Err(String::from_utf8_lossy(&output.stderr).to_string());
                }
            }
            _ => return Err(format!("Desteklenmeyen uygulama tipi: {}", config.app_type)),
        }

        Ok(())
    }

    pub fn list_runtime_apps() -> Result<Vec<RuntimeAppInfo>, String> {
        let guard = runtime_state().lock().map_err(|e| e.to_string())?;
        Ok(guard.apps.values().cloned().collect())
    }

    pub fn node_install_dependencies(dir: &str) -> Result<String, String> {
        if cfg!(target_os = "windows") {
            return Err("Node dependency install is not supported on Windows hosts.".to_string());
        }
        let ok = NodeManager::new()
            .install_dependencies(dir)
            .map_err(|e| e.to_string())?;
        if ok {
            Ok("Node dependencies installed.".to_string())
        } else {
            Err("Node dependencies install failed.".to_string())
        }
    }

    pub fn node_start(req: &NodeAppRequest) -> Result<String, String> {
        if cfg!(target_os = "windows") {
            return Err("Node runtime start is not supported on Windows hosts.".to_string());
        }
        let ok = NodeManager::new()
            .start_app(&req.app_name, &req.start_script, &req.dir)
            .map_err(|e| e.to_string())?;
        if ok {
            let mut guard = runtime_state().lock().map_err(|e| e.to_string())?;
            guard.apps.insert(
                req.app_name.clone(),
                RuntimeAppInfo {
                    app_name: req.app_name.clone(),
                    runtime: "node".to_string(),
                    dir: req.dir.clone(),
                    status: "running".to_string(),
                },
            );
            Ok("Node app started.".to_string())
        } else {
            Err("Node app start failed.".to_string())
        }
    }

    pub fn node_stop(app_name: &str) -> Result<String, String> {
        if cfg!(target_os = "windows") {
            return Err("Node runtime stop is not supported on Windows hosts.".to_string());
        }
        let ok = NodeManager::new()
            .stop_app(app_name)
            .map_err(|e| e.to_string())?;
        if ok {
            let mut guard = runtime_state().lock().map_err(|e| e.to_string())?;
            if let Some(app) = guard.apps.get_mut(app_name) {
                app.status = "stopped".to_string();
            }
            Ok("Node app stopped.".to_string())
        } else {
            Err("Node app stop failed.".to_string())
        }
    }

    pub fn python_create_venv(dir: &str) -> Result<String, String> {
        if cfg!(target_os = "windows") {
            return Err("Python venv creation is not supported on Windows hosts.".to_string());
        }
        let ok = PythonManager::new()
            .create_venv(dir)
            .map_err(|e| e.to_string())?;
        if ok {
            Ok("Python venv created.".to_string())
        } else {
            Err("Python venv creation failed.".to_string())
        }
    }

    pub fn python_install_requirements(dir: &str) -> Result<String, String> {
        if cfg!(target_os = "windows") {
            return Err(
                "Python requirements install is not supported on Windows hosts.".to_string(),
            );
        }
        let ok = PythonManager::new()
            .install_requirements(dir)
            .map_err(|e| e.to_string())?;
        if ok {
            Ok("Python requirements installed.".to_string())
        } else {
            Err("Python requirements install failed.".to_string())
        }
    }

    pub fn python_start(req: &PythonAppRequest) -> Result<String, String> {
        if cfg!(target_os = "windows") {
            return Err("Python runtime start is not supported on Windows hosts.".to_string());
        }

        let ok = PythonManager::new()
            .start_gunicorn(&req.app_name, &req.wsgi_module, &req.dir, req.port)
            .map_err(|e| e.to_string())?;
        if ok {
            let mut guard = runtime_state().lock().map_err(|e| e.to_string())?;
            guard.apps.insert(
                req.app_name.clone(),
                RuntimeAppInfo {
                    app_name: req.app_name.clone(),
                    runtime: "python".to_string(),
                    dir: req.dir.clone(),
                    status: "running".to_string(),
                },
            );
            Ok("Python app started.".to_string())
        } else {
            Err("Python app start failed.".to_string())
        }
    }
}
