#!/usr/bin/env python3
# SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
# SPDX-License-Identifier: Apache-2.0
"""Generate modules/base/data/languages.json from Odoo res.lang.csv.

Enabling a Language (IsActive) does not install PO/terminology packs.

Usage:
  python3 scripts/base/generate_language_seeds.py --write
  python3 scripts/base/generate_language_seeds.py --check
"""

from __future__ import annotations

import argparse
import csv
import json
import re
import sys
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[2]
DEFAULT_CSV = Path(__file__).resolve().parent / "data" / "res.lang.csv"
DEFAULT_OUT = REPO_ROOT / "modules" / "base" / "data" / "languages.json"

# Odoo strftime → dayjs tokens. Unknown %-directives fail the generator.
STRFTIME_TO_DAYJS = {
    "%Y": "YYYY",
    "%y": "YY",
    "%m": "MM",
    "%d": "DD",
    "%H": "HH",
    "%I": "hh",
    "%M": "mm",
    "%S": "ss",
    "%p": "A",
    "%a": "ddd",
    "%A": "dddd",
    "%b": "MMM",
    "%B": "MMMM",
    "%%": "%",
}

ACTIVE_CODES = frozenset({"en_US", "zh_CN"})

# Bilingual Name overlays for the two default-active languages (not machine-translated catalog).
NAME_OVERLAYS: dict[str, dict[str, str]] = {
    "en_US": {
        "en_US": "English (US)",
        "zh_CN": "英语（美国）",
    },
    "zh_CN": {
        "en_US": "Chinese (Simplified)",
        "zh_CN": "简体中文",
    },
}

TOKEN_RE = re.compile(r"%%|%[-]?[A-Za-z]")


def convert_strftime(pattern: str) -> str:
    """Convert an Odoo strftime pattern to dayjs; raise on unknown tokens."""

    def repl(match: re.Match[str]) -> str:
        token = match.group(0)
        # Normalize optional flag form (%-d) to base token for lookup.
        key = "%" + token[-1] if token.startswith("%-") else token
        if key == "%%" or token == "%%":
            return "%"
        mapped = STRFTIME_TO_DAYJS.get(key)
        if mapped is None:
            raise ValueError(f"unsupported strftime token {token!r} in {pattern!r}")
        return mapped

    return TOKEN_RE.sub(repl, pattern)


def direction_to_ltr_rtl(raw: str) -> str:
    text = (raw or "").strip().lower().replace("_", "-")
    if text in {"rtl", "right-to-left"} or "right-to-left" in text:
        return "rtl"
    return "ltr"


def person_name_format(code: str) -> str:
    main = code.split("_", 1)[0].lower()
    if main in {"zh", "ja", "ko"}:
        return "{LastName}{FirstName}"
    return "{FirstName} {LastName}"


def xml_id_for_code(code: str) -> str:
    return "language_" + code.lower()


def first_day_of_week(week_start: str) -> int:
    return int(week_start) % 7


def row_to_record(row: dict[str, str], *, catalog_only: bool) -> dict | None:
    code = (row.get("code") or "").strip()
    if not code or "@" in code:
        return None
    # Default-active en_US / zh_CN stay authoritative in bootstrap.json.
    if catalog_only and code in ACTIVE_CODES:
        return None

    name_en = (row.get("name") or "").strip() or code
    name = NAME_OVERLAYS.get(code) or {"en_US": name_en}

    values: dict = {
        "Name": name,
        "Code": code,
        "IsActive": code in ACTIVE_CODES,
        "Direction": direction_to_ltr_rtl(row.get("direction") or ""),
        "DecimalSeparator": row.get("decimal_point") or ".",
        "ThousandSeparator": row.get("thousands_sep") if row.get("thousands_sep") is not None else ",",
        "Grouping": (row.get("grouping") or "[3,0]").strip() or "[3,0]",
        "DateFormat": convert_strftime(row.get("date_format") or "%Y-%m-%d"),
        "TimeFormat": convert_strftime(row.get("time_format") or "%H:%M:%S"),
        "FirstDayOfWeek": first_day_of_week(row.get("week_start") or "1"),
        "PersonNameFormat": person_name_format(code),
    }

    return {
        "name": xml_id_for_code(code),
        "model": "Language",
        "noupdate": True,
        "values": values,
    }


def generate_records(csv_path: Path, *, catalog_only: bool = True) -> list[dict]:
    with csv_path.open(newline="", encoding="utf-8") as fh:
        reader = csv.DictReader(fh)
        records: list[dict] = []
        for row in reader:
            rec = row_to_record(row, catalog_only=catalog_only)
            if rec is not None:
                records.append(rec)
    records.sort(key=lambda rec: rec["values"]["Code"])
    return records


def render_json(records: list[dict]) -> str:
    payload = {"records": records}
    return json.dumps(payload, ensure_ascii=False, indent=2) + "\n"


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--csv", type=Path, default=DEFAULT_CSV, help="Odoo res.lang.csv path")
    parser.add_argument("--out", type=Path, default=DEFAULT_OUT, help="Output languages.json path")
    mode = parser.add_mutually_exclusive_group(required=True)
    mode.add_argument("--write", action="store_true", help="Write languages.json")
    mode.add_argument("--check", action="store_true", help="Exit 1 if languages.json is stale")
    args = parser.parse_args(argv)

    if not args.csv.is_file():
        print(f"error: csv not found: {args.csv}", file=sys.stderr)
        return 2

    records = generate_records(args.csv, catalog_only=True)
    if len(records) < 80:
        print(f"error: expected >= 80 inactive catalog rows, got {len(records)}", file=sys.stderr)
        return 2

    active = sorted(r["values"]["Code"] for r in records if r["values"]["IsActive"])
    if active:
        print(f"error: languages.json must not mark rows active (bootstrap owns en_US/zh_CN); got {active}", file=sys.stderr)
        return 2
    codes = {r["values"]["Code"] for r in records}
    if codes & ACTIVE_CODES:
        print(f"error: languages.json must omit bootstrap-owned codes {sorted(ACTIVE_CODES)}", file=sys.stderr)
        return 2

    text = render_json(records)
    if args.write:
        args.out.parent.mkdir(parents=True, exist_ok=True)
        args.out.write_text(text, encoding="utf-8")
        print(f"wrote {args.out} ({len(records)} languages)")
        return 0

    if not args.out.is_file():
        print(f"error: missing {args.out}; run with --write", file=sys.stderr)
        return 1
    current = args.out.read_text(encoding="utf-8")
    if current != text:
        print(f"error: {args.out} is stale; run: python3 scripts/base/generate_language_seeds.py --write", file=sys.stderr)
        return 1
    print(f"ok: {args.out} up to date ({len(records)} languages)")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
