use serde::{Deserialize, Serialize};

/// Machine Learning tabanlı Web Application Firewall (WAF)
/// HTTP isteklerini analiz eder, zararlı kalıpları yakalar.
pub struct MlWaf;

#[derive(Serialize, Deserialize, Debug)]
pub struct HttpRequest {
    pub method: String,
    pub path: String,
    pub query: String,
    pub body: String,
    pub user_agent: String,
    pub ip: String,
}

#[derive(Serialize, Deserialize, Debug)]
pub struct WafVerdict {
    pub allowed: bool,
    pub score: f64,
    pub reason: String,
}

impl MlWaf {
    /// HTTP isteğini analiz eder ve zararlı olup olmadığını kontrol eder
    pub fn inspect(req: &HttpRequest) -> WafVerdict {
        let mut score: f64 = 0.0;
        let mut reasons = Vec::new();

        let combined = format!("{} {} {}", req.path, req.query, req.body).to_lowercase();

        // SQL Injection kalıpları
        let sqli_patterns = [
            "' or '1'='1",
            "union select",
            "drop table",
            "insert into",
            "delete from",
            "1=1",
            "' or 1=1--",
            "admin'--",
            "sleep(",
            "benchmark(",
            "load_file(",
        ];
        for pattern in &sqli_patterns {
            if combined.contains(pattern) {
                score += 0.4;
                reasons.push(format!("SQLi pattern: {}", pattern));
            }
        }

        // XSS kalıpları
        let xss_patterns = [
            "<script",
            "javascript:",
            "onerror=",
            "onload=",
            "document.cookie",
            "eval(",
            "alert(",
        ];
        for pattern in &xss_patterns {
            if combined.contains(pattern) {
                score += 0.35;
                reasons.push(format!("XSS pattern: {}", pattern));
            }
        }

        // Path Traversal
        let traversal_patterns = ["../", "..\\", "/etc/passwd", "/proc/self"];
        for pattern in &traversal_patterns {
            if combined.contains(pattern) {
                score += 0.5;
                reasons.push(format!("Path traversal: {}", pattern));
            }
        }

        // Command Injection
        let cmdi_patterns = ["; ls", "| cat", "&& rm", "`whoami`", "$(id)"];
        for pattern in &cmdi_patterns {
            if combined.contains(pattern) {
                score += 0.5;
                reasons.push(format!("Command injection: {}", pattern));
            }
        }

        // Şüpheli User-Agent
        let bad_agents = ["sqlmap", "nikto", "nessus", "acunetix", "nmap"];
        let ua_lower = req.user_agent.to_lowercase();
        for agent in &bad_agents {
            if ua_lower.contains(agent) {
                score += 0.6;
                reasons.push(format!("Malicious scanner: {}", agent));
            }
        }

        let allowed = score < 0.3;
        let reason = if reasons.is_empty() {
            "Clean request".to_string()
        } else {
            reasons.join("; ")
        };

        WafVerdict {
            allowed,
            score: score.min(1.0),
            reason,
        }
    }
}
