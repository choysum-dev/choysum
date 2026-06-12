#!/usr/bin/env python3
# SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
# SPDX-License-Identifier: LGPL-3.0-or-later

from __future__ import annotations

import argparse
import json
import os
import pathlib
import subprocess
import sys


REPO_ROOT = pathlib.Path(__file__).resolve().parents[2]
MODULES_ROOT = REPO_ROOT / "modules"


def is_all_zero_sha(value):
    text = (value or "").strip()
    return bool(text) and set(text) == {"0"}


def validate_schema_baseline():
    required_top = {"name", "version", "choysum", "publishConfig"}
    required_choysum = {"moduleName", "application", "entryPoints"}
    optional_entry_keys = {"service", "web"}

    errors = []
    for package_path in sorted(MODULES_ROOT.glob("*/package.json")):
        module_dir_name = package_path.parent.name
        rel = package_path.relative_to(REPO_ROOT)
        try:
            data = json.loads(package_path.read_text(encoding="utf-8"))
        except json.JSONDecodeError as exc:
            errors.append(f"{rel}: invalid JSON: {exc}")
            continue
        if not isinstance(data, dict):
            errors.append(f"{rel}: package.json must be a JSON object")
            continue

        missing_top = sorted(required_top - set(data.keys()))
        if missing_top:
            errors.append(f"{rel}: missing top-level keys: {', '.join(missing_top)}")
            continue

        choysum = data.get("choysum")
        if not isinstance(choysum, dict):
            errors.append(f"{rel}: choysum must be an object")
            continue

        missing_choysum = sorted(required_choysum - set(choysum.keys()))
        if missing_choysum:
            errors.append(f"{rel}: missing choysum keys: {', '.join(missing_choysum)}")
            continue

        module_name = choysum.get("moduleName")
        if not isinstance(module_name, str) or not module_name.strip():
            errors.append(f"{rel}: choysum.moduleName must be a non-empty string")
        elif module_name != module_dir_name:
            errors.append(
                f"{rel}: choysum.moduleName '{module_name}' must match directory name '{module_dir_name}'"
            )

        entry_points = choysum.get("entryPoints")
        if not isinstance(entry_points, dict) or not entry_points:
            errors.append(f"{rel}: choysum.entryPoints must be a non-empty object")
        else:
            unknown_keys = sorted(set(entry_points.keys()) - optional_entry_keys)
            if unknown_keys:
                errors.append(f"{rel}: choysum.entryPoints has unsupported keys: {', '.join(unknown_keys)}")
            if not any(key in entry_points for key in ("service", "web")):
                errors.append(f"{rel}: choysum.entryPoints must include at least one of service/web")
            for key, value in entry_points.items():
                if not isinstance(value, str) or not value.strip():
                    errors.append(f"{rel}: choysum.entryPoints.{key} must be a non-empty string")

        publish_cfg = data.get("publishConfig")
        if not isinstance(publish_cfg, dict) or publish_cfg.get("access") != "public":
            errors.append(f"{rel}: publishConfig.access must equal 'public'")

    if errors:
        print("preflight schema validation failed:", file=sys.stderr)
        for item in errors:
            print(f"- {item}", file=sys.stderr)
        raise SystemExit(1)

    print("preflight schema validation passed")


def discover_target_modules():
    target_module = os.environ.get("TARGET_MODULE", "").strip()
    event_name = os.environ.get("EVENT_NAME", "").strip()
    before = os.environ.get("EVENT_BEFORE", "").strip()
    head = os.environ.get("EVENT_SHA", "").strip()
    run_ref = os.environ.get("RUN_REF", "").strip()

    github_output = os.environ.get("GITHUB_OUTPUT", "").strip()
    if not github_output:
        raise SystemExit("GITHUB_OUTPUT is required")

    available_modules = sorted(
        child.name
        for child in MODULES_ROOT.iterdir()
        if child.is_dir() and child.name not in {".choysum", "tmp"}
    )

    if target_module:
        if target_module not in available_modules:
            raise SystemExit(f"target_module '{target_module}' does not exist under modules/")
        modules = [target_module]
    elif event_name == "push":
        modules = []
        if before and not is_all_zero_sha(before):
            try:
                diff = subprocess.run(
                    ["git", "diff", "--name-only", f"{before}...{head}"],
                    check=True,
                    capture_output=True,
                    text=True,
                )
                changed = [line.strip() for line in diff.stdout.splitlines() if line.strip()]
                seen = set()
                for path in changed:
                    parts = path.split("/")
                    if len(parts) >= 3 and parts[0] == "modules" and parts[1] in available_modules:
                        if parts[1] not in seen:
                            modules.append(parts[1])
                            seen.add(parts[1])
            except subprocess.CalledProcessError:
                modules = []
        if not modules:
            modules = available_modules
    else:
        modules = available_modules

    modules = sorted(dict.fromkeys(modules))
    has_modules = "true" if modules else "false"

    output_path = pathlib.Path(github_output)
    with output_path.open("a", encoding="utf-8") as fh:
        print(f"modules_json={json.dumps(modules, separators=(',', ':'))}", file=fh)
        print(f"has_modules={has_modules}", file=fh)
        print("artifact_name=module-discovery", file=fh)

    artifact_dir = REPO_ROOT / ".choysum" / "tmp"
    artifact_dir.mkdir(parents=True, exist_ok=True)
    report = {
        "modules": modules,
        "event": event_name,
        "target_module": target_module,
        "run_ref": run_ref,
    }
    (artifact_dir / "module-discovery.json").write_text(
        json.dumps(report, indent=2) + "\n",
        encoding="utf-8",
    )


def build_parser():
    parser = argparse.ArgumentParser(description="Preflight helpers for modules publish workflow")
    subparsers = parser.add_subparsers(dest="command", required=True)

    subparsers.add_parser("validate-schema-baseline")
    subparsers.add_parser("discover-target-modules")

    return parser


def main():
    parser = build_parser()
    args = parser.parse_args()

    if args.command == "validate-schema-baseline":
        validate_schema_baseline()
        return

    if args.command == "discover-target-modules":
        discover_target_modules()
        return

    raise SystemExit(f"Unsupported command: {args.command}")


if __name__ == "__main__":
    main()
