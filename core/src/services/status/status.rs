use serde::{Deserialize, Serialize};
use std::process::Command;

#[derive(Serialize, Deserialize, Debug)]
pub struct SystemMetrics {
    pub cpu_usage: f64,
    pub cpu_cores: u32,
    pub cpu_model: String,
    pub ram_usage: f64,
    pub ram_used: String,
    pub ram_total: String,
    pub disk_usage: f64,
    pub disk_used: String,
    pub disk_total: String,
    pub uptime_seconds: u64,
    pub uptime_human: String,
    pub load_avg: String,
}

#[derive(Serialize, Deserialize, Debug)]
pub struct ServiceStatus {
    pub name: String,
    pub status: String, // "running", "stopped", "inactive"
    pub desc: String,
}

#[derive(Serialize, Deserialize, Debug)]
pub struct ProcessInfo {
    pub pid: u32,
    pub user: String,
    pub cpu: f64,
    pub mem: f64,
    pub command: String,
}

#[derive(Serialize, Deserialize, Debug)]
pub struct NetworkStats {
    pub tx_rate: String,
    pub rx_rate: String,
    pub tx_total: String,
    pub rx_total: String,
    pub http_conns: u32,
    pub ssh_conns: u32,
    pub mysql_conns: u32,
    pub pg_conns: u32,
}

pub struct StatusManager;

impl StatusManager {
    fn is_dev_mode() -> bool {
        cfg!(windows) || !std::path::Path::new("/proc/loadavg").exists()
    }

    /// Sistem metriklerini dÃ¶ndÃ¼rÃ¼r
    pub fn get_metrics() -> Result<SystemMetrics, String> {
        if Self::is_dev_mode() {
            return Ok(SystemMetrics {
                cpu_usage: 23.5,
                cpu_cores: 8,
                cpu_model: "AMD EPYC 7443P".to_string(),
                ram_usage: 58.0,
                ram_used: "11.6 GB".to_string(),
                ram_total: "20 GB".to_string(),
                disk_usage: 42.0,
                disk_used: "84 GB".to_string(),
                disk_total: "200 GB".to_string(),
                uptime_seconds: 4_086_000,
                uptime_human: "47 gÃ¼n, 14 saat, 22 dk".to_string(),
                load_avg: "0.45, 0.38, 0.41".to_string(),
            });
        }

        // CPU cores
        let cpu_cores = std::fs::read_to_string("/proc/cpuinfo")
            .unwrap_or_default()
            .lines()
            .filter(|l| l.starts_with("processor"))
            .count() as u32;

        let cpu_model = std::fs::read_to_string("/proc/cpuinfo")
            .unwrap_or_default()
            .lines()
            .find(|l| l.starts_with("model name"))
            .map(|l| l.split(':').nth(1).unwrap_or("").trim().to_string())
            .unwrap_or_else(|| "Unknown".to_string());

        // Load Average
        let load_avg = std::fs::read_to_string("/proc/loadavg")
            .unwrap_or_default()
            .split_whitespace()
            .take(3)
            .collect::<Vec<_>>()
            .join(", ");
        
        let cpu_usage = std::fs::read_to_string("/proc/loadavg")
            .unwrap_or_default()
            .split_whitespace()
            .next()
            .unwrap_or("0")
            .parse::<f64>()
            .unwrap_or(0.0)
            / cpu_cores.max(1) as f64 * 100.0;

        // Memory
        let meminfo = std::fs::read_to_string("/proc/meminfo").unwrap_or_default();
        let mem_total = Self::parse_meminfo(&meminfo, "MemTotal");
        let mem_avail = Self::parse_meminfo(&meminfo, "MemAvailable");
        let mem_used = mem_total.saturating_sub(mem_avail);
        let ram_usage = if mem_total > 0 { (mem_used as f64 / mem_total as f64) * 100.0 } else { 0.0 };

        // Disk
        let df_output = Command::new("df")
            .args(&["-B1", "/"])
            .output()
            .map(|o| String::from_utf8_lossy(&o.stdout).to_string())
            .unwrap_or_default();
        let (disk_total, disk_used, disk_pct) = Self::parse_df(&df_output);

        // Uptime
        let uptime_str = std::fs::read_to_string("/proc/uptime").unwrap_or_default();
        let uptime_secs = uptime_str.split_whitespace().next().unwrap_or("0").parse::<f64>().unwrap_or(0.0) as u64;
        let days = uptime_secs / 86400;
        let hours = (uptime_secs % 86400) / 3600;
        let mins = (uptime_secs % 3600) / 60;

        Ok(SystemMetrics {
            cpu_usage: (cpu_usage * 10.0).round() / 10.0,
            cpu_cores,
            cpu_model,
            ram_usage: (ram_usage * 10.0).round() / 10.0,
            ram_used: Self::format_bytes(mem_used * 1024),
            ram_total: Self::format_bytes(mem_total * 1024),
            disk_usage: disk_pct,
            disk_used,
            disk_total,
            uptime_seconds: uptime_secs,
            uptime_human: format!("{} gÃ¼n, {} saat, {} dk", days, hours, mins),
            load_avg,
        })
    }

    /// Servislerin durumlarÄ±nÄ± dÃ¶ndÃ¼rÃ¼r
    pub fn get_services() -> Result<Vec<ServiceStatus>, String> {
        let service_list = vec![
            ("lshttpd", "OpenLiteSpeed", "Web Sunucusu"),
            ("mariadb", "MariaDB", "VeritabanÄ± Sunucusu"),
            ("postgresql", "PostgreSQL", "PostgreSQL 16"),
            ("php8.3-fpm", "PHP 8.3-FPM", "PHP Ä°ÅŸleme Motoru"),
            ("postfix", "Postfix", "SMTP Sunucu"),
            ("dovecot", "Dovecot", "IMAP/POP3 Sunucu"),
            ("pdns", "PowerDNS", "DNS Sunucu"),
            ("redis-server", "Redis", "Cache Sunucu"),
            ("docker", "Docker", "Container Engine"),
            ("fail2ban", "Fail2Ban", "Brute-force KorumasÄ±"),
        ];

        if Self::is_dev_mode() {
            return Ok(service_list.iter().map(|(_, name, desc)| {
                ServiceStatus {
                    name: name.to_string(),
                    status: "running".to_string(),
                    desc: desc.to_string(),
                }
            }).collect());
        }

        let mut results = Vec::new();
        for (unit, name, desc) in &service_list {
            let output = Command::new("systemctl")
                .args(&["is-active", unit])
                .output();
            let status = match output {
                Ok(o) => {
                    let s = String::from_utf8_lossy(&o.stdout).trim().to_string();
                    if s == "active" { "running" } else { "stopped" }
                }
                Err(_) => "stopped",
            };
            results.push(ServiceStatus {
                name: name.to_string(),
                status: status.to_string(),
                desc: desc.to_string(),
            });
        }
        Ok(results)
    }

    /// En Ã§ok kaynak tÃ¼keten sÃ¼reÃ§leri dÃ¶ndÃ¼rÃ¼r
    pub fn get_processes() -> Result<Vec<ProcessInfo>, String> {
        if Self::is_dev_mode() {
            return Ok(vec![
                ProcessInfo { pid: 1, user: "root".into(), cpu: 0.1, mem: 0.3, command: "/sbin/init".into() },
                ProcessInfo { pid: 1842, user: "mysql".into(), cpu: 8.5, mem: 12.4, command: "/usr/sbin/mariadbd".into() },
                ProcessInfo { pid: 2103, user: "postgres".into(), cpu: 3.2, mem: 5.1, command: "postgres: writer process".into() },
                ProcessInfo { pid: 3456, user: "nobody".into(), cpu: 45.2, mem: 8.7, command: "lshttpd (openlitespeed)".into() },
                ProcessInfo { pid: 4521, user: "www-data".into(), cpu: 12.1, mem: 4.3, command: "php-fpm: pool www".into() },
                ProcessInfo { pid: 5678, user: "root".into(), cpu: 0.5, mem: 1.2, command: "fail2ban-server".into() },
            ]);
        }

        let output = Command::new("ps")
            .args(&["aux", "--sort=-%cpu"])
            .output()
            .map_err(|e| format!("ps Ã§alÄ±ÅŸtÄ±rÄ±lamadÄ±: {}", e))?;

        let stdout = String::from_utf8_lossy(&output.stdout);
        let mut procs = Vec::new();
        for (i, line) in stdout.lines().enumerate() {
            if i == 0 || i > 20 { continue; } // skip header, limit to top 20
            let parts: Vec<&str> = line.split_whitespace().collect();
            if parts.len() >= 11 {
                procs.push(ProcessInfo {
                    pid: parts[1].parse().unwrap_or(0),
                    user: parts[0].to_string(),
                    cpu: parts[2].parse().unwrap_or(0.0),
                    mem: parts[3].parse().unwrap_or(0.0),
                    command: parts[10..].join(" "),
                });
            }
        }
        Ok(procs)
    }

    /// Servis baÅŸlat/durdur/yeniden baÅŸlat
    pub fn control_service(name: &str, action: &str) -> Result<String, String> {
        let normalized_action = action.trim().to_ascii_lowercase();
        let normalized_name = name.trim().to_string();

        if normalized_action == "kill" {
            if normalized_name.is_empty() {
                return Err("process id is required for kill action.".to_string());
            }

            if Self::is_dev_mode() {
                return Ok(format!("(Dev Mode) process {} killed.", normalized_name));
            }

            let output = Command::new("kill")
                .args(&["-9", &normalized_name])
                .output()
                .map_err(|e| format!("kill calistirilamadi: {}", e))?;

            if output.status.success() {
                return Ok(format!("Process {} killed.", normalized_name));
            }

            return Err(String::from_utf8_lossy(&output.stderr).to_string());
        }

        if Self::is_dev_mode() {
            return Ok(format!("(Dev Mode) {} servisi {} edildi.", normalized_name, normalized_action));
        }

        let output = Command::new("systemctl")
            .args(&[&normalized_action, &normalized_name])
            .output()
            .map_err(|e| format!("systemctl {} calistirilamadi: {}", normalized_action, e))?;

        if output.status.success() {
            Ok(format!("{} servisi {} edildi.", normalized_name, normalized_action))
        } else {
            Err(String::from_utf8_lossy(&output.stderr).to_string())
        }
    }

    // Helpers
    fn parse_meminfo(content: &str, key: &str) -> u64 {
        content.lines()
            .find(|l| l.starts_with(key))
            .and_then(|l| l.split_whitespace().nth(1))
            .and_then(|v| v.parse::<u64>().ok())
            .unwrap_or(0)
    }

    fn parse_df(output: &str) -> (String, String, f64) {
        if let Some(line) = output.lines().nth(1) {
            let parts: Vec<&str> = line.split_whitespace().collect();
            if parts.len() >= 5 {
                let total = parts[1].parse::<u64>().unwrap_or(0);
                let used = parts[2].parse::<u64>().unwrap_or(0);
                let pct = parts[4].trim_end_matches('%').parse::<f64>().unwrap_or(0.0);
                return (Self::format_bytes(total), Self::format_bytes(used), pct);
            }
        }
        ("0 B".to_string(), "0 B".to_string(), 0.0)
    }

    fn format_bytes(bytes: u64) -> String {
        const KB: u64 = 1024;
        const MB: u64 = KB * 1024;
        const GB: u64 = MB * 1024;
        const TB: u64 = GB * 1024;

        if bytes >= TB { format!("{:.1} TB", bytes as f64 / TB as f64) }
        else if bytes >= GB { format!("{:.1} GB", bytes as f64 / GB as f64) }
        else if bytes >= MB { format!("{:.1} MB", bytes as f64 / MB as f64) }
        else if bytes >= KB { format!("{:.1} KB", bytes as f64 / KB as f64) }
        else { format!("{} B", bytes) }
    }
}

