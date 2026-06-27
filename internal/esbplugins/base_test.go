// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package esbplugins

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/choysum-dev/choysum/internal/parser"
)

func TestHandleParserResults(t *testing.T) {
	plugin := &BasePlugin{
		ParserResultChan: make(chan *parser.ParserResult, 2),
	}
	plugin.Wg.Add(1)
	go func() {
		defer plugin.Wg.Done()
		plugin.ParserResultChan <- &parser.ParserResult{
			Path:    "pages/index.ts",
			Exports: map[string]*parser.Export{"default": {ReferenceIdent: "Page", ModuleSpecPath: "pages/index"}},
		}
		plugin.ParserResultChan <- &parser.ParserResult{
			Path:    "components/button.vue",
			Exports: map[string]*parser.Export{"Button": {ReferenceIdent: "Button", ModuleSpecPath: "components/button.vue"}},
		}
	}()

	results := plugin.HandleParserResults()
	if len(results) != 2 {
		t.Fatalf("expected 2 parser results, got %d", len(results))
	}
	if results[0].Path != "components/button.vue" || results[1].Path != "pages/index.ts" {
		t.Fatalf("expected results to preserve reverse collection order, got %#v", []string{results[0].Path, results[1].Path})
	}
	if len(plugin.ParserResults) != 2 {
		t.Fatalf("expected plugin.ParserResults to be populated, got %d entries", len(plugin.ParserResults))
	}
	if _, ok := plugin.TsExports["pages/index"]; !ok {
		t.Fatalf("expected ts export map for ts file, got %#v", plugin.TsExports)
	}
	if _, ok := plugin.TsExports["pages"]; !ok {
		t.Fatalf("expected directory alias for index.ts export, got %#v", plugin.TsExports)
	}
	if _, ok := plugin.TsExports["components/button.vue"]; !ok {
		t.Fatalf("expected vue export map, got %#v", plugin.TsExports)
	}
}

func TestFindModuleSpecAndReferenceIdent(t *testing.T) {
	t.Run("returns direct export", func(t *testing.T) {
		plugin := &BasePlugin{
			TsExports: map[string]map[string]*parser.Export{
				"ui/button": {
					"Button": {ReferenceIdent: "Button", ModuleSpecPath: "ui/button"},
				},
			},
		}

		moduleSpec, referenceIdent := plugin.FindModuleSpecAndReferenceIdent("ui/button", "Button")
		if moduleSpec != "ui/button" || referenceIdent != "Button" {
			t.Fatalf("unexpected resolved export: %q %q", moduleSpec, referenceIdent)
		}
	})

	t.Run("maps parser default export class name back to default", func(t *testing.T) {
		plugin := &BasePlugin{
			TsExports: map[string]map[string]*parser.Export{
				"ui/modal": {
					"default": {ReferenceIdent: "ModalView", ModuleSpecPath: "ui/modal"},
				},
			},
		}

		moduleSpec, referenceIdent := plugin.FindModuleSpecAndReferenceIdent("ui/modal", "ModalView")
		if moduleSpec != "ui/modal" || referenceIdent != "default" {
			t.Fatalf("unexpected default export resolution: %q %q", moduleSpec, referenceIdent)
		}
	})

	t.Run("resolves wildcard exports recursively", func(t *testing.T) {
		plugin := &BasePlugin{
			TsExports: map[string]map[string]*parser.Export{
				"ui/index": {
					"*": {Wildcard: []*parser.Export{{ModuleSpecPath: "ui/forms"}}},
				},
				"ui/forms": {
					"*": {Wildcard: []*parser.Export{{ModuleSpecPath: "ui/fields"}}},
				},
				"ui/fields": {
					"Field": {ReferenceIdent: "Field", ModuleSpecPath: "ui/fields"},
				},
			},
		}

		moduleSpec, referenceIdent := plugin.FindModuleSpecAndReferenceIdent("ui/index", "Field")
		if moduleSpec != "ui/fields" || referenceIdent != "Field" {
			t.Fatalf("unexpected wildcard resolution: %q %q", moduleSpec, referenceIdent)
		}
	})

	t.Run("returns empty when module export is missing", func(t *testing.T) {
		plugin := &BasePlugin{
			TsExports: map[string]map[string]*parser.Export{
				"ui/index": {},
			},
		}

		moduleSpec, referenceIdent := plugin.FindModuleSpecAndReferenceIdent("ui/index", "Missing")
		if moduleSpec != "" || referenceIdent != "" {
			t.Fatalf("expected empty resolution, got %q %q", moduleSpec, referenceIdent)
		}
	})

	t.Run("resolves exports across symlinked module spec aliases", func(t *testing.T) {
		realRoot := filepath.Join(t.TempDir(), "real")
		realServiceDir := filepath.Join(realRoot, "modules", "core", "service")
		if err := os.MkdirAll(filepath.Join(realServiceDir, "models"), 0o755); err != nil {
			t.Fatalf("mkdir real service directory: %v", err)
		}
		if err := os.WriteFile(filepath.Join(realServiceDir, "index.ts"), []byte("export default class BaseModel {}\n"), 0o644); err != nil {
			t.Fatalf("write real index file: %v", err)
		}
		if err := os.WriteFile(filepath.Join(realServiceDir, "models", "currency.ts"), []byte("export class Currency {}\n"), 0o644); err != nil {
			t.Fatalf("write real model file: %v", err)
		}

		aliasRoot := filepath.Join(t.TempDir(), "alias")
		if err := os.Symlink(realRoot, aliasRoot); err != nil {
			t.Skipf("symlink not supported in this environment: %v", err)
		}

		aliasServiceSpec := filepath.Join(aliasRoot, "modules", "core", "service")
		aliasModelSpec := filepath.Join(aliasServiceSpec, "models", "currency")
		realServiceSpec := filepath.Join(realRoot, "modules", "core", "service")
		realModelSpec := filepath.Join(realServiceSpec, "models", "currency")

		plugin := &BasePlugin{
			TsExports: map[string]map[string]*parser.Export{
				aliasServiceSpec: {
					"default": {ReferenceIdent: "BaseModel", ModuleSpecPath: aliasServiceSpec},
				},
				aliasModelSpec: {
					"Currency": {ReferenceIdent: "Currency", ModuleSpecPath: aliasModelSpec},
				},
			},
		}

		moduleSpec, referenceIdent := plugin.FindModuleSpecAndReferenceIdent(realServiceSpec, "BaseModel")
		if moduleSpec != aliasServiceSpec || referenceIdent != "default" {
			t.Fatalf("unexpected symlinked directory export resolution: %q %q", moduleSpec, referenceIdent)
		}

		moduleSpec, referenceIdent = plugin.FindModuleSpecAndReferenceIdent(realModelSpec, "Currency")
		if moduleSpec != aliasModelSpec || referenceIdent != "Currency" {
			t.Fatalf("unexpected symlinked extensionless export resolution: %q %q", moduleSpec, referenceIdent)
		}
	})
}

func TestGenerateTsExportsMapDirectly(t *testing.T) {
	plugin := &BasePlugin{}
	results := []*parser.ParserResult{
		{Path: "models/user.ts", Exports: map[string]*parser.Export{"User": {ReferenceIdent: "User", ModuleSpecPath: "models/user"}}},
		{Path: "views/home.vue", Exports: map[string]*parser.Export{"default": {ReferenceIdent: "HomeView", ModuleSpecPath: "views/home.vue"}}},
	}

	plugin.generateTsExportsMap(results)

	if got := plugin.TsExports["models/user"]["User"].ModuleSpecPath; got != "models/user" {
		t.Fatalf("unexpected ts export module path: %q", got)
	}
	if got := plugin.TsExports["views/home.vue"]["default"].ReferenceIdent; got != "HomeView" {
		t.Fatalf("unexpected vue export reference: %q", got)
	}
}

func TestGenerateTsExportsMapPreservesDotDirectories(t *testing.T) {
	plugin := &BasePlugin{}
	results := []*parser.ParserResult{
		{
			Path: "/Users/demo/.choysum/modules/base/service/models/locale.ts",
			Exports: map[string]*parser.Export{
				"default": {ReferenceIdent: "Locale", ModuleSpecPath: "/Users/demo/.choysum/modules/base/service/models/locale"},
			},
		},
	}

	plugin.generateTsExportsMap(results)

	if _, ok := plugin.TsExports["/Users/demo/.choysum/modules/base/service/models/locale"]; !ok {
		t.Fatalf("expected ts export map key to preserve dot directories, got %#v", plugin.TsExports)
	}
	if _, badKey := plugin.TsExports["/Users/demo/"]; badKey {
		t.Fatalf("unexpected truncated ts export key for dot directory path: %#v", plugin.TsExports)
	}
}
