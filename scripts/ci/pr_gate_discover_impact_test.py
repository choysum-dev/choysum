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
        self.assertTrue(mod.is_go_test_path(".github/workflows/prepare-embedded-assets.yml"))
        self.assertTrue(mod.is_go_test_path(".github/workflows/build-choysum-cli.yml"))
        self.assertFalse(mod.is_go_test_path("scripts/ci/modules_npm_trust.py"))
        self.assertFalse(mod.is_go_test_path(".github/workflows/i18n-status.yml"))
        self.assertFalse(mod.is_go_test_path(".github/workflows/pr-agent.yml"))
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
        # Derive from BUILD_PREFIXES so renamed/added prefixes stay covered.
        build_paths = [f"{prefix}marker.ts" for prefix in mod.BUILD_PREFIXES]
        self.assertGreater(len(build_paths), 0)
        for path in build_paths:
            with self.subTest(path=path):
                with tempfile.TemporaryDirectory() as tmp:
                    modules = pathlib.Path(tmp) / "modules"
                    (modules / "auth" / "e2e").mkdir(parents=True)
                    (modules / "auth" / "e2e" / "smoke.spec.ts").write_text(
                        "// smoke\n", encoding="utf-8"
                    )

                    def fake_run(cmd, *args, **kwargs):
                        stdout = path + "\n" if "diff" in cmd else ""
                        return Mock(stdout=stdout, returncode=0)

                    with patch.object(mod, "MODULES_ROOT", modules), patch.object(
                        mod.subprocess, "run", side_effect=fake_run
                    ):
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
        with patch.object(mod, "has_smoke_spec", return_value=True):
            out = mod.merge_group_or_dispatch_outputs(["auth"], "merge-group")
        self.assertEqual(out["run_full_matrix"], "true")
        self.assertEqual(out["run_go_test"], "true")
        self.assertEqual(out["run_pr_smoke_e2e"], "true")
        self.assertEqual(out["impacted_smoke_e2e_modules_json"], '["auth"]')
        self.assertNotIn("run_bootstrap_verify", out)

    def test_workflow_only_pr_agent_does_not_expand_module_matrix(self):
        mod = load_mod()
        with tempfile.TemporaryDirectory() as tmp:
            modules = pathlib.Path(tmp) / "modules"
            (modules / "auth" / "e2e").mkdir(parents=True)
            (modules / "auth" / "e2e" / "smoke.spec.ts").write_text("// smoke\n", encoding="utf-8")

            changed = [".github/workflows/pr-agent.yml"]

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

            self.assertEqual(out["run_full_matrix"], "false")
            self.assertEqual(out["impacted_modules_json"], "[]")
            self.assertEqual(out["run_go_test"], "false")
            self.assertEqual(out["reason"], "no-impactable-tests")

    def test_i18n_status_workflow_does_not_expand_module_matrix(self):
        mod = load_mod()
        with tempfile.TemporaryDirectory() as tmp:
            modules = pathlib.Path(tmp) / "modules"
            (modules / "auth").mkdir(parents=True)

            changed = [".github/workflows/i18n-status.yml"]

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

            self.assertEqual(out["run_full_matrix"], "false")
            self.assertEqual(out["impacted_modules_json"], "[]")
            self.assertEqual(out["run_go_test"], "false")

    def test_github_actions_composite_edits_expand_full_matrix(self):
        mod = load_mod()
        self.assertTrue(mod.is_shared_path(".github/actions/setup-chromium/action.yml"))
        with tempfile.TemporaryDirectory() as tmp:
            modules = pathlib.Path(tmp) / "modules"
            (modules / "auth" / "e2e").mkdir(parents=True)
            (modules / "auth" / "e2e" / "smoke.spec.ts").write_text("// smoke\n", encoding="utf-8")

            changed = [".github/actions/setup-chromium/action.yml"]

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
            self.assertEqual(out["run_full_matrix"], "true")
            self.assertEqual(out["impacted_modules_json"], '["auth"]')

    def test_shared_workflow_exact_covers_reusable_workflows_used_by_gate(self):
        import re

        mod = load_mod()
        workflows = REPO_ROOT / ".github" / "workflows"
        referenced = set()
        pending = [p.removeprefix(".github/workflows/") for p in mod.SHARED_WORKFLOW_EXACT]
        seen = set()
        while pending:
            name = pending.pop()
            if name in seen:
                continue
            seen.add(name)
            path = workflows / name
            if not path.is_file():
                # uses: paths are repo-relative (.github/workflows/...).
                path = REPO_ROOT / name
            if not path.is_file():
                continue
            text = path.read_text(encoding="utf-8")
            found = set(
                re.findall(r"uses:\s*\./(\.github/workflows/[^\s\"']+)", text)
            )
            referenced.update(found)
            for ref in found:
                pending.append(ref.removeprefix(".github/workflows/"))
        missing = referenced - set(mod.SHARED_WORKFLOW_EXACT)
        self.assertFalse(
            missing,
            f"gate-referenced workflows missing from SHARED_WORKFLOW_EXACT: {sorted(missing)}",
        )

    def test_gate_reusable_workflows_still_expand_full_matrix(self):
        mod = load_mod()
        for path in sorted(mod.SHARED_WORKFLOW_EXACT):
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

                    self.assertEqual(out["reason"], "shared-runtime")
                    self.assertEqual(out["run_full_matrix"], "true")
                    self.assertEqual(out["impacted_modules_json"], '["auth"]')

    def test_codeql_workflow_edit_does_not_expand_module_matrix(self):
        # CodeQL shares the embed cache key but runs as its own workflow; editing
        # only codeql.yml must not pull the 12-module gate matrix.
        mod = load_mod()
        with tempfile.TemporaryDirectory() as tmp:
            modules = pathlib.Path(tmp) / "modules"
            (modules / "auth").mkdir(parents=True)

            changed = [".github/workflows/codeql.yml"]

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

            self.assertEqual(out["run_full_matrix"], "false")
            self.assertEqual(out["impacted_modules_json"], "[]")
            self.assertEqual(out["run_go_test"], "false")

    def test_no_workflow_still_references_removed_bootstrap_verify(self):
        workflows = REPO_ROOT / ".github" / "workflows"
        paths = sorted(workflows.glob("*.yml")) + sorted(workflows.glob("*.yaml"))
        self.assertGreater(len(paths), 0)
        for path in paths:
            text = path.read_text(encoding="utf-8")
            # Catch job ids / needs / uses, not only the deleted filename.
            self.assertNotIn("bootstrap-verify", text, path.name)
            self.assertNotIn("run_bootstrap_verify", text, path.name)


if __name__ == "__main__":
    unittest.main()
