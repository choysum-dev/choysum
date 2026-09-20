#!/usr/bin/env python3
# SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
# SPDX-License-Identifier: LGPL-3.0-or-later
"""Unit tests for forbid_core_engine_imports.py."""

from __future__ import annotations

import importlib.util
import pathlib
import tempfile
import unittest

SCRIPT = pathlib.Path(__file__).resolve().parent / "forbid_core_engine_imports.py"
REPO_ROOT = pathlib.Path(__file__).resolve().parents[2]


def load_mod():
    spec = importlib.util.spec_from_file_location("forbid_core_engine_imports", SCRIPT)
    mod = importlib.util.module_from_spec(spec)
    assert spec.loader is not None
    spec.loader.exec_module(mod)
    return mod


class ForbidCoreEngineImportsTest(unittest.TestCase):
    def test_is_forbidden_prefixes(self):
        mod = load_mod()
        self.assertTrue(mod.is_forbidden("@/core/service/orm/repository/types"))
        self.assertTrue(mod.is_forbidden("@/core/service/orm/repository/types/groupby"))
        self.assertTrue(mod.is_forbidden("@/core/service/orm/repository/authz"))
        self.assertTrue(mod.is_forbidden("@/core/service/orm/relation/processor"))
        self.assertTrue(mod.is_forbidden("@choysum-dev/core/service/orm/repository/query"))
        # Exact repository barrel stays allowed (SF-D).
        self.assertFalse(mod.is_forbidden("@/core/service/orm/repository"))
        self.assertFalse(mod.is_forbidden("@/core/service/api/query"))
        self.assertFalse(mod.is_forbidden("@/core/service/orm/model"))

    def test_scan_detects_violation_and_allows_clean_tree(self):
        mod = load_mod()
        with tempfile.TemporaryDirectory() as tmp:
            root = pathlib.Path(tmp)
            app = root / "auth" / "service"
            app.mkdir(parents=True)
            bad = app / "bad.ts"
            bad.write_text(
                "import { x } from '@/core/service/orm/repository/authz';\n",
                encoding="utf-8",
            )
            good_app = root / "base" / "service"
            good_app.mkdir(parents=True)
            (good_app / "ok.ts").write_text(
                "import { RepositoryFactory } from '@/core/service/orm/repository';\n"
                "import type { SoftDeleteOptions } from '@/core/service/api/query';\n",
                encoding="utf-8",
            )
            # core tree must be ignored even with forbidden-looking imports.
            core = root / "core" / "service" / "orm" / "repository"
            core.mkdir(parents=True)
            (core / "internal.ts").write_text(
                "import type { X } from '@/core/service/orm/repository/types';\n",
                encoding="utf-8",
            )

            violations = mod.scan(root)
            self.assertEqual(len(violations), 1)
            self.assertEqual(violations[0][0], bad)
            self.assertEqual(violations[0][2], "@/core/service/orm/repository/authz")

    def test_repo_modules_are_clean(self):
        mod = load_mod()
        violations = mod.scan(REPO_ROOT / "modules")
        self.assertEqual(violations, [], violations)

    def test_main_ok_on_repo(self):
        mod = load_mod()
        self.assertEqual(mod.main([]), 0)


if __name__ == "__main__":
    unittest.main()
