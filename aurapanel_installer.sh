#!/usr/bin/env bash
# Entrypoint used by verified release bundle

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

if [ -x "$SCRIPT_DIR/installer/aurapanel.sh" ]; then
  exec "$SCRIPT_DIR/installer/aurapanel.sh" "$@"
fi

if [ -x "$SCRIPT_DIR/aurapanel.sh" ]; then
  exec "$SCRIPT_DIR/aurapanel.sh" "$@"
fi

echo "[aurapanel-installer] Missing aurapanel.sh in verified bundle" >&2
exit 1
