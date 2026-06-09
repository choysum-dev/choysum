// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package webprebuildplugin

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

	"github.com/choysum-dev/choysum/internal/esbplugins"
	"github.com/choysum-dev/choysum/internal/parser"
	"github.com/choysum-dev/choysum/internal/testing/jsexecutortest"
	"github.com/choysum-dev/choysum/internal/testing/scopetest"
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
	return &parser.ParserResult{Path: path, RawContent: content}, nil
}

func newTestScope(t *testing.T) scope.Scope {
	t.Helper()
	modulesPath := t.TempDir()
	return &stubScope{
		ctx: context.Background(),
		cfg: &config.Config{
			ModulesPath: modulesPath,
			DistPath:    filepath.Join(t.TempDir(), "dist"),
			Compile:     config.NewDefaultCompileConfig(),
			Server:      config.NewDefaultServerConfig(),
			Log:         config.NewDefaultLogConfig(),
		},
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

func newPluginForTest(t *testing.T, parserImpl parser.Parser) *WebPrebuildPlugin {
	t.Helper()
	testRuntimeScope := newTestScope(t)
	testRuntimeOpts := runtimeOptionsFromScope(testRuntimeScope)
	plugin := NewWebPrebuildPlugin(testRuntimeScope, &meta.IrModule{Name: "web"}, filepath.Join(testRuntimeOpts.modulesPath, "app", "web", "index.ts"), WithParser(parserImpl))
	if plugin == nil {
		t.Fatal("expected web prebuild plugin to be created")
	}
	plugin.ParserResultChan = make(chan *parser.ParserResult, 8)
	plugin.ParserResults = make([]*parser.ParserResult, 0)
	return plugin
}

func textElement(tag string, attrs []html.Attribute, text string) *html.Node {
	node := &html.Node{Type: html.ElementNode, Data: tag, Attr: attrs}
	if text != "" {
		node.AppendChild(&html.Node{Type: html.TextNode, Data: text})
	}
	return node
}

func captureOnLoad(t *testing.T, plugin api.Plugin, buildOptions *api.BuildOptions) func(api.OnLoadArgs) (api.OnLoadResult, error) {
	t.Helper()
	var onLoad func(api.OnLoadArgs) (api.OnLoadResult, error)
	plugin.Setup(api.PluginBuild{
		InitialOptions: buildOptions,
		OnLoad: func(options api.OnLoadOptions, callback func(api.OnLoadArgs) (api.OnLoadResult, error)) {
			onLoad = callback
		},
	})
	if onLoad == nil {
		t.Fatal("expected OnLoad callback to be registered")
	}
	return onLoad
}

func TestPrebuildVuePluginOnLoadUsesTSLoaderAndPublishesResult(t *testing.T) {
	plugin := newPluginForTest(t, fakeParser{parseFn: func(pathAlias map[string]string, path string, content string) (*parser.ParserResult, error) {
		if got := pathAlias["@/*"]; !strings.HasSuffix(strings.ReplaceAll(got, "\\", "/"), "/src/*") {
			t.Fatalf("unexpected path alias map: %#v", pathAlias)
		}
		if !strings.Contains(content, `@use './styles/base.scss' as *;`) || !strings.Contains(content, `@forward './styles/tokens.scss';`) {
			t.Fatalf("expected vue style path to be rewritten, got %q", content)
		}
		return &parser.ParserResult{
			Path:          path,
			RawContent:    content,
			RawScriptNode: textElement("script", []html.Attribute{{Key: "lang", Val: "ts"}}, "const value = 1"),
		}, nil
	}})

	vuePath := filepath.Join(t.TempDir(), "src", "App.vue")
	if err := os.MkdirAll(filepath.Dir(vuePath), 0o755); err != nil {
		t.Fatalf("mkdir vue dir: %v", err)
	}
	if err := os.WriteFile(vuePath, []byte("<template><div/></template><style>@use '@/styles/base.scss' as *; @forward '@/styles/tokens.scss';</style><script lang=\"ts\">const value = 1</script>"), 0o644); err != nil {
		t.Fatalf("write vue file: %v", err)
	}

	onLoad := captureOnLoad(t, plugin.prebuildVuePlugin(), &api.BuildOptions{TsconfigRaw: `{"compilerOptions":{"paths":{"@/*":["src/*"]}}}`, AbsWorkingDir: filepath.Dir(filepath.Dir(vuePath))})
	result, err := onLoad(api.OnLoadArgs{Path: vuePath})
	if err != nil {
		t.Fatalf("onLoad returned error: %v", err)
	}
	if result.Loader != api.LoaderTS {
		t.Fatalf("loader = %v, want LoaderTS", result.Loader)
	}
	if result.Contents == nil || !strings.Contains(*result.Contents, "const value = 1") || !strings.Contains(*result.Contents, "export default {}") {
		t.Fatalf("unexpected onLoad contents: %#v", result.Contents)
	}

	select {
	case parsed := <-plugin.ParserResultChan:
		if parsed == nil || parsed.Path != vuePath {
			t.Fatalf("unexpected parser result published: %#v", parsed)
		}
	case <-time.After(time.Second):
		t.Fatal("expected parser result to be published")
	}
}

func TestPrebuildVuePluginOnLoadUsesScriptSetupAndPreservesExistingDefaultExport(t *testing.T) {
	plugin := newPluginForTest(t, fakeParser{parseFn: func(pathAlias map[string]string, path string, content string) (*parser.ParserResult, error) {
		return &parser.ParserResult{
			Path:               path,
			RawContent:         content,
			RawScriptSetupNode: textElement("script", nil, "const setupValue = 2"),
			RawScriptNode:      textElement("script", nil, "export default { name: 'App' }"),
		}, nil
	}})

	vuePath := filepath.Join(t.TempDir(), "App.vue")
	if err := os.WriteFile(vuePath, []byte("<script setup>const setupValue = 2</script><script>export default { name: 'App' }</script>"), 0o644); err != nil {
		t.Fatalf("write vue file: %v", err)
	}

	onLoad := captureOnLoad(t, plugin.prebuildVuePlugin(), &api.BuildOptions{})
	result, err := onLoad(api.OnLoadArgs{Path: vuePath})
	if err != nil {
		t.Fatalf("onLoad returned error: %v", err)
	}
	if result.Loader != api.LoaderJS {
		t.Fatalf("loader = %v, want LoaderJS", result.Loader)
	}
	if result.Contents == nil {
		t.Fatal("expected onLoad contents")
	}
	if strings.Count(*result.Contents, "export default") != 1 || !strings.Contains(*result.Contents, "const setupValue = 2") {
		t.Fatalf("unexpected onLoad contents: %q", *result.Contents)
	}
}

func TestPrebuildVuePluginOnLoadPropagatesFileAndParserErrors(t *testing.T) {
	plugin := newPluginForTest(t, fakeParser{parseFn: func(pathAlias map[string]string, path string, content string) (*parser.ParserResult, error) {
		return nil, errors.New("parse failed")
	}})
	onLoad := captureOnLoad(t, plugin.prebuildVuePlugin(), &api.BuildOptions{})

	if _, err := onLoad(api.OnLoadArgs{Path: filepath.Join(t.TempDir(), "missing.vue")}); err == nil {
		t.Fatal("expected missing file error")
	}

	vuePath := filepath.Join(t.TempDir(), "Broken.vue")
	if err := os.WriteFile(vuePath, []byte("<script>const x = 1</script>"), 0o644); err != nil {
		t.Fatalf("write vue file: %v", err)
	}
	if _, err := onLoad(api.OnLoadArgs{Path: vuePath}); err == nil || !strings.Contains(err.Error(), "parse failed") {
		t.Fatalf("expected parser error, got %v", err)
	}
}

func TestDefinePluginsAndSetters(t *testing.T) {
	plugin := newPluginForTest(t, fakeParser{})
	plugins := plugin.DefinePlugins(plugin.Env, jsexecutortest.NewUninitializedExecutor(), &meta.IrModule{Name: "web"}, esbplugins.WithEntryPointImports([]string{"./boot"}), esbplugins.WithIndexHtmlOutFile("stage/index.html"))
	if len(plugins) != 2 {
		t.Fatalf("plugins len = %d, want 2", len(plugins))
	}
	if plugins[0].Name != "choysum-web-vue-prebuild" || plugins[1].Name != "choysum-web-ts" {
		t.Fatalf("unexpected plugin names: %#v", plugins)
	}
	if got := strings.ReplaceAll(plugin.IndexHtmlOutFile, "\\", "/"); got != "stage/index.html" {
		t.Fatalf("index html out file = %q, want stage/index.html", got)
	}
	if got := strings.Join(plugin.EntryPointImports, ","); got != "./boot" {
		t.Fatalf("entry point imports = %q, want ./boot", got)
	}

	plugin = NewWebPrebuildPlugin(plugin.Env, &meta.IrModule{Name: "web"}, plugin.EntryPoint)
	if plugin == nil || plugin.Parser == nil {
		t.Fatal("expected default parser to be initialized")
	}
}

func TestWebPrebuildPluginDefinePlugins_BindsRuntimeState(t *testing.T) {
	baseScope := newTestScope(t)
	runtimeScope := newTestScope(t)
	baseOpts := runtimeOptionsFromScope(baseScope)
	runtimeOpts := runtimeOptionsFromScope(runtimeScope)
	baseModule := &meta.IrModule{Name: "base", Path: filepath.Join(baseOpts.modulesPath, "base", "web", "index.ts"), ApplicationStr: "base"}
	runtimeModule := &meta.IrModule{Name: "runtime", Path: filepath.Join(runtimeOpts.modulesPath, "runtime", "web", "index.ts"), ApplicationStr: "runtime"}

	plugin := NewWebPrebuildPlugin(baseScope, baseModule, filepath.Join(baseOpts.modulesPath, "base", "web", "index.ts"), WithParser(fakeParser{}))
	if plugin == nil {
		t.Fatal("expected web prebuild plugin")
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
	if _, ok := plugin.Parser.(fakeParser); !ok {
		t.Fatalf("expected custom parser to survive DefinePlugins, got %T", plugin.Parser)
	}
}
