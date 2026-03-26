use serde::{Deserialize, Serialize};
use std::sync::{Mutex, OnceLock};

#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct WireguardConfig {
    pub node_name: String,
    pub ip_address: String,
    pub pub_key: String,
}

#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct FederatedModeStatus {
    pub mode: String,
    pub primary: bool,
    pub max_peers: usize,
}

#[derive(Default)]
struct FederatedState {
    nodes: Vec<WireguardConfig>,
}

fn federated_state() -> &'static Mutex<FederatedState> {
    static STATE: OnceLock<Mutex<FederatedState>> = OnceLock::new();
    STATE.get_or_init(|| Mutex::new(FederatedState::default()))
}

pub struct FederatedManager;

impl FederatedManager {
    fn mode() -> String {
        let raw = std::env::var("AURAPANEL_FEDERATION_MODE")
            .unwrap_or_else(|_| "active-passive".to_string())
            .trim()
            .to_ascii_lowercase();

        match raw.as_str() {
            "active-active" => "active-active".to_string(),
            _ => "active-passive".to_string(),
        }
    }

    fn primary() -> bool {
        std::env::var("AURAPANEL_FEDERATION_PRIMARY")
            .map(|v| {
                let normalized = v.trim().to_ascii_lowercase();
                normalized == "1" || normalized == "true" || normalized == "yes" || normalized == "on"
            })
            .unwrap_or(true)
    }

    pub fn mode_status() -> FederatedModeStatus {
        let mode = Self::mode();
        FederatedModeStatus {
            mode: mode.clone(),
            primary: Self::primary(),
            max_peers: if mode == "active-passive" { 1 } else { usize::MAX },
        }
    }

    /// Adds a peer node into the federation topology.
    pub async fn add_cluster_node(config: &WireguardConfig) -> Result<(), String> {
        let mode = Self::mode();
        let is_primary = Self::primary();

        if mode == "active-passive" && !is_primary {
            return Err(
                "This node is configured as passive; join operations are blocked in active-passive mode."
                    .to_string(),
            );
        }

        let mut guard = federated_state().lock().map_err(|e| e.to_string())?;

        if mode == "active-passive"
            && guard.nodes.iter().all(|n| n.node_name != config.node_name)
            && !guard.nodes.is_empty()
        {
            return Err("Active-passive mode allows a single standby peer for this node.".to_string());
        }

        if let Some(existing) = guard.nodes.iter_mut().find(|n| n.node_name == config.node_name) {
            existing.ip_address = config.ip_address.clone();
            existing.pub_key = config.pub_key.clone();
        } else {
            guard.nodes.push(config.clone());
        }

        Ok(())
    }

    pub fn list_cluster_nodes() -> Result<Vec<WireguardConfig>, String> {
        let guard = federated_state().lock().map_err(|e| e.to_string())?;
        Ok(guard.nodes.clone())
    }
}
