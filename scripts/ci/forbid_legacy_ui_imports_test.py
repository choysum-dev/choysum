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

    def test_iter_scan_files_ignores_ancestor_skip_dir_names(self):
        """Checkout under …/dist/… must not skip the entire modules tree."""
        mod = load_mod()
        with tempfile.TemporaryDirectory() as tmp:
            # Simulate repo checked out under a directory named "dist".
            modules = pathlib.Path(tmp) / "dist" / "choysum" / "modules"
            app = modules / "partner" / "web"
            app.mkdir(parents=True)
            bad = app / "leak.ts"
            bad.write_text("import { ElButton } from 'element-plus';\n", encoding="utf-8")
            files = mod.iter_scan_files(modules)
            self.assertEqual([p.resolve() for p in files], [bad.resolve()])
            violations = mod.scan(modules)
            self.assertEqual(len(violations), 1)
            self.assertEqual(violations[0][3], "element-plus")

    def test_strip_comments_preserves_url_before_import(self):
        mod = load_mod()
        text = (
            "const u = 'https://example.com/x'; import { ElButton } from 'element-plus';\n"
            'const v = "https://cdn.example/a"; import Chart from "vue-echarts";\n'
            "const w = `https://x`; // real comment\n"
            "import Ok from 'vue'; // trailing\n"
            "/* block */ import { ElIcon } from '@element-plus/icons-vue';\n"
        )
        lines = dict(mod.strip_comments(text))
        self.assertIn("element-plus", lines[1])
        self.assertIn("vue-echarts", lines[2])
        self.assertIn("`https://x`", lines[3])
        self.assertNotIn("real comment", lines[3])
        self.assertIn("from 'vue'", lines[4])
        self.assertNotIn("trailing", lines[4])
        self.assertIn("@element-plus/icons-vue", lines[5])
        # scan_file still sees the banned imports after string-aware strip.
        with tempfile.TemporaryDirectory() as tmp:
            p = pathlib.Path(tmp) / "x.ts"
            p.write_text(text, encoding="utf-8")
            hits = mod.scan_file(p)
            rules = {h[2] for h in hits}
            self.assertEqual(rules, {"element-plus", "echarts"})

    def test_scan_export_star_from_banned_modules(self):
        mod = load_mod()
        with tempfile.TemporaryDirectory() as tmp:
            p = pathlib.Path(tmp) / "reexport.ts"
            p.write_text(
                "export * from 'element-plus';\n"
                "export { ElButton } from 'element-plus';\n"
                "export * as icons from '@element-plus/icons-vue';\n",
                encoding="utf-8",
            )
            hits = mod.scan_file(p)
            rules = {h[2] for h in hits}
            self.assertEqual(rules, {"element-plus"})
            self.assertGreaterEqual(len(hits), 3)

    def test_scan_vue_el_tags_in_template_only(self):
        mod = load_mod()
        with tempfile.TemporaryDirectory() as tmp:
            root = pathlib.Path(tmp)
            app = root / "web" / "web"
            app.mkdir(parents=True)
            bad = app / "Leak.vue"
            bad.write_text(
                "<!--\nSPDX\n-->\n"
                "<template>\n"
                "  <!-- <el-button>commented</el-button> -->\n"
                "  <div>\n"
                "    <el-select-v2 multiple />\n"
                "    <el-tree :data=\"nodes\" />\n"
                "  </div>\n"
                "</template>\n"
                "<script setup lang=\"ts\">\n"
                "const stubs = { 'el-select-v2': true, ElSelectV2: true };\n"
                "const note = '<el-button>in string</el-button>';\n"
                "</script>\n",
                encoding="utf-8",
            )
            good = app / "Ok.vue"
            good.write_text(
                "<template>\n"
                "  <RelationCombobox />\n"
                "  <template #footer><span /></template>\n"
                "</template>\n"
                "<script setup lang=\"ts\">\n"
                "const stubs = { 'el-select-v2': true };\n"
                "</script>\n",
                encoding="utf-8",
            )

            hits = mod.scan_file(bad)
            rules = [h[2] for h in hits]
            self.assertEqual(rules, ["el-tag", "el-tag"])
            tags = {h[1] for h in hits}
            self.assertEqual(tags, {"el-select-v2", "el-tree"})

            self.assertEqual(mod.scan_file(good), [])

            violations = mod.scan(root)
            self.assertEqual(len(violations), 2)
            self.assertTrue(all(v[3] == "el-tag" for v in violations))

    def test_scan_vue_el_tags_line_maps_past_comments(self):
        """Commented-out el-* must not steal the line number of a later live tag."""
        mod = load_mod()
        text = (
            "<template>\n"
            "  <!-- <el-select /> -->\n"
            "  <el-select multiple />\n"
            "</template>\n"
            "<script setup>const x = 1;</script>\n"
        )
        hits = mod.scan_vue_el_tags(pathlib.Path("x.vue"), text)
        self.assertEqual([(h[0], h[1]) for h in hits], [(3, "el-select")])

    def test_extract_vue_template_nested_slot_templates(self):
        mod = load_mod()
        text = (
            "<template>\n"
            "  <el-form>\n"
            "    <template #default><span /></template>\n"
            "  </el-form>\n"
            "</template>\n"
            "<script setup>const x = '<el-alert />';</script>\n"
        )
        body = mod.extract_vue_template_body(text)
        self.assertIsNotNone(body)
        assert body is not None
        self.assertIn("<el-form>", body)
        self.assertIn("<template #default>", body)
        self.assertNotIn("<script", body)
        hits = mod.scan_vue_el_tags(pathlib.Path("x.vue"), text)
        self.assertEqual([h[1] for h in hits], ["el-form"])


if __name__ == "__main__":
    unittest.main()
