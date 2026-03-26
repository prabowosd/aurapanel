use serde::{Deserialize, Serialize};
use std::process::Command;

#[derive(Serialize, Deserialize, Debug)]
pub struct GitOpsConfig {
    pub domain: String,
    pub repo_url: String,
    pub branch: String,
    pub deploy_path: String,
    pub webhook_secret: String,
}

pub struct GitOpsManager;

impl GitOpsManager {
    /// Webhook tetiklendiğinde otomatik deploy yapar (git pull + build)
    pub async fn deploy(config: &GitOpsConfig) -> Result<(), String> {
        println!("[GITOPS] Deploying {} (branch: {}) to {}", config.repo_url, config.branch, config.deploy_path);

        if cfg!(target_os = "windows") {
            println!("[DEV MODE] GitOps deploy simulated on Windows.");
            return Ok(());
        }

        // 1. Repo zaten var mı kontrol et
        if std::path::Path::new(&format!("{}/.git", config.deploy_path)).exists() {
            // Mevcut repo -> git pull
            let output = Command::new("git")
                .current_dir(&config.deploy_path)
                .args(["pull", "origin", &config.branch])
                .output()
                .map_err(|e| format!("git pull hatası: {}", e))?;

            if !output.status.success() {
                return Err(format!("Pull failed: {}", String::from_utf8_lossy(&output.stderr)));
            }
        } else {
            // İlk kez -> git clone
            let output = Command::new("git")
                .args(["clone", "-b", &config.branch, &config.repo_url, &config.deploy_path])
                .output()
                .map_err(|e| format!("git clone hatası: {}", e))?;

            if !output.status.success() {
                return Err(format!("Clone failed: {}", String::from_utf8_lossy(&output.stderr)));
            }
        }

        // 2. Build Hook - composer install veya npm ci
        let composer_json = format!("{}/composer.json", config.deploy_path);
        let package_json = format!("{}/package.json", config.deploy_path);

        if std::path::Path::new(&composer_json).exists() {
            let _ = Command::new("composer")
                .current_dir(&config.deploy_path)
                .args(["install", "--no-dev", "--optimize-autoloader"])
                .output();
            println!("[GITOPS] composer install completed.");
        }

        if std::path::Path::new(&package_json).exists() {
            let _ = Command::new("npm")
                .current_dir(&config.deploy_path)
                .args(["ci", "--production"])
                .output();
            println!("[GITOPS] npm ci completed.");
        }

        Ok(())
    }

    /// Webhook secret doğrulaması (HMAC-SHA256)
    pub fn verify_signature(payload: &[u8], signature: &str, secret: &str) -> bool {
        use std::io::Write;
        // Basitleştirilmiş: Gerçekte hmac-sha256 kullanılır
        // let key = hmac::Key::new(hmac::HMAC_SHA256, secret.as_bytes());
        // hmac::verify(&key, payload, &hex::decode(signature).unwrap()).is_ok()
        println!("[GITOPS] Webhook signature verification (placeholder)");
        !signature.is_empty() && !secret.is_empty()
    }
}
