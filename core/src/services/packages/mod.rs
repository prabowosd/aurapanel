use serde::{Deserialize, Serialize};
use std::fs;
use std::path::Path;

/// Hosting/Reseller paketi
#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct HostingPackage {
    pub id: u64,
    pub name: String,
    pub plan_type: String,   // "hosting", "reseller"
    pub disk_gb: u32,
    pub bandwidth_gb: u32,   // 0 = unlimited
    pub domains: u32,        // 0 = unlimited
    pub databases: u32,      // 0 = unlimited
    pub emails: u32,         // 0 = unlimited
}

/// Paket oluşturma isteği
#[derive(Serialize, Deserialize, Debug)]
pub struct CreatePackageRequest {
    pub name: String,
    pub plan_type: String,
    pub disk_gb: u32,
    pub bandwidth_gb: u32,
    pub domains: u32,
    pub databases: u32,
    pub emails: u32,
}

const PACKAGES_DB_PATH: &str = "/var/lib/aurapanel/packages.json";

pub struct PackageManager;

impl PackageManager {
    fn dev_mode() -> bool {
        !Path::new("/var/lib/aurapanel").exists()
    }

    /// Paketleri listele
    pub fn list_packages() -> Result<Vec<HostingPackage>, String> {
        if Self::dev_mode() {
            return Ok(vec![
                HostingPackage { id: 1, name: "Başlangıç Hosting".into(), plan_type: "hosting".into(), disk_gb: 10, bandwidth_gb: 0, domains: 1, databases: 3, emails: 10 },
                HostingPackage { id: 2, name: "Kurumsal Hosting".into(), plan_type: "hosting".into(), disk_gb: 50, bandwidth_gb: 0, domains: 0, databases: 0, emails: 0 },
                HostingPackage { id: 3, name: "Bayi V1".into(), plan_type: "reseller".into(), disk_gb: 100, bandwidth_gb: 0, domains: 0, databases: 0, emails: 0 },
            ]);
        }

        let json_str = fs::read_to_string(PACKAGES_DB_PATH)
            .unwrap_or_else(|_| "[]".to_string());
        serde_json::from_str(&json_str)
            .map_err(|e| format!("Paket listesi okunamadı: {}", e))
    }

    /// Paket oluştur  
    pub fn create_package(req: &CreatePackageRequest) -> Result<String, String> {
        if Self::dev_mode() {
            return Ok(format!("[Dev Mode] Paket '{}' oluşturuldu.", req.name));
        }

        let mut packages = Self::list_packages()?;
        let new_id = packages.iter().map(|p| p.id).max().unwrap_or(0) + 1;
        packages.push(HostingPackage {
            id: new_id,
            name: req.name.clone(),
            plan_type: req.plan_type.clone(),
            disk_gb: req.disk_gb,
            bandwidth_gb: req.bandwidth_gb,
            domains: req.domains,
            databases: req.databases,
            emails: req.emails,
        });
        Self::save_packages(&packages)?;
        Ok(format!("Paket '{}' başarıyla oluşturuldu.", req.name))
    }

    /// Paket sil
    pub fn delete_package(id: u64) -> Result<String, String> {
        if Self::dev_mode() {
            return Ok(format!("[Dev Mode] Paket #{} silindi.", id));
        }

        let mut packages = Self::list_packages()?;
        let before = packages.len();
        packages.retain(|p| p.id != id);
        if packages.len() == before {
            return Err(format!("Paket #{} bulunamadı.", id));
        }
        Self::save_packages(&packages)?;
        Ok(format!("Paket #{} başarıyla silindi.", id))
    }

    fn save_packages(packages: &[HostingPackage]) -> Result<(), String> {
        fs::create_dir_all("/var/lib/aurapanel")
            .map_err(|e| format!("Dizin oluşturulamadı: {}", e))?;
        let json = serde_json::to_string_pretty(packages)
            .map_err(|e| format!("JSON hatası: {}", e))?;
        fs::write(PACKAGES_DB_PATH, json)
            .map_err(|e| format!("Dosya yazılamadı: {}", e))
    }
}
