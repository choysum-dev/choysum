#!/usr/bin/env python3
# SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
# SPDX-License-Identifier: LGPL-3.0-or-later
"""Count production `as any` under modules/*/service (BT-8a / hardcut §3).

Scope: modules/<app>/service/**/*.ts
Production excludes: *.test.ts and **/tests/**

Buckets (overlapping labels for diagnostics; total is unique match count):
  super.     — match line contains "super."
  Search(    — match line contains "Search("
  $choysum   — match line contains "$choysum"
  other      — none of the above

Exit 0 on success. With --max N, exit 1 when production total exceeds N.
"""

from __future__ import annotations

import argparse
import re
import sys
from collections import defaultdict
from pathlib import Path

AS_ANY = re.compile(r"\bas any\b")
REPO_ROOT = Path(__file__).resolve().parents[2]


def is_production(path: Path, modules_root: Path) -> bool:
    if path.name.endswith((".test.ts", ".test.tsx")):
        return False
    try:
        rel_parts = path.relative_to(modules_root).parts
    except ValueError:
        rel_parts = path.parts
    if "tests" in rel_parts:
        return False
    return True


def iter_service_ts(modules_root: Path) -> list[Path]:
    out: list[Path] = []
    if not modules_root.is_dir():
        return out
    for app_dir in sorted(modules_root.iterdir()):
        service = app_dir / "service"
        if not service.is_dir():
            continue
        out.extend(sorted(service.rglob("*.ts")))
    return out


def classify_line(line: str) -> str:
    if "super." in line:
        return "super."
    if "Search(" in line:
        return "Search("
    if "$choysum" in line:
        return "$choysum"
    return "other"


def count_file(path: Path) -> tuple[int, dict[str, int]]:
    text = path.read_text(encoding="utf-8", errors="replace")
    buckets: dict[str, int] = defaultdict(int)
    total = 0
    for line in text.splitlines():
        n = len(AS_ANY.findall(line))
        if n == 0:
            continue
        total += n
        buckets[classify_line(line)] += n
    return total, buckets


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--modules",
        type=Path,
        default=REPO_ROOT / "modules",
        help="modules root (default: <repo>/modules)",
    )
    parser.add_argument(
        "--max",
        type=int,
        default=None,
        metavar="N",
        help="fail (exit 1) when production as-any total exceeds N",
    )
    parser.add_argument(
        "--top",
        type=int,
        default=20,
        help="number of top files to print (default: 20)",
    )
    parser.add_argument(
        "--include-tests",
        action="store_true",
        help="also count test files (reported separately)",
    )
    args = parser.parse_args(argv)

    modules_root: Path = args.modules.resolve()
    by_module: dict[str, int] = defaultdict(int)
    by_file: list[tuple[int, str]] = []
    bucket_totals: dict[str, int] = defaultdict(int)
    prod_total = 0
    prod_files = 0
    test_total = 0
    test_files = 0

    for path in iter_service_ts(modules_root):
        n, buckets = count_file(path)
        if n == 0:
            continue
        rel = path.relative_to(modules_root.parent) if modules_root.parent in path.parents else path
        rel_s = str(rel).replace("\\", "/")
        app = path.relative_to(modules_root).parts[0] if path.is_relative_to(modules_root) else "?"
        if is_production(path, modules_root):
            prod_total += n
            prod_files += 1
            by_module[app] += n
            by_file.append((n, rel_s))
            for k, v in buckets.items():
                bucket_totals[k] += v
        else:
            test_total += n
            test_files += 1

    print(f"production: {prod_total} as any in {prod_files} files")
    if args.include_tests:
        print(f"tests:      {test_total} as any in {test_files} files")
    print()
    print("by module:")
    for app, n in sorted(by_module.items(), key=lambda kv: (-kv[1], kv[0])):
        print(f"  {app:24} {n}")
    print()
    print("buckets (production; a line may match only one label):")
    for label in ("super.", "Search(", "$choysum", "other"):
        print(f"  {label:12} {bucket_totals.get(label, 0)}")
    print()
    print(f"top {args.top} files:")
    for n, rel in sorted(by_file, key=lambda x: (-x[0], x[1]))[: args.top]:
        print(f"  {n:4}  {rel}")

    if args.max is not None and prod_total > args.max:
        print(
            f"\nFAIL: production as any {prod_total} exceeds --max {args.max}",
            file=sys.stderr,
        )
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
