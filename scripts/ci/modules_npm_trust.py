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
import re
import subprocess
import sys
import urllib.parse


REPO_ROOT = pathlib.Path(__file__).resolve().parents[2]
MODULES_ROOT = REPO_ROOT / "modules"

TRUST_REPO = "choysum-dev/choysum"
TRUST_FILE = "modules-publish.yml"
TRUST_ENVIRONMENT = "npm-publish"
MIN_NPM_VERSION = (11, 15, 0)
EXPECTED_REPO_HOST = "github.com"
EXPECTED_REPO_PATH = "choysum-dev/choysum"


def run(cmd, *, cwd=None, check=False):
    return subprocess.run(
        cmd,
        cwd=str(cwd) if cwd else None,
        check=check,
        capture_output=True,
        text=True,
    )


def compact(text, limit=2000):
    value = (text or "").strip()
    if len(value) <= limit:
        return value
    return value[-limit:]


def parse_semver_tuple(value: str):
    text = (value or "").strip()
    match = re.match(r"^(\d+)\.(\d+)\.(\d+)", text)
    if not match:
        return None
    return tuple(int(part) for part in match.groups())


def normalize_github_repository(url: str):
    """Return (host, owner/repo) for supported GitHub URL forms, else None."""
    text = (url or "").strip()
    if not text:
        return None

    lowered = text.lower()
    if lowered.startswith("git+"):
        text = text[4:]
        lowered = text.lower()

    # git@github.com:owner/repo(.git)
    scp = re.match(r"^git@([^:]+):(.+)$", text)
    if scp:
        host = scp.group(1).lower()
        path = scp.group(2)
    elif "://" in text:
        parsed = urllib.parse.urlparse(text)
        host = (parsed.hostname or "").lower()
        path = parsed.path or ""
    elif lowered.startswith("github.com/"):
        host = "github.com"
        path = text.split("/", 1)[1]
    else:
        return None

    path = path.strip("/")
    if path.endswith(".git"):
        path = path[: -len(".git")]
    parts = [part for part in path.split("/") if part]
    if len(parts) < 2:
        return None
    owner_repo = f"{parts[0]}/{parts[1]}".lower()
    return host, owner_repo


def repository_identifies_choysum(repository) -> bool:
    if isinstance(repository, str):
        url = repository
    elif isinstance(repository, dict):
        url = str(repository.get("url") or "")
    else:
        return False
    normalized = normalize_github_repository(url)
    if normalized is None:
        return False
    host, owner_repo = normalized
    return host == EXPECTED_REPO_HOST and owner_repo == EXPECTED_REPO_PATH


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
    repository = data.get("repository")
    if not repository_identifies_choysum(repository):
        return None, (
            f"{package_path}: repository.url must identify {TRUST_REPO} "
            "(required for npm Trusted Publishing / provenance)"
        )
    return {"name": name, "version": version, "path": package_path, "dir": module_dir}, None


def discover_modules(module=None, package_json=None):
    if package_json:
        path = pathlib.Path(package_json).resolve()
        if path.name != "package.json":
            raise SystemExit(f"--package-json must be named package.json, got: {path.name}")
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
        # Some CLI builds emit one JSON object per line.
        entries = []
        for line in raw.splitlines():
            line = line.strip()
            if not line:
                continue
            try:
                entries.append(json.loads(line))
            except json.JSONDecodeError:
                return None, f"invalid trust list JSON for {name}: {raw[:500]}"
        return entries, None
    if data is None:
        return [], None
    if isinstance(data, dict):
        if "configurations" in data and isinstance(data["configurations"], list):
            return data["configurations"], None
        if "packages" in data and isinstance(data["packages"], list):
            return data["packages"], None
        return [data], None
    if isinstance(data, list):
        return data, None
    return None, f"unexpected trust list payload type for {name}: {type(data)}"


def _collect_values(obj, wanted_keys: set[str]):
    found = []
    if isinstance(obj, dict):
        for key, value in obj.items():
            if str(key).lower().replace("-", "").replace("_", "") in wanted_keys:
                if isinstance(value, (str, int, bool)):
                    found.append(value)
                elif isinstance(value, list):
                    found.extend(item for item in value if isinstance(item, (str, int, bool, dict)))
                elif isinstance(value, dict):
                    found.append(value)
            found.extend(_collect_values(value, wanted_keys))
    elif isinstance(obj, list):
        for item in obj:
            found.extend(_collect_values(item, wanted_keys))
    return found


def _norm(value) -> str:
    return str(value).strip().lower()


def _allows_npm_publish(entry: dict) -> bool:
    flags = _collect_values(
        entry,
        {
            "allowpublish",
            "allowstagepublish",
            "permissions",
            "allowedactions",
            "actions",
            "allowedaction",
        },
    )
    blob = json.dumps(entry).lower()
    has_publish = False
    stage_only = False
    publish_values = {
        "publish",
        "npm publish",
        "allow-publish",
        "allowpublish",
        "createpackage",
    }
    stage_values = {
        "stage",
        "stage publish",
        "npm stage publish",
        "allow-stage-publish",
        "allowstagepublish",
        "createstagedpackage",
    }
    for flag in flags:
        text = _norm(flag).replace("_", "").replace("-", "")
        # Keep hyphenated forms too via original _norm(flag).
        raw = _norm(flag)
        if raw in {"true", "1", "yes"}:
            continue
        if raw in publish_values or text in {"createpackage", "allowpublish"}:
            has_publish = True
        if raw in stage_values or text in {"createstagedpackage", "allowstagepublish"}:
            stage_only = True
        if "stage" in raw and "publish" in raw and "allow-publish" not in raw and "createpackage" not in text:
            stage_only = True
        if isinstance(flag, dict):
            nested = json.dumps(flag).lower()
            if "allow-publish" in nested or "createpackage" in nested.replace("_", "").replace("-", ""):
                has_publish = True
            if "createstagedpackage" in nested.replace("_", "").replace("-", ""):
                stage_only = True
    compact_blob = blob.replace(" ", "").replace("_", "").replace("-", "")
    if "allow-publish" in blob or "createpackage" in compact_blob or '"allowpublish":true' in compact_blob:
        has_publish = True
    if (
        ("allow-stage-publish" in blob or "createstagedpackage" in compact_blob)
        and "allow-publish" not in blob
        and "createpackage" not in compact_blob
    ):
        stage_only = True
    if has_publish:
        return True
    if stage_only:
        return False
    # Unknown schema: do not treat as publish-capable.
    return False


def trust_matches(entry: dict, *, package_name: str) -> bool:
    if not isinstance(entry, dict):
        return False

    repos = [_norm(v) for v in _collect_values(entry, {"repository", "repo", "repositoryname", "ownerrepo"})]
    files = [
        _norm(v)
        for v in _collect_values(
            entry,
            {"file", "workflow", "workflowfile", "workflowfilename", "filename"},
        )
    ]
    envs = [_norm(v) for v in _collect_values(entry, {"environment", "env", "environmentname"})]
    providers = [_norm(v) for v in _collect_values(entry, {"provider", "oidcprovider", "type"})]
    packages = [_norm(v) for v in _collect_values(entry, {"package", "pkg", "name", "packagename"})]

    repo_ok = any(
        TRUST_REPO.lower() == value
        or value.endswith("/" + TRUST_REPO.lower())
        or TRUST_REPO.lower() in value
        for value in repos
    ) or (
        # Fallback when list payload only embeds repo in nested URL strings.
        any(TRUST_REPO.lower() in _norm(v) for v in repos)
    )
    if not repo_ok:
        return False

    file_ok = any(value == TRUST_FILE.lower() or value.endswith("/" + TRUST_FILE.lower()) for value in files)
    if not file_ok:
        return False

    env_ok = any(value == TRUST_ENVIRONMENT.lower() for value in envs)
    if not env_ok:
        return False

    if providers:
        provider_ok = any("github" in value for value in providers)
        if not provider_ok:
            return False

    if packages:
        pkg = package_name.lower()
        package_ok = any(value == pkg or value.endswith(pkg) for value in packages)
        if not package_ok:
            return False

    if not _allows_npm_publish(entry):
        return False

    return True


def bind_trust(name: str, *, apply: bool, rebind: bool):
    entries, err = trust_list(name)
    if err:
        # Interactive auth prompts often land here; surface stderr.
        return "error", err

    matching = [e for e in (entries or []) if trust_matches(e, package_name=name)]
    others = [e for e in (entries or []) if e not in matching]

    if matching and not others:
        return "skip", "trust already matches publish-capable contract"
    if matching and others and not rebind:
        return "error", "mixed trust configs; re-run with --rebind to replace"
    if others and not matching and not rebind:
        return "error", (
            "existing trust config does not match publish-capable contract; "
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
    version_text = (ver.stdout or "").strip()
    print(f"npm version: {version_text}")
    parsed = parse_semver_tuple(version_text)
    if parsed is None or parsed < MIN_NPM_VERSION:
        required = ".".join(str(part) for part in MIN_NPM_VERSION)
        problems.append(f"npm {required}+ required for npm trust (got {version_text or 'unknown'})")

    help_proc = run(["npm", "trust", "--help"])
    if help_proc.returncode != 0:
        required = ".".join(str(part) for part in MIN_NPM_VERSION)
        problems.append(f"npm trust unavailable; upgrade npm (>={required})")

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
