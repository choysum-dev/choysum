#!/usr/bin/env python3
# SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
# SPDX-License-Identifier: Apache-2.0
"""Unit tests for generate_language_seeds.py (stdlib unittest)."""

from __future__ import annotations

import json
import tempfile
import unittest
from pathlib import Path

import sys
from pathlib import Path as _Path

sys.path.insert(0, str(_Path(__file__).resolve().parent))

from generate_language_seeds import (
    convert_strftime,
    direction_to_ltr_rtl,
    first_day_of_week,
    generate_records,
    main,
    person_name_format,
    xml_id_for_code,
)

CSV = Path(__file__).resolve().parent / "data" / "res.lang.csv"


class GenerateLanguageSeedsTest(unittest.TestCase):
    def test_convert_strftime_common_tokens(self) -> None:
        self.assertEqual(convert_strftime("%m/%d/%Y"), "MM/DD/YYYY")
        self.assertEqual(convert_strftime("%Y年%m月%d日"), "YYYY年MM月DD日")
        self.assertEqual(convert_strftime("%d %b, %Y"), "DD MMM, YYYY")
        self.assertEqual(convert_strftime("%I:%M:%S %p"), "hh:mm:ss A")

    def test_convert_strftime_rejects_unknown(self) -> None:
        with self.assertRaises(ValueError):
            convert_strftime("%Z")

    def test_direction_left_to_right_is_ltr(self) -> None:
        self.assertEqual(direction_to_ltr_rtl("Left-to-Right"), "ltr")
        self.assertEqual(direction_to_ltr_rtl("Right-to-Left"), "rtl")

    def test_person_name_and_week_start(self) -> None:
        self.assertEqual(person_name_format("ja_JP"), "{LastName}{FirstName}")
        self.assertEqual(person_name_format("fr_FR"), "{FirstName} {LastName}")
        self.assertEqual(first_day_of_week("7"), 0)
        self.assertEqual(first_day_of_week("1"), 1)
        self.assertEqual(xml_id_for_code("zh_CN"), "language_zh_cn")

    def test_generate_catalog_omits_bootstrap_active(self) -> None:
        records = generate_records(CSV, catalog_only=True)
        self.assertGreaterEqual(len(records), 80)
        codes = [r["values"]["Code"] for r in records]
        self.assertTrue(all("@" not in c for c in codes))
        self.assertNotIn("en_US", codes)
        self.assertNotIn("zh_CN", codes)
        self.assertFalse(any(r["values"]["IsActive"] for r in records))

        by = {r["values"]["Code"]: r["values"] for r in records}
        self.assertEqual(by["ja_JP"]["PersonNameFormat"], "{LastName}{FirstName}")
        self.assertEqual(by["ar_001"]["Direction"], "rtl")
        self.assertNotIn("%", by["fr_FR"]["DateFormat"])

    def test_check_and_write_roundtrip(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            out = Path(tmp) / "languages.json"
            self.assertEqual(main(["--csv", str(CSV), "--out", str(out), "--write"]), 0)
            self.assertEqual(main(["--csv", str(CSV), "--out", str(out), "--check"]), 0)
            payload = json.loads(out.read_text(encoding="utf-8"))
            self.assertGreaterEqual(len(payload["records"]), 80)
            # Stale file fails --check.
            out.write_text('{"records":[]}\n', encoding="utf-8")
            self.assertEqual(main(["--csv", str(CSV), "--out", str(out), "--check"]), 1)


if __name__ == "__main__":
    unittest.main()
