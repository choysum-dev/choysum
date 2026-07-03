#!/usr/bin/env python3
# SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
# SPDX-License-Identifier: LGPL-3.0-or-later

from __future__ import annotations

import argparse
import json
import os
import pathlib
import re
import shutil
import subprocess
import sys
import urllib.error
import urllib.parse
import urllib.request
from datetime import datetime, timezone


REPO_ROOT = pathlib.Path(__file__).resolve().parents[2]
TMP_DIR = REPO_ROOT / ".choysum" / "tmp"


def utc_now_iso():
    return datetime.now(timezone.utc).isoformat()


def ensure_tmp_dir():
    TMP_DIR.mkdir(parents=True, exist_ok=True)
    return TMP_DIR


def read_json(path):
    return json.loads(path.read_text(encoding="utf-8"))


def write_json(path, payload):
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(payload, indent=2) + "\n", encoding="utf-8")


def compact(text, limit=8000):
    value = (text or "").strip()
    if len(value) <= limit:
        return value
    return value[-limit:]


def load_json_env(env_name):
    raw = os.environ.get(env_name)
    if raw is None:
        raise SystemExit(f"{env_name} env is required")
    try:
        return json.loads(raw)
    except Exception as exc:
        raise SystemExit(f"{env_name} is not valid JSON: {exc}")


def env_bool(name, default=False):
    raw = os.environ.get(name)
    if raw is None:
        return default
    return raw.strip().lower() == "true"


SEMVER_COMPARATOR_PATTERN = re.compile(
    r"^(<=|>=|<|>|=|~|\^)?(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$"
)


def is_valid_semver_constraint(value):
    text = (value or "").strip()
    if not text:
        return False

    disjunctions = [item.strip() for item in text.split("||")]
    if any(not item for item in disjunctions):
        return False

    for item in disjunctions:
        comparators = [token.strip() for token in item.split() if token.strip()]
        if not comparators:
            return False
        for comparator in comparators:
            if not SEMVER_COMPARATOR_PATTERN.match(comparator):
                return False

    return True


def verify_gate():
    modules = load_json_env("MODULES_JSON")
    modules_root = REPO_ROOT / "modules"
    required_cli_range = os.environ.get("REQUIRED_CHOYSUM_CLI_RANGE", ">=0.0.0-0 <0.0.0").strip()

    errors = []
    for module in modules:
        package_path = modules_root / module / "package.json"
        if not package_path.is_file():
            errors.append(f"modules/{module}/package.json is missing")
            continue

        data = read_json(package_path)
        if not isinstance(data, dict):
            errors.append(f"modules/{module}/package.json is not a JSON object")
            continue

        name = data.get("name")
        version = data.get("version")
        choysum = data.get("choysum")
        module_name = choysum.get("moduleName") if isinstance(choysum, dict) else None
        cli_range = choysum.get("cli") if isinstance(choysum, dict) else None

        if not isinstance(name, str) or not name.startswith("@"):
            errors.append(f"modules/{module}/package.json: name must be a scoped package")

        if not isinstance(version, str) or not version.strip():
            errors.append(f"modules/{module}/package.json: version must be non-empty")

        if module_name != module:
            errors.append(
                f"modules/{module}/package.json: choysum.moduleName '{module_name}' must equal directory '{module}'"
            )

        if not isinstance(cli_range, str) or not cli_range.strip():
            errors.append(f"modules/{module}/package.json: choysum.cli must be a non-empty semver constraint string")
        else:
            normalized_cli_range = cli_range.strip()
            if not is_valid_semver_constraint(normalized_cli_range):
                errors.append(
                    f"modules/{module}/package.json: choysum.cli '{normalized_cli_range}' is not a valid semver constraint"
                )
            if required_cli_range and normalized_cli_range != required_cli_range:
                errors.append(
                    f"modules/{module}/package.json: choysum.cli '{normalized_cli_range}' must equal required policy '{required_cli_range}'"
                )

    if errors:
        print("verify-module-gate failed:", file=sys.stderr)
        for item in errors:
            print(f"- {item}", file=sys.stderr)
        raise SystemExit(1)

    print("verify-module-gate metadata checks passed")


def publish_single_module():
    output_dir = ensure_tmp_dir()
    modules_root = REPO_ROOT / "modules"

    module = os.environ.get("MODULE_NAME", "").strip()
    if not module:
        raise SystemExit("MODULE_NAME env is required")

    dry_run = env_bool("DRY_RUN", default=True)
    semver_pattern = re.compile(
        r"^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$"
    )

    result_file = output_dir / f"publish-result-{module}.json"
    error_file = output_dir / f"publish-error-{module}.json"

    errors = []
    result = {
        "module": module,
        "status": "failed",
    }

    package_path = modules_root / module / "package.json"
    if not package_path.is_file():
        errors.append(
            {
                "module": module,
                "operation": "load_package_json",
                "error": "missing_package_json",
            }
        )
    else:
        package_json = read_json(package_path)
        if not isinstance(package_json, dict):
            errors.append(
                {
                    "module": module,
                    "operation": "load_package_json",
                    "error": "invalid_package_json",
                    "message": "package.json is not a JSON object",
                }
            )
        else:
            name = package_json.get("name")
            version = package_json.get("version")
            choysum = package_json.get("choysum")
            module_name = choysum.get("moduleName") if isinstance(choysum, dict) else None

            result["name"] = name
            result["version"] = version

            if not isinstance(name, str) or not name.startswith("@"):
                errors.append(
                    {
                        "module": module,
                        "operation": "publish_precheck",
                        "error": "invalid_package_name",
                        "message": "name must be a scoped package",
                    }
                )

            if not isinstance(version, str) or not semver_pattern.match(version):
                errors.append(
                    {
                        "module": module,
                        "operation": "publish_precheck",
                        "error": "invalid_package_version",
                        "message": "version must be valid semver",
                    }
                )

            if module_name != module:
                errors.append(
                    {
                        "module": module,
                        "operation": "publish_precheck",
                        "error": "module_name_mismatch",
                        "message": f"choysum.moduleName '{module_name}' must match directory '{module}'",
                    }
                )

            if not errors:
                pack = subprocess.run(
                    ["npm", "pack", "--dry-run", "--json"],
                    cwd=str(modules_root / module),
                    check=False,
                    capture_output=True,
                    text=True,
                )
                if pack.returncode != 0:
                    errors.append(
                        {
                            "module": module,
                            "operation": "npm_pack_dry_run",
                            "command": "npm pack --dry-run --json",
                            "exitCode": pack.returncode,
                            "stdout": compact(pack.stdout),
                            "stderr": compact(pack.stderr),
                        }
                    )

            if not errors:
                identifier = f"{name}@{version}"
                view = subprocess.run(
                    ["npm", "view", identifier, "version"],
                    check=False,
                    capture_output=True,
                    text=True,
                )

                version_exists = view.returncode == 0 and view.stdout.strip() == version
                if version_exists:
                    result["status"] = "already_published"
                else:
                    view_text = f"{view.stdout}\n{view.stderr}".lower()
                    view_not_found = (
                        (view.returncode == 0 and not view.stdout.strip())
                        or "e404" in view_text
                        or "404 not found" in view_text
                    )
                    if not view_not_found:
                        errors.append(
                            {
                                "module": module,
                                "operation": "npm_view",
                                "command": f"npm view {identifier} version",
                                "exitCode": view.returncode,
                                "stdout": compact(view.stdout),
                                "stderr": compact(view.stderr),
                            }
                        )
                    elif dry_run:
                        result["status"] = "published"
                        result["dryRun"] = True
                        result["note"] = "would_publish"
                    else:
                        publish = subprocess.run(
                            ["npm", "publish", "--access", "public"],
                            cwd=str(modules_root / module),
                            check=False,
                            capture_output=True,
                            text=True,
                        )
                        if publish.returncode == 0:
                            result["status"] = "published"
                        else:
                            errors.append(
                                {
                                    "module": module,
                                    "operation": "npm_publish",
                                    "command": "npm publish --access public",
                                    "exitCode": publish.returncode,
                                    "stdout": compact(publish.stdout),
                                    "stderr": compact(publish.stderr),
                                }
                            )

    if errors:
        result["status"] = "failed"
        result["errorRef"] = f"publish-error-{module}.json"

    write_json(result_file, result)
    write_json(
        error_file,
        {
            "generatedAt": utc_now_iso(),
            "module": module,
            "errors": errors,
        },
    )

    if result.get("status") == "failed":
        raise SystemExit("module publish shard failed")


def aggregate_publish_results():
    expected_modules = load_json_env("EXPECTED_MODULES_JSON")
    output_dir = ensure_tmp_dir()
    shard_dir = output_dir / "publish-results"

    results = []
    errors = []

    for module in expected_modules:
        result_path = shard_dir / f"publish-result-{module}.json"
        error_path = shard_dir / f"publish-error-{module}.json"

        if not result_path.is_file():
            results.append(
                {
                    "module": module,
                    "status": "failed",
                    "error": "missing_publish_result_artifact",
                    "errorRef": f"publish-error-{module}.json",
                }
            )
            errors.append(
                {
                    "module": module,
                    "operation": "aggregate_publish_results",
                    "error": "missing_publish_result_artifact",
                }
            )
            continue

        result = read_json(result_path)
        results.append(result)

        if error_path.is_file():
            payload = read_json(error_path)
            for item in payload.get("errors", []):
                errors.append(item)
        elif result.get("status") == "failed":
            errors.append(
                {
                    "module": module,
                    "operation": "aggregate_publish_results",
                    "error": "missing_publish_error_artifact",
                }
            )

    results.sort(key=lambda item: item.get("module", ""))
    write_json(output_dir / "published-modules.json", results)
    write_json(
        output_dir / "publish-errors.json",
        {
            "generatedAt": utc_now_iso(),
            "errors": errors,
        },
    )

    failed = [item for item in results if item.get("status") == "failed"]
    if failed:
        print("publish-npm-registry aggregate found failed module shards", file=sys.stderr)
        raise SystemExit("One or more module publish shards failed")


def sync_precheck():
    output_dir = ensure_tmp_dir()
    token = os.environ.get("MODULES_DIRECTORY_SYNC_TOKEN", "").strip()

    errors = []
    summary = {
        "status": "precheck_passed",
        "note": "modules-directory precheck completed",
    }

    if not token:
        errors.append(
            {
                "operation": "sync_precheck",
                "error": "missing_secret",
                "message": "MODULES_DIRECTORY_SYNC_TOKEN is not configured",
            }
        )
        summary = {
            "status": "precheck_failed",
            "note": "missing MODULES_DIRECTORY_SYNC_TOKEN",
        }
    else:
        request = urllib.request.Request(
            "https://api.github.com/repos/choysum-dev/modules-directory",
            headers={
                "Accept": "application/vnd.github+json",
                "Authorization": f"Bearer {token}",
                "X-GitHub-Api-Version": "2022-11-28",
                "User-Agent": "choysum-publish-script",
            },
        )
        try:
            with urllib.request.urlopen(request, timeout=20) as response:
                _ = response.read().decode("utf-8", errors="replace")
        except urllib.error.HTTPError as exc:
            errors.append(
                {
                    "operation": "sync_precheck",
                    "error": "http_error",
                    "status": exc.code,
                    "response": exc.read().decode("utf-8", errors="replace"),
                }
            )
            summary = {
                "status": "precheck_failed",
                "note": "modules-directory API returned non-2xx",
            }
        except Exception as exc:
            errors.append(
                {
                    "operation": "sync_precheck",
                    "error": "request_exception",
                    "message": str(exc),
                }
            )
            summary = {
                "status": "precheck_failed",
                "note": "modules-directory API request exception",
            }

    write_json(
        output_dir / "sync-errors.json",
        {
            "generatedAt": utc_now_iso(),
            "errors": errors,
        },
    )
    write_json(output_dir / "sync-summary.json", summary)

    if errors:
        raise SystemExit("sync-catalog-directory-pr precheck failed")


def sync_modules_per_module_pr():
    output_dir = ensure_tmp_dir()
    published_path = output_dir / "published-modules.json"
    summary_path = output_dir / "sync-summary.json"
    errors_path = output_dir / "sync-errors.json"

    token = os.environ.get("MODULES_DIRECTORY_SYNC_TOKEN", "").strip()
    dry_run = env_bool("DRY_RUN", default=True)
    source_run_url = os.environ.get("SOURCE_RUN_URL", "").strip()
    source_run_ref = os.environ.get("SOURCE_RUN_REF", "").strip()
    source_sha = os.environ.get("SOURCE_SHA", "").strip()

    api_base = "https://api.github.com/repos/choysum-dev/modules-directory"
    repo_url = "https://github.com/choysum-dev/modules-directory.git"
    repo_clone_dir = output_dir / "modules-directory"
    redaction_values = []
    if token:
        redaction_values.append(token)

    errors = []
    results = []

    def run_cmd(command, cwd=None, check=True, env=None):
        safe_command = []
        for part in command:
            value = str(part)
            for secret in redaction_values:
                value = value.replace(secret, "***")
            safe_command.append(value)

        try:
            proc = subprocess.run(
                command,
                cwd=str(cwd) if cwd is not None else None,
                check=False,
                capture_output=True,
                text=True,
                env=env,
            )
        except Exception as exc:
            message = str(exc)
            for secret in redaction_values:
                message = message.replace(secret, "***")
            raise RuntimeError(f"command execution failed ({' '.join(safe_command)}): {message}") from None

        safe_stdout = proc.stdout or ""
        safe_stderr = proc.stderr or ""
        for secret in redaction_values:
            safe_stdout = safe_stdout.replace(secret, "***")
            safe_stderr = safe_stderr.replace(secret, "***")

        if check and proc.returncode != 0:
            raise RuntimeError(
                f"command failed ({' '.join(safe_command)}): exit={proc.returncode}; "
                f"stdout={compact(safe_stdout)}; stderr={compact(safe_stderr)}"
            )
        return proc

    askpass_path = output_dir / "git-askpass.sh"

    def ensure_askpass_script():
        askpass_path.write_bytes(
            b"#!/bin/sh\n"
            b"case \"$1\" in\n"
            b"  *[Uu]sername*) printf '%s\\n' 'x-access-token' ;;\n"
            b"  *) printf '%s\\n' \"$CHOYSUM_SYNC_GIT_TOKEN\" ;;\n"
            b"esac\n"
        )
        askpass_path.chmod(0o700)

    def run_git_with_auth(command, cwd=None, check=True):
        ensure_askpass_script()

        auth_env = os.environ.copy()
        auth_env["LC_ALL"] = "C"
        auth_env["GIT_TERMINAL_PROMPT"] = "0"
        auth_env["GIT_ASKPASS"] = askpass_path.as_posix()
        auth_env["CHOYSUM_SYNC_GIT_TOKEN"] = token

        return run_cmd(command, cwd=cwd, check=check, env=auth_env)

    def api_json(method, url, payload=None):
        body = None
        if payload is not None:
            body = json.dumps(payload).encode("utf-8")
        request = urllib.request.Request(
            url,
            data=body,
            method=method,
            headers={
                "Accept": "application/vnd.github+json",
                "Authorization": f"Bearer {token}",
                "X-GitHub-Api-Version": "2022-11-28",
                "Content-Type": "application/json",
                "User-Agent": "choysum-publish-script",
            },
        )
        try:
            with urllib.request.urlopen(request, timeout=30) as response:
                raw = response.read().decode("utf-8", errors="replace")
                if not raw:
                    return None
                return json.loads(raw)
        except urllib.error.HTTPError as exc:
            message = exc.read().decode("utf-8", errors="replace")
            raise RuntimeError(
                f"GitHub API {method} {url} failed: status={exc.code}, response={compact(message)}"
            )

    def sanitize_module(module):
        cleaned = re.sub(r"[^A-Za-z0-9._-]+", "-", module).strip("-")
        return cleaned or "module"

    def find_open_pr(branch_name):
        query = urllib.parse.urlencode(
            {
                "state": "open",
                "head": f"choysum-dev:{branch_name}",
            }
        )
        payload = api_json("GET", f"{api_base}/pulls?{query}")
        if isinstance(payload, list) and payload:
            return payload[0]
        return None

    def ensure_repo_ready():
        if repo_clone_dir.exists():
            shutil.rmtree(repo_clone_dir)
        run_git_with_auth(["git", "clone", "--depth", "1", repo_url, str(repo_clone_dir)])
        run_cmd(["git", "config", "user.name", "choysum-ci-bot"], cwd=repo_clone_dir)
        run_cmd(["git", "config", "user.email", "bot@choysum.dev"], cwd=repo_clone_dir)
        run_git_with_auth(["git", "fetch", "origin", "+refs/heads/*:refs/remotes/origin/*"], cwd=repo_clone_dir)

    def remote_branch_exists(branch_name):
        probe = run_cmd(
            ["git", "rev-parse", "--verify", f"origin/{branch_name}"],
            cwd=repo_clone_dir,
            check=False,
        )
        return probe.returncode == 0

    def find_pointer_path(module):
        existing = list((repo_clone_dir / "modules").glob(f"*/{module}.json"))
        if len(existing) == 1:
            return existing[0], None
        if len(existing) > 1:
            return None, f"multiple_pointer_files_found: {[str(p.relative_to(repo_clone_dir)) for p in existing]}"
        return repo_clone_dir / "modules" / "official" / f"{module}.json", None

    published_items = []
    if not token:
        errors.append(
            {
                "operation": "sync_modules",
                "error": "missing_secret",
                "message": "MODULES_DIRECTORY_SYNC_TOKEN is not configured",
            }
        )

    if not published_path.is_file():
        errors.append(
            {
                "operation": "sync_modules",
                "error": "missing_published_modules_artifact",
                "message": "published-modules.json is not available",
            }
        )

    if not errors:
        try:
            published_items = read_json(published_path)
        except Exception as exc:
            errors.append(
                {
                    "operation": "sync_modules",
                    "error": "invalid_published_modules_json",
                    "message": str(exc),
                }
            )

    if not errors and not isinstance(published_items, list):
        errors.append(
            {
                "operation": "sync_modules",
                "error": "invalid_published_modules_artifact",
                "message": "published-modules.json must be an array",
            }
        )

    syncable_status = {"published", "already_published"}
    candidates = []
    if not errors:
        for item in published_items:
            if isinstance(item, dict) and item.get("status") in syncable_status:
                candidates.append(item)

    if not errors and not dry_run:
        try:
            ensure_repo_ready()
        except Exception as exc:
            errors.append(
                {
                    "operation": "sync_modules",
                    "error": "clone_modules_directory_failed",
                    "message": str(exc),
                }
            )

    if not errors:
        for item in candidates:
            module = str(item.get("module", "")).strip()
            package = str(item.get("name", "")).strip()
            publish_status = str(item.get("status", "")).strip()

            if not module or not package:
                errors.append(
                    {
                        "operation": "sync_module",
                        "error": "missing_module_or_package",
                        "module": module,
                        "package": package,
                    }
                )
                continue

            branch_name = f"sync/modules-update-{sanitize_module(module)}"

            try:
                open_pr = find_open_pr(branch_name)
                pr_url = open_pr.get("html_url") if isinstance(open_pr, dict) else None

                if dry_run:
                    status = "dry_run_would_update_open_pr" if pr_url else "dry_run_would_create_pr"
                    results.append(
                        {
                            "module": module,
                            "package": package,
                            "publishStatus": publish_status,
                            "branch": branch_name,
                            "status": status,
                            "prUrl": pr_url,
                        }
                    )
                    continue

                has_remote_branch = remote_branch_exists(branch_name)
                if open_pr and has_remote_branch:
                    run_cmd(["git", "checkout", "-B", branch_name, f"origin/{branch_name}"], cwd=repo_clone_dir)
                else:
                    run_cmd(["git", "checkout", "-B", branch_name, "origin/main"], cwd=repo_clone_dir)

                pointer_path, pointer_path_error = find_pointer_path(module)
                if pointer_path_error:
                    raise RuntimeError(pointer_path_error)
                pointer_path.parent.mkdir(parents=True, exist_ok=True)

                existing = {}
                if pointer_path.is_file():
                    existing = read_json(pointer_path)
                    if not isinstance(existing, dict):
                        existing = {}

                pointer = dict(existing)
                pointer.setdefault("$schema", "../../schemas/catalog-entry.schema.json")
                pointer["package"] = package
                trust = pointer.get("trust")
                if trust not in {"official", "verified", "community"}:
                    pointer["trust"] = "official"
                maintainers = pointer.get("maintainers")
                if not isinstance(maintainers, list) or not maintainers:
                    pointer["maintainers"] = [{"github": "choysum-dev"}]

                new_content = json.dumps(pointer, indent=2) + "\n"
                old_content = pointer_path.read_text(encoding="utf-8") if pointer_path.is_file() else ""

                if new_content == old_content:
                    results.append(
                        {
                            "module": module,
                            "package": package,
                            "publishStatus": publish_status,
                            "branch": branch_name,
                            "status": "no_change",
                            "prUrl": pr_url,
                        }
                    )
                    continue

                pointer_path.write_text(new_content, encoding="utf-8")
                rel_path = str(pointer_path.relative_to(repo_clone_dir))
                run_cmd(["git", "add", rel_path], cwd=repo_clone_dir)
                run_cmd(["git", "commit", "-m", f"chore(modules): sync pointer for {module}"], cwd=repo_clone_dir)
                push_cmd = ["git", "push", f"--force-with-lease={branch_name}", "origin", branch_name]

                run_git_with_auth(push_cmd, cwd=repo_clone_dir)

                if open_pr:
                    results.append(
                        {
                            "module": module,
                            "package": package,
                            "publishStatus": publish_status,
                            "branch": branch_name,
                            "status": "updated_open_pr",
                            "prUrl": pr_url,
                        }
                    )
                    continue

                title = f"chore(modules): sync pointer for {module}"
                body = "\n".join(
                    [
                        "Automated sync from choysum modules publish workflow.",
                        "",
                        f"- Module: {module}",
                        f"- Package: {package}",
                        f"- Publish status: {publish_status}",
                        f"- Source run: {source_run_url}",
                        f"- Source ref: {source_run_ref}",
                        f"- Source sha: {source_sha}",
                    ]
                )
                created_pr = api_json(
                    "POST",
                    f"{api_base}/pulls",
                    {
                        "title": title,
                        "head": branch_name,
                        "base": "main",
                        "body": body,
                    },
                )
                results.append(
                    {
                        "module": module,
                        "package": package,
                        "publishStatus": publish_status,
                        "branch": branch_name,
                        "status": "created_pr",
                        "prUrl": created_pr.get("html_url") if isinstance(created_pr, dict) else None,
                    }
                )
            except Exception as exc:
                errors.append(
                    {
                        "operation": "sync_module",
                        "module": module,
                        "package": package,
                        "branch": branch_name,
                        "error": "sync_module_failed",
                        "message": str(exc),
                    }
                )
                results.append(
                    {
                        "module": module,
                        "package": package,
                        "publishStatus": publish_status,
                        "branch": branch_name,
                        "status": "failed",
                    }
                )

    for item in published_items:
        if not isinstance(item, dict):
            continue
        status = item.get("status")
        if status in syncable_status:
            continue
        results.append(
            {
                "module": item.get("module"),
                "package": item.get("name"),
                "publishStatus": status,
                "status": "skipped_non_syncable_status",
            }
        )

    catalog_rebuild_triggered = None
    catalog_rebuild_error = None
    if not dry_run and any(c.get("status") == "published" for c in candidates) and not errors:
        try:
            api_json(
                "POST",
                f"{api_base}/actions/workflows/build-catalog.yml/dispatches",
                {"ref": "main"},
            )
            catalog_rebuild_triggered = True
        except Exception as exc:
            catalog_rebuild_triggered = False
            catalog_rebuild_error = str(exc)
            # Non-fatal: the hourly schedule cron in modules-directory serves as fallback.

    write_json(
        summary_path,
        {
            "generatedAt": utc_now_iso(),
            "mode": "per_module_pr",
            "dryRun": dry_run,
            "sourceRunUrl": source_run_url,
            "sourceRunRef": source_run_ref,
            "sourceSha": source_sha,
            "candidateCount": len(candidates),
            "catalogRebuildTriggered": catalog_rebuild_triggered,
            "catalogRebuildError": catalog_rebuild_error,
            "results": results,
        },
    )
    write_json(
        errors_path,
        {
            "generatedAt": utc_now_iso(),
            "errors": errors,
        },
    )

    if errors:
        raise SystemExit("sync-catalog-directory-pr per-module sync failed")


def build_parser():
    parser = argparse.ArgumentParser(description="Utilities for modules publish workflow")
    subparsers = parser.add_subparsers(dest="command", required=True)

    subparsers.add_parser("verify-gate")
    subparsers.add_parser("publish-single-module")
    subparsers.add_parser("aggregate-publish-results")
    subparsers.add_parser("sync-precheck")
    subparsers.add_parser("sync-modules-per-module-pr")

    return parser


def main():
    parser = build_parser()
    args = parser.parse_args()

    if args.command == "verify-gate":
        verify_gate()
        return

    if args.command == "publish-single-module":
        publish_single_module()
        return

    if args.command == "aggregate-publish-results":
        aggregate_publish_results()
        return

    if args.command == "sync-precheck":
        sync_precheck()
        return

    if args.command == "sync-modules-per-module-pr":
        sync_modules_per_module_pr()
        return

    raise SystemExit(f"Unsupported command: {args.command}")


if __name__ == "__main__":
    main()
