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
	"runtime"
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
	plugin := NewWebPlugin(testRuntimeScope, &meta.Module{Name: "web"}, filepath.Join(testRuntimeOpts.modulesPath, "app", "web", "index.ts"), WithParser(parserImpl)).(*WebPlugin)
	plugin.ParserResultChan = make(chan *parser.ParserResult, 8)
	plugin.ParserResults = make([]*parser.ParserResult, 0)
	return plugin
}

func captureWebTsOnResolve(t *testing.T, plugin api.Plugin, buildOptions *api.BuildOptions) func(api.OnResolveArgs) (api.OnResolveResult, error) {
	t.Helper()

	var onResolve func(api.OnResolveArgs) (api.OnResolveResult, error)
	plugin.Setup(api.PluginBuild{
		InitialOptions: buildOptions,
		OnLoad: func(api.OnLoadOptions, func(api.OnLoadArgs) (api.OnLoadResult, error)) {
		},
		OnResolve: func(options api.OnResolveOptions, callback func(api.OnResolveArgs) (api.OnResolveResult, error)) {
			onResolve = callback
		},
	})
	if onResolve == nil {
		t.Fatal("expected web ts plugin to register an OnResolve callback")
	}
	return onResolve
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

	webRoot := t.TempDir()
	assetsOut := filepath.Join(webRoot, "assets")
	if err := os.MkdirAll(assetsOut, 0o755); err != nil {
		t.Fatalf("mkdir assets out: %v", err)
	}
	processor := plugin.htmlIconProcessor("/web/")
	build := &api.PluginBuild{InitialOptions: &api.BuildOptions{Outdir: assetsOut}}
	opts := &vueplugin.Options{IndexHtmlOptions: vueplugin.IndexHtmlOptions{
		SourceFile: htmlFile,
		OutFile:    filepath.Join(webRoot, "index.html"),
	}}

	for _, markup := range []string{
		`<html><head></head><body></body></html>`,
		`<html><head><link rel="icon"></head><body></body></html>`,
		`<html><head><link rel="icon" href="missing.ico"></head><body></body></html>`,
		`<html><head><link rel="icon" href="https://cdn.example/favicon.ico"></head><body></body></html>`,
		`<html><head><link rel="icon" href="favicon.ico?v=2"></head><body></body></html>`,
		`<html><head><link rel="icon" href="../../etc/passwd"></head><body></body></html>`,
		`<html><head><link rel="icon" href="."></head><body></body></html>`,
		`<html><head><link rel="icon" href=".."></head><body></body></html>`,
		// Base("favicon.ico/.") is "."; Join still resolves to the real file.
		`<html><head><link rel="icon" href="favicon.ico/."></head><body></body></html>`,
		`<html><head><link rel="icon" href="/"></head><body></body></html>`,
	} {
		doc := parseHTMLDoc(t, markup)
		if err := processor(doc, &api.BuildResult{}, opts, build); err != nil {
			t.Fatalf("processor unexpected error for %q: %v", markup, err)
		}
	}
	if _, err := os.Stat(filepath.Join(webRoot, "passwd")); err == nil {
		t.Fatal("traversal href must not publish outside htmlDir basename into web root")
	}
	// Query-stripped relative href should still publish basename beside index.html.
	if got, err := os.ReadFile(filepath.Join(webRoot, "favicon.ico")); err != nil || string(got) != "ico" {
		t.Fatalf("favicon after query href = %q err=%v", string(got), err)
	}

	doc := parseHTMLDoc(t, `<html><head><link rel="icon" href="favicon.ico"></head><body></body></html>`)
	if err := processor(doc, &api.BuildResult{}, opts, build); err != nil {
		t.Fatalf("processor success case: %v", err)
	}
	if got := htmlquery.FindOne(doc, `//link[@rel="icon"]`); got == nil || htmlquery.SelectAttr(got, "href") != "/web/favicon.ico" {
		t.Fatalf("unexpected icon href after processing: %q", renderHTML(t, doc))
	}
	content, err := os.ReadFile(filepath.Join(webRoot, "favicon.ico"))
	if err != nil {
		t.Fatalf("read copied favicon: %v", err)
	}
	if string(content) != "ico" {
		t.Fatalf("copied favicon = %q, want ico", string(content))
	}

	// Source under assets/ must still publish as /web/favicon.ico at web root
	// (not under esbuild Outdir assets/, which cleanup may prune).
	assetsDir := filepath.Join(htmlDir, "assets")
	if err := os.MkdirAll(assetsDir, 0o755); err != nil {
		t.Fatalf("mkdir assets: %v", err)
	}
	nestedIcon := filepath.Join(assetsDir, "favicon.ico")
	if err := os.WriteFile(nestedIcon, []byte("nested-ico"), 0o644); err != nil {
		t.Fatalf("write nested icon: %v", err)
	}
	if err := os.WriteFile(filepath.Join(assetsDir, "logo-192.png"), []byte("png"), 0o644); err != nil {
		t.Fatalf("write logo-192: %v", err)
	}
	webRoot2 := t.TempDir()
	assetsOut2 := filepath.Join(webRoot2, "assets")
	if err := os.MkdirAll(assetsOut2, 0o755); err != nil {
		t.Fatalf("mkdir assets out2: %v", err)
	}
	build2 := &api.PluginBuild{InitialOptions: &api.BuildOptions{Outdir: assetsOut2}}
	opts2 := &vueplugin.Options{IndexHtmlOptions: vueplugin.IndexHtmlOptions{
		SourceFile: htmlFile,
		OutFile:    filepath.Join(webRoot2, "index.html"),
	}}
	doc2 := parseHTMLDoc(t, `<html><head><link rel="icon" href="assets/favicon.ico"></head><body></body></html>`)
	if err := processor(doc2, &api.BuildResult{}, opts2, build2); err != nil {
		t.Fatalf("processor assets/ href: %v", err)
	}
	if got := htmlquery.FindOne(doc2, `//link[@rel="icon"]`); got == nil || htmlquery.SelectAttr(got, "href") != "/web/favicon.ico" {
		t.Fatalf("assets/ icon href = %q, want /web/favicon.ico", renderHTML(t, doc2))
	}
	nestedOut, err := os.ReadFile(filepath.Join(webRoot2, "favicon.ico"))
	if err != nil || string(nestedOut) != "nested-ico" {
		t.Fatalf("copied nested favicon = %q err=%v", string(nestedOut), err)
	}
	touch, err := os.ReadFile(filepath.Join(webRoot2, "apple-touch-icon.png"))
	if err != nil || string(touch) != "png" {
		t.Fatalf("apple-touch copy = %q err=%v", string(touch), err)
	}

	// Parent-of-assets fallback when IndexHtml OutFile is unset.
	webRoot3 := t.TempDir()
	assetsOut3 := filepath.Join(webRoot3, "assets")
	if err := os.MkdirAll(assetsOut3, 0o755); err != nil {
		t.Fatalf("mkdir assets out3: %v", err)
	}
	opts3 := &vueplugin.Options{IndexHtmlOptions: vueplugin.IndexHtmlOptions{SourceFile: htmlFile}}
	build3 := &api.PluginBuild{InitialOptions: &api.BuildOptions{Outdir: assetsOut3}}
	doc3 := parseHTMLDoc(t, `<html><head><link rel="icon" href="favicon.ico"></head><body></body></html>`)
	if err := processor(doc3, &api.BuildResult{}, opts3, build3); err != nil {
		t.Fatalf("processor without OutFile: %v", err)
	}
	if _, err := os.Stat(filepath.Join(webRoot3, "favicon.ico")); err != nil {
		t.Fatalf("expected favicon beside assets/ via fallback: %v", err)
	}
}

func TestWebDistRootForIconsAndCopyHelpers(t *testing.T) {
	t.Parallel()
	if pathUnderHTMLDir("", "x") || pathUnderHTMLDir("x", "") {
		t.Fatal("empty htmlDir or candidate must be rejected")
	}
	if pathUnderHTMLDir(".", "/tmp/outside") {
		t.Fatal("absolute candidate must be rejected when Rel fails or escapes")
	}
	if got := webDistRootForIcons(nil, ""); got != "" {
		t.Fatalf("empty root = %q", got)
	}
	if got := webDistRootForIcons(nil, "/tmp/plain-out"); got != "/tmp/plain-out" {
		t.Fatalf("non-assets outdir = %q", got)
	}
	if got := stripURLQueryFragment("  a.ico?v=1#x "); got != "a.ico" {
		t.Fatalf("strip = %q", got)
	}
	dir := t.TempDir()
	if err := copyWebBuildAsset(dir, filepath.Join(dir, "missing-parent", "out"), "nope.png", "x.png"); !os.IsNotExist(err) {
		t.Fatalf("missing source err=%v", err)
	}
	blocked := filepath.Join(dir, "not-a-dir")
	if err := os.WriteFile(blocked, []byte("x"), 0o644); err != nil {
		t.Fatalf("write blocker: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "src.png"), []byte("png"), 0o644); err != nil {
		t.Fatalf("write src: %v", err)
	}
	if err := copyWebBuildAsset(dir, blocked, "src.png", "out.png"); err == nil {
		t.Fatal("expected mkdir/write failure when dest parent is a file")
	}

	plugin := newPluginForTest(t, fakeParser{})
	htmlDir := t.TempDir()
	htmlFile := filepath.Join(htmlDir, "index.html")
	if err := os.WriteFile(filepath.Join(htmlDir, "favicon.ico"), []byte("ico"), 0o644); err != nil {
		t.Fatalf("write favicon: %v", err)
	}
	processor := plugin.htmlIconProcessor("/web/")
	iconDoc := func() *html.Node {
		return parseHTMLDoc(t, `<html><head><link rel="icon" href="favicon.ico"></head><body></body></html>`)
	}
	opts := &vueplugin.Options{IndexHtmlOptions: vueplugin.IndexHtmlOptions{SourceFile: htmlFile}}
	if err := processor(iconDoc(), &api.BuildResult{}, opts, &api.PluginBuild{InitialOptions: &api.BuildOptions{}}); err == nil {
		t.Fatal("expected error when web dist root is empty")
	}
	// Favicon ok but apple-touch source missing → debug skip path.
	webRoot := t.TempDir()
	optsOK := &vueplugin.Options{IndexHtmlOptions: vueplugin.IndexHtmlOptions{
		SourceFile: htmlFile,
		OutFile:    filepath.Join(webRoot, "index.html"),
	}}
	if err := processor(iconDoc(), &api.BuildResult{}, optsOK, &api.PluginBuild{InitialOptions: &api.BuildOptions{Outdir: filepath.Join(webRoot, "assets")}}); err != nil {
		t.Fatalf("processor without logo-192: %v", err)
	}

	// apple-touch write failure (dest is a directory) fails the build.
	assetsSrc := filepath.Join(htmlDir, "assets")
	if err := os.MkdirAll(assetsSrc, 0o755); err != nil {
		t.Fatalf("mkdir assets src: %v", err)
	}
	if err := os.WriteFile(filepath.Join(assetsSrc, "logo-192.png"), []byte("png"), 0o644); err != nil {
		t.Fatalf("write logo-192: %v", err)
	}
	webRoot4 := t.TempDir()
	if err := os.Mkdir(filepath.Join(webRoot4, "apple-touch-icon.png"), 0o755); err != nil {
		t.Fatalf("mkdir apple-touch blocker: %v", err)
	}
	opts4 := &vueplugin.Options{IndexHtmlOptions: vueplugin.IndexHtmlOptions{
		SourceFile: htmlFile,
		OutFile:    filepath.Join(webRoot4, "index.html"),
	}}
	if err := processor(iconDoc(), &api.BuildResult{}, opts4, &api.PluginBuild{InitialOptions: &api.BuildOptions{Outdir: filepath.Join(webRoot4, "assets")}}); err == nil {
		t.Fatal("expected apple-touch write failure to fail the build")
	}

	// Favicon mkdir/write failures must use an html tree without logo-192, otherwise
	// apple-touch copy fails first and never reaches the favicon branches.
	htmlFavOnly := t.TempDir()
	htmlFavFile := filepath.Join(htmlFavOnly, "index.html")
	if err := os.WriteFile(filepath.Join(htmlFavOnly, "favicon.ico"), []byte("ico"), 0o644); err != nil {
		t.Fatalf("write fav-only icon: %v", err)
	}
	favDoc := func() *html.Node {
		return parseHTMLDoc(t, `<html><head><link rel="icon" href="favicon.ico"></head><body></body></html>`)
	}

	// MkdirAll failure: web root nested under a file.
	blocker := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatalf("write blocker: %v", err)
	}
	opts5 := &vueplugin.Options{IndexHtmlOptions: vueplugin.IndexHtmlOptions{
		SourceFile: htmlFavFile,
		OutFile:    filepath.Join(blocker, "web", "index.html"),
	}}
	if err := processor(favDoc(), &api.BuildResult{}, opts5, &api.PluginBuild{InitialOptions: &api.BuildOptions{Outdir: filepath.Join(blocker, "web", "assets")}}); err == nil || !strings.Contains(err.Error(), "favicon destination directory") {
		t.Fatalf("expected favicon mkdir failure under file path, got %v", err)
	}

	// WriteFile failure: destination path is an existing directory.
	webRootDirBlock := t.TempDir()
	if err := os.Mkdir(filepath.Join(webRootDirBlock, "favicon.ico"), 0o755); err != nil {
		t.Fatalf("mkdir favicon blocker: %v", err)
	}
	optsWrite := &vueplugin.Options{IndexHtmlOptions: vueplugin.IndexHtmlOptions{
		SourceFile: htmlFavFile,
		OutFile:    filepath.Join(webRootDirBlock, "index.html"),
	}}
	if err := processor(favDoc(), &api.BuildResult{}, optsWrite, &api.PluginBuild{InitialOptions: &api.BuildOptions{Outdir: filepath.Join(webRootDirBlock, "assets")}}); err == nil || !strings.Contains(err.Error(), "write favicon file") {
		t.Fatalf("expected favicon write failure, got %v", err)
	}

	// WriteFile failure: read-only web root (root / Windows bypass mode bits).
	if runtime.GOOS != "windows" && os.Geteuid() != 0 {
		webRootRO := t.TempDir()
		optsRO := &vueplugin.Options{IndexHtmlOptions: vueplugin.IndexHtmlOptions{
			SourceFile: htmlFavFile,
			OutFile:    filepath.Join(webRootRO, "index.html"),
		}}
		if err := os.Chmod(webRootRO, 0o555); err != nil {
			t.Fatalf("chmod: %v", err)
		}
		t.Cleanup(func() { _ = os.Chmod(webRootRO, 0o755) })
		if err := processor(favDoc(), &api.BuildResult{}, optsRO, &api.PluginBuild{InitialOptions: &api.BuildOptions{Outdir: filepath.Join(webRootRO, "assets")}}); err == nil {
			t.Fatal("expected write failure on read-only web root")
		}
	}

	// Apple-touch still publishes when <link rel="icon"> is absent.
	htmlDirNoIcon := t.TempDir()
	assetsNoIcon := filepath.Join(htmlDirNoIcon, "assets")
	if err := os.MkdirAll(assetsNoIcon, 0o755); err != nil {
		t.Fatalf("mkdir assets: %v", err)
	}
	if err := os.WriteFile(filepath.Join(assetsNoIcon, "logo-192.png"), []byte("touch"), 0o644); err != nil {
		t.Fatalf("write logo: %v", err)
	}
	webRootNoIcon := t.TempDir()
	optsNoIcon := &vueplugin.Options{IndexHtmlOptions: vueplugin.IndexHtmlOptions{
		SourceFile: filepath.Join(htmlDirNoIcon, "index.html"),
		OutFile:    filepath.Join(webRootNoIcon, "index.html"),
	}}
	docNoIcon := parseHTMLDoc(t, `<html><head></head><body></body></html>`)
	if err := processor(docNoIcon, &api.BuildResult{}, optsNoIcon, &api.PluginBuild{InitialOptions: &api.BuildOptions{Outdir: filepath.Join(webRootNoIcon, "assets")}}); err != nil {
		t.Fatalf("processor without icon tag: %v", err)
	}
	if got, err := os.ReadFile(filepath.Join(webRootNoIcon, "apple-touch-icon.png")); err != nil || string(got) != "touch" {
		t.Fatalf("apple-touch without icon tag = %q err=%v", string(got), err)
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
	bootIdx := strings.Index(content, "import './boot';")
	bodyIdx := strings.Index(content, "console.log('start')")
	if bootIdx < 0 || bodyIdx < 0 || bootIdx < bodyIdx {
		t.Fatalf("expected non-store entrypoint imports appended after original body, got %q", content)
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
		VueComponent:           &meta.Component{},
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
	plugins := plugin.DefinePlugins(plugin.Env, jsexecutortest.NewUninitializedExecutor(), &meta.Module{Name: "web"}, esbplugins.WithEntryPointImports([]string{"./boot"}), esbplugins.WithIndexHtmlOutFile("stage/index.html"))
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
	baseModule := &meta.Module{Name: "base", Path: filepath.Join(baseOpts.modulesPath, "base", "web", "index.ts"), ApplicationStr: "base"}
	runtimeModule := &meta.Module{Name: "runtime", Path: filepath.Join(runtimeOpts.modulesPath, "runtime", "web", "index.ts"), ApplicationStr: "runtime"}

	plugin, ok := NewWebPlugin(baseScope, baseModule, filepath.Join(baseOpts.modulesPath, "base", "web", "index.ts")).(*WebPlugin)
	if !ok {
		t.Fatalf("expected *WebPlugin, got %T", plugin)
	}

	var gotScope scope.Scope
	var gotModule *meta.Module
	plugin.parserFactory = func(runtimeScope scope.Scope, module *meta.Module) parser.Parser {
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

func TestWebPluginTsPluginOnResolveUsesModulesPathImporterMapping(t *testing.T) {
	runtimeScope := newTestScope(t)
	runtimeOpts := runtimeOptionsFromScope(runtimeScope)
	plugin := NewWebPlugin(runtimeScope, &meta.Module{Name: "web", Path: filepath.Join(runtimeOpts.modulesPath, "web")}, filepath.Join(runtimeOpts.modulesPath, "auth", "web", "index.ts"), WithParser(fakeParser{})).(*WebPlugin)

	resolvePath := filepath.Join(t.TempDir(), "child.ts")
	if err := os.WriteFile(resolvePath, []byte("export default {}\n"), 0o644); err != nil {
		t.Fatalf("write resolve target file: %v", err)
	}
	externalParent := filepath.Join(t.TempDir(), "external", "parent.ts")
	plugin.ParserResults = []*parser.ParserResult{{
		Path:             resolvePath,
		VueAppImportTree: []string{externalParent, resolvePath},
	}}

	onResolve := captureWebTsOnResolve(t, plugin.TsPlugin(), &api.BuildOptions{})
	result, err := onResolve(api.OnResolveArgs{
		Path:       resolvePath,
		ResolveDir: filepath.Dir(resolvePath),
		Importer:   filepath.Join(runtimeOpts.modulesPath, "auth", "web", "index.ts"),
	})
	if err != nil {
		t.Fatalf("OnResolve callback error = %v", err)
	}
	if result.Path != externalParent {
		t.Fatalf("OnResolve callback path = %q, want %q", result.Path, externalParent)
	}
}

func TestWebPluginIsEntryPointPath(t *testing.T) {
	plugin := newPluginForTest(t, fakeParser{})
	plugin.EntryPoint = "/repo/web/index.ts"

	if !plugin.isEntryPointPath("/repo/web/index.ts") {
		t.Fatal("expected exact entry path to match")
	}
	if !plugin.isEntryPointPath("/repo/web/./index.ts") {
		t.Fatal("expected cleaned entry path to match")
	}
	if plugin.isEntryPointPath("   ") {
		t.Fatal("expected blank path not to match entry path")
	}

	plugin.EntryPoint = "   "
	if plugin.isEntryPointPath("/repo/web/index.ts") {
		t.Fatal("expected blank entry point not to match")
	}
}

func TestWebPluginPrioritizedEntryPointImports(t *testing.T) {
	plugin := newPluginForTest(t, fakeParser{})
	plugin.EntryPointImports = []string{
		"/repo/generated/api/web/crm/stores/index.ts",
		"/repo/app/boot.ts",
		"/repo/generated/web/crm/service.ts",
		"",
	}

	ordered := plugin.prioritizedEntryPointImports()
	if len(ordered) != 4 {
		t.Fatalf("prioritizedEntryPointImports() len = %d, want 4", len(ordered))
	}
	if ordered[0] != "/repo/generated/api/web/crm/stores/index.ts" {
		t.Fatalf("expected store import first, got %#v", ordered)
	}
	if ordered[1] != "/repo/app/boot.ts" || ordered[2] != "/repo/generated/web/crm/service.ts" || ordered[3] != "" {
		t.Fatalf("expected non-store imports to keep relative order, got %#v", ordered)
	}

	plugin.EntryPointImports = nil
	if got := plugin.prioritizedEntryPointImports(); got != nil {
		t.Fatalf("prioritizedEntryPointImports(nil) = %#v, want nil", got)
	}
}

func TestInjectEntryPointContent_PrependsOnlyStoreImportsBeforeAppReexport(t *testing.T) {
	plugin := newPluginForTest(t, fakeParser{})
	plugin.EntryPointImports = []string{
		"/repo/generated/web/base/stores/index.ts",
		"/repo/modules/base/web/index.ts",
	}
	parserResult := &parser.ParserResult{
		Exports: map[string]*parser.Export{
			"default": {ModuleSpecPath: "./app", ReferenceIdent: "default"},
		},
	}

	content, changed, err := plugin.injectEntryPointContent(
		"/repo/modules/web/web/index.ts",
		"export { default } from './app';\n",
		parserResult,
	)
	if err != nil {
		t.Fatalf("injectEntryPointContent: %v", err)
	}
	if !changed {
		t.Fatal("expected content to change")
	}

	storeIdx := strings.Index(content, "import '/repo/generated/web/base/stores/index.ts';")
	moduleIdx := strings.Index(content, "import '/repo/modules/base/web/index.ts';")
	reexportIdx := strings.Index(content, "export { default } from './app';")
	mountIdx := strings.Index(content, ".mount('#app')")
	if storeIdx < 0 || moduleIdx < 0 || reexportIdx < 0 || mountIdx < 0 {
		t.Fatalf("missing expected fragments: %q", content)
	}
	// stores first (factory registration), then app, then other module entries, then mount
	if !(storeIdx < reexportIdx && reexportIdx < moduleIdx && moduleIdx < mountIdx) {
		t.Fatalf("expected stores → app reexport → module entry → mount, got %q", content)
	}
}
