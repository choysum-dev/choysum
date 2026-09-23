// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package policy

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/internal/parser"
)

func writePartnerWebModule(t *testing.T, modulesPath, name string) string {
	t.Helper()
	webDir := filepath.Join(modulesPath, name, "web")
	if err := os.MkdirAll(webDir, 0o755); err != nil {
		t.Fatal(err)
	}
	pkg := fmt.Sprintf(`{
  "name": "@choysum-dev/%s",
  "choysum": { "moduleName": "%s", "application": "%s" }
}`, name, name, name)
	if err := os.WriteFile(filepath.Join(modulesPath, name, "package.json"), []byte(pkg), 0o644); err != nil {
		t.Fatal(err)
	}
	return webDir
}

func TestAssertNoForbiddenUiImports_IgnoresHTMLCommentedScript(t *testing.T) {
	modulesPath := t.TempDir()
	webDir := writePartnerWebModule(t, modulesPath, "partner")
	vue := `<template><div /></template>
<!--
<script setup lang="ts">
import { DialogRoot } from 'reka-ui';
</script>
-->
<script setup lang="ts">
import { ref } from 'vue';
const n = ref(1);
</script>
`
	if err := os.WriteFile(filepath.Join(webDir, "Doc.vue"), []byte(vue), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := AssertNoForbiddenUiImports(modulesPath, "partner"); err != nil {
		t.Fatalf("commented script must be ignored: %v", err)
	}
}

func TestAssertNoForbiddenUiImports_AllowsTypeOnlyImport(t *testing.T) {
	modulesPath := t.TempDir()
	webDir := writePartnerWebModule(t, modulesPath, "partner")
	src := "import type { DialogRoot } from 'reka-ui';\nexport type { DialogRoot } from 'reka-ui';\nexport type X = DialogRoot;\n"
	if err := os.WriteFile(filepath.Join(webDir, "types.ts"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := AssertNoForbiddenUiImports(modulesPath, "partner"); err != nil {
		t.Fatalf("type-only import/re-export must be allowed: %v", err)
	}
}

func TestAssertNoForbiddenUiImports_RejectsCRLFVue(t *testing.T) {
	modulesPath := t.TempDir()
	webDir := writePartnerWebModule(t, modulesPath, "partner")
	vue := "<template><div /></template>\r\n<script setup lang=\"ts\">\r\nimport { DialogRoot } from 'reka-ui';\r\nexport const x = DialogRoot;\r\n</script>\r\n"
	if err := os.WriteFile(filepath.Join(webDir, "CRLF.vue"), []byte(vue), 0o644); err != nil {
		t.Fatal(err)
	}
	err := AssertNoForbiddenUiImports(modulesPath, "partner")
	if err == nil || !strings.Contains(err.Error(), "reka-ui") {
		t.Fatalf("expected reka-ui ban from CRLF SFC, got %v", err)
	}
}

func TestAssertNoForbiddenUiImports_RejectsScriptAttrWithGt(t *testing.T) {
	modulesPath := t.TempDir()
	webDir := writePartnerWebModule(t, modulesPath, "partner")
	vue := `<template><div /></template>
<script setup lang="ts" data-x=">">
import { DialogRoot } from 'reka-ui';
export const x = DialogRoot;
</script>
`
	if err := os.WriteFile(filepath.Join(webDir, "AttrGt.vue"), []byte(vue), 0o644); err != nil {
		t.Fatal(err)
	}
	err := AssertNoForbiddenUiImports(modulesPath, "partner")
	if err == nil || !strings.Contains(err.Error(), "reka-ui") {
		t.Fatalf("expected reka-ui ban with '>' in script attr, got %v", err)
	}
}

func TestAssertNoForbiddenUiImports_RejectsCompactOneLineVue(t *testing.T) {
	modulesPath := t.TempDir()
	webDir := writePartnerWebModule(t, modulesPath, "partner")
	vue := `<template><div/></template><script setup lang="ts">import { DialogRoot } from 'reka-ui';export const x = DialogRoot;</script><style scoped></style>`
	if err := os.WriteFile(filepath.Join(webDir, "Compact.vue"), []byte(vue), 0o644); err != nil {
		t.Fatal(err)
	}
	err := AssertNoForbiddenUiImports(modulesPath, "partner")
	if err == nil || !strings.Contains(err.Error(), "reka-ui") {
		t.Fatalf("expected reka-ui ban from compact one-line SFC, got %v", err)
	}
}

func TestAssertNoForbiddenUiImports_RejectsScriptWithCommentLiteral(t *testing.T) {
	modulesPath := t.TempDir()
	webDir := writePartnerWebModule(t, modulesPath, "partner")
	// A "<!--" string in script plus a later "-->" must not blank the real
	// script block (only HTML comments that embed <script> are masked).
	vue := `<template><div /></template>
<script setup lang="ts">
const marker = "<!--";
import { DialogRoot } from 'reka-ui';
export const x = DialogRoot;
</script>
<!-- trailing -->
`
	if err := os.WriteFile(filepath.Join(webDir, "Literal.vue"), []byte(vue), 0o644); err != nil {
		t.Fatal(err)
	}
	err := AssertNoForbiddenUiImports(modulesPath, "partner")
	if err == nil || !strings.Contains(err.Error(), "reka-ui") {
		t.Fatalf("expected reka-ui ban despite comment literal, got %v", err)
	}
}

func TestAssertNoForbiddenUiImports_RejectsScriptWithCloseTagLiteral(t *testing.T) {
	modulesPath := t.TempDir()
	webDir := writePartnerWebModule(t, modulesPath, "partner")
	vue := `<template><div /></template>
<script setup lang="ts">
const close = "</script>";
import { DialogRoot } from 'reka-ui';
export const x = DialogRoot;
</script>
`
	if err := os.WriteFile(filepath.Join(webDir, "CloseLit.vue"), []byte(vue), 0o644); err != nil {
		t.Fatal(err)
	}
	err := AssertNoForbiddenUiImports(modulesPath, "partner")
	if err == nil || !strings.Contains(err.Error(), "reka-ui") {
		t.Fatalf("expected reka-ui ban despite </script> literal, got %v", err)
	}
}

func TestAssertNoForbiddenUiImports_RejectsWebModuleUntilCutover(t *testing.T) {
	modulesPath := t.TempDir()
	webDir := writePartnerWebModule(t, modulesPath, "web")
	src := "import { DialogRoot } from 'reka-ui';\nexport const x = DialogRoot;\n"
	if err := os.WriteFile(filepath.Join(webDir, "leak.ts"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	err := AssertNoForbiddenUiImports(modulesPath, "web")
	if err == nil || !strings.Contains(err.Error(), "reka-ui") {
		t.Fatalf("product web must still be gated until cutover, got %v", err)
	}
}

func TestAssertNoForbiddenUiImports_SkipsDeclarationFiles(t *testing.T) {
	modulesPath := t.TempDir()
	webDir := writePartnerWebModule(t, modulesPath, "partner")
	for _, name := range []string{"ambient.d.ts", "ambient.d.mts", "ambient.d.cts"} {
		src := "import { DialogRoot } from 'reka-ui';\nexport {};\n"
		if err := os.WriteFile(filepath.Join(webDir, name), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := AssertNoForbiddenUiImports(modulesPath, "partner"); err != nil {
		t.Fatalf("declaration files must be skipped: %v", err)
	}
}

func TestAssertNoForbiddenUiImports_MultiScriptAndJSX(t *testing.T) {
	modulesPath := t.TempDir()
	webDir := writePartnerWebModule(t, modulesPath, "partner")
	vue := `<template><div /></template>
<script lang="ts">
export default {};
</script>
<script setup lang="ts">
import { DialogRoot } from 'reka-ui';
</script>
`
	if err := os.WriteFile(filepath.Join(webDir, "Multi.vue"), []byte(vue), 0o644); err != nil {
		t.Fatal(err)
	}
	jsx := "import { DialogRoot } from 'reka-ui';\nexport const x = DialogRoot;\n"
	if err := os.WriteFile(filepath.Join(webDir, "Leak.jsx"), []byte(jsx), 0o644); err != nil {
		t.Fatal(err)
	}
	tsx := "import { DialogRoot } from 'reka-ui';\nexport const y = DialogRoot;\n"
	if err := os.WriteFile(filepath.Join(webDir, "Leak.tsx"), []byte(tsx), 0o644); err != nil {
		t.Fatal(err)
	}
	err := AssertNoForbiddenUiImports(modulesPath, "partner")
	if err == nil || !strings.Contains(err.Error(), "reka-ui") {
		t.Fatalf("expected reka-ui ban from multi-script/jsx, got %v", err)
	}
}

func TestAssertNoForbiddenUiImports_EmptyScriptAndNoScript(t *testing.T) {
	modulesPath := t.TempDir()
	webDir := writePartnerWebModule(t, modulesPath, "partner")
	if err := os.WriteFile(filepath.Join(webDir, "Empty.vue"), []byte(`<template><div/></template>
<script setup lang="ts">
</script>
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(webDir, "NoScript.vue"), []byte(`<template><div/></template>
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := AssertNoForbiddenUiImports(modulesPath, "partner"); err != nil {
		t.Fatalf("empty/no script vue must pass: %v", err)
	}
}

func TestAssertNoForbiddenUiImports_SkipsScanDirs(t *testing.T) {
	modulesPath := t.TempDir()
	webDir := writePartnerWebModule(t, modulesPath, "partner")
	leak := "import { DialogRoot } from 'reka-ui';\nexport const x = DialogRoot;\n"
	for _, dir := range []string{"node_modules", "dist", ".choysum", "tmp", ".git", "public", "coverage"} {
		nested := filepath.Join(webDir, dir)
		if err := os.MkdirAll(nested, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(nested, "leak.ts"), []byte(leak), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(webDir, "readme.md"), []byte("import 'reka-ui'"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := AssertNoForbiddenUiImports(modulesPath, "partner"); err != nil {
		t.Fatalf("skipped dirs / non-source must not fail: %v", err)
	}
}

func TestScanForbiddenUiImportsOnDisk_EdgeInputs(t *testing.T) {
	if v, err := ScanForbiddenUiImportsOnDisk(ForbiddenUiImportScanInput{}); err != nil || v != nil {
		t.Fatalf("empty input: got %v %v", v, err)
	}
	modulesPath := t.TempDir()
	if v, err := ScanForbiddenUiImportsOnDisk(ForbiddenUiImportScanInput{
		ModulesPath: modulesPath,
		ModuleName:  "choy_ui",
		ModuleRoot:  filepath.Join(modulesPath, "choy_ui"),
	}); err != nil || v != nil {
		t.Fatalf("kit host: got %v %v", v, err)
	}

	rootFile := filepath.Join(modulesPath, "filemod")
	if err := os.WriteFile(rootFile, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := ScanForbiddenUiImportsOnDisk(ForbiddenUiImportScanInput{
		ModulesPath: modulesPath,
		ModuleName:  "filemod",
		ModuleRoot:  rootFile,
	})
	if err == nil || !strings.Contains(err.Error(), "not a directory") {
		t.Fatalf("expected not a directory, got %v", err)
	}

	modRoot := filepath.Join(modulesPath, "noweb")
	if err := os.MkdirAll(modRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if v, err := ScanForbiddenUiImportsOnDisk(ForbiddenUiImportScanInput{
		ModulesPath: modulesPath,
		ModuleName:  "noweb",
		ModuleRoot:  modRoot,
	}); err != nil || v != nil {
		t.Fatalf("no web: got %v %v", v, err)
	}

	webFileMod := filepath.Join(modulesPath, "webfile")
	if err := os.MkdirAll(webFileMod, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(webFileMod, "web"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if v, err := ScanForbiddenUiImportsOnDisk(ForbiddenUiImportScanInput{
		ModulesPath: modulesPath,
		ModuleName:  "webfile",
		ModuleRoot:  webFileMod,
	}); err != nil || v != nil {
		t.Fatalf("web file: got %v %v", v, err)
	}
}

func TestScanForbiddenUiImportsOnDisk_StatErrors(t *testing.T) {
	modulesPath := t.TempDir()
	modRoot := filepath.Join(modulesPath, "solo")
	if err := os.MkdirAll(filepath.Join(modRoot, "web"), 0o755); err != nil {
		t.Fatal(err)
	}

	orig := statPath
	t.Cleanup(func() { statPath = orig })

	statPath = func(path string) (os.FileInfo, error) {
		return nil, fmt.Errorf("permission denied")
	}
	_, err := ScanForbiddenUiImportsOnDisk(ForbiddenUiImportScanInput{
		ModulesPath: modulesPath,
		ModuleName:  "solo",
		ModuleRoot:  modRoot,
	})
	if err == nil || !strings.Contains(err.Error(), "stat module root") {
		t.Fatalf("expected stat module root error, got %v", err)
	}

	calls := 0
	statPath = func(path string) (os.FileInfo, error) {
		calls++
		if calls == 1 {
			return orig(path)
		}
		return nil, fmt.Errorf("permission denied")
	}
	_, err = ScanForbiddenUiImportsOnDisk(ForbiddenUiImportScanInput{
		ModulesPath: modulesPath,
		ModuleName:  "solo",
		ModuleRoot:  modRoot,
	})
	if err == nil || !strings.Contains(err.Error(), "stat web dir") {
		t.Fatalf("expected stat web dir error, got %v", err)
	}
}

func TestScanForbiddenUiImportsOnDisk_WalkAndParseErrors(t *testing.T) {
	modulesPath := t.TempDir()
	webDir := writePartnerWebModule(t, modulesPath, "solo")
	modRoot := filepath.Join(modulesPath, "solo")

	origWalk := walkWebTree
	t.Cleanup(func() { walkWebTree = origWalk })

	walkWebTree = func(_ string, walkFn fs.WalkDirFunc) error {
		return walkFn("", nil, fmt.Errorf("walk denied"))
	}
	_, err := ScanForbiddenUiImportsOnDisk(ForbiddenUiImportScanInput{
		ModulesPath: modulesPath,
		ModuleName:  "solo",
		ModuleRoot:  modRoot,
		PathAlias:   ModulePathAliasForBoundary(modulesPath),
	})
	if err == nil || !strings.Contains(err.Error(), "walk denied") {
		t.Fatalf("expected walk error, got %v", err)
	}

	walkWebTree = origWalk
	if err := os.WriteFile(filepath.Join(webDir, "empty.ts"), []byte("   \n\t"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ScanForbiddenUiImportsOnDisk(ForbiddenUiImportScanInput{
		ModulesPath: modulesPath,
		ModuleName:  "solo",
		ModuleRoot:  modRoot,
		PathAlias:   ModulePathAliasForBoundary(modulesPath),
	}); err != nil {
		t.Fatalf("whitespace-only source must be skipped, got %v", err)
	}

	origParse := parseWebImportFile
	t.Cleanup(func() { parseWebImportFile = origParse })
	parseWebImportFile = func(map[string]string, string, []byte) (*parser.ParserResult, error) {
		return nil, fmt.Errorf("parse boom")
	}
	if err := os.WriteFile(filepath.Join(webDir, "ok.ts"), []byte("export {};\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = ScanForbiddenUiImportsOnDisk(ForbiddenUiImportScanInput{
		ModulesPath: modulesPath,
		ModuleName:  "solo",
		ModuleRoot:  modRoot,
		PathAlias:   ModulePathAliasForBoundary(modulesPath),
	})
	if err == nil || !strings.Contains(err.Error(), "ok.ts") {
		t.Fatalf("expected parse error for ok.ts, got %v", err)
	}

	parseWebImportFile = func(map[string]string, string, []byte) (*parser.ParserResult, error) {
		return nil, nil
	}
	if _, err := ScanForbiddenUiImportsOnDisk(ForbiddenUiImportScanInput{
		ModulesPath: modulesPath,
		ModuleName:  "solo",
		ModuleRoot:  modRoot,
		PathAlias:   ModulePathAliasForBoundary(modulesPath),
	}); err != nil {
		t.Fatalf("nil parser result must be skipped, got %v", err)
	}
}

func TestScanForbiddenUiImportsOnDisk_ReadFileError(t *testing.T) {
	modulesPath := t.TempDir()
	webDir := writePartnerWebModule(t, modulesPath, "solo")
	modRoot := filepath.Join(modulesPath, "solo")
	path := filepath.Join(webDir, "leak.ts")
	if err := os.WriteFile(path, []byte("export {};\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(path, 0o644) })

	_, err := ScanForbiddenUiImportsOnDisk(ForbiddenUiImportScanInput{
		ModulesPath: modulesPath,
		ModuleName:  "solo",
		ModuleRoot:  modRoot,
		PathAlias:   ModulePathAliasForBoundary(modulesPath),
	})
	if err == nil {
		t.Skip("chmod 0 still readable in this environment")
	}
	if !strings.Contains(err.Error(), "read ") {
		t.Fatalf("expected read error, got %v", err)
	}
}

func TestCheckForbiddenUiImportsOnDisk_EmptyAndAlias(t *testing.T) {
	if err := CheckForbiddenUiImportsOnDisk("", "partner", nil); err != nil {
		t.Fatal(err)
	}
	if err := CheckForbiddenUiImportsOnDisk(t.TempDir(), "", nil); err != nil {
		t.Fatal(err)
	}
	if err := CheckForbiddenUiImportsOnDisk(t.TempDir(), "choy_ui", nil); err != nil {
		t.Fatal(err)
	}

	modulesPath := t.TempDir()
	webDir := writePartnerWebModule(t, modulesPath, "partner")
	if err := os.WriteFile(filepath.Join(webDir, "ok.ts"), []byte("export {};\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := CheckForbiddenUiImportsOnDisk(modulesPath, "partner", nil); err != nil {
		t.Fatal(err)
	}
}

func TestCheckForbiddenUiImports_UnitBranches(t *testing.T) {
	if CheckForbiddenUiImports(ForbiddenUiImportScanInput{}, nil) != nil {
		t.Fatal("empty input")
	}
	if CheckForbiddenUiImports(ForbiddenUiImportScanInput{
		ModuleName: "choy_ui",
		ModuleRoot: "/x",
	}, nil) != nil {
		t.Fatal("kit host")
	}

	root := "/modules/partner"
	results := []*parser.ParserResult{
		nil,
		{Path: "/modules/partner/service/x.ts", Imports: map[string]*parser.Import{
			"X": {ModuleSpecText: "reka-ui", Line: 1, Column: 1},
		}},
		{
			Path: "/modules/partner/web/a.ts",
			Imports: map[string]*parser.Import{
				"nil":      nil,
				"typeOnly": {ModuleSpecText: "reka-ui", IsTypeOnly: true, Line: 1, Column: 1},
				"B":        {ModuleSpecText: "reka-ui", Line: 2, Column: 5},
				"A":        {ModuleSpecText: "@unovis/vue", Line: 2, Column: 1},
			},
			DynamicImports: []*parser.Import{
				nil,
				{ModuleSpecText: "reka-ui", IsTypeOnly: true, Line: 3, Column: 1},
				{ModuleSpecText: "reka-ui", Line: 4, Column: 1},
			},
			Exports: map[string]*parser.Export{
				"typeRe": {ModuleSpecPath: "reka-ui", IsTypeOnly: true, Line: 5, Column: 1},
				"*": {
					Wildcard: []*parser.Export{
						nil,
						{ModuleSpecPath: "reka-ui", IsTypeOnly: true, Line: 6, Column: 1},
						{ModuleSpecPath: "reka-ui", Line: 7, Column: 1},
					},
				},
				"named": {ModuleSpecPath: "../components/vendor/ui/button", Line: 8, Column: 1},
			},
		},
		{
			Path: "/modules/partner/web/b.ts",
			Imports: map[string]*parser.Import{
				"Z": {ModuleSpecText: "reka-ui", Line: 1, Column: 2},
				"Y": {ModuleSpecText: "reka-ui", Line: 1, Column: 1},
				"W": {ModuleSpecText: "ui/button", Line: 1, Column: 1},
			},
		},
	}
	violations := CheckForbiddenUiImports(ForbiddenUiImportScanInput{
		ModuleName: "partner",
		ModuleRoot: root,
	}, results)
	if len(violations) < 4 {
		t.Fatalf("expected multiple violations, got %#v", violations)
	}

	// Same statement, multiple bindings → one violation per line/rule after dedupe.
	dup := CheckForbiddenUiImports(ForbiddenUiImportScanInput{
		ModuleName: "partner",
		ModuleRoot: root,
	}, []*parser.ParserResult{{
		Path: "/modules/partner/web/dup.ts",
		Imports: map[string]*parser.Import{
			"A": {ModuleSpecText: "reka-ui", Line: 1, Column: 10},
			"B": {ModuleSpecText: "reka-ui", Line: 1, Column: 13},
			"C": {ModuleSpecText: "reka-ui", Line: 1, Column: 16},
		},
	}})
	if len(dup) != 1 || dup[0].Rule != "reka-ui" {
		t.Fatalf("expected one deduped reka-ui violation, got %#v", dup)
	}

	sortedRules := CheckForbiddenUiImports(ForbiddenUiImportScanInput{
		ModuleName: "partner",
		ModuleRoot: root,
	}, []*parser.ParserResult{{
		Path: "/modules/partner/web/rules.ts",
		Imports: map[string]*parser.Import{
			"U": {ModuleSpecText: "@unovis/vue", Line: 2, Column: 1},
			"R": {ModuleSpecText: "reka-ui", Line: 2, Column: 20},
		},
	}})
	if len(sortedRules) != 2 || sortedRules[0].Rule > sortedRules[1].Rule {
		t.Fatalf("expected rule-ordered violations, got %#v", sortedRules)
	}

	sameCol := CheckForbiddenUiImports(ForbiddenUiImportScanInput{
		ModuleName: "partner",
		ModuleRoot: root,
	}, []*parser.ParserResult{{
		Path: "/modules/partner/web/spec.ts",
		Imports: map[string]*parser.Import{
			"A": {ModuleSpecText: "reka-ui/Dialog", Line: 1, Column: 1},
			"B": {ModuleSpecText: "reka-ui", Line: 1, Column: 1},
		},
	}})
	if len(sameCol) != 1 {
		t.Fatalf("expected deduped same-line reka-ui, got %#v", sameCol)
	}
}

func TestClassifyAndPathHelpers_ExtraCases(t *testing.T) {
	cases := []struct {
		spec string
		want string
	}{
		{"", ""},
		{"  ", ""},
		{"@unovis", "@unovis"},
		{"ui", "ui/*"},
		{"/components/vendor/ui", "ui/*"},
		{"/components/ui", "ui/*"},
		{"/web/components/ui", "ui/*"},
		{"/web/components/ui/", "ui/*"},
		{"internal", "internal/*"},
		{"/components/internal", "internal/*"},
		{"/web/components/internal", "internal/*"},
		{"/web/components/internal/", "internal/*"},
		{"choy_ui/web/components/x", "choy_ui-deep"},
		{"choy_ui/web/lib/x", "choy_ui-deep"},
		{"choy_ui/other", ""},
		{"@/choy_ui/web/lib/utils", "choy_ui-deep"},
	}
	for _, tc := range cases {
		if got := classifyForbiddenUiImport(tc.spec); got != tc.want {
			t.Fatalf("classify(%q)=%q want %q", tc.spec, got, tc.want)
		}
	}

	if appendForbiddenUiSpec(nil, "p", `""`, 1, 1) != nil {
		t.Fatal("empty quoted spec")
	}
	backtick := appendForbiddenUiSpec(nil, "p", "`reka-ui`", 1, 1)
	if len(backtick) != 1 || backtick[0].SpecText != "reka-ui" || backtick[0].Rule != "reka-ui" {
		t.Fatalf("backtick specifier: %#v", backtick)
	}
	if appendForbiddenUiExports(nil, "p", nil) != nil {
		t.Fatal("nil export")
	}
	// Parent ModuleSpecPath fallback when wildcard entries lack a specifier.
	got := appendForbiddenUiExports(nil, "p", &parser.Export{
		ModuleSpecPath: "reka-ui",
		Line:           3,
		Column:         1,
		Wildcard: []*parser.Export{
			nil,
			{ModuleSpecPath: "", Line: 3, Column: 1},
		},
	})
	if len(got) != 1 || got[0].SpecText != "reka-ui" || got[0].Line != 3 {
		t.Fatalf("parent fallback: %#v", got)
	}
}

func TestIsModuleWebSource_Edges(t *testing.T) {
	if IsModuleWebSource("", "/x/web") {
		t.Fatal("empty root")
	}
	if IsModuleWebSource("/modules/partner", "") {
		t.Fatal("empty path")
	}
	if IsModuleWebSource(".", "/modules/partner/web") {
		t.Fatal("dot root")
	}
	if IsModuleWebSource("/modules/partner", "/modules/partner") {
		t.Fatal("exact root is not web")
	}
	if !IsModuleWebSource("/modules/partner", "/modules/partner/web") {
		t.Fatal("web dir itself")
	}
}

func TestWebImportHelpers_Direct(t *testing.T) {
	if !shouldSkipWebScanDir("node_modules") || shouldSkipWebScanDir("pages") {
		t.Fatal("shouldSkipWebScanDir")
	}
	if isWebImportSource("x.d.ts") || isWebImportSource("x.d.mts") || isWebImportSource("x.d.cts") {
		t.Fatal("declarations")
	}
	if !isWebImportSource("x.vue") || !isWebImportSource("x.mts") || isWebImportSource("x.css") {
		t.Fatal("isWebImportSource")
	}

	if got := webImportVirtualPath("/a/Foo.vue", 0); !strings.HasSuffix(got, "Foo.vue.webimport.ts") {
		t.Fatalf("vue virtual path: %s", got)
	}
	if got := webImportVirtualPath("/a/Foo.ts", 0); !strings.HasSuffix(got, "Foo.ts.webimport.ts") {
		t.Fatalf("ts virtual path: %s", got)
	}
	if got := webImportVirtualPath("/a/Foo.tsx", 1); !strings.HasSuffix(got, "Foo.tsx.webimport.1.tsx") {
		t.Fatalf("tsx indexed: %s", got)
	}
	if got := webImportVirtualPath("/a/Foo.jsx", 0); !strings.HasSuffix(got, "Foo.jsx.webimport.jsx") {
		t.Fatalf("jsx: %s", got)
	}

	if itoa(0) != "0" || itoa(12) != "12" || itoa(7) != "7" {
		t.Fatalf("itoa: %q %q %q", itoa(0), itoa(12), itoa(7))
	}

	adjustParserResultLines(nil, 1)
	adjustParserResultLines(&parser.ParserResult{}, 0)
	r := &parser.ParserResult{
		Imports: map[string]*parser.Import{
			"a": nil,
			"b": {Line: 0},
			"c": {Line: 2},
		},
		DynamicImports: []*parser.Import{
			nil,
			{Line: 0},
			{Line: 3},
		},
		Exports: map[string]*parser.Export{
			"n": nil,
			"e": {Line: 0, Wildcard: []*parser.Export{nil, {Line: 0}, {Line: 4}}},
			"f": {Line: 5},
		},
	}
	adjustParserResultLines(r, 10)
	if r.Imports["c"].Line != 12 || r.DynamicImports[2].Line != 13 {
		t.Fatalf("import lines: %#v", r)
	}
	if r.Exports["f"].Line != 15 || r.Exports["e"].Wildcard[2].Line != 14 {
		t.Fatalf("export lines: %#v", r.Exports)
	}

	masked := maskVueHTMLComments([]byte("a<!--x\ny-->b"))
	if string(masked) != "a<!--x\ny-->b" {
		t.Fatalf("non-script comment must stay: %q", masked)
	}
	scriptComment := []byte("a<!--\n<script>import 'reka-ui'</script>\n-->b")
	maskedScript := maskVueHTMLComments(scriptComment)
	if bytes.Contains(maskedScript, []byte("<script>")) {
		t.Fatalf("script-bearing comment must be blanked: %q", maskedScript)
	}
}

func TestFormatForbiddenUiImportError_NoLine(t *testing.T) {
	err := FormatForbiddenUiImportError([]ForbiddenUiImportViolation{{
		SourcePath: "web/x.ts",
		SpecText:   "reka-ui",
		Rule:       "reka-ui",
	}})
	if err == nil || !strings.Contains(err.Error(), "web/x.ts imports") {
		t.Fatalf("got %v", err)
	}
}
