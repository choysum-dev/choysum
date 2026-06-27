// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package backendplugin

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/choysum-dev/choysum/internal/esbplugins"
	"github.com/choysum-dev/choysum/internal/parser"
	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/evanw/esbuild/pkg/api"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type testScope struct {
	ctx     context.Context
	cfg     *config.Config
	logger  *slog.Logger
	session *scope.Session
}

func (e *testScope) Run(fn func(runtimeScope scope.Scope) error) error { return fn(e) }
func (e *testScope) Transactor() scope.Transactor                      { return scopetest.NewPassthroughTransactor(e) }
func (e *testScope) Session() *scope.Session                           { return e.session }
func (e *testScope) WithContext(ctx context.Context) scope.Scope {
	return &testScope{ctx: ctx, cfg: e.cfg, logger: e.logger, session: e.session}
}
func (e *testScope) Context() context.Context { return e.ctx }
func (e *testScope) Logger() *slog.Logger {
	if e.logger != nil {
		return e.logger
	}
	return slog.Default()
}
func (e *testScope) Config() *config.Config { return e.cfg }

func (e *testScope) FactoryInput() scope.FactoryInput {
	return scopetest.FactoryInputFromConfig(e.Config())
}

func newPluginTestScope() scope.Scope {
	return &testScope{
		ctx:    context.Background(),
		cfg:    &config.Config{ModulesPath: "/virtual/modules"},
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

func newPluginSessionTestScope(t *testing.T) (*testScope, *gorm.DB) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "backendplugin.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}

	return &testScope{
		ctx:     context.Background(),
		cfg:     &config.Config{ModulesPath: filepath.Join(t.TempDir(), "modules")},
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		session: &scope.Session{DB: db},
	}, db
}

func migrateBackendPluginMetadata(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.AutoMigrate(&meta.IrApplication{}, &meta.IrModule{}, &meta.IrModel{}, &meta.IrField{}, &meta.IrDecorator{}, &meta.IrArgument{}); err != nil {
		t.Fatalf("automigrate metadata: %v", err)
	}
}

func TestBackendPluginPathWithinRoot_ResolvesSymlinkAliases(t *testing.T) {
	realRoot := filepath.Join(t.TempDir(), "real")
	moduleRealRoot := filepath.Join(realRoot, "modules", "base")
	insideRealPath := filepath.Join(moduleRealRoot, "service", "models", "currency.ts")
	if err := os.MkdirAll(filepath.Dir(insideRealPath), 0o755); err != nil {
		t.Fatalf("mkdir inside real path: %v", err)
	}
	if err := os.WriteFile(insideRealPath, []byte("export {};\n"), 0o644); err != nil {
		t.Fatalf("write inside real file: %v", err)
	}

	aliasRoot := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(realRoot, aliasRoot); err != nil {
		t.Skipf("symlink not supported in this environment: %v", err)
	}

	moduleAliasRoot := filepath.Join(aliasRoot, "modules", "base")
	insideAliasPath := filepath.Join(moduleAliasRoot, "service", "models", "currency.ts")
	if !backendPluginPathWithinRoot(insideRealPath, moduleAliasRoot) {
		t.Fatalf("expected real path %q to be within alias module root %q", insideRealPath, moduleAliasRoot)
	}
	if !backendPluginPathWithinRoot(insideAliasPath, moduleAliasRoot) {
		t.Fatalf("expected alias path %q to be within alias module root %q", insideAliasPath, moduleAliasRoot)
	}

	outsideRealPath := filepath.Join(realRoot, "modules", "auth", "service", "models", "user.ts")
	if err := os.MkdirAll(filepath.Dir(outsideRealPath), 0o755); err != nil {
		t.Fatalf("mkdir outside real path: %v", err)
	}
	if err := os.WriteFile(outsideRealPath, []byte("export {};\n"), 0o644); err != nil {
		t.Fatalf("write outside real file: %v", err)
	}
	if backendPluginPathWithinRoot(outsideRealPath, moduleAliasRoot) {
		t.Fatalf("expected outside path %q not to be within alias module root %q", outsideRealPath, moduleAliasRoot)
	}
}

func TestBackendPluginSameModuleSpecPath_ResolvesSymlinkAndExtensionlessSpecs(t *testing.T) {
	realRoot := filepath.Join(t.TempDir(), "real")
	realDecoratorSpec := filepath.Join(realRoot, "modules", "core", "service", "orm", "decorator", "model")
	realDecoratorFile := realDecoratorSpec + ".ts"
	if err := os.MkdirAll(filepath.Dir(realDecoratorFile), 0o755); err != nil {
		t.Fatalf("mkdir decorator dir: %v", err)
	}
	if err := os.WriteFile(realDecoratorFile, []byte("export const Model = () => null;\n"), 0o644); err != nil {
		t.Fatalf("write decorator file: %v", err)
	}

	aliasRoot := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(realRoot, aliasRoot); err != nil {
		t.Skipf("symlink not supported in this environment: %v", err)
	}

	aliasDecoratorSpec := filepath.Join(aliasRoot, "modules", "core", "service", "orm", "decorator", "model")
	if !backendPluginSameModuleSpecPath(realDecoratorSpec, aliasDecoratorSpec) {
		t.Fatalf("expected extensionless module specs to match across symlink aliases: %q vs %q", realDecoratorSpec, aliasDecoratorSpec)
	}
	if !backendPluginSameModuleSpecPath(realDecoratorFile, aliasDecoratorSpec) {
		t.Fatalf("expected .ts and extensionless specs to match across symlink aliases: %q vs %q", realDecoratorFile, aliasDecoratorSpec)
	}

	realDecoratorDir := filepath.Dir(realDecoratorFile)
	realDecoratorIndexFile := filepath.Join(realDecoratorDir, "index.ts")
	if err := os.WriteFile(realDecoratorIndexFile, []byte("export { Model } from './model';\n"), 0o644); err != nil {
		t.Fatalf("write decorator index file: %v", err)
	}
	aliasDecoratorIndexSpec := filepath.Join(aliasRoot, "modules", "core", "service", "orm", "decorator", "index")
	if !backendPluginSameModuleSpecPath(realDecoratorDir, aliasDecoratorIndexSpec) {
		t.Fatalf("expected directory and index specs to match across symlink aliases: %q vs %q", realDecoratorDir, aliasDecoratorIndexSpec)
	}
	if !backendPluginSameModuleSpecPath(realDecoratorIndexFile, aliasDecoratorIndexSpec) {
		t.Fatalf("expected index.ts and index specs to match across symlink aliases: %q vs %q", realDecoratorIndexFile, aliasDecoratorIndexSpec)
	}

	realFieldSpec := filepath.Join(realRoot, "modules", "core", "service", "orm", "decorator", "field")
	realFieldFile := realFieldSpec + ".ts"
	if err := os.WriteFile(realFieldFile, []byte("export const Field = () => null;\n"), 0o644); err != nil {
		t.Fatalf("write field file: %v", err)
	}
	if backendPluginSameModuleSpecPath(realFieldSpec, aliasDecoratorSpec) {
		t.Fatalf("expected different decorator specs not to match: %q vs %q", realFieldSpec, aliasDecoratorSpec)
	}
}

func mustIndex(t *testing.T, s string, sub string) int {
	t.Helper()
	idx := strings.Index(s, sub)
	if idx < 0 {
		t.Fatalf("substring %q not found in %q", sub, s)
	}
	return idx
}

func newBackendPluginForInjectTest(t *testing.T, runtimeScope scope.Scope) *BackendPlugin {
	t.Helper()
	modelDecoratorModuleSpec, _ := meta.ModelDecoratorModuleSpec(runtimeScope)

	return &BackendPlugin{
		BasePlugin: &esbplugins.BasePlugin{
			Env:    runtimeScope,
			Module: &meta.IrModule{Path: "/virtual/modules/auth", ApplicationStr: "auth"},
			TsExports: map[string]map[string]*parser.Export{
				modelDecoratorModuleSpec: {
					"Model": {
						ModuleSpecPath: modelDecoratorModuleSpec,
						ReferenceIdent: "Model",
					},
				},
			},
		},
	}
}

func TestInjectModelApplication_InsertsSecondArgUsingStableFields(t *testing.T) {
	testRuntimeScope := newPluginTestScope()
	plugin := newBackendPluginForInjectTest(t, testRuntimeScope)
	modelDecoratorModuleSpec, _ := meta.ModelDecoratorModuleSpec(testRuntimeScope)

	raw := "@Model('User')\nexport default class User {}\n"
	argEnd := mustIndex(t, raw, "'User'") + len("'User'")

	result := &parser.ParserResult{
		Path:       "/virtual/modules/auth/service/user.ts",
		RawContent: raw,
		ModelClassNode: &parser.Class{
			Decorators: []*parser.Decorator{
				{
					ModuleSpecPath: modelDecoratorModuleSpec,
					ReferenceIdent: "Model",
					Arguments: []*parser.Argument{
						{Type: "Literal", End: argEnd},
					},
				},
			},
		},
	}

	if err := plugin.injectModelApplication([]*parser.ParserResult{result}); err != nil {
		t.Fatalf("injectModelApplication failed: %v", err)
	}

	if !strings.Contains(result.Content, "@Model('User', { application: 'auth' })") {
		t.Fatalf("expected application option to be inserted, got:\n%s", result.Content)
	}
}

func TestInjectModelApplication_SkipsWhenApplicationAlreadyExists(t *testing.T) {
	testRuntimeScope := newPluginTestScope()
	plugin := newBackendPluginForInjectTest(t, testRuntimeScope)
	modelDecoratorModuleSpec, _ := meta.ModelDecoratorModuleSpec(testRuntimeScope)

	raw := "@Model('User', { application: 'legacy' })\nexport default class User {}\n"
	literalStart := mustIndex(t, raw, "'legacy'")
	literalEnd := literalStart + len("'legacy'")
	objectEnd := mustIndex(t, raw, "}") + 1

	result := &parser.ParserResult{
		Path:       "/virtual/modules/auth/service/user.ts",
		RawContent: raw,
		ModelClassNode: &parser.Class{
			Decorators: []*parser.Decorator{
				{
					ModuleSpecPath: modelDecoratorModuleSpec,
					ReferenceIdent: "Model",
					Arguments: []*parser.Argument{
						{Type: "Literal", End: mustIndex(t, raw, "'User'") + len("'User'")},
						{
							Type: "ObjectLiteral",
							End:  objectEnd,
							ObjectProperties: []*parser.ObjectProperty{
								{
									Name:       "application",
									ValueStart: literalStart,
									ValueEnd:   literalEnd,
								},
							},
						},
					},
				},
			},
		},
	}

	if err := plugin.injectModelApplication([]*parser.ParserResult{result}); err != nil {
		t.Fatalf("injectModelApplication failed: %v", err)
	}

	if result.Content != raw {
		t.Fatalf("expected content unchanged when application already exists, got:\n%s", result.Content)
	}
}

func TestInjectModelApplication_MultipleEditsSameFile_WithMultibytePrefix(t *testing.T) {
	testRuntimeScope := newPluginTestScope()
	plugin := newBackendPluginForInjectTest(t, testRuntimeScope)
	modelDecoratorModuleSpec, _ := meta.ModelDecoratorModuleSpec(testRuntimeScope)

	raw := "// emoji😀prefix\n@Model('First')\nexport class First {}\n\n@Model('Second', {\n  readonly: true\n})\nexport class Second {}\n"

	firstArgEnd := mustIndex(t, raw, "'First'") + len("'First'")
	secondArgEnd := mustIndex(t, raw, "'Second'") + len("'Second'")
	secondObjectEnd := mustIndex(t, raw, "})") + 1

	firstResult := &parser.ParserResult{
		Path:       "/virtual/modules/auth/service/multi_model.ts",
		RawContent: raw,
		ModelClassNode: &parser.Class{
			Decorators: []*parser.Decorator{
				{
					ModuleSpecPath: modelDecoratorModuleSpec,
					ReferenceIdent: "Model",
					Arguments: []*parser.Argument{
						{Type: "Literal", End: firstArgEnd},
					},
				},
			},
		},
	}

	secondResult := &parser.ParserResult{
		Path:       "/virtual/modules/auth/service/multi_model.ts",
		RawContent: raw,
		ModelClassNode: &parser.Class{
			Decorators: []*parser.Decorator{
				{
					ModuleSpecPath: modelDecoratorModuleSpec,
					ReferenceIdent: "Model",
					Arguments: []*parser.Argument{
						{Type: "Literal", End: secondArgEnd},
						{Type: "ObjectLiteral", End: secondObjectEnd, ObjectProperties: []*parser.ObjectProperty{}},
					},
				},
			},
		},
	}

	if err := plugin.injectModelApplication([]*parser.ParserResult{firstResult, secondResult}); err != nil {
		t.Fatalf("injectModelApplication failed: %v", err)
	}

	if firstResult.Content != secondResult.Content {
		t.Fatalf("expected shared file content to stay consistent across parser results\nfirst:\n%s\nsecond:\n%s", firstResult.Content, secondResult.Content)
	}

	if !strings.Contains(firstResult.Content, "// emoji😀prefix") {
		t.Fatalf("expected multibyte prefix to remain unchanged, got:\n%s", firstResult.Content)
	}

	if !strings.Contains(firstResult.Content, "@Model('First', { application: 'auth' })") {
		t.Fatalf("expected first decorator application injection, got:\n%s", firstResult.Content)
	}

	if !strings.Contains(firstResult.Content, "@Model('Second', {") ||
		!strings.Contains(firstResult.Content, "readonly: true") ||
		!strings.Contains(firstResult.Content, "application: 'auth'") {
		t.Fatalf("expected second decorator object literal injection, got:\n%s", firstResult.Content)
	}

	if count := strings.Count(firstResult.Content, "application: 'auth'"); count != 2 {
		t.Fatalf("expected two application injections, got %d\n%s", count, firstResult.Content)
	}
}

func TestInjectModelApplication_MultipleEditsSameFile_ReverseOrder(t *testing.T) {
	testRuntimeScope := newPluginTestScope()
	plugin := newBackendPluginForInjectTest(t, testRuntimeScope)
	modelDecoratorModuleSpec, _ := meta.ModelDecoratorModuleSpec(testRuntimeScope)

	raw := "// emoji😀prefix\n@Model('First')\nexport class First {}\n\n@Model('Second', {\n  readonly: true\n})\nexport class Second {}\n"

	firstArgEnd := mustIndex(t, raw, "'First'") + len("'First'")
	secondArgEnd := mustIndex(t, raw, "'Second'") + len("'Second'")
	secondObjectEnd := mustIndex(t, raw, "})") + 1

	firstResult := &parser.ParserResult{
		Path:       "/virtual/modules/auth/service/multi_model.ts",
		RawContent: raw,
		ModelClassNode: &parser.Class{
			Decorators: []*parser.Decorator{
				{
					ModuleSpecPath: modelDecoratorModuleSpec,
					ReferenceIdent: "Model",
					Arguments: []*parser.Argument{
						{Type: "Literal", End: firstArgEnd},
					},
				},
			},
		},
	}

	secondResult := &parser.ParserResult{
		Path:       "/virtual/modules/auth/service/multi_model.ts",
		RawContent: raw,
		ModelClassNode: &parser.Class{
			Decorators: []*parser.Decorator{
				{
					ModuleSpecPath: modelDecoratorModuleSpec,
					ReferenceIdent: "Model",
					Arguments: []*parser.Argument{
						{Type: "Literal", End: secondArgEnd},
						{Type: "ObjectLiteral", End: secondObjectEnd, ObjectProperties: []*parser.ObjectProperty{}},
					},
				},
			},
		},
	}

	// Process in reverse order to ensure raw offset mapping is independent from parse result order.
	if err := plugin.injectModelApplication([]*parser.ParserResult{secondResult, firstResult}); err != nil {
		t.Fatalf("injectModelApplication failed: %v", err)
	}

	if firstResult.Content != secondResult.Content {
		t.Fatalf("expected shared file content to stay consistent across parser results\nfirst:\n%s\nsecond:\n%s", firstResult.Content, secondResult.Content)
	}

	if !strings.Contains(firstResult.Content, "@Model('First', { application: 'auth' })") {
		t.Fatalf("expected first decorator application injection, got:\n%s", firstResult.Content)
	}

	if !strings.Contains(firstResult.Content, "@Model('Second', {") ||
		!strings.Contains(firstResult.Content, "readonly: true") ||
		!strings.Contains(firstResult.Content, "application: 'auth'") {
		t.Fatalf("expected second decorator object literal injection, got:\n%s", firstResult.Content)
	}

	if count := strings.Count(firstResult.Content, "application: 'auth'"); count != 2 {
		t.Fatalf("expected two application injections, got %d\n%s", count, firstResult.Content)
	}
}

func TestInjectModelApplication_EmptyModulesPathStillProcessesModelDecorator(t *testing.T) {
	testRuntimeScope, db := newPluginSessionTestScope(t)
	migrateBackendPluginMetadata(t, db)
	testRuntimeScope.cfg.ModulesPath = ""
	plugin := newBackendPluginForInjectTest(t, testRuntimeScope)
	modelDecoratorModuleSpec, _ := meta.ModelDecoratorModuleSpec(testRuntimeScope)

	raw := "@Model('User')\nexport default class User {}\n"
	argEnd := mustIndex(t, raw, "'User'") + len("'User'")

	result := &parser.ParserResult{
		Path:       "/external/modules/user.ts",
		RawContent: raw,
		ModelClassNode: &parser.Class{
			Decorators: []*parser.Decorator{
				{
					ModuleSpecPath: modelDecoratorModuleSpec,
					ReferenceIdent: "Model",
					Arguments: []*parser.Argument{
						{Type: "Literal", End: argEnd},
					},
				},
			},
		},
	}

	if err := plugin.injectModelApplication([]*parser.ParserResult{result}); err == nil || !strings.Contains(err.Error(), "application not found for model path") {
		t.Fatalf("expected application resolution failure for empty modules path, got %v", err)
	}
}

func TestInjectModelApplication_DerivesExternalModuleNameFromNormalizedPathWithoutModelTable(t *testing.T) {
	testRuntimeScope, db := newPluginSessionTestScope(t)
	if err := db.AutoMigrate(&meta.IrModule{}); err != nil {
		t.Fatalf("automigrate modules failed: %v", err)
	}

	realRoot := filepath.Join(t.TempDir(), "real")
	realModulesPath := filepath.Join(realRoot, "modules")
	if err := os.MkdirAll(realModulesPath, 0o755); err != nil {
		t.Fatalf("mkdir real modules path: %v", err)
	}
	aliasRoot := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(realRoot, aliasRoot); err != nil {
		t.Skipf("symlink not supported in this environment: %v", err)
	}

	testRuntimeScope.cfg.ModulesPath = filepath.Join(aliasRoot, "modules")
	if err := db.Create(&meta.IrModule{Name: "crm", ApplicationStr: "crm"}).Error; err != nil {
		t.Fatalf("seed external module failed: %v", err)
	}

	plugin := newBackendPluginForInjectTest(t, testRuntimeScope)
	plugin.Module.Path = filepath.Join(realModulesPath, "auth")
	modelDecoratorModuleSpec, _ := meta.ModelDecoratorModuleSpec(testRuntimeScope)

	raw := "@Model('Lead')\nexport default class Lead {}\n"
	argEnd := mustIndex(t, raw, "'Lead'") + len("'Lead'")
	externalModelPath := filepath.Join(realModulesPath, "crm", "service", "lead.ts")
	if err := os.MkdirAll(filepath.Dir(externalModelPath), 0o755); err != nil {
		t.Fatalf("mkdir external model dir: %v", err)
	}
	if err := os.WriteFile(externalModelPath, []byte("export default class Lead {}\n"), 0o644); err != nil {
		t.Fatalf("write external model file: %v", err)
	}
	result := &parser.ParserResult{
		Path:       externalModelPath,
		RawContent: raw,
		ModelClassNode: &parser.Class{Decorators: []*parser.Decorator{{
			ModuleSpecPath: modelDecoratorModuleSpec,
			ReferenceIdent: "Model",
			Arguments:      []*parser.Argument{{Type: "Literal", End: argEnd}},
		}}},
	}

	if err := plugin.injectModelApplication([]*parser.ParserResult{result}); err != nil {
		t.Fatalf("injectModelApplication failed: %v", err)
	}
	if !strings.Contains(result.Content, "application: 'crm'") {
		t.Fatalf("expected external module application to be derived from module path, got:\n%s", result.Content)
	}
}

type fakeParser struct {
	parseFn func(pathAlias map[string]string, path string, content string) (*parser.ParserResult, error)
}

func (p fakeParser) Parse(pathAlias map[string]string, path string, content string) (*parser.ParserResult, error) {
	if p.parseFn != nil {
		return p.parseFn(pathAlias, path, content)
	}
	return &parser.ParserResult{Path: path, RawContent: content}, nil
}

func captureBackendOnLoad(t *testing.T, plugin api.Plugin, buildOptions *api.BuildOptions) func(api.OnLoadArgs) (api.OnLoadResult, error) {
	t.Helper()

	var onLoad func(api.OnLoadArgs) (api.OnLoadResult, error)
	plugin.Setup(api.PluginBuild{
		InitialOptions: buildOptions,
		OnLoad: func(options api.OnLoadOptions, callback func(api.OnLoadArgs) (api.OnLoadResult, error)) {
			onLoad = callback
		},
	})
	if onLoad == nil {
		t.Fatal("expected backend plugin to register an OnLoad callback")
	}
	return onLoad
}

func TestBackendPluginConstructorsAndParserResults(t *testing.T) {
	testRuntimeScope := newPluginTestScope()
	moduleRef := &meta.IrModule{Path: "/virtual/modules/auth", ApplicationStr: "auth"}
	customParser := fakeParser{}

	created, ok := NewBackendPlugin(testRuntimeScope, moduleRef, "service/index.ts", WithParser(customParser)).(*BackendPlugin)
	if !ok {
		t.Fatalf("expected *BackendPlugin, got %T", created)
	}
	if created.Env != testRuntimeScope || created.Module != moduleRef || created.EntryPoint != "service/index.ts" {
		t.Fatalf("unexpected plugin base state: %#v", created.BasePlugin)
	}
	if _, ok := created.Parser.(fakeParser); !ok {
		t.Fatalf("expected WithParser to override parser, got %T", created.Parser)
	}
	if created.ParserResultChan == nil || created.TsExports == nil || created.ParserResults == nil {
		t.Fatalf("expected NewBackendPlugin to initialize parser state, got %#v", created.BasePlugin)
	}

	entryImports := []string{"/virtual/.choysum/generated/service/auth/index.ts", "/virtual/.choysum/generated/service/base/index.ts"}
	created.SetEntryPointImports(entryImports)
	entryImports[0] = "/virtual/.choysum/generated/service/changed/index.ts"
	if len(created.EntryPointImports) != 2 || created.EntryPointImports[0] != "/virtual/.choysum/generated/service/auth/index.ts" || created.EntryPointImports[1] != "/virtual/.choysum/generated/service/base/index.ts" {
		t.Fatalf("expected SetEntryPointImports to clone imports, got %#v", created.EntryPointImports)
	}

	results := []*parser.ParserResult{}
	if err := created.SetParserResults(results); err != nil {
		t.Fatalf("SetParserResults() error = %v", err)
	}
	if len(created.ParserResults) != 0 {
		t.Fatalf("expected ParserResults to be replaced, got %#v", created.ParserResults)
	}

	pluginResults, err := created.GetParserResults()
	if err != nil {
		t.Fatalf("GetParserResults() error = %v", err)
	}
	if len(pluginResults) != 0 {
		t.Fatalf("expected empty parser results, got %#v", pluginResults)
	}
	if created.TsExports == nil {
		t.Fatal("expected HandleParserResults to keep TsExports initialized")
	}
}

func TestBackendPluginDefinePlugins(t *testing.T) {
	testRuntimeScope := newPluginTestScope()
	plugin := &BackendPlugin{BasePlugin: &esbplugins.BasePlugin{
		Env:              testRuntimeScope,
		Module:           &meta.IrModule{Path: "/virtual/modules/auth", ApplicationStr: "auth"},
		ParserResultChan: make(chan *parser.ParserResult),
		TsExports:        make(map[string]map[string]*parser.Export),
		ParserResults:    make([]*parser.ParserResult, 0),
	}}

	defined := plugin.DefinePlugins(testRuntimeScope, nil, plugin.Module)
	if len(defined) != 1 {
		t.Fatalf("expected one esbuild plugin, got %d", len(defined))
	}
	if defined[0].Name != "choysum-backend-inherit" {
		t.Fatalf("unexpected esbuild plugin name: %q", defined[0].Name)
	}

	configured := plugin.DefinePlugins(testRuntimeScope, nil, plugin.Module, func(esbplugins.EsbPlugin) {
		plugin.EntryPoint = "changed.ts"
	})
	if len(configured) != 1 || configured[0].Name != "choysum-backend-inherit" {
		t.Fatalf("unexpected configured plugin result: %#v", configured)
	}
	if plugin.EntryPoint != "changed.ts" {
		t.Fatalf("expected DefinePlugins options to mutate plugin, got entry point %q", plugin.EntryPoint)
	}

	var _ api.Plugin = configured[0]
	if WithParser(nil) == nil {
		t.Fatal("expected WithParser to return a non-nil option function")
	}
}

func TestBackendPluginDefinePlugins_BindsRuntimeState(t *testing.T) {
	baseScope := newPluginTestScope()
	runtimeScope := newPluginTestScope()
	baseModule := &meta.IrModule{Path: "/virtual/modules/auth", ApplicationStr: "auth"}
	runtimeModule := &meta.IrModule{Path: "/runtime/modules/crm", ApplicationStr: "crm"}

	plugin, ok := NewBackendPlugin(baseScope, baseModule, "service/index.ts").(*BackendPlugin)
	if !ok {
		t.Fatalf("expected *BackendPlugin, got %T", plugin)
	}

	var gotScope scope.Scope
	var gotModule *meta.IrModule
	plugin.parserFactory = func(runtimeScope scope.Scope, module *meta.IrModule) parser.Parser {
		gotScope = runtimeScope
		gotModule = module
		return fakeParser{}
	}

	defined := plugin.DefinePlugins(runtimeScope, nil, runtimeModule)
	if len(defined) != 1 {
		t.Fatalf("expected one esbuild plugin, got %d", len(defined))
	}
	if plugin.Env != runtimeScope {
		t.Fatalf("expected runtime scope to replace base scope")
	}
	if plugin.Module != runtimeModule {
		t.Fatalf("expected runtime module to replace base module")
	}
	if gotScope != runtimeScope {
		t.Fatalf("parser factory scope = %#v, want %#v", gotScope, runtimeScope)
	}
	if gotModule != runtimeModule {
		t.Fatalf("parser factory module = %#v, want %#v", gotModule, runtimeModule)
	}
	if _, ok := plugin.Parser.(fakeParser); !ok {
		t.Fatalf("expected parser factory result to be installed, got %T", plugin.Parser)
	}
	var _ api.Plugin = defined[0]
}

func TestBackendPluginDefinePluginsOnLoad_AppendsEntryPointImports(t *testing.T) {
	testRuntimeScope := newPluginTestScope()
	moduleDir := filepath.Join(t.TempDir(), "auth")
	entryPath := filepath.Join(moduleDir, "service", "index.ts")
	if err := os.MkdirAll(filepath.Dir(entryPath), 0o755); err != nil {
		t.Fatalf("mkdir entry dir: %v", err)
	}
	workspaceRoot := filepath.Dir(moduleDir)
	authImport := filepath.ToSlash(filepath.Clean(filepath.Join(workspaceRoot, ".choysum", "generated", "service", "auth", "index.ts")))
	baseImport := filepath.ToSlash(filepath.Clean(filepath.Join(workspaceRoot, ".choysum", "generated", "service", "base", "index.ts")))
	baseStmt := fmt.Sprintf("import '%s';\n", baseImport)
	if err := os.WriteFile(entryPath, []byte(baseStmt+"export const ready = true\n"), 0o644); err != nil {
		t.Fatalf("write entry file: %v", err)
	}

	plugin := &BackendPlugin{BasePlugin: &esbplugins.BasePlugin{
		Env:              testRuntimeScope,
		Module:           &meta.IrModule{Path: moduleDir, ApplicationStr: "auth"},
		EntryPoint:       "./service/index.ts",
		ParserResultChan: make(chan *parser.ParserResult, 1),
		TsExports:        make(map[string]map[string]*parser.Export),
		ParserResults:    make([]*parser.ParserResult, 0),
	}}
	plugin.SetEntryPointImports([]string{authImport, baseImport, authImport})
	plugin.Parser = fakeParser{parseFn: func(pathAlias map[string]string, gotPath string, content string) (*parser.ParserResult, error) {
		if gotPath != entryPath {
			t.Fatalf("unexpected parse path %q", gotPath)
		}
		if strings.Count(content, fmt.Sprintf("import '%s';", baseImport)) != 1 {
			t.Fatalf("expected existing base import to remain single, got content:\n%s", content)
		}
		if strings.Count(content, fmt.Sprintf("import '%s';", authImport)) != 1 {
			t.Fatalf("expected auth import to be appended exactly once, got content:\n%s", content)
		}
		return &parser.ParserResult{Path: gotPath, RawContent: content}, nil
	}}

	onLoad := captureBackendOnLoad(t, plugin.DefinePlugins(testRuntimeScope, nil, plugin.Module)[0], &api.BuildOptions{})
	result, err := onLoad(api.OnLoadArgs{Path: entryPath})
	if err != nil {
		t.Fatalf("onLoad returned error: %v", err)
	}
	if result.Contents == nil {
		t.Fatal("expected onLoad contents to be populated")
	}
	if strings.Count(*result.Contents, fmt.Sprintf("import '%s';", baseImport)) != 1 {
		t.Fatalf("expected base import once in output, got:\n%s", *result.Contents)
	}
	if strings.Count(*result.Contents, fmt.Sprintf("import '%s';", authImport)) != 1 {
		t.Fatalf("expected auth import once in output, got:\n%s", *result.Contents)
	}
}

func TestBackendPluginDefinePluginsOnLoad_AppendsEntryPointImports_WhenEntryPathResolvesSymlink(t *testing.T) {
	testRuntimeScope := newPluginTestScope()
	rootDir := t.TempDir()
	realDir := filepath.Join(rootDir, "real")
	linkDir := filepath.Join(rootDir, "link")
	if err := os.MkdirAll(realDir, 0o755); err != nil {
		t.Fatalf("mkdir real dir: %v", err)
	}
	if err := os.Symlink(realDir, linkDir); err != nil {
		t.Fatalf("create symlink: %v", err)
	}

	entryFromEntryPoint := filepath.Join(linkDir, "__choysum_bundles_entry.ts")
	entryFromOnLoad := filepath.Join(realDir, "__choysum_bundles_entry.ts")
	baseImport := filepath.ToSlash(filepath.Clean(filepath.Join(rootDir, ".choysum", "generated", "service", "base", "index.ts")))
	metaImport := filepath.ToSlash(filepath.Clean(filepath.Join(rootDir, ".choysum", "generated", "service", "meta", "index.ts")))
	if err := os.WriteFile(entryFromOnLoad, []byte(fmt.Sprintf("import '%s';\n", baseImport)), 0o644); err != nil {
		t.Fatalf("write entry file: %v", err)
	}

	plugin := &BackendPlugin{BasePlugin: &esbplugins.BasePlugin{
		Env:              testRuntimeScope,
		Module:           &meta.IrModule{Path: filepath.Join(rootDir, "module"), ApplicationStr: "bundles"},
		EntryPoint:       entryFromEntryPoint,
		ParserResultChan: make(chan *parser.ParserResult, 1),
		TsExports:        make(map[string]map[string]*parser.Export),
		ParserResults:    make([]*parser.ParserResult, 0),
	}}
	plugin.SetEntryPointImports([]string{baseImport, metaImport})
	plugin.Parser = fakeParser{parseFn: func(pathAlias map[string]string, gotPath string, content string) (*parser.ParserResult, error) {
		if gotPath != entryFromOnLoad {
			t.Fatalf("unexpected parse path %q", gotPath)
		}
		if strings.Count(content, fmt.Sprintf("import '%s';", baseImport)) != 1 {
			t.Fatalf("expected base import once, got content:\n%s", content)
		}
		if strings.Count(content, fmt.Sprintf("import '%s';", metaImport)) != 1 {
			t.Fatalf("expected meta import once, got content:\n%s", content)
		}
		return &parser.ParserResult{Path: gotPath, RawContent: content}, nil
	}}

	onLoad := captureBackendOnLoad(t, plugin.DefinePlugins(testRuntimeScope, nil, plugin.Module)[0], &api.BuildOptions{})
	result, err := onLoad(api.OnLoadArgs{Path: entryFromOnLoad})
	if err != nil {
		t.Fatalf("onLoad returned error: %v", err)
	}
	if result.Contents == nil {
		t.Fatal("expected onLoad contents to be populated")
	}
	if strings.Count(*result.Contents, fmt.Sprintf("import '%s';", baseImport)) != 1 {
		t.Fatalf("expected base import once in output, got:\n%s", *result.Contents)
	}
	if strings.Count(*result.Contents, fmt.Sprintf("import '%s';", metaImport)) != 1 {
		t.Fatalf("expected meta import once in output, got:\n%s", *result.Contents)
	}
}

func TestBackendPluginDefinePluginsOnLoad_UsesCachedContentAndPublishesResult(t *testing.T) {
	testRuntimeScope := newPluginTestScope()
	path := "/virtual/modules/auth/service/user.ts"
	plugin := &BackendPlugin{BasePlugin: &esbplugins.BasePlugin{
		Env:              testRuntimeScope,
		Module:           &meta.IrModule{Path: "/virtual/modules/auth", ApplicationStr: "auth"},
		ParserResultChan: make(chan *parser.ParserResult, 1),
		TsExports:        make(map[string]map[string]*parser.Export),
		ParserResults: []*parser.ParserResult{{
			Path:       path,
			RawContent: "const rawValue = 1\n",
			Content:    "const cachedValue = 2\n",
		}},
	}}

	plugin.Parser = fakeParser{parseFn: func(pathAlias map[string]string, gotPath string, content string) (*parser.ParserResult, error) {
		if gotPath != path {
			t.Fatalf("unexpected parse path %q", gotPath)
		}
		if got := pathAlias["@/*"]; !strings.HasSuffix(strings.ReplaceAll(got, "\\", "/"), "/src/*") {
			t.Fatalf("unexpected path aliases: %#v", pathAlias)
		}
		if content != "const cachedValue = 2\n" {
			t.Fatalf("expected cached content, got %q", content)
		}
		return &parser.ParserResult{Path: gotPath, RawContent: content}, nil
	}}

	onLoad := captureBackendOnLoad(t, plugin.DefinePlugins(testRuntimeScope, nil, plugin.Module)[0], &api.BuildOptions{
		TsconfigRaw:   `{"compilerOptions":{"paths":{"@/*":["src/*"]}}}`,
		AbsWorkingDir: "/virtual/modules/auth",
	})

	result, err := onLoad(api.OnLoadArgs{Path: path})
	if err != nil {
		t.Fatalf("onLoad returned error: %v", err)
	}
	if result.Loader != api.LoaderTS {
		t.Fatalf("loader = %v, want LoaderTS", result.Loader)
	}
	if result.Contents == nil || *result.Contents != "const cachedValue = 2\n" {
		t.Fatalf("unexpected contents: %#v", result.Contents)
	}

	select {
	case parsed := <-plugin.ParserResultChan:
		if parsed == nil || parsed.Path != path || parsed.RawContent != "const cachedValue = 2\n" {
			t.Fatalf("unexpected parser result: %#v", parsed)
		}
	case <-time.After(time.Second):
		t.Fatal("expected parser result to be published")
	}
}

func TestBackendPluginDefinePluginsOnLoad_ReadsFileAndNormalizesCRLF(t *testing.T) {
	testRuntimeScope := newPluginTestScope()
	path := filepath.Join(t.TempDir(), "service", "user.ts")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir test path: %v", err)
	}
	if err := os.WriteFile(path, []byte("const value = 1\r\nexport default value\r\n"), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	plugin := &BackendPlugin{BasePlugin: &esbplugins.BasePlugin{
		Env:              testRuntimeScope,
		Module:           &meta.IrModule{Path: filepath.Dir(filepath.Dir(path)), ApplicationStr: "auth"},
		ParserResultChan: make(chan *parser.ParserResult, 1),
		TsExports:        make(map[string]map[string]*parser.Export),
		ParserResults:    make([]*parser.ParserResult, 0),
	}}
	plugin.Parser = fakeParser{parseFn: func(pathAlias map[string]string, gotPath string, content string) (*parser.ParserResult, error) {
		if len(pathAlias) != 0 {
			t.Fatalf("expected empty path alias map, got %#v", pathAlias)
		}
		if gotPath != path {
			t.Fatalf("unexpected parse path %q", gotPath)
		}
		if strings.Contains(content, "\r") {
			t.Fatalf("expected CRLF to be normalized, got %q", content)
		}
		return &parser.ParserResult{Path: gotPath, RawContent: content}, nil
	}}

	onLoad := captureBackendOnLoad(t, plugin.DefinePlugins(testRuntimeScope, nil, plugin.Module)[0], &api.BuildOptions{})

	result, err := onLoad(api.OnLoadArgs{Path: path})
	if err != nil {
		t.Fatalf("onLoad returned error: %v", err)
	}
	if result.Contents == nil || *result.Contents != "const value = 1\nexport default value\n" {
		t.Fatalf("unexpected normalized contents: %#v", result.Contents)
	}
}

func TestBackendPluginDefinePluginsOnLoad_ErrorPaths(t *testing.T) {
	t.Run("invalid_tsconfig", func(t *testing.T) {
		testRuntimeScope := newPluginTestScope()
		plugin := &BackendPlugin{BasePlugin: &esbplugins.BasePlugin{
			Env:              testRuntimeScope,
			Module:           &meta.IrModule{Path: "/virtual/modules/auth", ApplicationStr: "auth"},
			ParserResultChan: make(chan *parser.ParserResult, 1),
			TsExports:        make(map[string]map[string]*parser.Export),
			ParserResults: []*parser.ParserResult{{
				Path:       "/virtual/modules/auth/service/user.ts",
				RawContent: "const value = 1\n",
			}},
		}}
		plugin.Parser = fakeParser{parseFn: func(pathAlias map[string]string, path string, content string) (*parser.ParserResult, error) {
			t.Fatal("parser should not be called when tsconfig parsing fails")
			return nil, nil
		}}

		onLoad := captureBackendOnLoad(t, plugin.DefinePlugins(testRuntimeScope, nil, plugin.Module)[0], &api.BuildOptions{TsconfigRaw: "{"})
		if _, err := onLoad(api.OnLoadArgs{Path: "/virtual/modules/auth/service/user.ts"}); err == nil {
			t.Fatal("expected invalid tsconfig error")
		}
	})

	t.Run("parser_error", func(t *testing.T) {
		testRuntimeScope := newPluginTestScope()
		plugin := &BackendPlugin{BasePlugin: &esbplugins.BasePlugin{
			Env:              testRuntimeScope,
			Module:           &meta.IrModule{Path: "/virtual/modules/auth", ApplicationStr: "auth"},
			ParserResultChan: make(chan *parser.ParserResult, 1),
			TsExports:        make(map[string]map[string]*parser.Export),
			ParserResults: []*parser.ParserResult{{
				Path:       "/virtual/modules/auth/service/user.ts",
				RawContent: "const value = 1\n",
			}},
		}}
		wantErr := errors.New("parse failed")
		plugin.Parser = fakeParser{parseFn: func(pathAlias map[string]string, path string, content string) (*parser.ParserResult, error) {
			return nil, wantErr
		}}

		onLoad := captureBackendOnLoad(t, plugin.DefinePlugins(testRuntimeScope, nil, plugin.Module)[0], &api.BuildOptions{})
		if _, err := onLoad(api.OnLoadArgs{Path: "/virtual/modules/auth/service/user.ts"}); !errors.Is(err, wantErr) {
			t.Fatalf("expected parser error %v, got %v", wantErr, err)
		}
	})
}

func TestInjectModelApplication_UsesExternalModuleApplication(t *testing.T) {
	testRuntimeScope, db := newPluginSessionTestScope(t)
	migrateBackendPluginMetadata(t, db)
	testRuntimeOpts := runtimeOptionsFromScope(testRuntimeScope)
	if err := db.Create(&meta.IrModule{Name: "crm", ApplicationStr: "crm-app", Path: filepath.Join(testRuntimeOpts.modulesPath, "crm")}).Error; err != nil {
		t.Fatalf("seed external module: %v", err)
	}

	plugin := newBackendPluginForInjectTest(t, testRuntimeScope)
	plugin.Module.Path = filepath.Join(testRuntimeOpts.modulesPath, "auth")

	modelDecoratorModuleSpec, _ := meta.ModelDecoratorModuleSpec(testRuntimeScope)
	raw := "@Model('Partner')\nexport default class Partner {}\n"
	argEnd := mustIndex(t, raw, "'Partner'") + len("'Partner'")
	result := &parser.ParserResult{
		Path:       filepath.Join(testRuntimeOpts.modulesPath, "crm", "service", "partner.ts"),
		RawContent: raw,
		ModelClassNode: &parser.Class{Decorators: []*parser.Decorator{{
			ModuleSpecPath: modelDecoratorModuleSpec,
			ReferenceIdent: "Model",
			Arguments:      []*parser.Argument{{Type: "Literal", End: argEnd}},
		}}},
	}

	if err := plugin.injectModelApplication([]*parser.ParserResult{result}); err != nil {
		t.Fatalf("injectModelApplication failed: %v", err)
	}
	if !strings.Contains(result.Content, "application: 'crm-app'") {
		t.Fatalf("expected external module application to be injected, got:\n%s", result.Content)
	}
}

func TestInjectModelApplication_ReturnsErrorWhenExternalApplicationMissing(t *testing.T) {
	testRuntimeScope, db := newPluginSessionTestScope(t)
	migrateBackendPluginMetadata(t, db)
	testRuntimeOpts := runtimeOptionsFromScope(testRuntimeScope)

	plugin := newBackendPluginForInjectTest(t, testRuntimeScope)
	plugin.Module.Path = filepath.Join(testRuntimeOpts.modulesPath, "auth")

	modelDecoratorModuleSpec, _ := meta.ModelDecoratorModuleSpec(testRuntimeScope)
	raw := "@Model('Partner')\nexport default class Partner {}\n"
	argEnd := mustIndex(t, raw, "'Partner'") + len("'Partner'")
	result := &parser.ParserResult{
		Path:       filepath.Join(testRuntimeOpts.modulesPath, "missing", "service", "partner.ts"),
		RawContent: raw,
		ModelClassNode: &parser.Class{Decorators: []*parser.Decorator{{
			ModuleSpecPath: modelDecoratorModuleSpec,
			ReferenceIdent: "Model",
			Arguments:      []*parser.Argument{{Type: "Literal", End: argEnd}},
		}}},
	}

	err := plugin.injectModelApplication([]*parser.ParserResult{result})
	if err == nil {
		t.Fatal("expected missing external application error")
	}
	if !strings.Contains(err.Error(), "application not found for model path") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInjectModelApplication_UsesExternalModelApplication(t *testing.T) {
	testRuntimeScope, db := newPluginSessionTestScope(t)
	migrateBackendPluginMetadata(t, db)
	testRuntimeOpts := runtimeOptionsFromScope(testRuntimeScope)

	app := &meta.IrApplication{Name: "crm-app"}
	if err := db.Create(app).Error; err != nil {
		t.Fatalf("create application: %v", err)
	}
	mod := &meta.IrModule{Name: "crm", ApplicationStr: "crm-app", ApplicationId: app.Id, Path: "/external/modules/crm"}
	if err := db.Create(mod).Error; err != nil {
		t.Fatalf("create module: %v", err)
	}
	model := &meta.IrModel{Name: "Partner", Path: "/external/models/partner.ts", ModuleId: mod.Id}
	if err := db.Create(model).Error; err != nil {
		t.Fatalf("create model: %v", err)
	}

	plugin := newBackendPluginForInjectTest(t, testRuntimeScope)
	plugin.Module.Path = filepath.Join(testRuntimeOpts.modulesPath, "auth")

	modelDecoratorModuleSpec, _ := meta.ModelDecoratorModuleSpec(testRuntimeScope)
	raw := "@Model('Partner')\nexport default class Partner {}\n"
	argEnd := mustIndex(t, raw, "'Partner'") + len("'Partner'")
	result := &parser.ParserResult{
		Path:       "/external/models/partner.ts",
		RawContent: raw,
		ModelClassNode: &parser.Class{Decorators: []*parser.Decorator{{
			ModuleSpecPath: modelDecoratorModuleSpec,
			ReferenceIdent: "Model",
			Arguments:      []*parser.Argument{{Type: "Literal", End: argEnd}},
		}}},
	}

	if err := plugin.injectModelApplication([]*parser.ParserResult{result}); err != nil {
		t.Fatalf("injectModelApplication failed: %v", err)
	}
	if !strings.Contains(result.Content, "application: 'crm-app'") {
		t.Fatalf("expected external model application to be injected, got:\n%s", result.Content)
	}
}

func TestInjectModelApplication_UsesRuntimeScopeFromDefinePlugins(t *testing.T) {
	baseScope, baseDB := newPluginSessionTestScope(t)
	runtimeScope, runtimeDB := newPluginSessionTestScope(t)
	migrateBackendPluginMetadata(t, baseDB)
	migrateBackendPluginMetadata(t, runtimeDB)
	runtimeOpts := runtimeOptionsFromScope(runtimeScope)

	plugin := newBackendPluginForInjectTest(t, baseScope)
	runtimeModule := &meta.IrModule{Path: filepath.Join(runtimeOpts.modulesPath, "auth"), ApplicationStr: "auth"}
	plugin.DefinePlugins(runtimeScope, nil, runtimeModule)
	runtimeModelDecoratorModuleSpec, _ := meta.ModelDecoratorModuleSpec(runtimeScope)
	plugin.TsExports = map[string]map[string]*parser.Export{
		runtimeModelDecoratorModuleSpec: {
			"Model": {
				ModuleSpecPath: runtimeModelDecoratorModuleSpec,
				ReferenceIdent: "Model",
			},
		},
	}

	if err := runtimeDB.Create(&meta.IrModule{Name: "crm", ApplicationStr: "crm-app", Path: filepath.Join(runtimeOpts.modulesPath, "crm")}).Error; err != nil {
		t.Fatalf("seed runtime external module: %v", err)
	}

	raw := "@Model('Partner')\nexport default class Partner {}\n"
	argEnd := mustIndex(t, raw, "'Partner'") + len("'Partner'")
	result := &parser.ParserResult{
		Path:       filepath.Join(runtimeOpts.modulesPath, "crm", "service", "partner.ts"),
		RawContent: raw,
		ModelClassNode: &parser.Class{Decorators: []*parser.Decorator{{
			ModuleSpecPath: runtimeModelDecoratorModuleSpec,
			ReferenceIdent: "Model",
			Arguments:      []*parser.Argument{{Type: "Literal", End: argEnd}},
		}}},
	}

	if err := plugin.injectModelApplication([]*parser.ParserResult{result}); err != nil {
		t.Fatalf("injectModelApplication failed after runtime scope bind: %v", err)
	}
	if !strings.Contains(result.Content, "application: 'crm-app'") {
		t.Fatalf("expected runtime scope application injection, got:\n%s", result.Content)
	}
}

func TestInjectModelApplication_InjectsIntoEmptyObjectLiteral(t *testing.T) {
	testRuntimeScope := newPluginTestScope()
	plugin := newBackendPluginForInjectTest(t, testRuntimeScope)
	modelDecoratorModuleSpec, _ := meta.ModelDecoratorModuleSpec(testRuntimeScope)

	raw := "@Model('User', {})\nexport default class User {}\n"
	objectEnd := mustIndex(t, raw, "}") + 1
	result := &parser.ParserResult{
		Path:       "/virtual/modules/auth/service/user.ts",
		RawContent: raw,
		ModelClassNode: &parser.Class{Decorators: []*parser.Decorator{{
			ModuleSpecPath: modelDecoratorModuleSpec,
			ReferenceIdent: "Model",
			Arguments: []*parser.Argument{
				{Type: "Literal", End: mustIndex(t, raw, "'User'") + len("'User'")},
				{Type: "ObjectLiteral", End: objectEnd, ObjectProperties: []*parser.ObjectProperty{}},
			},
		}}},
	}

	if err := plugin.injectModelApplication([]*parser.ParserResult{result}); err != nil {
		t.Fatalf("injectModelApplication failed: %v", err)
	}
	if !strings.Contains(result.Content, "@Model('User', {\napplication: 'auth'})") {
		t.Fatalf("expected injection into empty object literal without leading comma, got:\n%s", result.Content)
	}
}

func TestInjectModelApplication_SkipsUnsupportedDecorators(t *testing.T) {
	testRuntimeScope := newPluginTestScope()
	plugin := newBackendPluginForInjectTest(t, testRuntimeScope)
	modelDecoratorModuleSpec, _ := meta.ModelDecoratorModuleSpec(testRuntimeScope)

	noArgsRaw := "@Model()\nexport default class User {}\n"
	noArgsResult := &parser.ParserResult{
		Path:       "/virtual/modules/auth/service/no_args.ts",
		RawContent: noArgsRaw,
		ModelClassNode: &parser.Class{Decorators: []*parser.Decorator{{
			ModuleSpecPath: modelDecoratorModuleSpec,
			ReferenceIdent: "Model",
		}}},
	}

	nonModelRaw := "@Other('User')\nexport default class User {}\n"
	nonModelResult := &parser.ParserResult{
		Path:       "/virtual/modules/auth/service/no_model.ts",
		RawContent: nonModelRaw,
		ModelClassNode: &parser.Class{Decorators: []*parser.Decorator{{
			ModuleSpecPath: "@/other",
			ReferenceIdent: "Other",
		}}},
	}

	if err := plugin.injectModelApplication([]*parser.ParserResult{noArgsResult, nonModelResult, {Path: "/virtual/modules/auth/service/no_class.ts"}}); err != nil {
		t.Fatalf("expected unsupported decorators to be skipped, got %v", err)
	}
	if noArgsResult.Content != noArgsRaw {
		t.Fatalf("expected no-args model decorator to remain unchanged, got:\n%s", noArgsResult.Content)
	}
	if nonModelResult.Content != nonModelRaw {
		t.Fatalf("expected non-model decorator to remain unchanged, got:\n%s", nonModelResult.Content)
	}
}

func TestBackendPluginReplaceModuleSpecReferenceIdent(t *testing.T) {
	testRuntimeScope := newPluginTestScope()
	modelDecoratorModuleSpec, modelDecoratorReferenceIdent := meta.ModelDecoratorModuleSpec(testRuntimeScope)
	serviceDecoratorModuleSpec, serviceDecoratorReferenceIdent := meta.ServiceDecoratorModuleSpec(testRuntimeScope)
	fieldDecoratorModuleSpec, fieldDecoratorReferenceIdent := meta.FieldDecoratorModuleSpec(testRuntimeScope)
	baseModelModuleSpec, _ := meta.BaseModelModuleSpec(testRuntimeScope)

	plugin := &BackendPlugin{BasePlugin: &esbplugins.BasePlugin{
		Env:    testRuntimeScope,
		Module: &meta.IrModule{Path: "/virtual/modules/auth", ApplicationStr: "auth"},
		TsExports: map[string]map[string]*parser.Export{
			"@/base": {
				"BaseModel": {ModuleSpecPath: baseModelModuleSpec, ReferenceIdent: "default"},
			},
		},
	}}

	autoMigrate := true
	result := &parser.ParserResult{
		Path: "/virtual/modules/auth/service/models/partner.ts",
		Model: &meta.IrModel{
			Decorators: []*meta.IrDecorator{{
				Name:           "Model",
				ModuleSpecPath: modelDecoratorModuleSpec,
				ReferenceIdent: modelDecoratorReferenceIdent,
				Arguments: []*meta.IrArgument{
					{Value: `'Partner'`, Type: "Literal"},
					{Value: `{"tableName":"crm_partner","application":"crm","autoMigrate":true,"readonly":true}`, Type: "ObjectLiteral"},
				},
			}},
			Services: []*meta.IrService{
				{
					Name:                  "Create",
					AccessibilityModifier: "public",
					IsStatic:              true,
					Decorators:            []*meta.IrDecorator{{Name: "Service", ModuleSpecPath: serviceDecoratorModuleSpec, ReferenceIdent: serviceDecoratorReferenceIdent}},
				},
				{
					Name:                  "Search",
					AccessibilityModifier: "public",
					IsStatic:              true,
					Decorators:            []*meta.IrDecorator{{Name: "Other", ModuleSpecPath: "@/other", ReferenceIdent: "Other"}},
				},
				{
					Name:                  "helper",
					AccessibilityModifier: "public",
					IsStatic:              true,
					Decorators:            []*meta.IrDecorator{{Name: "Service", ModuleSpecPath: serviceDecoratorModuleSpec, ReferenceIdent: serviceDecoratorReferenceIdent}},
				},
			},
			Fields: []*meta.IrField{
				{
					Name:           "Name",
					ModuleSpecPath: "@/types",
					ReferenceIdent: "string",
					Decorators:     []*meta.IrDecorator{{Name: "Field", ModuleSpecPath: fieldDecoratorModuleSpec, ReferenceIdent: fieldDecoratorReferenceIdent}},
				},
				{
					Name:       "Ignored",
					Decorators: []*meta.IrDecorator{{Name: "Other", ModuleSpecPath: "@/other", ReferenceIdent: "Other"}},
				},
			},
			AutoMigrate: &autoMigrate,
		},
		ModelExtendsProperty: &parser.PropertyNode{ModuleSpecPath: "@/base", ReferenceIdent: "BaseModel"},
	}

	if err := plugin.replaceModuleSpecReferenceIdent([]*parser.ParserResult{result}); err != nil {
		t.Fatalf("replaceModuleSpecReferenceIdent() error = %v", err)
	}
	if result.Model.RawExtends != baseModelModuleSpec+".ts" || result.Model.Extends != result.Model.RawExtends {
		t.Fatalf("unexpected model extends: raw=%q extends=%q", result.Model.RawExtends, result.Model.Extends)
	}
	if result.Model.Name != "Partner" || result.Model.Application != "crm" || result.Model.ModelTable != "crm_partner" {
		t.Fatalf("unexpected model metadata: name=%q application=%q table=%q", result.Model.Name, result.Model.Application, result.Model.ModelTable)
	}
	if result.Model.AutoMigrate == nil || !*result.Model.AutoMigrate || !result.Model.Readonly || result.Model.Abstract {
		t.Fatalf("unexpected model flags after replace: autoMigrate=%v readonly=%v abstract=%v", result.Model.AutoMigrate, result.Model.Readonly, result.Model.Abstract)
	}
	if len(result.Model.Services) != 2 || result.Model.Services[0].Name != "Create" || result.Model.Services[1].Name != "Search" {
		t.Fatalf("expected only convention-matched services to remain, got %#v", result.Model.Services)
	}
	if len(result.Model.Fields) != 1 || result.Model.Fields[0].Name != "Name" {
		t.Fatalf("expected only field-decorated fields to remain, got %#v", result.Model.Fields)
	}

	badResult := &parser.ParserResult{
		Path:                 "/virtual/modules/auth/service/models/bad.ts",
		Model:                &meta.IrModel{},
		ModelExtendsProperty: &parser.PropertyNode{ModuleSpecPath: "@/base", ReferenceIdent: "NamedExport"},
	}
	if err := plugin.replaceModuleSpecReferenceIdent([]*parser.ParserResult{badResult}); err == nil || !strings.Contains(err.Error(), "model should extend default") {
		t.Fatalf("expected default-export validation error, got %v", err)
	}
}

func TestBackendPluginReplaceModuleSpecReferenceIdent_AdditionalPaths(t *testing.T) {
	testRuntimeScope := newPluginTestScope()
	modelDecoratorModuleSpec, modelDecoratorReferenceIdent := meta.ModelDecoratorModuleSpec(testRuntimeScope)
	serviceDecoratorModuleSpec, serviceDecoratorReferenceIdent := meta.ServiceDecoratorModuleSpec(testRuntimeScope)
	fieldDecoratorModuleSpec, fieldDecoratorReferenceIdent := meta.FieldDecoratorModuleSpec(testRuntimeScope)
	baseModelModuleSpec, _ := meta.BaseModelModuleSpec(testRuntimeScope)

	plugin := &BackendPlugin{BasePlugin: &esbplugins.BasePlugin{
		Env:    testRuntimeScope,
		Module: &meta.IrModule{Path: "/virtual/modules/auth", ApplicationStr: "auth"},
		TsExports: map[string]map[string]*parser.Export{
			"@/base": {
				"default": {ModuleSpecPath: baseModelModuleSpec, ReferenceIdent: "BaseModelClass"},
			},
			"@/service-namespace": {
				"*": {Wildcard: []*parser.Export{{ModuleSpecPath: "@/service-impl"}}},
			},
			"@/service-impl": {
				"Service":    {ModuleSpecPath: serviceDecoratorModuleSpec, ReferenceIdent: serviceDecoratorReferenceIdent},
				"ResultType": {ModuleSpecPath: "@/types/result", ReferenceIdent: "ResultType"},
			},
			"@/field-namespace": {
				"*": {Wildcard: []*parser.Export{{ModuleSpecPath: "@/field-impl"}}},
			},
			"@/field-impl": {
				"Field":        {ModuleSpecPath: fieldDecoratorModuleSpec, ReferenceIdent: fieldDecoratorReferenceIdent},
				"RelatedModel": {ModuleSpecPath: "/virtual/modules/auth/service/models/category", ReferenceIdent: "default"},
			},
		},
	}}

	result := &parser.ParserResult{
		Path: "/virtual/modules/auth/service/models/order.ts",
		Model: &meta.IrModel{
			Decorators: []*meta.IrDecorator{{
				Name:           "Model",
				ModuleSpecPath: modelDecoratorModuleSpec,
				ReferenceIdent: modelDecoratorReferenceIdent,
				Arguments:      []*meta.IrArgument{{Value: `'Order'`, Type: "Literal"}},
			}},
			Services: []*meta.IrService{
				{
					Name:                  "List",
					AccessibilityModifier: "public",
					IsStatic:              true,
					Decorators: []*meta.IrDecorator{{
						Name:           "Service",
						ModuleSpecPath: "@/service-namespace",
						ReferenceIdent: "Service",
					}},
					TypeParameters: []*meta.IrTypeParameter{{
						Name:           "TResult",
						ModuleSpecPath: "@/service-namespace",
						ReferenceIdent: "ResultType",
					}},
				},
			},
			Fields: []*meta.IrField{
				{
					Name:           "Category",
					ModuleSpecPath: "@/field-namespace",
					ReferenceIdent: "RelatedModel",
					Decorators: []*meta.IrDecorator{{
						Name:           "Field",
						ModuleSpecPath: "@/field-namespace",
						ReferenceIdent: "Field",
						Arguments: []*meta.IrArgument{{
							Type:           "Identifier",
							ModuleSpecPath: "@/field-namespace",
							ReferenceIdent: "RelatedModel",
						}},
					}},
				},
			},
		},
		ModelExtendsProperty: &parser.PropertyNode{ModuleSpecPath: "@/base", ReferenceIdent: "BaseModelClass"},
	}

	if err := plugin.replaceModuleSpecReferenceIdent([]*parser.ParserResult{result}); err != nil {
		t.Fatalf("replaceModuleSpecReferenceIdent() error = %v", err)
	}

	if result.Model.RawExtends != baseModelModuleSpec+".ts" || result.ModelExtendsProperty.ReferenceIdent != "default" {
		t.Fatalf("unexpected default export resolution: extends=%q property=%#v", result.Model.RawExtends, result.ModelExtendsProperty)
	}
	if result.Model.Name != "Order" || result.Model.Application != "auth" || result.Model.ModelTable != "auth_order" {
		t.Fatalf("unexpected derived model metadata: %#v", result.Model)
	}
	if len(result.Model.Services) != 1 {
		t.Fatalf("expected service to be preserved after wildcard resolution, got %#v", result.Model.Services)
	}
	service := result.Model.Services[0]
	if len(service.TypeParameters) != 1 || service.TypeParameters[0].ModuleSpecPath != "@/types/result" || service.TypeParameters[0].ReferenceIdent != "ResultType" {
		t.Fatalf("unexpected type parameter rewrite: %#v", service.TypeParameters)
	}
	if len(result.Model.Fields) != 1 {
		t.Fatalf("expected field to be preserved after wildcard resolution, got %#v", result.Model.Fields)
	}
	field := result.Model.Fields[0]
	if field.ModuleSpecPath != "/virtual/modules/auth/service/models/category" || field.ReferenceIdent != "default" {
		t.Fatalf("unexpected field module rewrite: %#v", field)
	}
	if len(field.Decorators) != 1 || field.Decorators[0].ModuleSpecPath != fieldDecoratorModuleSpec || field.Decorators[0].ReferenceIdent != fieldDecoratorReferenceIdent {
		t.Fatalf("unexpected field decorator rewrite: %#v", field.Decorators)
	}
	if len(field.Decorators[0].Arguments) != 1 || field.Decorators[0].Arguments[0].ModuleSpecPath != "/virtual/modules/auth/service/models/category" || field.Decorators[0].Arguments[0].ReferenceIdent != "default" {
		t.Fatalf("unexpected field argument rewrite: %#v", field.Decorators[0].Arguments)
	}
}

func TestBackendPluginReplaceModuleSpecReferenceIdent_SkipsExternalPaths(t *testing.T) {
	testRuntimeScope := newPluginTestScope()
	plugin := &BackendPlugin{BasePlugin: &esbplugins.BasePlugin{
		Env:    testRuntimeScope,
		Module: &meta.IrModule{Path: "/virtual/modules/auth", ApplicationStr: "auth"},
	}}

	result := &parser.ParserResult{
		Path: "/virtual/modules/external/service/models/partner.ts",
		Model: &meta.IrModel{
			Name:        "External",
			Application: "legacy",
		},
	}

	if err := plugin.replaceModuleSpecReferenceIdent([]*parser.ParserResult{result}); err != nil {
		t.Fatalf("expected external path to be skipped, got %v", err)
	}
	if result.Model.Application != "legacy" {
		t.Fatalf("expected external model to remain untouched, got %#v", result.Model)
	}
}

func TestBackendPluginSetFieldMeta(t *testing.T) {
	testRuntimeScope := newPluginTestScope()
	fieldDecoratorModuleSpec, fieldDecoratorReferenceIdent := meta.FieldDecoratorModuleSpec(testRuntimeScope)
	modelDecoratorModuleSpec, modelDecoratorReferenceIdent := meta.ModelDecoratorModuleSpec(testRuntimeScope)
	plugin := &BackendPlugin{BasePlugin: &esbplugins.BasePlugin{
		Env:    testRuntimeScope,
		Module: &meta.IrModule{Path: "/virtual/modules/auth", ApplicationStr: "auth"},
	}}

	targetModel := &meta.IrModel{
		Name: "Category",
		Path: "/virtual/modules/auth/service/models/category.ts",
		Decorators: []*meta.IrDecorator{{
			Name:           "Model",
			ModuleSpecPath: modelDecoratorModuleSpec,
			ReferenceIdent: modelDecoratorReferenceIdent,
			Arguments: []*meta.IrArgument{{
				Value: `{"parentField":"ParentId"}`,
				Type:  "ObjectLiteral",
			}},
		}},
	}

	goodResult := &parser.ParserResult{
		Path: "/virtual/modules/auth/service/models/partner.ts",
		Model: &meta.IrModel{
			Name: "Partner",
			Fields: []*meta.IrField{
				{
					Name: "Code",
					Decorators: []*meta.IrDecorator{{
						Name:           "Field",
						ModuleSpecPath: fieldDecoratorModuleSpec,
						ReferenceIdent: fieldDecoratorReferenceIdent,
						Arguments:      []*meta.IrArgument{{Value: `{"type":"char","selection":[{"value":"draft","label":"Draft"}],"select":{"size":16}}`, Type: "ObjectLiteral"}},
					}},
				},
				{
					Name: "Amount",
					Decorators: []*meta.IrDecorator{{
						Name:           "Field",
						ModuleSpecPath: fieldDecoratorModuleSpec,
						ReferenceIdent: fieldDecoratorReferenceIdent,
						Arguments:      []*meta.IrArgument{{Value: `{"type":"decimal","column":{"precision":12,"scale":4,"round":"half_up","scaleField":"currencyScale","unique":true}}`, Type: "ObjectLiteral"}},
					}},
				},
				{
					Name:           "Category",
					ModuleSpecPath: "/virtual/modules/auth/service/models/category",
					Decorators: []*meta.IrDecorator{{
						Name:           "Field",
						ModuleSpecPath: fieldDecoratorModuleSpec,
						ReferenceIdent: fieldDecoratorReferenceIdent,
						Arguments:      []*meta.IrArgument{{Value: `{"type":"ManyToOne"}`, Type: "ObjectLiteral"}},
					}},
				},
				{
					Name:           "Children",
					ModuleSpecPath: "/virtual/modules/auth/service/models/category",
					Decorators: []*meta.IrDecorator{{
						Name:           "Field",
						ModuleSpecPath: fieldDecoratorModuleSpec,
						ReferenceIdent: fieldDecoratorReferenceIdent,
						Arguments:      []*meta.IrArgument{{Value: `{"type":"OneToMany","relation":{"inverseField":"ParentId"}}`, Type: "ObjectLiteral"}},
					}},
				},
				{
					Name:           "Tags",
					ModuleSpecPath: "/virtual/modules/auth/service/models/category",
					Decorators: []*meta.IrDecorator{{
						Name:           "Field",
						ModuleSpecPath: fieldDecoratorModuleSpec,
						ReferenceIdent: fieldDecoratorReferenceIdent,
						Arguments:      []*meta.IrArgument{{Value: `{"type":"ManyToMany","relation":{"joinField":"PartnerId","inverseJoinField":"CategoryId","joinModel":"auth.PartnerCategory"}}`, Type: "ObjectLiteral"}},
					}},
				},
				{
					Name: "ExternalCategory",
					Decorators: []*meta.IrDecorator{{
						Name:           "Field",
						ModuleSpecPath: fieldDecoratorModuleSpec,
						ReferenceIdent: fieldDecoratorReferenceIdent,
						Arguments:      []*meta.IrArgument{{Value: `{"type":"ManyToOneRef","targetModel":"auth.Category"}`, Type: "ObjectLiteral"}},
					}},
				},
				{
					Name: "ExternalTags",
					Decorators: []*meta.IrDecorator{{
						Name:           "Field",
						ModuleSpecPath: fieldDecoratorModuleSpec,
						ReferenceIdent: fieldDecoratorReferenceIdent,
						Arguments:      []*meta.IrArgument{{Value: `{"type":"ManyToManyRef","targetModel":"auth.Tag"}`, Type: "ObjectLiteral"}},
					}},
				},
			},
		},
	}

	if err := plugin.setFieldMeta([]*parser.ParserResult{{Model: targetModel}, goodResult}); err != nil {
		t.Fatalf("setFieldMeta() error = %v", err)
	}
	codeField := goodResult.Model.Fields[0]
	if codeField.FieldType != "char" || codeField.Size != 16 || !codeField.IsReadonly || codeField.Selection == "" {
		t.Fatalf("unexpected char field metadata: %#v", codeField)
	}
	amountField := goodResult.Model.Fields[1]
	if amountField.FieldType != "decimal" || amountField.Precision != 12 || amountField.Scale != 4 || amountField.ScaleField != "currencyScale" || !amountField.Indexed {
		t.Fatalf("unexpected decimal field metadata: %#v", amountField)
	}
	if amountField.Round == nil || *amountField.Round != "ROUND_HALF_UP" {
		t.Fatalf("unexpected round metadata: %#v", amountField.Round)
	}
	categoryField := goodResult.Model.Fields[2]
	if categoryField.Relation != "ManyToOne" || categoryField.RelationModel != "auth.Category" || categoryField.RelationModelParentField != "" {
		t.Fatalf("unexpected many-to-one metadata: %#v", categoryField)
	}
	childrenField := goodResult.Model.Fields[3]
	if childrenField.Relation != "OneToMany" || childrenField.RelationModel != "auth.Category" || childrenField.RelationInverseField != "ParentId" || childrenField.RelationModelParentField != "" {
		t.Fatalf("unexpected one-to-many metadata: %#v", childrenField)
	}
	tagsField := goodResult.Model.Fields[4]
	if tagsField.Relation != "ManyToMany" || tagsField.RelationModel != "auth.Category" || tagsField.RelationJoinField != "PartnerId" || tagsField.RelationInverseJoinField != "CategoryId" || tagsField.RelationJoinModel != "auth.PartnerCategory" || tagsField.RelationModelParentField != "" {
		t.Fatalf("unexpected many-to-many metadata: %#v", tagsField)
	}
	externalCategoryField := goodResult.Model.Fields[5]
	if externalCategoryField.Relation != "ManyToOne" || externalCategoryField.RelationModel != "auth.Category" {
		t.Fatalf("unexpected many-to-one-ref metadata: %#v", externalCategoryField)
	}
	externalTagsField := goodResult.Model.Fields[6]
	if externalTagsField.Relation != "ManyToMany" || externalTagsField.RelationModel != "auth.Tag" {
		t.Fatalf("unexpected many-to-many-ref metadata: %#v", externalTagsField)
	}

	badResult := &parser.ParserResult{
		Path: "/virtual/modules/auth/service/models/bad.ts",
		Model: &meta.IrModel{
			Name: "BadModel",
			Fields: []*meta.IrField{
				{
					Name: "Code",
					Decorators: []*meta.IrDecorator{{
						Name:           "Field",
						ModuleSpecPath: fieldDecoratorModuleSpec,
						ReferenceIdent: fieldDecoratorReferenceIdent,
						Arguments: []*meta.IrArgument{{
							Value: `{"type":"char"}`,
							Type:  "ObjectLiteral",
						}},
					}},
				},
			},
		},
	}
	if err := plugin.setFieldMeta([]*parser.ParserResult{badResult}); err == nil || !strings.Contains(err.Error(), "missing required size") {
		t.Fatalf("expected missing size validation error, got %v", err)
	}
}

func TestBackendPluginSetFieldMeta_AdditionalPaths(t *testing.T) {
	testRuntimeScope := newPluginTestScope()
	fieldDecoratorModuleSpec, fieldDecoratorReferenceIdent := meta.FieldDecoratorModuleSpec(testRuntimeScope)
	plugin := &BackendPlugin{BasePlugin: &esbplugins.BasePlugin{
		Env:    testRuntimeScope,
		Module: &meta.IrModule{Path: "/virtual/modules/auth", ApplicationStr: "auth"},
	}}

	result := &parser.ParserResult{
		Path: "/virtual/modules/auth/service/models/advanced.ts",
		Model: &meta.IrModel{
			Name: "Advanced",
			Fields: []*meta.IrField{
				{
					Name: "VirtualCode",
					Decorators: []*meta.IrDecorator{{
						Name:           "Field",
						ModuleSpecPath: fieldDecoratorModuleSpec,
						ReferenceIdent: fieldDecoratorReferenceIdent,
						Arguments:      []*meta.IrArgument{{Value: `{"type":"char","select":{"size":8,"round":2,"scaleField":"currencyScale"}}`, Type: "ObjectLiteral"}},
					}},
				},
				{
					Name: "StoredCode",
					Decorators: []*meta.IrDecorator{{
						Name:           "Field",
						ModuleSpecPath: fieldDecoratorModuleSpec,
						ReferenceIdent: fieldDecoratorReferenceIdent,
						Arguments:      []*meta.IrArgument{{Value: `{"type":"char","column":{"size":32,"compute":"concat(code)","primaryKey":true,"uniqueIndex":"uq_advanced_code","notNull":true}}`, Type: "ObjectLiteral"}},
					}},
				},
				{
					Name: "Ignored",
					Decorators: []*meta.IrDecorator{{
						Name:           "Other",
						ModuleSpecPath: fieldDecoratorModuleSpec,
						ReferenceIdent: fieldDecoratorReferenceIdent,
						Arguments:      []*meta.IrArgument{{Value: `{"type":"char","column":{"size":16}}`, Type: "ObjectLiteral"}},
					}},
				},
				{
					Name: "NoArgs",
					Decorators: []*meta.IrDecorator{{
						Name:           "Field",
						ModuleSpecPath: fieldDecoratorModuleSpec,
						ReferenceIdent: fieldDecoratorReferenceIdent,
					}},
				},
			},
		},
	}

	if err := plugin.setFieldMeta([]*parser.ParserResult{result}); err != nil {
		t.Fatalf("setFieldMeta() error = %v", err)
	}

	virtualField := result.Model.Fields[0]
	if virtualField.FieldType != "char" || virtualField.Size != 8 || !virtualField.IsReadonly || virtualField.ScaleField != "currencyScale" {
		t.Fatalf("unexpected virtual field metadata: %#v", virtualField)
	}
	if virtualField.Round == nil || *virtualField.Round != "ROUND_CEIL" {
		t.Fatalf("unexpected select-derived round metadata: %#v", virtualField.Round)
	}

	storedField := result.Model.Fields[1]
	if storedField.FieldType != "char" || storedField.Size != 32 || !storedField.IsReadonly || !storedField.NotNull || !storedField.Indexed {
		t.Fatalf("unexpected stored field metadata: %#v", storedField)
	}

	ignoredField := result.Model.Fields[2]
	if ignoredField.FieldType != "" || ignoredField.Size != 0 || ignoredField.Indexed {
		t.Fatalf("expected non-Field decorator to be ignored, got %#v", ignoredField)
	}

	noArgsField := result.Model.Fields[3]
	if noArgsField.FieldType != "" || noArgsField.Size != 0 {
		t.Fatalf("expected zero-arg field decorator to be ignored, got %#v", noArgsField)
	}
}

func TestBackendPluginSetFieldMeta_UsesRuntimeScopeFromDefinePlugins(t *testing.T) {
	baseScope, baseDB := newPluginSessionTestScope(t)
	runtimeScope, runtimeDB := newPluginSessionTestScope(t)
	migrateBackendPluginMetadata(t, baseDB)
	migrateBackendPluginMetadata(t, runtimeDB)
	baseOpts := runtimeOptionsFromScope(baseScope)
	runtimeOpts := runtimeOptionsFromScope(runtimeScope)

	app := &meta.IrApplication{Name: "crm-app"}
	if err := runtimeDB.Create(app).Error; err != nil {
		t.Fatalf("create runtime application: %v", err)
	}
	mod := &meta.IrModule{Name: "crm", ApplicationStr: "crm-app", ApplicationId: app.Id, Path: filepath.Join(runtimeOpts.modulesPath, "crm")}
	if err := runtimeDB.Create(mod).Error; err != nil {
		t.Fatalf("create runtime module: %v", err)
	}
	if err := runtimeDB.Create(&meta.IrModel{Name: "Category", Path: "/external/models/category.ts", ModuleId: mod.Id}).Error; err != nil {
		t.Fatalf("create runtime model: %v", err)
	}

	runtimeModule := &meta.IrModule{Path: filepath.Join(runtimeOpts.modulesPath, "auth"), ApplicationStr: "auth"}
	plugin := &BackendPlugin{BasePlugin: &esbplugins.BasePlugin{
		Env:    baseScope,
		Module: &meta.IrModule{Path: filepath.Join(baseOpts.modulesPath, "auth"), ApplicationStr: "auth"},
	}}
	plugin.DefinePlugins(runtimeScope, nil, runtimeModule)

	fieldDecoratorModuleSpec, fieldDecoratorReferenceIdent := meta.FieldDecoratorModuleSpec(runtimeScope)
	result := &parser.ParserResult{
		Path: filepath.Join(runtimeModule.Path, "service", "models", "partner.ts"),
		Model: &meta.IrModel{
			Name: "Partner",
			Fields: []*meta.IrField{{
				Name:           "Category",
				ModuleSpecPath: "/external/models/category",
				Decorators: []*meta.IrDecorator{{
					Name:           "Field",
					ModuleSpecPath: fieldDecoratorModuleSpec,
					ReferenceIdent: fieldDecoratorReferenceIdent,
					Arguments:      []*meta.IrArgument{{Value: `{"type":"ManyToOne"}`, Type: "ObjectLiteral"}},
				}},
			}},
		},
	}

	if err := plugin.setFieldMeta([]*parser.ParserResult{result}); err != nil {
		t.Fatalf("setFieldMeta() error = %v", err)
	}
	field := result.Model.Fields[0]
	if field.Relation != "ManyToOne" || field.RelationModel != "crm-app.Category" {
		t.Fatalf("expected runtime scope relation model, got %#v", field)
	}
}

func TestBackendPluginSetFieldMeta_SkipAndErrorPaths(t *testing.T) {
	testRuntimeScope := newPluginTestScope()
	fieldDecoratorModuleSpec, fieldDecoratorReferenceIdent := meta.FieldDecoratorModuleSpec(testRuntimeScope)
	plugin := &BackendPlugin{BasePlugin: &esbplugins.BasePlugin{
		Env:    testRuntimeScope,
		Module: &meta.IrModule{Path: "/virtual/modules/auth", ApplicationStr: "auth"},
	}}

	t.Run("invalid_field_json", func(t *testing.T) {
		badResult := &parser.ParserResult{
			Path: "/virtual/modules/auth/service/models/bad_json.ts",
			Model: &meta.IrModel{
				Name: "BadJSON",
				Fields: []*meta.IrField{{
					Name: "Code",
					Decorators: []*meta.IrDecorator{{
						Name:           "Field",
						ModuleSpecPath: fieldDecoratorModuleSpec,
						ReferenceIdent: fieldDecoratorReferenceIdent,
						Arguments:      []*meta.IrArgument{{Value: `{`, Type: "ObjectLiteral"}},
					}},
				}},
			},
		}
		if err := plugin.setFieldMeta([]*parser.ParserResult{badResult}); err == nil || !strings.Contains(err.Error(), "parse Field options failed") {
			t.Fatalf("expected invalid field options error, got %v", err)
		}
	})

	t.Run("skip_validation_for_non_migrating_model", func(t *testing.T) {
		noMigrate := false
		result := &parser.ParserResult{
			Path: "/virtual/modules/auth/service/models/no_migrate.ts",
			Model: &meta.IrModel{
				Name:        "NoMigrate",
				AutoMigrate: &noMigrate,
				Fields: []*meta.IrField{{
					Name: "Code",
					Decorators: []*meta.IrDecorator{{
						Name:           "Field",
						ModuleSpecPath: fieldDecoratorModuleSpec,
						ReferenceIdent: fieldDecoratorReferenceIdent,
						Arguments:      []*meta.IrArgument{{Value: `{"type":"char"}`, Type: "ObjectLiteral"}},
					}},
				}},
			},
		}
		if err := plugin.setFieldMeta([]*parser.ParserResult{result}); err != nil {
			t.Fatalf("expected missing-size validation to be skipped, got %v", err)
		}
	})

	t.Run("skip_external_module_paths", func(t *testing.T) {
		result := &parser.ParserResult{
			Path: "/virtual/modules/external/service/models/partner.ts",
			Model: &meta.IrModel{
				Name: "External",
				Fields: []*meta.IrField{{
					Name: "Code",
					Decorators: []*meta.IrDecorator{{
						Name:           "Field",
						ModuleSpecPath: fieldDecoratorModuleSpec,
						ReferenceIdent: fieldDecoratorReferenceIdent,
						Arguments:      []*meta.IrArgument{{Value: `{"type":"char","column":{"size":16}}`, Type: "ObjectLiteral"}},
					}},
				}},
			},
		}
		if err := plugin.setFieldMeta([]*parser.ParserResult{result}); err != nil {
			t.Fatalf("expected external path to be skipped, got %v", err)
		}
		if result.Model.Fields[0].FieldType != "" {
			t.Fatalf("expected skipped external field metadata to remain empty, got %#v", result.Model.Fields[0])
		}
	})
}

func TestBackendPluginRuntimeOptionsHelpers(t *testing.T) {
	if got := newRuntimeOptions(scope.PathsRuntimeOptions{}, false); got.modulesPath != "" {
		t.Fatalf("newRuntimeOptions(no paths) = %#v, want empty modulesPath", got)
	}

	if got := runtimeOptionsFromScope(nil); got.modulesPath != "" {
		t.Fatalf("runtimeOptionsFromScope(nil) = %#v, want empty modulesPath", got)
	}

	runtimeScope := newPluginTestScope()
	if got := runtimeOptionsFromScope(runtimeScope); strings.TrimSpace(got.modulesPath) == "" {
		t.Fatalf("runtimeOptionsFromScope(scope) = %#v, want non-empty modulesPath", got)
	}

	if hasRuntimeOptions(runtimeOptions{modulesPath: "   "}) {
		t.Fatal("hasRuntimeOptions() = true for whitespace modulesPath")
	}
	if !hasRuntimeOptions(runtimeOptions{modulesPath: "/workspace/modules"}) {
		t.Fatal("hasRuntimeOptions() = false for non-empty modulesPath")
	}

	if err := (runtimeOptions{}).Validate(); err == nil || !strings.Contains(err.Error(), "modulesPath is required") {
		t.Fatalf("Validate(empty) error = %v, want modulesPath required", err)
	}
	if err := (runtimeOptions{modulesPath: "/workspace/modules"}).Validate(); err != nil {
		t.Fatalf("Validate(valid) error = %v", err)
	}

	if got := (*BackendPlugin)(nil).resolvedRuntimeOptions(); got.modulesPath != "" {
		t.Fatalf("nil plugin resolvedRuntimeOptions = %#v, want empty modulesPath", got)
	}

	pluginFromEnv := &BackendPlugin{BasePlugin: &esbplugins.BasePlugin{Env: runtimeScope}}
	if got := pluginFromEnv.resolvedRuntimeOptions(); strings.TrimSpace(got.modulesPath) == "" {
		t.Fatalf("resolvedRuntimeOptions(env fallback) = %#v, want non-empty modulesPath", got)
	}

	pluginPreferExplicit := &BackendPlugin{
		BasePlugin:     &esbplugins.BasePlugin{Env: runtimeScope},
		runtimeOptions: runtimeOptions{modulesPath: "/explicit/modules"},
	}
	if got := pluginPreferExplicit.resolvedRuntimeOptions(); got.modulesPath != "/explicit/modules" {
		t.Fatalf("resolvedRuntimeOptions(explicit) = %#v, want explicit modulesPath", got)
	}
}

func TestBackendPluginGetParserResults_PublishesAndInjects(t *testing.T) {
	testRuntimeScope, db := newPluginSessionTestScope(t)
	migrateBackendPluginMetadata(t, db)
	modelDecoratorModuleSpec, _ := meta.ModelDecoratorModuleSpec(testRuntimeScope)
	raw := "@Model('Partner')\nexport default class Partner {}\n"
	argEnd := mustIndex(t, raw, "'Partner'") + len("'Partner'")

	plugin := &BackendPlugin{BasePlugin: &esbplugins.BasePlugin{
		Env:              testRuntimeScope,
		Module:           &meta.IrModule{Path: "/virtual/modules/auth", ApplicationStr: "auth"},
		ParserResultChan: make(chan *parser.ParserResult, 2),
		TsExports:        make(map[string]map[string]*parser.Export),
		ParserResults:    make([]*parser.ParserResult, 0),
	}}
	plugin.ParserResultChan <- &parser.ParserResult{
		Path: modelDecoratorModuleSpec + ".ts",
		Exports: map[string]*parser.Export{
			"Model": {ModuleSpecPath: modelDecoratorModuleSpec, ReferenceIdent: "Model"},
		},
	}
	plugin.ParserResultChan <- &parser.ParserResult{
		Path:       "/virtual/modules/auth/service/models/partner.ts",
		RawContent: raw,
		Model:      &meta.IrModel{Name: "Partner", Path: "/virtual/modules/auth/service/models/partner.ts"},
		ModelClassNode: &parser.Class{Decorators: []*parser.Decorator{{
			Name:           "Model",
			ModuleSpecPath: modelDecoratorModuleSpec,
			ReferenceIdent: "Model",
			Arguments:      []*parser.Argument{{Type: "Literal", End: argEnd}},
		}}},
	}

	results, err := plugin.GetParserResults()
	if err != nil {
		t.Fatalf("GetParserResults() error = %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected two parser results, got %d", len(results))
	}

	if plugin.TsExports[modelDecoratorModuleSpec]["Model"] == nil {
		t.Fatalf("expected TsExports to include model decorator export, got %#v", plugin.TsExports)
	}

	var modelResult *parser.ParserResult
	for _, result := range results {
		if result.Path == "/virtual/modules/auth/service/models/partner.ts" {
			modelResult = result
			break
		}
	}
	if modelResult == nil {
		t.Fatalf("expected model parser result in %#v", results)
	}
	if !strings.Contains(modelResult.Content, "application: 'auth'") {
		t.Fatalf("expected GetParserResults to inject model application, got:\n%s", modelResult.Content)
	}
}

func TestBackendPluginGetParserResults_PropagatesSetFieldMetaError(t *testing.T) {
	testRuntimeScope := newPluginTestScope()
	fieldDecoratorModuleSpec, fieldDecoratorReferenceIdent := meta.FieldDecoratorModuleSpec(testRuntimeScope)
	plugin := &BackendPlugin{BasePlugin: &esbplugins.BasePlugin{
		Env:              testRuntimeScope,
		Module:           &meta.IrModule{Path: "/virtual/modules/auth", ApplicationStr: "auth"},
		ParserResultChan: make(chan *parser.ParserResult, 1),
		TsExports:        make(map[string]map[string]*parser.Export),
		ParserResults:    make([]*parser.ParserResult, 0),
	}}
	plugin.ParserResultChan <- &parser.ParserResult{
		Path: "/virtual/modules/auth/service/models/bad.ts",
		Model: &meta.IrModel{
			Name: "Bad",
			Fields: []*meta.IrField{{
				Name: "Code",
				Decorators: []*meta.IrDecorator{{
					Name:           "Field",
					ModuleSpecPath: fieldDecoratorModuleSpec,
					ReferenceIdent: fieldDecoratorReferenceIdent,
					Arguments:      []*meta.IrArgument{{Value: `{`, Type: "ObjectLiteral"}},
				}},
			}},
		},
	}

	if _, err := plugin.GetParserResults(); err == nil || !strings.Contains(err.Error(), "parse Field options failed") {
		t.Fatalf("expected GetParserResults to propagate setFieldMeta error, got %v", err)
	}
}
