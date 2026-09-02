// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package backendplugin

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/choysum-dev/choysum/internal/esbplugins"
	"github.com/choysum-dev/choysum/internal/module/policy"
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

	t.Run("lookup fills gap before name heuristic", func(t *testing.T) {
		lookup := policy.ModuleApplicationLookupFromMap(map[string]string{"mapped_module": "mapped"})
		got := resolveSourceApplication(&meta.Module{Name: "mapped_module"}, modulesPath, lookup)
		if got != "mapped" {
			t.Fatalf("application = %q, want mapped", got)
		}
	})
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

func TestEnforceServiceImportBoundary_NilModule(t *testing.T) {
	plugin := &BackendPlugin{}
	if err := plugin.enforceServiceImportBoundary(nil); err != nil {
		t.Fatalf("nil module should noop, got %v", err)
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
