use anyhow::{anyhow, Result};
use totp_rs::{Algorithm, Secret, TOTP};

pub fn generate_totp_secret(account_name: &str) -> Result<(String, String)> {
    let mut bytes = vec![0u8; 20];
    getrandom::getrandom(&mut bytes)
        .map_err(|e| anyhow!("secure random generation failed: {}", e))?;

    let secret = Secret::Raw(bytes);
    let totp = TOTP::new(
        Algorithm::SHA1,
        6,
        1,
        30,
        secret.to_bytes().map_err(|e| anyhow!("{}", e))?,
        Some("AuraPanel".to_string()),
        account_name.to_string(),
    )
    .map_err(|e| anyhow!("{}", e))?;

    let qr_code = totp.get_qr_base64().map_err(|e| anyhow!("{}", e))?;
    let secret_str = secret.to_encoded().to_string();

    Ok((secret_str, qr_code))
}

pub fn verify_totp(secret_str: &str, token: &str) -> Result<bool> {
    let secret = Secret::Encoded(secret_str.to_string());
    let totp = TOTP::new(
        Algorithm::SHA1,
        6,
        1,
        30,
        secret.to_bytes().map_err(|e| anyhow!("{}", e))?,
        Some("AuraPanel".to_string()),
        "".to_string(),
    )
    .map_err(|e| anyhow!("{}", e))?;

    totp.check_current(token).map_err(|e| anyhow!("{}", e))
}
