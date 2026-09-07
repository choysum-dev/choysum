// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package frontend

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	_ "github.com/choysum-dev/choysum/internal/defaultengine"
	_ "github.com/choysum-dev/choysum/internal/defaultjsexecutor"
	"github.com/choysum-dev/choysum/internal/testing/coverage"
	"github.com/choysum-dev/choysum/internal/vueplugin"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsengine/quickjsengine"
	"github.com/choysum-dev/choysum/pkg/jsexecutor"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/evanw/esbuild/pkg/api"
)

// spikeBuildScope is a minimal scope for NewCompilerExecutor (mirrors bootstrap web gen).
type spikeBuildScope struct {
	ctx context.Context
	cfg *config.Config
}

func (s *spikeBuildScope) Run(fn func(scope.Scope) error) error { return fn(s) }
func (s *spikeBuildScope) Session() *scope.Session              { return nil }
func (s *spikeBuildScope) Transactor() scope.Transactor         { return nil }
func (s *spikeBuildScope) WithContext(ctx context.Context) scope.Scope {
	clone := *s
	clone.ctx = ctx
	return &clone
}
func (s *spikeBuildScope) Context() context.Context { return s.ctx }
func (s *spikeBuildScope) Logger() *slog.Logger     { return slog.Default() }
func (s *spikeBuildScope) FactoryInput() scope.FactoryInput {
	if s.cfg == nil {
		return nil
	}
	return &spikeFactoryInput{cfg: s.cfg}
}

type spikeFactoryInput struct{ cfg *config.Config }

func (i *spikeFactoryInput) Environment() string                  { return "" }
func (i *spikeFactoryInput) ModulesPath() string                  { return "" }
func (i *spikeFactoryInput) DistPath() string                     { return "" }
func (i *spikeFactoryInput) TmpPath() string                      { return "" }
func (i *spikeFactoryInput) DefaultChoysumPath() string           { return "" }
func (i *spikeFactoryInput) ConfigPath() string                   { return "" }
func (i *spikeFactoryInput) ESMUpstreamURL() string               { return "" }
func (i *spikeFactoryInput) NpmRegistryURL() string               { return "" }
func (i *spikeFactoryInput) ModuleCatalogIndexURL() string        { return "" }
func (i *spikeFactoryInput) CompileConfig() *config.CompileConfig { return nil }
func (i *spikeFactoryInput) AuthConfig() *config.AuthConfig       { return nil }
func (i *spikeFactoryInput) TaskConfig() *config.TaskConfig       { return nil }
func (i *spikeFactoryInput) LogConfig() *config.LogConfig         { return nil }
func (i *spikeFactoryInput) ServerConfig() *config.ServerConfig   { return i.cfg.Server }

// TestVueSFCCoverageSpike_P0 proves narrowed-A P0:
// vuesfc → esbuild+sourcemap → InstrumentJSFile → QuickJS createApp().mount()
// → WriteLcov contains hits on SpikeCounter.vue script lines.
func TestVueSFCCoverageSpike_P0(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("no caller")
	}
	fixtureDir := filepath.Join(filepath.Dir(thisFile), "testdata", "vue_cov_spike")
	for _, name := range []string{"SpikeCounter.vue", "entry.ts", "vue_stub.js"} {
		if _, err := os.Stat(filepath.Join(fixtureDir, name)); err != nil {
			t.Fatalf("fixture %s: %v", name, err)
		}
	}

	work := t.TempDir()
	for _, name := range []string{"SpikeCounter.vue", "entry.ts", "vue_stub.js"} {
		raw, err := os.ReadFile(filepath.Join(fixtureDir, name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(work, name), raw, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	vuePath := filepath.Join(work, "SpikeCounter.vue")
	entryPath := filepath.Join(work, "entry.ts")
	stubPath := filepath.Join(work, "vue_stub.js")
	outJS := filepath.Join(work, "out", "bundle.js")
	if err := os.MkdirAll(filepath.Dir(outJS), 0o755); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		Server: &config.ServerConfig{
			JsEngineFactory:   "quickjs",
			JsExecutorFactory: "default",
		},
	}
	runtimeScope := &spikeBuildScope{ctx: context.Background(), cfg: cfg}
	executor, err := jsexecutor.NewCompilerExecutor(runtimeScope)
	if err != nil {
		t.Fatalf("NewCompilerExecutor: %v", err)
	}
	if err := executor.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { _ = executor.Stop() })

	result := api.Build(api.BuildOptions{
		EntryPoints: []string{entryPath},
		Outfile:     outJS,
		Bundle:      true,
		Format:      api.FormatIIFE,
		Platform:    api.PlatformBrowser,
		Target:      api.ES2020,
		Write:       true,
		Sourcemap:   api.SourceMapLinked,
		Alias: map[string]string{
			"vue": stubPath,
		},
		Plugins: []api.Plugin{
			vueplugin.NewPlugin(vueplugin.WithJsExecutor(executor)),
		},
		Define: map[string]string{
			"import.meta.env.MODE": "'test'",
			"import.meta.env.PROD": "false",
			"import.meta.env.DEV":  "true",
			"import.meta.env.SSR":  "false",
		},
	})
	if len(result.Errors) > 0 {
		var b strings.Builder
		for _, e := range result.Errors {
			b.WriteString(e.Text)
			b.WriteByte('\n')
		}
		t.Fatalf("esbuild: %s", b.String())
	}
	if _, err := os.Stat(outJS); err != nil {
		t.Fatalf("missing bundle: %v", err)
	}
	if _, err := os.Stat(outJS + ".map"); err != nil {
		t.Fatalf("missing sourcemap: %v", err)
	}

	if err := coverage.InstrumentJSFile(outJS); err != nil {
		t.Fatalf("InstrumentJSFile: %v", err)
	}
	bundleJS, err := os.ReadFile(outJS)
	if err != nil {
		t.Fatal(err)
	}

	engine, err := quickjsengine.NewFactory()()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = engine.Close() })
	if err := engine.Load([]*jsengine.JsScript{
		{FileName: outJS, Content: string(bundleJS)},
	}); err != nil {
		t.Fatalf("Load instrumented bundle: %v", err)
	}

	qjs, ok := engine.(*quickjsengine.QuickjsEngine)
	if !ok {
		t.Fatalf("engine type %T", engine)
	}
	covVal := qjs.Ctx.Eval(`(typeof globalThis.__coverage__ === "undefined") ? "null" : JSON.stringify(globalThis.__coverage__)`)
	defer covVal.Free()
	if covVal.IsException() {
		t.Fatalf("read __coverage__: %v", qjs.Ctx.Exception())
	}
	coverageJSON := covVal.String()
	if coverageJSON == "" || coverageJSON == "null" {
		t.Fatal("expected non-empty __coverage__ after mount")
	}

	tmpRoot := t.TempDir()
	runID := "vue-cov-spike"
	if err := coverage.WriteCoverageJSONWithRunIDAndTmpRoot(work, "spike", runID, coverageJSON, tmpRoot); err != nil {
		t.Fatalf("WriteCoverageJSON: %v", err)
	}
	reportDir := filepath.Join(tmpRoot, "reports")
	if err := coverage.WriteLcov(context.Background(), coverage.ReportOptions{
		RepoRoot:  work,
		TmpRoot:   tmpRoot,
		ReportDir: reportDir,
		Reporters: []string{"lcovonly"},
		RunID:     runID,
	}); err != nil {
		t.Fatalf("WriteLcov: %v", err)
	}

	lcovPath := filepath.Join(reportDir, "lcov.info")
	lcovRaw, err := os.ReadFile(lcovPath)
	if err != nil {
		t.Fatalf("read lcov: %v", err)
	}
	lcov := string(lcovRaw)
	if !strings.Contains(lcov, "SpikeCounter.vue") {
		t.Fatalf("lcov missing SpikeCounter.vue; got:\n%s", lcov)
	}
	// Require distinctive SpikeCounter.vue script lines (not any nonzero DA in the record).
	vueIdx := strings.Index(lcov, "SpikeCounter.vue")
	chunk := lcov[vueIdx:]
	if end := strings.Index(chunk, "\nend_of_record"); end >= 0 {
		chunk = chunk[:end]
	}
	hitLines := map[int]bool{}
	for _, rawLine := range strings.Split(chunk, "\n") {
		line := strings.TrimSpace(rawLine)
		if !strings.HasPrefix(line, "DA:") {
			continue
		}
		parts := strings.Split(strings.TrimPrefix(line, "DA:"), ",")
		if len(parts) != 2 || parts[1] == "0" {
			continue
		}
		var n int
		if _, err := fmt.Sscanf(parts[0], "%d", &n); err == nil {
			hitLines[n] = true
		}
	}
	// Script lines in testdata/vue_cov_spike/SpikeCounter.vue (SPIKE_MARKER, spikeLabel body, call site).
	for _, want := range []int{8, 11, 14} {
		if !hitLines[want] {
			t.Fatalf("expected DA hit for SpikeCounter.vue script line %d; hits=%v chunk:\n%s\nfull:\n%s", want, hitLines, chunk, lcov)
		}
	}
	_ = vuePath
}
