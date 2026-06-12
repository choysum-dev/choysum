#!/usr/bin/env python3
# SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
# SPDX-License-Identifier: LGPL-3.0-or-later

from __future__ import annotations

import json
import os
import pathlib
import subprocess
import sys
from collections import deque
from pathlib import PurePosixPath


REPO_ROOT = pathlib.Path(__file__).resolve().parents[2]
MODULES_ROOT = REPO_ROOT / "modules"
IGNORED_MODULES = {".choysum", "tmp"}
HIGH_FANOUT_MODULES = {"core", "web", "base"}
BUILD_PREFIXES = (
    "internal/bootstrap/web/",
    "pkg/jsengine/scripts/vuesfc/",
)
SHARED_PREFIXES = (
    "cmd/",
    "internal/",
    "pkg/",
    ".github/workflows/",
)
SHARED_EXACT = {
    "main.go",
    "main_test.go",
    "go.mod",
    "package.json",
    "package-lock.json",
    "codecov.yml",
    "config.sample.yaml",
}


def sorted_json(values):
    return json.dumps(sorted(values), separators=(",", ":"))


def list_modules():
    if not MODULES_ROOT.is_dir():
        return []
    return sorted(
        child.name
        for child in MODULES_ROOT.iterdir()
        if child.is_dir() and child.name not in IGNORED_MODULES
    )


def has_smoke_spec(module):
    return (MODULES_ROOT / module / "e2e" / "smoke.spec.ts").is_file()


def classify_module_path(path, module_names):
    pure = PurePosixPath(path)
    parts = pure.parts
    if len(parts) < 3 or parts[0] != "modules":
        return None, None

    module = parts[1]
    if module in IGNORED_MODULES or module not in module_names:
        return None, None

    rel = "/".join(parts[2:])
    name = PurePosixPath(rel).name.lower()
    lowered = rel.lower()

    if name.endswith(".md") or name.startswith("readme"):
        return module, "doc"
    if lowered.endswith((".test.ts", ".test.tsx", ".spec.ts", ".spec.tsx")):
        return module, "test"
    if rel.startswith("web/"):
        return module, "local"
    if rel.startswith("service/tests/"):
        return module, "local"
    if rel.startswith("e2e/"):
        return module, "local"

    return module, "contract"


def is_doc_only_path(path):
    pure = PurePosixPath(path)
    name = pure.name.lower()
    if name == "license":
        return True
    if path.startswith("cla/"):
        return True
    if name.endswith(".md"):
        return True
    if name.startswith("readme"):
        return True
    return False


def is_build_pipeline_path(path):
    return any(path.startswith(prefix) for prefix in BUILD_PREFIXES)


def is_shared_path(path):
    return path in SHARED_EXACT or any(path.startswith(prefix) for prefix in SHARED_PREFIXES)


def build_reverse_graph(modules):
    reverse_graph = {module: set() for module in modules}
    for module in modules:
        package_path = MODULES_ROOT / module / "package.json"
        if not package_path.is_file():
            continue

        try:
            with package_path.open("r", encoding="utf-8") as fh:
                package_json = json.load(fh)
        except (json.JSONDecodeError, OSError):
            continue

        choysum = package_json.get("choysum") if isinstance(package_json, dict) else {}
        depends = choysum.get("depends") if isinstance(choysum, dict) else []
        if not isinstance(depends, list):
            continue

        for dep in depends:
            dep_name = str(dep).strip()
            if dep_name in reverse_graph:
                reverse_graph[dep_name].add(module)

    return reverse_graph


def reverse_closure(reverse_graph, start):
    seen = {start}
    queue = deque([start])
    while queue:
        current = queue.popleft()
        for nxt in reverse_graph.get(current, set()):
            if nxt in seen:
                continue
            seen.add(nxt)
            queue.append(nxt)
    seen.discard(start)
    return seen


def merge_group_or_dispatch_outputs(modules, reason):
    impacted_modules = set(modules)
    smoke_modules = {module for module in impacted_modules if has_smoke_spec(module)}
    return {
        "docs_only": "false",
        "direct_modules_json": "[]",
        "impacted_direct_e2e_no_smoke_modules_json": "[]",
        "impacted_modules_json": sorted_json(impacted_modules),
        "impacted_smoke_e2e_modules_json": sorted_json(smoke_modules),
        "run_full_matrix": "true",
        "run_bootstrap_verify": "true",
        "run_pr_smoke_e2e": "true" if smoke_modules else "false",
        "reason": reason,
    }


def pull_request_outputs(modules):
    module_names = set(modules)
    reverse_graph = build_reverse_graph(modules)

    base_sha = os.environ.get("PR_BASE_SHA", "").strip()
    head_sha = os.environ.get("PR_HEAD_SHA", "").strip()
    if not base_sha or not head_sha:
        raise SystemExit("discover-impact: pull_request event missing base/head sha")

    try:
        diff = subprocess.run(
            ["git", "diff", "--name-only", f"{base_sha}...{head_sha}"],
            check=True,
            capture_output=True,
            text=True,
        )
        changed_paths = [line.strip() for line in diff.stdout.splitlines() if line.strip()]
        fallback_full = False
    except subprocess.CalledProcessError:
        changed_paths = []
        fallback_full = True

    direct_modules = set()
    direct_e2e_modules = set()
    impacted_modules = set()
    contract_modules = set()
    local_modules = set()
    build_hit = False
    shared_hit = fallback_full
    docs_only = not fallback_full

    for path in changed_paths:
        module, kind = classify_module_path(path, module_names)
        if module is not None:
            direct_modules.add(module)
            if kind == "doc":
                continue
            docs_only = False
            impacted_modules.add(module)
            if path.startswith(f"modules/{module}/e2e/"):
                direct_e2e_modules.add(module)
            if kind == "contract":
                contract_modules.add(module)
            else:
                local_modules.add(module)
            continue

        if is_doc_only_path(path):
            continue

        docs_only = False
        if is_build_pipeline_path(path):
            build_hit = True
            shared_hit = True
            continue
        if is_shared_path(path):
            shared_hit = True
            continue

        shared_hit = True

    final_impacted = set()
    if docs_only:
        run_full = False
        run_bootstrap = False
        reason = "docs-only"
    else:
        final_impacted.update(impacted_modules)
        for module in sorted(contract_modules):
            final_impacted.update(reverse_closure(reverse_graph, module))

        if build_hit:
            run_full = True
            run_bootstrap = True
            reason = "build-pipeline"
        elif shared_hit:
            run_full = True
            run_bootstrap = False
            reason = "shared-runtime"
        elif contract_modules & HIGH_FANOUT_MODULES:
            run_full = True
            run_bootstrap = False
            reason = f"fanout-module:{sorted(contract_modules & HIGH_FANOUT_MODULES)[0]}"
        elif len(final_impacted) > 4:
            run_full = True
            run_bootstrap = False
            reason = f"fanout-threshold:{len(final_impacted)}"
        else:
            run_full = False
            run_bootstrap = False
            if contract_modules:
                reason = f"module-contract:{sorted(contract_modules)[0]}"
            elif local_modules:
                reason = f"module-local:{sorted(local_modules)[0]}"
            else:
                reason = "no-impactable-tests"

        if run_full:
            final_impacted = set(modules)

    smoke_modules = {module for module in final_impacted if has_smoke_spec(module)}
    direct_e2e_no_smoke_modules = {
        module for module in direct_e2e_modules if module in final_impacted and not has_smoke_spec(module)
    }
    outputs = {
        "docs_only": "true" if docs_only else "false",
        "direct_modules_json": sorted_json(direct_modules),
        "impacted_direct_e2e_no_smoke_modules_json": sorted_json(direct_e2e_no_smoke_modules),
        "impacted_modules_json": sorted_json(final_impacted),
        "impacted_smoke_e2e_modules_json": sorted_json(smoke_modules),
        "run_full_matrix": "true" if run_full else "false",
        "run_bootstrap_verify": "true" if run_bootstrap else "false",
        "run_pr_smoke_e2e": "true" if smoke_modules else "false",
        "reason": reason,
    }

    print("discover-impact: changed files")
    for path in changed_paths:
        print(f" - {path}")
    print("discover-impact: outputs")
    for key in sorted(outputs):
        print(f"   {key}={outputs[key]}")

    return outputs


def main():
    modules = list_modules()
    event_name = os.environ.get("GITHUB_EVENT_NAME", "").strip()

    if event_name == "merge_group":
        outputs = merge_group_or_dispatch_outputs(modules, "merge-group")
        print("discover-impact: merge_group -> full matrix")
    elif event_name != "pull_request":
        outputs = merge_group_or_dispatch_outputs(modules, "workflow-dispatch")
        print("discover-impact: workflow_dispatch -> full matrix")
    else:
        outputs = pull_request_outputs(modules)

    github_output = os.environ.get("GITHUB_OUTPUT", "").strip()
    if not github_output:
        raise SystemExit("GITHUB_OUTPUT is required")

    with open(github_output, "a", encoding="utf-8") as fh:
        for key, value in outputs.items():
            print(f"{key}={value}", file=fh)


if __name__ == "__main__":
    main()
