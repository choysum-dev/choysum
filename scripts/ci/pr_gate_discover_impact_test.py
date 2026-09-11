#!/usr/bin/env python3
# SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
# SPDX-License-Identifier: LGPL-3.0-or-later

"""Focused checks for PR Gate e2e routing helpers."""

from __future__ import annotations

import importlib.util
import pathlib
import tempfile
import unittest
from unittest.mock import Mock, patch


SCRIPT = pathlib.Path(__file__).resolve().parent / "pr_gate_discover_impact.py"


def load_mod():
    spec = importlib.util.spec_from_file_location("pr_gate_discover_impact", SCRIPT)
    mod = importlib.util.module_from_spec(spec)
    assert spec.loader is not None
    spec.loader.exec_module(mod)
    return mod


class DiscoverImpactE2ERoutingTest(unittest.TestCase):
    def test_direct_e2e_edit_on_smoke_module_gets_full_suite(self):
        mod = load_mod()
        with tempfile.TemporaryDirectory() as tmp:
            root = pathlib.Path(tmp)
            modules = root / "modules"
            (modules / "auth" / "e2e").mkdir(parents=True)
            (modules / "auth" / "e2e" / "smoke.spec.ts").write_text("// smoke\n", encoding="utf-8")
            (modules / "auth" / "e2e" / "switch_company_scope.spec.ts").write_text("// full\n", encoding="utf-8")
            (modules / "base" / "e2e").mkdir(parents=True)
            (modules / "base" / "e2e" / "language_format.spec.ts").write_text("// base\n", encoding="utf-8")

            changed = [
                "modules/auth/e2e/switch_company_scope.spec.ts",
                "modules/auth/e2e/utils/switchCompany.ts",
            ]

            with patch.object(mod, "MODULES_ROOT", modules), patch.object(
                mod.subprocess, "run"
            ) as run_mock:
                run_mock.return_value = Mock(
                    stdout="\n".join(changed) + "\n",
                    returncode=0,
                )
                with patch.dict(
                    mod.os.environ,
                    {"PR_BASE_SHA": "base", "PR_HEAD_SHA": "head"},
                    clear=False,
                ):
                    out = mod.pull_request_outputs(["auth", "base"])

            self.assertEqual(out["impacted_direct_e2e_modules_json"], '["auth"]')
            self.assertEqual(out["impacted_smoke_e2e_modules_json"], "[]")
            self.assertEqual(out["run_pr_smoke_e2e"], "false")

    def test_fanout_keeps_smoke_without_full_e2e(self):
        mod = load_mod()
        with tempfile.TemporaryDirectory() as tmp:
            root = pathlib.Path(tmp)
            modules = root / "modules"
            (modules / "auth" / "e2e").mkdir(parents=True)
            (modules / "auth" / "e2e" / "smoke.spec.ts").write_text("// smoke\n", encoding="utf-8")
            (modules / "auth" / "web").mkdir(parents=True)
            (modules / "auth" / "web" / "x.ts").write_text("// local\n", encoding="utf-8")

            changed = ["modules/auth/web/x.ts"]

            with patch.object(mod, "MODULES_ROOT", modules), patch.object(
                mod.subprocess, "run"
            ) as run_mock:
                run_mock.return_value = Mock(
                    stdout="\n".join(changed) + "\n",
                    returncode=0,
                )
                with patch.dict(
                    mod.os.environ,
                    {"PR_BASE_SHA": "base", "PR_HEAD_SHA": "head"},
                    clear=False,
                ):
                    out = mod.pull_request_outputs(["auth"])

            self.assertEqual(out["impacted_direct_e2e_modules_json"], "[]")
            self.assertEqual(out["impacted_smoke_e2e_modules_json"], '["auth"]')
            self.assertEqual(out["run_pr_smoke_e2e"], "true")


if __name__ == "__main__":
    unittest.main()
