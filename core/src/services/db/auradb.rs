use anyhow::{anyhow, Result};
use jsonwebtoken::{decode, encode, Algorithm, DecodingKey, EncodingKey, Header, Validation};
use serde::{Deserialize, Serialize};
use std::time::{SystemTime, UNIX_EPOCH};

const BRIDGE_ISSUER: &str = "aurapanel-core";
const BRIDGE_SUBJECT: &str = "auradb-bridge";
const DEFAULT_TTL_SECONDS: u64 = 600;
const MAX_TTL_SECONDS: u64 = 3600;

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct DbBridgeProfile {
    pub domain: String,
    pub engine: String,
    pub db_name: String,
    pub db_user: String,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct DbBridgeTicket {
    pub token: String,
    pub expires_at: u64,
    pub profile: DbBridgeProfile,
}

#[derive(Debug, Serialize, Deserialize)]
struct DbBridgeClaims {
    iss: String,
    sub: String,
    iat: u64,
    exp: u64,
    domain: String,
    engine: String,
    db_name: String,
    db_user: String,
}

fn now_ts() -> u64 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map(|d| d.as_secs())
        .unwrap_or(0)
}

fn sanitize_domain(value: &str) -> String {
    value.trim().to_ascii_lowercase().trim_matches('.').to_string()
}

fn sanitize_name(value: &str) -> String {
    value.trim().to_string()
}

fn normalize_engine(engine: &str) -> Option<String> {
    let e = engine.trim().to_ascii_lowercase();
    match e.as_str() {
        "mysql" | "mariadb" => Some("mariadb".to_string()),
        "postgres" | "postgresql" => Some("postgresql".to_string()),
        _ => None,
    }
}

fn bridge_secret() -> Result<String> {
    let secret = std::env::var("AURAPANEL_JWT_SECRET")
        .map_err(|_| anyhow!("AURAPANEL_JWT_SECRET is required for secure DB bridge."))?;
    let secret = secret.trim().to_string();
    if secret.is_empty() {
        return Err(anyhow!("AURAPANEL_JWT_SECRET cannot be empty for secure DB bridge."));
    }
    Ok(secret)
}

fn build_profile(domain: &str, engine: &str, db_name: &str, db_user: &str) -> Result<DbBridgeProfile> {
    let domain = sanitize_domain(domain);
    let engine = normalize_engine(engine).ok_or_else(|| anyhow!("Unsupported database engine."))?;
    let db_name = sanitize_name(db_name);
    let db_user = sanitize_name(db_user);

    if domain.is_empty() || db_name.is_empty() || db_user.is_empty() {
        return Err(anyhow!("domain, db_name and db_user are required for bridge creation."));
    }

    Ok(DbBridgeProfile {
        domain,
        engine,
        db_name,
        db_user,
    })
}

fn issue_bridge_ticket(profile: DbBridgeProfile, ttl_seconds: Option<u64>) -> Result<DbBridgeTicket> {
    let now = now_ts();
    let ttl = ttl_seconds.unwrap_or(DEFAULT_TTL_SECONDS).clamp(60, MAX_TTL_SECONDS);
    let exp = now.saturating_add(ttl);

    let claims = DbBridgeClaims {
        iss: BRIDGE_ISSUER.to_string(),
        sub: BRIDGE_SUBJECT.to_string(),
        iat: now,
        exp,
        domain: profile.domain.clone(),
        engine: profile.engine.clone(),
        db_name: profile.db_name.clone(),
        db_user: profile.db_user.clone(),
    };

    let secret = bridge_secret()?;
    let token = encode(
        &Header::new(Algorithm::HS256),
        &claims,
        &EncodingKey::from_secret(secret.as_bytes()),
    )?;

    Ok(DbBridgeTicket {
        token,
        expires_at: exp,
        profile,
    })
}

fn resolve_bridge_ticket(token: &str) -> Result<DbBridgeProfile> {
    let token = token.trim();
    if token.is_empty() {
        return Err(anyhow!("Bridge token is required."));
    }

    let secret = bridge_secret()?;
    let mut validation = Validation::new(Algorithm::HS256);
    validation.set_issuer(&[BRIDGE_ISSUER]);
    validation.validate_exp = true;

    let decoded = decode::<DbBridgeClaims>(
        token,
        &DecodingKey::from_secret(secret.as_bytes()),
        &validation,
    )?;

    let claims = decoded.claims;
    if claims.sub != BRIDGE_SUBJECT {
        return Err(anyhow!("Invalid bridge token subject."));
    }

    build_profile(&claims.domain, &claims.engine, &claims.db_name, &claims.db_user)
}

pub struct DbExplorerManager;

impl DbExplorerManager {
    pub fn new() -> Self {
        Self {}
    }

    pub fn create_bridge_ticket(
        domain: &str,
        engine: &str,
        db_name: &str,
        db_user: &str,
        ttl_seconds: Option<u64>,
    ) -> Result<DbBridgeTicket> {
        let profile = build_profile(domain, engine, db_name, db_user)?;
        issue_bridge_ticket(profile, ttl_seconds)
    }

    pub fn resolve_bridge_token(token: &str) -> Result<DbBridgeProfile> {
        resolve_bridge_ticket(token)
    }

    pub fn execute_query(&self, db_type: &str, connection_string: &str, query: &str) -> Result<String> {
        println!("Executing {} query on {}: {}", db_type, connection_string, query);

        let mock_result = serde_json::json!({
            "status": "success",
            "mode": if crate::runtime::simulation_enabled() { "simulated" } else { "dry-run" },
            "rows_affected": 0,
            "data": []
        });

        Ok(mock_result.to_string())
    }

    pub fn execute_query_with_bridge(&self, bridge_token: &str, query: &str) -> Result<String> {
        let profile = Self::resolve_bridge_token(bridge_token)?;
        let connection_string = format!(
            "bridge://{}/{}/{}@{}",
            profile.domain, profile.engine, profile.db_name, profile.db_user
        );
        self.execute_query(&profile.engine, &connection_string, query)
    }

    pub fn list_tables(&self, _db_type: &str, _connection_string: &str) -> Result<Vec<String>> {
        Ok(vec!["users".to_string(), "posts".to_string()])
    }

    pub fn list_tables_with_bridge(&self, bridge_token: &str) -> Result<Vec<String>> {
        let _ = Self::resolve_bridge_token(bridge_token)?;
        self.list_tables("bridge", "bridge")
    }

    pub fn create_database(&self, db_name: &str, user: &str, pass: &str) -> Result<bool> {
        println!("Creating database {} for user {} with password {}", db_name, user, pass);
        Ok(true)
    }
}
