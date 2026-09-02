// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package backendplugin

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/internal/esbplugins"
	"github.com/choysum-dev/choysum/internal/module/policy"
	"github.com/choysum-dev/choysum/internal/parser"
	"github.com/choysum-dev/choysum/pkg/meta"
)

func TestResolveSourceApplication_PrefersPackageJSONAndApplicationStr(t *testing.T) {
	modulesPath := filepath.Join(t.TempDir(), "modules")
	writeBoundaryPackageJSON(t, modulesPath, "partner_bank", "partner")

	t.Run("package.json wins over module name heuristic", func(t *testing.T) {
		got := resolveSourceApplication(&meta.Module{
			Name:           "partner_bank",
			ApplicationStr: "partner_bank",
		}, modulesPath, nil)
		if got != "partner" {
			t.Fatalf("application = %q, want partner", got)
		}
	})

	t.Run("ApplicationStr wins when package.json is absent", func(t *testing.T) {
		got := resolveSourceApplication(&meta.Module{
			Name:           "enterprise_module",
			ApplicationStr: "enterprise",
		}, modulesPath, nil)
		if got != "enterprise" {
			t.Fatalf("application = %q, want enterprise", got)
		}
	})

	t.Run("ApplicationStr wins when package.json omits application", func(t *testing.T) {
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
		got := resolveSourceApplication(&meta.Module{
			Name:           "enterprise_module",
			ApplicationStr: "enterprise",
		}, modulesPath, nil)
		if got != "enterprise" {
			t.Fatalf("application = %q, want enterprise", got)
		}
	})

	t.Run("falls back to module name heuristic without package metadata", func(t *testing.T) {
		got := resolveSourceApplication(&meta.Module{Name: "partner_bank"}, modulesPath, nil)
		if got != "partner" {
			t.Fatalf("application = %q, want partner", got)
		}
	})

	t.Run("lookup with empty application falls through", func(t *testing.T) {
		lookup := policy.ModuleApplicationLookup(func(string) (string, bool) { return "", true })
		got := resolveSourceApplication(&meta.Module{Name: "auth"}, modulesPath, lookup)
		if got != "auth" {
			t.Fatalf("application = %q, want auth", got)
		}
	})

	t.Run("lookup fills gap before name heuristic", func(t *testing.T) {
		lookup := policy.ModuleApplicationLookupFromMap(map[string]string{"mapped_module": "mapped"})
		got := resolveSourceApplication(&meta.Module{Name: "mapped_module"}, modulesPath, lookup)
		if got != "mapped" {
			t.Fatalf("application = %q, want mapped", got)
		}
	})

	t.Run("falls back to module ApplicationStr when name empty", func(t *testing.T) {
		got := resolveSourceApplication(&meta.Module{ApplicationStr: "orphan"}, modulesPath, nil)
		if got != "orphan" {
			t.Fatalf("application = %q, want orphan", got)
		}
	})

	t.Run("nil module returns empty", func(t *testing.T) {
		if got := resolveSourceApplication(nil, modulesPath, nil); got != "" {
			t.Fatalf("application = %q, want empty", got)
		}
	})
}

func TestReadModuleApplicationFromExistingPackageJSON(t *testing.T) {
	modulesPath := filepath.Join(t.TempDir(), "modules")
	if app, ok := readModuleApplicationFromExistingPackageJSON("", "auth"); ok || app != "" {
		t.Fatalf("empty modules path = %q ok=%v", app, ok)
	}
	if app, ok := readModuleApplicationFromExistingPackageJSON(modulesPath, ""); ok || app != "" {
		t.Fatalf("empty module name = %q ok=%v", app, ok)
	}
	if app, ok := readModuleApplicationFromExistingPackageJSON(modulesPath, "missing"); ok || app != "" {
		t.Fatalf("missing package.json = %q ok=%v", app, ok)
	}

	dir := filepath.Join(modulesPath, "no_app")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	content := `{
  "name": "@test/no_app",
  "version": "0.0.0-test",
  "choysum": { "moduleName": "no_app" }
}`
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if app, ok := readModuleApplicationFromExistingPackageJSON(modulesPath, "no_app"); ok || app != "" {
		t.Fatalf("missing application field = %q ok=%v", app, ok)
	}

	writeBoundaryPackageJSON(t, modulesPath, "auth", "auth")
	if app, ok := readModuleApplicationFromExistingPackageJSON(modulesPath, "auth"); !ok || app != "auth" {
		t.Fatalf("explicit application = %q ok=%v", app, ok)
	}
}

func TestMergeModuleApplicationsFromDisk_SkipsNonModuleDirs(t *testing.T) {
	modulesPath := filepath.Join(t.TempDir(), "modules")
	for _, dir := range []string{"node_modules", ".choysum", "auth"} {
		path := filepath.Join(modulesPath, dir)
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
	}
	writeBoundaryPackageJSON(t, modulesPath, "auth", "auth")

	appByModule := make(map[string]string)
	mergeModuleApplicationsFromDisk(modulesPath, appByModule)
	if _, ok := appByModule["node_modules"]; ok {
		t.Fatal("node_modules should not be indexed")
	}
	if appByModule["auth"] != "auth" {
		t.Fatalf("auth app = %q", appByModule["auth"])
	}

	mergeModuleApplicationsFromDisk("", appByModule)
	mergeModuleApplicationsFromDisk(modulesPath, nil)
	mergeModuleApplicationsFromDisk(filepath.Join(t.TempDir(), "missing"), make(map[string]string))
}

func TestModuleApplicationLookup_NilEnvDoesNotPanic(t *testing.T) {
	modulesPath := filepath.Join(t.TempDir(), "modules")
	writeBoundaryPackageJSON(t, modulesPath, "auth", "auth")
	plugin := &BackendPlugin{BasePlugin: &esbplugins.BasePlugin{
		Module: &meta.Module{Name: "auth", ApplicationStr: "auth"},
	}}
	if lookup := plugin.moduleApplicationLookup(nil, modulesPath); lookup == nil {
		t.Fatal("expected lookup")
	}
}

func TestModuleApplicationLookup_SessionFallback(t *testing.T) {
	testScope, db := newPluginSessionTestScope(t)
	migrateBackendPluginMetadata(t, db)
	modulesPath := testScope.cfg.ModulesPath
	writeBoundaryPackageJSON(t, modulesPath, "auth", "auth")

	if err := db.Create(&meta.Module{Name: "session_only", ApplicationStr: "session_app", Path: filepath.Join(modulesPath, "session_only")}).Error; err != nil {
		t.Fatalf("create module row: %v", err)
	}

	plugin := &BackendPlugin{BasePlugin: &esbplugins.BasePlugin{
		Env:    testScope,
		Module: &meta.Module{Name: "auth", Path: filepath.Join(modulesPath, "auth"), ApplicationStr: "auth"},
	}}
	plugin.runtimeOptions = runtimeOptions{modulesPath: modulesPath}

	lookup := plugin.moduleApplicationLookup([]*parser.ParserResult{{
		Imports: map[string]*parser.Import{
			"X": {ModuleSpecPath: filepath.Join(modulesPath, "session_only", "service", "models", "x")},
		},
	}}, modulesPath)
	if app, ok := lookup("session_only"); !ok || app != "session_app" {
		t.Fatalf("session lookup session_only = %q ok=%v", app, ok)
	}
}

func TestEnforceServiceImportBoundary_NilModule(t *testing.T) {
	plugin := &BackendPlugin{}
	if err := plugin.enforceServiceImportBoundary(nil); err != nil {
		t.Fatalf("nil module should noop, got %v", err)
	}
}

func TestEnforceServiceImportBoundary_EmptyModulePath(t *testing.T) {
	plugin := &BackendPlugin{BasePlugin: &esbplugins.BasePlugin{
		Module: &meta.Module{Name: "auth", ApplicationStr: "auth"},
	}}
	plugin.runtimeOptions = runtimeOptions{modulesPath: t.TempDir()}
	if err := plugin.enforceServiceImportBoundary(nil); err != nil {
		t.Fatalf("empty module path should noop, got %v", err)
	}
}

func TestEnforceServiceImportBoundary_EmptySourceApplication(t *testing.T) {
	plugin := &BackendPlugin{BasePlugin: &esbplugins.BasePlugin{
		Module: &meta.Module{Name: "", Path: t.TempDir(), ApplicationStr: ""},
	}}
	plugin.runtimeOptions = runtimeOptions{modulesPath: t.TempDir()}
	if err := plugin.enforceServiceImportBoundary(nil); err != nil {
		t.Fatalf("empty source app should noop, got %v", err)
	}
}

func TestEnforceServiceImportBoundary_ReturnsViolationError(t *testing.T) {
	modulesPath := filepath.Join(t.TempDir(), "modules")
	partnerRoot := filepath.Join(modulesPath, "partner")
	authSpec := filepath.Join(modulesPath, "auth", "service", "models", "role")
	source := filepath.Join(partnerRoot, "service", "models", "partner.ts")
	writeBoundaryPackageJSON(t, modulesPath, "partner", "partner")
	writeBoundaryPackageJSON(t, modulesPath, "auth", "auth")

	plugin := &BackendPlugin{BasePlugin: &esbplugins.BasePlugin{
		Module: &meta.Module{Name: "partner", Path: partnerRoot, ApplicationStr: "partner"},
	}}
	plugin.runtimeOptions = runtimeOptions{modulesPath: modulesPath}

	err := plugin.enforceServiceImportBoundary([]*parser.ParserResult{{
		Path: source,
		Imports: map[string]*parser.Import{
			"Role": {
				ModuleSpecPath: authSpec,
				ModuleSpecText: "@/auth/service/models/role",
				IsTypeOnly:     false,
				Line:           1,
				Column:         1,
			},
		},
	}})
	if err == nil || !strings.Contains(err.Error(), "cross-app service import boundary violation") {
		t.Fatalf("enforceServiceImportBoundary() error = %v, want boundary failure", err)
	}
}

func TestResolveSourceApplication_EmptyNameUsesApplicationStr(t *testing.T) {
	got := resolveSourceApplication(&meta.Module{ApplicationStr: "enterprise"}, t.TempDir(), nil)
	if got != "enterprise" {
		t.Fatalf("application = %q, want enterprise", got)
	}
}

func TestModuleApplicationLookup_SkipsSessionWhenDiskAlreadyIndexed(t *testing.T) {
	testScope, db := newPluginSessionTestScope(t)
	migrateBackendPluginMetadata(t, db)
	modulesPath := testScope.cfg.ModulesPath
	writeBoundaryPackageJSON(t, modulesPath, "auth", "auth")
	if err := db.Create(&meta.Module{Name: "auth", ApplicationStr: "session_override", Path: filepath.Join(modulesPath, "auth")}).Error; err != nil {
		t.Fatalf("create module row: %v", err)
	}

	plugin := &BackendPlugin{BasePlugin: &esbplugins.BasePlugin{
		Env:    testScope,
		Module: &meta.Module{Name: "auth", Path: filepath.Join(modulesPath, "auth"), ApplicationStr: "auth"},
	}}
	plugin.runtimeOptions = runtimeOptions{modulesPath: modulesPath}
	lookup := plugin.moduleApplicationLookup(nil, modulesPath)
	if app, ok := lookup("auth"); !ok || app != "auth" {
		t.Fatalf("disk application should win, got %q ok=%v", app, ok)
	}
}

func TestMergeModuleApplicationsFromDisk_SkipsInvalidPackageJSON(t *testing.T) {
	modulesPath := filepath.Join(t.TempDir(), "modules")
	dir := filepath.Join(modulesPath, "broken")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte("{"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	appByModule := make(map[string]string)
	mergeModuleApplicationsFromDisk(modulesPath, appByModule)
	if len(appByModule) != 0 {
		t.Fatalf("expected no entries, got %#v", appByModule)
	}
}

func TestModuleApplicationLookup_IgnoresSessionRowsWithEmptyFields(t *testing.T) {
	testScope, db := newPluginSessionTestScope(t)
	migrateBackendPluginMetadata(t, db)
	modulesPath := testScope.cfg.ModulesPath
	writeBoundaryPackageJSON(t, modulesPath, "auth", "auth")
	for _, row := range []meta.Module{
		{Name: "", ApplicationStr: "orphan", Path: filepath.Join(modulesPath, "orphan")},
		{Name: "blank_app", ApplicationStr: "", Path: filepath.Join(modulesPath, "blank_app")},
	} {
		if err := db.Create(&row).Error; err != nil {
			t.Fatalf("create module row: %v", err)
		}
	}

	plugin := &BackendPlugin{BasePlugin: &esbplugins.BasePlugin{
		Env:    testScope,
		Module: &meta.Module{Name: "auth", Path: filepath.Join(modulesPath, "auth"), ApplicationStr: "auth"},
	}}
	plugin.runtimeOptions = runtimeOptions{modulesPath: modulesPath}
	lookup := plugin.moduleApplicationLookup([]*parser.ParserResult{{
		Imports: map[string]*parser.Import{
			"X": {ModuleSpecPath: filepath.Join(modulesPath, "blank_app", "service", "models", "x")},
		},
	}}, modulesPath)
	if app, ok := lookup("blank_app"); !ok || app != "blank_app" {
		t.Fatalf("blank_app fallback = %q ok=%v", app, ok)
	}
}

func writeBoundaryPackageJSON(t *testing.T, modulesPath, moduleName, application string) {
	t.Helper()
	dir := filepath.Join(modulesPath, moduleName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", dir, err)
	}
	content := `{
  "name": "@test/` + moduleName + `",
  "version": "0.0.0-test",
  "choysum": {
    "moduleName": "` + moduleName + `",
    "application": "` + application + `"
  }
}`
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile package.json: %v", err)
	}
}
