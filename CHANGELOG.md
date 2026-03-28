# Changelog

## 2026-03-28

### Changed
- Terminal websocket upgrades now bypass persistence wrapping and keep `http.Hijacker` support intact through the panel-service request chain.
- Website provisioning now snapshots and restores runtime state on failure, preventing orphaned owners and stale site counters.
- Website advanced-config reads no longer mutate state under a read lock.
- OpenLiteSpeed tuning now stages changes on save and only reloads the runtime on explicit apply.
- Redis topology is now separated between host-level systemd isolations and Docker templates, with host Redis inventory exposed in Ops Center and Docker templates carrying explicit runtime metadata and default ports/volumes.

## 2026-03-26

### Added
- Website lifecycle parity updates including `/vhost/update` and advanced website controls.
- DNS reconciliation and DNSSEC workflow endpoints.
- Guided website/database flow endpoints.
- Mail operational endpoints for forwards, catch-all, routing, DKIM, and webmail SSO.
- SFTP parity endpoints (`list`, `delete`, `password reset`) and state-backed management.
- Backup restore endpoint and hardened storage/minio flow.
- Core route/service tests, gateway auth/proxy contract tests, frontend vitest smoke tests.
- CI workflow with build/test/security gates.
- API v1 contract freeze document: `docs/api_contract_v1.md`.

### Changed
- Frontend auth storage strategy hardened with session-default and optional persistent remember mode.
- eBPF event pipeline moved from static list to collector-backed flow.
- Server status frontend aligned with backend payload contract.
- CTO architecture decisions codified:
  - gateway-only production API entry with core loopback bind enforcement
  - active-passive federated topology defaults
  - internal MinIO as default backup target
  - fail-closed security policy enforcement in production startup

### Security
- Added CI secret scanning and dependency/static analysis gates.
