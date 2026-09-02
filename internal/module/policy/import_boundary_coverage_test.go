// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package policy

import (
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
				parser.SideEffectImportKey: {
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
	serviceRoot := filepath.Join(moduleRoot, "service")
	if err := os.MkdirAll(serviceRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(serviceRoot, "blocked.ts"), []byte("export {};"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := os.Chmod(serviceRoot, 0o000); err != nil {
		t.Skipf("chmod not permitted: %v", err)
	}
	defer os.Chmod(serviceRoot, 0o755)

	_, err := ScanServiceImportBoundaryOnDisk(ServiceImportBoundaryScanInput{
		ModulesPath:       modulesPath,
		ModuleRoot:        moduleRoot,
		SourceApplication: "solo",
		PathAlias:         ModulePathAliasForBoundary(modulesPath),
	})
	if err == nil {
		t.Fatal("expected walk/stat error")
	}
}

func TestShouldSkipModulesDirEntry(t *testing.T) {
	for _, name := range []string{".choysum", "tmp", "node_modules"} {
		if !shouldSkipModulesDirEntry(name) {
			t.Fatalf("%q should be skipped", name)
		}
	}
}
