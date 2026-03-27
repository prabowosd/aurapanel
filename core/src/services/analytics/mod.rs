use chrono::{DateTime, Datelike, Duration, FixedOffset, Timelike, Utc};
use serde::{Deserialize, Serialize};
use std::collections::{BTreeMap, HashMap, HashSet};
use std::fs::File;
use std::io::{BufRead, BufReader};
use std::path::{Path, PathBuf};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TrafficQuery {
    pub domain: String,
    pub hours: Option<u32>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TrafficResponse {
    pub domain: String,
    pub range_hours: u32,
    pub source_log: String,
    pub totals: TrafficTotals,
    pub series: Vec<TrafficBucket>,
    pub top_paths: Vec<PathStat>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TrafficTotals {
    pub hits: u64,
    pub visitors: usize,
    pub bandwidth_bytes: u64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TrafficBucket {
    pub bucket: String,
    pub hits: u64,
    pub visitors: usize,
    pub bandwidth_bytes: u64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PathStat {
    pub path: String,
    pub hits: u64,
    pub bandwidth_bytes: u64,
}

#[derive(Debug, Clone)]
struct AccessRecord {
    ip: String,
    timestamp: DateTime<Utc>,
    path: String,
    bytes: u64,
}

pub struct AnalyticsManager;

impl AnalyticsManager {
    pub fn website_traffic(query: &TrafficQuery) -> Result<TrafficResponse, String> {
        let domain = query.domain.trim().to_ascii_lowercase();
        if domain.is_empty() {
            return Err("Domain zorunludur.".to_string());
        }
        let hours = query.hours.unwrap_or(24).clamp(1, 24 * 14);
        let cutoff = Utc::now() - Duration::hours(i64::from(hours));

        let log_path = locate_access_log(&domain)
            .ok_or_else(|| format!("Access log bulunamadi (domain: {}).", domain))?;
        let file = File::open(&log_path).map_err(|e| format!("Access log acilamadi: {}", e))?;
        let reader = BufReader::new(file);

        let mut total_hits = 0u64;
        let mut total_bandwidth = 0u64;
        let mut unique_visitors = HashSet::new();
        let mut bucket_hits: BTreeMap<String, u64> = BTreeMap::new();
        let mut bucket_bandwidth: BTreeMap<String, u64> = BTreeMap::new();
        let mut bucket_visitors: HashMap<String, HashSet<String>> = HashMap::new();
        let mut paths: HashMap<String, (u64, u64)> = HashMap::new();

        for line in reader.lines().map_while(Result::ok) {
            if let Some(record) = parse_access_log_line(&line) {
                if record.timestamp < cutoff {
                    continue;
                }
                total_hits = total_hits.saturating_add(1);
                total_bandwidth = total_bandwidth.saturating_add(record.bytes);
                unique_visitors.insert(record.ip.clone());

                let bucket = hourly_bucket(record.timestamp);
                *bucket_hits.entry(bucket.clone()).or_insert(0) += 1;
                *bucket_bandwidth.entry(bucket.clone()).or_insert(0) += record.bytes;
                bucket_visitors
                    .entry(bucket.clone())
                    .or_default()
                    .insert(record.ip);

                let path_entry = paths.entry(record.path).or_insert((0, 0));
                path_entry.0 = path_entry.0.saturating_add(1);
                path_entry.1 = path_entry.1.saturating_add(record.bytes);
            }
        }

        let mut series = Vec::new();
        for (bucket, hits) in bucket_hits {
            let visitors = bucket_visitors
                .get(&bucket)
                .map(|set| set.len())
                .unwrap_or_default();
            let bandwidth_bytes = *bucket_bandwidth.get(&bucket).unwrap_or(&0);
            series.push(TrafficBucket {
                bucket,
                hits,
                visitors,
                bandwidth_bytes,
            });
        }

        let mut top_paths = paths
            .into_iter()
            .map(|(path, (hits, bandwidth_bytes))| PathStat {
                path,
                hits,
                bandwidth_bytes,
            })
            .collect::<Vec<_>>();
        top_paths.sort_by(|a, b| {
            b.hits
                .cmp(&a.hits)
                .then(b.bandwidth_bytes.cmp(&a.bandwidth_bytes))
        });
        top_paths.truncate(12);

        Ok(TrafficResponse {
            domain,
            range_hours: hours,
            source_log: log_path.to_string_lossy().to_string(),
            totals: TrafficTotals {
                hits: total_hits,
                visitors: unique_visitors.len(),
                bandwidth_bytes: total_bandwidth,
            },
            series,
            top_paths,
        })
    }
}

fn hourly_bucket(ts: DateTime<Utc>) -> String {
    format!(
        "{:04}-{:02}-{:02} {:02}:00",
        ts.year(),
        ts.month(),
        ts.day(),
        ts.hour()
    )
}

fn locate_access_log(domain: &str) -> Option<PathBuf> {
    let candidates = vec![
        format!("/usr/local/lsws/logs/{}.access.log", domain),
        format!("/usr/local/lsws/logs/{}/access.log", domain),
        format!("/home/{}/logs/access.log", domain),
        format!("/var/log/nginx/{}.access.log", domain),
        format!("/var/log/nginx/{}.log", domain),
    ];

    for candidate in candidates {
        let path = PathBuf::from(&candidate);
        if path.exists() && path.is_file() {
            return Some(path);
        }
    }

    let fallback_dir = Path::new("/usr/local/lsws/logs");
    if fallback_dir.exists() {
        if let Ok(entries) = std::fs::read_dir(fallback_dir) {
            for entry in entries.flatten() {
                let path = entry.path();
                let name = path
                    .file_name()
                    .and_then(|v| v.to_str())
                    .unwrap_or_default()
                    .to_ascii_lowercase();
                if name.contains(domain) && name.contains("access") && path.is_file() {
                    return Some(path);
                }
            }
        }
    }
    None
}

fn parse_access_log_line(line: &str) -> Option<AccessRecord> {
    let mut parts = line.split_whitespace();
    let ip = parts.next()?.to_string();

    let ts_start = line.find('[')?;
    let ts_end = line[ts_start + 1..].find(']')? + ts_start + 1;
    let ts_raw = &line[ts_start + 1..ts_end];
    let ts = DateTime::parse_from_str(ts_raw, "%d/%b/%Y:%H:%M:%S %z")
        .ok()?
        .with_timezone(&Utc);

    let req_start = line[ts_end + 1..].find('"')? + ts_end + 1;
    let req_end = line[req_start + 1..].find('"')? + req_start + 1;
    let req_raw = &line[req_start + 1..req_end];
    let req_parts: Vec<&str> = req_raw.split_whitespace().collect();
    let path = req_parts.get(1).unwrap_or(&"/").to_string();

    let tail = line[req_end + 1..].trim();
    let tail_parts: Vec<&str> = tail.split_whitespace().collect();
    let bytes = tail_parts
        .get(1)
        .and_then(|value| value.parse::<u64>().ok())
        .unwrap_or(0);

    Some(AccessRecord {
        ip,
        timestamp: ts,
        path,
        bytes,
    })
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn parse_common_log_line_works() {
        let line = r#"203.0.113.5 - - [27/Mar/2026:07:12:01 +0000] "GET /index.php HTTP/1.1" 200 512 "-" "curl/8.6.0""#;
        let parsed = parse_access_log_line(line).expect("should parse");
        assert_eq!(parsed.ip, "203.0.113.5");
        assert_eq!(parsed.path, "/index.php");
        assert_eq!(parsed.bytes, 512);
    }
}
