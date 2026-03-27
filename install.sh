#!/usr/bin/env bash
# AuraPanel bootstrap wrapper
# Usage: curl -sSL https://raw.githubusercontent.com/mkoyazilim/aurapanel/main/install.sh | bash

set -euo pipefail

download_file() {
  local url="$1"
  local output="$2"

  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$url" -o "$output"
    return 0
  fi

  if command -v wget >/dev/null 2>&1; then
    wget -q "$url" -O "$output"
    return 0
  fi

  return 1
}

cleanup() {
  rm -f aurapanel.sh aurapanel_bootstrap.sh
}

trap cleanup EXIT

AURAPANEL_GH_OWNER="${AURAPANEL_GH_OWNER:-mkoyazilim}"
AURAPANEL_GH_REPO="${AURAPANEL_GH_REPO:-aurapanel}"
AURAPANEL_GH_REF="${AURAPANEL_GH_REF:-main}"
RAW_BASE_URL="https://raw.githubusercontent.com/${AURAPANEL_GH_OWNER}/${AURAPANEL_GH_REPO}/${AURAPANEL_GH_REF}"

INSTALLER_BASE_URL="${AURAPANEL_INSTALLER_BASE:-${RAW_BASE_URL}/installer}"

if [ -n "${AURAPANEL_MANIFEST_URL:-}" ] || [ -n "${AURAPANEL_RELEASE_BASE:-}" ] || [ -n "${AURAPANEL_BOOTSTRAP_URL:-}" ]; then
  if [ -n "${AURAPANEL_BOOTSTRAP_URL:-}" ]; then
    BOOTSTRAP_URL="$AURAPANEL_BOOTSTRAP_URL"
  elif [ -n "${AURAPANEL_RELEASE_BASE:-}" ]; then
    BOOTSTRAP_URL="${AURAPANEL_RELEASE_BASE%/}/aurapanel_bootstrap.sh"
  else
    BOOTSTRAP_URL="${RAW_BASE_URL}/aurapanel_bootstrap.sh"
  fi

  download_file "$BOOTSTRAP_URL" "aurapanel_bootstrap.sh" || {
    echo "Unable to download aurapanel_bootstrap.sh from: $BOOTSTRAP_URL"
    exit 1
  }

  chmod +x aurapanel_bootstrap.sh
  ./aurapanel_bootstrap.sh "$@"
  exit $?
fi

MAIN_INSTALLER_URL="${AURAPANEL_MAIN_INSTALLER_URL:-${INSTALLER_BASE_URL}/aurapanel.sh}"
download_file "$MAIN_INSTALLER_URL" "aurapanel.sh" || {
  echo "Unable to download aurapanel.sh from: $MAIN_INSTALLER_URL"
  exit 1
}

chmod +x aurapanel.sh
./aurapanel.sh "$@"
