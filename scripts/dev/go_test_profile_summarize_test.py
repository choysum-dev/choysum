#!/usr/bin/env python3
# SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
# SPDX-License-Identifier: LGPL-3.0-or-later

"""Exit-code contract for go_test_profile_summarize.py."""

from __future__ import annotations

import contextlib
import importlib.util
import io
import pathlib
import tempfile
import unittest


SCRIPT = pathlib.Path(__file__).resolve().parent / "go_test_profile_summarize.py"
FIXTURE = pathlib.Path(__file__).resolve().parent / "testdata" / "go_test_profile_sample.jsonl"


def load_mod():
    spec = importlib.util.spec_from_file_location("go_test_profile_summarize", SCRIPT)
    mod = importlib.util.module_from_spec(spec)
    assert spec.loader is not None
    spec.loader.exec_module(mod)
    return mod


def _summarize_text(text: str) -> tuple[int, str]:
    mod = load_mod()
    with tempfile.NamedTemporaryFile("w", suffix=".jsonl", encoding="utf-8", delete=False) as fh:
        fh.write(text)
        path = pathlib.Path(fh.name)
    try:
        buf = io.StringIO()
        with contextlib.redirect_stdout(buf):
            code = mod.summarize(path, slow_secs=0.5, pkg_top=20, test_top=40, timing_path=None)
        return code, buf.getvalue()
    finally:
        path.unlink(missing_ok=True)


class SummarizeExitCodeTest(unittest.TestCase):
    def test_sample_fixture_exits_nonzero_and_lists_named_failure(self):
        mod = load_mod()
        buf = io.StringIO()
        with contextlib.redirect_stdout(buf):
            code = mod.summarize(FIXTURE, slow_secs=0.5, pkg_top=20, test_top=40, timing_path=None)
        self.assertEqual(code, 1)
        self.assertIn("TestCLIUpgradeFlowWithGlobalRegistryIndex", buf.getvalue())

    def test_package_fail_without_named_test_exits_nonzero(self):
        code, out = _summarize_text(
            '{"Action":"fail","Package":"example.com/compilefail","Elapsed":0.2}\n'
        )
        self.assertEqual(code, 1)
        self.assertIn("(package)", out)

    def test_build_fail_without_elapsed_exits_nonzero(self):
        code, out = _summarize_text(
            '{"Action":"build-fail","Package":"example.com/buildfail"}\n'
        )
        self.assertEqual(code, 1)
        self.assertIn("example.com/buildfail", out)

    def test_all_pass_exits_zero(self):
        code, out = _summarize_text(
            '{"Action":"pass","Package":"example.com/ok","Elapsed":0.4}\n'
            '{"Action":"pass","Package":"example.com/ok","Test":"TestOK","Elapsed":0.01}\n'
        )
        self.assertEqual(code, 0)
        self.assertIn("(none)", out)


if __name__ == "__main__":
    unittest.main()
