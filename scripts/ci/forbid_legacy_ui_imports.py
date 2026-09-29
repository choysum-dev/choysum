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

# Opening Element Plus tags in Vue templates (`<el-button`, `<el-select-v2`, …).
EL_TAG_RE = re.compile(r"<(el-[A-Za-z][\w-]*)\b")

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
    """Return (original_line_no, code) pairs with comments removed, string-aware."""
    lines_out: list[tuple[int, str]] = []
    in_block = False
    for i, line in enumerate(text.splitlines(), start=1):
        out: list[str] = []
        j = 0
        quote: str | None = None
        while j < len(line):
            ch = line[j]
            nxt = line[j + 1] if j + 1 < len(line) else ""
            if in_block:
                if ch == "*" and nxt == "/":
                    in_block = False
                    j += 2
                    continue
                j += 1
                continue
            if quote is not None:
                if ch == "\\" and j + 1 < len(line):
                    out.append(line[j : j + 2])
                    j += 2
                    continue
                if ch == quote:
                    quote = None
                out.append(ch)
                j += 1
                continue
            if ch in ("'", '"', "`"):
                quote = ch
                out.append(ch)
                j += 1
                continue
            if ch == "/" and nxt == "/":
                break
            if ch == "/" and nxt == "*":
                in_block = True
                j += 2
                continue
            out.append(ch)
            j += 1
        lines_out.append((i, "".join(out)))
    return lines_out


def iter_scan_files(modules_root: Path) -> list[Path]:
    out: list[Path] = []
    modules_root = modules_root.resolve()
    for path in modules_root.rglob("*"):
        if not path.is_file() or path.suffix not in SCAN_SUFFIXES:
            continue
        try:
            rel_parts = path.resolve().relative_to(modules_root).parts
        except ValueError:
            continue
        if any(part in SKIP_DIR_NAMES for part in rel_parts):
            continue
        out.append(path)
    return sorted(out)


def strip_html_comments(text: str) -> str:
    """Remove HTML/XML comments (non-greedy, DOTALL)."""
    return re.sub(r"<!--.*?-->", "", text, flags=re.DOTALL)


def extract_vue_template_body(text: str) -> str | None:
    """Return the first top-level <template>…</template> body, nesting-aware."""
    open_re = re.compile(r"<template\b[^>]*>", re.IGNORECASE)
    close_re = re.compile(r"</template\s*>", re.IGNORECASE)
    m = open_re.search(text)
    if not m:
        return None
    start = m.end()
    depth = 1
    pos = start
    while depth > 0:
        next_open = open_re.search(text, pos)
        next_close = close_re.search(text, pos)
        if not next_close:
            return text[start:]
        if next_open and next_open.start() < next_close.start():
            depth += 1
            pos = next_open.end()
            continue
        depth -= 1
        if depth == 0:
            return text[start : next_close.start()]
        pos = next_close.end()
    return None


def line_number_at(text: str, index: int) -> int:
    """1-based line number for a byte/char offset into text."""
    return text.count("\n", 0, index) + 1


def scan_vue_el_tags(path: Path, text: str) -> list[tuple[int, str, str]]:
    """Flag <el-*> opening tags inside the SFC template (not script stubs)."""
    body = extract_vue_template_body(text)
    if body is None:
        return []
    cleaned = strip_html_comments(body)
    # Map cleaned offsets back approximately via original body search of the tag text.
    hits: list[tuple[int, str, str]] = []
    # Locate the template body start in the original file for accurate line numbers.
    open_re = re.compile(r"<template\b[^>]*>", re.IGNORECASE)
    m = open_re.search(text)
    body_start = m.end() if m else 0
    for match in EL_TAG_RE.finditer(cleaned):
        tag = match.group(1)
        # Prefer the first occurrence of this exact open-tag snippet in the raw body.
        snippet = match.group(0)
        rel = body.find(snippet)
        abs_index = body_start + rel if rel >= 0 else body_start + match.start()
        hits.append((line_number_at(text, abs_index), tag, "el-tag"))
    return hits


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
    if path.suffix == ".vue":
        hits.extend(scan_vue_el_tags(path, text))
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
        "Legacy Element Plus / echarts / O*.vue imports and <el-*> template tags "
        "are banned after Choy UI Kit cutover. Import Choy* from @/web instead.",
        file=sys.stderr,
    )
    return 1


if __name__ == "__main__":
    raise SystemExit(main())
