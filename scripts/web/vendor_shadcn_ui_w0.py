#!/usr/bin/env python3
# SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
# SPDX-License-Identifier: Apache-2.0
"""Adapt shadcn-vue new-york-v4 ui registry into modules/web vendor/ui (W0).

Keeps existing hand-adapted families; copies missing families with import
rewrites, lucide package rename, local cva, density height patches, SPDX.
"""

from __future__ import annotations

import argparse
import re
import shutil
from pathlib import Path

REPO = Path(__file__).resolve().parents[2]
DEST = REPO / "modules/web/web/components/vendor/ui"

# Already product-patched; keep and only ensure thin index barrels.
KEEP = {
    "badge",
    "button",
    "card",
    "chart",
    "checkbox",
    "combobox",
    "dialog",
    "dropdown-menu",
    "input",
    "popover",
    "scroll-area",
    "select",
    "separator",
    "skeleton",
    "switch",
    "tabs",
    "textarea",
    "toast",
    "tooltip",
}

SPDX_VUE = """<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

"""

SPDX_TS = """// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

"""

HEIGHT_REPLACEMENTS = [
    (re.compile(r"\bh-9\b"), "h-control"),
    (re.compile(r"\bh-10\b"), "h-control-lg"),
    (re.compile(r"\bh-8\b"), "h-control-sm"),
    (re.compile(r"\bsize-9\b"), "size-control"),
    (re.compile(r"\bsize-8\b"), "size-control"),
    (re.compile(r"\bsize-10\b"), "size-control"),
    (re.compile(r"\bshadow-xs\b"), "shadow-sm"),
]

# Map @lucide/vue Icon-suffixed names → lucide-vue-next exports used in Choysum.
ICON_RENAMES = {
    "CircleCheckIcon": "CircleCheck",
    "InfoIcon": "Info",
    "Loader2Icon": "Loader2",
    "OctagonXIcon": "OctagonX",
    "TriangleAlertIcon": "TriangleAlert",
    "XIcon": "X",
    "ChevronDownIcon": "ChevronDown",
    "ChevronLeftIcon": "ChevronLeft",
    "ChevronRightIcon": "ChevronRight",
    "ChevronUpIcon": "ChevronUp",
    "CheckIcon": "Check",
    "SearchIcon": "Search",
    "PlusIcon": "Plus",
    "MinusIcon": "Minus",
    "PanelLeftIcon": "PanelLeft",
    "MoreHorizontalIcon": "MoreHorizontal",
    "ArrowLeftIcon": "ArrowLeft",
    "ArrowRightIcon": "ArrowRight",
    "CalendarIcon": "Calendar",
    "CopyIcon": "Copy",
    "GripVerticalIcon": "GripVertical",
    "EllipsisIcon": "Ellipsis",
    "EllipsisVerticalIcon": "EllipsisVertical",
}


def rel_utils(depth: int) -> str:
    return "/".join([".."] * depth) + "/lib/utils"


def rel_cva(depth: int) -> str:
    return "/".join([".."] * depth) + "/lib/cva"


def rewrite_lucide_block(text: str) -> str:
    def repl_import(match: re.Match[str]) -> str:
        names = match.group(1)
        parts = [p.strip() for p in names.split(",")]
        out = []
        for part in parts:
            if not part:
                continue
            if " as " in part:
                src_name, alias = [x.strip() for x in part.split(" as ", 1)]
            else:
                src_name, alias = part, None
            mapped = ICON_RENAMES.get(src_name, src_name[:-4] if src_name.endswith("Icon") else src_name)
            if alias and alias != mapped:
                out.append(f"{mapped} as {alias}")
            elif alias:
                out.append(mapped)
            else:
                out.append(mapped)
        return "import { " + ", ".join(out) + " } from 'lucide-vue-next'"

    text = re.sub(
        r"""import\s*\{([^}]+)\}\s*from\s*["']@lucide/vue["']""",
        repl_import,
        text,
    )
    # Template / script references that still use Icon suffix after import rewrite.
    for old, new in ICON_RENAMES.items():
        text = re.sub(rf"\b{old}\b", new, text)
    # Generic Icon suffix leftovers from lucide package.
    text = re.sub(r"\b([A-Z][A-Za-z0-9]+)Icon\b", r"\1", text)
    return text


def rewrite_file(path: Path, dest_file: Path, family: str) -> str:
    text = path.read_text(encoding="utf-8")
    # Depth from dest_file to web/: vendor/ui/<family>/file → 4 levels to web/
    # modules/web/web/components/vendor/ui/<family>/X → lib is web/lib → depth 5 from family file
    rel_to_ui = dest_file.relative_to(DEST)
    depth_to_web = 2 + len(rel_to_ui.parts)  # components/vendor + ui parts... wait
    # dest: modules/web/web/components/vendor/ui/<family>/File.vue
    # lib:  modules/web/web/lib/utils.ts
    # from File.vue: ../../../../lib/utils (ui→vendor→components→web) = 4
    depth = 3 + (len(rel_to_ui.parts) - 1)  # family file: parts=(family,file) → depth 4

    text = text.replace("@/lib/utils", rel_utils(depth))
    text = re.sub(
        r"""from\s*["']class-variance-authority["']""",
        f"from '{rel_cva(depth)}'",
        text,
    )
    text = rewrite_lucide_block(text)

    nest = max(len(rel_to_ui.parts) - 1, 1)
    prefix = "../" * nest
    text = re.sub(
        r"""from\s*(["'])@/registry/new-york-v4/ui/([^"']+)\1""",
        lambda m: f"from {m.group(1)}{prefix}{m.group(2)}{m.group(1)}",
        text,
    )

    for pattern, repl in HEIGHT_REPLACEMENTS:
        text = pattern.sub(repl, text)

    # Prefer logical padding where controls mirror (contract patch).
    # Do not rewrite left-*/right-* — centering (left-1/2) and overlays need physical axes.
    text = re.sub(r"\bpl-(\[?[0-9a-z./]+\]?)", r"ps-\1", text)
    text = re.sub(r"\bpr-(\[?[0-9a-z./]+\]?)", r"pe-\1", text)

    if path.suffix == ".vue" and "SPDX-FileCopyrightText" not in text:
        text = SPDX_VUE + text
    elif path.suffix == ".ts" and "SPDX-FileCopyrightText" not in text:
        text = SPDX_TS + text
    return text


def ensure_keep_index(family: str) -> None:
    """Thin named re-export for cross-family imports from new L2."""
    fam_dir = DEST / family
    index = fam_dir / "index.ts"
    if index.exists():
        return
    vue_files = sorted(fam_dir.glob("*.vue"))
    if not vue_files:
        return
    lines = [SPDX_TS.rstrip(), ""]
    for vf in vue_files:
        name = vf.stem
        lines.append(f"export {{ default as {name} }} from './{name}.vue';")
    # Prefer default Button export convenience
    if (fam_dir / f"{family_title(family)}.vue").exists() is False:
        pass
    index.write_text("\n".join(lines) + "\n", encoding="utf-8")


def family_title(family: str) -> str:
    return "".join(p.title() for p in family.split("-"))


def copy_family(family: str, src: Path) -> None:
    src_dir = src / family
    dest_dir = DEST / family
    if dest_dir.exists():
        shutil.rmtree(dest_dir)
    dest_dir.mkdir(parents=True)
    for path in sorted(src_dir.rglob("*")):
        if path.is_dir():
            continue
        if path.name == ".DS_Store":
            continue
        rel = path.relative_to(src_dir)
        dest_file = dest_dir / rel
        dest_file.parent.mkdir(parents=True, exist_ok=True)
        if path.suffix in {".vue", ".ts"}:
            dest_file.write_text(rewrite_file(path, dest_file, family), encoding="utf-8")
        else:
            shutil.copy2(path, dest_file)


def write_barrel() -> None:
    families = sorted(
        p.name
        for p in DEST.iterdir()
        if p.is_dir() and not p.name.startswith(".")
    )
    lines = [
        SPDX_TS.rstrip(),
        "",
        "/**",
        " * Internal barrel for vendored L2 primitives under components/vendor/ui.",
        " * Domain modules must import Choy* from `@/web` only.",
        " */",
        "",
    ]
    for fam in families:
        index = DEST / fam / "index.ts"
        if index.exists():
            lines.append(f"export * from './{fam}';")
            continue
        for vf in sorted((DEST / fam).glob("*.vue")):
            lines.append(f"export {{ default as {vf.stem} }} from './{fam}/{vf.name}';")
        for tf in sorted((DEST / fam).glob("*.ts")):
            if tf.name == "index.ts":
                continue
            # export helpers selectively — skip private-looking
            if tf.stem.startswith("use") or tf.stem in {"utils", "chartTypes", "interface"}:
                lines.append(f"export * from './{fam}/{tf.stem}';")
    # toast helper used by product
    if (DEST / "toast" / "useToast.ts").exists():
        if "export * from './toast/useToast'" not in "\n".join(lines):
            lines.append("export { toast } from './toast/useToast';")
    DEST.joinpath("index.ts").write_text("\n".join(lines) + "\n", encoding="utf-8")


def main() -> None:
    parser = argparse.ArgumentParser(
        description="Adapt a trusted shadcn-vue new-york-v4 ui tree into vendor/ui",
    )
    parser.add_argument(
        "--src",
        type=Path,
        required=True,
        help="trusted upstream UI directory (…/registry/new-york-v4/ui)",
    )
    src = parser.parse_args().src
    if not src.is_dir():
        raise SystemExit(f"missing upstream tree: {src}")
    missing = sorted(
        p.name
        for p in src.iterdir()
        if p.is_dir() and p.name not in KEEP and not p.name.startswith("_")
    )
    print(f"copying {len(missing)} families…")
    for fam in missing:
        copy_family(fam, src)
        print(f"  + {fam}")
    for fam in sorted(KEEP):
        ensure_keep_index(fam)
        print(f"  = keep {fam} (index)")
    write_barrel()
    print("wrote vendor/ui/index.ts")


if __name__ == "__main__":
    main()
