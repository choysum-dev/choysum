#!/usr/bin/env python3
# SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
# SPDX-License-Identifier: LGPL-3.0-or-later
"""Forbid legacy UI imports after Choy UI Kit cutover.

Product modules under modules/ must not import Element Plus, echarts, or
legacy O*.vue / OV*.vue engines. Fixture / testdata trees may be allowlisted.

Exit 0 when clean; exit 1 and print violations otherwise.
Exit 2 when --modules is not a directory.
"""

from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[2]

SPEC_RE = re.compile(
    r"""(?:from|import|export)\s+(?:type\s+)?(?:\{[^}]*\}\s+from\s+|)\s*['"]([^'"]+)['"]"""
    r"""|import\s*\(\s*['"]([^'"]+)['"]"""
    r"""|require\s*\(\s*['"]([^'"]+)['"]"""
)

# Filename / path segments that still look like legacy engines.
O_VUE_RE = re.compile(r"(?:^|/)O[A-Z]\w+\.vue\b|(?:^|/)OV[A-Z]\w+\.vue\b")

SCAN_SUFFIXES = {".ts", ".tsx", ".vue", ".js", ".mjs", ".cjs"}

# Directory name segments skipped entirely (fixtures / generated / vendor-ish).
SKIP_DIR_NAMES = {
    "node_modules",
    ".git",
    "dist",
    "fixtures",
    "testdata",
    "__fixtures__",
}

FORBIDDEN_PACKAGE_PREFIXES = (
    "element-plus",
    "@element-plus/",
    "echarts",
    "vue-echarts",
)


def is_forbidden_package(spec: str) -> str | None:
    """Return a rule id when the import specifier is a banned package."""
    base = spec.split("?", 1)[0].strip()
    lower = base.lower()
    if lower == "element-plus" or lower.startswith("element-plus/"):
        return "element-plus"
    if lower == "@element-plus/icons-vue" or lower.startswith("@element-plus/"):
        return "element-plus"
    if lower == "echarts" or lower.startswith("echarts/"):
        return "echarts"
    if lower == "vue-echarts" or lower.startswith("vue-echarts/"):
        return "echarts"
    return None


def is_forbidden_o_vue(spec: str) -> bool:
    """True when the specifier path names a legacy O*.vue / OV*.vue file."""
    base = spec.split("?", 1)[0].strip()
    return O_VUE_RE.search(base) is not None


def strip_comments(text: str) -> list[tuple[int, str]]:
    """Return (original_line_no, code) pairs with // and /* */ removed."""
    lines_out: list[tuple[int, str]] = []
    in_block = False
    for i, line in enumerate(text.splitlines(), start=1):
        code = line
        if in_block:
            end = code.find("*/")
            if end == -1:
                continue
            code, in_block = code[end + 2 :], False
        while "/*" in code:
            start = code.find("/*")
            end = code.find("*/", start + 2)
            if end == -1:
                code, in_block = code[:start], True
                break
            code = code[:start] + code[end + 2 :]
        code = code.split("//", 1)[0]
        lines_out.append((i, code))
    return lines_out


def iter_scan_files(modules_root: Path) -> list[Path]:
    out: list[Path] = []
    for path in modules_root.rglob("*"):
        if not path.is_file() or path.suffix not in SCAN_SUFFIXES:
            continue
        if any(part in SKIP_DIR_NAMES for part in path.parts):
            continue
        out.append(path)
    return sorted(out)


def scan_file(path: Path) -> list[tuple[int, str, str]]:
    """Return (line, spec, rule) violations in one file."""
    text = path.read_text(encoding="utf-8", errors="replace")
    hits: list[tuple[int, str, str]] = []
    for line_no, code in strip_comments(text):
        for m in SPEC_RE.finditer(code):
            spec = next(g for g in m.groups() if g)
            rule = is_forbidden_package(spec)
            if rule:
                hits.append((line_no, spec, rule))
                continue
            if is_forbidden_o_vue(spec):
                hits.append((line_no, spec, "o-star-vue"))
    return hits


def scan(modules_root: Path) -> list[tuple[Path, int, str, str]]:
    violations: list[tuple[Path, int, str, str]] = []
    for path in iter_scan_files(modules_root):
        for line, spec, rule in scan_file(path):
            violations.append((path, line, spec, rule))
    return violations


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--modules",
        type=Path,
        default=REPO_ROOT / "modules",
        help="modules root (default: <repo>/modules)",
    )
    args = parser.parse_args(argv)

    modules_root = args.modules.resolve()
    if not modules_root.is_dir():
        print(
            f"forbid_legacy_ui_imports: modules root not found: {modules_root}",
            file=sys.stderr,
        )
        return 2

    violations = scan(modules_root)
    if not violations:
        print("forbid_legacy_ui_imports: ok (0 violations)")
        return 0

    print(f"forbid_legacy_ui_imports: {len(violations)} violation(s)", file=sys.stderr)
    for path, line, spec, rule in violations:
        try:
            rel = path.relative_to(REPO_ROOT)
        except ValueError:
            rel = path
        print(f"  {rel}:{line}: [{rule}] {spec}", file=sys.stderr)
    print(
        "Legacy Element Plus / echarts / O*.vue imports are banned after Choy UI Kit cutover. "
        "Import Choy* from @/web instead.",
        file=sys.stderr,
    )
    return 1


if __name__ == "__main__":
    raise SystemExit(main())
