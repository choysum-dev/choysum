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
REPO_ROOT = pathlib.Path(__file__).resolve().parents[2]


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


class DiscoverImpactGoTestRoutingTest(unittest.TestCase):
    def test_is_go_test_path_covers_ci_harness_and_go_test_workflows(self):
        mod = load_mod()
        self.assertTrue(mod.is_go_test_path("scripts/ci/go_test_shards.py"))
        self.assertTrue(mod.is_go_test_path("scripts/ci/pr_gate_discover_impact.py"))
        self.assertTrue(mod.is_go_test_path("scripts/ci/install_chromium.py"))
        self.assertTrue(mod.is_go_test_path(".github/workflows/pr-gate.yml"))
        self.assertTrue(mod.is_go_test_path(".github/workflows/mainline-verify.yml"))
        self.assertFalse(mod.is_go_test_path("scripts/ci/modules_npm_trust.py"))
        self.assertFalse(mod.is_go_test_path(".github/workflows/i18n-status.yml"))
        self.assertFalse(mod.is_go_test_path("AGENTS.md"))
        self.assertFalse(mod.is_go_test_path("CONTRIBUTING.md"))

    def test_ci_harness_edit_arms_run_go_test_under_shared_runtime(self):
        mod = load_mod()
        with tempfile.TemporaryDirectory() as tmp:
            modules = pathlib.Path(tmp) / "modules"
            (modules / "auth" / "e2e").mkdir(parents=True)
            (modules / "auth" / "e2e" / "smoke.spec.ts").write_text("// smoke\n", encoding="utf-8")

            changed = [
                "scripts/ci/go_test_shards.py",
                ".github/workflows/pr-gate.yml",
                "AGENTS.md",
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
                    out = mod.pull_request_outputs(["auth"])

            self.assertEqual(out["reason"], "shared-runtime")
            self.assertEqual(out["run_go_test"], "true")
            self.assertEqual(out["run_full_matrix"], "true")
            self.assertNotIn("run_bootstrap_verify", out)

    def test_each_build_pipeline_prefix_arms_full_matrix_without_bootstrap_verify(self):
        mod = load_mod()
        # Each BUILD_PREFIXES entry must independently classify as build-pipeline;
        # bundling paths would let one hit mask a regression in another.
        build_paths = [
            "internal/bootstrap/web/src/main.ts",
            "pkg/jsengine/scripts/vuesfc/src/index.ts",
            "pkg/jsengine/scripts/vuevirtual/src/index.ts",
        ]
        for path in build_paths:
            with self.subTest(path=path):
                with tempfile.TemporaryDirectory() as tmp:
                    modules = pathlib.Path(tmp) / "modules"
                    (modules / "auth" / "e2e").mkdir(parents=True)
                    (modules / "auth" / "e2e" / "smoke.spec.ts").write_text(
                        "// smoke\n", encoding="utf-8"
                    )

                    with patch.object(mod, "MODULES_ROOT", modules), patch.object(
                        mod.subprocess, "run"
                    ) as run_mock:
                        run_mock.return_value = Mock(
                            stdout=path + "\n",
                            returncode=0,
                        )
                        with patch.dict(
                            mod.os.environ,
                            {"PR_BASE_SHA": "base", "PR_HEAD_SHA": "head"},
                            clear=False,
                        ):
                            out = mod.pull_request_outputs(["auth"])

                    self.assertEqual(out["reason"], "build-pipeline")
                    self.assertEqual(out["run_full_matrix"], "true")
                    self.assertEqual(out["run_go_test"], "true")
                    self.assertEqual(out["run_pr_smoke_e2e"], "true")
                    self.assertEqual(out["impacted_smoke_e2e_modules_json"], '["auth"]')
                    self.assertNotIn("run_bootstrap_verify", out)

    def test_merge_group_outputs_omit_run_bootstrap_verify(self):
        mod = load_mod()
        out = mod.merge_group_or_dispatch_outputs(["auth"], "merge-group")
        self.assertEqual(out["run_full_matrix"], "true")
        self.assertEqual(out["run_go_test"], "true")
        self.assertNotIn("run_bootstrap_verify", out)

    def test_no_workflow_still_references_removed_bootstrap_verify(self):
        workflows = REPO_ROOT / ".github" / "workflows"
        for path in sorted(workflows.glob("*.yml")):
            text = path.read_text(encoding="utf-8")
            self.assertNotIn("bootstrap-verify.yml", text, path.name)
            self.assertNotIn("run_bootstrap_verify", text, path.name)


if __name__ == "__main__":
    unittest.main()
