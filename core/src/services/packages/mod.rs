use serde::{Deserialize, Serialize};
use std::fs;
use std::path::{Path, PathBuf};

#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct HostingPackage {
    pub id: u64,
    pub name: String,
    pub plan_type: String,
    pub disk_gb: u32,
    pub bandwidth_gb: u32,
    pub domains: u32,
    pub databases: u32,
    pub emails: u32,
    pub cpu_limit: Option<u32>, // CPU percentage (e.g., 100 for 1 core)
    pub ram_mb: Option<u32>,    // RAM limit in MB
    pub io_limit: Option<u32>,  // I/O limit in MB/s
}

#[derive(Serialize, Deserialize, Debug)]
pub struct CreatePackageRequest {
    pub name: String,
    pub plan_type: String,
    pub disk_gb: u32,
    pub bandwidth_gb: u32,
    pub domains: u32,
    pub databases: u32,
    pub emails: u32,
    pub cpu_limit: Option<u32>,
    pub ram_mb: Option<u32>,
    pub io_limit: Option<u32>,
}

#[derive(Serialize, Deserialize, Debug)]
pub struct UpdatePackageRequest {
    pub id: u64,
    pub name: Option<String>,
    pub plan_type: Option<String>,
    pub disk_gb: Option<u32>,
    pub bandwidth_gb: Option<u32>,
    pub domains: Option<u32>,
    pub databases: Option<u32>,
    pub emails: Option<u32>,
    pub cpu_limit: Option<u32>,
    pub ram_mb: Option<u32>,
    pub io_limit: Option<u32>,
}

pub struct PackageManager;

impl PackageManager {
    pub fn list_packages() -> Result<Vec<HostingPackage>, String> {
        let path = packages_db_path();
        if !path.exists() {
            return Ok(Vec::new());
        }

        let json_str =
            fs::read_to_string(path).map_err(|e| format!("Paket listesi okunamadi: {}", e))?;
        serde_json::from_str(&json_str).map_err(|e| format!("Paket listesi parse edilemedi: {}", e))
    }

    pub fn create_package(req: &CreatePackageRequest) -> Result<String, String> {
        if req.name.trim().is_empty() {
            return Err("Paket adi zorunludur".to_string());
        }

        let plan_type = normalize_plan_type(&req.plan_type);
        let mut packages = Self::list_packages()?;
        if packages
            .iter()
            .any(|p| p.name.eq_ignore_ascii_case(req.name.trim()))
        {
            return Err(format!("Paket '{}' zaten mevcut.", req.name.trim()));
        }

        let new_id = packages.iter().map(|p| p.id).max().unwrap_or(0) + 1;
        packages.push(HostingPackage {
            id: new_id,
            name: req.name.trim().to_string(),
            plan_type,
            disk_gb: req.disk_gb,
            bandwidth_gb: req.bandwidth_gb,
            domains: req.domains,
            databases: req.databases,
            emails: req.emails,
            cpu_limit: req.cpu_limit,
            ram_mb: req.ram_mb,
            io_limit: req.io_limit,
        });

        save_packages(&packages)?;
        Ok(format!(
            "Paket '{}' basariyla olusturuldu.",
            req.name.trim()
        ))
    }

    pub fn delete_package(id: u64) -> Result<String, String> {
        let mut packages = Self::list_packages()?;
        let before = packages.len();
        packages.retain(|p| p.id != id);
        if packages.len() == before {
            return Err(format!("Paket #{} bulunamadi.", id));
        }

        save_packages(&packages)?;
        Ok(format!("Paket #{} basariyla silindi.", id))
    }

    pub fn update_package(req: &UpdatePackageRequest) -> Result<String, String> {
        let mut packages = Self::list_packages()?;

        // Validate name uniqueness before mutable borrow
        if let Some(name) = req.name.as_deref() {
            let name = name.trim();
            if name.is_empty() {
                return Err("Paket adi bos olamaz.".to_string());
            }
            if packages
                .iter()
                .any(|p| p.id != req.id && p.name.eq_ignore_ascii_case(name))
            {
                return Err(format!("Paket adi '{}' zaten kullaniliyor.", name));
            }
        }

        let pos = packages
            .iter()
            .position(|p| p.id == req.id)
            .ok_or_else(|| format!("Paket #{} bulunamadi.", req.id))?;

        let pkg = &mut packages[pos];
        if let Some(name) = req.name.as_deref() {
            pkg.name = name.trim().to_string();
        }
        if let Some(ref pt) = req.plan_type {
            pkg.plan_type = normalize_plan_type(pt);
        }
        if let Some(v) = req.disk_gb {
            pkg.disk_gb = v;
        }
        if let Some(v) = req.bandwidth_gb {
            pkg.bandwidth_gb = v;
        }
        if let Some(v) = req.domains {
            pkg.domains = v;
        }
        if let Some(v) = req.databases {
            pkg.databases = v;
        }
        if let Some(v) = req.emails {
            pkg.emails = v;
        }
        if let Some(v) = req.cpu_limit {
            pkg.cpu_limit = Some(v);
        }
        if let Some(v) = req.ram_mb {
            pkg.ram_mb = Some(v);
        }
        if let Some(v) = req.io_limit {
            pkg.io_limit = Some(v);
        }

        let updated_name = packages[pos].name.clone();
        save_packages(&packages)?;
        Ok(format!("Paket '{}' basariyla guncellendi.", updated_name))
    }

    pub fn get_package_by_name(name: &str) -> Result<Option<HostingPackage>, String> {
        let packages = Self::list_packages()?;
        Ok(packages
            .into_iter()
            .find(|p| p.name.eq_ignore_ascii_case(name)))
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

fn packages_db_path() -> PathBuf {
    state_root().join("packages.json")
}

fn save_packages(packages: &[HostingPackage]) -> Result<(), String> {
    let path = packages_db_path();
    if let Some(parent) = path.parent() {
        fs::create_dir_all(parent).map_err(|e| format!("Dizin olusturulamadi: {}", e))?;
    }

    let json = serde_json::to_string_pretty(packages).map_err(|e| format!("JSON hatasi: {}", e))?;
    fs::write(path, json).map_err(|e| format!("Dosya yazilamadi: {}", e))
}

fn normalize_plan_type(value: &str) -> String {
    let cleaned = value.trim().to_ascii_lowercase();
    match cleaned.as_str() {
        "reseller" => "reseller".to_string(),
        _ => "hosting".to_string(),
    }
}
