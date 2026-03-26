#!/usr/bin/env bash
# AuraPanel Installation Script
# Supported OS: Ubuntu 22.04/24.04, Debian 12+, AlmaLinux 8/9, Rocky Linux 8/9
# Usage: sudo bash install.sh

set -euo pipefail

GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m'

PROJECT_DIR="/opt/aurapanel"
GATEWAY_ENV_DIR="/etc/aurapanel"
GATEWAY_ENV_FILE="${GATEWAY_ENV_DIR}/aurapanel.env"
CORE_ENV_FILE="${GATEWAY_ENV_DIR}/aurapanel-core.env"
REPO_URL="https://github.com/mkoyazilim/aurapanel.git"

log() {
  echo -e "${BLUE}[$(date '+%H:%M:%S')]${NC} $*"
}

ok() {
  echo -e "${GREEN}$*${NC}"
}

fail() {
  echo -e "${RED}$*${NC}"
  exit 1
}

if [ "${EUID}" -ne 0 ]; then
  fail "Please run as root."
fi

echo -e "${BLUE}=================================================${NC}"
echo -e "${GREEN} AuraPanel - Installation Script ${NC}"
echo -e "${BLUE}=================================================${NC}"

if [ -f /etc/os-release ]; then
  . /etc/os-release
  OS_ID="${ID}"
else
  fail "Unsupported OS: /etc/os-release not found."
fi

log "Installing system prerequisites..."
case "${OS_ID}" in
  ubuntu|debian)
    apt-get update -y
    apt-get install -y curl wget git rsync build-essential cmake pkg-config libssl-dev gcc ufw ca-certificates openssl
    ;;
  almalinux|rocky|centos)
    dnf update -y
    dnf groupinstall -y "Development Tools"
    dnf install -y curl wget git rsync cmake openssl-devel openssl gcc firewalld ca-certificates
    ;;
  *)
    fail "Unsupported OS: ${OS_ID}."
    ;;
esac

log "Ensuring Rust toolchain..."
if ! command -v cargo >/dev/null 2>&1; then
  curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y
  # shellcheck disable=SC1090
  source "${HOME}/.cargo/env"
fi
if [ -f "${HOME}/.cargo/env" ]; then
  # shellcheck disable=SC1090
  source "${HOME}/.cargo/env"
fi

log "Ensuring Go toolchain..."
if ! command -v go >/dev/null 2>&1; then
  GO_TARBALL="go1.22.1.linux-amd64.tar.gz"
  wget -q "https://go.dev/dl/${GO_TARBALL}" -O "/tmp/${GO_TARBALL}"
  rm -rf /usr/local/go
  tar -C /usr/local -xzf "/tmp/${GO_TARBALL}"
  rm -f "/tmp/${GO_TARBALL}"
fi
export PATH="$PATH:/usr/local/go/bin"

log "Preparing project directory at ${PROJECT_DIR}..."
mkdir -p "${PROJECT_DIR}"

if [ -d "$(pwd)/core" ] && [ -d "$(pwd)/api-gateway" ] && [ -d "$(pwd)/frontend" ]; then
  log "Copying current workspace into ${PROJECT_DIR}..."
  rsync -a --delete \
    --exclude '.git' \
    --exclude 'core/target' \
    --exclude 'frontend/node_modules' \
    --exclude 'api-gateway/apigw' \
    "$(pwd)/" "${PROJECT_DIR}/"
else
  if [ ! -d "${PROJECT_DIR}/.git" ]; then
    log "Cloning repository from ${REPO_URL}..."
    rm -rf "${PROJECT_DIR}"
    git clone "${REPO_URL}" "${PROJECT_DIR}"
  else
    log "Updating existing repository..."
    git -C "${PROJECT_DIR}" fetch --all
    git -C "${PROJECT_DIR}" pull --ff-only
  fi
fi

log "Building Rust core..."
cd "${PROJECT_DIR}/core"
cargo build --release

log "Building Go API gateway..."
cd "${PROJECT_DIR}/api-gateway"
/usr/local/go/bin/go mod tidy
/usr/local/go/bin/go build -o apigw .

log "Preparing gateway environment..."
mkdir -p "${GATEWAY_ENV_DIR}" "${PROJECT_DIR}/logs"
chmod 700 "${GATEWAY_ENV_DIR}"

if [ ! -f "${GATEWAY_ENV_FILE}" ]; then
  ADMIN_PASS="$(openssl rand -base64 18 | tr -d '\n')"
  JWT_SECRET="$(openssl rand -hex 32 | tr -d '\n')"

  cat <<EOF > "${GATEWAY_ENV_FILE}"
AURAPANEL_ADMIN_EMAIL=admin@server.com
AURAPANEL_ADMIN_PASSWORD=${ADMIN_PASS}
AURAPANEL_JWT_SECRET=${JWT_SECRET}
AURAPANEL_JWT_ISSUER=aurapanel-gateway
AURAPANEL_JWT_AUDIENCE=aurapanel-ui
AURAPANEL_ALLOWED_ORIGINS=http://127.0.0.1:5173,http://localhost:5173
AURAPANEL_CORE_URL=http://127.0.0.1:8000
AURAPANEL_GATEWAY_ONLY=1
EOF

  chmod 600 "${GATEWAY_ENV_FILE}"
  echo "${ADMIN_PASS}" > "${PROJECT_DIR}/logs/initial_password.txt"
  chmod 600 "${PROJECT_DIR}/logs/initial_password.txt"
  ok "Initial admin password written to ${PROJECT_DIR}/logs/initial_password.txt"
else
  ok "Using existing gateway env file: ${GATEWAY_ENV_FILE}"
fi

if [ ! -f "${CORE_ENV_FILE}" ]; then
  RESTIC_PASS="$(openssl rand -hex 24 | tr -d '\n')"
  MINIO_ACCESS="backup$(openssl rand -hex 3 | tr -d '\n')"
  MINIO_SECRET="$(openssl rand -hex 24 | tr -d '\n')"

  cat <<EOF > "${CORE_ENV_FILE}"
AURAPANEL_RUNTIME_MODE=production
AURAPANEL_SECURITY_POLICY=fail-closed
AURAPANEL_GATEWAY_ONLY=1
AURAPANEL_CORE_BIND_ADDR=127.0.0.1:8000
AURAPANEL_FEDERATION_MODE=active-passive
AURAPANEL_FEDERATION_PRIMARY=1
AURAPANEL_BACKUP_TARGET=internal-minio
AURAPANEL_BACKUP_MINIO_ENDPOINT=http://127.0.0.1:9000
AURAPANEL_BACKUP_MINIO_BUCKET=aurapanel-backups
AURAPANEL_BACKUP_MINIO_ACCESS_KEY=${MINIO_ACCESS}
AURAPANEL_BACKUP_MINIO_SECRET_KEY=${MINIO_SECRET}
AURAPANEL_BACKUP_RESTIC_PASSWORD=${RESTIC_PASS}
EOF

  chmod 600 "${CORE_ENV_FILE}"
  ok "Core policy env file created: ${CORE_ENV_FILE}"
else
  ok "Using existing core env file: ${CORE_ENV_FILE}"
fi

log "Configuring systemd services..."
cat <<EOF > /etc/systemd/system/aurapanel-core.service
[Unit]
Description=AuraPanel Core (Rust)
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=${PROJECT_DIR}/core
ExecStart=${PROJECT_DIR}/core/target/release/aurapanel-core
Restart=on-failure
Environment=RUST_LOG=info
EnvironmentFile=-${CORE_ENV_FILE}

[Install]
WantedBy=multi-user.target
EOF

cat <<EOF > /etc/systemd/system/aurapanel-api.service
[Unit]
Description=AuraPanel API Gateway (Go)
After=network.target aurapanel-core.service
Requires=aurapanel-core.service

[Service]
Type=simple
User=root
WorkingDirectory=${PROJECT_DIR}/api-gateway
ExecStart=${PROJECT_DIR}/api-gateway/apigw
Restart=on-failure
EnvironmentFile=-${GATEWAY_ENV_FILE}

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable aurapanel-core aurapanel-api
systemctl restart aurapanel-core aurapanel-api

ok "AuraPanel services are enabled and running."
ok "API Gateway: http://YOUR_SERVER_IP:8080"
ok "Core API: http://127.0.0.1:8000/api/v1/health"
