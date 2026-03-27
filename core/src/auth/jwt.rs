use jsonwebtoken::{decode, encode, DecodingKey, EncodingKey, Header, Validation};
use serde::{Deserialize, Serialize};
use std::time::{SystemTime, UNIX_EPOCH};

#[derive(Debug, Serialize, Deserialize)]
pub struct Claims {
    pub sub: String,  // username or user_id
    pub role: String, // admin, reseller, user
    pub exp: usize,
    #[serde(default)]
    pub iss: Option<String>,
    #[serde(default)]
    pub aud: Vec<String>,
    #[serde(default)]
    pub iat: Option<usize>,
    #[serde(default)]
    pub nbf: Option<usize>,
}

fn jwt_issuer() -> String {
    std::env::var("AURAPANEL_JWT_ISSUER")
        .ok()
        .map(|value| value.trim().to_string())
        .filter(|value| !value.is_empty())
        .unwrap_or_else(|| "aurapanel-gateway".to_string())
}

fn jwt_audience() -> String {
    std::env::var("AURAPANEL_JWT_AUDIENCE")
        .ok()
        .map(|value| value.trim().to_string())
        .filter(|value| !value.is_empty())
        .unwrap_or_else(|| "aurapanel-ui".to_string())
}

fn secret_key() -> Result<Vec<u8>, String> {
    if let Ok(value) = std::env::var("AURAPANEL_JWT_SECRET") {
        let trimmed = value.trim();
        if !trimmed.is_empty() {
            return Ok(trimmed.as_bytes().to_vec());
        }
    }

    if cfg!(debug_assertions) {
        return Ok(b"aurapanel_dev_only_secret_change_me".to_vec());
    }

    Err("AURAPANEL_JWT_SECRET tanimli degil.".to_string())
}

pub fn create_token(user_id: &str, role: &str) -> Result<String, String> {
    let now = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .expect("Time went backwards")
        .as_secs();
    let expiration = now + 24 * 3600; // 24 hours

    let claims = Claims {
        sub: user_id.to_string(),
        role: role.to_string(),
        exp: expiration as usize,
        iss: Some(jwt_issuer()),
        aud: vec![jwt_audience()],
        iat: Some(now as usize),
        nbf: Some(now as usize),
    };

    let secret = secret_key()?;
    encode(
        &Header::default(),
        &claims,
        &EncodingKey::from_secret(&secret),
    )
    .map_err(|e| e.to_string())
}

pub fn verify_token(token: &str) -> Result<Claims, String> {
    let secret = secret_key()?;
    let mut validation = Validation::new(jsonwebtoken::Algorithm::HS256);
    let issuer = jwt_issuer();
    let audience = jwt_audience();
    validation.set_issuer(&[issuer.as_str()]);
    validation.set_audience(&[audience.as_str()]);
    validation.leeway = 30;
    let token_data = decode::<Claims>(token, &DecodingKey::from_secret(&secret), &validation)
        .map_err(|e| e.to_string())?;
    Ok(token_data.claims)
}
