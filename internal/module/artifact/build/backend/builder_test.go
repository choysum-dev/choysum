// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package backendbuilder

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/internal/esbplugins"
	"github.com/choysum-dev/choysum/internal/module/artifact/build/injectappmodel"
	modulegenerator "github.com/choysum-dev/choysum/internal/module/artifact/generate"
	module "github.com/choysum-dev/choysum/internal/module/artifact/result"
	modmeta "github.com/choysum-dev/choysum/internal/module/meta"
	"github.com/choysum-dev/choysum/internal/parser"
	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/jsexecutor"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/evanw/esbuild/pkg/api"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

)

type builderTestScope struct {
	ctx     context.Context
	cfg     *config.Config
	logger  *slog.Logger
	session *scope.Session
}

type builderTestTransaction struct {
	ctx     context.Context
	session *scope.Session
}

func withParserResults(result *module.BuildResult, parserResults ...*parser.ParserResult) *module.BuildResult {
	return module.WithParserResults(result, parserResults)
}

func parserResultsOf(result *module.BuildResult) []*parser.ParserResult {
	return module.ParserResults(result)
}

func (e *builderTestScope) Run(fn func(runtimeScope scope.Scope) error) error { return fn(e) }
func (e *builderTestScope) Transactor() scope.Transactor {
	return scopetest.NewPassthroughTransactor(e)
}
func (e *builderTestScope) Session() *scope.Session {
	if e == nil {
		return nil
	}
	if tx, ok := scope.TransactionFromContext(e.ctx); ok {
		if sess := tx.Session(); sess != nil {
			return sess
		}
	}
	if sess, ok := scope.SessionFromContext(e.ctx); ok {
		return sess
	}
	return e.session
}
func (e *builderTestScope) WithContext(ctx context.Context) scope.Scope {
	clone := *e
	clone.ctx = ctx
	return &clone
}
func (e *builderTestScope) Context() context.Context { return e.ctx }
func (e *builderTestScope) Logger() *slog.Logger     { return e.logger }
func (e *builderTestScope) Config() *config.Config   { return e.cfg }

func (e *builderTestScope) FactoryInput() scope.FactoryInput {
	return scopetest.FactoryInputFromConfig(e.Config())
}

func (tx *builderTestTransaction) Context() context.Context {
	if tx == nil {
		return nil
	}
	return tx.ctx
}

func (tx *builderTestTransaction) Session() *scope.Session {
	if tx == nil {
		return nil
	}
	return tx.session
}

func (tx *builderTestTransaction) Savepoint(string) error           { return nil }
func (tx *builderTestTransaction) RollbackToSavepoint(string) error { return nil }
func (tx *builderTestTransaction) ReleaseSavepoint(string) error    { return nil }

type stubEsbPlugin struct {
	name                string
	parserResults       []*parser.ParserResult
	entryImports        []string
	entryPoint          string
	virtualSources      map[string]string
	getParserResultsErr error
}

type fixedParser struct {
	result *parser.ParserResult
	err    error
}

func (p *stubEsbPlugin) DefinePlugins(_ scope.Scope, _ jsexecutor.ScriptExecutor, _ *meta.Module, options ...esbplugins.EsbPluginOptions) []api.Plugin {
	for _, opt := range options {
		if opt != nil {
			opt(p)
		}
	}
	return []api.Plugin{{Name: p.name, Setup: func(api.PluginBuild) {}}}
}

func (p *stubEsbPlugin) GetParserResults() ([]*parser.ParserResult, error) {
	if p.getParserResultsErr != nil {
		return nil, p.getParserResultsErr
	}
	return p.parserResults, nil
}

func (p *stubEsbPlugin) SetParserResults(parserResults []*parser.ParserResult) error {
	p.parserResults = parserResults
	return nil
}

func (p *stubEsbPlugin) SetEntryPointImports(imports []string) {
	p.entryImports = append([]string(nil), imports...)
}

func (p *stubEsbPlugin) SetEntryPoint(path string) {
	p.entryPoint = strings.TrimSpace(path)
}

func (p *stubEsbPlugin) RegisterVirtualSource(path string, contents string) {
	if p.virtualSources == nil {
		p.virtualSources = make(map[string]string)
	}
	p.virtualSources[path] = contents
}

func (p fixedParser) Parse(pathAlias map[string]string, path string, content string) (*parser.ParserResult, error) {
	return p.result, p.err
}

func normalizedAbsImportPath(path string) string {
	absPath := path
	if resolved, err := filepath.Abs(path); err == nil {
		absPath = resolved
	}
	return filepath.ToSlash(filepath.Clean(absPath))
}

func newBuilderTestScope() *builderTestScope {
	return &builderTestScope{
		ctx: context.Background(),
		cfg: &config.Config{
			ModulesPath:        "/virtual/modules",
			DistPath:           "/virtual/dist",
			DefaultChoysumPath: filepath.Join(os.TempDir(), "choysum-backendbuilder-default"),
			Compile: &config.CompileConfig{
				BundleMode:  string(config.BundleModeBundle),
				Minify:      true,
				TreeShaking: false,
				SourceMap:   true,
			},
			Auth: &config.AuthConfig{
				GrpcAuthentication: false,
				GrpcMethodAccess:   true,
				GrpcRecordRule:     false,
				GrpcCompanyFilter:  true,
				GrpcFieldRule:      false,
				AuthzDecisionLog:   "deny",
				AuthzDecisionAudit: true,
			},
			Task: &config.TaskConfig{
				Dispatch: &config.TaskDispatchConfig{DefaultMaxAttempts: 7},
			},
			BackendEnv: map[string]any{"CUSTOM_FLAG": "present"},
		},
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

func TestBuildOptionsSelectsPluginsAndInjectsBackendEnv(t *testing.T) {
	testRuntimeScope := newBuilderTestScope()
	builder := &ModuleBuilder{
		runtimeScope:   testRuntimeScope,
		module:         &meta.Module{ApplicationStr: "auth"},
		entryPoint:     "/virtual/modules/auth/service/index.ts",
		prebuildPlugin: &stubEsbPlugin{name: "prebuild"},
		buildPlugin:    &stubEsbPlugin{name: "build"},
		outFileName:    "bundle.js",
		globalName:     "AuthApp",
	}

	prebuildOpts := builder.buildOptions(true)
	if got, want := prebuildOpts.Outfile, filepath.Join("/virtual/dist", "apps", "auth", "bundle.js"); got != want {
		t.Fatalf("prebuild outfile = %q, want %q", got, want)
	}
	if len(prebuildOpts.Plugins) != 2 || prebuildOpts.Plugins[1].Name != "prebuild" {
		t.Fatalf("unexpected prebuild plugins: %#v", prebuildOpts.Plugins)
	}
	if prebuildOpts.Write {
		t.Fatal("expected prebuild buildOptions to disable direct writes")
	}
	if prebuildOpts.Sourcemap != api.SourceMapInline || prebuildOpts.TreeShaking != api.TreeShakingFalse {
		t.Fatalf("unexpected sourcemap/tree shaking config: sourcemap=%v treeShaking=%v", prebuildOpts.Sourcemap, prebuildOpts.TreeShaking)
	}
	if prebuildOpts.GlobalName != "AuthApp" || !prebuildOpts.MinifyWhitespace || !prebuildOpts.MinifyIdentifiers || !prebuildOpts.MinifySyntax {
		t.Fatalf("unexpected build flags: %#v", prebuildOpts)
	}

	var injected map[string]any
	if err := json.Unmarshal([]byte(prebuildOpts.Define["import.meta.env"]), &injected); err != nil {
		t.Fatalf("unmarshal define env: %v", err)
	}
	if injected["CUSTOM_FLAG"] != "present" || injected["CHOYSUM_AUTHZ_DECISION_LOG"] != "deny" {
		t.Fatalf("unexpected injected env payload: %#v", injected)
	}
	for key, want := range map[string]bool{
		"CHOYSUM_GRPC_AUTHENTICATION_ENABLED":  false,
		"CHOYSUM_GRPC_METHOD_ACCESS_ENABLED":   true,
		"CHOYSUM_GRPC_RECORD_RULE_ENABLED":     false,
		"CHOYSUM_GRPC_COMPANY_FILTER_ENABLED":  true,
		"CHOYSUM_GRPC_FIELD_RULE_ENABLED":      false,
		"CHOYSUM_AUTHZ_DECISION_AUDIT_ENABLED": true,
	} {
		got, ok := injected[key].(bool)
		if !ok || got != want {
			t.Fatalf("env[%q] = %#v, want %v", key, injected[key], want)
		}
	}
	if got := injected["CHOYSUM_TASK_DEFAULT_MAX_ATTEMPTS"]; got != float64(7) {
		t.Fatalf("unexpected task attempts value: %#v", got)
	}

	builder.distAppDirOverride = "/tmp/staged/auth"
	buildOpts := builder.buildOptions(false)
	if got, want := buildOpts.Outfile, filepath.Join("/tmp/staged/auth", "bundle.js"); got != want {
		t.Fatalf("build outfile = %q, want %q", got, want)
	}
	if len(buildOpts.Plugins) != 2 || buildOpts.Plugins[0].Name != "choysum-esm-resolver" || buildOpts.Plugins[1].Name != "build" {
		t.Fatalf("unexpected build plugins: %#v", buildOpts.Plugins)
	}

	builder.module.ApplicationStr = ""
	if buildOpts := builder.buildOptions(false); buildOpts.Write {
		t.Fatal("expected modules without application string to keep writes disabled")
	}
}

func TestEntryPointImportsCollectsInstalledServiceApplicationAliases(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:backendbuilder-entry-imports?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&meta.Module{}); err != nil {
		t.Fatalf("auto migrate modules: %v", err)
	}

	testRuntimeScope := newBuilderTestScope()
	testRuntimeScope.cfg.ModulesPath = filepath.Join(t.TempDir(), "modules")
	testRuntimeScope.session = &scope.Session{DB: db}
	expectedByApp := map[string]string{}
	for _, app := range []string{"crm", "hr"} {
		_, _, serviceDir, err := modulegenerator.WorkspaceGeneratedAPITargets(testRuntimeScope.cfg.ModulesPath, app, testRuntimeScope.cfg.DefaultChoysumPath)
		if err != nil {
			t.Fatalf("WorkspaceGeneratedAPITargets(%s) error = %v", app, err)
		}
		serviceIndex := filepath.Join(serviceDir, "index.ts")
		expectedByApp[app] = normalizedAbsImportPath(serviceIndex)
		if err := os.MkdirAll(filepath.Dir(serviceIndex), 0o755); err != nil {
			t.Fatalf("mkdir service index dir for %s: %v", app, err)
		}
		if err := os.WriteFile(serviceIndex, []byte("export * from './service';\n"), 0o644); err != nil {
			t.Fatalf("write service index for %s: %v", app, err)
		}
	}

	for _, mod := range []*meta.Module{
		{Name: "crm_mod", ApplicationStr: "crm", Status: meta.Installed, ServiceEntryPoint: "./service/index.ts"},
		{Name: "hr_mod", ApplicationStr: "hr", Status: meta.Installed, ServiceEntryPoint: "./service/index.ts"},
		{Name: "missing_entry", ApplicationStr: "missing", Status: meta.Installed},
		{Name: "draft", ApplicationStr: "draft", Status: meta.Uninstalled, ServiceEntryPoint: "./service/index.ts"},
	} {
		if err := db.Create(mod).Error; err != nil {
			t.Fatalf("create module %s: %v", mod.Name, err)
		}
	}

	b := &ModuleBuilder{
		runtimeScope: testRuntimeScope,
		module:       &meta.Module{Name: "auth", Path: filepath.Join(testRuntimeScope.cfg.ModulesPath, "auth")},
	}

	imports := b.entryPointImports()
	importSet := make(map[string]bool, len(imports))
	for _, item := range imports {
		importSet[item] = true
	}

	if !importSet[expectedByApp["crm"]] {
		t.Fatalf("expected workspace generated service import, got %#v", imports)
	}
	if !importSet[expectedByApp["hr"]] {
		t.Fatalf("expected installed app alias import, got %#v", imports)
	}
	_, _, missingDir, err := modulegenerator.WorkspaceGeneratedAPITargets(testRuntimeScope.cfg.ModulesPath, "missing", testRuntimeScope.cfg.DefaultChoysumPath)
	if err != nil {
		t.Fatalf("WorkspaceGeneratedAPITargets(missing) error = %v", err)
	}
	missingImportPath := normalizedAbsImportPath(filepath.Join(missingDir, "index.ts"))
	if importSet[missingImportPath] {
		t.Fatalf("expected module without service entrypoint to be skipped, got %#v", imports)
	}
	_, _, draftDir, err := modulegenerator.WorkspaceGeneratedAPITargets(testRuntimeScope.cfg.ModulesPath, "draft", testRuntimeScope.cfg.DefaultChoysumPath)
	if err != nil {
		t.Fatalf("WorkspaceGeneratedAPITargets(draft) error = %v", err)
	}
	draftImportPath := normalizedAbsImportPath(filepath.Join(draftDir, "index.ts"))
	if importSet[draftImportPath] {
		t.Fatalf("expected uninstalled module to be skipped, got %#v", imports)
	}
}

func TestBuildOptionsPassesEntryPointImportsToPlugins(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:backendbuilder-entry-options?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&meta.Module{}); err != nil {
		t.Fatalf("auto migrate modules: %v", err)
	}

	testRuntimeScope := newBuilderTestScope()
	testRuntimeScope.cfg.ModulesPath = filepath.Join(t.TempDir(), "modules")
	testRuntimeScope.session = &scope.Session{DB: db}
	_, _, crmServiceDir, err := modulegenerator.WorkspaceGeneratedAPITargets(testRuntimeScope.cfg.ModulesPath, "crm", testRuntimeScope.cfg.DefaultChoysumPath)
	if err != nil {
		t.Fatalf("WorkspaceGeneratedAPITargets(crm) error = %v", err)
	}
	crmServiceIndex := filepath.Join(crmServiceDir, "index.ts")
	expectedImport := normalizedAbsImportPath(crmServiceIndex)
	if err := os.MkdirAll(filepath.Dir(crmServiceIndex), 0o755); err != nil {
		t.Fatalf("mkdir crm service index dir: %v", err)
	}
	if err := os.WriteFile(crmServiceIndex, []byte("export * from './service';\n"), 0o644); err != nil {
		t.Fatalf("write crm service index: %v", err)
	}

	if err := db.Create(&meta.Module{Name: "crm_mod", ApplicationStr: "crm", Status: meta.Installed, ServiceEntryPoint: "./service/index.ts"}).Error; err != nil {
		t.Fatalf("create module: %v", err)
	}

	prebuildPlugin := &stubEsbPlugin{name: "prebuild"}
	buildPlugin := &stubEsbPlugin{name: "build"}
	builder := &ModuleBuilder{
		runtimeScope:   testRuntimeScope,
		module:         &meta.Module{Name: "auth", ApplicationStr: "auth"},
		entryPoint:     filepath.Join(testRuntimeScope.cfg.ModulesPath, "auth", "service", "index.ts"),
		prebuildPlugin: prebuildPlugin,
		buildPlugin:    buildPlugin,
	}

	_ = builder.buildOptions(true)
	if len(prebuildPlugin.entryImports) != 1 || prebuildPlugin.entryImports[0] != expectedImport {
		t.Fatalf("expected prebuild plugin to receive service entrypoint imports, got %#v", prebuildPlugin.entryImports)
	}

	_ = builder.buildOptions(false)
	if len(buildPlugin.entryImports) != 1 || buildPlugin.entryImports[0] != expectedImport {
		t.Fatalf("expected build plugin to receive service entrypoint imports, got %#v", buildPlugin.entryImports)
	}
}

func TestNormalizeImportHelpers(t *testing.T) {
	if got := normalizeModuleSpecPath("  /virtual/a.ts  "); got != "/virtual/a" {
		t.Fatalf("normalizeModuleSpecPath() = %q", got)
	}

	imports := map[string]*parser.Import{
		"Parent": {ReferenceIdent: "default", ModuleSpecPath: "/virtual/parent.ts"},
		"Named":  {ReferenceIdent: "Named", ModuleSpecPath: "/virtual/other"},
	}
	if ident, found, hasDefault := findDefaultImportIdentifierByModulePath(imports, "/virtual/parent"); ident != "Parent" || !found || !hasDefault {
		t.Fatalf("unexpected default import lookup result: ident=%q found=%v hasDefault=%v", ident, found, hasDefault)
	}
	if ident, found, hasDefault := findDefaultImportIdentifierByModulePath(imports, "/virtual/other.ts"); ident != "" || !found || hasDefault {
		t.Fatalf("unexpected named import lookup result: ident=%q found=%v hasDefault=%v", ident, found, hasDefault)
	}
	if ident, found, hasDefault := findDefaultImportIdentifierByModulePath(nil, "/virtual/missing"); ident != "" || found || hasDefault {
		t.Fatalf("unexpected empty import lookup result: ident=%q found=%v hasDefault=%v", ident, found, hasDefault)
	}

	if got := insertImportIntoImportRegion("export const x = 1\n", nil, "import Foo from './foo';"); !strings.HasPrefix(got, "import Foo from './foo';\n") {
		t.Fatalf("expected import to be inserted at top, got %q", got)
	}
	withLeadingNewline := insertImportIntoImportRegion("\nexport const x = 1\n", nil, "import Bar from './bar';")
	if !strings.HasPrefix(withLeadingNewline, "import Bar from './bar';\n") {
		t.Fatalf("expected leading newline content to receive import prefix, got %q", withLeadingNewline)
	}
}

func TestValidateInheritanceAndCircularDependencies(t *testing.T) {
	builder := &ModuleBuilder{}
	root := &meta.Model{Name: "Partner", Path: "/models/root"}
	child := &meta.Model{Name: "Partner", Path: "/models/child", Extends: "/models/root"}
	orphan := &meta.Model{Name: "Partner", Path: "/models/orphan", Extends: "/models/missing"}
	if err := builder.checkInheritanceChain([]*meta.Model{root, child, orphan}, map[string]*meta.Model{
		root.Path:  root,
		child.Path: child,
	}); err == nil || !strings.Contains(err.Error(), "not in the same inheritance component") {
		t.Fatalf("expected disconnected inheritance chain error, got %v", err)
	}

	cyclicA := &meta.Model{Name: "Partner", Path: "/models/a", Extends: "/models/b"}
	cyclicB := &meta.Model{Name: "Partner", Path: "/models/b", Extends: "/models/a"}
	if err := builder.checkCircularDependency(cyclicA, map[string]*meta.Model{
		cyclicA.Path: cyclicA,
		cyclicB.Path: cyclicB,
	}, map[string]bool{}); err == nil || !strings.Contains(err.Error(), "circular dependency") {
		t.Fatalf("expected circular dependency error, got %v", err)
	}

	buildResult := withParserResults(&module.BuildResult{},
		&parser.ParserResult{Model: cyclicA},
		&parser.ParserResult{Model: cyclicB},
	)
	if err := builder.validate(buildResult); err == nil || !strings.Contains(err.Error(), "circular dependency detected") {
		t.Fatalf("expected validate to surface circular dependency, got %v", err)
	}

	buildResult = withParserResults(&module.BuildResult{}, &parser.ParserResult{Model: root}, &parser.ParserResult{Model: child})
	if err := builder.validate(buildResult); err != nil {
		t.Fatalf("validate() unexpected error = %v", err)
	}
}

func TestGetNewExtendsAndUpdatePrebuildResult(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:backendbuilder-more?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(modmeta.CatalogEntities()...); err != nil {
		t.Fatalf("ensure dual store: %v", err)
	}
	seedMod := sql.NullString{String: "seed-extends", Valid: true}
	rows := []*meta.Model{
		{Name: "Partner", Path: "/models/base", Application: "auth", BaseModel: meta.BaseModel{Id: sql.NullString{String: "base", Valid: true}}, ModuleId: seedMod},
		{Name: "Partner", Path: "/models/latest", Application: "auth", BaseModel: meta.BaseModel{Id: sql.NullString{String: "latest", Valid: true}}, ModuleId: seedMod},
		// Newer same-name declaration in another application must not win.
		{Name: "Partner", Path: "/models/foreign-latest", Application: "crm", BaseModel: meta.BaseModel{Id: sql.NullString{String: "foreign", Valid: true}}, ModuleId: seedMod},
	}
	if _, err := modmeta.ReplaceModuleDeclarations(db, seedMod.String, rows); err != nil {
		t.Fatalf("seed models: %v", err)
	}

	testRuntimeScope := newBuilderTestScope()
	testRuntimeScope.session = &scope.Session{DB: db}
	builder := &ModuleBuilder{
		runtimeScope: testRuntimeScope,
		module:       &meta.Module{ApplicationStr: "auth"},
		tsPathAlias:  map[string]string{},
	}

	current := &meta.Model{Name: "Partner", Path: "/models/current", Application: "auth", Extends: "/models/base"}
	latest, err := builder.getNewExtends(current)
	if err != nil {
		t.Fatalf("getNewExtends() error = %v", err)
	}
	if latest == nil || latest.Path != "/models/latest" {
		t.Fatalf("unexpected latest extends model: %#v", latest)
	}

	rawContent := "export default class Partner extends BaseModel {}\n"
	extendsStart := strings.Index(rawContent, "extends BaseModel")
	if extendsStart < 0 {
		t.Fatal("failed to locate extends clause in test fixture")
	}
	builder.tsParser = fixedParser{result: &parser.ParserResult{
		ModelExtendsProperty: &parser.PropertyNode{Text: "extends BaseModel", Start: extendsStart, End: extendsStart + len("extends BaseModel")},
		Imports:              map[string]*parser.Import{},
	}}
	buildResult := withParserResults(&module.BuildResult{}, &parser.ParserResult{
		Path:       "/models/current.ts",
		RawContent: rawContent,
		Model: &meta.Model{
			Name:        "Partner",
			Path:        "/models/current",
			Application: "auth",
			RawExtends:  "/models/base",
			Extends:     "/models/base",
		},
	})
	if err := builder.updatePrebuildResult(buildResult); err != nil {
		t.Fatalf("updatePrebuildResult() error = %v", err)
	}
	updated := parserResultsOf(buildResult)[0]
	if updated.Model.Extends != "/models/latest" {
		t.Fatalf("expected model extends to be rewritten to latest path, got %q", updated.Model.Extends)
	}
	if !strings.Contains(updated.Content, "from '/models/latest';") || !strings.Contains(updated.Content, "extends model_") {
		t.Fatalf("expected updated content to import latest model, got %q", updated.Content)
	}
}

func TestRefreshModelExtendsPropertyErrors(t *testing.T) {
	builder := &ModuleBuilder{
		runtimeScope: newBuilderTestScope(),
		module:       &meta.Module{ApplicationStr: "auth"},
		tsPathAlias:  map[string]string{},
	}
	parseResult := &parser.ParserResult{Path: "/models/partner.ts", Model: &meta.Model{Name: "Partner"}, Content: "export default class Partner extends BaseModel {}"}

	builder.tsParser = fixedParser{result: nil}
	if err := builder.refreshModelExtendsProperty(parseResult); err == nil || !strings.Contains(err.Error(), "returned nil result") {
		t.Fatalf("expected nil parser result error, got %v", err)
	}

	builder.tsParser = fixedParser{result: &parser.ParserResult{Imports: map[string]*parser.Import{}}}
	if err := builder.refreshModelExtendsProperty(parseResult); err == nil || !strings.Contains(err.Error(), "model extends property missing") {
		t.Fatalf("expected missing extends property error, got %v", err)
	}
}

func TestPersistHelpersAndBuild(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:backendbuilder-persist?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&meta.Application{}, &meta.Module{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	if err := db.AutoMigrate(modmeta.CatalogEntities()...); err != nil {
		t.Fatalf("ensure dual store: %v", err)
	}

	testRuntimeScope := newBuilderTestScope()
	testRuntimeScope.session = &scope.Session{DB: db}
	builder := &ModuleBuilder{runtimeScope: testRuntimeScope, module: &meta.Module{Name: "base", Path: "/virtual/modules/base"}}

	result, err := builder.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if result == nil || result.Module == nil || !result.Module.Id.Valid {
		t.Fatalf("expected Build to persist module metadata, got %#v", result)
	}

	seedModel := &meta.Model{BaseModel: meta.BaseModel{Id: sql.NullString{String: "stale", Valid: true}}, Name: "Stale", Path: "/models/stale", Application: "partner", ModuleId: sql.NullString{String: "module-1", Valid: true}}
	if err := db.Create(seedModel).Error; err != nil {
		t.Fatalf("seed stale model: %v", err)
	}
	if _, err := modmeta.ReplaceModuleDeclarations(db, "module-1", []*meta.Model{{
		BaseModel:   meta.BaseModel{Id: sql.NullString{String: "prev-raw", Valid: true}},
		Name:        "LegacyRaw",
		Path:        "/models/legacy-raw",
		Application: "partner",
		ModuleId:    sql.NullString{String: "module-1", Valid: true},
	}}); err != nil {
		t.Fatalf("seed previous raw model: %v", err)
	}
	models := []*meta.Model{
		{Name: "PartnerOld", Path: "/models/partner", Application: "partner"},
		{Name: "Partner", Path: "/models/partner", Application: "partner"},
		nil,
		{Name: "Order", Path: "/models/order", Application: "partner"},
	}
	if err := builder.persistModuleModels("module-1", models); err != nil {
		t.Fatalf("persistModuleModels() error = %v", err)
	}
	persisted, err := modmeta.ListDeclarations(db, modmeta.DeclarationQuery{ModuleID: "module-1"})
	if err != nil {
		t.Fatalf("query persisted raw models: %v", err)
	}
	sort.Slice(persisted, func(i, j int) bool { return persisted[i].Path < persisted[j].Path })
	if len(persisted) != 2 || persisted[0].Path != "/models/order" || persisted[1].Name != "Partner" {
		t.Fatalf("unexpected persisted raw models: %#v", persisted)
	}
	var effective []*meta.Model
	if err := db.Where("application = ?", "partner").Order("path ASC").Find(&effective).Error; err != nil {
		t.Fatalf("query effective models: %v", err)
	}
	if len(effective) != 2 || effective[0].Path != "/models/order" || effective[1].Path != "/models/partner" || effective[1].Name != "Partner" {
		t.Fatalf("unexpected effective models: %#v", effective)
	}

	if _, err := modmeta.ReplaceModuleDeclarations(db, "hist-old", []*meta.Model{{
		BaseModel: meta.BaseModel{Id: sql.NullString{String: "aaa", Valid: true}},
		Name:      "Partner",
		Path:      "/models/history",
		ModuleId:  sql.NullString{String: "hist-old", Valid: true},
	}}); err != nil {
		t.Fatalf("seed older history model: %v", err)
	}
	if _, err := modmeta.ReplaceModuleDeclarations(db, "hist-new", []*meta.Model{{
		BaseModel: meta.BaseModel{Id: sql.NullString{String: "zzz", Valid: true}},
		Name:      "Partner",
		Path:      "/models/history",
		ModuleId:  sql.NullString{String: "hist-new", Valid: true},
		Fields: []*meta.Field{{
			BaseModel: meta.BaseModel{Id: sql.NullString{String: "field1", Valid: true}},
			Name:      "Name",
			Decorators: []*meta.Decorator{{
				BaseModel: meta.BaseModel{Id: sql.NullString{String: "fielddec", Valid: true}},
				Name:      "Field",
				Arguments: []*meta.Argument{{
					BaseModel: meta.BaseModel{Id: sql.NullString{String: "arg1", Valid: true}},
					Type:      "Literal",
					Value:     "'name'",
				}},
			}},
		}},
		Services: []*meta.Service{{
			BaseModel: meta.BaseModel{Id: sql.NullString{String: "svc1", Valid: true}},
			Name:      "List",
			Decorators: []*meta.Decorator{{
				BaseModel: meta.BaseModel{Id: sql.NullString{String: "svcdec", Valid: true}},
				Name:      "Service",
				Arguments: []*meta.Argument{{
					BaseModel: meta.BaseModel{Id: sql.NullString{String: "arg2", Valid: true}},
					Type:      "Literal",
					Value:     "'list'",
				}},
			}},
			TypeParameters: []*meta.TypeParameter{{
				BaseModel: meta.BaseModel{Id: sql.NullString{String: "tp1", Valid: true}},
				Name:      "T",
			}},
			Parameters: []*meta.Parameter{
				{BaseModel: meta.BaseModel{Id: sql.NullString{String: "param1", Valid: true}}, Name: "this"},
				{BaseModel: meta.BaseModel{Id: sql.NullString{String: "param2", Valid: true}}, Name: "query"},
			},
		}},
	}}); err != nil {
		t.Fatalf("seed latest history model: %v", err)
	}

	decls, err := modmeta.ListDeclarations(db, modmeta.DeclarationQuery{Path: "/models/history", PreloadTree: true})
	if err != nil {
		t.Fatalf("ListDeclarations() error = %v", err)
	}
	if len(decls) == 0 || decls[0] == nil {
		t.Fatalf("expected tip declaration for /models/history")
	}
	loaded := decls[0]
	if loaded.Id.String != "zzz" || len(loaded.Fields) != 1 || len(loaded.Fields[0].Decorators) != 1 || len(loaded.Fields[0].Decorators[0].Arguments) != 1 {
		t.Fatalf("unexpected loaded model fields: %#v", loaded)
	}
	if len(loaded.Services) != 1 || len(loaded.Services[0].Decorators) != 1 || len(loaded.Services[0].TypeParameters) != 1 || len(loaded.Services[0].Parameters) != 1 || loaded.Services[0].Parameters[0].Name != "query" {
		t.Fatalf("unexpected loaded model services: %#v", loaded.Services)
	}

	missing, err := modmeta.ListDeclarations(db, modmeta.DeclarationQuery{Path: "/models/does-not-exist", PreloadTree: true})
	if err != nil {
		t.Fatalf("missing path error = %v", err)
	}
	if len(missing) != 0 {
		t.Fatalf("expected empty for missing path, got %#v", missing)
	}
	if err := db.Migrator().DropTable("meta_raw_model"); err != nil {
		t.Fatalf("drop meta_raw_model: %v", err)
	}
	if _, err := modmeta.ListDeclarations(db, modmeta.DeclarationQuery{Path: "/models/history", PreloadTree: true}); err == nil || !strings.Contains(err.Error(), "list declarations") {
		t.Fatalf("expected list error after drop, got %v", err)
	}
}

func TestPathWithinModuleRoot_ResolvesSymlinkAliases(t *testing.T) {
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
	if !pathWithinModuleRoot(insideRealPath, moduleAliasRoot) {
		t.Fatalf("expected real path %q to be within alias module root %q", insideRealPath, moduleAliasRoot)
	}
	if !pathWithinModuleRoot(insideAliasPath, moduleAliasRoot) {
		t.Fatalf("expected alias path %q to be within alias module root %q", insideAliasPath, moduleAliasRoot)
	}

	outsideRealPath := filepath.Join(realRoot, "modules", "auth", "service", "models", "user.ts")
	if err := os.MkdirAll(filepath.Dir(outsideRealPath), 0o755); err != nil {
		t.Fatalf("mkdir outside real path: %v", err)
	}
	if err := os.WriteFile(outsideRealPath, []byte("export {};\n"), 0o644); err != nil {
		t.Fatalf("write outside real file: %v", err)
	}
	if pathWithinModuleRoot(outsideRealPath, moduleAliasRoot) {
		t.Fatalf("expected outside path %q not to be within alias module root %q", outsideRealPath, moduleAliasRoot)
	}
}

func TestGetTsParserAndPathAliasParsesTsconfig(t *testing.T) {
	modulesDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(modulesDir, "tsconfig.json"), []byte(`{"compilerOptions":{"baseUrl":".","paths":{"@/*":["./*"]}}}`), 0o644); err != nil {
		t.Fatalf("write tsconfig: %v", err)
	}
	builder := &ModuleBuilder{
		runtimeScope: &builderTestScope{
			ctx:    context.Background(),
			cfg:    &config.Config{ModulesPath: modulesDir},
			logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		},
		module:   &meta.Module{ApplicationStr: "auth"},
		tsParser: fixedParser{},
	}

	parsed, alias, err := builder.getTsParserAndPathAlias()
	if err != nil {
		t.Fatalf("getTsParserAndPathAlias() error = %v", err)
	}
	if parsed != builder.tsParser || alias["@/*"] != filepath.Join(modulesDir, "*") {
		t.Fatalf("unexpected parser/path alias result: parser=%T alias=%#v", parsed, alias)
	}
}

func TestNewModuleBuilderOptionsAndBundleToDirCtx(t *testing.T) {
	testRuntimeScope := newBuilderTestScope()
	moduleRef := &meta.Module{ApplicationStr: "auth"}
	prebuildPlugin := &stubEsbPlugin{name: "prebuild"}
	buildPlugin := &stubEsbPlugin{name: "build"}

	configured, ok := NewModuleBuilder(testRuntimeScope, nil, moduleRef, "service/index.ts",
		WithPrebuildPlugin(prebuildPlugin),
		WithBuildPlugin(buildPlugin),
		WithPublishDist(false),
		WithOutFileName("bundle.js"),
		WithGlobalName("AuthApp"),
		WithInjectRegistry(injectappmodel.NewRegistryWithDefaults()),
	).(*ModuleBuilder)
	if !ok {
		t.Fatalf("expected *ModuleBuilder, got %T", configured)
	}
	configuredPrebuild, ok := configured.prebuildPlugin.(*stubEsbPlugin)
	if !ok {
		t.Fatalf("expected configured prebuild plugin concrete type, got %T", configured.prebuildPlugin)
	}
	configuredBuild, ok := configured.buildPlugin.(*stubEsbPlugin)
	if !ok {
		t.Fatalf("expected configured build plugin concrete type, got %T", configured.buildPlugin)
	}
	if configuredPrebuild != prebuildPlugin || configuredBuild != buildPlugin || configured.publishDist || configured.outFileName != "bundle.js" || configured.globalName != "AuthApp" || configured.injectRegistry == nil {
		t.Fatalf("unexpected configured builder: %#v", configured)
	}

	defaults, ok := NewModuleBuilder(testRuntimeScope, nil, moduleRef, "service/index.ts").(*ModuleBuilder)
	if !ok {
		t.Fatalf("expected *ModuleBuilder, got %T", defaults)
	}
	if defaults.prebuildPlugin == nil || defaults.buildPlugin == nil || !defaults.publishDist || defaults.outFileName != "index.js" || defaults.globalName != "auth" {
		t.Fatalf("unexpected default builder configuration: %#v", defaults)
	}

	bundleBuilder := &ModuleBuilder{module: &meta.Module{ApplicationStr: "auth"}}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := bundleBuilder.BundleToDirCtx(canceled, "/tmp/staged/auth"); err == nil {
		t.Fatal("expected canceled context to abort BundleToDirCtx")
	}
	if bundleBuilder.distAppDirOverride != "" {
		t.Fatalf("expected canceled BundleToDirCtx to leave override unchanged, got %q", bundleBuilder.distAppDirOverride)
	}

	result, err := bundleBuilder.BundleToDirCtx(context.TODO(), "/tmp/staged/auth")
	if err != nil {
		t.Fatalf("BundleToDirCtx(nil) error = %v", err)
	}
	if result == nil || bundleBuilder.distAppDirOverride != "" {
		t.Fatalf("expected BundleToDirCtx to restore override, result=%#v override=%q", result, bundleBuilder.distAppDirOverride)
	}
}

func TestBundleToDirCtx_UsesContextSessionForRuntimeState(t *testing.T) {
	baseDB, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "backendbuilder_base.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open base sqlite: %v", err)
	}
	runtimeDB, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "backendbuilder_runtime.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open runtime sqlite: %v", err)
	}
	if err := baseDB.AutoMigrate(&meta.Module{}); err != nil {
		t.Fatalf("migrate base modules: %v", err)
	}
	if err := runtimeDB.AutoMigrate(&meta.Module{}); err != nil {
		t.Fatalf("migrate runtime modules: %v", err)
	}

	testRuntimeScope := newBuilderTestScope()
	testRuntimeScope.session = &scope.Session{DB: baseDB}
	testRuntimeScope.cfg.ModulesPath = filepath.Join(t.TempDir(), "modules")
	testRuntimeScope.cfg.DistPath = filepath.Join(t.TempDir(), "dist")
	testRuntimeScope.cfg.DefaultChoysumPath = filepath.Join(t.TempDir(), ".choysum")
	if err := os.MkdirAll(testRuntimeScope.cfg.ModulesPath, 0o755); err != nil {
		t.Fatalf("mkdir modules path: %v", err)
	}
	if err := os.WriteFile(filepath.Join(testRuntimeScope.cfg.ModulesPath, "tsconfig.json"), []byte(`{"compilerOptions":{"baseUrl":".","paths":{"@/*":["./*"]}}}`), 0o644); err != nil {
		t.Fatalf("write tsconfig: %v", err)
	}
	modulePath := filepath.Join(testRuntimeScope.cfg.ModulesPath, "auth")
	entryPoint := filepath.Join(modulePath, "service", "index.ts")
	if err := os.MkdirAll(filepath.Dir(entryPoint), 0o755); err != nil {
		t.Fatalf("mkdir entry dir: %v", err)
	}
	if err := os.WriteFile(entryPoint, []byte("export const answer = 42\n"), 0o644); err != nil {
		t.Fatalf("write entry point: %v", err)
	}

	_, _, serviceDir, err := modulegenerator.WorkspaceGeneratedAPITargets(testRuntimeScope.cfg.ModulesPath, "crm", testRuntimeScope.cfg.DefaultChoysumPath)
	if err != nil {
		t.Fatalf("WorkspaceGeneratedAPITargets(crm) error = %v", err)
	}
	serviceIndex := filepath.Join(serviceDir, "index.ts")
	if err := os.MkdirAll(filepath.Dir(serviceIndex), 0o755); err != nil {
		t.Fatalf("mkdir service index dir: %v", err)
	}
	if err := os.WriteFile(serviceIndex, []byte("export * from './service';\n"), 0o644); err != nil {
		t.Fatalf("write service index: %v", err)
	}
	if err := runtimeDB.Create(&meta.Module{Name: "crm", Status: meta.Installed, ApplicationStr: "crm", ServiceEntryPoint: "service/index.ts"}).Error; err != nil {
		t.Fatalf("seed runtime installed module: %v", err)
	}

	prebuildPlugin := &stubEsbPlugin{name: "prebuild"}
	buildPlugin := &stubEsbPlugin{name: "build"}
	var parserRuntimeScope scope.Scope
	builder := &ModuleBuilder{
		runtimeScope:   testRuntimeScope,
		module:         &meta.Module{Name: "auth", ApplicationStr: "auth", Path: modulePath},
		entryPoint:     entryPoint,
		prebuildPlugin: prebuildPlugin,
		buildPlugin:    buildPlugin,
		publishDist:    false,
		outFileName:    "index.js",
		globalName:     "AuthApp",
		// Empty registry: this test covers entry imports / runtime session wiring, not C2 inject.
		injectRegistry: injectappmodel.NewRegistry(),
		tsParserFactory: func(runtimeScope scope.Scope, module *meta.Module) parser.Parser {
			parserRuntimeScope = runtimeScope
			return fixedParser{}
		},
	}

	runtimeScope := &builderTestScope{ctx: context.Background(), cfg: testRuntimeScope.cfg, logger: testRuntimeScope.logger, session: &scope.Session{DB: runtimeDB}}
	ctx := scope.ContextWithScope(context.Background(), runtimeScope)
	if _, err := builder.BundleToDirCtx(ctx, filepath.Join(t.TempDir(), "stage")); err != nil {
		t.Fatalf("BundleToDirCtx(runtime scope) error = %v", err)
	}

	wantImport := normalizedAbsImportPath(serviceIndex)
	if len(prebuildPlugin.entryImports) != 1 || prebuildPlugin.entryImports[0] != wantImport {
		t.Fatalf("prebuild entry imports = %#v, want [%q]", prebuildPlugin.entryImports, wantImport)
	}
	if len(buildPlugin.entryImports) != 1 || buildPlugin.entryImports[0] != wantImport {
		t.Fatalf("build entry imports = %#v, want [%q]", buildPlugin.entryImports, wantImport)
	}
	if parserRuntimeScope == nil || parserRuntimeScope.Session() == nil || parserRuntimeScope.Session().DB != runtimeDB {
		t.Fatalf("expected runtime parser env to use runtime DB, got %#v", parserRuntimeScope)
	}
}

func TestBundleToDirCtx_PreservesRuntimeTransactionWhenCallerContextHasNoTransaction(t *testing.T) {
	baseDB, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "backendbuilder_base_fallback.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open base sqlite: %v", err)
	}
	runtimeDB, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "backendbuilder_runtime_fallback.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open runtime sqlite: %v", err)
	}
	if err := baseDB.AutoMigrate(&meta.Module{}); err != nil {
		t.Fatalf("migrate base modules: %v", err)
	}
	if err := runtimeDB.AutoMigrate(&meta.Module{}); err != nil {
		t.Fatalf("migrate runtime modules: %v", err)
	}

	testRuntimeScope := newBuilderTestScope()
	testRuntimeScope.session = &scope.Session{DB: baseDB}
	testRuntimeScope.cfg.ModulesPath = filepath.Join(t.TempDir(), "modules")
	testRuntimeScope.cfg.DistPath = filepath.Join(t.TempDir(), "dist")
	testRuntimeScope.cfg.DefaultChoysumPath = filepath.Join(t.TempDir(), ".choysum")
	if err := os.MkdirAll(testRuntimeScope.cfg.ModulesPath, 0o755); err != nil {
		t.Fatalf("mkdir modules path: %v", err)
	}
	if err := os.WriteFile(filepath.Join(testRuntimeScope.cfg.ModulesPath, "tsconfig.json"), []byte(`{"compilerOptions":{"baseUrl":".","paths":{"@/*":["./*"]}}}`), 0o644); err != nil {
		t.Fatalf("write tsconfig: %v", err)
	}
	modulePath := filepath.Join(testRuntimeScope.cfg.ModulesPath, "auth")
	entryPoint := filepath.Join(modulePath, "service", "index.ts")
	if err := os.MkdirAll(filepath.Dir(entryPoint), 0o755); err != nil {
		t.Fatalf("mkdir entry dir: %v", err)
	}
	if err := os.WriteFile(entryPoint, []byte("export const answer = 42\n"), 0o644); err != nil {
		t.Fatalf("write entry point: %v", err)
	}

	_, _, serviceDir, err := modulegenerator.WorkspaceGeneratedAPITargets(testRuntimeScope.cfg.ModulesPath, "crm", testRuntimeScope.cfg.DefaultChoysumPath)
	if err != nil {
		t.Fatalf("WorkspaceGeneratedAPITargets(crm) error = %v", err)
	}
	serviceIndex := filepath.Join(serviceDir, "index.ts")
	if err := os.MkdirAll(filepath.Dir(serviceIndex), 0o755); err != nil {
		t.Fatalf("mkdir service index dir: %v", err)
	}
	if err := os.WriteFile(serviceIndex, []byte("export * from './service';\n"), 0o644); err != nil {
		t.Fatalf("write service index: %v", err)
	}
	if err := runtimeDB.Create(&meta.Module{Name: "crm", Status: meta.Installed, ApplicationStr: "crm", ServiceEntryPoint: "service/index.ts"}).Error; err != nil {
		t.Fatalf("seed runtime installed module: %v", err)
	}

	prebuildPlugin := &stubEsbPlugin{name: "prebuild"}
	buildPlugin := &stubEsbPlugin{name: "build"}
	var parserRuntimeScope scope.Scope
	builder := &ModuleBuilder{
		runtimeScope:   testRuntimeScope,
		module:         &meta.Module{Name: "auth", ApplicationStr: "auth", Path: modulePath},
		entryPoint:     entryPoint,
		prebuildPlugin: prebuildPlugin,
		buildPlugin:    buildPlugin,
		publishDist:    false,
		outFileName:    "index.js",
		globalName:     "AuthApp",
		// Empty registry: this test covers transaction preservation, not C2 inject.
		injectRegistry: injectappmodel.NewRegistry(),
		tsParserFactory: func(runtimeScope scope.Scope, module *meta.Module) parser.Parser {
			parserRuntimeScope = runtimeScope
			return fixedParser{}
		},
	}

	txSession := &scope.Session{DB: runtimeDB}
	tx := &builderTestTransaction{session: txSession}
	tx.ctx = scope.ContextWithTransaction(context.Background(), tx)
	builder.runtimeScope = &builderTestScope{ctx: tx.ctx, cfg: testRuntimeScope.cfg, logger: testRuntimeScope.logger, session: nil}

	if _, err := builder.BundleToDirCtx(context.Background(), filepath.Join(t.TempDir(), "stage")); err != nil {
		t.Fatalf("BundleToDirCtx(background) error = %v", err)
	}

	if parserRuntimeScope == nil || parserRuntimeScope.Session() == nil || parserRuntimeScope.Session().DB != runtimeDB {
		t.Fatalf("expected runtime parser env to preserve transaction DB, got %#v", parserRuntimeScope)
	}
	if len(prebuildPlugin.entryImports) != 1 || len(buildPlugin.entryImports) != 1 {
		t.Fatalf("expected entry imports to be resolved via preserved runtime transaction, got prebuild=%#v build=%#v", prebuildPlugin.entryImports, buildPlugin.entryImports)
	}
}

func TestUpdateModelExtends_UsesStableFieldsWithoutAstNode(t *testing.T) {
	b := &ModuleBuilder{}
	raw := "export default class Child extends BaseModel {}\n"
	extendsText := "extends BaseModel"
	start := strings.Index(raw, extendsText)
	if start < 0 {
		t.Fatalf("failed to find extends text in fixture")
	}

	r := &parser.ParserResult{
		RawContent: raw,
		Model: &meta.Model{
			RawExtends: "/virtual/modules/base/models/base_model.ts",
			Extends:    "/virtual/modules/ext/models/base_model.ts",
		},
		ModelExtendsProperty: &parser.PropertyNode{
			Line:  1,
			Text:  extendsText,
			Start: start,
			End:   start + len(extendsText),
		},
	}

	extendedModel := &meta.Model{Path: "/virtual/modules/ext/models/base_model"}
	if err := b.updateModelExtends(r, extendedModel); err != nil {
		t.Fatalf("updateModelExtends failed: %v", err)
	}

	if !strings.Contains(r.Content, "extends model_") {
		t.Fatalf("expected extends statement to be rewritten, got: %s", r.Content)
	}

	re := regexp.MustCompile(`import model_[a-z0-9]+ from '/virtual/modules/ext/models/base_model';`)
	if !re.MatchString(r.Content) {
		t.Fatalf("expected rewritten import statement to be appended, got: %s", r.Content)
	}
}

func TestUpdateModelExtends_ReturnsErrorWhenOffsetsBecomeStale(t *testing.T) {
	b := &ModuleBuilder{}

	raw := "@Model('Partner')\nexport default class Partner extends PartnerBase {}\n"
	extendsText := "extends PartnerBase"
	start := strings.Index(raw, extendsText)
	if start < 0 {
		t.Fatalf("failed to find extends text in raw fixture")
	}

	// Simulate prior content edits (e.g. decorator option injection) that shift offsets.
	content := strings.Replace(raw, "@Model('Partner')", "@Model('Partner', { application: 'partner' })", 1)

	r := &parser.ParserResult{
		RawContent: raw,
		Content:    content,
		Path:       "/virtual/modules/partner_commercial/service/models/partner.ts",
		Model: &meta.Model{
			RawExtends: "/virtual/modules/partner_bank/service/models/partner.ts",
			Extends:    "/virtual/modules/partner_legacy/service/models/partner.ts",
		},
		ModelExtendsProperty: &parser.PropertyNode{
			Line:  2,
			Text:  extendsText,
			Start: start,
			End:   start + len(extendsText),
		},
	}

	extendedModel := &meta.Model{Path: "/virtual/modules/partner_bank/service/models/partner"}
	err := b.updateModelExtends(r, extendedModel)
	if err == nil {
		t.Fatalf("expected stale offsets to fail without range match")
	}
}

func TestUpdateModelExtends_ReturnsErrorOnMismatchedExtendsSnippet(t *testing.T) {
	b := &ModuleBuilder{}

	raw := "export default class Child extends OtherBase {}\n"
	extendsText := "extends BaseModel"
	start := strings.Index(raw, "extends OtherBase")
	if start < 0 {
		t.Fatalf("failed to find extends text in raw fixture")
	}

	r := &parser.ParserResult{
		RawContent: raw,
		Path:       "/virtual/modules/test/service/models/child.ts",
		Model: &meta.Model{
			RawExtends: "/virtual/modules/base/service/models/base_model.ts",
			Extends:    "/virtual/modules/ext/service/models/base_model.ts",
		},
		ModelExtendsProperty: &parser.PropertyNode{
			Line:  1,
			Text:  extendsText,
			Start: start,
			End:   start + len("extends OtherBase"),
		},
	}

	extendedModel := &meta.Model{Path: "/virtual/modules/ext/service/models/base_model"}
	err := b.updateModelExtends(r, extendedModel)
	if err == nil {
		t.Fatalf("expected mismatched extends snippet to fail rewrite")
	}
}

func TestUpdateModelExtends_PreservesWhitespaceAroundExtendsSlice(t *testing.T) {
	b := &ModuleBuilder{}
	raw := "export default class Child extends BaseModel {}\n"
	start := strings.Index(raw, " extends BaseModel")
	if start < 0 {
		t.Fatalf("failed to find extends text with leading whitespace in fixture")
	}

	r := &parser.ParserResult{
		RawContent: raw,
		Path:       "/virtual/modules/test/service/models/child.ts",
		Model: &meta.Model{
			RawExtends: "/virtual/modules/base/service/models/base_model.ts",
			Extends:    "/virtual/modules/ext/service/models/base_model.ts",
		},
		ModelExtendsProperty: &parser.PropertyNode{
			Line:  1,
			Text:  "extends BaseModel",
			Start: start,
			End:   start + len(" extends BaseModel"),
		},
	}

	extendedModel := &meta.Model{Path: "/virtual/modules/ext/service/models/base_model"}
	if err := b.updateModelExtends(r, extendedModel); err != nil {
		t.Fatalf("updateModelExtends failed: %v", err)
	}

	if !strings.Contains(r.Content, "class Child extends model_") {
		t.Fatalf("expected whitespace before extends to be preserved, got: %s", r.Content)
	}
}

func TestUpdateModelExtends_UsesDeterministicAlias(t *testing.T) {
	b := &ModuleBuilder{}
	raw := "export default class Child extends BaseModel {}\n"
	extendsText := "extends BaseModel"
	start := strings.Index(raw, extendsText)
	if start < 0 {
		t.Fatalf("failed to find extends text in fixture")
	}

	buildInput := func() *parser.ParserResult {
		return &parser.ParserResult{
			RawContent: raw,
			Path:       "/virtual/modules/test/service/models/child.ts",
			Model: &meta.Model{
				RawExtends: "/virtual/modules/base/service/models/base_model.ts",
				Extends:    "/virtual/modules/ext/service/models/base_model.ts",
			},
			ModelExtendsProperty: &parser.PropertyNode{
				Line:  1,
				Text:  extendsText,
				Start: start,
				End:   start + len(extendsText),
			},
		}
	}

	extendedModel := &meta.Model{Path: "/virtual/modules/ext/service/models/base_model"}

	r1 := buildInput()
	if err := b.updateModelExtends(r1, extendedModel); err != nil {
		t.Fatalf("first updateModelExtends failed: %v", err)
	}
	r2 := buildInput()
	if err := b.updateModelExtends(r2, extendedModel); err != nil {
		t.Fatalf("second updateModelExtends failed: %v", err)
	}

	re := regexp.MustCompile(`extends\s+(model_[a-f0-9]{12})`)
	m1 := re.FindStringSubmatch(r1.Content)
	m2 := re.FindStringSubmatch(r2.Content)
	if len(m1) != 2 || len(m2) != 2 {
		t.Fatalf("failed to extract deterministic model alias from rewritten content")
	}
	if m1[1] != m2[1] {
		t.Fatalf("expected deterministic alias, got %s vs %s", m1[1], m2[1])
	}
}

func TestUpdateModelExtends_ReusesExistingDefaultImportWithoutDuplication(t *testing.T) {
	b := &ModuleBuilder{}
	raw := "import ParentModel from '/virtual/modules/ext/service/models/base_model';\n\nexport default class Child extends BaseModel {}\n"
	extendsText := "extends BaseModel"
	start := strings.Index(raw, extendsText)
	if start < 0 {
		t.Fatalf("failed to find extends text in fixture")
	}

	r := &parser.ParserResult{
		RawContent: raw,
		Path:       "/virtual/modules/test/service/models/child.ts",
		Model: &meta.Model{
			RawExtends: "/virtual/modules/base/service/models/base_model.ts",
			Extends:    "/virtual/modules/ext/service/models/base_model.ts",
		},
		ModelExtendsProperty: &parser.PropertyNode{
			Line:  3,
			Text:  extendsText,
			Start: start,
			End:   start + len(extendsText),
		},
		Imports: map[string]*parser.Import{
			"ParentModel": {
				ReferenceIdent: "default",
				ModuleSpecPath: "/virtual/modules/ext/service/models/base_model",
				Start:          0,
				End:            strings.Index(raw, "\n"),
			},
		},
	}

	extendedModel := &meta.Model{Path: "/virtual/modules/ext/service/models/base_model"}
	if err := b.updateModelExtends(r, extendedModel); err != nil {
		t.Fatalf("updateModelExtends failed: %v", err)
	}

	if !strings.Contains(r.Content, "class Child extends ParentModel") {
		t.Fatalf("expected existing default import identifier to be reused, got: %s", r.Content)
	}
	if strings.Count(r.Content, "from '/virtual/modules/ext/service/models/base_model'") != 1 {
		t.Fatalf("expected no duplicate import from same module path, got: %s", r.Content)
	}
}

func TestUpdateModelExtends_InsertsImportIntoImportRegion(t *testing.T) {
	b := &ModuleBuilder{}
	raw := "import Foo from './foo';\n\nexport default class Child extends BaseModel {}\n"
	extendsText := "extends BaseModel"
	start := strings.Index(raw, extendsText)
	if start < 0 {
		t.Fatalf("failed to find extends text in fixture")
	}

	r := &parser.ParserResult{
		RawContent: raw,
		Path:       "/virtual/modules/test/service/models/child.ts",
		Model: &meta.Model{
			RawExtends: "/virtual/modules/base/service/models/base_model.ts",
			Extends:    "/virtual/modules/ext/service/models/base_model.ts",
		},
		ModelExtendsProperty: &parser.PropertyNode{
			Line:  3,
			Text:  extendsText,
			Start: start,
			End:   start + len(extendsText),
		},
		Imports: map[string]*parser.Import{
			"Foo": {
				ReferenceIdent: "default",
				ModuleSpecPath: "/virtual/modules/test/service/models/foo",
				Start:          0,
				End:            strings.Index(raw, "\n"),
			},
		},
	}

	extendedModel := &meta.Model{Path: "/virtual/modules/ext/service/models/base_model"}
	if err := b.updateModelExtends(r, extendedModel); err != nil {
		t.Fatalf("updateModelExtends failed: %v", err)
	}

	importFooIdx := strings.Index(r.Content, "import Foo from './foo';")
	newImportIdx := strings.Index(r.Content, "from '/virtual/modules/ext/service/models/base_model';")
	classIdx := strings.Index(r.Content, "export default class Child")
	if importFooIdx < 0 || newImportIdx < 0 || classIdx < 0 {
		t.Fatalf("expected import region and class declaration to exist, got: %s", r.Content)
	}
	if !(importFooIdx < newImportIdx && newImportIdx < classIdx) {
		t.Fatalf("expected new import inserted into import region before class declaration, got: %s", r.Content)
	}
}

func TestPersistApplicationLookupAndModelDeleteErrors(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:backendbuilder-persist-errors?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&meta.Application{}, &meta.Module{}, &meta.Model{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	testRuntimeScope := newBuilderTestScope()
	testRuntimeScope.session = &scope.Session{DB: db}
	builder := &ModuleBuilder{
		runtimeScope: testRuntimeScope,
		module:       &meta.Module{Name: "auth", Path: "/virtual/modules/auth", ApplicationStr: "auth"},
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB() error = %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("sqlDB.Close() error = %v", err)
	}
	buildResult := &module.BuildResult{Module: builder.module}
	if err := builder.persist(buildResult); err == nil || !strings.Contains(err.Error(), "error getting application by name") {
		t.Fatalf("persist() with closed DB error = %v, want application lookup failure", err)
	}

	db, err = gorm.Open(sqlite.Open("file:backendbuilder-persist-models?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("reopen sqlite: %v", err)
	}
	if err := db.AutoMigrate(modmeta.CatalogEntities()...); err != nil {
		t.Fatalf("ensure dual store: %v", err)
	}
	testRuntimeScope.session = &scope.Session{DB: db}
	if err := db.Migrator().DropTable("meta_raw_model"); err != nil {
		t.Fatalf("drop meta_raw_model: %v", err)
	}
	err = builder.persistModuleModels("module-1", []*meta.Model{{Name: "Partner", Path: "/models/partner"}})
	if err == nil {
		t.Fatal("expected persistModuleModels delete error")
	}

	db, err = gorm.Open(sqlite.Open("file:backendbuilder-persist-app-fallback?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite for fallback: %v", err)
	}
	if err := db.AutoMigrate(&meta.Application{}, &meta.Module{}); err != nil {
		t.Fatalf("auto migrate fallback db: %v", err)
	}
	if err := db.AutoMigrate(modmeta.CatalogEntities()...); err != nil {
		t.Fatalf("ensure dual store fallback: %v", err)
	}
	testRuntimeScope.session = &scope.Session{DB: db}
	builder.module = &meta.Module{Name: "crm", Path: "/virtual/modules/crm", ApplicationStr: "crm"}
	buildResult = &module.BuildResult{Module: builder.module}
	if err := builder.persist(buildResult); err != nil {
		t.Fatalf("persist() without existing application error = %v", err)
	}
	if builder.module.Application == nil || builder.module.Application.Name != "crm" {
		t.Fatalf("expected fallback application object, got %#v", builder.module.Application)
	}
}

func TestGetNewExtends_FindError(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:backendbuilder-getnewextends-find?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(modmeta.CatalogEntities()...); err != nil {
		t.Fatalf("ensure dual store: %v", err)
	}
	testRuntimeScope := newBuilderTestScope()
	testRuntimeScope.session = &scope.Session{DB: db}
	builder := &ModuleBuilder{
		runtimeScope: testRuntimeScope,
		module:       &meta.Module{ApplicationStr: "auth"},
	}
	if err := db.Migrator().DropTable("meta_raw_model"); err != nil {
		t.Fatalf("drop meta_raw_model: %v", err)
	}
	current := &meta.Model{Name: "Partner", Path: "/models/current", Extends: "/models/base"}
	if _, err := builder.getNewExtends(current); err == nil || !strings.Contains(err.Error(), "error getting last models") {
		t.Fatalf("expected find error, got %v", err)
	}
}

func TestPersistModuleModels_ErrorBranches(t *testing.T) {
	openPersistDB := func(name string) (*gorm.DB, *ModuleBuilder) {
		t.Helper()
		db, err := gorm.Open(sqlite.Open("file:backendbuilder-persist-branches-"+name+"?mode=memory&cache=shared"), &gorm.Config{})
		if err != nil {
			t.Fatalf("open sqlite: %v", err)
		}
		if err := db.AutoMigrate(modmeta.CatalogEntities()...); err != nil {
			t.Fatalf("ensure dual store: %v", err)
		}
		testRuntimeScope := newBuilderTestScope()
		testRuntimeScope.session = &scope.Session{DB: db}
		return db, &ModuleBuilder{runtimeScope: testRuntimeScope}
	}
	models := []*meta.Model{{Name: "Partner", Path: "/models/partner", Application: "partner"}}

	t.Run("list previous raw models", func(t *testing.T) {
		db, builder := openPersistDB("prev-raw")
		if err := db.Migrator().DropTable("meta_raw_model"); err != nil {
			t.Fatal(err)
		}
		err := builder.persistModuleModels("module-1", models)
		if err == nil || !strings.Contains(err.Error(), "list previous raw models") {
			t.Fatalf("expected list previous raw error, got %v", err)
		}
	})

	t.Run("list previous effective models", func(t *testing.T) {
		db, builder := openPersistDB("prev-eff")
		if err := db.Exec("ALTER TABLE meta_model RENAME COLUMN application TO application_broken").Error; err != nil {
			t.Fatal(err)
		}
		err := builder.persistModuleModels("module-1", models)
		if err == nil || !strings.Contains(err.Error(), "list previous effective models") {
			t.Fatalf("expected list previous effective error, got %v", err)
		}
	})

	t.Run("delete previous raw models", func(t *testing.T) {
		db, builder := openPersistDB("delete-raw")
		if _, err := modmeta.ReplaceModuleDeclarations(db, "module-1", []*meta.Model{{
			BaseModel:   meta.BaseModel{Id: sql.NullString{String: "del-raw", Valid: true}},
			Name:        "Old",
			Path:        "/models/old",
			Application: "partner",
			ModuleId:    sql.NullString{String: "module-1", Valid: true},
		}}); err != nil {
			t.Fatal(err)
		}
		boom := errors.New("forced raw delete")
		const cbTag = "force-raw-module-delete"
		if err := db.Callback().Delete().Before("gorm:delete").Register(cbTag, func(tx *gorm.DB) {
			if tx.Statement != nil && tx.Statement.Table == "meta_raw_model" {
				_ = tx.AddError(boom)
			}
		}); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = db.Callback().Delete().Remove(cbTag) })
		err := builder.persistModuleModels("module-1", nil)
		if err == nil || !strings.Contains(err.Error(), "delete previous raw models") {
			t.Fatalf("expected delete raw error, got %v", err)
		}
	})

	t.Run("persist raw model", func(t *testing.T) {
		db, builder := openPersistDB("persist-raw")
		boom := errors.New("forced raw create")
		const cbTag = "force-raw-model-create"
		if err := db.Callback().Create().Before("gorm:create").Register(cbTag, func(tx *gorm.DB) {
			if tx.Statement != nil && tx.Statement.Table == "meta_raw_model" {
				_ = tx.AddError(boom)
			}
		}); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = db.Callback().Create().Remove(cbTag) })
		err := builder.persistModuleModels("module-1", models)
		if err == nil || !strings.Contains(err.Error(), "persist raw model") {
			t.Fatalf("expected persist raw error, got %v", err)
		}
	})

	t.Run("flush effective models", func(t *testing.T) {
		db, builder := openPersistDB("flush")
		const cbTag = "drop-effective-after-raw-create"
		if err := db.Callback().Create().After("gorm:after_create").Register(cbTag, func(tx *gorm.DB) {
			if tx.Statement != nil && tx.Statement.Table == "meta_raw_model" {
				_ = tx.Migrator().DropTable(&meta.Model{})
			}
		}); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = db.Callback().Create().Remove(cbTag) })
		err := builder.persistModuleModels("module-1", models)
		if err == nil || !strings.Contains(err.Error(), "flush effective models") {
			t.Fatalf("expected flush error, got %v", err)
		}
	})
}
