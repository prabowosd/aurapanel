use chrono::Utc;
use serde::{Deserialize, Serialize};
use std::fs;
use std::path::{Path, PathBuf};

#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct AuditEntry {
    pub id: u64,
    pub timestamp: String,
    pub username: String,
    pub action: String,
    pub detail: String,
    pub ip: String,
}

pub struct AuditLogger;

impl AuditLogger {
    fn log_path() -> PathBuf {
        let state_dir = if let Ok(dir) = std::env::var("AURAPANEL_STATE_DIR") {
            PathBuf::from(dir)
        } else if Path::new("/var/lib/aurapanel").exists() {
            PathBuf::from("/var/lib/aurapanel")
        } else {
            std::env::temp_dir().join("aurapanel")
        };
        state_dir.join("audit_log.json")
    }

    fn load_entries() -> Vec<AuditEntry> {
        let path = Self::log_path();
        if !path.exists() {
            return Vec::new();
        }
        fs::read_to_string(path)
            .ok()
            .and_then(|s| serde_json::from_str(&s).ok())
            .unwrap_or_default()
    }

    fn save_entries(entries: &[AuditEntry]) -> Result<(), String> {
        let path = Self::log_path();
        if let Some(parent) = path.parent() {
            fs::create_dir_all(parent).map_err(|e| e.to_string())?;
        }
        let json = serde_json::to_string_pretty(entries).map_err(|e| e.to_string())?;
        fs::write(path, json).map_err(|e| e.to_string())
    }

    /// Records an action to the audit log. Call this from any handler after a
    /// successful operation. Silently ignores write failures to avoid blocking.
    pub fn log(username: &str, action: &str, detail: &str, ip: &str) {
        let mut entries = Self::load_entries();
        let next_id = entries.iter().map(|e| e.id).max().unwrap_or(0) + 1;
        entries.push(AuditEntry {
            id: next_id,
            timestamp: Utc::now().to_rfc3339(),
            username: username.to_string(),
            action: action.to_string(),
            detail: detail.to_string(),
            ip: ip.to_string(),
        });
        // Retain at most 10 000 entries to avoid unbounded growth
        if entries.len() > 10_000 {
            let drain_to = entries.len() - 10_000;
            entries.drain(..drain_to);
        }
        let _ = Self::save_entries(&entries);
    }

    /// Returns a page of audit entries (newest first), optionally filtered by
    /// username. Returns `(entries, total_count)`.
    pub fn list(
        filter_user: Option<&str>,
        page: usize,
        per_page: usize,
    ) -> Result<(Vec<AuditEntry>, usize), String> {
        let mut entries = Self::load_entries();
        entries.reverse();

        if let Some(user) = filter_user {
            entries.retain(|e| e.username == user);
        }

        let total = entries.len();
        let start = page * per_page;
        let page_entries = entries.into_iter().skip(start).take(per_page).collect();
        Ok((page_entries, total))
    }
}
