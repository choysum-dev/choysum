#!/usr/bin/env python3
# SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
# SPDX-License-Identifier: LGPL-3.0-or-later

import json
import posixpath
import re
import sys
from pathlib import Path

IGNORED_MODULE_DIRS = {".choysum", "tmp"}
ALLOWED_ENTRY_KEYS = {"service", "web"}
SEMVER_RE = re.compile(r"^\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$")
WINDOWS_DRIVE_RE = re.compile(r"^[A-Za-z]:[/\\\\]")


def load_json(path: Path):
    return json.loads(path.read_text(encoding="utf-8"))


def validate_relative_path(raw_value, field):
    if not isinstance(raw_value, str):
        return f"{field} must be a string"

    value = raw_value.strip()
    if value == "":
        return f"{field} cannot be empty"

    normalized = value.replace("\\\\", "/")
    if normalized.startswith("/"):
        return f"{field} must be a relative path, got {raw_value!r}"
    if WINDOWS_DRIVE_RE.match(value):
        return f"{field} must be a relative path, got {raw_value!r}"

    if any(seg == ".." for seg in normalized.split("/")):
        return f"{field} cannot contain parent traversal, got {raw_value!r}"

    cleaned = posixpath.normpath(normalized)
    if cleaned in {"", "."}:
        return f"{field} must point to a file path, got {raw_value!r}"

    return None


def discover_modules(addons_root: Path):
    modules = []
    if not addons_root.is_dir():
        return modules

    for child in sorted(addons_root.iterdir(), key=lambda item: item.name):
        if not child.is_dir():
            continue
        if child.name in IGNORED_MODULE_DIRS:
            continue
        if child.name.startswith("."):
            continue
        modules.append(child.name)
    return modules


def main():
    repo_root = Path(__file__).resolve().parents[2]
    addons_root = repo_root / "addons"

    modules = discover_modules(addons_root)
    module_set = set(modules)

    if not modules:
        print("No addon modules discovered under addons/.")
        return 1

    errors = []

    for module_name in modules:
        module_dir = addons_root / module_name
        package_json_path = module_dir / "package.json"
        manifest_json_path = module_dir / "manifest.json"

        if manifest_json_path.exists():
            errors.append(f"[{module_name}] manifest.json must not exist")

        if not package_json_path.is_file():
            errors.append(f"[{module_name}] package.json is missing")
            continue

        try:
            package_json = load_json(package_json_path)
        except Exception as exc:  # noqa: BLE001
            errors.append(f"[{module_name}] package.json parse failed: {exc}")
            continue

        choysum = package_json.get("choysum")
        if not isinstance(choysum, dict):
            errors.append(f"[{module_name}] choysum must be an object")
            continue

        choysum_module_name = str(choysum.get("moduleName", "")).strip()
        if choysum_module_name != module_name:
            errors.append(
                f"[{module_name}] choysum.moduleName must equal directory name, got {choysum_module_name!r}"
            )

        version = package_json.get("version")
        if not isinstance(version, str) or version.strip() == "":
            errors.append(f"[{module_name}] version is required")
        else:
            trimmed_version = version.strip()
            if trimmed_version.startswith("v"):
                errors.append(f"[{module_name}] version must be SemVer without leading 'v', got {trimmed_version!r}")
            elif not SEMVER_RE.match(trimmed_version):
                errors.append(f"[{module_name}] version must be valid SemVer, got {trimmed_version!r}")

        depends = choysum.get("depends")
        if depends is None:
            depends = []
        if not isinstance(depends, list):
            errors.append(f"[{module_name}] choysum.depends must be an array")
        else:
            for idx, dep in enumerate(depends):
                if not isinstance(dep, str) or dep.strip() == "":
                    errors.append(f"[{module_name}] choysum.depends[{idx}] must be a non-empty string")
                    continue
                dep_name = dep.strip()
                if dep_name not in module_set:
                    errors.append(f"[{module_name}] choysum.depends[{idx}] references unknown module {dep_name!r}")

        entry_points = choysum.get("entryPoints")
        if not isinstance(entry_points, dict) or not entry_points:
            errors.append(f"[{module_name}] choysum.entryPoints must be a non-empty object")
        else:
            for key, value in entry_points.items():
                if key not in ALLOWED_ENTRY_KEYS:
                    errors.append(f"[{module_name}] choysum.entryPoints key {key!r} is not allowed")
                    continue

                path_error = validate_relative_path(value, f"choysum.entryPoints.{key}")
                if path_error:
                    errors.append(f"[{module_name}] {path_error}")
                    continue

                entry_path = module_dir / value
                if not entry_path.is_file():
                    errors.append(
                        f"[{module_name}] choysum.entryPoints.{key} file does not exist: {value!r}"
                    )

    if errors:
        print("Addon package contract check failed:")
        for err in errors:
            print(f" - {err}")
        return 1

    print(f"Addon package contract check passed for {len(modules)} modules.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
