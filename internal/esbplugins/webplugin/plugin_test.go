// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package webplugin

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/antchfx/htmlquery"
	"github.com/choysum-dev/choysum/internal/esbplugins"
	"github.com/choysum-dev/choysum/internal/parser"
	"github.com/choysum-dev/choysum/internal/testing/jsexecutortest"
	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/internal/vueplugin"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/evanw/esbuild/pkg/api"
	"golang.org/x/net/html"
)

type stubScope struct {
	ctx    context.Context
	cfg    *config.Config
	logger *slog.Logger
}

func (e *stubScope) Run(fn func(scope.Scope) error) error { return fn(e) }

func (e *stubScope) Transactor() scope.Transactor {
	return scopetest.NewPassthroughTransactor(e)
}

func (e *stubScope) Session() *scope.Session { return nil }

func (e *stubScope) WithContext(ctx context.Context) scope.Scope {
	return &stubScope{ctx: ctx, cfg: e.cfg, logger: e.logger}
}

func (e *stubScope) Context() context.Context {
	if e.ctx != nil {
		return e.ctx
	}
	return context.Background()
}

func (e *stubScope) Logger() *slog.Logger { return e.logger }

func (e *stubScope) Config() *config.Config { return e.cfg }

func (e *stubScope) FactoryInput() scope.FactoryInput {
	return scopetest.FactoryInputFromConfig(e.Config())
}

type fakeParser struct {
	parseFn func(pathAlias map[string]string, path string, content string) (*parser.ParserResult, error)
}

func (p fakeParser) Parse(pathAlias map[string]string, path string, content string) (*parser.ParserResult, error) {
	if p.parseFn != nil {
		return p.parseFn(pathAlias, path, content)
	}
	return nil, nil
}

func newTestScope(t *testing.T) scope.Scope {
	t.Helper()
	return &stubScope{
		ctx: context.Background(),
		cfg: &config.Config{
			ModulesPath: t.TempDir(),
			DistPath:    filepath.Join(t.TempDir(), "dist"),
			Compile:     config.NewDefaultCompileConfig(),
			Server:      config.NewDefaultServerConfig(),
			Log:         config.NewDefaultLogConfig(),
		},
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

func newPluginForTest(t *testing.T, parserImpl parser.Parser) *WebPlugin {
	t.Helper()
	testRuntimeScope := newTestScope(t)
	testRuntimeOpts := runtimeOptionsFromScope(testRuntimeScope)
	plugin := NewWebPlugin(testRuntimeScope, &meta.IrModule{Name: "web"}, filepath.Join(testRuntimeOpts.modulesPath, "app", "web", "index.ts"), WithParser(parserImpl)).(*WebPlugin)
	plugin.ParserResultChan = make(chan *parser.ParserResult, 8)
	plugin.ParserResults = make([]*parser.ParserResult, 0)
	return plugin
}

func renderHTML(t *testing.T, node *html.Node) string {
	t.Helper()
	var builder strings.Builder
	if err := html.Render(&builder, node); err != nil {
		t.Fatalf("render html: %v", err)
	}
	return builder.String()
}

func parseHTMLDoc(t *testing.T, markup string) *html.Node {
	t.Helper()
	doc, err := htmlquery.Parse(strings.NewReader(markup))
	if err != nil {
		t.Fatalf("parse html: %v", err)
	}
	return doc
}

func TestHtmlIconProcessorCopiesFileAndUpdatesHref(t *testing.T) {
	plugin := newPluginForTest(t, fakeParser{})
	htmlDir := t.TempDir()
	htmlFile := filepath.Join(htmlDir, "index.html")
	iconPath := filepath.Join(htmlDir, "favicon.ico")
	if err := os.WriteFile(iconPath, []byte("ico"), 0o644); err != nil {
		t.Fatalf("write icon: %v", err)
	}

	processor := plugin.htmlIconProcessor("/web/")
	build := &api.PluginBuild{InitialOptions: &api.BuildOptions{Outdir: t.TempDir()}}
	opts := &vueplugin.Options{IndexHtmlOptions: vueplugin.IndexHtmlOptions{SourceFile: htmlFile}}

	for _, markup := range []string{
		`<html><head></head><body></body></html>`,
		`<html><head><link rel="icon"></head><body></body></html>`,
		`<html><head><link rel="icon" href="missing.ico"></head><body></body></html>`,
	} {
		doc := parseHTMLDoc(t, markup)
		if err := processor(doc, &api.BuildResult{}, opts, build); err != nil {
			t.Fatalf("processor unexpected error for %q: %v", markup, err)
		}
	}

	doc := parseHTMLDoc(t, `<html><head><link rel="icon" href="favicon.ico"></head><body></body></html>`)
	if err := processor(doc, &api.BuildResult{}, opts, build); err != nil {
		t.Fatalf("processor success case: %v", err)
	}
	if got := htmlquery.FindOne(doc, `//link[@rel="icon"]`); got == nil || htmlquery.SelectAttr(got, "href") != "/web/favicon.ico" {
		t.Fatalf("unexpected icon href after processing: %q", renderHTML(t, doc))
	}
	content, err := os.ReadFile(filepath.Join(build.InitialOptions.Outdir, "favicon.ico"))
	if err != nil {
		t.Fatalf("read copied favicon: %v", err)
	}
	if string(content) != "ico" {
		t.Fatalf("copied favicon = %q, want ico", string(content))
	}
}

func TestSecurityHtmlProcessorAddsExpectedMetaTags(t *testing.T) {
	plugin := newPluginForTest(t, fakeParser{})

	missingHead := &html.Node{Type: html.DocumentNode}
	if err := plugin.securityHtmlProcessor()(missingHead, &api.BuildResult{}, &vueplugin.Options{}, &api.PluginBuild{}); err == nil || !strings.Contains(err.Error(), "head tag not found") {
		t.Fatalf("expected missing head error, got %v", err)
	}

	plugin.runtimeOptions.serverEnvironment = "production"
	plugin.runtimeOptions.serverEnabledTLS = true
	prodDoc := parseHTMLDoc(t, `<html><head><title>x</title></head><body></body></html>`)
	if err := plugin.securityHtmlProcessor()(prodDoc, &api.BuildResult{}, &vueplugin.Options{}, &api.PluginBuild{}); err != nil {
		t.Fatalf("production security processor: %v", err)
	}
	prodHTML := renderHTML(t, prodDoc)
	for _, want := range []string{"Content-Security-Policy", "upgrade-insecure-requests", "Strict-Transport-Security", "Permissions-Policy", "strict-origin-when-cross-origin", "X-Content-Type-Options"} {
		if !strings.Contains(prodHTML, want) {
			t.Fatalf("production html missing %q in %q", want, prodHTML)
		}
	}

	plugin.runtimeOptions.serverEnvironment = "development"
	plugin.runtimeOptions.serverEnabledTLS = false
	devDoc := parseHTMLDoc(t, `<html><head></head><body></body></html>`)
	if err := plugin.securityHtmlProcessor()(devDoc, &api.BuildResult{}, &vueplugin.Options{}, &api.PluginBuild{}); err != nil {
		t.Fatalf("development security processor: %v", err)
	}
	devHTML := renderHTML(t, devDoc)
	if strings.Contains(devHTML, "Permissions-Policy") || !strings.Contains(devHTML, "origin-when-cross-origin") || !strings.Contains(devHTML, "unsafe-eval") {
		t.Fatalf("unexpected development security html: %q", devHTML)
	}
}

func TestBuildEndProcessorCleansStaleFiles(t *testing.T) {
	plugin := newPluginForTest(t, fakeParser{})
	processor := plugin.BuildEndProcessor()
	outDir := t.TempDir()
	keep := filepath.Join(outDir, "keep.js")
	stale := filepath.Join(outDir, "stale.js")
	if err := os.WriteFile(keep, []byte("keep"), 0o644); err != nil {
		t.Fatalf("write keep: %v", err)
	}
	if err := os.WriteFile(stale, []byte("stale"), 0o644); err != nil {
		t.Fatalf("write stale: %v", err)
	}

	if err := processor(&api.BuildResult{}, &api.BuildOptions{Write: false, Outdir: outDir}); err != nil {
		t.Fatalf("expected no-op when write=false, got %v", err)
	}
	if _, err := os.Stat(stale); err != nil {
		t.Fatalf("stale file removed during no-op: %v", err)
	}

	result := &api.BuildResult{OutputFiles: []api.OutputFile{{Path: keep, Contents: []byte("keep")}}}
	if err := processor(result, &api.BuildOptions{Write: true, Outdir: outDir}); err != nil {
		t.Fatalf("build end processor: %v", err)
	}
	if _, err := os.Stat(keep); err != nil {
		t.Fatalf("expected keep file to remain: %v", err)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Fatalf("expected stale file to be removed, got %v", err)
	}
}

func TestHandleTsFileProcessesEntryPointAndPublishesParserResult(t *testing.T) {
	plugin := newPluginForTest(t, fakeParser{parseFn: func(pathAlias map[string]string, path string, content string) (*parser.ParserResult, error) {
		if len(pathAlias) != 0 {
			t.Fatalf("unexpected pathAlias: %#v", pathAlias)
		}
		return &parser.ParserResult{
			Path:       path,
			RawContent: content,
			Exports: map[string]*parser.Export{
				"default": {ModuleSpecPath: "./App.vue", ReferenceIdent: "default"},
			},
		}, nil
	}})
	plugin.EntryPointImports = []string{"./boot"}
	entryPath := plugin.EntryPoint
	if err := os.MkdirAll(filepath.Dir(entryPath), 0o755); err != nil {
		t.Fatalf("mkdir entry dir: %v", err)
	}
	if err := os.WriteFile(entryPath, []byte("console.log('start')\n"), 0o644); err != nil {
		t.Fatalf("write entry file: %v", err)
	}

	content, err := plugin.handleTsFile(api.OnLoadArgs{Path: entryPath}, api.PluginBuild{InitialOptions: &api.BuildOptions{}})
	if err != nil {
		t.Fatalf("handleTsFile entry point: %v", err)
	}
	if !strings.Contains(content, "import './boot';") || !strings.Contains(content, ".mount('#app')") || !strings.Contains(content, "import ") {
		t.Fatalf("unexpected transformed content: %q", content)
	}

	select {
	case parsed := <-plugin.ParserResultChan:
		if parsed == nil || parsed.Content != content {
			t.Fatalf("unexpected parser result published: %#v", parsed)
		}
	case <-time.After(time.Second):
		t.Fatal("expected parser result to be published")
	}
}

func TestHandleTsFileProcessesEntryPoint_WhenEntryPathResolvesSymlink(t *testing.T) {
	plugin := newPluginForTest(t, fakeParser{parseFn: func(pathAlias map[string]string, path string, content string) (*parser.ParserResult, error) {
		if len(pathAlias) != 0 {
			t.Fatalf("unexpected pathAlias: %#v", pathAlias)
		}
		return &parser.ParserResult{
			Path:       path,
			RawContent: content,
			Exports: map[string]*parser.Export{
				"default": {ModuleSpecPath: "./App.vue", ReferenceIdent: "default"},
			},
		}, nil
	}})

	rootDir := t.TempDir()
	realDir := filepath.Join(rootDir, "real")
	linkDir := filepath.Join(rootDir, "link")
	if err := os.MkdirAll(realDir, 0o755); err != nil {
		t.Fatalf("mkdir real dir: %v", err)
	}
	if err := os.Symlink(realDir, linkDir); err != nil {
		t.Skipf("symlink not supported: %v", err)
	}

	entryFromEntryPoint := filepath.Join(linkDir, "index.ts")
	entryFromOnLoad := filepath.Join(realDir, "index.ts")
	plugin.EntryPoint = entryFromEntryPoint
	plugin.EntryPointImports = []string{"./boot"}

	if err := os.WriteFile(entryFromOnLoad, []byte("console.log('start')\n"), 0o644); err != nil {
		t.Fatalf("write entry file: %v", err)
	}

	content, err := plugin.handleTsFile(api.OnLoadArgs{Path: entryFromOnLoad}, api.PluginBuild{InitialOptions: &api.BuildOptions{}})
	if err != nil {
		t.Fatalf("handleTsFile entry point via symlink: %v", err)
	}
	if !strings.Contains(content, "import './boot';") || !strings.Contains(content, ".mount('#app')") {
		t.Fatalf("unexpected transformed content: %q", content)
	}
}

func TestHandleTsFileReturnsErrorWhenDefaultExportMissing(t *testing.T) {
	plugin := newPluginForTest(t, fakeParser{parseFn: func(pathAlias map[string]string, path string, content string) (*parser.ParserResult, error) {
		return &parser.ParserResult{Path: path, RawContent: content, Exports: map[string]*parser.Export{}}, nil
	}})
	entryPath := plugin.EntryPoint
	if err := os.MkdirAll(filepath.Dir(entryPath), 0o755); err != nil {
		t.Fatalf("mkdir entry dir: %v", err)
	}
	if err := os.WriteFile(entryPath, []byte("console.log('start')\n"), 0o644); err != nil {
		t.Fatalf("write entry file: %v", err)
	}

	_, err := plugin.handleTsFile(api.OnLoadArgs{Path: entryPath}, api.PluginBuild{InitialOptions: &api.BuildOptions{}})
	if err == nil || !strings.Contains(err.Error(), "default export not found") {
		t.Fatalf("expected missing default export error, got %v", err)
	}
}

func TestReplaceModuleSpecReferenceIdentAndDefinePlugins(t *testing.T) {
	plugin := newPluginForTest(t, fakeParser{})
	plugin.TsExports = map[string]map[string]*parser.Export{
		"components": {
			"Button": {ModuleSpecPath: "resolved/button.vue", ReferenceIdent: "default"},
		},
		"base": {
			"default": {ModuleSpecPath: "resolved/base.vue", ReferenceIdent: "BasePage"},
		},
	}
	results := []*parser.ParserResult{{
		Path:                   "view.vue",
		VueComponent:           &meta.IrComponent{},
		VueComponentsPropertys: []*parser.PropertyNode{{ModuleSpecPath: "components", ReferenceIdent: "Button"}},
		VueExtendsProperty:     &parser.PropertyNode{ModuleSpecPath: "base", ReferenceIdent: "BasePage"},
	}}
	if err := plugin.replaceModuleSpecReferenceIdent(results); err != nil {
		t.Fatalf("replaceModuleSpecReferenceIdent: %v", err)
	}
	if got := results[0].VueComponentsPropertys[0].ModuleSpecPath; got != "resolved/button.vue" {
		t.Fatalf("component module spec = %q, want resolved/button.vue", got)
	}
	if got := results[0].VueExtendsProperty.ModuleSpecPath; got != "resolved/base.vue" || results[0].VueExtendsProperty.ReferenceIdent != "default" {
		t.Fatalf("extends property not rewritten: %#v", results[0].VueExtendsProperty)
	}
	if results[0].VueComponent.RawExtends != "resolved/base.vue" || results[0].VueComponent.Extends != "resolved/base.vue" {
		t.Fatalf("vue component extends not rewritten: %#v", results[0].VueComponent)
	}

	if err := plugin.SetParserResults(results); err != nil {
		t.Fatalf("SetParserResults: %v", err)
	}
	plugin.SetEntryPointImports([]string{"./a", "./b"})
	plugin.SetIndexHtmlOutFile("custom/index.html")
	plugins := plugin.DefinePlugins(plugin.Env, jsexecutortest.NewUninitializedExecutor(), &meta.IrModule{Name: "web"}, esbplugins.WithEntryPointImports([]string{"./boot"}), esbplugins.WithIndexHtmlOutFile("stage/index.html"))
	if len(plugins) != 2 || plugins[0].Name != "choysum-web-vue" || plugins[1].Name != "choysum-web-ts" {
		t.Fatalf("unexpected define plugins result: %#v", plugins)
	}
	if strings.ReplaceAll(plugin.IndexHtmlOutFile, "\\", "/") != "stage/index.html" {
		t.Fatalf("index html out file = %q, want stage/index.html", plugin.IndexHtmlOutFile)
	}
	if strings.Join(plugin.EntryPointImports, ",") != "./boot" {
		t.Fatalf("entry point imports = %#v, want ./boot", plugin.EntryPointImports)
	}
	if _, ok := plugin.Parser.(fakeParser); !ok {
		t.Fatalf("expected custom parser to survive DefinePlugins, got %T", plugin.Parser)
	}
}

func TestWebPluginDefinePlugins_BindsRuntimeState(t *testing.T) {
	baseScope := newTestScope(t)
	runtimeScope := newTestScope(t)
	baseOpts := runtimeOptionsFromScope(baseScope)
	runtimeOpts := runtimeOptionsFromScope(runtimeScope)
	baseModule := &meta.IrModule{Name: "base", Path: filepath.Join(baseOpts.modulesPath, "base", "web", "index.ts"), ApplicationStr: "base"}
	runtimeModule := &meta.IrModule{Name: "runtime", Path: filepath.Join(runtimeOpts.modulesPath, "runtime", "web", "index.ts"), ApplicationStr: "runtime"}

	plugin, ok := NewWebPlugin(baseScope, baseModule, filepath.Join(baseOpts.modulesPath, "base", "web", "index.ts")).(*WebPlugin)
	if !ok {
		t.Fatalf("expected *WebPlugin, got %T", plugin)
	}

	var gotScope scope.Scope
	var gotModule *meta.IrModule
	plugin.parserFactory = func(runtimeScope scope.Scope, module *meta.IrModule) parser.Parser {
		gotScope = runtimeScope
		gotModule = module
		return fakeParser{}
	}

	plugins := plugin.DefinePlugins(runtimeScope, jsexecutortest.NewUninitializedExecutor(), runtimeModule)
	if len(plugins) != 2 {
		t.Fatalf("expected two plugins, got %d", len(plugins))
	}
	if plugin.Env != runtimeScope {
		t.Fatal("expected runtime scope to replace base scope")
	}
	if plugin.Module != runtimeModule {
		t.Fatal("expected runtime module to replace base module")
	}
	if gotScope != runtimeScope {
		t.Fatalf("parser factory scope = %#v, want %#v", gotScope, runtimeScope)
	}
	if gotModule != runtimeModule {
		t.Fatalf("parser factory module = %#v, want %#v", gotModule, runtimeModule)
	}
	if _, ok := plugin.Parser.(fakeParser); !ok {
		t.Fatalf("expected runtime parser factory result, got %T", plugin.Parser)
	}
}

func TestVueLoadProcessorAndResolveProcessor(t *testing.T) {
	resolvePlugin := newPluginForTest(t, fakeParser{parseFn: func(pathAlias map[string]string, path string, content string) (*parser.ParserResult, error) {
		return &parser.ParserResult{Path: path, Content: strings.ToUpper(content)}, nil
	}})
	resolvePlugin.ParserResults = []*parser.ParserResult{{
		Path:             "/repo/components/Parent.vue",
		VueAppImportTree: []string{"/repo/components/Parent.vue", "/repo/components/Child.vue"},
		Content:          "<template><div/></template>",
	}}
	buildOptions := &api.BuildOptions{TsconfigRaw: `{"compilerOptions":{"paths":{"@/*":["src/*"]}}}`, AbsWorkingDir: "/repo"}
	args := &api.OnResolveArgs{Importer: "/repo/components/Parent.vue", Path: "@/components/Child.vue"}
	if result, err := resolvePlugin.VueResolveProcessor()(args, buildOptions); err != nil || result != nil {
		t.Fatalf("resolve processor returned result=%#v err=%v", result, err)
	}
	if args.Path != "/repo/src/components/Child.vue" {
		t.Fatalf("resolved path = %q, want /repo/src/components/Child.vue", args.Path)
	}

	loaded, err := resolvePlugin.VueLoadProcessor()("<template><span>ok</span></template>", api.OnLoadArgs{Path: "/repo/components/Parent.vue"}, &api.BuildOptions{})
	if err != nil {
		t.Fatalf("vue load processor: %v", err)
	}
	if loaded != "<template><div/></template>" {
		t.Fatalf("loaded content = %q, want parser result content", loaded)
	}
	select {
	case result := <-resolvePlugin.ParserResultChan:
		if result == nil || result.Path != "/repo/components/Parent.vue" {
			t.Fatalf("unexpected vue parser result: %#v", result)
		}
	case <-time.After(time.Second):
		t.Fatal("expected vue parser result to be published")
	}

	errPlugin := newPluginForTest(t, fakeParser{parseFn: func(pathAlias map[string]string, path string, content string) (*parser.ParserResult, error) {
		return nil, errors.New("parse failed")
	}})
	if _, err := errPlugin.VueLoadProcessor()("<template/>", api.OnLoadArgs{Path: "/repo/components/Error.vue"}, &api.BuildOptions{}); err == nil || !strings.Contains(err.Error(), "parse failed") {
		t.Fatalf("expected vue load parse error, got %v", err)
	}
}
