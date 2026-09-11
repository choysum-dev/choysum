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

func TestScanIllegalE2EMarksMultilineAndComments(t *testing.T) {
	dir := t.TempDir()
	multi := filepath.Join(dir, "multi.spec.ts")
	noise := filepath.Join(dir, "noise.spec.ts")
	escaped := filepath.Join(dir, "escaped.spec.ts")
	dup := filepath.Join(dir, "dup.spec.ts")
	ident := filepath.Join(dir, "ident.spec.ts")

	if err := os.WriteFile(multi, []byte("const m = await import(\n  '@playwright/test'\n);\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	body := "// import { test } from '@playwright/test'\n" +
		"/* require('node:fs') */\n" +
		"import { test } from '@choysum/e2e';\n" +
		"const s = 'import(\"@playwright/test\")';\n" +
		"const t = `node:path`;\n"
	if err := os.WriteFile(noise, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(escaped, []byte("const s = \"abc\\\"def\";\nimport { test } from '@choysum/e2e';\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dup, []byte("import '@playwright/test';\nimport { test } from '@playwright/test';\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ident, []byte("const myimport = 'x';\nimport { test } from '@choysum/e2e';\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	marks, err := ScanIllegalE2EMarks([]string{multi, noise, escaped, dup, ident})
	if err != nil {
		t.Fatal(err)
	}
	// multiline + dup file (one mark per line, deduped per line → 2 on dup + 1 multi)
	pw := 0
	for _, m := range marks {
		if m.Kind == "playwright" {
			pw++
		}
	}
	if pw < 2 {
		t.Fatalf("expected multiline+dup playwright marks, got %#v", marks)
	}
	for _, m := range marks {
		if m.Path == noise || m.Path == escaped || m.Path == ident {
			t.Fatalf("false positive on %s: %#v", m.Path, m)
		}
	}
}

func TestScanIllegalE2EMarksReadError(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.spec.ts")
	_, err := ScanIllegalE2EMarks([]string{missing})
	if err == nil || !strings.Contains(err.Error(), "read ") {
		t.Fatalf("Scan got %v", err)
	}
	err = CheckIllegalE2EMarks([]string{missing})
	if err == nil || !strings.Contains(err.Error(), "read ") {
		t.Fatalf("Check got %v", err)
	}
}

func TestBlankJSCommentsAndNonModuleStrings(t *testing.T) {
	in := "" +
		"// line comment\n" +
		"/* block\ncomment */\n" +
		"const a = 'keep\\'me';\n" +
		"const b = \"x\\\"y\";\n" +
		"const c = `tpl\nline`;\n" +
		"from 'mod';\n" +
		"import('mod')\n" +
		"require(\"mod\")\n" +
		"import 'mod'\n"
	out := blankJSCommentsAndNonModuleStrings(in)
	if !strings.Contains(out, "from 'mod'") || !strings.Contains(out, "import('mod')") {
		t.Fatalf("module strings must remain: %q", out)
	}
	if strings.Contains(out, "line comment") || strings.Contains(out, "block") {
		t.Fatalf("comments should be blanked: %q", out)
	}
	if strings.Contains(out, "keep") || strings.Contains(out, "tpl") {
		t.Fatalf("non-module strings should be blanked: %q", out)
	}
	if strings.Count(out, "\n") != strings.Count(in, "\n") {
		t.Fatalf("newline count changed")
	}

	// Edge: quote at start, short ident, parenthesized import with whitespace.
	_ = blankJSCommentsAndNonModuleStrings("'alone'")
	_ = blankJSCommentsAndNonModuleStrings("(\n  'mod'\n)")
	_ = isModuleSpecifierContext("from 'x'", 5)
	_ = isModuleSpecifierContext("'x'", 0)
	_ = hasIdentBefore("ab", 0, "import")
	_ = hasIdentBefore("ximport", 6, "import")

	// Same-line duplicate matches exercise seen[lineNo] dedupe.
	sameLine := filepath.Join(t.TempDir(), "same.spec.ts")
	if err := os.WriteFile(sameLine, []byte("import '@playwright/test'; export * from '@playwright/test';\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	marks, err := ScanIllegalE2EMarks([]string{sameLine})
	if err != nil {
		t.Fatal(err)
	}
	if len(marks) != 1 {
		t.Fatalf("expected 1 deduped mark, got %#v", marks)
	}

	// Escaped chars inside kept module specifier + spaced import(.
	_ = blankJSCommentsAndNonModuleStrings("from 'a\\b'")
	_ = blankJSCommentsAndNonModuleStrings("from `mod\nname`")
	esc := filepath.Join(t.TempDir(), "esc.spec.ts")
	if err := os.WriteFile(esc, []byte("const x = await import  (\n  'node:fs'\n);\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	marks, err = ScanIllegalE2EMarks([]string{esc})
	if err != nil || len(marks) != 1 || marks[0].Kind != "node-builtin" {
		t.Fatalf("spaced module import: marks=%#v err=%v", marks, err)
	}
}

func TestFindIllegalMarksEmptySnippetFallback(t *testing.T) {
	// Empty source lines force the match-text snippet fallback.
	marks := findIllegalMarks("x.spec.ts", "", "import('@playwright/test')", illegalPlaywrightImportRE, "playwright")
	if len(marks) != 1 || marks[0].Line == "" {
		t.Fatalf("got %#v", marks)
	}
}

func TestScanIllegalE2EMarksIgnoresCommentsWithoutImport(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ok.spec.ts")
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
