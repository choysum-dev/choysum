#!/usr/bin/env python3
# SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
# SPDX-License-Identifier: LGPL-3.0-or-later
"""Unit tests for forbid_choy_ui_kit_gates.py."""

from __future__ import annotations

import importlib.util
import pathlib
import tempfile
import unittest

SCRIPT = pathlib.Path(__file__).resolve().parent / "forbid_choy_ui_kit_gates.py"
REPO_ROOT = pathlib.Path(__file__).resolve().parents[2]


def load_mod():
    spec = importlib.util.spec_from_file_location("forbid_choy_ui_kit_gates", SCRIPT)
    mod = importlib.util.module_from_spec(spec)
    assert spec.loader is not None
    spec.loader.exec_module(mod)
    return mod


class ForbidChoyUiKitGatesTest(unittest.TestCase):
    def test_domain_rule(self):
        mod = load_mod()
        self.assertEqual(mod.domain_rule("../vendor/ui/button/Button.vue"), "domain-vendor-ui")
        self.assertEqual(mod.domain_rule("reka-ui"), "domain-reka-ui")
        self.assertEqual(mod.domain_rule("@unovis/vue"), "domain-unovis")
        self.assertIsNone(mod.domain_rule("@/web"))
        self.assertIsNone(mod.domain_rule("vue"))

    def test_height_re(self):
        mod = load_mod()
        self.assertTrue(mod.HEIGHT_RE.search('class="h-9 w-full"'))
        self.assertTrue(mod.HEIGHT_RE.search("h-10"))
        self.assertTrue(mod.HEIGHT_RE.search("h-8"))
        self.assertTrue(mod.HEIGHT_RE.search("h-10 w-10"))
        self.assertIsNone(mod.HEIGHT_RE.search("h-control"))
        self.assertIsNone(mod.HEIGHT_RE.search("h-96"))
        self.assertIsNone(mod.HEIGHT_RE.search("min-h-8"))
        self.assertIsNone(mod.HEIGHT_RE.search("max-h-10"))
        self.assertIsNone(mod.HEIGHT_RE.search("size-10"))

    def test_comments_are_not_imports_or_heights(self):
        mod = load_mod()
        with tempfile.TemporaryDirectory() as tmp:
            root = pathlib.Path(tmp)
            partner = root / "partner" / "web"
            partner.mkdir(parents=True)
            (partner / "ok.ts").write_text(
                "// import Button from '@/web/web/components/vendor/ui/button/Button.vue';\n"
                "import { ChoyButton } from '@/web';\n",
                encoding="utf-8",
            )
            vendor = root / "web" / "web" / "components" / "vendor" / "ui" / "button"
            vendor.mkdir(parents=True)
            (vendor / "Button.vue").write_text(
                "<!-- default was h-9 -->\n"
                '// class="h-9"\n'
                '<button class="h-control min-h-8">x</button>\n',
                encoding="utf-8",
            )
            field = root / "web" / "web" / "components" / "field"
            field.mkdir(parents=True)
            (field / "ChoyX.vue").write_text(
                "// import Dialog from '@/web/web/components/vendor/ui/dialog/Dialog.vue';\n",
                encoding="utf-8",
            )
            self.assertEqual(mod.scan(root), [])

    def test_scan_tmp_tree(self):
        mod = load_mod()
        with tempfile.TemporaryDirectory() as tmp:
            root = pathlib.Path(tmp)
            partner = root / "partner" / "web"
            partner.mkdir(parents=True)
            (partner / "leak.ts").write_text(
                "import Button from '@/web/web/components/vendor/ui/button/Button.vue';\n",
                encoding="utf-8",
            )
            web_vendor = root / "web" / "web" / "components" / "vendor" / "ui" / "button"
            web_vendor.mkdir(parents=True)
            (web_vendor / "Button.vue").write_text(
                '<button class="h-9">x</button>\n',
                encoding="utf-8",
            )
            field = root / "web" / "web" / "components" / "field"
            field.mkdir(parents=True)
            (field / "ChoyX.vue").write_text(
                "import Dialog from '@/web/web/components/vendor/ui/dialog/Dialog.vue';\n",
                encoding="utf-8",
            )
            barrel = root / "web" / "web" / "components" / "layout"
            barrel.mkdir(parents=True)
            (barrel / "choyDialog.ts").write_text(
                "export { default as ChoyDialog } from '../vendor/ui/dialog/Dialog.vue';\n",
                encoding="utf-8",
            )
            violations = mod.scan(root)
            rules = {v[3] for v in violations}
            self.assertEqual(rules, {"domain-vendor-ui", "l2-h-control", "kit-host-dialog"})

    def test_iter_files_ignores_ancestor_skip_dir_names(self):
        mod = load_mod()
        with tempfile.TemporaryDirectory() as tmp:
            modules = pathlib.Path(tmp) / "dist" / "choysum" / "modules"
            partner = modules / "partner" / "web"
            partner.mkdir(parents=True)
            leak = partner / "leak.ts"
            leak.write_text(
                "import Button from '@/web/web/components/vendor/ui/button/Button.vue';\n",
                encoding="utf-8",
            )
            files = mod.iter_files(modules / "partner" / "web")
            self.assertEqual([p.resolve() for p in files], [leak.resolve()])
            violations = mod.scan(modules)
            self.assertEqual(len(violations), 1)
            self.assertEqual(violations[0][3], "domain-vendor-ui")

    def test_repo_is_clean(self):
        mod = load_mod()
        modules = REPO_ROOT / "modules"
        self.assertTrue(modules.is_dir())
        self.assertEqual(mod.scan(modules), [])


if __name__ == "__main__":
    unittest.main()
