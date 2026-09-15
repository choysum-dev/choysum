#!/usr/bin/env python3
# SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
# SPDX-License-Identifier: LGPL-3.0-or-later
"""Partition `go list ./...` into CI go-test shards with union/overlap checks.

Shards (intent):
  cmd      ./cmd/... ./internal/cli/...
  testing  ./internal/testing/...          (needs Chromium)
  module   ./internal/module/...
  runtime  ./internal/server/... ./internal/esmresolver/...
           ./internal/typecheck/... ./internal/defaultengine/...
           ./internal/defaultjsexecutor/...
  rest     explicit difference of go list ./... minus the above
"""

from __future__ import annotations

import argparse
import json
import pathlib
import subprocess
from typing import Iterable

REPO_ROOT = pathlib.Path(__file__).resolve().parents[2]

# Named shards with go-list patterns. Order matters only for display.
# `rest` is computed as the complement and must not appear here.
SHARD_SPECS: list[dict[str, object]] = [
    {
        "name": "cmd",
        "patterns": ["./cmd/...", "./internal/cli/..."],
        "chromium": False,
    },
    {
        "name": "testing",
        "patterns": ["./internal/testing/..."],
        "chromium": True,
    },
    {
        "name": "module",
        "patterns": ["./internal/module/..."],
        "chromium": False,
    },
    {
        "name": "runtime",
        "patterns": [
            "./internal/server/...",
            "./internal/esmresolver/...",
            "./internal/typecheck/...",
            "./internal/defaultengine/...",
            "./internal/defaultjsexecutor/...",
        ],
        "chromium": False,
    },
]


def run_go_list(patterns: Iterable[str], *, cwd: pathlib.Path = REPO_ROOT) -> list[str]:
    cmd = ["go", "list", *patterns]
    proc = subprocess.run(
        cmd,
        cwd=cwd,
        check=False,
        capture_output=True,
        text=True,
    )
    if proc.returncode != 0:
        raise SystemExit(
            f"go list failed ({proc.returncode}): {' '.join(cmd)}\n{proc.stderr.strip()}"
        )
    return [line.strip() for line in proc.stdout.splitlines() if line.strip()]


def partition_packages(cwd: pathlib.Path = REPO_ROOT) -> dict[str, list[str]]:
    all_pkgs = run_go_list(["./..."], cwd=cwd)
    all_set = set(all_pkgs)
    assigned: set[str] = set()
    shards: dict[str, list[str]] = {}

    for spec in SHARD_SPECS:
        name = str(spec["name"])
        patterns = list(spec["patterns"])  # type: ignore[arg-type]
        pkgs = set(run_go_list(patterns, cwd=cwd))
        if not pkgs:
            raise SystemExit(f"shard {name}: patterns matched no packages: {patterns}")
        unknown = pkgs - all_set
        if unknown:
            raise SystemExit(f"shard {name}: packages not in go list ./...: {sorted(unknown)}")
        overlap = pkgs & assigned
        if overlap:
            raise SystemExit(f"shard {name}: overlaps prior shards: {sorted(overlap)}")
        shards[name] = sorted(pkgs)
        assigned |= pkgs

    rest = sorted(all_set - assigned)
    if not rest:
        raise SystemExit(
            "shard rest: complement is empty; every package is covered by a named shard"
        )
    shards["rest"] = rest
    return shards


def shard_meta() -> list[dict[str, object]]:
    rows: list[dict[str, object]] = []
    for spec in SHARD_SPECS:
        rows.append(
            {
                "shard": spec["name"],
                "chromium": "true" if spec["chromium"] else "false",
            }
        )
    rows.append({"shard": "rest", "chromium": "false"})
    return rows


def coverprofile_basenames() -> list[str]:
    return [f"shard-{row['shard']}.out" for row in shard_meta()]


def check_partition(cwd: pathlib.Path = REPO_ROOT) -> dict[str, list[str]]:
    shards = partition_packages(cwd=cwd)
    all_pkgs = run_go_list(["./..."], cwd=cwd)
    union: list[str] = []
    seen: set[str] = set()
    for name, pkgs in shards.items():
        for pkg in pkgs:
            if pkg in seen:
                raise SystemExit(f"duplicate package across shards: {pkg} (in {name})")
            seen.add(pkg)
            union.append(pkg)
    if set(union) != set(all_pkgs):
        missing = sorted(set(all_pkgs) - set(union))
        extra = sorted(set(union) - set(all_pkgs))
        raise SystemExit(f"union mismatch: missing={missing} extra={extra}")
    return shards


def merge_coverprofiles(
    inputs: list[pathlib.Path],
    output: pathlib.Path,
    *,
    require_all_shards: bool = False,
) -> None:
    if not inputs:
        raise SystemExit("merge-coverprofiles: no input files")
    if require_all_shards:
        got = sorted(path.name for path in inputs)
        expected = sorted(coverprofile_basenames())
        if got != expected:
            raise SystemExit(
                "merge-coverprofiles: shard set mismatch: "
                f"got={got} expected={expected}"
            )
    mode: str | None = None
    body: list[str] = []
    for path in inputs:
        text = path.read_text(encoding="utf-8")
        lines = text.splitlines()
        if not lines or not lines[0].startswith("mode:"):
            raise SystemExit(f"invalid coverprofile (missing mode): {path}")
        parts = lines[0].split(None, 1)
        if len(parts) < 2 or not parts[1].strip():
            raise SystemExit(f"invalid coverprofile (empty mode): {path}")
        file_mode = parts[1].strip()
        if mode is None:
            mode = file_mode
        elif mode != file_mode:
            raise SystemExit(f"cover mode mismatch: {mode} vs {file_mode} ({path})")
        body.extend(line for line in lines[1:] if line.strip())
    output.parent.mkdir(parents=True, exist_ok=True)
    output.write_text(
        "mode: " + (mode or "set") + "\n" + "\n".join(body) + ("\n" if body else ""),
        encoding="utf-8",
    )


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest="command", required=True)

    sub.add_parser("check", help="Verify shard union equals go list ./... with no overlap")
    sub.add_parser("matrix", help="Print GitHub Actions matrix JSON to stdout")
    sub.add_parser(
        "coverprofile-names",
        help="Print expected shard coverprofile basenames (one per line)",
    )

    packages = sub.add_parser("packages", help="Print packages for one shard (one per line)")
    packages.add_argument("shard", help="Shard name (cmd|testing|module|runtime|rest)")

    merge = sub.add_parser("merge-coverprofiles", help="Merge go coverprofiles into one file")
    merge.add_argument("--output", "-o", required=True, type=pathlib.Path)
    merge.add_argument(
        "--require-all-shards",
        action="store_true",
        help="Require input basenames to match coverprofile-names exactly",
    )
    merge.add_argument("inputs", nargs="+", type=pathlib.Path)

    return parser


def main(argv: list[str] | None = None) -> int:
    parser = build_parser()
    args = parser.parse_args(argv)

    if args.command == "check":
        shards = check_partition()
        for name, pkgs in shards.items():
            print(f"{name}: {len(pkgs)} packages")
        print(f"total: {sum(len(p) for p in shards.values())} packages")
        return 0

    if args.command == "matrix":
        print(json.dumps(shard_meta(), separators=(",", ":")))
        return 0

    if args.command == "coverprofile-names":
        for name in coverprofile_basenames():
            print(name)
        return 0

    if args.command == "packages":
        shards = partition_packages()
        if args.shard not in shards:
            raise SystemExit(f"unknown shard: {args.shard}; want one of {sorted(shards)}")
        for pkg in shards[args.shard]:
            print(pkg)
        return 0

    if args.command == "merge-coverprofiles":
        merge_coverprofiles(
            args.inputs,
            args.output,
            require_all_shards=args.require_all_shards,
        )
        print(f"merged {len(args.inputs)} profiles -> {args.output}")
        return 0

    parser.error(f"unknown command: {args.command}")
    return 2


if __name__ == "__main__":
    raise SystemExit(main())
