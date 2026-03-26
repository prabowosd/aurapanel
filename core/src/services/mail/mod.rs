use serde::{Deserialize, Serialize};
use std::collections::hash_map::DefaultHasher;
use std::fs;
use std::hash::{Hash, Hasher};
use std::path::{Path, PathBuf};
use std::time::{SystemTime, UNIX_EPOCH};

#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct MailboxConfig {
    pub domain: String,
    pub username: String,
    pub password: String,
    pub quota_mb: u32,
}

#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct LocalMailbox {
    pub id: u32,
    pub address: String,
    pub domain: String,
    pub quota_mb: u32,
    pub used_mb: u32,
}

#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct MailForwardConfig {
    pub domain: String,
    pub source: String,
    pub target: String,
}

#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct MailForwardRule {
    pub domain: String,
    pub source: String,
    pub target: String,
    pub created_at: u64,
}

#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct MailCatchAllConfig {
    pub domain: String,
    pub enabled: bool,
    pub target: String,
}

#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct MailCatchAllRule {
    pub domain: String,
    pub enabled: bool,
    pub target: String,
    pub updated_at: u64,
}

#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct MailRoutingConfig {
    pub domain: String,
    pub pattern: String,
    pub target: String,
    pub priority: u32,
}

#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct MailRoutingRule {
    pub id: String,
    pub domain: String,
    pub pattern: String,
    pub target: String,
    pub priority: u32,
    pub created_at: u64,
}

#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct MailDkimRecord {
    pub domain: String,
    pub selector: String,
    pub public_key: String,
    pub private_key: String,
    pub updated_at: u64,
}

#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct MailWebmailSsoRequest {
    pub address: String,
    pub ttl_seconds: Option<u64>,
}

#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct MailWebmailSsoLink {
    pub url: String,
    pub expires_at: u64,
}

#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct MailForwardDeleteRequest {
    pub domain: String,
    pub source: String,
}

#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct MailRoutingDeleteRequest {
    pub domain: String,
    pub id: String,
}

#[derive(Serialize, Deserialize, Debug, Clone)]
struct MailState {
    #[serde(default)]
    mailboxes: Vec<LocalMailbox>,
    #[serde(default)]
    forwards: Vec<MailForwardRule>,
    #[serde(default)]
    catch_all: Vec<MailCatchAllRule>,
    #[serde(default)]
    routing: Vec<MailRoutingRule>,
    #[serde(default)]
    dkim: Vec<MailDkimRecord>,
}

impl Default for MailState {
    fn default() -> Self {
        Self {
            mailboxes: Vec::new(),
            forwards: Vec::new(),
            catch_all: Vec::new(),
            routing: Vec::new(),
            dkim: Vec::new(),
        }
    }
}

pub struct MailManager;

impl MailManager {
    fn backend() -> String {
        std::env::var("AURAPANEL_MAIL_BACKEND")
            .unwrap_or_else(|_| "local".to_string())
            .trim()
            .to_ascii_lowercase()
    }

    fn now_ts() -> u64 {
        SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .map(|d| d.as_secs())
            .unwrap_or(0)
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

    fn storage_path() -> PathBuf {
        Self::state_root().join("mail_state.json")
    }

    fn legacy_mailboxes_path() -> PathBuf {
        Self::state_root().join("mailboxes.json")
    }

    fn ensure_parent(path: &Path) -> Result<(), String> {
        if let Some(parent) = path.parent() {
            fs::create_dir_all(parent).map_err(|e| e.to_string())?;
        }
        Ok(())
    }

    fn normalize_domain(domain: &str) -> String {
        domain.trim().trim_end_matches('.').to_ascii_lowercase()
    }

    fn sanitize_local_part(value: &str) -> String {
        value
            .trim()
            .to_ascii_lowercase()
            .chars()
            .filter(|c| c.is_ascii_alphanumeric() || *c == '.' || *c == '_' || *c == '-')
            .collect()
    }

    fn normalize_address(domain: &str, source: &str) -> String {
        let source = source.trim();
        if source.contains('@') {
            source.to_ascii_lowercase()
        } else {
            format!("{}@{}", Self::sanitize_local_part(source), Self::normalize_domain(domain))
        }
    }

    fn load_state() -> Result<MailState, String> {
        let path = Self::storage_path();
        if path.exists() {
            let raw = fs::read_to_string(&path).map_err(|e| e.to_string())?;
            return serde_json::from_str::<MailState>(&raw).map_err(|e| e.to_string());
        }

        // Legacy migration: old mailbox-only file
        let legacy = Self::legacy_mailboxes_path();
        if legacy.exists() {
            let raw = fs::read_to_string(&legacy).map_err(|e| e.to_string())?;
            let mailboxes = serde_json::from_str::<Vec<LocalMailbox>>(&raw).unwrap_or_default();
            let state = MailState {
                mailboxes,
                ..MailState::default()
            };
            Self::save_state(&state)?;
            return Ok(state);
        }

        Ok(MailState::default())
    }

    fn save_state(state: &MailState) -> Result<(), String> {
        let path = Self::storage_path();
        Self::ensure_parent(&path)?;
        let payload = serde_json::to_string_pretty(state).map_err(|e| e.to_string())?;
        fs::write(path, payload).map_err(|e| e.to_string())
    }

    fn validate_backend_for_write() -> Result<(), String> {
        let backend = Self::backend();
        if backend == "local" {
            return Ok(());
        }
        if crate::runtime::simulation_enabled() {
            println!("[DEV MODE] Mail backend '{}' simulated.", backend);
            return Ok(());
        }
        Err(format!(
            "Mail backend '{}' is not implemented yet. Set AURAPANEL_MAIL_BACKEND=local or enable AURAPANEL_DEV_SIMULATION=1.",
            backend
        ))
    }

    pub fn list_mailboxes() -> Vec<LocalMailbox> {
        Self::load_state().map(|s| s.mailboxes).unwrap_or_default()
    }

    pub async fn create_mailbox(config: &MailboxConfig) -> Result<(), String> {
        Self::validate_backend_for_write()?;

        let domain = Self::normalize_domain(&config.domain);
        let username = Self::sanitize_local_part(&config.username);
        if domain.is_empty() || username.is_empty() {
            return Err("domain and username are required.".to_string());
        }

        let email_address = format!("{}@{}", username, domain);

        let mut state = Self::load_state()?;
        if state.mailboxes.iter().any(|m| m.address == email_address) {
            return Ok(());
        }

        let next_id = state.mailboxes.iter().map(|m| m.id).max().unwrap_or(0) + 1;
        state.mailboxes.push(LocalMailbox {
            id: next_id,
            address: email_address,
            domain,
            quota_mb: config.quota_mb.max(128),
            used_mb: 0,
        });
        Self::save_state(&state)
    }

    pub async fn delete_mailbox(email: &str) -> Result<(), String> {
        Self::validate_backend_for_write()?;
        let address = email.trim().to_ascii_lowercase();
        if address.is_empty() {
            return Err("address is required.".to_string());
        }

        let mut state = Self::load_state()?;
        state.mailboxes.retain(|m| m.address != address);
        state.forwards.retain(|f| f.source != address);
        Self::save_state(&state)
    }

    pub fn list_forwards(domain: Option<&str>) -> Result<Vec<MailForwardRule>, String> {
        let mut rules = Self::load_state()?.forwards;
        if let Some(d) = domain {
            let d = Self::normalize_domain(d);
            rules.retain(|x| x.domain == d);
        }
        rules.sort_by(|a, b| a.domain.cmp(&b.domain).then(a.source.cmp(&b.source)));
        Ok(rules)
    }

    pub fn add_forward(config: &MailForwardConfig) -> Result<MailForwardRule, String> {
        Self::validate_backend_for_write()?;

        let domain = Self::normalize_domain(&config.domain);
        let source = Self::normalize_address(&domain, &config.source);
        let target = config.target.trim().to_ascii_lowercase();

        if domain.is_empty() || source.is_empty() || target.is_empty() || !target.contains('@') {
            return Err("domain, source and valid target address are required.".to_string());
        }

        let mut state = Self::load_state()?;
        let per_domain_count = state.forwards.iter().filter(|x| x.domain == domain).count();
        if per_domain_count >= 200 {
            return Err("Forward rule limit reached for domain (200).".to_string());
        }

        if let Some(existing) = state
            .forwards
            .iter_mut()
            .find(|x| x.domain == domain && x.source == source)
        {
            existing.target = target.clone();
            let updated = existing.clone();
            Self::save_state(&state)?;
            return Ok(updated);
        }

        let entry = MailForwardRule {
            domain,
            source,
            target,
            created_at: Self::now_ts(),
        };
        state.forwards.push(entry.clone());
        Self::save_state(&state)?;
        Ok(entry)
    }

    pub fn delete_forward(config: &MailForwardDeleteRequest) -> Result<(), String> {
        Self::validate_backend_for_write()?;

        let domain = Self::normalize_domain(&config.domain);
        let source = Self::normalize_address(&domain, &config.source);
        let mut state = Self::load_state()?;
        let before = state.forwards.len();
        state.forwards.retain(|x| !(x.domain == domain && x.source == source));
        if before == state.forwards.len() {
            return Err("Forward rule not found.".to_string());
        }
        Self::save_state(&state)
    }

    pub fn get_catch_all(domain: &str) -> Result<Option<MailCatchAllRule>, String> {
        let domain = Self::normalize_domain(domain);
        if domain.is_empty() {
            return Err("domain is required.".to_string());
        }
        let state = Self::load_state()?;
        Ok(state.catch_all.into_iter().find(|x| x.domain == domain))
    }

    pub fn set_catch_all(config: &MailCatchAllConfig) -> Result<MailCatchAllRule, String> {
        Self::validate_backend_for_write()?;

        let domain = Self::normalize_domain(&config.domain);
        let target = config.target.trim().to_ascii_lowercase();
        if domain.is_empty() {
            return Err("domain is required.".to_string());
        }
        if config.enabled && (target.is_empty() || !target.contains('@')) {
            return Err("target address is required when catch-all is enabled.".to_string());
        }

        let mut state = Self::load_state()?;
        let now = Self::now_ts();

        if let Some(existing) = state.catch_all.iter_mut().find(|x| x.domain == domain) {
            existing.enabled = config.enabled;
            existing.target = target.clone();
            existing.updated_at = now;
            let out = existing.clone();
            Self::save_state(&state)?;
            return Ok(out);
        }

        let entry = MailCatchAllRule {
            domain,
            enabled: config.enabled,
            target,
            updated_at: now,
        };
        state.catch_all.push(entry.clone());
        Self::save_state(&state)?;
        Ok(entry)
    }

    pub fn list_routing_rules(domain: Option<&str>) -> Result<Vec<MailRoutingRule>, String> {
        let mut items = Self::load_state()?.routing;
        if let Some(d) = domain {
            let d = Self::normalize_domain(d);
            items.retain(|x| x.domain == d);
        }
        items.sort_by(|a, b| a.priority.cmp(&b.priority).then(a.pattern.cmp(&b.pattern)));
        Ok(items)
    }

    pub fn add_routing_rule(config: &MailRoutingConfig) -> Result<MailRoutingRule, String> {
        Self::validate_backend_for_write()?;

        let domain = Self::normalize_domain(&config.domain);
        let pattern = config.pattern.trim().to_string();
        let target = config.target.trim().to_ascii_lowercase();

        if domain.is_empty() || pattern.is_empty() || target.is_empty() {
            return Err("domain, pattern and target are required.".to_string());
        }

        let mut state = Self::load_state()?;
        let per_domain_count = state.routing.iter().filter(|x| x.domain == domain).count();
        if per_domain_count >= 128 {
            return Err("Routing rule limit reached for domain (128).".to_string());
        }

        let id_seed = format!("{}:{}:{}", domain, pattern, Self::now_ts());
        let mut hasher = DefaultHasher::new();
        id_seed.hash(&mut hasher);
        let id = format!("rt-{:x}", hasher.finish());

        let entry = MailRoutingRule {
            id,
            domain,
            pattern,
            target,
            priority: config.priority,
            created_at: Self::now_ts(),
        };
        state.routing.push(entry.clone());
        Self::save_state(&state)?;
        Ok(entry)
    }

    pub fn delete_routing_rule(config: &MailRoutingDeleteRequest) -> Result<(), String> {
        Self::validate_backend_for_write()?;

        let domain = Self::normalize_domain(&config.domain);
        let id = config.id.trim().to_string();
        let mut state = Self::load_state()?;
        let before = state.routing.len();
        state.routing.retain(|x| !(x.domain == domain && x.id == id));
        if before == state.routing.len() {
            return Err("Routing rule not found.".to_string());
        }
        Self::save_state(&state)
    }

    pub fn get_dkim(domain: &str) -> Result<Option<MailDkimRecord>, String> {
        let domain = Self::normalize_domain(domain);
        if domain.is_empty() {
            return Err("domain is required.".to_string());
        }
        let state = Self::load_state()?;
        Ok(state.dkim.into_iter().find(|x| x.domain == domain))
    }

    pub fn rotate_dkim(domain: &str) -> Result<MailDkimRecord, String> {
        Self::validate_backend_for_write()?;

        let domain = Self::normalize_domain(domain);
        if domain.is_empty() {
            return Err("domain is required.".to_string());
        }

        let now = Self::now_ts();
        let selector = format!("s{}", now);
        let public_key = format!("v=DKIM1; k=rsa; p={:x}", now.saturating_mul(17));
        let private_key = format!("-----BEGIN PRIVATE KEY-----\n{:x}\n-----END PRIVATE KEY-----", now.saturating_mul(31));

        let mut state = Self::load_state()?;
        if let Some(existing) = state.dkim.iter_mut().find(|x| x.domain == domain) {
            existing.selector = selector;
            existing.public_key = public_key;
            existing.private_key = private_key;
            existing.updated_at = now;
            let out = existing.clone();
            Self::save_state(&state)?;
            return Ok(out);
        }

        let entry = MailDkimRecord {
            domain,
            selector,
            public_key,
            private_key,
            updated_at: now,
        };
        state.dkim.push(entry.clone());
        Self::save_state(&state)?;
        Ok(entry)
    }

    pub fn generate_webmail_sso_link(req: &MailWebmailSsoRequest) -> Result<MailWebmailSsoLink, String> {
        let address = req.address.trim().to_ascii_lowercase();
        if address.is_empty() || !address.contains('@') {
            return Err("valid address is required.".to_string());
        }

        let ttl = req.ttl_seconds.unwrap_or(300).clamp(60, 1800);
        let expires_at = Self::now_ts().saturating_add(ttl);
        let secret = std::env::var("AURAPANEL_JWT_SECRET").unwrap_or_else(|_| "aurapanel-mail-sso".to_string());
        let payload = format!("{}:{}", address, expires_at);
        let mut hasher = DefaultHasher::new();
        payload.hash(&mut hasher);
        secret.hash(&mut hasher);
        let sig = hasher.finish();
        let token = format!("{:x}.{:x}", sig, expires_at);

        let base = std::env::var("AURAPANEL_WEBMAIL_BASE_URL")
            .unwrap_or_else(|_| "https://webmail.local".to_string())
            .trim()
            .trim_end_matches('/')
            .to_string();

        Ok(MailWebmailSsoLink {
            url: format!("{}/sso?address={}&token={}", base, address, token),
            expires_at,
        })
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::sync::{Mutex, OnceLock};
    use std::time::{SystemTime, UNIX_EPOCH};

    fn test_lock() -> &'static Mutex<()> {
        static LOCK: OnceLock<Mutex<()>> = OnceLock::new();
        LOCK.get_or_init(|| Mutex::new(()))
    }

    fn setup_env(test_name: &str) -> std::path::PathBuf {
        let now = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .map(|d| d.as_nanos())
            .unwrap_or(0);
        let path = std::env::temp_dir().join(format!("aurapanel-mail-test-{}-{}", test_name, now));
        std::fs::create_dir_all(&path).expect("temp state dir");
        std::env::set_var("AURAPANEL_STATE_DIR", &path);
        std::env::set_var("AURAPANEL_MAIL_BACKEND", "local");
        path
    }

    fn teardown_env(path: &std::path::Path) {
        std::env::remove_var("AURAPANEL_STATE_DIR");
        std::env::remove_var("AURAPANEL_MAIL_BACKEND");
        let _ = std::fs::remove_dir_all(path);
    }

    #[test]
    fn forward_rule_lifecycle_works() {
        let _guard = test_lock().lock().expect("test lock");
        let state_dir = setup_env("forward");

        let created = MailManager::add_forward(&MailForwardConfig {
            domain: "Example.COM".to_string(),
            source: "info".to_string(),
            target: "target@example.net".to_string(),
        })
        .expect("forward create");
        assert_eq!(created.domain, "example.com");
        assert_eq!(created.source, "info@example.com");

        let listed = MailManager::list_forwards(Some("example.com")).expect("forward list");
        assert_eq!(listed.len(), 1);

        MailManager::delete_forward(&MailForwardDeleteRequest {
            domain: "example.com".to_string(),
            source: "info".to_string(),
        })
        .expect("forward delete");
        let listed_after = MailManager::list_forwards(Some("example.com")).expect("forward list");
        assert!(listed_after.is_empty());

        teardown_env(&state_dir);
    }

    #[test]
    fn catch_all_routing_and_dkim_work() {
        let _guard = test_lock().lock().expect("test lock");
        let state_dir = setup_env("mailops");

        let catch_all = MailManager::set_catch_all(&MailCatchAllConfig {
            domain: "example.com".to_string(),
            enabled: true,
            target: "ops@example.com".to_string(),
        })
        .expect("set catch all");
        assert!(catch_all.enabled);
        assert_eq!(catch_all.target, "ops@example.com");

        let route = MailManager::add_routing_rule(&MailRoutingConfig {
            domain: "example.com".to_string(),
            pattern: "invoice*".to_string(),
            target: "billing@example.com".to_string(),
            priority: 10,
        })
        .expect("routing create");
        assert!(route.id.starts_with("rt-"));

        let routing = MailManager::list_routing_rules(Some("example.com")).expect("routing list");
        assert_eq!(routing.len(), 1);

        let dkim = MailManager::rotate_dkim("example.com").expect("dkim rotate");
        assert_eq!(dkim.domain, "example.com");
        assert!(dkim.selector.starts_with('s'));

        teardown_env(&state_dir);
    }

    #[test]
    fn webmail_sso_requires_valid_address() {
        let _guard = test_lock().lock().expect("test lock");
        let state_dir = setup_env("sso");

        let invalid = MailManager::generate_webmail_sso_link(&MailWebmailSsoRequest {
            address: "invalid".to_string(),
            ttl_seconds: Some(300),
        });
        assert!(invalid.is_err());

        let link = MailManager::generate_webmail_sso_link(&MailWebmailSsoRequest {
            address: "user@example.com".to_string(),
            ttl_seconds: Some(120),
        })
        .expect("sso link");
        assert!(link.url.contains("address=user@example.com"));
        assert!(link.url.contains("token="));

        teardown_env(&state_dir);
    }
}
