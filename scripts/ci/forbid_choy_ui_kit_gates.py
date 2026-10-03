#!/usr/bin/env python3
# SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
# SPDX-License-Identifier: LGPL-3.0-or-later
"""W4 Choy UI kit gates: domain L2/engine imports, L2 density regression, kit-host Dialog path.

- Domain modules (not web) must not import vendor/ui, reka-ui, or @unovis/*.
- vendor/ui product SFCs must not regress default control heights to h-8/h-9/h-10.
- Import / Search / Field / Command palette must import Dialog via layout/choyDialog.

Exit 0 when clean; exit 1 on violations; exit 2 when --modules is not a directory.
"""

from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[2]

SPEC_RE = re.compile(
    r"""(?:from|import|export)\s+(?:type\s+)?(?:\{[^}]*\}\s+from\s+|\*\s*(?:as\s+\w+\s+)?from\s+|)\s*['"]([^'"]+)['"]"""
    r"""|import\s*\(\s*['"]([^'"]+)['"]"""
    r"""|require\s*\(\s*['"]([^'"]+)['"]"""
)

HEIGHT_RE = re.compile(r"(?:^|[^A-Za-z0-9_-])h-(?:8|9|10)(?:$|[^A-Za-z0-9_-])")

SCAN_SUFFIXES = {".ts", ".tsx", ".vue", ".js", ".mjs", ".cjs"}

SKIP_DIR_NAMES = {
    "node_modules",
    ".git",
    "dist",
    "fixtures",
    "testdata",
    "__fixtures__",
}

KIT_HOST_DIALOG_PREFIXES = (
    "web/web/import/",
    "web/web/export/",
    "web/web/components/field/",
    "web/web/components/view/search/",
    "web/web/components/layout/ChoyCommandPalette.vue",
)

DIALOG_BARREL = "web/web/components/layout/choyDialog.ts"


def is_skip_dir(path: Path) -> bool:
    return any(part in SKIP_DIR_NAMES for part in path.parts)


def iter_files(root: Path) -> list[Path]:
    out: list[Path] = []
    if not root.is_dir():
        return out
    for path in root.rglob("*"):
        if not path.is_file() or path.suffix not in SCAN_SUFFIXES:
            continue
        if is_skip_dir(path):
            continue
        out.append(path)
    return out


def specs_in(text: str) -> list[tuple[int, str]]:
    hits: list[tuple[int, str]] = []
    for i, line in enumerate(text.splitlines(), start=1):
        for m in SPEC_RE.finditer(line):
            spec = next(g for g in m.groups() if g)
            hits.append((i, spec.split("?", 1)[0].strip()))
    return hits


def domain_rule(spec: str) -> str | None:
    lower = spec.lower()
    if "vendor/ui" in lower.replace("\\", "/"):
        return "domain-vendor-ui"
    if lower == "reka-ui" or lower.startswith("reka-ui/"):
        return "domain-reka-ui"
    if lower == "@unovis/vue" or lower.startswith("@unovis/"):
        return "domain-unovis"
    return None


def scan_domain(modules_root: Path) -> list[tuple[Path, int, str, str]]:
    violations: list[tuple[Path, int, str, str]] = []
    for module_dir in sorted(p for p in modules_root.iterdir() if p.is_dir()):
        if module_dir.name == "web":
            continue
        web_root = module_dir / "web"
        if not web_root.is_dir():
            continue
        for path in iter_files(web_root):
            text = path.read_text(encoding="utf-8")
            for line, spec in specs_in(text):
                rule = domain_rule(spec)
                if rule:
                    violations.append((path, line, spec, rule))
    return violations


def scan_l2_heights(modules_root: Path) -> list[tuple[Path, int, str, str]]:
    violations: list[tuple[Path, int, str, str]] = []
    vendor = modules_root / "web" / "web" / "components" / "vendor" / "ui"
    for path in iter_files(vendor):
        text = path.read_text(encoding="utf-8")
        for i, line in enumerate(text.splitlines(), start=1):
            if HEIGHT_RE.search(line):
                violations.append((path, i, line.strip(), "l2-h-control"))
    return violations


def rel_web_path(path: Path, modules_root: Path) -> str:
    try:
        return path.relative_to(modules_root).as_posix()
    except ValueError:
        return path.as_posix()


def is_kit_host_dialog_surface(rel: str) -> bool:
    if rel == DIALOG_BARREL:
        return False
    for prefix in KIT_HOST_DIALOG_PREFIXES:
        if rel == prefix or rel.startswith(prefix):
            return True
    return False


def scan_kit_host_dialog(modules_root: Path) -> list[tuple[Path, int, str, str]]:
    violations: list[tuple[Path, int, str, str]] = []
    web = modules_root / "web"
    if not web.is_dir():
        return violations
    for path in iter_files(web):
        rel = rel_web_path(path, modules_root)
        if not is_kit_host_dialog_surface(rel):
            continue
        text = path.read_text(encoding="utf-8")
        for line, spec in specs_in(text):
            if "vendor/ui/dialog" in spec.replace("\\", "/").lower():
                violations.append((path, line, spec, "kit-host-dialog"))
    return violations


def scan(modules_root: Path) -> list[tuple[Path, int, str, str]]:
    return (
        scan_domain(modules_root)
        + scan_l2_heights(modules_root)
        + scan_kit_host_dialog(modules_root)
    )


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
        print(f"forbid_choy_ui_kit_gates: modules root not found: {modules_root}", file=sys.stderr)
        return 2

    violations = scan(modules_root)
    if not violations:
        print("forbid_choy_ui_kit_gates: ok (0 violations)")
        return 0

    print(f"forbid_choy_ui_kit_gates: {len(violations)} violation(s)", file=sys.stderr)
    for path, line, spec, rule in violations:
        try:
            rel = path.relative_to(REPO_ROOT)
        except ValueError:
            rel = path
        print(f"  {rel}:{line}: [{rule}] {spec}", file=sys.stderr)
    print(
        "W4 gates: domain modules import Choy* from @/web; L2 default heights stay "
        "h-control*; Import/Search/Field Dialog chrome goes through layout/choyDialog.",
        file=sys.stderr,
    )
    return 1


if __name__ == "__main__":
    raise SystemExit(main())
