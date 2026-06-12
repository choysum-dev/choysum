#!/usr/bin/env python3
# SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
# SPDX-License-Identifier: LGPL-3.0-or-later

from __future__ import annotations

import argparse
import json
import os
import pathlib


REPO_ROOT = pathlib.Path.cwd()
MODULES_ROOT = REPO_ROOT / "modules"
IGNORED_MODULES = {".choysum", "tmp"}


def discover_modules_and_e2e_modules():
    modules = []
    e2e_modules = []

    if MODULES_ROOT.is_dir():
        for child in sorted(MODULES_ROOT.iterdir(), key=lambda entry: entry.name):
            if not child.is_dir() or child.name in IGNORED_MODULES:
                continue

            modules.append(child.name)

            package_path = child / "package.json"
            if not package_path.is_file():
                continue

            package_json = json.loads(package_path.read_text(encoding="utf-8"))
            choysum = package_json.get("choysum") if isinstance(package_json, dict) else {}
            e2e = choysum.get("e2e") if isinstance(choysum, dict) else {}
            specs = e2e.get("specs") if isinstance(e2e, dict) else ""
            if isinstance(specs, str) and specs.strip():
                e2e_modules.append(child.name)

    return modules, e2e_modules


def discover_nightly_e2e_modules():
    _, e2e_modules = discover_modules_and_e2e_modules()
    return e2e_modules


def write_outputs(outputs):
    github_output = os.environ.get("GITHUB_OUTPUT", "").strip()
    if not github_output:
        raise SystemExit("GITHUB_OUTPUT is required")

    with open(github_output, "a", encoding="utf-8") as fh:
        for key, value in outputs.items():
            print(f"{key}={value}", file=fh)


def build_parser():
    parser = argparse.ArgumentParser(description="Module matrix discovery helpers for CI workflows")
    subparsers = parser.add_subparsers(dest="command", required=True)

    subparsers.add_parser("discover-mainline")
    subparsers.add_parser("discover-nightly")

    return parser


def main():
    parser = build_parser()
    args = parser.parse_args()

    if args.command == "discover-mainline":
        modules, e2e_modules = discover_modules_and_e2e_modules()
        outputs = {
            "modules_json": json.dumps(modules, separators=(",", ":")),
            "e2e_modules_json": json.dumps(e2e_modules, separators=(",", ":")),
        }
        write_outputs(outputs)
        return

    if args.command == "discover-nightly":
        e2e_modules = discover_nightly_e2e_modules()
        outputs = {
            "e2e_modules_json": json.dumps(e2e_modules, separators=(",", ":")),
        }
        write_outputs(outputs)
        return

    raise SystemExit(f"Unsupported command: {args.command}")


if __name__ == "__main__":
    main()
