#!/usr/bin/env python3
# SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
# SPDX-License-Identifier: LGPL-3.0-or-later

"""Unit tests for go_test_shards.py."""

from __future__ import annotations

import contextlib
import importlib.util
import io
import json
import pathlib
import re
import tempfile
import unittest
import unittest.mock as mock

SCRIPT = pathlib.Path(__file__).resolve().parent / "go_test_shards.py"
REPO_ROOT = pathlib.Path(__file__).resolve().parents[2]


def load_mod():
    spec = importlib.util.spec_from_file_location("go_test_shards", SCRIPT)
    mod = importlib.util.module_from_spec(spec)
    assert spec.loader is not None
    spec.loader.exec_module(mod)
    return mod


class GoTestShardsTest(unittest.TestCase):
    def test_check_partition_against_repo(self):
        mod = load_mod()
        shards = mod.check_partition()
        self.assertEqual(set(shards), {"cmd", "testing", "module", "runtime", "rest"})
        self.assertGreater(len(shards["cmd"]), 0)
        self.assertGreater(len(shards["testing"]), 0)
        self.assertGreater(len(shards["module"]), 0)
        self.assertGreater(len(shards["runtime"]), 0)
        self.assertGreater(len(shards["rest"]), 0)
        # Chromium only for testing.
        meta = {row["shard"]: row["chromium"] for row in mod.shard_meta()}
        self.assertEqual(meta["testing"], "true")
        self.assertEqual(meta["rest"], "false")

    def test_matrix_json_shape(self):
        mod = load_mod()
        rows = mod.shard_meta()
        self.assertEqual([r["shard"] for r in rows], ["cmd", "testing", "module", "runtime", "rest"])
        buf = io.StringIO()
        with contextlib.redirect_stdout(buf):
            self.assertEqual(mod.main(["matrix"]), 0)
        payload = json.loads(buf.getvalue())
        self.assertEqual(list(payload.keys()), ["include"])
        self.assertEqual(payload["include"], rows)

    def test_workflow_matrix_matches_shard_meta(self):
        mod = load_mod()
        expected = [(row["shard"], row["chromium"]) for row in mod.shard_meta()]
        for wf in ("pr-gate.yml", "mainline-verify.yml"):
            text = (REPO_ROOT / ".github" / "workflows" / wf).read_text(encoding="utf-8")
            found = re.findall(r'- shard: (\S+)\s*\n\s+chromium: "(\w+)"', text)
            self.assertEqual(found, expected, f"{wf} matrix mismatch")

    def test_empty_rest_rejected(self):
        mod = load_mod()
        with mock.patch.object(mod, "run_go_list") as gl:
            gl.side_effect = [
                ["a"],  # ./...
                ["a"],  # cmd covers everything
            ]
            with mock.patch.object(
                mod,
                "SHARD_SPECS",
                [{"name": "cmd", "patterns": ["./cmd/..."], "chromium": False}],
            ):
                with self.assertRaises(SystemExit) as cm:
                    mod.partition_packages()
                self.assertIn("complement is empty", str(cm.exception))

    def test_overlap_detected(self):
        mod = load_mod()
        with mock.patch.object(mod, "run_go_list") as gl:
            gl.side_effect = [
                ["a", "b", "c"],  # ./...
                ["a"],  # cmd
                ["a"],  # testing overlaps
            ]
            with mock.patch.object(
                mod,
                "SHARD_SPECS",
                [
                    {"name": "cmd", "patterns": ["./cmd/..."], "chromium": False},
                    {"name": "testing", "patterns": ["./internal/testing/..."], "chromium": True},
                ],
            ):
                with self.assertRaises(SystemExit) as cm:
                    mod.partition_packages()
                self.assertIn("overlaps", str(cm.exception))

    def test_empty_named_shard_rejected(self):
        mod = load_mod()
        with mock.patch.object(mod, "run_go_list") as gl:
            gl.side_effect = [
                ["a", "b"],  # ./...
                [],  # cmd empty
            ]
            with mock.patch.object(
                mod,
                "SHARD_SPECS",
                [{"name": "cmd", "patterns": ["./cmd/..."], "chromium": False}],
            ):
                with self.assertRaises(SystemExit) as cm:
                    mod.partition_packages()
                self.assertIn("matched no packages", str(cm.exception))

    def test_merge_coverprofiles(self):
        mod = load_mod()
        with tempfile.TemporaryDirectory() as tmp:
            root = pathlib.Path(tmp)
            a = root / "a.out"
            b = root / "b.out"
            out = root / "merged.out"
            a.write_text("mode: atomic\nfoo.go:1.2,3.4 1 1\n", encoding="utf-8")
            b.write_text("mode: atomic\nbar.go:1.2,3.4 1 0\n", encoding="utf-8")
            mod.merge_coverprofiles([a, b], out)
            text = out.read_text(encoding="utf-8")
            self.assertTrue(text.startswith("mode: atomic\n"))
            self.assertIn("foo.go:1.2,3.4 1 1", text)
            self.assertIn("bar.go:1.2,3.4 1 0", text)

    def test_merge_rejects_mode_mismatch(self):
        mod = load_mod()
        with tempfile.TemporaryDirectory() as tmp:
            root = pathlib.Path(tmp)
            a = root / "a.out"
            b = root / "b.out"
            a.write_text("mode: atomic\n", encoding="utf-8")
            b.write_text("mode: set\n", encoding="utf-8")
            with self.assertRaises(SystemExit):
                mod.merge_coverprofiles([a, b], root / "out.out")

    def test_merge_rejects_empty_mode(self):
        mod = load_mod()
        with tempfile.TemporaryDirectory() as tmp:
            root = pathlib.Path(tmp)
            a = root / "a.out"
            a.write_text("mode:\n", encoding="utf-8")
            with self.assertRaises(SystemExit) as cm:
                mod.merge_coverprofiles([a], root / "out.out")
            self.assertIn("empty mode", str(cm.exception))

    def test_merge_rejects_missing_mode(self):
        mod = load_mod()
        with tempfile.TemporaryDirectory() as tmp:
            root = pathlib.Path(tmp)
            a = root / "a.out"
            a.write_text("foo.go:1.2,3.4 1 1\n", encoding="utf-8")
            with self.assertRaises(SystemExit) as cm:
                mod.merge_coverprofiles([a], root / "out.out")
            self.assertIn("missing mode", str(cm.exception))

    def test_merge_require_all_shards(self):
        mod = load_mod()
        with tempfile.TemporaryDirectory() as tmp:
            root = pathlib.Path(tmp)
            paths = []
            for i, name in enumerate(mod.coverprofile_basenames()):
                path = root / name
                path.write_text(f"mode: atomic\np{i}.go:1.2,3.4 1 1\n", encoding="utf-8")
                paths.append(path)
            out = root / "merged.out"
            mod.merge_coverprofiles(paths, out, require_all_shards=True)
            # Drop one shard -> mismatch.
            with self.assertRaises(SystemExit) as cm:
                mod.merge_coverprofiles(paths[:-1], out, require_all_shards=True)
            self.assertIn("shard set mismatch", str(cm.exception))

    def test_merge_require_all_shards_rejects_empty_body(self):
        mod = load_mod()
        with tempfile.TemporaryDirectory() as tmp:
            root = pathlib.Path(tmp)
            paths = []
            for name in mod.coverprofile_basenames():
                path = root / name
                path.write_text("mode: atomic\n", encoding="utf-8")
                paths.append(path)
            with self.assertRaises(SystemExit) as cm:
                mod.merge_coverprofiles(paths, root / "out.out", require_all_shards=True)
            self.assertIn("no coverage lines", str(cm.exception))

    def test_run_go_list_failure_raises(self):
        mod = load_mod()
        with mock.patch.object(mod.subprocess, "run") as run_mock:
            run_mock.return_value = mock.Mock(returncode=1, stdout="", stderr="boom")
            with self.assertRaises(SystemExit) as cm:
                mod.run_go_list(["./..."])
            self.assertIn("go list failed", str(cm.exception))

    def test_packages_rejects_unknown_shard(self):
        mod = load_mod()
        with mock.patch.object(mod, "partition_packages", return_value={"cmd": []}):
            with self.assertRaises(SystemExit) as cm:
                mod.main(["packages", "nope"])
            self.assertIn("unknown shard", str(cm.exception))

    def test_shard_package_not_in_go_list_rejected(self):
        mod = load_mod()
        with mock.patch.object(mod, "run_go_list") as gl:
            gl.side_effect = [["a"], ["b"]]  # ./... ; shard returns unknown pkg
            with mock.patch.object(
                mod,
                "SHARD_SPECS",
                [{"name": "cmd", "patterns": ["./cmd/..."], "chromium": False}],
            ):
                with self.assertRaises(SystemExit) as cm:
                    mod.partition_packages()
                self.assertIn("not in go list", str(cm.exception))

    def test_merge_rejects_duplicate_blocks(self):
        mod = load_mod()
        with tempfile.TemporaryDirectory() as tmp:
            root = pathlib.Path(tmp)
            a = root / "a.out"
            b = root / "b.out"
            a.write_text("mode: atomic\nfoo.go:1.2,3.4 1 1\n", encoding="utf-8")
            b.write_text("mode: atomic\nfoo.go:1.2,3.4 1 0\n", encoding="utf-8")
            with self.assertRaises(SystemExit) as cm:
                mod.merge_coverprofiles([a, b], root / "out.out")
            self.assertIn("duplicate coverage block", str(cm.exception))

    def test_merge_rejects_missing_input_files(self):
        mod = load_mod()
        with tempfile.TemporaryDirectory() as tmp:
            root = pathlib.Path(tmp)
            missing = root / "missing.out"
            with self.assertRaises(SystemExit) as cm:
                mod.merge_coverprofiles([missing], root / "out.out")
            self.assertIn("missing input files", str(cm.exception))


if __name__ == "__main__":
    unittest.main()
