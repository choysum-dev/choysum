// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package policy

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/internal/parser"
	"github.com/choysum-dev/choysum/pkg/meta"
)

func TestModuleNameFromModulesPath_EdgeCases(t *testing.T) {
	if got := ModuleNameFromModulesPath("", "/x"); got != "" {
		t.Fatalf("empty modules path = %q", got)
	}
	if got := ModuleNameFromModulesPath("/modules", ""); got != "" {
		t.Fatalf("empty abs path = %q", got)
	}
	if got := ModuleNameFromModulesPath("/modules", "/outside/file.ts"); got != "" {
		t.Fatalf("outside modules = %q", got)
	}
	if got := ModuleNameFromModulesPath("/modules", "/modules/auth/service/a.ts"); got != "auth" {
		t.Fatalf("auth module = %q", got)
	}
}

func TestResolveModuleApplication_LookupAndFallback(t *testing.T) {
	lookup := ModuleApplicationLookupFromMap(map[string]string{"partner_bank": "partner"})
	if app, ok := ResolveModuleApplication("partner_bank", lookup); !ok || app != "partner" {
		t.Fatalf("lookup partner_bank = %q ok=%v", app, ok)
	}
	if app, ok := ResolveModuleApplication("auth", nil); !ok || app != "auth" {
		t.Fatalf("fallback auth = %q ok=%v", app, ok)
	}
	if _, ok := ResolveModuleApplication("", nil); ok {
		t.Fatal("empty name should not resolve")
	}
}

func TestModuleApplicationLookupWithDefault(t *testing.T) {
	lookup := ModuleApplicationLookupWithDefault(ModuleApplicationLookupFromMap(map[string]string{
		"meta": "meta",
	}))
	if app, ok := lookup("meta"); !ok || app != "meta" {
		t.Fatalf("map hit = %q ok=%v", app, ok)
	}
	if app, ok := lookup("partner_bank"); !ok || app != "partner" {
		t.Fatalf("fallback partner_bank = %q ok=%v", app, ok)
	}
	if lookup := ModuleApplicationLookupWithDefault(nil); lookup == nil {
		t.Fatal("expected non-nil lookup")
	}
}

func TestModuleApplicationLookupFromMap_NilMap(t *testing.T) {
	lookup := ModuleApplicationLookupFromMap(nil)
	if _, ok := lookup("auth"); ok {
		t.Fatal("nil map lookup should miss")
	}
}

func TestModuleApplicationLookupFromModule(t *testing.T) {
	if ModuleApplicationLookupFromModule(nil) != nil {
		t.Fatal("nil module should return nil lookup")
	}
	if ModuleApplicationLookupFromModule(&meta.Module{Name: "auth"}) != nil {
		t.Fatal("empty application should return nil lookup")
	}
	lookup := ModuleApplicationLookupFromModule(&meta.Module{Name: "auth", ApplicationStr: "auth"})
	if app, ok := lookup("auth"); !ok || app != "auth" {
		t.Fatalf("auth lookup = %q ok=%v", app, ok)
	}
}

func TestCollectModuleNamesFromParserResult(t *testing.T) {
	modulesPath := testModulesPath(t)
	names := make(map[string]struct{})
	CollectModuleNamesFromParserResult(modulesPath, nil, names)
	CollectModuleNamesFromParserResult(modulesPath, &parser.ParserResult{
		Imports: map[string]*parser.Import{
			"Role": {ModuleSpecPath: filepath.Join(modulesPath, "auth", "service", "models", "role")},
		},
		DynamicImports: []*parser.Import{
			{ModuleSpecPath: filepath.Join(modulesPath, "meta", "service", "models", "ui_resource")},
		},
		Exports: map[string]*parser.Export{
			"X": {
				ModuleSpecPath: filepath.Join(modulesPath, "base", "service", "models", "company"),
				Wildcard: []*parser.Export{{
					ModuleSpecPath: filepath.Join(modulesPath, "core", "service", "api", "dial"),
				}},
			},
		},
	}, names)
	for _, want := range []string{"auth", "meta", "base", "core"} {
		if _, ok := names[want]; !ok {
			t.Fatalf("missing module name %q in %#v", want, names)
		}
	}
}

func TestCheckServiceImportBoundary_RejectsSideEffectImport(t *testing.T) {
	modulesPath := testModulesPath(t)
	moduleRoot := filepath.Join(modulesPath, "partner")
	source := filepath.Join(moduleRoot, "service", "models", "partner.ts")
	authSpec := filepath.Join(modulesPath, "auth", "service", "models", "role")

	violations := CheckServiceImportBoundary(ServiceImportBoundaryInput{
		ModulesPath:       modulesPath,
		ModuleRoot:        moduleRoot,
		SourceApplication: "partner",
		Lookup:            testLookup(),
		ParserResults: []*parser.ParserResult{{
			Path: source,
			Imports: map[string]*parser.Import{
				parser.SideEffectImportMapKey(1, 1): {
					ModuleSpecPath: authSpec,
					IsTypeOnly:     false,
					Line:           1,
					Column:         1,
				},
			},
		}},
	})
	if len(violations) != 1 || violations[0].Kind != "import" {
		t.Fatalf("expected side-effect violation, got %#v", violations)
	}
}

func TestCheckServiceImportBoundary_AllowsExportType(t *testing.T) {
	modulesPath := testModulesPath(t)
	moduleRoot := filepath.Join(modulesPath, "partner")
	source := filepath.Join(moduleRoot, "service", "models", "partner.ts")
	authSpec := filepath.Join(modulesPath, "auth", "service", "models", "role")

	violations := CheckServiceImportBoundary(ServiceImportBoundaryInput{
		ModulesPath:       modulesPath,
		ModuleRoot:        moduleRoot,
		SourceApplication: "partner",
		Lookup:            testLookup(),
		ParserResults: []*parser.ParserResult{{
			Path: source,
			Exports: map[string]*parser.Export{
				"Role": {
					ModuleSpecPath: authSpec,
					IsTypeOnly:     true,
					Line:           2,
					Column:         1,
				},
			},
		}},
	})
	if len(violations) != 0 {
		t.Fatalf("expected no export type violations, got %#v", violations)
	}
}

func TestCheckServiceImportBoundary_RejectsValueExport(t *testing.T) {
	modulesPath := testModulesPath(t)
	moduleRoot := filepath.Join(modulesPath, "partner")
	source := filepath.Join(moduleRoot, "service", "models", "partner.ts")
	authSpec := filepath.Join(modulesPath, "auth", "service", "models", "role")

	violations := CheckServiceImportBoundary(ServiceImportBoundaryInput{
		ModulesPath:       modulesPath,
		ModuleRoot:        moduleRoot,
		SourceApplication: "partner",
		Lookup:            testLookup(),
		ParserResults: []*parser.ParserResult{{
			Path: source,
			Exports: map[string]*parser.Export{
				"Role": {
					ModuleSpecPath: authSpec,
					IsTypeOnly:     false,
					Line:           2,
					Column:         1,
				},
			},
		}},
	})
	if len(violations) != 1 || violations[0].Kind != "export" {
		t.Fatalf("expected export violation, got %#v", violations)
	}
}

func TestFormatImportBoundaryError_FallsBackToModuleSpecPath(t *testing.T) {
	err := FormatImportBoundaryError([]ImportBoundaryViolation{{
		SourcePath:        "/x.ts",
		Line:              1,
		Column:            1,
		SourceApplication: "partner",
		TargetApplication: "auth",
		Kind:              "import",
		ModuleSpecPath:    "/modules/auth/service/models/role",
	}})
	if err == nil || !strings.Contains(err.Error(), "/modules/auth/service/models/role") {
		t.Fatalf("error = %v", err)
	}
}

func TestBuildModuleApplicationLookupFromModulesDir(t *testing.T) {
	if _, err := BuildModuleApplicationLookupFromModulesDir(""); err == nil {
		t.Fatal("expected error for empty modules path")
	}
	modulesPath := filepath.Join(t.TempDir(), "modules")
	writeModulePackageJSON(t, modulesPath, "partner_bank", "partner")
	lookup, err := BuildModuleApplicationLookupFromModulesDir(modulesPath)
	if err != nil {
		t.Fatalf("BuildModuleApplicationLookupFromModulesDir() error = %v", err)
	}
	if app, ok := lookup("partner_bank"); !ok || app != "partner" {
		t.Fatalf("partner_bank app = %q ok=%v", app, ok)
	}
}

func TestReadModuleApplicationFromPackageJSON_Errors(t *testing.T) {
	if _, err := ReadModuleApplicationFromPackageJSON("/tmp", ""); err == nil {
		t.Fatal("expected error for empty module name")
	}
	modulesPath := filepath.Join(t.TempDir(), "modules")
	if app, err := ReadModuleApplicationFromPackageJSON(modulesPath, "missing"); err != nil || app != "missing" {
		t.Fatalf("missing package.json fallback = %q err=%v", app, err)
	}
}

func TestParseServiceSourceFile_Validation(t *testing.T) {
	if _, err := ParseServiceSourceFile(nil, "", []byte("export {};")); err == nil {
		t.Fatal("expected error for empty file path")
	}
}

func TestScanServiceImportBoundaryOnDisk_EdgeCases(t *testing.T) {
	if violations, err := ScanServiceImportBoundaryOnDisk(ServiceImportBoundaryScanInput{}); err != nil || violations != nil {
		t.Fatalf("empty input = violations %#v err=%v", violations, err)
	}
	modulesPath := filepath.Join(t.TempDir(), "modules")
	writeModulePackageJSON(t, modulesPath, "solo", "solo")
	moduleRoot := filepath.Join(modulesPath, "solo")
	if violations, err := ScanServiceImportBoundaryOnDisk(ServiceImportBoundaryScanInput{
		ModulesPath:       modulesPath,
		ModuleRoot:        moduleRoot,
		SourceApplication: "solo",
		PathAlias:         ModulePathAliasForBoundary(modulesPath),
	}); err != nil || len(violations) != 0 {
		t.Fatalf("no service dir = %#v err=%v", violations, err)
	}
}

func TestScanServiceImportBoundaryOnDisk_SkipsIgnoredPaths(t *testing.T) {
	modulesPath := filepath.Join(t.TempDir(), "modules")
	writeModulePackageJSON(t, modulesPath, "partner", "partner")
	moduleRoot := filepath.Join(modulesPath, "partner")
	serviceRoot := filepath.Join(moduleRoot, "service")
	for _, rel := range []string{
		"node_modules/pkg/index.ts",
		"dist/out.ts",
		"models/types.d.ts",
		"models/partner.ts",
	} {
		path := filepath.Join(serviceRoot, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		source := "import Role from '@/auth/service/models/role';\n"
		if strings.HasSuffix(rel, "partner.ts") {
			source = "export default {};\n"
		}
		if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
	}
	violations, err := ScanServiceImportBoundaryOnDisk(ServiceImportBoundaryScanInput{
		ModulesPath:       modulesPath,
		ModuleRoot:        moduleRoot,
		SourceApplication: "partner",
		PathAlias:         ModulePathAliasForBoundary(modulesPath),
	})
	if err != nil {
		t.Fatalf("ScanServiceImportBoundaryOnDisk() error = %v", err)
	}
	if len(violations) != 0 {
		t.Fatalf("expected skipped dirs/files to avoid violations, got %#v", violations)
	}
}

func TestCheckServiceImportBoundaryOnDisk_EmptyModuleName(t *testing.T) {
	if err := CheckServiceImportBoundaryOnDisk("/tmp/modules", "  ", nil); err != nil {
		t.Fatalf("empty module name should noop, got %v", err)
	}
}

func TestModulePathAliasForBoundary_Empty(t *testing.T) {
	if got := ModulePathAliasForBoundary(""); got != nil {
		t.Fatalf("empty modules path alias = %#v", got)
	}
}

func TestImportViolationKind_Dynamic(t *testing.T) {
	if got := importViolationKind(&parser.Import{IsDynamic: true}); got != "dynamic import" {
		t.Fatalf("kind = %q", got)
	}
	if got := importViolationKind(&parser.Import{}); got != "import" {
		t.Fatalf("kind = %q", got)
	}
}

func TestPathUnderModules_RelativePath(t *testing.T) {
	if pathUnderModules("/modules", "auth/service/models/role") {
		t.Fatal("relative spec path should not count")
	}
}

func TestAppendValueImportViolation_NilImport(t *testing.T) {
	got := appendValueImportViolation(nil, "", "", "import", nil, "/modules", testLookup())
	if got != nil {
		t.Fatalf("expected nil violations, got %#v", got)
	}
}

func TestReadModuleApplicationFromPackageJSON_InvalidJSON(t *testing.T) {
	modulesPath := filepath.Join(t.TempDir(), "modules")
	dir := filepath.Join(modulesPath, "broken")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte("{"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if _, err := ReadModuleApplicationFromPackageJSON(modulesPath, "broken"); err == nil {
		t.Fatal("expected decode error")
	}
}

func TestScanServiceImportBoundaryOnDisk_StatError(t *testing.T) {
	modulesPath := filepath.Join(t.TempDir(), "modules")
	writeModulePackageJSON(t, modulesPath, "solo", "solo")
	moduleRoot := filepath.Join(modulesPath, "solo")
	if err := os.MkdirAll(filepath.Join(moduleRoot, "service"), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	origStat := statPath
	statPath = func(string) (os.FileInfo, error) {
		return nil, fmt.Errorf("permission denied")
	}
	t.Cleanup(func() { statPath = origStat })

	_, err := ScanServiceImportBoundaryOnDisk(ServiceImportBoundaryScanInput{
		ModulesPath:       modulesPath,
		ModuleRoot:        moduleRoot,
		SourceApplication: "solo",
		PathAlias:         ModulePathAliasForBoundary(modulesPath),
	})
	if err == nil || !strings.Contains(err.Error(), "stat service dir") {
		t.Fatalf("expected stat service dir error, got %v", err)
	}
}

func TestScanServiceImportBoundaryOnDisk_WalkEntryError(t *testing.T) {
	modulesPath := filepath.Join(t.TempDir(), "modules")
	writeModulePackageJSON(t, modulesPath, "solo", "solo")
	moduleRoot := filepath.Join(modulesPath, "solo")
	if err := os.MkdirAll(filepath.Join(moduleRoot, "service"), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	origWalk := walkServiceTree
	walkServiceTree = func(_ string, walkFn fs.WalkDirFunc) error {
		return walkFn("", nil, fmt.Errorf("walk denied"))
	}
	t.Cleanup(func() { walkServiceTree = origWalk })

	_, err := ScanServiceImportBoundaryOnDisk(ServiceImportBoundaryScanInput{
		ModulesPath:       modulesPath,
		ModuleRoot:        moduleRoot,
		SourceApplication: "solo",
	})
	if err == nil || !strings.Contains(err.Error(), "walk denied") {
		t.Fatalf("expected walk entry error, got %v", err)
	}
}

func TestShouldSkipModulesDirEntry(t *testing.T) {
	for _, name := range []string{".choysum", ".git", ".vscode", "tmp", "node_modules"} {
		if !shouldSkipModulesDirEntry(name) {
			t.Fatalf("%q should be skipped", name)
		}
	}
	if shouldSkipModulesDirEntry("auth") {
		t.Fatal("auth should not be skipped")
	}
}

func TestCheckServiceImportBoundary_SortsViolationsDeterministically(t *testing.T) {
	modulesPath := testModulesPath(t)
	moduleRoot := filepath.Join(modulesPath, "partner")
	sourceA := filepath.Join(moduleRoot, "service", "models", "a.ts")
	sourceB := filepath.Join(moduleRoot, "service", "models", "b.ts")
	authSpec := filepath.Join(modulesPath, "auth", "service", "models", "role")
	metaSpec := filepath.Join(modulesPath, "meta", "service", "models", "ui_resource")

	violations := CheckServiceImportBoundary(ServiceImportBoundaryInput{
		ModulesPath:       modulesPath,
		ModuleRoot:        moduleRoot,
		SourceApplication: "partner",
		Lookup:            testLookup(),
		ParserResults: []*parser.ParserResult{
			{
				Path: sourceB,
				Imports: map[string]*parser.Import{
					"Role": {ModuleSpecPath: authSpec, ModuleSpecText: "@/auth/service/models/role", Line: 5, Column: 2},
					"UI":   {ModuleSpecPath: metaSpec, ModuleSpecText: "@/meta/service/models/ui_resource", Line: 5, Column: 1},
				},
			},
			{
				Path: sourceA,
				Imports: map[string]*parser.Import{
					"Role": {ModuleSpecPath: authSpec, ModuleSpecText: "@/auth/service/models/role", Line: 1, Column: 1},
				},
			},
		},
	})
	if len(violations) < 3 {
		t.Fatalf("expected at least 3 violations, got %#v", violations)
	}
	for i := 1; i < len(violations); i++ {
		prev, cur := violations[i-1], violations[i]
		if prev.SourcePath > cur.SourcePath {
			t.Fatalf("violations not sorted by path: %#v then %#v", prev, cur)
		}
		if prev.SourcePath == cur.SourcePath && prev.Line > cur.Line {
			t.Fatalf("violations not sorted by line: %#v then %#v", prev, cur)
		}
		if prev.SourcePath == cur.SourcePath && prev.Line == cur.Line && prev.Column > cur.Column {
			t.Fatalf("violations not sorted by column: %#v then %#v", prev, cur)
		}
	}
}

func TestReadExplicitModuleApplicationFromPackageJSON(t *testing.T) {
	if _, ok, err := ReadExplicitModuleApplicationFromPackageJSON("/tmp", ""); err == nil || ok {
		t.Fatal("expected error for empty module name")
	}
	modulesPath := filepath.Join(t.TempDir(), "modules")
	if _, ok, err := ReadExplicitModuleApplicationFromPackageJSON(modulesPath, "missing"); err != nil || ok {
		t.Fatalf("missing package.json = ok=%v err=%v", ok, err)
	}

	dir := filepath.Join(modulesPath, "enterprise_module")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	content := `{
  "name": "@test/enterprise_module",
  "version": "0.0.0-test",
  "choysum": { "moduleName": "enterprise_module" }
}`
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if _, ok, err := ReadExplicitModuleApplicationFromPackageJSON(modulesPath, "enterprise_module"); err != nil || ok {
		t.Fatalf("empty application = ok=%v err=%v", ok, err)
	}

	writeModulePackageJSON(t, modulesPath, "partner_bank", "partner")
	if app, ok, err := ReadExplicitModuleApplicationFromPackageJSON(modulesPath, "partner_bank"); err != nil || !ok || app != "partner" {
		t.Fatalf("explicit app = %q ok=%v err=%v", app, ok, err)
	}
}

func TestModuleNameFromModulesPath_RelativePathError(t *testing.T) {
	if got := ModuleNameFromModulesPath("modules", "modules"); got != "" {
		t.Fatalf("self rel = %q", got)
	}
}

func TestIsModuleServiceSource_EdgeCases(t *testing.T) {
	root := filepath.Join("/virtual/modules/auth")
	if IsModuleServiceSource("", filepath.Join(root, "service/a.ts")) {
		t.Fatal("empty module root should not match")
	}
	if IsModuleServiceSource(root, "") {
		t.Fatal("empty path should not match")
	}
	if IsModuleServiceSource(root, root) {
		t.Fatal("module root itself should not match")
	}
	if !IsModuleServiceSource(root, filepath.Join(root, "service")) {
		t.Fatal("service dir should match")
	}
}

func TestResolveModuleApplication_LookupEmptyApp(t *testing.T) {
	lookup := ModuleApplicationLookupFromMap(map[string]string{"auth": "  "})
	if app, ok := ResolveModuleApplication("auth", lookup); !ok || app != "auth" {
		t.Fatalf("fallback auth = %q ok=%v", app, ok)
	}
}

func TestCheckServiceImportBoundary_EmptySourceOrRoot(t *testing.T) {
	modulesPath := testModulesPath(t)
	if got := CheckServiceImportBoundary(ServiceImportBoundaryInput{
		ModulesPath: modulesPath,
		ModuleRoot:  filepath.Join(modulesPath, "partner"),
	}); got != nil {
		t.Fatalf("empty source app = %#v", got)
	}
	if got := CheckServiceImportBoundary(ServiceImportBoundaryInput{
		ModulesPath:       modulesPath,
		SourceApplication: "partner",
	}); got != nil {
		t.Fatalf("empty module root = %#v", got)
	}
}

func TestCheckServiceImportBoundary_SkipsNilResultAndUnresolvedTarget(t *testing.T) {
	modulesPath := testModulesPath(t)
	moduleRoot := filepath.Join(modulesPath, "partner")
	source := filepath.Join(moduleRoot, "service", "models", "partner.ts")
	outsideSpec := filepath.Join(t.TempDir(), "outside", "service", "x")
	violations := CheckServiceImportBoundary(ServiceImportBoundaryInput{
		ModulesPath:       modulesPath,
		ModuleRoot:        moduleRoot,
		SourceApplication: "partner",
		Lookup:            testLookup(),
		ParserResults: []*parser.ParserResult{
			nil,
			{
				Path: source,
				Imports: map[string]*parser.Import{
					"X": {
						ModuleSpecPath: outsideSpec,
						IsTypeOnly:     false,
						Line:           1,
						Column:         1,
					},
				},
			},
		},
	})
	if len(violations) != 0 {
		t.Fatalf("expected no violations, got %#v", violations)
	}
}

func TestAppendValueImportViolation_UsesImportTextFallback(t *testing.T) {
	modulesPath := testModulesPath(t)
	moduleRoot := filepath.Join(modulesPath, "partner")
	source := filepath.Join(moduleRoot, "service", "models", "partner.ts")
	authSpec := filepath.Join(modulesPath, "auth", "service", "models", "role")
	violations := CheckServiceImportBoundary(ServiceImportBoundaryInput{
		ModulesPath:       modulesPath,
		ModuleRoot:        moduleRoot,
		SourceApplication: "partner",
		Lookup:            testLookup(),
		ParserResults: []*parser.ParserResult{{
			Path: source,
			Imports: map[string]*parser.Import{
				"Role": {
					ModuleSpecPath: authSpec,
					Text:           "import Role from '@/auth/service/models/role'",
					IsTypeOnly:     false,
					Line:           2,
					Column:         1,
				},
			},
		}},
	})
	if len(violations) != 1 || violations[0].ModuleSpecText == "" {
		t.Fatalf("expected text fallback violation, got %#v", violations)
	}
}

func TestCollectModuleNamesFromParserResult_NilEntries(t *testing.T) {
	names := make(map[string]struct{})
	CollectModuleNamesFromParserResult(testModulesPath(t), &parser.ParserResult{
		Imports:        map[string]*parser.Import{"x": nil},
		DynamicImports: []*parser.Import{nil},
		Exports:        map[string]*parser.Export{"y": {Wildcard: []*parser.Export{nil}}},
	}, names)
	if len(names) != 0 {
		t.Fatalf("expected no names, got %#v", names)
	}
}

func TestBuildModuleApplicationLookupFromModulesDir_ReadDirError(t *testing.T) {
	if _, err := BuildModuleApplicationLookupFromModulesDir(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("expected read dir error")
	}
}

func TestParseServiceSourceFile_EmptyContent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "empty.ts")
	if _, err := ParseServiceSourceFile(nil, path, []byte("")); err == nil {
		t.Fatal("expected empty content error")
	}
	good := filepath.Join(t.TempDir(), "good.ts")
	if _, err := ParseServiceSourceFile(nil, good, []byte("export type { X } from './x';")); err != nil {
		t.Fatalf("ParseServiceSourceFile() error = %v", err)
	}
}

func TestScanServiceImportBoundaryOnDisk_ServiceNotDir(t *testing.T) {
	modulesPath := filepath.Join(t.TempDir(), "modules")
	writeModulePackageJSON(t, modulesPath, "solo", "solo")
	moduleRoot := filepath.Join(modulesPath, "solo")
	serviceFile := filepath.Join(moduleRoot, "service")
	if err := os.WriteFile(serviceFile, []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	violations, err := ScanServiceImportBoundaryOnDisk(ServiceImportBoundaryScanInput{
		ModulesPath:       modulesPath,
		ModuleRoot:        moduleRoot,
		SourceApplication: "solo",
		PathAlias:         ModulePathAliasForBoundary(modulesPath),
	})
	if err != nil || len(violations) != 0 {
		t.Fatalf("service file (not dir) = violations %#v err=%v", violations, err)
	}
}

func TestScanServiceImportBoundaryOnDisk_ParseError(t *testing.T) {
	modulesPath := filepath.Join(t.TempDir(), "modules")
	writeModulePackageJSON(t, modulesPath, "solo", "solo")
	moduleRoot := filepath.Join(modulesPath, "solo")
	serviceRoot := filepath.Join(moduleRoot, "service")
	if err := os.MkdirAll(serviceRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(serviceRoot, "broken.ts"), []byte(""), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if _, err := ScanServiceImportBoundaryOnDisk(ServiceImportBoundaryScanInput{
		ModulesPath:       modulesPath,
		ModuleRoot:        moduleRoot,
		SourceApplication: "solo",
		PathAlias:         ModulePathAliasForBoundary(modulesPath),
	}); err == nil {
		t.Fatal("expected parse error")
	}
}

func TestCheckServiceImportBoundaryOnDisk_ReadPackageJSONError(t *testing.T) {
	modulesPath := filepath.Join(t.TempDir(), "modules")
	dir := filepath.Join(modulesPath, "broken")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte("{"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := CheckServiceImportBoundaryOnDisk(modulesPath, "broken", nil); err == nil {
		t.Fatal("expected package.json decode error")
	}
}

func TestDefaultApplicationForModuleName_Empty(t *testing.T) {
	if got := DefaultApplicationForModuleName(""); got != "" {
		t.Fatalf("empty module name = %q", got)
	}
}

func TestPathUnderModules(t *testing.T) {
	modulesPath := testModulesPath(t)
	authSpec := filepath.Join(modulesPath, "auth", "service", "models", "role")
	if !pathUnderModules(modulesPath, authSpec) {
		t.Fatal("expected abs path under modules")
	}
	if pathUnderModules(modulesPath, "auth/service/models/role") {
		t.Fatal("relative path should not match")
	}
	if pathUnderModules("", authSpec) {
		t.Fatal("empty modules path should not match")
	}
}

func TestIsAllowedTarget(t *testing.T) {
	if !isAllowedTarget("", "auth") {
		t.Fatal("empty source should allow")
	}
	if !isAllowedTarget("partner", "") {
		t.Fatal("empty target should allow")
	}
	if !isAllowedTarget("partner", "core") {
		t.Fatal("core target should allow")
	}
	if !isAllowedTarget("partner", "partner") {
		t.Fatal("same app should allow")
	}
	if isAllowedTarget("partner", "auth") {
		t.Fatal("cross-app should not allow")
	}
}

func TestResolveTarget(t *testing.T) {
	modulesPath := testModulesPath(t)
	authSpec := filepath.Join(modulesPath, "auth", "service", "models", "role")
	mod, app, ok := resolveTarget(modulesPath, authSpec, testLookup())
	if !ok || mod != "auth" || app != "auth" {
		t.Fatalf("resolveTarget() = %q %q ok=%v", mod, app, ok)
	}
	if _, _, ok := resolveTarget(modulesPath, "", testLookup()); ok {
		t.Fatal("empty spec should miss")
	}
}

func TestReadModuleApplicationFromPackageJSON_EmptyModulesPath(t *testing.T) {
	if app, err := ReadModuleApplicationFromPackageJSON("", "partner_bank"); err != nil || app != "partner" {
		t.Fatalf("empty modules path fallback = %q err=%v", app, err)
	}
	if _, ok, err := ReadExplicitModuleApplicationFromPackageJSON("", "auth"); err != nil || ok {
		t.Fatalf("empty modules path explicit = ok=%v err=%v", ok, err)
	}
}

func TestIsServiceTypeScriptSource(t *testing.T) {
	if isServiceTypeScriptSource("model.js") {
		t.Fatal(".js should be skipped")
	}
	if !isServiceTypeScriptSource("model.ts") {
		t.Fatal(".ts should match")
	}
	if isServiceTypeScriptSource("model.d.ts") {
		t.Fatal(".d.ts should be skipped")
	}
}

func TestCheckServiceImportBoundary_AllowsExportTypeWildcard(t *testing.T) {
	modulesPath := testModulesPath(t)
	moduleRoot := filepath.Join(modulesPath, "partner")
	source := filepath.Join(moduleRoot, "service", "models", "partner.ts")
	authSpec := filepath.Join(modulesPath, "auth", "service", "models", "role")

	violations := CheckServiceImportBoundary(ServiceImportBoundaryInput{
		ModulesPath:       modulesPath,
		ModuleRoot:        moduleRoot,
		SourceApplication: "partner",
		Lookup:            testLookup(),
		ParserResults: []*parser.ParserResult{{
			Path: source,
			Exports: map[string]*parser.Export{
				"*": {
					IsTypeOnly: true,
					Wildcard: []*parser.Export{{
						ModuleSpecPath: authSpec,
						IsTypeOnly:     true,
						Line:           1,
						Column:         1,
					}},
				},
			},
		}},
	})
	if len(violations) != 0 {
		t.Fatalf("expected no export type wildcard violations, got %#v", violations)
	}
}

func TestCheckServiceImportBoundary_SkipsNilImportAndEmptyExportSpec(t *testing.T) {
	modulesPath := testModulesPath(t)
	moduleRoot := filepath.Join(modulesPath, "partner")
	source := filepath.Join(moduleRoot, "service", "models", "partner.ts")
	violations := CheckServiceImportBoundary(ServiceImportBoundaryInput{
		ModulesPath:       modulesPath,
		ModuleRoot:        moduleRoot,
		SourceApplication: "partner",
		Lookup:            testLookup(),
		ParserResults: []*parser.ParserResult{{
			Path: source,
			Imports: map[string]*parser.Import{
				"nil": nil,
			},
			Exports: map[string]*parser.Export{
				"empty": {
					ModuleSpecPath: "",
					IsTypeOnly:     false,
					Line:           1,
					Column:         1,
				},
			},
		}},
	})
	if len(violations) != 0 {
		t.Fatalf("expected no violations, got %#v", violations)
	}
}

func TestBuildModuleApplicationLookupFromModulesDir_SkipsEmptyApplication(t *testing.T) {
	modulesPath := filepath.Join(t.TempDir(), "modules")
	dir := filepath.Join(modulesPath, "empty_app")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	content := `{
  "name": "@test/empty_app",
  "version": "0.0.0-test",
  "choysum": { "moduleName": "empty_app" }
}`
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	writeModulePackageJSON(t, modulesPath, "auth", "auth")
	lookup, err := BuildModuleApplicationLookupFromModulesDir(modulesPath)
	if err != nil {
		t.Fatalf("BuildModuleApplicationLookupFromModulesDir() error = %v", err)
	}
	if app, ok := lookup("empty_app"); !ok || app != "empty_app" {
		t.Fatalf("empty_app fallback = %q ok=%v", app, ok)
	}
}

func TestCheckServiceImportBoundaryOnDisk_AllowsExportTypeWildcard(t *testing.T) {
	modulesPath := filepath.Join(t.TempDir(), "modules")
	writeModulePackageJSON(t, modulesPath, "partner", "partner")
	writeModulePackageJSON(t, modulesPath, "auth", "auth")
	serviceFile := filepath.Join(modulesPath, "partner", "service", "models", "partner.ts")
	if err := os.MkdirAll(filepath.Dir(serviceFile), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	source := "export type * from '@/auth/service/models/role';\n"
	if err := os.WriteFile(serviceFile, []byte(source), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := CheckServiceImportBoundaryOnDisk(modulesPath, "partner", ModulePathAliasForBoundary(modulesPath)); err != nil {
		t.Fatalf("CheckServiceImportBoundaryOnDisk() error = %v", err)
	}
}

func TestResolveTarget_EmptyModuleNameFromModulesRoot(t *testing.T) {
	modulesPath := testModulesPath(t)
	if _, _, ok := resolveTarget(modulesPath, modulesPath, testLookup()); ok {
		t.Fatal("modules root should not resolve")
	}
}

func TestModuleNameFromModulesPath_EmptyFirstSegment(t *testing.T) {
	modulesPath := filepath.Join(t.TempDir(), "modules")
	if got := ModuleNameFromModulesPath(modulesPath, filepath.Join(modulesPath, ".hidden", "file.ts")); got != ".hidden" {
		t.Fatalf("first segment = %q", got)
	}
}

func TestResolveModuleApplication_LookupReturnsEmptyApp(t *testing.T) {
	lookup := ModuleApplicationLookup(func(string) (string, bool) { return "", true })
	if app, ok := ResolveModuleApplication("auth", lookup); !ok || app != "auth" {
		t.Fatalf("fallback auth = %q ok=%v", app, ok)
	}
}

func TestReadExplicitModuleApplicationFromPackageJSON_ReadError(t *testing.T) {
	modulesPath := filepath.Join(t.TempDir(), "modules")
	dir := filepath.Join(modulesPath, "locked")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"choysum":{"application":"auth"}}`), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := os.Chmod(filepath.Join(dir, "package.json"), 0o000); err != nil {
		t.Skipf("chmod not permitted: %v", err)
	}
	defer os.Chmod(filepath.Join(dir, "package.json"), 0o644)
	if _, ok, err := ReadExplicitModuleApplicationFromPackageJSON(modulesPath, "locked"); err == nil || ok {
		t.Fatalf("expected read error, ok=%v err=%v", ok, err)
	}
}

func TestCheckServiceImportBoundaryOnDisk_DetectsMultipleSideEffectImports(t *testing.T) {
	modulesPath := filepath.Join(t.TempDir(), "modules")
	writeModulePackageJSON(t, modulesPath, "partner", "partner")
	writeModulePackageJSON(t, modulesPath, "auth", "auth")
	writeModulePackageJSON(t, modulesPath, "core", "core")
	serviceFile := filepath.Join(modulesPath, "partner", "service", "models", "partner.ts")
	if err := os.MkdirAll(filepath.Dir(serviceFile), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	source := "import '@/core/service/api/dial';\nimport '@/auth/service/models/role';\n"
	if err := os.WriteFile(serviceFile, []byte(source), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	err := CheckServiceImportBoundaryOnDisk(modulesPath, "partner", ModulePathAliasForBoundary(modulesPath))
	if err == nil || !strings.Contains(err.Error(), "partner -> auth") {
		t.Fatalf("CheckServiceImportBoundaryOnDisk() error = %v, want auth violation", err)
	}
}

func TestResolveModuleApplication_EmptyModuleName(t *testing.T) {
	if _, ok := ResolveModuleApplication("", nil); ok {
		t.Fatal("empty module name should not resolve")
	}
}

func TestBuildModuleApplicationLookupFromModulesDir_InvalidPackageJSON(t *testing.T) {
	modulesPath := filepath.Join(t.TempDir(), "modules")
	dir := filepath.Join(modulesPath, "broken")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte("{"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if _, err := BuildModuleApplicationLookupFromModulesDir(modulesPath); err == nil {
		t.Fatal("expected decode error")
	}
}

func TestCollectModuleNamesFromParserResult_SideEffectKeys(t *testing.T) {
	modulesPath := testModulesPath(t)
	names := make(map[string]struct{})
	CollectModuleNamesFromParserResult(modulesPath, &parser.ParserResult{
		Imports: map[string]*parser.Import{
			parser.SideEffectImportMapKey(1, 1): {
				ModuleSpecPath: filepath.Join(modulesPath, "auth", "service", "models", "role"),
			},
			parser.SideEffectImportMapKey(2, 1): {
				ModuleSpecPath: filepath.Join(modulesPath, "core", "service", "api", "dial"),
			},
		},
	}, names)
	for _, want := range []string{"auth", "core"} {
		if _, ok := names[want]; !ok {
			t.Fatalf("missing module %q in %#v", want, names)
		}
	}
}

func TestCollectModuleNamesFromParserResult_DynamicImports(t *testing.T) {
	modulesPath := testModulesPath(t)
	names := make(map[string]struct{})
	CollectModuleNamesFromParserResult(modulesPath, &parser.ParserResult{
		DynamicImports: []*parser.Import{{
			ModuleSpecPath: filepath.Join(modulesPath, "meta", "service", "models", "ui_resource"),
		}},
	}, names)
	if _, ok := names["meta"]; !ok {
		t.Fatalf("expected meta from dynamic import, got %#v", names)
	}
}

func TestCheckServiceImportBoundary_DynamicImportViolation(t *testing.T) {
	modulesPath := testModulesPath(t)
	moduleRoot := filepath.Join(modulesPath, "partner")
	source := filepath.Join(moduleRoot, "service", "models", "partner.ts")
	authSpec := filepath.Join(modulesPath, "auth", "service", "models", "role")
	violations := CheckServiceImportBoundary(ServiceImportBoundaryInput{
		ModulesPath:       modulesPath,
		ModuleRoot:        moduleRoot,
		SourceApplication: "partner",
		Lookup:            testLookup(),
		ParserResults: []*parser.ParserResult{{
			Path: source,
			DynamicImports: []*parser.Import{{
				ModuleSpecPath: authSpec,
				ModuleSpecText: "@/auth/service/models/role",
				IsDynamic:      true,
				IsTypeOnly:     false,
				Line:           3,
				Column:         7,
			}},
		}},
	})
	if len(violations) != 1 || violations[0].Kind != "dynamic import" {
		t.Fatalf("expected dynamic import violation, got %#v", violations)
	}
}

func TestCheckServiceImportBoundary_SkipsNonServicePath(t *testing.T) {
	modulesPath := testModulesPath(t)
	moduleRoot := filepath.Join(modulesPath, "partner")
	webPath := filepath.Join(moduleRoot, "web", "views", "list.ts")
	authSpec := filepath.Join(modulesPath, "auth", "service", "models", "role")
	violations := CheckServiceImportBoundary(ServiceImportBoundaryInput{
		ModulesPath:       modulesPath,
		ModuleRoot:        moduleRoot,
		SourceApplication: "partner",
		Lookup:            testLookup(),
		ParserResults: []*parser.ParserResult{{
			Path: webPath,
			Imports: map[string]*parser.Import{
				"Role": {ModuleSpecPath: authSpec, IsTypeOnly: false, Line: 1, Column: 1},
			},
		}},
	})
	if len(violations) != 0 {
		t.Fatalf("expected no violations outside service tree, got %#v", violations)
	}
}

func TestCheckServiceImportBoundary_ExportValueViolation(t *testing.T) {
	modulesPath := testModulesPath(t)
	moduleRoot := filepath.Join(modulesPath, "partner")
	source := filepath.Join(moduleRoot, "service", "models", "partner.ts")
	authSpec := filepath.Join(modulesPath, "auth", "service", "models", "role")
	violations := CheckServiceImportBoundary(ServiceImportBoundaryInput{
		ModulesPath:       modulesPath,
		ModuleRoot:        moduleRoot,
		SourceApplication: "partner",
		Lookup:            testLookup(),
		ParserResults: []*parser.ParserResult{{
			Path: source,
			Exports: map[string]*parser.Export{
				"Role": {
					ModuleSpecPath: authSpec,
					IsTypeOnly:     false,
					Line:           1,
					Column:         1,
				},
			},
		}},
	})
	if len(violations) != 1 || violations[0].Kind != "export" {
		t.Fatalf("expected export violation, got %#v", violations)
	}
}

func TestPathUnderModules_EmptySpec(t *testing.T) {
	modulesPath := testModulesPath(t)
	if pathUnderModules(modulesPath, "") {
		t.Fatal("empty spec should not match")
	}
}

func TestResolveTarget_LookupMissUsesDefault(t *testing.T) {
	modulesPath := testModulesPath(t)
	spec := filepath.Join(modulesPath, "auth", "service", "models", "role")
	mod, app, ok := resolveTarget(modulesPath, spec, nil)
	if !ok || mod != "auth" || app != "auth" {
		t.Fatalf("resolveTarget() = %q %q ok=%v", mod, app, ok)
	}
}

func TestBuildModuleApplicationLookupFromModulesDir_SkipsFiles(t *testing.T) {
	modulesPath := filepath.Join(t.TempDir(), "modules")
	if err := os.MkdirAll(modulesPath, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(modulesPath, "readme.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	writeModulePackageJSON(t, modulesPath, "auth", "auth")
	lookup, err := BuildModuleApplicationLookupFromModulesDir(modulesPath)
	if err != nil {
		t.Fatalf("BuildModuleApplicationLookupFromModulesDir() error = %v", err)
	}
	if app, ok := lookup("auth"); !ok || app != "auth" {
		t.Fatalf("auth lookup = %q ok=%v", app, ok)
	}
}

func TestReadExplicitModuleApplicationFromPackageJSON_EmptyModuleName(t *testing.T) {
	if _, ok, err := ReadExplicitModuleApplicationFromPackageJSON("/modules", ""); err == nil || ok {
		t.Fatalf("empty module name = ok=%v err=%v", ok, err)
	}
}

func TestReadModuleApplicationFromPackageJSON_EmptyModuleName(t *testing.T) {
	if _, err := ReadModuleApplicationFromPackageJSON("/modules", ""); err == nil {
		t.Fatal("expected empty module name error")
	}
}

func TestParseServiceSourceFile_EmptyPath(t *testing.T) {
	if _, err := ParseServiceSourceFile(nil, "", []byte("export {};")); err == nil {
		t.Fatal("expected empty path error")
	}
}

func TestScanServiceImportBoundaryOnDisk_SkipsScanDirs(t *testing.T) {
	modulesPath := filepath.Join(t.TempDir(), "modules")
	writeModulePackageJSON(t, modulesPath, "solo", "solo")
	moduleRoot := filepath.Join(modulesPath, "solo")
	serviceRoot := filepath.Join(moduleRoot, "service", "node_modules")
	if err := os.MkdirAll(serviceRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	violations, err := ScanServiceImportBoundaryOnDisk(ServiceImportBoundaryScanInput{
		ModulesPath:       modulesPath,
		ModuleRoot:        moduleRoot,
		SourceApplication: "solo",
		PathAlias:         ModulePathAliasForBoundary(modulesPath),
	})
	if err != nil || len(violations) != 0 {
		t.Fatalf("skipped scan dir = violations %#v err=%v", violations, err)
	}
}

func TestCheckServiceImportBoundaryOnDisk_EmptyInputs(t *testing.T) {
	if err := CheckServiceImportBoundaryOnDisk("", "auth", nil); err != nil {
		t.Fatalf("empty modules path = %v", err)
	}
	if err := CheckServiceImportBoundaryOnDisk("/modules", "", nil); err != nil {
		t.Fatalf("empty module name = %v", err)
	}
}

func TestModuleNameFromModulesPath_RelativeBaseError(t *testing.T) {
	if got := ModuleNameFromModulesPath("modules", filepath.Join(testModulesPath(t), "auth", "service", "x.ts")); got != "" {
		t.Fatalf("relative modules base = %q, want empty", got)
	}
}

func TestIsModuleServiceSource_RelativeRootError(t *testing.T) {
	root := filepath.Join(testModulesPath(t), "auth")
	if IsModuleServiceSource("auth", filepath.Join(root, "service", "x.ts")) {
		t.Fatal("relative module root should not match")
	}
}

func TestPathUnderModules_RelativeBaseError(t *testing.T) {
	spec := filepath.Join(testModulesPath(t), "auth", "service", "x.ts")
	if pathUnderModules("modules", spec) {
		t.Fatal("relative modules path should not match")
	}
}

func TestResolveModuleApplication_LookupMissWithEmptyDefault(t *testing.T) {
	if app, ok := ResolveModuleApplication("   ", nil); ok || app != "" {
		t.Fatalf("whitespace module = %q ok=%v", app, ok)
	}
}

func TestCheckServiceImportBoundary_SkipsNilDynamicImportAndExport(t *testing.T) {
	modulesPath := testModulesPath(t)
	moduleRoot := filepath.Join(modulesPath, "partner")
	source := filepath.Join(moduleRoot, "service", "models", "partner.ts")
	violations := CheckServiceImportBoundary(ServiceImportBoundaryInput{
		ModulesPath:       modulesPath,
		ModuleRoot:        moduleRoot,
		SourceApplication: "partner",
		Lookup:            testLookup(),
		ParserResults: []*parser.ParserResult{{
			Path:           source,
			DynamicImports: []*parser.Import{nil},
			Exports:        map[string]*parser.Export{"nil": nil},
		}},
	})
	if len(violations) != 0 {
		t.Fatalf("expected no violations, got %#v", violations)
	}
}

func TestCollectModuleNamesFromParserResult_NilExportEntry(t *testing.T) {
	names := make(map[string]struct{})
	CollectModuleNamesFromParserResult(testModulesPath(t), &parser.ParserResult{
		Exports: map[string]*parser.Export{"nil": nil},
	}, names)
	if len(names) != 0 {
		t.Fatalf("expected no names, got %#v", names)
	}
}

func TestBuildModuleApplicationLookupFromModulesDir_SkipsNodeModules(t *testing.T) {
	modulesPath := filepath.Join(t.TempDir(), "modules")
	broken := filepath.Join(modulesPath, "node_modules", "broken")
	if err := os.MkdirAll(broken, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(broken, "package.json"), []byte("{"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	writeModulePackageJSON(t, modulesPath, "auth", "auth")
	lookup, err := BuildModuleApplicationLookupFromModulesDir(modulesPath)
	if err != nil {
		t.Fatalf("node_modules should be skipped, err=%v", err)
	}
	if app, ok := lookup("auth"); !ok || app != "auth" {
		t.Fatalf("auth lookup = %q ok=%v", app, ok)
	}
}

func TestReadModuleApplicationFromPackageJSON_ReadError(t *testing.T) {
	modulesPath := filepath.Join(t.TempDir(), "modules")
	dir := filepath.Join(modulesPath, "locked")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"choysum":{"application":"auth"}}`), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := os.Chmod(filepath.Join(dir, "package.json"), 0o000); err != nil {
		t.Skipf("chmod not permitted: %v", err)
	}
	defer os.Chmod(filepath.Join(dir, "package.json"), 0o644)
	if _, err := ReadModuleApplicationFromPackageJSON(modulesPath, "locked"); err == nil {
		t.Fatal("expected read error")
	}
}

func TestScanServiceImportBoundaryOnDisk_LookupBuildError(t *testing.T) {
	modulesPath := filepath.Join(t.TempDir(), "modules")
	writeModulePackageJSON(t, modulesPath, "solo", "solo")
	dir := filepath.Join(modulesPath, "broken")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte("{"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	moduleRoot := filepath.Join(modulesPath, "solo")
	serviceRoot := filepath.Join(moduleRoot, "service")
	if err := os.MkdirAll(serviceRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(serviceRoot, "ok.ts"), []byte("export {};"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if _, err := ScanServiceImportBoundaryOnDisk(ServiceImportBoundaryScanInput{
		ModulesPath:       modulesPath,
		ModuleRoot:        moduleRoot,
		SourceApplication: "solo",
		PathAlias:         ModulePathAliasForBoundary(modulesPath),
	}); err == nil {
		t.Fatal("expected lookup build error")
	}
}

func TestScanServiceImportBoundaryOnDisk_ReadSourceError(t *testing.T) {
	modulesPath := filepath.Join(t.TempDir(), "modules")
	writeModulePackageJSON(t, modulesPath, "solo", "solo")
	moduleRoot := filepath.Join(modulesPath, "solo")
	serviceRoot := filepath.Join(moduleRoot, "service")
	if err := os.MkdirAll(serviceRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	sourceFile := filepath.Join(serviceRoot, "blocked.ts")
	if err := os.WriteFile(sourceFile, []byte("export {};"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := os.Chmod(sourceFile, 0o000); err != nil {
		t.Skipf("chmod not permitted: %v", err)
	}
	defer os.Chmod(sourceFile, 0o644)
	if _, err := ScanServiceImportBoundaryOnDisk(ServiceImportBoundaryScanInput{
		ModulesPath:       modulesPath,
		ModuleRoot:        moduleRoot,
		SourceApplication: "solo",
		PathAlias:         ModulePathAliasForBoundary(modulesPath),
	}); err == nil {
		t.Fatal("expected read source error")
	}
}

func TestScanServiceImportBoundaryOnDisk_MissingServiceDir(t *testing.T) {
	modulesPath := filepath.Join(t.TempDir(), "modules")
	writeModulePackageJSON(t, modulesPath, "solo", "solo")
	moduleRoot := filepath.Join(modulesPath, "solo")
	violations, err := ScanServiceImportBoundaryOnDisk(ServiceImportBoundaryScanInput{
		ModulesPath:       modulesPath,
		ModuleRoot:        moduleRoot,
		SourceApplication: "solo",
	})
	if err != nil || len(violations) != 0 {
		t.Fatalf("missing service dir = violations %#v err=%v", violations, err)
	}
}

func TestReadExplicitModuleApplicationFromPackageJSON_InvalidJSON(t *testing.T) {
	modulesPath := filepath.Join(t.TempDir(), "modules")
	dir := filepath.Join(modulesPath, "broken")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte("{"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if _, ok, err := ReadExplicitModuleApplicationFromPackageJSON(modulesPath, "broken"); err == nil || ok {
		t.Fatalf("expected decode error, ok=%v err=%v", ok, err)
	}
}

func TestCheckServiceImportBoundaryOnDisk_PropagatesScanError(t *testing.T) {
	modulesPath := filepath.Join(t.TempDir(), "modules")
	writeModulePackageJSON(t, modulesPath, "solo", "solo")
	moduleRoot := filepath.Join(modulesPath, "solo")
	serviceRoot := filepath.Join(moduleRoot, "service")
	if err := os.MkdirAll(serviceRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	sourceFile := filepath.Join(serviceRoot, "blocked.ts")
	if err := os.WriteFile(sourceFile, []byte("export {};"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := os.Chmod(sourceFile, 0o000); err != nil {
		t.Skipf("chmod not permitted: %v", err)
	}
	defer os.Chmod(sourceFile, 0o644)
	if err := CheckServiceImportBoundaryOnDisk(modulesPath, "solo", ModulePathAliasForBoundary(modulesPath)); err == nil {
		t.Fatal("expected scan read error")
	}
}

func TestParseServiceSourceFile_IncludesExportsFromSingleParse(t *testing.T) {
	path := filepath.Join(t.TempDir(), "model.ts")
	content := "export type { Role } from '@/auth/service/models/role';\n"
	result, err := ParseServiceSourceFile(ModulePathAliasForBoundary(testModulesPath(t)), path, []byte(content))
	if err != nil {
		t.Fatalf("ParseServiceSourceFile() error = %v", err)
	}
	if len(result.Exports) == 0 {
		t.Fatal("expected exports from ParseImport")
	}
}
