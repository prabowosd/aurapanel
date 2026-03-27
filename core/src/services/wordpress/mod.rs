
use serde::{de::DeserializeOwned, Deserialize, Serialize};
use std::fs;
use std::io::{Read, Write};
use std::path::{Path, PathBuf};
use std::process::{Command, Stdio};
use std::time::{SystemTime, UNIX_EPOCH};

use crate::services::db::{DbConfig, MariaDbManager, PostgresManager};
use crate::services::nitro::{NitroEngine, VHostConfig};
use crate::services::websites::WebsitesManager;

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct WordPressSiteSummary {
    pub domain: String,
    pub owner: String,
    pub docroot: String,
    pub php_version: String,
    pub wordpress_version: String,
    pub title: Option<String>,
    pub site_url: Option<String>,
    pub admin_email: Option<String>,
    pub db_name: Option<String>,
    pub db_user: Option<String>,
    pub db_host: Option<String>,
    pub db_engine: String,
    pub active_theme: Option<String>,
    pub total_plugins: u32,
    pub active_plugins: u32,
    pub status: String,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct WordPressPluginInfo {
    pub name: String,
    pub title: Option<String>,
    pub status: String,
    pub version: Option<String>,
    pub update: Option<String>,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct WordPressThemeInfo {
    pub name: String,
    pub title: Option<String>,
    pub status: String,
    pub version: Option<String>,
    pub update: Option<String>,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct WordPressBackupEntry {
    pub id: String,
    pub domain: String,
    pub backup_type: String,
    pub file_name: String,
    pub file_path: String,
    pub size_bytes: u64,
    pub created_at: u64,
    pub status: String,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct WordPressStagingEntry {
    pub id: String,
    pub source_domain: String,
    pub staging_domain: String,
    pub owner: String,
    pub created_at: u64,
    pub status: String,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct WordPressExtensionActionRequest {
    pub domain: String,
    #[serde(default)]
    pub names: Vec<String>,
    #[serde(default)]
    pub all: bool,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct WordPressBackupRequest {
    pub domain: String,
    pub backup_type: String,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct WordPressBackupRestoreRequest {
    pub id: String,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct WordPressStagingRequest {
    pub source_domain: String,
    pub staging_domain: String,
}

#[derive(Debug, Clone)]
struct WordPressConfig {
    db_name: String,
    db_user: String,
    db_pass: String,
    db_host: String,
}

#[derive(Debug, Clone)]
struct SiteCandidate {
    domain: String,
    owner: String,
    php_version: String,
    status: String,
    docroot: PathBuf,
}

#[derive(Debug, Serialize, Deserialize, Default)]
struct WordPressState {
    #[serde(default)]
    backups: Vec<WordPressBackupEntry>,
    #[serde(default)]
    staging: Vec<WordPressStagingEntry>,
}

#[derive(Debug, Deserialize)]
struct RawWpExtension {
    name: String,
    #[serde(default)]
    title: Option<String>,
    status: String,
    #[serde(default)]
    version: Option<String>,
    #[serde(default)]
    update: Option<String>,
}

pub struct WordPressManager;

impl WordPressManager {
    pub fn scan_sites() -> Result<Vec<WordPressSiteSummary>, String> {
        Self::list_sites()
    }

    pub fn list_sites() -> Result<Vec<WordPressSiteSummary>, String> {
        let mut items = Vec::new();

        for site in Self::site_candidates()? {
            if !Self::is_wordpress_docroot(&site.docroot) {
                continue;
            }

            let config = Self::parse_wp_config(&site.docroot).ok();
            let plugins = Self::list_plugins_for_path(&site.docroot).unwrap_or_default();
            let themes = Self::list_themes_for_path(&site.docroot).unwrap_or_default();
            let active_theme = themes
                .iter()
                .find(|theme| theme.status.eq_ignore_ascii_case("active"))
                .map(|theme| theme.name.clone());
            let active_plugins = plugins
                .iter()
                .filter(|plugin| plugin.status.eq_ignore_ascii_case("active"))
                .count() as u32;

            items.push(WordPressSiteSummary {
                domain: site.domain.clone(),
                owner: site.owner.clone(),
                docroot: site.docroot.display().to_string(),
                php_version: site.php_version.clone(),
                wordpress_version: Self::detect_wordpress_version(&site.docroot)
                    .unwrap_or_else(|| "Unknown".to_string()),
                title: Self::wp_option(&site.docroot, "blogname").ok(),
                site_url: Self::wp_option(&site.docroot, "siteurl").ok(),
                admin_email: Self::wp_option(&site.docroot, "admin_email")
                    .ok()
                    .or_else(|| config.as_ref().map(|_| format!("admin@{}", site.domain))),
                db_name: config.as_ref().map(|cfg| cfg.db_name.clone()),
                db_user: config.as_ref().map(|cfg| cfg.db_user.clone()),
                db_host: config.as_ref().map(|cfg| cfg.db_host.clone()),
                db_engine: Self::resolve_db_engine(&site.domain),
                active_theme,
                total_plugins: plugins.len() as u32,
                active_plugins,
                status: site.status.clone(),
            });
        }

        items.sort_by(|a, b| a.domain.cmp(&b.domain));
        Ok(items)
    }

    pub fn list_plugins(domain: &str) -> Result<Vec<WordPressPluginInfo>, String> {
        let site = Self::find_site(domain)?;
        Self::list_plugins_for_path(&site.docroot)
    }

    pub fn update_plugins(req: &WordPressExtensionActionRequest) -> Result<String, String> {
        let site = Self::find_site(&req.domain)?;
        let mut args = vec!["plugin", "update"];
        if req.all || req.names.is_empty() {
            args.push("--all");
        } else {
            for name in &req.names {
                args.push(name.as_str());
            }
        }
        Self::run_wp_capture(&site.docroot, &args)?;
        Ok("WordPress plugin guncellemesi tamamlandi.".to_string())
    }

    pub fn delete_plugins(req: &WordPressExtensionActionRequest) -> Result<String, String> {
        if req.names.is_empty() {
            return Err("Silinecek plugin secilmedi.".to_string());
        }
        let site = Self::find_site(&req.domain)?;
        let mut args = vec!["plugin", "delete"];
        for name in &req.names {
            args.push(name.as_str());
        }
        Self::run_wp_capture(&site.docroot, &args)?;
        Ok("Secilen pluginler silindi.".to_string())
    }

    pub fn list_themes(domain: &str) -> Result<Vec<WordPressThemeInfo>, String> {
        let site = Self::find_site(domain)?;
        Self::list_themes_for_path(&site.docroot)
    }

    pub fn update_themes(req: &WordPressExtensionActionRequest) -> Result<String, String> {
        let site = Self::find_site(&req.domain)?;
        let mut args = vec!["theme", "update"];
        if req.all || req.names.is_empty() {
            args.push("--all");
        } else {
            for name in &req.names {
                args.push(name.as_str());
            }
        }
        Self::run_wp_capture(&site.docroot, &args)?;
        Ok("WordPress theme guncellemesi tamamlandi.".to_string())
    }

    pub fn delete_themes(req: &WordPressExtensionActionRequest) -> Result<String, String> {
        if req.names.is_empty() {
            return Err("Silinecek theme secilmedi.".to_string());
        }
        let site = Self::find_site(&req.domain)?;
        let mut args = vec!["theme", "delete"];
        for name in &req.names {
            args.push(name.as_str());
        }
        Self::run_wp_capture(&site.docroot, &args)?;
        Ok("Secilen themeler silindi.".to_string())
    }
    pub fn list_backups(domain: Option<&str>) -> Result<Vec<WordPressBackupEntry>, String> {
        let mut items = Self::load_state()?.backups;
        if let Some(domain) = domain {
            let domain = Self::sanitize_domain(domain);
            items.retain(|item| item.domain == domain);
        }
        items.sort_by(|a, b| b.created_at.cmp(&a.created_at));
        Ok(items)
    }

    pub fn backup_file_path(id: &str) -> Result<PathBuf, String> {
        let state = Self::load_state()?;
        let entry = state
            .backups
            .iter()
            .find(|item| item.id == id)
            .ok_or_else(|| "WordPress backup bulunamadi.".to_string())?;
        let path = PathBuf::from(&entry.file_path);
        if !path.exists() {
            return Err("WordPress backup dosyasi bulunamadi.".to_string());
        }
        Ok(path)
    }

    pub fn create_backup(req: &WordPressBackupRequest) -> Result<WordPressBackupEntry, String> {
        if cfg!(target_os = "windows") {
            return Err("WordPress backup is not supported on Windows hosts.".to_string());
        }

        let site = Self::find_site(&req.domain)?;
        let config = Self::parse_wp_config(&site.docroot)?;
        let backup_type = match req.backup_type.trim().to_lowercase().as_str() {
            "full" | "files" | "database" => req.backup_type.trim().to_lowercase(),
            _ => return Err("Desteklenmeyen backup tipi. full/files/database kullanin.".to_string()),
        };

        let ts = Self::now_ts();
        let id = format!("wpbk_{}_{}", Self::slugify(&site.domain), ts);
        let domain_dir = Self::backups_root().join(Self::slugify(&site.domain));
        fs::create_dir_all(&domain_dir)
            .map_err(|e| format!("Backup dizini olusturulamadi: {}", e))?;

        let (file_name, file_path) = if backup_type == "database" {
            let dump_path = domain_dir.join(format!("{}_database.sql", id));
            Self::dump_database(&site.domain, &config, &dump_path)?;
            (dump_path.file_name().unwrap().to_string_lossy().to_string(), dump_path)
        } else if backup_type == "files" {
            let archive_path = domain_dir.join(format!("{}_files.tar.gz", id));
            Self::archive_directory_contents(&site.docroot, &archive_path)?;
            (archive_path.file_name().unwrap().to_string_lossy().to_string(), archive_path)
        } else {
            let bundle_root = Self::temp_root().join(&id);
            let site_bundle = bundle_root.join("site");
            fs::create_dir_all(&site_bundle)
                .map_err(|e| format!("Gecici backup dizini olusturulamadi: {}", e))?;
            Self::copy_dir_recursive(&site.docroot, &site_bundle)?;
            Self::dump_database(&site.domain, &config, &bundle_root.join("database.sql"))?;
            fs::write(
                bundle_root.join("manifest.json"),
                format!(
                    "{{\n  \"domain\": \"{}\",\n  \"created_at\": {}\n}}\n",
                    site.domain, ts
                ),
            )
            .map_err(|e| format!("Backup manifest yazilamadi: {}", e))?;
            let archive_path = domain_dir.join(format!("{}_full.tar.gz", id));
            Self::archive_directory_contents(&bundle_root, &archive_path)?;
            let _ = fs::remove_dir_all(&bundle_root);
            (archive_path.file_name().unwrap().to_string_lossy().to_string(), archive_path)
        };

        let metadata = fs::metadata(&file_path)
            .map_err(|e| format!("Backup dosyasi dogrulanamadi: {}", e))?;
        let entry = WordPressBackupEntry {
            id: id.clone(),
            domain: site.domain.clone(),
            backup_type,
            file_name,
            file_path: file_path.display().to_string(),
            size_bytes: metadata.len(),
            created_at: ts,
            status: "ready".to_string(),
        };

        let mut state = Self::load_state()?;
        state.backups.retain(|item| item.id != id);
        state.backups.push(entry.clone());
        Self::save_state(&state)?;
        Ok(entry)
    }

    pub fn restore_backup(req: &WordPressBackupRestoreRequest) -> Result<String, String> {
        if cfg!(target_os = "windows") {
            return Err("WordPress backup restore is not supported on Windows hosts.".to_string());
        }

        let state = Self::load_state()?;
        let entry = state
            .backups
            .iter()
            .find(|item| item.id == req.id)
            .cloned()
            .ok_or_else(|| "WordPress backup bulunamadi.".to_string())?;
        let site = Self::find_site(&entry.domain)?;
        let config = Self::parse_wp_config(&site.docroot)?;
        let backup_path = PathBuf::from(&entry.file_path);
        if !backup_path.exists() {
            return Err("Backup dosyasi bulunamadi.".to_string());
        }

        match entry.backup_type.as_str() {
            "database" => {
                Self::import_database(&entry.domain, &config, &backup_path)?;
            }
            "files" => {
                Self::clear_directory(&site.docroot)?;
                Self::extract_archive(&backup_path, &site.docroot)?;
                Self::flatten_single_child_directory(&site.docroot)?;
            }
            "full" => {
                let extract_root = Self::temp_root().join(format!("restore_{}", entry.id));
                fs::create_dir_all(&extract_root)
                    .map_err(|e| format!("Gecici restore dizini olusturulamadi: {}", e))?;
                Self::extract_archive(&backup_path, &extract_root)?;
                let bundle_root = if extract_root.join("site").exists() || extract_root.join("database.sql").exists() {
                    extract_root.clone()
                } else {
                    Self::single_child_directory(&extract_root).unwrap_or_else(|| extract_root.clone())
                };
                let restored_site = bundle_root.join("site");
                let restored_db = bundle_root.join("database.sql");
                if restored_site.exists() {
                    Self::clear_directory(&site.docroot)?;
                    Self::copy_dir_recursive(&restored_site, &site.docroot)?;
                }
                if restored_db.exists() {
                    Self::import_database(&entry.domain, &config, &restored_db)?;
                }
                let _ = fs::remove_dir_all(&extract_root);
            }
            other => return Err(format!("Restore desteklenmiyor: {}", other)),
        }

        Ok("WordPress backup geri yuklendi.".to_string())
    }

    pub fn list_staging(domain: Option<&str>) -> Result<Vec<WordPressStagingEntry>, String> {
        let mut items = Self::load_state()?.staging;
        if let Some(domain) = domain {
            let domain = Self::sanitize_domain(domain);
            items.retain(|item| item.source_domain == domain || item.staging_domain == domain);
        }
        items.sort_by(|a, b| b.created_at.cmp(&a.created_at));
        Ok(items)
    }

    pub fn create_staging(req: &WordPressStagingRequest) -> Result<WordPressStagingEntry, String> {
        if cfg!(target_os = "windows") {
            return Err("WordPress staging is not supported on Windows hosts.".to_string());
        }

        let source = Self::find_site(&req.source_domain)?;
        let source_config = Self::parse_wp_config(&source.docroot)?;
        let staging_domain = Self::sanitize_domain(&req.staging_domain);
        if staging_domain.is_empty() {
            return Err("Staging domain zorunludur.".to_string());
        }
        if staging_domain == source.domain {
            return Err("Staging domain kaynak domain ile ayni olamaz.".to_string());
        }
        if Self::site_candidates()?.iter().any(|site| site.domain == staging_domain) {
            return Err("Bu staging domain zaten mevcut.".to_string());
        }

        NitroEngine::create_vhost(&VHostConfig {
            domain: staging_domain.clone(),
            user: source.owner.clone(),
            php_version: source.php_version.clone(),
        })?;

        let target_docroot = Self::docroot_for(&source.owner, &staging_domain);
        Self::clear_directory(&target_docroot)?;
        Self::copy_dir_recursive(&source.docroot, &target_docroot)?;

        let engine = Self::resolve_db_engine(&source.domain);
        let db_slug = Self::slugify(&staging_domain);
        let db_name = Self::truncate(&format!("wp_{}_db", db_slug), 48);
        let db_user = Self::truncate(&format!("wp_{}_usr", db_slug), 32);
        let db_pass = Self::random_secret(20);
        let db_config = DbConfig {
            db_name: db_name.clone(),
            db_user: db_user.clone(),
            db_pass: db_pass.clone(),
            host: Some("localhost".to_string()),
            site_domain: Some(staging_domain.clone()),
            owner: Some(source.owner.clone()),
        };

        match engine.as_str() {
            "postgresql" => {
                PostgresManager::create_database(&db_config)?;
            }
            _ => {
                MariaDbManager::create_database(&db_config)?;
            }
        }

        let temp_dump = Self::temp_root().join(format!("{}_stage.sql", Self::slugify(&staging_domain)));
        Self::dump_database(&source.domain, &source_config, &temp_dump)?;
        let target_config = WordPressConfig {
            db_name: db_name.clone(),
            db_user: db_user.clone(),
            db_pass: db_pass.clone(),
            db_host: "localhost".to_string(),
        };
        Self::rewrite_wp_config(&target_docroot, &target_config)?;
        Self::import_database(&source.domain, &target_config, &temp_dump)?;
        let _ = fs::remove_file(&temp_dump);

        let source_url = Self::wp_option(&source.docroot, "siteurl")
            .unwrap_or_else(|_| format!("https://{}", source.domain));
        let target_url = format!("https://{}", staging_domain);
        let _ = Self::run_wp_capture(
            &target_docroot,
            &["search-replace", &source_url, &target_url, "--skip-columns=guid"],
        );
        let source_http = source_url.replace("https://", "http://");
        let target_http = target_url.replace("https://", "http://");
        if source_http != source_url {
            let _ = Self::run_wp_capture(
                &target_docroot,
                &["search-replace", &source_http, &target_http, "--skip-columns=guid"],
            );
        }

        let entry = WordPressStagingEntry {
            id: format!("wpstage_{}_{}", Self::slugify(&source.domain), Self::now_ts()),
            source_domain: source.domain,
            staging_domain,
            owner: source.owner,
            created_at: Self::now_ts(),
            status: "ready".to_string(),
        };

        let mut state = Self::load_state()?;
        state.staging.push(entry.clone());
        Self::save_state(&state)?;
        Ok(entry)
    }
    fn site_candidates() -> Result<Vec<SiteCandidate>, String> {
        let mut items = Vec::new();
        for value in NitroEngine::list_vhosts()? {
            let domain = value
                .get("domain")
                .and_then(|v| v.as_str())
                .unwrap_or_default()
                .trim()
                .to_lowercase();
            let owner = value
                .get("owner")
                .or_else(|| value.get("user"))
                .and_then(|v| v.as_str())
                .unwrap_or("aura")
                .trim()
                .to_string();
            if domain.is_empty() || owner.is_empty() {
                continue;
            }
            let php_version = value
                .get("php_version")
                .or_else(|| value.get("php"))
                .and_then(|v| v.as_str())
                .unwrap_or("8.3")
                .to_string();
            let status = value
                .get("status")
                .and_then(|v| v.as_str())
                .unwrap_or("active")
                .to_string();
            items.push(SiteCandidate {
                domain: domain.clone(),
                owner: owner.clone(),
                php_version,
                status,
                docroot: Self::docroot_for(&owner, &domain),
            });
        }
        Ok(items)
    }

    fn find_site(domain: &str) -> Result<SiteCandidate, String> {
        let normalized = Self::sanitize_domain(domain);
        Self::site_candidates()?
            .into_iter()
            .find(|site| site.domain == normalized && Self::is_wordpress_docroot(&site.docroot))
            .ok_or_else(|| format!("WordPress sitesi bulunamadi: {}", normalized))
    }

    fn list_plugins_for_path(docroot: &Path) -> Result<Vec<WordPressPluginInfo>, String> {
        let raw: Vec<RawWpExtension> = Self::run_wp_json(
            docroot,
            &["plugin", "list", "--format=json", "--fields=name,title,status,version,update"],
        )?;
        Ok(raw
            .into_iter()
            .map(|item| WordPressPluginInfo {
                name: item.name,
                title: item.title,
                status: item.status,
                version: item.version,
                update: item.update,
            })
            .collect())
    }

    fn list_themes_for_path(docroot: &Path) -> Result<Vec<WordPressThemeInfo>, String> {
        let raw: Vec<RawWpExtension> = Self::run_wp_json(
            docroot,
            &["theme", "list", "--format=json", "--fields=name,title,status,version,update"],
        )?;
        Ok(raw
            .into_iter()
            .map(|item| WordPressThemeInfo {
                name: item.name,
                title: item.title,
                status: item.status,
                version: item.version,
                update: item.update,
            })
            .collect())
    }

    fn wp_option(docroot: &Path, key: &str) -> Result<String, String> {
        Self::run_wp_text(docroot, &["option", "get", key])
    }

    fn detect_wordpress_version(docroot: &Path) -> Option<String> {
        if let Ok(version) = Self::run_wp_text(docroot, &["core", "version"]) {
            if !version.is_empty() {
                return Some(version);
            }
        }

        let version_file = docroot.join("wp-includes").join("version.php");
        let content = fs::read_to_string(version_file).ok()?;
        for line in content.lines() {
            if line.contains("$wp_version") {
                let value = line
                    .split('=')
                    .nth(1)
                    .unwrap_or_default()
                    .trim()
                    .trim_matches(';')
                    .trim()
                    .trim_matches('"')
                    .trim_matches('\'')
                    .to_string();
                if !value.is_empty() {
                    return Some(value);
                }
            }
        }
        None
    }

    fn is_wordpress_docroot(docroot: &Path) -> bool {
        docroot.join("wp-config.php").exists()
            || docroot.join("wp-includes").join("version.php").exists()
    }

    fn parse_wp_config(docroot: &Path) -> Result<WordPressConfig, String> {
        let content = fs::read_to_string(docroot.join("wp-config.php"))
            .map_err(|e| format!("wp-config okunamadi: {}", e))?;

        let db_name = Self::extract_wp_define(&content, "DB_NAME")
            .ok_or_else(|| "DB_NAME bulunamadi.".to_string())?;
        let db_user = Self::extract_wp_define(&content, "DB_USER")
            .ok_or_else(|| "DB_USER bulunamadi.".to_string())?;
        let db_pass = Self::extract_wp_define(&content, "DB_PASSWORD")
            .ok_or_else(|| "DB_PASSWORD bulunamadi.".to_string())?;
        let db_host = Self::extract_wp_define(&content, "DB_HOST")
            .unwrap_or_else(|| "localhost".to_string());

        Ok(WordPressConfig {
            db_name,
            db_user,
            db_pass,
            db_host,
        })
    }

    fn rewrite_wp_config(docroot: &Path, config: &WordPressConfig) -> Result<(), String> {
        let path = docroot.join("wp-config.php");
        let content = fs::read_to_string(&path)
            .map_err(|e| format!("wp-config okunamadi: {}", e))?;
        let content = Self::replace_wp_define(&content, "DB_NAME", &config.db_name);
        let content = Self::replace_wp_define(&content, "DB_USER", &config.db_user);
        let content = Self::replace_wp_define(&content, "DB_PASSWORD", &config.db_pass);
        let content = Self::replace_wp_define(&content, "DB_HOST", &config.db_host);
        fs::write(path, content).map_err(|e| format!("wp-config guncellenemedi: {}", e))
    }

    fn extract_wp_define(content: &str, key: &str) -> Option<String> {
        for line in content.lines() {
            let normalized = line.replace('"', "'");
            if !normalized.contains("define") || !normalized.contains(&format!("'{}'", key)) {
                continue;
            }
            let mut parts = normalized.split(',');
            let _ = parts.next()?;
            let value = parts
                .next()?
                .trim()
                .trim_end_matches(')')
                .trim_end_matches(';')
                .trim()
                .trim_matches('"')
                .trim_matches('\'')
                .to_string();
            if !value.is_empty() {
                return Some(value);
            }
        }
        None
    }

    fn replace_wp_define(content: &str, key: &str, value: &str) -> String {
        let mut out = Vec::new();
        let replacement = format!("define( '{}', '{}' );", key, value.replace('\'', "\\'"));
        for line in content.lines() {
            let normalized = line.replace('"', "'");
            if normalized.contains("define") && normalized.contains(&format!("'{}'", key)) {
                out.push(replacement.clone());
            } else {
                out.push(line.to_string());
            }
        }
        out.join("\n")
    }

    fn resolve_db_engine(domain: &str) -> String {
        WebsitesManager::list_db_links(Some(domain))
            .ok()
            .and_then(|links| links.into_iter().next())
            .map(|link| link.engine)
            .unwrap_or_else(|| "mariadb".to_string())
    }

    fn dump_database(domain: &str, config: &WordPressConfig, output: &Path) -> Result<(), String> {
        match Self::resolve_db_engine(domain).as_str() {
            "postgresql" => Self::dump_postgres(config, output),
            _ => Self::dump_mariadb(config, output),
        }
    }

    fn import_database(domain: &str, config: &WordPressConfig, dump_path: &Path) -> Result<(), String> {
        match Self::resolve_db_engine(domain).as_str() {
            "postgresql" => Self::import_postgres(config, dump_path),
            _ => Self::import_mariadb(config, dump_path),
        }
    }
    fn dump_mariadb(config: &WordPressConfig, output: &Path) -> Result<(), String> {
        let (host, _port) = Self::split_host_port(&config.db_host, "3306");
        let output_data = Command::new("mysqldump")
            .arg("-u")
            .arg(&config.db_user)
            .arg(format!("-p{}", config.db_pass))
            .arg("-h")
            .arg(host)
            .arg(&config.db_name)
            .output()
            .map_err(|e| format!("mysqldump basarisiz: {}", e))?;
        if !output_data.status.success() {
            return Err(String::from_utf8_lossy(&output_data.stderr).trim().to_string());
        }
        fs::write(output, output_data.stdout).map_err(|e| format!("MariaDB dump yazilamadi: {}", e))
    }

    fn import_mariadb(config: &WordPressConfig, dump_path: &Path) -> Result<(), String> {
        let sql = fs::read(dump_path).map_err(|e| format!("Dump dosyasi okunamadi: {}", e))?;
        let (host, _port) = Self::split_host_port(&config.db_host, "3306");
        let mut child = Command::new("mysql")
            .arg("-u")
            .arg(&config.db_user)
            .arg(format!("-p{}", config.db_pass))
            .arg("-h")
            .arg(host)
            .arg(&config.db_name)
            .stdin(Stdio::piped())
            .stdout(Stdio::null())
            .stderr(Stdio::piped())
            .spawn()
            .map_err(|e| format!("mysql import baslatilamadi: {}", e))?;
        child
            .stdin
            .as_mut()
            .ok_or_else(|| "mysql stdin kullanilamadi.".to_string())?
            .write_all(&sql)
            .map_err(|e| format!("MariaDB import yazilamadi: {}", e))?;
        let result = child.wait_with_output().map_err(|e| format!("mysql import tamamlama hatasi: {}", e))?;
        if !result.status.success() {
            return Err(String::from_utf8_lossy(&result.stderr).trim().to_string());
        }
        Ok(())
    }

    fn dump_postgres(config: &WordPressConfig, output: &Path) -> Result<(), String> {
        let (host, port) = Self::split_host_port(&config.db_host, "5432");
        let result = Command::new("pg_dump")
            .env("PGPASSWORD", &config.db_pass)
            .arg("-U")
            .arg(&config.db_user)
            .arg("-h")
            .arg(host)
            .arg("-p")
            .arg(port)
            .arg("-d")
            .arg(&config.db_name)
            .arg("-f")
            .arg(output)
            .output()
            .map_err(|e| format!("pg_dump basarisiz: {}", e))?;
        if !result.status.success() {
            return Err(String::from_utf8_lossy(&result.stderr).trim().to_string());
        }
        Ok(())
    }

    fn import_postgres(config: &WordPressConfig, dump_path: &Path) -> Result<(), String> {
        let (host, port) = Self::split_host_port(&config.db_host, "5432");
        let result = Command::new("psql")
            .env("PGPASSWORD", &config.db_pass)
            .arg("-U")
            .arg(&config.db_user)
            .arg("-h")
            .arg(host)
            .arg("-p")
            .arg(port)
            .arg("-d")
            .arg(&config.db_name)
            .arg("-f")
            .arg(dump_path)
            .output()
            .map_err(|e| format!("psql import basarisiz: {}", e))?;
        if !result.status.success() {
            return Err(String::from_utf8_lossy(&result.stderr).trim().to_string());
        }
        Ok(())
    }

    fn split_host_port(host: &str, default_port: &str) -> (String, String) {
        let trimmed = host.trim();
        if let Some((host, port)) = trimmed.rsplit_once(':') {
            if !host.is_empty() && port.chars().all(|ch| ch.is_ascii_digit()) {
                return (host.to_string(), port.to_string());
            }
        }
        (if trimmed.is_empty() { "localhost".to_string() } else { trimmed.to_string() }, default_port.to_string())
    }

    fn archive_directory(source: &Path, output: &Path) -> Result<(), String> {
        let parent = source.parent().ok_or_else(|| "Archive source parent bulunamadi.".to_string())?;
        let name = source.file_name().ok_or_else(|| "Archive source ismi bulunamadi.".to_string())?;
        let result = Command::new("tar")
            .arg("-czf")
            .arg(output)
            .arg("-C")
            .arg(parent)
            .arg(name)
            .output()
            .map_err(|e| format!("tar komutu calistirilamadi: {}", e))?;
        if !result.status.success() {
            return Err(String::from_utf8_lossy(&result.stderr).trim().to_string());
        }
        Ok(())
    }

    fn archive_directory_contents(source: &Path, output: &Path) -> Result<(), String> {
        let result = Command::new("tar")
            .arg("-czf")
            .arg(output)
            .arg("-C")
            .arg(source)
            .arg(".")
            .output()
            .map_err(|e| format!("tar komutu calistirilamadi: {}", e))?;
        if !result.status.success() {
            return Err(String::from_utf8_lossy(&result.stderr).trim().to_string());
        }
        Ok(())
    }

    fn extract_archive(archive: &Path, destination: &Path) -> Result<(), String> {
        fs::create_dir_all(destination).map_err(|e| format!("Extract dizini olusturulamadi: {}", e))?;
        let result = Command::new("tar")
            .arg("-xzf")
            .arg(archive)
            .arg("-C")
            .arg(destination)
            .output()
            .map_err(|e| format!("Archive extract basarisiz: {}", e))?;
        if !result.status.success() {
            return Err(String::from_utf8_lossy(&result.stderr).trim().to_string());
        }
        Ok(())
    }

    fn single_child_directory(path: &Path) -> Option<PathBuf> {
        let mut dirs = Vec::new();
        let mut files = 0usize;
        for entry in fs::read_dir(path).ok()? {
            let entry = entry.ok()?;
            let file_type = entry.file_type().ok()?;
            if file_type.is_dir() {
                dirs.push(entry.path());
            } else {
                files += 1;
            }
        }
        if files == 0 && dirs.len() == 1 {
            return dirs.into_iter().next();
        }
        None
    }

    fn flatten_single_child_directory(path: &Path) -> Result<(), String> {
        let Some(child) = Self::single_child_directory(path) else {
            return Ok(());
        };
        Self::copy_dir_recursive(&child, path)?;
        fs::remove_dir_all(&child).map_err(|e| format!("Gecici klasor kaldirilamadi: {}", e))?;
        Ok(())
    }

    fn run_wp_json<T: DeserializeOwned>(docroot: &Path, args: &[&str]) -> Result<T, String> {
        let raw = Self::run_wp_capture(docroot, args)?;
        serde_json::from_str(&raw).map_err(|e| format!("wp json parse hatasi: {}", e))
    }

    fn run_wp_text(docroot: &Path, args: &[&str]) -> Result<String, String> {
        Self::run_wp_capture(docroot, args).map(|text| text.trim().to_string())
    }

    fn run_wp_capture(docroot: &Path, args: &[&str]) -> Result<String, String> {
        let mut command = Command::new("wp");
        command.arg("--allow-root");
        command.arg(format!("--path={}", docroot.display()));
        for arg in args {
            command.arg(arg);
        }
        let output = command.output().map_err(|e| format!("wp-cli calistirilamadi: {}", e))?;
        if !output.status.success() {
            let stderr = String::from_utf8_lossy(&output.stderr).trim().to_string();
            if stderr.is_empty() {
                return Err(String::from_utf8_lossy(&output.stdout).trim().to_string());
            }
            return Err(stderr);
        }
        Ok(String::from_utf8_lossy(&output.stdout).trim().to_string())
    }

    fn copy_dir_recursive(source: &Path, destination: &Path) -> Result<(), String> {
        fs::create_dir_all(destination).map_err(|e| format!("Hedef dizin olusturulamadi: {}", e))?;
        for entry in fs::read_dir(source).map_err(|e| format!("Kaynak dizin okunamadi: {}", e))? {
            let entry = entry.map_err(|e| format!("Dizin girdisi okunamadi: {}", e))?;
            let src_path = entry.path();
            let dst_path = destination.join(entry.file_name());
            if entry.file_type().map_err(|e| format!("Dosya tipi okunamadi: {}", e))?.is_dir() {
                Self::copy_dir_recursive(&src_path, &dst_path)?;
            } else {
                if let Some(parent) = dst_path.parent() {
                    fs::create_dir_all(parent).map_err(|e| format!("Alt dizin olusturulamadi: {}", e))?;
                }
                fs::copy(&src_path, &dst_path).map_err(|e| format!("Dosya kopyalanamadi: {}", e))?;
            }
        }
        Ok(())
    }

    fn clear_directory(path: &Path) -> Result<(), String> {
        fs::create_dir_all(path).map_err(|e| format!("Dizin olusturulamadi: {}", e))?;
        for entry in fs::read_dir(path).map_err(|e| format!("Dizin okunamadi: {}", e))? {
            let entry = entry.map_err(|e| format!("Dizin girdisi okunamadi: {}", e))?;
            let item_path = entry.path();
            if entry.file_type().map_err(|e| format!("Dosya tipi okunamadi: {}", e))?.is_dir() {
                fs::remove_dir_all(&item_path).map_err(|e| format!("Klasor silinemedi: {}", e))?;
            } else {
                fs::remove_file(&item_path).map_err(|e| format!("Dosya silinemedi: {}", e))?;
            }
        }
        Ok(())
    }

    fn storage_root() -> PathBuf {
        if let Ok(path) = std::env::var("AURAPANEL_STATE_DIR") {
            let trimmed = path.trim();
            if !trimmed.is_empty() {
                return PathBuf::from(trimmed);
            }
        }
        if Path::new("/var/lib/aurapanel").exists() {
            return PathBuf::from("/var/lib/aurapanel");
        }
        std::env::temp_dir().join("aurapanel")
    }

    fn temp_root() -> PathBuf {
        let path = Self::storage_root().join("wordpress_tmp");
        let _ = fs::create_dir_all(&path);
        path
    }

    fn backups_root() -> PathBuf {
        let path = Self::storage_root().join("wordpress_backups");
        let _ = fs::create_dir_all(&path);
        path
    }

    fn state_path() -> PathBuf {
        Self::storage_root().join("wordpress_manager.json")
    }

    fn load_state() -> Result<WordPressState, String> {
        let path = Self::state_path();
        if !path.exists() {
            return Ok(WordPressState::default());
        }
        let raw = fs::read_to_string(&path).map_err(|e| format!("WordPress state okunamadi: {}", e))?;
        serde_json::from_str(&raw).map_err(|e| format!("WordPress state parse edilemedi: {}", e))
    }

    fn save_state(state: &WordPressState) -> Result<(), String> {
        let path = Self::state_path();
        if let Some(parent) = path.parent() {
            fs::create_dir_all(parent).map_err(|e| format!("WordPress state dizini olusturulamadi: {}", e))?;
        }
        let json = serde_json::to_string_pretty(state).map_err(|e| format!("WordPress state json olusturulamadi: {}", e))?;
        fs::write(path, json).map_err(|e| format!("WordPress state yazilamadi: {}", e))
    }

    fn docroot_for(owner: &str, domain: &str) -> PathBuf {
        PathBuf::from(format!("/home/{}/public_html/{}", owner, domain))
    }

    fn sanitize_domain(domain: &str) -> String {
        domain.trim().to_lowercase().trim_matches('.').to_string()
    }

    fn slugify(value: &str) -> String {
        let mut out = String::new();
        for ch in value.chars() {
            if ch.is_ascii_alphanumeric() {
                out.push(ch.to_ascii_lowercase());
            } else {
                out.push('_');
            }
        }
        while out.contains("__") {
            out = out.replace("__", "_");
        }
        out.trim_matches('_').to_string()
    }

    fn truncate(value: &str, max_len: usize) -> String {
        value.chars().take(max_len).collect()
    }

    fn now_ts() -> u64 {
        SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .map(|d| d.as_secs())
            .unwrap_or(0)
    }

    fn random_secret(len: usize) -> String {
        let mut bytes = vec![0u8; len.max(8)];
        if let Ok(mut file) = fs::File::open("/dev/urandom") {
            if file.read_exact(&mut bytes).is_ok() {
                return bytes
                    .into_iter()
                    .map(|b| match b % 62 {
                        n @ 0..=9 => (b'0' + n) as char,
                        n @ 10..=35 => (b'a' + (n - 10)) as char,
                        n => (b'A' + (n - 36)) as char,
                    })
                    .take(len)
                    .collect();
            }
        }
        let fallback = Self::now_ts().to_string();
        fallback.repeat((len / fallback.len()) + 1).chars().take(len).collect()
    }
}
