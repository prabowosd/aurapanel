use serde::{Deserialize, Serialize};
use std::process::Command;

#[derive(Serialize, Deserialize, Debug)]
pub struct SslConfig {
    pub domain: String,
    pub email: String,
    pub webroot: Option<String>,
}

pub struct SslManager;

impl SslManager {
    /// Let's Encrypt üzerinden otomatik SSL Sertifikası alır (ACME HTTP-01 challenge)
    pub async fn issue_certificate(config: &SslConfig) -> Result<(), String> {
        let webroot = config.webroot.as_deref()
            .unwrap_or("/usr/local/lsws/Example/html");

        println!("[ACME] Issuing SSL for {} via Let's Encrypt (email: {})", config.domain, config.email);

        if !std::path::Path::new("/usr/bin/certbot").exists() {
            println!("[DEV MODE] certbot not found. Simulating SSL issuance.");
            return Ok(());
        }

        let output = Command::new("certbot")
            .args([
                "certonly",
                "--webroot",
                "-w", webroot,
                "-d", &config.domain,
                "-d", &format!("www.{}", config.domain),
                "--email", &config.email,
                "--agree-tos",
                "--non-interactive",
            ])
            .output()
            .map_err(|e| format!("certbot çalıştırılamadı: {}", e))?;

        if !output.status.success() {
            let stderr = String::from_utf8_lossy(&output.stderr);
            return Err(format!("SSL alınamadı: {}", stderr));
        }

        // OLS'ye SSL yollarını bağla
        Self::bind_ssl_to_ols(&config.domain)?;

        Ok(())
    }

    /// Mevcut sertifikaları yeniler (cron tarafından çağrılabilir)
    pub fn renew_all() -> Result<(), String> {
        println!("[ACME] Running certbot renew for all domains...");
        if !std::path::Path::new("/usr/bin/certbot").exists() {
            println!("[DEV MODE] certbot not found. Skipping renewal.");
            return Ok(());
        }

        let _ = Command::new("certbot")
            .args(["renew", "--quiet"])
            .output()
            .map_err(|e| format!("Yenileme başarısız: {}", e))?;

        Ok(())
    }

    /// SSL sertifika yollarını OLS vhost config'ine yazar
    fn bind_ssl_to_ols(domain: &str) -> Result<(), String> {
        let ssl_block = format!(r#"
vhssl {{
  keyFile         /etc/letsencrypt/live/{domain}/privkey.pem
  certFile        /etc/letsencrypt/live/{domain}/fullchain.pem
  certChain       1
}}
"#, domain = domain);

        let vhconf_path = format!("/usr/local/lsws/conf/vhosts/{}/vhconf.conf", domain);

        if std::path::Path::new(&vhconf_path).exists() {
            // Dosyanın sonuna SSL bloğunu ekle
            let mut content = std::fs::read_to_string(&vhconf_path)
                .map_err(|e| format!("vhconf okunamadı: {}", e))?;
            content.push_str(&ssl_block);
            std::fs::write(&vhconf_path, content)
                .map_err(|e| format!("vhconf yazılamadı: {}", e))?;
        } else {
            println!("[DEV MODE] VHost config not found for {}. SSL binding simulated.", domain);
        }

        Ok(())
    }
}
