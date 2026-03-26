use serde::{Deserialize, Serialize};
use std::process::Command;

pub struct WebmailManager;

impl WebmailManager {
    /// Roundcube Webmail'i kurar ve yapılandırır
    pub fn install_roundcube(domain: &str, db_host: &str, db_name: &str, db_user: &str, db_pass: &str) -> Result<(), String> {
        println!("[WEBMAIL] Installing Roundcube for {}", domain);

        if cfg!(target_os = "windows") {
            println!("[DEV MODE] Roundcube installation simulated.");
            return Ok(());
        }

        // 1. Roundcube'u İndir (Composer veya tar.gz)
        let webmail_dir = format!("/usr/local/lsws/Example/html/webmail");
        let _ = Command::new("git")
            .args(["clone", "--depth", "1", "https://github.com/roundcube/roundcubemail.git", &webmail_dir])
            .output()
            .map_err(|e| format!("Roundcube indirilemedi: {}", e))?;

        // 2. config.inc.php oluştur
        let config_content = format!(r#"<?php
$config['db_dsnw'] = 'mysql://{}:{}@{}/{}';
$config['default_host'] = 'ssl://127.0.0.1';
$config['default_port'] = 993;
$config['smtp_server'] = 'tls://127.0.0.1';
$config['smtp_port'] = 587;
$config['product_name'] = 'AuraPanel Webmail';
$config['des_key'] = '{}';
$config['plugins'] = ['archive', 'zipdownload', 'markasjunk'];
?>"#, db_user, db_pass, db_host, db_name, "aurapanel_rc_secret_key_32ch");

        let config_path = format!("{}/config/config.inc.php", webmail_dir);
        std::fs::write(&config_path, config_content)
            .map_err(|e| format!("Roundcube config yazılamadı: {}", e))?;

        Ok(())
    }
}
