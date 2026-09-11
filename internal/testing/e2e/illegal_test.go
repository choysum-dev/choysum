// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScanIllegalE2EMarksEmpty(t *testing.T) {
	marks, err := ScanIllegalE2EMarks(nil)
	if err != nil || len(marks) != 0 {
		t.Fatalf("got marks=%v err=%v", marks, err)
	}
	if err := CheckIllegalE2EMarks(nil); err != nil {
		t.Fatalf("Check: %v", err)
	}
}

func TestScanIllegalE2EMarksPlaywrightAndNodeBuiltins(t *testing.T) {
	dir := t.TempDir()
	pw := filepath.Join(dir, "pw.spec.ts")
	nodeFS := filepath.Join(dir, "fs.spec.ts")
	nodePath := filepath.Join(dir, "path.spec.ts")
	nodeCrypto := filepath.Join(dir, "crypto.spec.ts")
	requireForm := filepath.Join(dir, "require.spec.ts")
	ok := filepath.Join(dir, "ok.spec.ts")

	write := func(path, body string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(pw, "import { test } from '@playwright/test';\n")
	write(nodeFS, "import fs from 'node:fs';\n")
	write(nodePath, "import { join } from \"node:path\";\n")
	write(nodeCrypto, "const c = await import('node:crypto');\n")
	write(requireForm, "const p = require('node:path');\n")
	write(ok, "import { test } from '@choysum/e2e';\ntest('ok', async () => {});\n")

	marks, err := ScanIllegalE2EMarks([]string{pw, nodeFS, nodePath, nodeCrypto, requireForm, ok, "  "})
	if err != nil {
		t.Fatal(err)
	}
	if len(marks) != 5 {
		t.Fatalf("expected 5 marks, got %#v", marks)
	}
	kinds := map[string]int{}
	for _, m := range marks {
		kinds[m.Kind]++
	}
	if kinds["playwright"] != 1 || kinds["node-builtin"] != 4 {
		t.Fatalf("kinds=%v marks=%#v", kinds, marks)
	}

	err = CheckIllegalE2EMarks([]string{pw, ok})
	if err == nil || !strings.Contains(err.Error(), "@playwright/test") || !strings.Contains(err.Error(), pw) {
		t.Fatalf("Check: %v", err)
	}
}

func TestScanIllegalE2EMarksReadError(t *testing.T) {
	_, err := ScanIllegalE2EMarks([]string{filepath.Join(t.TempDir(), "missing.spec.ts")})
	if err == nil || !strings.Contains(err.Error(), "read ") {
		t.Fatalf("got %v", err)
	}
}

func TestScanIllegalE2EMarksIgnoresCommentsWithoutImport(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ok.spec.ts")
	// Mention in a string / comment without import/require form should not match
	// when the regex requires from/import/require — bare words alone are fine.
	body := "// see playwright docs\nimport { test } from '@choysum/e2e';\nconst s = 'node:fs';\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	marks, err := ScanIllegalE2EMarks([]string{path})
	if err != nil {
		t.Fatal(err)
	}
	if len(marks) != 0 {
		t.Fatalf("unexpected marks: %#v", marks)
	}
}
