#!/usr/bin/env python3
# SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
# SPDX-License-Identifier: LGPL-3.0-or-later
"""Idempotent local bootstrap + npm Trusted Publishing binder for Choysum modules.

Default (no --apply): preview only.
With --apply: for each target module, bootstrap-publish if the package name is
missing on the registry, then bind Trusted Publishing if not already matching
the contract. Does not bump/republish existing package versions (CI handles that).

Trust contract (must match Modules Publish workflow):
  repo        = choysum-dev/choysum
  file        = modules-publish.yml
  environment = npm-publish
  allow-publish
"""

from __future__ import annotations

import argparse
import json
import pathlib
import subprocess
import sys


REPO_ROOT = pathlib.Path(__file__).resolve().parents[2]
MODULES_ROOT = REPO_ROOT / "modules"

TRUST_REPO = "choysum-dev/choysum"
TRUST_FILE = "modules-publish.yml"
TRUST_ENVIRONMENT = "npm-publish"


def run(cmd, *, cwd=None, check=False):
    return subprocess.run(
        cmd,
        cwd=str(cwd) if cwd else None,
        check=check,
        capture_output=True,
        text=True,
    )


def load_package(module_dir: pathlib.Path):
    package_path = module_dir / "package.json"
    if not package_path.is_file():
        return None, f"missing package.json under {module_dir}"
    try:
        data = json.loads(package_path.read_text(encoding="utf-8"))
    except json.JSONDecodeError as exc:
        return None, f"invalid JSON in {package_path}: {exc}"
    if not isinstance(data, dict):
        return None, f"{package_path} is not a JSON object"
    name = data.get("name")
    version = data.get("version")
    choysum = data.get("choysum")
    module_name = choysum.get("moduleName") if isinstance(choysum, dict) else None
    if not isinstance(name, str) or not name.startswith("@"):
        return None, f"{package_path}: name must be a scoped package"
    if module_name != module_dir.name:
        return None, (
            f"{package_path}: choysum.moduleName '{module_name}' "
            f"must equal directory '{module_dir.name}'"
        )
    return {"name": name, "version": version, "path": package_path, "dir": module_dir}, None


def discover_modules(module=None, package_json=None):
    if package_json:
        path = pathlib.Path(package_json).resolve()
        if not path.is_file():
            raise SystemExit(f"--package-json not found: {path}")
        info, err = load_package(path.parent)
        if err:
            raise SystemExit(err)
        return [info]

    if module:
        module_dir = pathlib.Path(module)
        if not module_dir.is_absolute():
            candidate = MODULES_ROOT / module
            if candidate.is_dir():
                module_dir = candidate
            elif module_dir.is_dir():
                module_dir = module_dir.resolve()
            else:
                module_dir = candidate
        if not module_dir.is_dir():
            raise SystemExit(f"module directory not found: {module}")
        info, err = load_package(module_dir)
        if err:
            raise SystemExit(err)
        return [info]

    modules = []
    for child in sorted(MODULES_ROOT.iterdir()):
        if not child.is_dir() or child.name in {".choysum", "tmp"}:
            continue
        info, err = load_package(child)
        if err:
            print(f"skip {child.name}: {err}", file=sys.stderr)
            continue
        modules.append(info)
    return modules


def package_exists(name: str) -> bool:
    proc = run(["npm", "view", name, "version"])
    if proc.returncode == 0 and proc.stdout.strip():
        return True
    text = f"{proc.stdout}\n{proc.stderr}".lower()
    if "e404" in text or "404 not found" in text:
        return False
    # Ambiguous: treat as exists to avoid accidental bootstrap.
    print(
        f"warning: npm view {name} inconclusive (exit={proc.returncode}); "
        "treating as exists to skip bootstrap",
        file=sys.stderr,
    )
    return True


def trust_list(name: str):
    proc = run(["npm", "trust", "list", name, "--json"])
    if proc.returncode != 0:
        return None, compact(proc.stderr or proc.stdout)
    raw = (proc.stdout or "").strip()
    if not raw:
        return [], None
    try:
        data = json.loads(raw)
    except json.JSONDecodeError:
        return None, f"invalid trust list JSON for {name}: {raw[:500]}"
    if data is None:
        return [], None
    if isinstance(data, dict):
        # Some CLI versions return a single object or {configurations:[...]}.
        if "configurations" in data and isinstance(data["configurations"], list):
            return data["configurations"], None
        return [data], None
    if isinstance(data, list):
        return data, None
    return None, f"unexpected trust list payload type for {name}: {type(data)}"


def compact(text, limit=2000):
    value = (text or "").strip()
    if len(value) <= limit:
        return value
    return value[-limit:]


def trust_matches(entry: dict) -> bool:
    if not isinstance(entry, dict):
        return False
    # Be tolerant of nested provider payloads.
    blob = json.dumps(entry).lower()
    repo_ok = TRUST_REPO.lower() in blob or TRUST_REPO.replace("/", "%2f").lower() in blob
    file_ok = TRUST_FILE.lower() in blob
    env_ok = TRUST_ENVIRONMENT.lower() in blob
    return repo_ok and file_ok and env_ok


def bind_trust(name: str, *, apply: bool, rebind: bool):
    entries, err = trust_list(name)
    if err:
        # Interactive auth prompts often land here; surface stderr.
        return "error", err

    matching = [e for e in (entries or []) if trust_matches(e)]
    others = [e for e in (entries or []) if e not in matching]

    if matching and not others:
        return "skip", "trust already matches contract"
    if matching and others and not rebind:
        return "error", "mixed trust configs; re-run with --rebind to replace"
    if others and not matching and not rebind:
        return "error", (
            "existing trust config does not match contract; "
            "re-run with --rebind to revoke and recreate"
        )

    if not apply:
        action = "rebind" if (others or matching) else "bind"
        return "would_" + action, "would run npm trust github …"

    if rebind and (entries or []):
        for entry in entries:
            trust_id = None
            if isinstance(entry, dict):
                trust_id = entry.get("id") or entry.get("_id") or entry.get("trustId")
            if not trust_id:
                return "error", f"cannot revoke: missing id in {entry!r}"
            rev = run(["npm", "trust", "revoke", name, f"--id={trust_id}", "--yes"])
            if rev.returncode != 0:
                return "error", compact(rev.stderr or rev.stdout)

    cmd = [
        "npm",
        "trust",
        "github",
        name,
        "--repo",
        TRUST_REPO,
        "--file",
        TRUST_FILE,
        "--environment",
        TRUST_ENVIRONMENT,
        "--allow-publish",
        "--yes",
    ]
    proc = run(cmd)
    if proc.returncode != 0:
        return "error", compact(proc.stderr or proc.stdout)
    return "bound", "trusted publisher configured"


def bootstrap_publish(info, *, apply: bool):
    name = info["name"]
    if package_exists(name):
        return "skip", "package already on registry"
    if not apply:
        return "would_publish", f"would npm publish {name}@{info.get('version')} from {info['dir']}"

    pack = run(["npm", "pack", "--dry-run"], cwd=info["dir"])
    if pack.returncode != 0:
        return "error", compact(pack.stderr or pack.stdout)

    pub = run(["npm", "publish", "--access", "public"], cwd=info["dir"])
    if pub.returncode != 0:
        return "error", compact(pub.stderr or pub.stdout)
    return "published", f"published {name}@{info.get('version')}"


def doctor():
    problems = []
    who = run(["npm", "whoami"])
    if who.returncode != 0:
        problems.append("npm whoami failed; run interactive npm login (not bypass-2FA GAT for trust)")
    else:
        print(f"npm whoami: {who.stdout.strip()}")

    ver = run(["npm", "--version"])
    print(f"npm version: {(ver.stdout or '').strip()}")
    help_proc = run(["npm", "trust", "--help"])
    if help_proc.returncode != 0:
        problems.append("npm trust unavailable; upgrade npm (>= 11.5.1 recommended)")

    print(f"trust contract: repo={TRUST_REPO} file={TRUST_FILE} env={TRUST_ENVIRONMENT}")
    if problems:
        for item in problems:
            print(f"doctor: {item}", file=sys.stderr)
        raise SystemExit(1)
    print("doctor: ok")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--module", help="Module name or path under modules/")
    parser.add_argument("--package-json", help="Path to a module package.json")
    parser.add_argument(
        "--apply",
        action="store_true",
        help="Perform bootstrap publish / trust bind (default is preview)",
    )
    parser.add_argument(
        "--rebind",
        action="store_true",
        help="Revoke conflicting trust configs before binding",
    )
    parser.add_argument(
        "--doctor",
        action="store_true",
        help="Check local npm login / CLI only",
    )
    args = parser.parse_args()

    if args.doctor:
        doctor()
        return

    modules = discover_modules(module=args.module, package_json=args.package_json)
    if not modules:
        raise SystemExit("no publishable modules found")

    mode = "apply" if args.apply else "preview"
    print(f"mode={mode} modules={len(modules)}")

    failures = 0
    for info in modules:
        name = info["name"]
        print(f"\n== {info['dir'].name} ({name}) ==")
        pub_status, pub_msg = bootstrap_publish(info, apply=args.apply)
        print(f"publish: {pub_status}: {pub_msg}")
        if pub_status == "error":
            failures += 1
            continue

        trust_status, trust_msg = bind_trust(name, apply=args.apply, rebind=args.rebind)
        print(f"trust: {trust_status}: {trust_msg}")
        if trust_status == "error":
            failures += 1

    if failures:
        raise SystemExit(f"completed with {failures} failure(s)")
    print("\ndone")


if __name__ == "__main__":
    main()
