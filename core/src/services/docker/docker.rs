use serde::{Deserialize, Serialize};
use std::process::Command;

#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct ContainerInfo {
    pub id: String,
    pub name: String,
    pub image: String,
    pub status: String,
    pub ports: String,
    pub created: String,
}

#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct ImageInfo {
    pub id: String,
    pub repository: String,
    pub tag: String,
    pub size: String,
    pub created: String,
}

#[derive(Serialize, Deserialize, Debug)]
pub struct CreateContainerConfig {
    pub name: String,
    pub image: String,
    pub ports: Vec<String>,
    pub env: Vec<String>,
    pub volumes: Vec<String>,
    pub restart_policy: Option<String>,
    pub memory_limit: Option<String>,
    pub cpu_limit: Option<String>,
}

#[derive(Serialize, Deserialize, Debug)]
pub struct PullImageConfig {
    pub image: String,
    pub tag: Option<String>,
}

pub struct DockerManager;

impl DockerManager {
    fn simulation_enabled() -> bool {
        crate::runtime::simulation_enabled()
    }

    fn docker_unavailable_error() -> String {
        "docker is not available on this host. Install docker or enable AURAPANEL_DEV_SIMULATION=1.".to_string()
    }

    pub fn list_containers() -> Result<Vec<ContainerInfo>, String> {
        if !Self::is_docker_available() {
            if !Self::simulation_enabled() {
                return Err(Self::docker_unavailable_error());
            }

            return Ok(vec![
                ContainerInfo {
                    id: "abc123def456".into(),
                    name: "nginx-proxy".into(),
                    image: "nginx:latest".into(),
                    status: "Up 3 days".into(),
                    ports: "0.0.0.0:80->80/tcp".into(),
                    created: "3 days ago".into(),
                },
                ContainerInfo {
                    id: "789ghi012jkl".into(),
                    name: "mysql-db".into(),
                    image: "mariadb:11".into(),
                    status: "Up 3 days".into(),
                    ports: "3306/tcp".into(),
                    created: "3 days ago".into(),
                },
                ContainerInfo {
                    id: "mno345pqr678".into(),
                    name: "redis-cache".into(),
                    image: "redis:7-alpine".into(),
                    status: "Exited (0) 2 hours ago".into(),
                    ports: "6379/tcp".into(),
                    created: "5 days ago".into(),
                },
            ]);
        }

        let output = Command::new("docker")
            .args(["ps", "-a", "--format", "{{.ID}}|{{.Names}}|{{.Image}}|{{.Status}}|{{.Ports}}|{{.CreatedAt}}"])
            .output()
            .map_err(|e| format!("docker ps failed: {}", e))?;

        let stdout = String::from_utf8_lossy(&output.stdout);
        let containers = stdout
            .lines()
            .filter(|line| !line.is_empty())
            .map(|line| {
                let parts: Vec<&str> = line.splitn(6, '|').collect();
                ContainerInfo {
                    id: parts.first().unwrap_or(&"").to_string(),
                    name: parts.get(1).unwrap_or(&"").to_string(),
                    image: parts.get(2).unwrap_or(&"").to_string(),
                    status: parts.get(3).unwrap_or(&"").to_string(),
                    ports: parts.get(4).unwrap_or(&"").to_string(),
                    created: parts.get(5).unwrap_or(&"").to_string(),
                }
            })
            .collect();

        Ok(containers)
    }

    pub fn create_container(config: &CreateContainerConfig) -> Result<String, String> {
        println!("[DOCKER] Creating container: {} from image: {}", config.name, config.image);

        if !Self::is_docker_available() {
            if Self::simulation_enabled() {
                println!("[DEV MODE] Docker not available. Simulating container creation.");
                return Ok(format!("simulated-container-id-{}", config.name));
            }
            return Err(Self::docker_unavailable_error());
        }

        let mut args = vec!["run".to_string(), "-d".to_string(), "--name".to_string(), config.name.clone()];

        for port in &config.ports {
            args.push("-p".to_string());
            args.push(port.clone());
        }

        for env in &config.env {
            args.push("-e".to_string());
            args.push(env.clone());
        }

        for volume in &config.volumes {
            args.push("-v".to_string());
            args.push(volume.clone());
        }

        if let Some(policy) = &config.restart_policy {
            args.push("--restart".to_string());
            args.push(policy.clone());
        }

        if let Some(memory) = &config.memory_limit {
            args.push("--memory".to_string());
            args.push(memory.clone());
        }

        if let Some(cpu) = &config.cpu_limit {
            args.push("--cpus".to_string());
            args.push(cpu.clone());
        }

        args.push(config.image.clone());

        let output = Command::new("docker")
            .args(&args)
            .output()
            .map_err(|e| format!("docker run failed: {}", e))?;

        if !output.status.success() {
            return Err(format!(
                "container create failed: {}",
                String::from_utf8_lossy(&output.stderr)
            ));
        }

        Ok(String::from_utf8_lossy(&output.stdout).trim().to_string())
    }

    pub fn start_container(id: &str) -> Result<(), String> {
        Self::docker_cmd(&["start", id])
    }

    pub fn stop_container(id: &str) -> Result<(), String> {
        Self::docker_cmd(&["stop", id])
    }

    pub fn restart_container(id: &str) -> Result<(), String> {
        Self::docker_cmd(&["restart", id])
    }

    pub fn remove_container(id: &str, force: bool) -> Result<(), String> {
        if force {
            Self::docker_cmd(&["rm", "-f", id])
        } else {
            Self::docker_cmd(&["rm", id])
        }
    }

    pub fn container_logs(id: &str, tail: u32) -> Result<String, String> {
        if !Self::is_docker_available() {
            if Self::simulation_enabled() {
                return Ok(format!("[DEV MODE] Simulated logs for container {}", id));
            }
            return Err(Self::docker_unavailable_error());
        }

        let output = Command::new("docker")
            .args(["logs", "--tail", &tail.to_string(), id])
            .output()
            .map_err(|e| format!("docker logs failed: {}", e))?;

        Ok(String::from_utf8_lossy(&output.stdout).to_string()
            + &String::from_utf8_lossy(&output.stderr).to_string())
    }

    pub fn list_images() -> Result<Vec<ImageInfo>, String> {
        if !Self::is_docker_available() {
            if !Self::simulation_enabled() {
                return Err(Self::docker_unavailable_error());
            }

            return Ok(vec![
                ImageInfo {
                    id: "sha256:abc123".into(),
                    repository: "nginx".into(),
                    tag: "latest".into(),
                    size: "187MB".into(),
                    created: "2 weeks ago".into(),
                },
                ImageInfo {
                    id: "sha256:def456".into(),
                    repository: "mariadb".into(),
                    tag: "11".into(),
                    size: "405MB".into(),
                    created: "3 weeks ago".into(),
                },
                ImageInfo {
                    id: "sha256:ghi789".into(),
                    repository: "redis".into(),
                    tag: "7-alpine".into(),
                    size: "32MB".into(),
                    created: "1 month ago".into(),
                },
            ]);
        }

        let output = Command::new("docker")
            .args(["images", "--format", "{{.ID}}|{{.Repository}}|{{.Tag}}|{{.Size}}|{{.CreatedAt}}"])
            .output()
            .map_err(|e| format!("docker images failed: {}", e))?;

        let stdout = String::from_utf8_lossy(&output.stdout);
        let images = stdout
            .lines()
            .filter(|line| !line.is_empty())
            .map(|line| {
                let parts: Vec<&str> = line.splitn(5, '|').collect();
                ImageInfo {
                    id: parts.first().unwrap_or(&"").to_string(),
                    repository: parts.get(1).unwrap_or(&"").to_string(),
                    tag: parts.get(2).unwrap_or(&"").to_string(),
                    size: parts.get(3).unwrap_or(&"").to_string(),
                    created: parts.get(4).unwrap_or(&"").to_string(),
                }
            })
            .collect();

        Ok(images)
    }

    pub fn pull_image(config: &PullImageConfig) -> Result<(), String> {
        let full_image = match &config.tag {
            Some(tag) => format!("{}:{}", config.image, tag),
            None => format!("{}:latest", config.image),
        };

        println!("[DOCKER] Pulling image: {}", full_image);

        if !Self::is_docker_available() {
            if Self::simulation_enabled() {
                println!("[DEV MODE] Docker pull simulated for {}", full_image);
                return Ok(());
            }
            return Err(Self::docker_unavailable_error());
        }

        let output = Command::new("docker")
            .args(["pull", &full_image])
            .output()
            .map_err(|e| format!("docker pull failed: {}", e))?;

        if !output.status.success() {
            return Err(format!("image pull failed: {}", String::from_utf8_lossy(&output.stderr)));
        }

        Ok(())
    }

    pub fn remove_image(id: &str, force: bool) -> Result<(), String> {
        if force {
            Self::docker_cmd(&["rmi", "-f", id])
        } else {
            Self::docker_cmd(&["rmi", id])
        }
    }

    fn is_docker_available() -> bool {
        Command::new("docker")
            .arg("--version")
            .output()
            .map(|o| o.status.success())
            .unwrap_or(false)
    }

    fn docker_cmd(args: &[&str]) -> Result<(), String> {
        if !Self::is_docker_available() {
            if Self::simulation_enabled() {
                println!("[DEV MODE] docker {} simulated.", args.join(" "));
                return Ok(());
            }
            return Err(Self::docker_unavailable_error());
        }

        let output = Command::new("docker")
            .args(args)
            .output()
            .map_err(|e| format!("docker command failed: {}", e))?;

        if !output.status.success() {
            return Err(String::from_utf8_lossy(&output.stderr).to_string());
        }

        Ok(())
    }
}
