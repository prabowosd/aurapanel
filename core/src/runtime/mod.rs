use std::net::SocketAddr;
use std::sync::OnceLock;

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum RuntimeMode {
    Development,
    Staging,
    Production,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum SecurityPolicy {
    FailClosed,
    FailOpen,
}

fn parse_flag(value: &str) -> bool {
    let normalized = value.trim().to_ascii_lowercase();
    normalized == "1" || normalized == "true" || normalized == "yes" || normalized == "on"
}

fn parse_mode(value: &str) -> RuntimeMode {
    match value.trim().to_ascii_lowercase().as_str() {
        "production" | "prod" => RuntimeMode::Production,
        "staging" | "stage" => RuntimeMode::Staging,
        _ => RuntimeMode::Development,
    }
}

fn parse_security_policy(value: &str) -> SecurityPolicy {
    match value.trim().to_ascii_lowercase().as_str() {
        "fail-open" | "open" => SecurityPolicy::FailOpen,
        _ => SecurityPolicy::FailClosed,
    }
}

pub fn mode() -> RuntimeMode {
    static MODE: OnceLock<RuntimeMode> = OnceLock::new();
    *MODE.get_or_init(|| {
        let raw = std::env::var("AURAPANEL_RUNTIME_MODE").unwrap_or_else(|_| "development".to_string());
        parse_mode(&raw)
    })
}

pub fn mode_name() -> &'static str {
    match mode() {
        RuntimeMode::Development => "development",
        RuntimeMode::Staging => "staging",
        RuntimeMode::Production => "production",
    }
}

pub fn simulation_flag_set() -> bool {
    std::env::var("AURAPANEL_DEV_SIMULATION")
        .map(|v| parse_flag(&v))
        .unwrap_or(false)
}

pub fn simulation_enabled() -> bool {
    simulation_flag_set() && mode() != RuntimeMode::Production
}

pub fn is_production() -> bool {
    mode() == RuntimeMode::Production
}

pub fn security_policy() -> SecurityPolicy {
    static POLICY: OnceLock<SecurityPolicy> = OnceLock::new();
    *POLICY.get_or_init(|| {
        let raw = std::env::var("AURAPANEL_SECURITY_POLICY")
            .unwrap_or_else(|_| "fail-closed".to_string());
        parse_security_policy(&raw)
    })
}

pub fn security_policy_name() -> &'static str {
    match security_policy() {
        SecurityPolicy::FailClosed => "fail-closed",
        SecurityPolicy::FailOpen => "fail-open",
    }
}

pub fn fail_closed() -> bool {
    security_policy() == SecurityPolicy::FailClosed
}

pub fn core_bind_addr() -> String {
    std::env::var("AURAPANEL_CORE_BIND_ADDR")
        .unwrap_or_else(|_| "127.0.0.1:8000".to_string())
        .trim()
        .to_string()
}

pub fn gateway_only_enabled() -> bool {
    std::env::var("AURAPANEL_GATEWAY_ONLY")
        .map(|v| parse_flag(&v))
        .unwrap_or(true)
}

fn is_loopback_bind(addr: &str) -> bool {
    if let Ok(parsed) = addr.parse::<SocketAddr>() {
        return parsed.ip().is_loopback();
    }

    addr.starts_with("127.0.0.1:") || addr.starts_with("[::1]:") || addr.eq_ignore_ascii_case("localhost:8000")
}

pub fn validate_startup() -> Result<(), String> {
    if is_production() && simulation_flag_set() {
        return Err("AURAPANEL_DEV_SIMULATION cannot be enabled in production runtime mode".to_string());
    }

    if is_production() && security_policy() == SecurityPolicy::FailOpen {
        return Err("AURAPANEL_SECURITY_POLICY=fail-open is not allowed in production. Use fail-closed.".to_string());
    }

    if is_production() && gateway_only_enabled() && !is_loopback_bind(&core_bind_addr()) {
        return Err(format!(
            "Gateway-only topology requires core loopback bind. Set AURAPANEL_CORE_BIND_ADDR=127.0.0.1:8000 (got: {}).",
            core_bind_addr()
        ));
    }

    Ok(())
}

#[cfg(test)]
mod tests {
    use super::{
        is_loopback_bind, parse_flag, parse_mode, parse_security_policy, RuntimeMode, SecurityPolicy,
    };

    #[test]
    fn parse_flag_accepts_truthy_values() {
        assert!(parse_flag("1"));
        assert!(parse_flag("true"));
        assert!(parse_flag("YES"));
        assert!(parse_flag("On"));
        assert!(!parse_flag("0"));
        assert!(!parse_flag("false"));
    }

    #[test]
    fn parse_mode_maps_expected_variants() {
        assert_eq!(parse_mode("prod"), RuntimeMode::Production);
        assert_eq!(parse_mode("production"), RuntimeMode::Production);
        assert_eq!(parse_mode("stage"), RuntimeMode::Staging);
        assert_eq!(parse_mode("staging"), RuntimeMode::Staging);
        assert_eq!(parse_mode("development"), RuntimeMode::Development);
        assert_eq!(parse_mode("unknown"), RuntimeMode::Development);
    }

    #[test]
    fn parse_security_policy_maps_expected_variants() {
        assert_eq!(parse_security_policy("fail-closed"), SecurityPolicy::FailClosed);
        assert_eq!(parse_security_policy("open"), SecurityPolicy::FailOpen);
        assert_eq!(parse_security_policy("FAIL-OPEN"), SecurityPolicy::FailOpen);
    }

    #[test]
    fn loopback_bind_detection_works() {
        assert!(is_loopback_bind("127.0.0.1:8000"));
        assert!(is_loopback_bind("[::1]:8000"));
        assert!(!is_loopback_bind("0.0.0.0:8000"));
    }
}
