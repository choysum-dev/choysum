#!/usr/bin/env python3
# SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
# SPDX-License-Identifier: LGPL-3.0-or-later
"""Summarize `go test -json` into package totals and slow tests.

Package elapsed is the `Action=pass|fail` event with no Test field.
Named sums include parent and subtests, so they can exceed package elapsed.
Exit 1 when any named test failed or a package/build-fail event is present.
"""

from __future__ import annotations

import argparse
import json
import sys
from collections import defaultdict
from pathlib import Path


def _short_pkg(pkg: str) -> str:
    if pkg.endswith(".test"):
        pkg = pkg[: -len(".test")]
    return pkg.rsplit("/", 1)[-1]


def summarize(jsonl_path: Path, *, slow_secs: float, pkg_top: int, test_top: int, timing_path: Path | None) -> int:
    by_pkg: dict[str, dict] = defaultdict(lambda: {"n": 0, "sum": 0.0, "pkg_elapsed": None, "status": "", "failed": []})
    slow: list[tuple[float, str, str, str]] = []
    parse_errors = 0
    events = 0

    with jsonl_path.open(encoding="utf-8", errors="replace") as fh:
        for line in fh:
            line = line.strip()
            if not line.startswith("{"):
                continue
            try:
                ev = json.loads(line)
            except json.JSONDecodeError:
                parse_errors += 1
                continue
            events += 1
            action = ev.get("Action")
            pkg = ev.get("Package") or ""
            test = ev.get("Test")
            elapsed = ev.get("Elapsed")
            if action in ("pass", "fail", "skip") and test and isinstance(elapsed, (int, float)):
                st = by_pkg[pkg]
                st["n"] += 1
                st["sum"] += float(elapsed)
                if action == "fail":
                    st["failed"].append(test)
                if elapsed >= slow_secs:
                    slow.append((float(elapsed), pkg, test, action))
            if action in ("pass", "fail") and not test and isinstance(elapsed, (int, float)):
                st = by_pkg[pkg]
                st["pkg_elapsed"] = float(elapsed)
                st["status"] = action
            elif action == "build-fail" and pkg:
                # Go 1.24+ emits compile failures as build-fail with no Elapsed.
                by_pkg[pkg]["status"] = "fail"

    if timing_path is not None and timing_path.exists():
        print("=== wall (POSIX time) ===")
        print(timing_path.read_text(encoding="utf-8", errors="replace").rstrip())
        print()

    print(f"=== package totals (top {pkg_top} by elapsed) ===")
    ranked = sorted(by_pkg.items(), key=lambda item: -(item[1].get("pkg_elapsed") or 0.0))
    if not ranked:
        print("(no package events)")
    for pkg, st in ranked[:pkg_top]:
        elapsed = st.get("pkg_elapsed")
        elapsed_s = f"{elapsed:6.1f}s" if elapsed is not None else "   n/a"
        fail_n = len(st["failed"])
        fail_s = f"  fail={fail_n}" if fail_n else ""
        status = st["status"] or "?"
        print(f"{elapsed_s}  tests={st['n']:4d}  named_sum={st['sum']:.1f}s  {status}{fail_s}  {pkg}")

    print()
    print(f"=== tests >= {slow_secs:g}s ===")
    slow_sorted = sorted(slow, reverse=True)
    if not slow_sorted:
        print("(none)")
    for elapsed, pkg, test, action in slow_sorted[:test_top]:
        print(f"{elapsed:6.2f}s  {action:4s}  {_short_pkg(pkg):20s}  {test}")
    if len(slow_sorted) > test_top:
        print(f"... {len(slow_sorted) - test_top} more")

    failed_rows = [(pkg, name) for pkg, st in by_pkg.items() for name in st["failed"]]
    package_failed = [pkg for pkg, st in by_pkg.items() if st["status"] == "fail"]
    print()
    print("=== failed tests ===")
    if not failed_rows and not package_failed:
        print("(none)")
    else:
        for pkg, name in failed_rows:
            print(f"  {pkg}  {name}")
        for pkg in package_failed:
            if not by_pkg[pkg]["failed"]:
                print(f"  {pkg}  (package)")

    print()
    print(f"json_events={events}  parse_errors={parse_errors}  packages={len(by_pkg)}  slow_tests={len(slow_sorted)}")
    return 1 if failed_rows or package_failed else 0


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("jsonl", type=Path, help="go test -json stream")
    parser.add_argument("--timing", type=Path, help="POSIX time -p output from /usr/bin/time")
    parser.add_argument("--slow", type=float, default=0.5, help="list tests at or above this many seconds")
    parser.add_argument("--pkg-top", type=int, default=20)
    parser.add_argument("--test-top", type=int, default=40)
    args = parser.parse_args()
    if not args.jsonl.exists():
        print(f"missing jsonl: {args.jsonl}", file=sys.stderr)
        return 2
    return summarize(
        args.jsonl,
        slow_secs=args.slow,
        pkg_top=args.pkg_top,
        test_top=args.test_top,
        timing_path=args.timing,
    )


if __name__ == "__main__":
    sys.exit(main())
