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

	// Subpaths (node:fs/promises, @playwright/test/reporter) must also fail the gate.
	pwSub := filepath.Join(dir, "pw-sub.spec.ts")
	fsSub := filepath.Join(dir, "fs-sub.spec.ts")
	write(pwSub, "import { reporter } from '@playwright/test/reporter';\n")
	write(fsSub, "import fs from 'node:fs/promises';\n")
	marks, err = ScanIllegalE2EMarks([]string{pwSub, fsSub})
	if err != nil {
		t.Fatal(err)
	}
	if len(marks) != 2 {
		t.Fatalf("expected 2 subpath marks, got %#v", marks)
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
	if pw != 3 {
		t.Fatalf("expected 3 playwright marks (1 multiline + 2 dup), got %#v", marks)
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

func TestScanIllegalE2EMarksIgnoresTopLevelRegexp(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "re.spec.ts")
	// Top-level regexp text must not false-positive as an illegal import.
	body := "const re = /from 'node:fs'/;\nimport { test } from '@choysum/e2e';\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	marks, err := ScanIllegalE2EMarks([]string{path})
	if err != nil {
		t.Fatal(err)
	}
	if len(marks) != 0 {
		t.Fatalf("unexpected marks from regexp literal: %#v", marks)
	}
}

func TestScanIllegalE2EMarksTemplateInterpAndBackticks(t *testing.T) {
	dir := t.TempDir()
	interp := filepath.Join(dir, "interp.spec.ts")
	backtickPW := filepath.Join(dir, "bt-pw.spec.ts")
	backtickNode := filepath.Join(dir, "bt-node.spec.ts")
	nested := filepath.Join(dir, "nested.spec.ts")
	reClass := filepath.Join(dir, "re-class.spec.ts")

	write := func(path, body string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(interp, "const x = `prefix ${await import(\"node:fs\")} suffix`;\n")
	write(backtickPW, "import { test } from `@playwright/test`;\n")
	write(backtickNode, "const m = await import(`node:path`);\n")
	write(nested, "const x = `a ${/* c */ await import('node:crypto') /* d */} b`;\n")
	// Regexp character-class `}` must not close `${...}` before the real import.
	write(reClass, "const x = `${/[}]/.test(s) ? import(\"node:fs\") : null}`;\n")

	marks, err := ScanIllegalE2EMarks([]string{interp, backtickPW, backtickNode, nested, reClass})
	if err != nil {
		t.Fatal(err)
	}
	kinds := map[string]int{}
	for _, m := range marks {
		kinds[m.Kind]++
	}
	if kinds["playwright"] != 1 || kinds["node-builtin"] != 4 {
		t.Fatalf("kinds=%v marks=%#v", kinds, marks)
	}

	// Nested template / quoted braces inside ${} exercise findTemplateInterpClose helpers.
	_ = blankJSCommentsAndNonModuleStrings("`outer ${'a{b}' + `inner ${1}`} end`")
	_ = blankJSCommentsAndNonModuleStrings("`x ${ // c\ny } z`")
	_ = blankJSCommentsAndNonModuleStrings("const s = 'a\\\nb';")   // line-continuation escape while blanking
	_ = blankJSCommentsAndNonModuleStrings("`x ${ {a: 1} } y`")     // nested braces bump depth
	_ = blankJSCommentsAndNonModuleStrings("`x ${ 'a\\'b' } y`")    // escaped quote inside interp
	_ = blankJSCommentsAndNonModuleStrings("`x ${ `a\\`b` } y`")    // escaped backtick in nested template
	_ = blankJSCommentsAndNonModuleStrings("`x ${ /* unterminated") // block comment hits EOF
	_ = blankJSCommentsAndNonModuleStrings("`x ${ 'unclosed")       // quoted string hits EOF
	_ = blankJSCommentsAndNonModuleStrings("`x ${ `unclosed")       // nested template hits EOF
	// Regexp helper edges: flags, escapes, division vs regexp, keywords, EOF.
	_ = blankJSCommentsAndNonModuleStrings("`x ${ /a\\/b/gi } y`")
	_ = blankJSCommentsAndNonModuleStrings("`x ${ return /foo/ } y`")
	_ = blankJSCommentsAndNonModuleStrings("`x ${ a / b } y`") // division, not regexp
	_ = blankJSCommentsAndNonModuleStrings("`x ${ /unterminated")
	_ = blankJSCommentsAndNonModuleStrings("`x ${ /foo\n } y`") // newline aborts regexp literal
	_ = skipJSRegexpLiteral("/foo\nbar", 0)
	_ = canStartJSRegexp(")/", 1)
	_ = canStartJSRegexp("]/", 1)
	_ = canStartJSRegexp("}/", 1)
	_ = canStartJSRegexp("\"/", 1)
	_ = canStartJSRegexp("++/", 2)
	_ = canStartJSRegexp("1/", 1)
	_ = canStartJSRegexp("/", 0)
	_ = canStartJSRegexp("+/", 1)
	_ = skipJSRegexpLiteral("x", 0)
	_ = isJSRegexpFlag('g')
	_ = isJSRegexpFlag('z')
}
