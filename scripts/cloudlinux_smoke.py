#!/usr/bin/env python3
"""AuraPanel CloudLinux staging smoke runner.

Runs a controlled endpoint chain against a live AuraPanel API:
1) cloudlinux/status
2) cloudlinux/actions
3) cloudlinux/profiles
4) cloudlinux/rollout/plan
5) cloudlinux/rollout/apply (dry-run)
6) cloudlinux/rollout/history verification
7) optional cloudlinux/rollout/apply (live, gated)
"""

from __future__ import annotations

import argparse
import datetime as dt
import json
import os
import sys
import traceback
import urllib.error
import urllib.parse
import urllib.request
from typing import Any, Dict, List, Optional


def utc_now() -> str:
    return dt.datetime.now(dt.timezone.utc).isoformat()


def bool_env(name: str, default: bool = False) -> bool:
    raw = os.getenv(name, "").strip().lower()
    if raw == "":
        return default
    return raw in {"1", "true", "yes", "on"}


def first_non_empty(*values: str) -> str:
    for value in values:
        if value and value.strip():
            return value.strip()
    return ""


class ApiError(RuntimeError):
    pass


class AuraApiClient:
    def __init__(self, base_url: str, token: str = "") -> None:
        self.base_url = base_url.rstrip("/")
        self.token = token.strip()

    def _url(self, path: str) -> str:
        if not path.startswith("/"):
            path = "/" + path
        return self.base_url + path

    def request(self, method: str, path: str, payload: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
        url = self._url(path)
        headers = {
            "Accept": "application/json",
        }
        data: Optional[bytes] = None
        if payload is not None:
            data = json.dumps(payload).encode("utf-8")
            headers["Content-Type"] = "application/json"
        if self.token:
            headers["Authorization"] = f"Bearer {self.token}"

        req = urllib.request.Request(url=url, method=method.upper(), headers=headers, data=data)
        try:
            with urllib.request.urlopen(req, timeout=60) as resp:
                raw = resp.read().decode("utf-8", errors="replace")
        except urllib.error.HTTPError as exc:
            body = exc.read().decode("utf-8", errors="replace")
            raise ApiError(f"{method.upper()} {path} -> HTTP {exc.code}: {body}") from exc
        except urllib.error.URLError as exc:
            raise ApiError(f"{method.upper()} {path} -> network error: {exc}") from exc

        try:
            parsed = json.loads(raw) if raw else {}
        except json.JSONDecodeError as exc:
            raise ApiError(f"{method.upper()} {path} -> invalid JSON response: {raw[:300]}") from exc
        return parsed


class SmokeRunner:
    def __init__(self, args: argparse.Namespace) -> None:
        self.args = args
        self.client = AuraApiClient(args.base_url, token=args.token)
        self.steps: List[Dict[str, Any]] = []
        self.report: Dict[str, Any] = {
            "started_at": utc_now(),
            "base_url": args.base_url,
            "enable_live": bool(args.enable_live),
            "steps": self.steps,
            "summary": {},
        }

    def log(self, msg: str) -> None:
        print(f"[cloudlinux-smoke] {msg}")

    def run_step(self, name: str, fn) -> Any:
        self.log(f"STEP -> {name}")
        step = {"name": name, "started_at": utc_now(), "ok": False, "details": {}}
        self.steps.append(step)
        try:
            result = fn()
            step["ok"] = True
            step["finished_at"] = utc_now()
            if isinstance(result, dict):
                step["details"] = result
            elif result is not None:
                step["details"] = {"result": result}
            self.log(f"STEP OK -> {name}")
            return result
        except Exception as exc:  # pylint: disable=broad-except
            step["ok"] = False
            step["finished_at"] = utc_now()
            step["error"] = str(exc)
            step["trace"] = traceback.format_exc(limit=2)
            self.log(f"STEP FAIL -> {name}: {exc}")
            raise

    def ensure_success_payload(self, payload: Dict[str, Any], step_name: str) -> Dict[str, Any]:
        status = str(payload.get("status", "")).strip().lower()
        if status != "success":
            message = payload.get("message", "Unexpected status")
            raise ApiError(f"{step_name} returned status={status!r}: {message}")
        data = payload.get("data")
        if data is None:
            raise ApiError(f"{step_name} returned no data payload")
        if not isinstance(data, dict) and not isinstance(data, list):
            return {"value": data}
        return data

    def login_if_needed(self) -> None:
        if self.client.token:
            self.log("Token already provided, login step skipped.")
            return
        email = self.args.email
        password = self.args.password
        if not email or not password:
            raise ApiError("No token provided and login credentials are missing.")

        payload = self.client.request("POST", "/auth/login", {
            "email": email,
            "password": password,
        })
        status = str(payload.get("status", "")).strip().lower()
        if status != "success":
            raise ApiError(f"/auth/login failed: {payload}")

        token = str(payload.get("token", "")).strip()
        if not token:
            raise ApiError("/auth/login succeeded but token is missing in response")
        self.client.token = token

    def run(self) -> None:
        self.run_step("login", self.step_login)

        status_data = self.run_step("cloudlinux-status", self.step_status)
        self.run_step("cloudlinux-actions", self.step_actions)
        profile_data = self.run_step("cloudlinux-profiles", self.step_profiles)
        plan_data = self.run_step("cloudlinux-rollout-plan", self.step_rollout_plan)
        dry_run_data = self.run_step("cloudlinux-rollout-apply-dry", self.step_rollout_apply_dry)
        self.run_step("cloudlinux-rollout-history", lambda: self.step_rollout_history(dry_run_data))

        if self.args.enable_live:
            apply_enabled = bool((plan_data.get("summary") or {}).get("apply_enabled", False))
            if apply_enabled:
                self.run_step("cloudlinux-rollout-apply-live", self.step_rollout_apply_live)
            else:
                self.steps.append({
                    "name": "cloudlinux-rollout-apply-live",
                    "started_at": utc_now(),
                    "finished_at": utc_now(),
                    "ok": True,
                    "details": {
                        "skipped": True,
                        "reason": "apply_enabled=false",
                    },
                })
                self.log("Live apply requested but skipped because apply_enabled=false.")

        self.report["summary"] = {
            "cloudlinux_available": bool(status_data.get("available", False)),
            "profiles_count": len(profile_data.get("profiles", [])) if isinstance(profile_data, dict) else 0,
            "plan_users": len(plan_data.get("users", [])) if isinstance(plan_data, dict) else 0,
            "all_steps_ok": all(bool(step.get("ok")) for step in self.steps),
            "finished_at": utc_now(),
        }

    def step_login(self) -> Dict[str, Any]:
        self.login_if_needed()
        return {"token_present": bool(self.client.token)}

    def step_status(self) -> Dict[str, Any]:
        payload = self.client.request("GET", "/cloudlinux/status")
        data = self.ensure_success_payload(payload, "cloudlinux/status")
        if not isinstance(data, dict):
            raise ApiError("cloudlinux/status returned unexpected data format")
        required = ["available", "enabled", "features", "commands"]
        missing = [key for key in required if key not in data]
        if missing:
            raise ApiError(f"cloudlinux/status missing fields: {', '.join(missing)}")
        return {
            "available": bool(data.get("available")),
            "enabled": bool(data.get("enabled")),
            "distro": str(data.get("distro", "")),
        }

    def step_actions(self) -> Dict[str, Any]:
        payload = self.client.request("GET", "/cloudlinux/actions")
        data = self.ensure_success_payload(payload, "cloudlinux/actions")
        if not isinstance(data, dict):
            raise ApiError("cloudlinux/actions returned unexpected data format")
        actions = data.get("actions")
        history = data.get("history")
        if not isinstance(actions, list):
            raise ApiError("cloudlinux/actions data.actions is not a list")
        if not isinstance(history, list):
            raise ApiError("cloudlinux/actions data.history is not a list")
        return {"actions": len(actions), "history": len(history)}

    def step_profiles(self) -> Dict[str, Any]:
        payload = self.client.request("GET", "/cloudlinux/profiles")
        data = self.ensure_success_payload(payload, "cloudlinux/profiles")
        if not isinstance(data, dict):
            raise ApiError("cloudlinux/profiles returned unexpected data format")
        summary = data.get("summary")
        profiles = data.get("profiles")
        if not isinstance(summary, dict) or not isinstance(profiles, list):
            raise ApiError("cloudlinux/profiles payload format is invalid")
        return {
            "summary_total_packages": int(summary.get("total_packages", 0)),
            "profiles": len(profiles),
        }

    def step_rollout_plan(self) -> Dict[str, Any]:
        query_parts = []
        if self.args.package:
            query_parts.append("package=" + urllib.parse.quote(self.args.package))
        if self.args.only_ready:
            query_parts.append("only_ready=1")
        path = "/cloudlinux/rollout/plan"
        if query_parts:
            path += "?" + "&".join(query_parts)

        payload = self.client.request("GET", path)
        data = self.ensure_success_payload(payload, "cloudlinux/rollout/plan")
        if not isinstance(data, dict):
            raise ApiError("cloudlinux/rollout/plan returned unexpected data format")
        summary = data.get("summary")
        users = data.get("users")
        if not isinstance(summary, dict) or not isinstance(users, list):
            raise ApiError("cloudlinux/rollout/plan payload format is invalid")
        return {
            "summary": summary,
            "users": users,
            "script_preview_count": len(data.get("script_preview", [])) if isinstance(data.get("script_preview"), list) else 0,
        }

    def _apply_payload_base(self, dry_run: bool) -> Dict[str, Any]:
        payload: Dict[str, Any] = {
            "dry_run": bool(dry_run),
            "only_ready": bool(self.args.only_ready),
            "max_users": int(self.args.max_users),
        }
        if self.args.package:
            payload["package"] = self.args.package
        if self.args.usernames:
            payload["usernames"] = list(self.args.usernames)
        return payload

    def step_rollout_apply_dry(self) -> Dict[str, Any]:
        payload = self.client.request("POST", "/cloudlinux/rollout/apply", self._apply_payload_base(dry_run=True))
        data = self.ensure_success_payload(payload, "cloudlinux/rollout/apply dry-run")
        if not isinstance(data, dict):
            raise ApiError("cloudlinux/rollout/apply dry-run returned unexpected format")
        if not bool(data.get("dry_run", False)):
            raise ApiError("dry-run apply response returned dry_run=false")
        return {
            "id": str(data.get("id", "")),
            "planned_users": int(data.get("planned_users", 0)),
            "results": len(data.get("results", [])) if isinstance(data.get("results"), list) else 0,
        }

    def step_rollout_history(self, dry_run_data: Dict[str, Any]) -> Dict[str, Any]:
        expected_id = str(dry_run_data.get("id", ""))
        payload = self.client.request("GET", "/cloudlinux/rollout/history")
        data = self.ensure_success_payload(payload, "cloudlinux/rollout/history")
        if not isinstance(data, list):
            raise ApiError("cloudlinux/rollout/history returned non-list payload")
        if expected_id:
            if not any(str(item.get("id", "")) == expected_id for item in data if isinstance(item, dict)):
                raise ApiError(f"rollout history does not include newly created dry-run id={expected_id}")
        return {"history_rows": len(data), "contains_dry_run_id": bool(expected_id)}

    def step_rollout_apply_live(self) -> Dict[str, Any]:
        plan_payload = self.client.request("GET", "/cloudlinux/rollout/plan")
        plan_data = self.ensure_success_payload(plan_payload, "cloudlinux/rollout/plan for live apply")
        summary = plan_data.get("summary") if isinstance(plan_data, dict) else {}
        confirm_token = "APPLY_CLOUDLINUX"
        if isinstance(summary, dict):
            confirm_token = first_non_empty(str(summary.get("confirm_token", "")), confirm_token)

        payload = self._apply_payload_base(dry_run=False)
        payload["confirm"] = confirm_token

        response = self.client.request("POST", "/cloudlinux/rollout/apply", payload)
        data = self.ensure_success_payload(response, "cloudlinux/rollout/apply live")
        if not isinstance(data, dict):
            raise ApiError("cloudlinux/rollout/apply live returned unexpected format")
        if bool(data.get("dry_run", True)):
            raise ApiError("live apply response returned dry_run=true")
        return {
            "id": str(data.get("id", "")),
            "attempted_users": int(data.get("attempted_users", 0)),
            "succeeded": int(data.get("succeeded", 0)),
            "failed": int(data.get("failed", 0)),
            "confirm_token_used": confirm_token,
        }


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="AuraPanel CloudLinux staging smoke test runner")
    parser.add_argument(
        "--base-url",
        default=first_non_empty(os.getenv("AURAPANEL_API_BASE_URL", ""), "http://127.0.0.1:8090/api/v1"),
        help="API base URL (default: %(default)s)",
    )
    parser.add_argument(
        "--email",
        default=first_non_empty(os.getenv("AURAPANEL_ADMIN_EMAIL", ""), "admin@server.com"),
        help="Login email/username when token is not provided",
    )
    parser.add_argument(
        "--password",
        default=first_non_empty(os.getenv("AURAPANEL_ADMIN_PASSWORD", ""), "password123"),
        help="Login password when token is not provided",
    )
    parser.add_argument(
        "--token",
        default=first_non_empty(os.getenv("AURAPANEL_TOKEN", ""), ""),
        help="Optional bearer token; skips login when provided",
    )
    parser.add_argument(
        "--package",
        default=first_non_empty(os.getenv("CLOUDLINUX_SMOKE_PACKAGE", ""), ""),
        help="Optional package filter for rollout plan/apply",
    )
    parser.add_argument(
        "--only-ready",
        action="store_true",
        default=bool_env("CLOUDLINUX_SMOKE_ONLY_READY", True),
        help="Request rollout plan/apply with only_ready=1",
    )
    parser.add_argument(
        "--max-users",
        type=int,
        default=int(first_non_empty(os.getenv("CLOUDLINUX_SMOKE_MAX_USERS", ""), "25")),
        help="max_users for rollout apply",
    )
    parser.add_argument(
        "--usernames",
        nargs="*",
        default=[],
        help="Optional usernames for targeted rollout apply",
    )
    parser.add_argument(
        "--enable-live",
        action="store_true",
        default=bool_env("CLOUDLINUX_SMOKE_ENABLE_LIVE", False),
        help="Enable live apply step (still gated by server-side apply flag)",
    )
    parser.add_argument(
        "--report",
        default=first_non_empty(os.getenv("CLOUDLINUX_SMOKE_REPORT", ""), "plans/cloudlinux_smoke_report_latest.json"),
        help="Output JSON report path",
    )
    return parser.parse_args()


def write_report(path: str, report: Dict[str, Any]) -> None:
    target = path.strip()
    if not target:
        return
    directory = os.path.dirname(target)
    if directory:
        os.makedirs(directory, exist_ok=True)
    with open(target, "w", encoding="utf-8") as fh:
        json.dump(report, fh, indent=2, ensure_ascii=True)


def main() -> int:
    args = parse_args()
    runner = SmokeRunner(args)
    try:
        runner.run()
        runner.report["ok"] = True
        write_report(args.report, runner.report)
        runner.log(f"Smoke completed successfully. Report: {args.report}")
        return 0
    except Exception as exc:  # pylint: disable=broad-except
        runner.report["ok"] = False
        runner.report["error"] = str(exc)
        runner.report.setdefault("summary", {})["finished_at"] = utc_now()
        write_report(args.report, runner.report)
        runner.log(f"Smoke failed. Report: {args.report}")
        return 1


if __name__ == "__main__":
    sys.exit(main())
