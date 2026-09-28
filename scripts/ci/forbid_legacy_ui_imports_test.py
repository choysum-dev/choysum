#!/usr/bin/env python3
# SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
# SPDX-License-Identifier: LGPL-3.0-or-later
"""Unit tests for forbid_legacy_ui_imports.py."""

from __future__ import annotations

import importlib.util
import pathlib
import tempfile
import unittest

SCRIPT = pathlib.Path(__file__).resolve().parent / "forbid_legacy_ui_imports.py"
REPO_ROOT = pathlib.Path(__file__).resolve().parents[2]


def load_mod():
    spec = importlib.util.spec_from_file_location("forbid_legacy_ui_imports", SCRIPT)
    mod = importlib.util.module_from_spec(spec)
    assert spec.loader is not None
    spec.loader.exec_module(mod)
    return mod


class ForbidLegacyUiImportsTest(unittest.TestCase):
    def test_is_forbidden_package(self):
        mod = load_mod()
        self.assertEqual(mod.is_forbidden_package("element-plus"), "element-plus")
        self.assertEqual(mod.is_forbidden_package("element-plus/es/button"), "element-plus")
        self.assertEqual(mod.is_forbidden_package("@element-plus/icons-vue"), "element-plus")
        self.assertEqual(mod.is_forbidden_package("echarts"), "echarts")
        self.assertEqual(mod.is_forbidden_package("echarts/core"), "echarts")
        self.assertEqual(mod.is_forbidden_package("vue-echarts"), "echarts")
        self.assertIsNone(mod.is_forbidden_package("vue"))
        self.assertIsNone(mod.is_forbidden_package("@/web"))
        self.assertIsNone(mod.is_forbidden_package("reka-ui"))

    def test_is_forbidden_o_vue(self):
        mod = load_mod()
        self.assertTrue(mod.is_forbidden_o_vue("@/web/web/components/view/OFormView.vue"))
        self.assertTrue(mod.is_forbidden_o_vue("../OListView.vue"))
        self.assertTrue(mod.is_forbidden_o_vue("./OVTable.vue"))
        self.assertFalse(mod.is_forbidden_o_vue("@/web/web/components/view/FormView.vue"))
        self.assertFalse(mod.is_forbidden_o_vue("@/web/web/components/field/OneToManyField.vue"))
        self.assertFalse(mod.is_forbidden_o_vue("@/web"))

    def test_scan_detects_and_skips_fixtures(self):
        mod = load_mod()
        with tempfile.TemporaryDirectory() as tmp:
            root = pathlib.Path(tmp)
            app = root / "partner" / "web"
            app.mkdir(parents=True)
            bad = app / "leak.ts"
            bad.write_text(
                "import { ElButton } from 'element-plus';\n"
                "import Chart from 'vue-echarts';\n"
                "import Form from '@/web/web/components/view/OFormView.vue';\n",
                encoding="utf-8",
            )
            good = app / "ok.ts"
            good.write_text(
                "import { ChoyButton } from '@/web';\n"
                "import Form from '@/web/web/components/view/FormView.vue';\n",
                encoding="utf-8",
            )
            fixtures = root / "web" / "fixtures"
            fixtures.mkdir(parents=True)
            (fixtures / "legacy.ts").write_text(
                "import { ElButton } from 'element-plus';\n",
                encoding="utf-8",
            )

            violations = mod.scan(root)
            self.assertEqual(len(violations), 3)
            rules = {v[3] for v in violations}
            self.assertEqual(rules, {"element-plus", "echarts", "o-star-vue"})


if __name__ == "__main__":
    unittest.main()
