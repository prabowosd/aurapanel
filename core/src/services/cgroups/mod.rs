use std::process::Command;

pub struct CGroupManager;

impl CGroupManager {
    pub fn apply_limits(
        user: &str,
        cpu_limit: Option<u32>,
        ram_mb: Option<u32>,
        io_limit: Option<u32>,
    ) -> Result<(), String> {
        if cfg!(windows) {
            // systemd/cgroups are Linux-only.
            return Ok(());
        }

        let username = user.trim();
        if username.is_empty() {
            return Err("Kullanici adi zorunludur.".to_string());
        }

        let mut properties: Vec<String> = Vec::new();
        if let Some(cpu) = cpu_limit.filter(|v| *v > 0) {
            properties.push(format!("CPUQuota={}%", cpu));
        }
        if let Some(ram) = ram_mb.filter(|v| *v > 0) {
            properties.push(format!("MemoryMax={}M", ram));
        }
        if let Some(io) = io_limit.filter(|v| *v > 0) {
            // Systemd IO bandwidth properties require a block device path.
            // /dev/sda is a pragmatic default and can be overridden in future.
            properties.push(format!("IOReadBandwidthMax=/dev/sda {}M", io));
            properties.push(format!("IOWriteBandwidthMax=/dev/sda {}M", io));
        }

        if properties.is_empty() {
            return Ok(());
        }

        let slice_name = format!("user-{}.slice", username);
        let output = Command::new("systemctl")
            .arg("set-property")
            .arg(&slice_name)
            .args(&properties)
            .output()
            .map_err(|e| format!("systemctl calistirilamadi: {}", e))?;

        if output.status.success() {
            Ok(())
        } else {
            Err(String::from_utf8_lossy(&output.stderr).trim().to_string())
        }
    }
}
