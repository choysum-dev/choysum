#!/usr/bin/env python3
# SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
# SPDX-License-Identifier: LGPL-3.0-or-later
"""Forbid non-core imports of core engine deep paths (SF-7 / TS7 equivalent).

Spike (2026-09-20): Choysum resolves `@/*` via modules/tsconfig paths and does
**not** consult package.json `exports`. This script is the in-tree gate that
locks SF-1's engine-barrel retreat (direction §3.2) for non-core modules.

Forbidden (prefix match on import specifiers):
  @/core/service/orm/repository/types
  @/core/service/orm/repository/{query,read,write,authz,projection,validation}
  @/core/service/orm/relation

Allowed:
  @/core/service/orm/repository   (exact; SF-D Factory / Repository / db)

Scans modules/<app>/**/*.{ts,tsx} excluding modules/core/**.
Also resolves relative imports that traverse into modules/core/...
Exit 0 when clean; exit 1 and print violations otherwise.
Exit 2 when --modules is not a directory.
"""

from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[2]

# Import / export / dynamic import / require string literals.
SPEC_RE = re.compile(
    r"""(?:from|import|export)\s+(?:type\s+)?(?:\{[^}]*\}\s+from\s+|)\s*['"]([^'"]+)['"]"""
    r"""|import\s*\(\s*['"]([^'"]+)['"]"""
    r"""|require\s*\(\s*['"]([^'"]+)['"]"""
)

MODULE_SUFFIXES = (".tsx", ".ts", ".mjs", ".cjs", ".js", ".json")

FORBIDDEN_PREFIXES = (
    "@/core/service/orm/repository/types",
    "@/core/service/orm/repository/query",
    "@/core/service/orm/repository/read",
    "@/core/service/orm/repository/write",
    "@/core/service/orm/repository/authz",
    "@/core/service/orm/repository/projection",
    "@/core/service/orm/repository/validation",
    "@/core/service/orm/relation",
)

# Also catch published-package style deep imports if they appear.
FORBIDDEN_PREFIXES_NPM = tuple(
    p.replace("@/core/", "@choysum-dev/core/") for p in FORBIDDEN_PREFIXES
)


def strip_module_suffix(spec: str) -> str:
    for suffix in MODULE_SUFFIXES:
        if spec.endswith(suffix):
            return spec[: -len(suffix)]
    return spec


def is_forbidden(spec: str) -> bool:
    normalized = strip_module_suffix(spec)
    for prefix in FORBIDDEN_PREFIXES + FORBIDDEN_PREFIXES_NPM:
        if normalized == prefix or normalized.startswith(prefix + "/"):
            return True
    return False


def alias_spec_for_relative(path: Path, spec: str, modules_root: Path) -> str:
    """Map a relative import into `@/<app>/...` when it lands under modules_root."""
    if not spec.startswith("."):
        return spec
    try:
        resolved = (path.parent / spec).resolve()
        rel = resolved.relative_to(modules_root.resolve())
    except (OSError, ValueError):
        return spec
    return "@/" + rel.as_posix()


def iter_non_core_ts(modules_root: Path) -> list[Path]:
    out: list[Path] = []
    for app_dir in sorted(modules_root.iterdir()):
        if not app_dir.is_dir() or app_dir.name == "core":
            continue
        out.extend(sorted(p for p in app_dir.rglob("*") if p.suffix in {".ts", ".tsx"}))
    return out


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


def scan_file(path: Path, modules_root: Path) -> list[tuple[int, str]]:
    text = path.read_text(encoding="utf-8", errors="replace")
    hits: list[tuple[int, str]] = []
    for line_no, code in strip_comments(text):
        for m in SPEC_RE.finditer(code):
            spec = next(g for g in m.groups() if g)
            target = alias_spec_for_relative(path, spec, modules_root)
            if is_forbidden(target):
                hits.append((line_no, spec))
    return hits


def scan(modules_root: Path) -> list[tuple[Path, int, str]]:
    violations: list[tuple[Path, int, str]] = []
    for path in iter_non_core_ts(modules_root):
        for line, spec in scan_file(path, modules_root):
            violations.append((path, line, spec))
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
            f"forbid_core_engine_imports: modules root not found: {modules_root}",
            file=sys.stderr,
        )
        return 2

    violations = scan(modules_root)
    if not violations:
        print("forbid_core_engine_imports: ok (0 violations)")
        return 0

    print(f"forbid_core_engine_imports: {len(violations)} violation(s)", file=sys.stderr)
    for path, line, spec in violations:
        try:
            rel = path.relative_to(REPO_ROOT)
        except ValueError:
            rel = path
        print(f"  {rel}:{line}: {spec}", file=sys.stderr)
    print(
        "Use @/core/service/api/* (or exact @/core/service/orm/repository for Factory). "
        "See SF-7 / TS7 in core-type-surface-hardcut-plan.",
        file=sys.stderr,
    )
    return 1


if __name__ == "__main__":
    raise SystemExit(main())
