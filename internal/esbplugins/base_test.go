// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package esbplugins

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/choysum-dev/choysum/internal/parser"
	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
)

type basePluginTestScope struct{}

func (s *basePluginTestScope) Run(fn func(scope.Scope) error) error { return fn(s) }
func (s *basePluginTestScope) Transactor() scope.Transactor {
	return scopetest.NewPassthroughTransactor(s)
}
func (s *basePluginTestScope) Session() *scope.Session { return nil }
func (s *basePluginTestScope) WithContext(ctx context.Context) scope.Scope {
	if ctx == nil {
		ctx = context.Background()
	}
	return s
}
func (s *basePluginTestScope) Context() context.Context { return context.Background() }
func (s *basePluginTestScope) Logger() *slog.Logger     { return slog.Default() }

func TestNewBasePluginInitializesSharedState(t *testing.T) {
	runtimeScope := &basePluginTestScope{}
	module := &meta.Module{Name: "auth", Path: "/virtual/modules/auth"}
	plugin := NewBasePlugin(runtimeScope, module, "./service/index.ts")
	if plugin == nil {
		t.Fatal("expected non-nil plugin")
	}
	if plugin.Env != runtimeScope || plugin.Module != module || plugin.EntryPoint != "./service/index.ts" {
		t.Fatalf("unexpected plugin state: %#v", plugin)
	}
	if plugin.ParserResultChan == nil || plugin.TsExports == nil || plugin.normalizedTsExp == nil || plugin.ParserResults == nil {
		t.Fatalf("expected initialized plugin maps/channels, got %#v", plugin)
	}
}

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

	t.Run("resolves exports across directory and index module spec forms", func(t *testing.T) {
		realRoot := filepath.Join(t.TempDir(), "real")
		realDecoratorDir := filepath.Join(realRoot, "modules", "core", "service", "orm", "decorator")
		if err := os.MkdirAll(realDecoratorDir, 0o755); err != nil {
			t.Fatalf("mkdir decorator directory: %v", err)
		}
		if err := os.WriteFile(filepath.Join(realDecoratorDir, "index.ts"), []byte("export const Model = () => null;\n"), 0o644); err != nil {
			t.Fatalf("write decorator index file: %v", err)
		}

		directorySpec := realDecoratorDir
		indexSpec := filepath.Join(realDecoratorDir, "index")
		plugin := &BasePlugin{
			TsExports: map[string]map[string]*parser.Export{
				directorySpec: {
					"Model": {ReferenceIdent: "Model", ModuleSpecPath: directorySpec},
				},
			},
		}

		moduleSpec, referenceIdent := plugin.FindModuleSpecAndReferenceIdent(indexSpec, "Model")
		if moduleSpec != directorySpec || referenceIdent != "Model" {
			t.Fatalf("unexpected directory/index export resolution: %q %q", moduleSpec, referenceIdent)
		}

		moduleSpec, referenceIdent = plugin.FindModuleSpecAndReferenceIdent(filepath.Join(realDecoratorDir, "index.ts"), "Model")
		if moduleSpec != directorySpec || referenceIdent != "Model" {
			t.Fatalf("unexpected directory/index.ts export resolution: %q %q", moduleSpec, referenceIdent)
		}
	})

	t.Run("resolves exports when separators differ between key and lookup", func(t *testing.T) {
		if filepath.Separator != '\\' {
			t.Skip("separator normalization via filepath.ToSlash differs on non-Windows platforms")
		}

		keySpec := `C:\repo\modules\core\service\orm\decorator`
		lookupSpec := filepath.ToSlash(keySpec)
		plugin := &BasePlugin{
			TsExports: map[string]map[string]*parser.Export{
				keySpec: {
					"Model": {ReferenceIdent: "Model", ModuleSpecPath: keySpec},
				},
			},
		}

		moduleSpec, referenceIdent := plugin.FindModuleSpecAndReferenceIdent(lookupSpec, "Model")
		if moduleSpec != keySpec || referenceIdent != "Model" {
			t.Fatalf("unexpected separator-normalized export resolution: %q %q", moduleSpec, referenceIdent)
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
			Path: "/Users/demo/.choysum/modules/base/service/models/language.ts",
			Exports: map[string]*parser.Export{
				"default": {ReferenceIdent: "Language", ModuleSpecPath: "/Users/demo/.choysum/modules/base/service/models/language"},
			},
		},
	}

	plugin.generateTsExportsMap(results)

	if _, ok := plugin.TsExports["/Users/demo/.choysum/modules/base/service/models/language"]; !ok {
		t.Fatalf("expected ts export map key to preserve dot directories, got %#v", plugin.TsExports)
	}
	if _, badKey := plugin.TsExports["/Users/demo/"]; badKey {
		t.Fatalf("unexpected truncated ts export key for dot directory path: %#v", plugin.TsExports)
	}
}

func TestNormalizeModuleSpecPathExistingIndexAliasesResolveToDirectory(t *testing.T) {
	decoratorDir := filepath.Join(t.TempDir(), "service", "orm", "decorator")
	if err := os.MkdirAll(decoratorDir, 0o755); err != nil {
		t.Fatalf("mkdir decorator directory: %v", err)
	}

	indexNoExt := filepath.Join(decoratorDir, "index")
	if err := os.WriteFile(indexNoExt, []byte("placeholder\n"), 0o644); err != nil {
		t.Fatalf("write index file without extension: %v", err)
	}
	indexVue := filepath.Join(decoratorDir, "index.vue")
	if err := os.WriteFile(indexVue, []byte("<template/>\n"), 0o644); err != nil {
		t.Fatalf("write index.vue file: %v", err)
	}

	expectedDir, err := filepath.EvalSymlinks(decoratorDir)
	if err != nil {
		expectedDir = filepath.Clean(decoratorDir)
	} else {
		expectedDir = filepath.Clean(expectedDir)
	}

	if got := NormalizeModuleSpecPath(indexNoExt); got != expectedDir {
		t.Fatalf("NormalizeModuleSpecPath(index no extension) = %q, want %q", got, expectedDir)
	}
	if got := NormalizeModuleSpecPath(indexVue); got != expectedDir {
		t.Fatalf("NormalizeModuleSpecPath(index.vue) = %q, want %q", got, expectedDir)
	}
}

func TestNormalizeModuleSpecPathRelativeSpecifierIgnoresCwdSymlink(t *testing.T) {
	cwd := t.TempDir()
	t.Chdir(cwd)

	targetDir := filepath.Join(t.TempDir(), "node_modules", "lodash")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatalf("mkdir symlink target: %v", err)
	}
	if err := os.Symlink(targetDir, filepath.Join(cwd, "lodash")); err != nil {
		t.Skipf("symlink not supported in this environment: %v", err)
	}

	if got := NormalizeModuleSpecPath("lodash"); got != "lodash" {
		t.Fatalf("NormalizeModuleSpecPath(relative specifier) = %q, want %q", got, "lodash")
	}
}

func TestNormalizePathAndPathWithinRootForNonExistingFileUnderSymlinkParent(t *testing.T) {
	realRoot := filepath.Join(t.TempDir(), "real")
	moduleRealRoot := filepath.Join(realRoot, "modules", "base")
	if err := os.MkdirAll(filepath.Join(moduleRealRoot, "service", "models"), 0o755); err != nil {
		t.Fatalf("mkdir module models directory: %v", err)
	}

	aliasRoot := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(realRoot, aliasRoot); err != nil {
		t.Skipf("symlink not supported in this environment: %v", err)
	}

	moduleAliasRoot := filepath.Join(aliasRoot, "modules", "base")
	nonExistingAliasPath := filepath.Join(moduleAliasRoot, "service", "models", "new_model.ts")

	expectedDir, err := filepath.EvalSymlinks(filepath.Dir(nonExistingAliasPath))
	if err != nil {
		t.Fatalf("resolve expected parent symlink: %v", err)
	}
	expectedPath := filepath.Clean(filepath.Join(expectedDir, filepath.Base(nonExistingAliasPath)))

	if got := NormalizePath(nonExistingAliasPath); got != expectedPath {
		t.Fatalf("NormalizePath(non-existing path under alias) = %q, want %q", got, expectedPath)
	}
	if !PathWithinRoot(nonExistingAliasPath, moduleAliasRoot) {
		t.Fatalf("expected non-existing alias path %q to be within alias root %q", nonExistingAliasPath, moduleAliasRoot)
	}
}

func TestNormalizePathForDeepNonExistingPathUsesNearestExistingSymlinkAncestor(t *testing.T) {
	realRoot := filepath.Join(t.TempDir(), "real")
	if err := os.MkdirAll(realRoot, 0o755); err != nil {
		t.Fatalf("mkdir real root: %v", err)
	}

	aliasRoot := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(realRoot, aliasRoot); err != nil {
		t.Skipf("symlink not supported in this environment: %v", err)
	}

	deepNonExisting := filepath.Join(aliasRoot, "modules", "crm", "service", "models", "lead.ts")
	resolvedAliasRoot, err := filepath.EvalSymlinks(aliasRoot)
	if err != nil {
		t.Fatalf("resolve alias root symlink: %v", err)
	}
	expected := filepath.Clean(filepath.Join(resolvedAliasRoot, "modules", "crm", "service", "models", "lead.ts"))

	if got := NormalizePath(deepNonExisting); got != expected {
		t.Fatalf("NormalizePath(deep non-existing path) = %q, want %q", got, expected)
	}
}
