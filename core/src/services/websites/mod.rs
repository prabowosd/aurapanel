use serde::{Deserialize, Serialize};
use std::fs;
use std::path::{Path, PathBuf};
use std::time::{SystemTime, UNIX_EPOCH};

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct CreateSubdomainRequest {
    pub parent_domain: String,
    pub subdomain: String,
    pub php_version: String,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct SubdomainEntry {
    pub fqdn: String,
    pub parent_domain: String,
    pub subdomain: String,
    pub php_version: String,
    pub ssl_enabled: bool,
    pub created_at: u64,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct ConvertSubdomainRequest {
    pub fqdn: String,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct SubdomainPhpUpdateRequest {
    pub fqdn: String,
    pub php_version: String,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct WebsiteDbLinkRequest {
    pub domain: String,
    pub engine: String,
    pub db_name: String,
    pub db_user: String,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct WebsiteDbLink {
    pub domain: String,
    pub engine: String,
    pub db_name: String,
    pub db_user: String,
    pub linked_at: u64,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct WebsiteAliasRequest {
    pub domain: String,
    pub alias: String,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct WebsiteAliasEntry {
    pub domain: String,
    pub alias: String,
    pub created_at: u64,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct WebsiteAdvancedConfig {
    pub domain: String,
    pub open_basedir: bool,
    pub rewrite_rules: String,
    pub vhost_config: String,
    pub updated_at: u64,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct WebsiteOpenBasedirRequest {
    pub domain: String,
    pub enabled: bool,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct WebsiteRewriteRequest {
    pub domain: String,
    pub rules: String,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct WebsiteVhostConfigRequest {
    pub domain: String,
    pub content: String,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct WebsiteCustomSslRequest {
    pub domain: String,
    pub cert_pem: String,
    pub key_pem: String,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct WebsiteCustomSslEntry {
    pub domain: String,
    pub cert_pem: String,
    pub key_pem: String,
    pub updated_at: u64,
}

#[derive(Debug, Serialize, Deserialize, Default)]
struct WebsiteWorkflowStore {
    #[serde(default)]
    subdomains: Vec<SubdomainEntry>,
    #[serde(default)]
    db_links: Vec<WebsiteDbLink>,
    #[serde(default)]
    aliases: Vec<WebsiteAliasEntry>,
    #[serde(default)]
    advanced_configs: Vec<WebsiteAdvancedConfig>,
    #[serde(default)]
    custom_ssl: Vec<WebsiteCustomSslEntry>,
}

pub struct WebsitesManager;

impl WebsitesManager {
    fn storage_path() -> PathBuf {
        let prod_dir = Path::new("/var/lib/aurapanel");
        if prod_dir.exists() {
            return prod_dir.join("websites_workflow.json");
        }

        let fallback_dir = std::env::temp_dir().join("aurapanel");
        fallback_dir.join("websites_workflow.json")
    }

    fn ensure_parent_dir(path: &Path) -> Result<(), String> {
        match path.parent() {
            Some(parent) => fs::create_dir_all(parent)
                .map_err(|e| format!("Workflow dizini olusturulamadi: {}", e)),
            None => Ok(()),
        }
    }

    fn load_store() -> Result<WebsiteWorkflowStore, String> {
        let path = Self::storage_path();
        if !path.exists() {
            return Ok(WebsiteWorkflowStore::default());
        }

        let raw = fs::read_to_string(&path)
            .map_err(|e| format!("Workflow kaydi okunamadi: {}", e))?;
        serde_json::from_str(&raw)
            .map_err(|e| format!("Workflow kaydi parse edilemedi: {}", e))
    }

    fn save_store(store: &WebsiteWorkflowStore) -> Result<(), String> {
        let path = Self::storage_path();
        Self::ensure_parent_dir(&path)?;
        let json = serde_json::to_string_pretty(store)
            .map_err(|e| format!("Workflow JSON olusturulamadi: {}", e))?;
        fs::write(path, json).map_err(|e| format!("Workflow kaydi yazilamadi: {}", e))
    }

    fn now_ts() -> u64 {
        SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .map(|d| d.as_secs())
            .unwrap_or(0)
    }

    fn normalize_engine(engine: &str) -> Option<&'static str> {
        let normalized = engine.trim().to_lowercase();
        match normalized.as_str() {
            "mariadb" | "mysql" => Some("mariadb"),
            "postgres" | "postgresql" => Some("postgresql"),
            _ => None,
        }
    }

    fn sanitize_domain(domain: &str) -> String {
        domain.trim().to_lowercase().trim_matches('.').to_string()
    }

    fn sanitize_sub_label(label: &str) -> String {
        label.trim().to_lowercase().trim_matches('.').to_string()
    }

    fn sanitize_text_blob(value: &str) -> String {
        value.replace('\r', "")
    }

    fn looks_like_pem(value: &str) -> bool {
        let normalized = value.trim();
        normalized.starts_with("-----BEGIN ") && normalized.contains("-----END ")
    }

    fn home_public_html_roots() -> Vec<PathBuf> {
        let mut roots = Vec::new();
        let home = Path::new("/home");
        if let Ok(users) = fs::read_dir(home) {
            for user in users.flatten() {
                let root = user.path().join("public_html");
                if root.exists() {
                    roots.push(root);
                }
            }
        }
        roots
    }

    fn find_docroot(domain: &str) -> Option<PathBuf> {
        for root in Self::home_public_html_roots() {
            let candidate = root.join(domain);
            if candidate.exists() {
                return Some(candidate);
            }
        }
        None
    }

    fn remove_docroots_for_domain(domain: &str) -> Vec<String> {
        let mut removed = Vec::new();
        for root in Self::home_public_html_roots() {
            let candidate = root.join(domain);
            if candidate.exists() {
                if fs::remove_dir_all(&candidate).is_ok() {
                    removed.push(candidate.display().to_string());
                }
            }
        }
        removed
    }

    fn vhost_conf_path(domain: &str) -> PathBuf {
        PathBuf::from("/usr/local/lsws/conf/vhosts")
            .join(domain)
            .join("vhconf.conf")
    }

    fn find_or_default_advanced_config<'a>(store: &'a mut WebsiteWorkflowStore, domain: &str) -> &'a mut WebsiteAdvancedConfig {
        if let Some(idx) = store.advanced_configs.iter().position(|x| x.domain == domain) {
            return &mut store.advanced_configs[idx];
        }

        store.advanced_configs.push(WebsiteAdvancedConfig {
            domain: domain.to_string(),
            open_basedir: false,
            rewrite_rules: String::new(),
            vhost_config: String::new(),
            updated_at: Self::now_ts(),
        });

        let idx = store.advanced_configs.len() - 1;
        &mut store.advanced_configs[idx]
    }

    pub fn list_subdomains(domain: Option<&str>) -> Result<Vec<SubdomainEntry>, String> {
        let mut items = Self::load_store()?.subdomains;
        if let Some(d) = domain {
            let d = Self::sanitize_domain(d);
            items.retain(|x| x.parent_domain == d);
        }
        items.sort_by(|a, b| a.fqdn.cmp(&b.fqdn));
        Ok(items)
    }

    pub fn create_subdomain(req: &CreateSubdomainRequest) -> Result<SubdomainEntry, String> {
        let parent = Self::sanitize_domain(&req.parent_domain);
        let label = Self::sanitize_sub_label(&req.subdomain);
        let php = req.php_version.trim();

        if parent.is_empty() {
            return Err("Parent domain bos olamaz".to_string());
        }
        if label.is_empty() || label.contains(' ') || label.contains('/') || label.contains('\\') {
            return Err("Subdomain gecersiz".to_string());
        }
        if php.is_empty() {
            return Err("PHP versiyonu bos olamaz".to_string());
        }

        let fqdn = format!("{}.{}", label, parent);
        let mut store = Self::load_store()?;

        if store.subdomains.iter().any(|s| s.fqdn == fqdn) {
            return Err(format!("{} zaten mevcut", fqdn));
        }

        let entry = SubdomainEntry {
            fqdn,
            parent_domain: parent,
            subdomain: label,
            php_version: php.to_string(),
            ssl_enabled: false,
            created_at: Self::now_ts(),
        };

        store.subdomains.push(entry.clone());
        Self::save_store(&store)?;
        Ok(entry)
    }

    pub fn delete_subdomain(fqdn: &str) -> Result<(), String> {
        Self::delete_subdomain_with_options(fqdn, false).map(|_| ())
    }

    pub fn delete_subdomain_with_options(fqdn: &str, delete_docroot: bool) -> Result<Vec<String>, String> {
        let fqdn = Self::sanitize_domain(fqdn);
        if fqdn.is_empty() {
            return Err("Subdomain FQDN bos olamaz".to_string());
        }

        let mut store = Self::load_store()?;
        let before = store.subdomains.len();
        store.subdomains.retain(|s| s.fqdn != fqdn);

        if before == store.subdomains.len() {
            return Err("Subdomain bulunamadi".to_string());
        }

        Self::save_store(&store)?;

        let removed = if delete_docroot {
            Self::remove_docroots_for_domain(&fqdn)
        } else {
            Vec::new()
        };

        Ok(removed)
    }

    pub fn update_subdomain_php(req: &SubdomainPhpUpdateRequest) -> Result<SubdomainEntry, String> {
        let fqdn = Self::sanitize_domain(&req.fqdn);
        let php_version = req.php_version.trim().to_string();

        if fqdn.is_empty() || php_version.is_empty() {
            return Err("FQDN ve PHP versiyonu zorunludur".to_string());
        }

        let mut store = Self::load_store()?;
        let subdomain = store
            .subdomains
            .iter_mut()
            .find(|s| s.fqdn == fqdn)
            .ok_or_else(|| "Subdomain bulunamadi".to_string())?;

        subdomain.php_version = php_version;
        let updated = subdomain.clone();
        Self::save_store(&store)?;
        Ok(updated)
    }

    pub fn consume_subdomain_for_conversion(req: &ConvertSubdomainRequest) -> Result<SubdomainEntry, String> {
        let fqdn = Self::sanitize_domain(&req.fqdn);
        if fqdn.is_empty() {
            return Err("Subdomain FQDN bos olamaz".to_string());
        }

        let mut store = Self::load_store()?;
        let index = store
            .subdomains
            .iter()
            .position(|s| s.fqdn == fqdn)
            .ok_or_else(|| "Subdomain bulunamadi".to_string())?;

        let removed = store.subdomains.remove(index);
        Self::save_store(&store)?;
        Ok(removed)
    }

    pub fn list_db_links(domain: Option<&str>) -> Result<Vec<WebsiteDbLink>, String> {
        let mut links = Self::load_store()?.db_links;
        if let Some(d) = domain {
            let d = Self::sanitize_domain(d);
            links.retain(|x| x.domain == d);
        }
        links.sort_by(|a, b| a.domain.cmp(&b.domain).then(a.db_name.cmp(&b.db_name)));
        Ok(links)
    }

    pub fn attach_db(req: &WebsiteDbLinkRequest) -> Result<WebsiteDbLink, String> {
        let domain = Self::sanitize_domain(&req.domain);
        let db_name = req.db_name.trim().to_string();
        let db_user = req.db_user.trim().to_string();
        let engine = Self::normalize_engine(&req.engine)
            .ok_or_else(|| "Desteklenmeyen database engine".to_string())?
            .to_string();

        if domain.is_empty() {
            return Err("Domain bos olamaz".to_string());
        }
        if db_name.is_empty() || db_user.is_empty() {
            return Err("DB adi ve DB kullanicisi zorunludur".to_string());
        }

        let mut store = Self::load_store()?;
        let linked_at = Self::now_ts();

        if let Some(existing) = store
            .db_links
            .iter_mut()
            .find(|x| x.domain == domain && x.engine == engine && x.db_name == db_name)
        {
            existing.db_user = db_user;
            existing.linked_at = linked_at;
            let updated = existing.clone();
            Self::save_store(&store)?;
            return Ok(updated);
        }

        let link = WebsiteDbLink {
            domain,
            engine,
            db_name,
            db_user,
            linked_at,
        };

        store.db_links.push(link.clone());
        Self::save_store(&store)?;
        Ok(link)
    }

    pub fn detach_db(domain: &str, engine: &str, db_name: &str) -> Result<(), String> {
        let domain = Self::sanitize_domain(domain);
        let db_name = db_name.trim().to_string();
        let engine = Self::normalize_engine(engine)
            .ok_or_else(|| "Desteklenmeyen database engine".to_string())?
            .to_string();

        if domain.is_empty() || db_name.is_empty() {
            return Err("Domain ve DB adi zorunludur".to_string());
        }

        let mut store = Self::load_store()?;
        let before = store.db_links.len();
        store
            .db_links
            .retain(|x| !(x.domain == domain && x.engine == engine && x.db_name == db_name));

        if before == store.db_links.len() {
            return Err("DB baglantisi bulunamadi".to_string());
        }

        Self::save_store(&store)
    }

    pub fn list_aliases(domain: Option<&str>) -> Result<Vec<WebsiteAliasEntry>, String> {
        let mut items = Self::load_store()?.aliases;
        if let Some(d) = domain {
            let d = Self::sanitize_domain(d);
            items.retain(|x| x.domain == d);
        }
        items.sort_by(|a, b| a.domain.cmp(&b.domain).then(a.alias.cmp(&b.alias)));
        Ok(items)
    }

    pub fn add_alias(req: &WebsiteAliasRequest) -> Result<WebsiteAliasEntry, String> {
        let domain = Self::sanitize_domain(&req.domain);
        let alias = Self::sanitize_domain(&req.alias);

        if domain.is_empty() || alias.is_empty() {
            return Err("Domain ve alias zorunludur".to_string());
        }
        if alias == domain {
            return Err("Alias ana domain ile ayni olamaz".to_string());
        }

        let mut store = Self::load_store()?;

        if store.aliases.iter().any(|x| x.alias == alias && x.domain != domain) {
            return Err("Alias baska bir domaine atanmis".to_string());
        }

        if let Some(existing) = store
            .aliases
            .iter()
            .find(|x| x.domain == domain && x.alias == alias)
            .cloned()
        {
            return Ok(existing);
        }

        let entry = WebsiteAliasEntry {
            domain,
            alias,
            created_at: Self::now_ts(),
        };

        store.aliases.push(entry.clone());
        Self::save_store(&store)?;
        Ok(entry)
    }

    pub fn delete_alias(domain: &str, alias: &str) -> Result<(), String> {
        let domain = Self::sanitize_domain(domain);
        let alias = Self::sanitize_domain(alias);
        if domain.is_empty() || alias.is_empty() {
            return Err("Domain ve alias zorunludur".to_string());
        }

        let mut store = Self::load_store()?;
        let before = store.aliases.len();
        store.aliases.retain(|x| !(x.domain == domain && x.alias == alias));

        if before == store.aliases.len() {
            return Err("Alias kaydi bulunamadi".to_string());
        }

        Self::save_store(&store)
    }

    pub fn get_advanced_config(domain: &str) -> Result<WebsiteAdvancedConfig, String> {
        let domain = Self::sanitize_domain(domain);
        if domain.is_empty() {
            return Err("Domain bos olamaz".to_string());
        }

        let mut store = Self::load_store()?;
        let mut entry = Self::find_or_default_advanced_config(&mut store, &domain).clone();

        if let Some(docroot) = Self::find_docroot(&domain) {
            let rewrite_file = docroot.join(".htaccess");
            if rewrite_file.exists() {
                if let Ok(raw) = fs::read_to_string(&rewrite_file) {
                    entry.rewrite_rules = raw;
                }
            }
        }

        let vhost_file = Self::vhost_conf_path(&domain);
        if vhost_file.exists() {
            if let Ok(raw) = fs::read_to_string(&vhost_file) {
                entry.vhost_config = raw.clone();
                if raw.to_lowercase().contains("open_basedir") {
                    entry.open_basedir = true;
                }
            }
        }

        if let Some(target) = store.advanced_configs.iter_mut().find(|x| x.domain == domain) {
            *target = entry.clone();
        }
        Self::save_store(&store)?;
        Ok(entry)
    }

    pub fn set_open_basedir(req: &WebsiteOpenBasedirRequest) -> Result<WebsiteAdvancedConfig, String> {
        let domain = Self::sanitize_domain(&req.domain);
        if domain.is_empty() {
            return Err("Domain bos olamaz".to_string());
        }

        let mut store = Self::load_store()?;
        let entry = Self::find_or_default_advanced_config(&mut store, &domain);
        entry.open_basedir = req.enabled;
        entry.updated_at = Self::now_ts();

        let vhost_file = Self::vhost_conf_path(&domain);
        if vhost_file.exists() {
            if let Ok(raw) = fs::read_to_string(&vhost_file) {
                let marker_prefix = "# AURAPANEL_OPEN_BASEDIR=";
                let mut lines: Vec<String> = raw.lines().map(|x| x.to_string()).collect();
                let mut marker_idx = None;
                for (i, line) in lines.iter().enumerate() {
                    if line.trim_start().starts_with(marker_prefix) {
                        marker_idx = Some(i);
                        break;
                    }
                }
                let new_marker = format!("{}{}", marker_prefix, if req.enabled { "1" } else { "0" });
                if let Some(i) = marker_idx {
                    lines[i] = new_marker;
                } else {
                    lines.insert(0, new_marker);
                }
                let mut updated_raw = lines.join("\n");
                if raw.ends_with('\n') {
                    updated_raw.push('\n');
                }
                let _ = fs::write(&vhost_file, updated_raw);
            }
        }

        let updated = entry.clone();
        Self::save_store(&store)?;
        Ok(updated)
    }

    pub fn save_rewrite_rules(req: &WebsiteRewriteRequest) -> Result<WebsiteAdvancedConfig, String> {
        let domain = Self::sanitize_domain(&req.domain);
        if domain.is_empty() {
            return Err("Domain bos olamaz".to_string());
        }

        let mut store = Self::load_store()?;
        let entry = Self::find_or_default_advanced_config(&mut store, &domain);
        entry.rewrite_rules = Self::sanitize_text_blob(&req.rules);
        entry.updated_at = Self::now_ts();

        if let Some(docroot) = Self::find_docroot(&domain) {
            let rewrite_file = docroot.join(".htaccess");
            let _ = fs::write(rewrite_file, &entry.rewrite_rules);
        }

        let updated = entry.clone();
        Self::save_store(&store)?;
        Ok(updated)
    }

    pub fn save_vhost_config(req: &WebsiteVhostConfigRequest) -> Result<WebsiteAdvancedConfig, String> {
        let domain = Self::sanitize_domain(&req.domain);
        if domain.is_empty() {
            return Err("Domain bos olamaz".to_string());
        }

        let mut store = Self::load_store()?;
        let entry = Self::find_or_default_advanced_config(&mut store, &domain);
        entry.vhost_config = Self::sanitize_text_blob(&req.content);
        entry.updated_at = Self::now_ts();

        let vhost_file = Self::vhost_conf_path(&domain);
        if vhost_file.exists() {
            let _ = fs::write(&vhost_file, &entry.vhost_config);
        }

        let updated = entry.clone();
        Self::save_store(&store)?;
        Ok(updated)
    }

    pub fn get_custom_ssl(domain: &str) -> Result<Option<WebsiteCustomSslEntry>, String> {
        let domain = Self::sanitize_domain(domain);
        if domain.is_empty() {
            return Err("Domain bos olamaz".to_string());
        }

        let store = Self::load_store()?;
        Ok(store.custom_ssl.into_iter().find(|x| x.domain == domain))
    }

    pub fn save_custom_ssl(req: &WebsiteCustomSslRequest) -> Result<WebsiteCustomSslEntry, String> {
        let domain = Self::sanitize_domain(&req.domain);
        if domain.is_empty() {
            return Err("Domain bos olamaz".to_string());
        }

        let cert_pem = Self::sanitize_text_blob(&req.cert_pem);
        let key_pem = Self::sanitize_text_blob(&req.key_pem);

        if cert_pem.trim().is_empty() || key_pem.trim().is_empty() {
            return Err("Sertifika ve private key zorunludur".to_string());
        }

        if !Self::looks_like_pem(&cert_pem) || !Self::looks_like_pem(&key_pem) {
            return Err("PEM formati gecersiz".to_string());
        }

        let mut store = Self::load_store()?;
        let now = Self::now_ts();

        if let Some(existing) = store.custom_ssl.iter_mut().find(|x| x.domain == domain) {
            existing.cert_pem = cert_pem;
            existing.key_pem = key_pem;
            existing.updated_at = now;
            let updated = existing.clone();
            Self::save_store(&store)?;
            return Ok(updated);
        }

        let entry = WebsiteCustomSslEntry {
            domain,
            cert_pem,
            key_pem,
            updated_at: now,
        };
        store.custom_ssl.push(entry.clone());
        Self::save_store(&store)?;
        Ok(entry)
    }

    pub fn cleanup_for_domain(domain: &str) -> Result<(), String> {
        let domain = Self::sanitize_domain(domain);
        if domain.is_empty() {
            return Err("Domain bos olamaz".to_string());
        }

        let mut store = Self::load_store()?;
        store.db_links.retain(|x| x.domain != domain);
        store
            .subdomains
            .retain(|x| x.parent_domain != domain && x.fqdn != domain);
        store.aliases.retain(|x| x.domain != domain && x.alias != domain);
        store.advanced_configs.retain(|x| x.domain != domain);
        store.custom_ssl.retain(|x| x.domain != domain);
        Self::save_store(&store)
    }
}
