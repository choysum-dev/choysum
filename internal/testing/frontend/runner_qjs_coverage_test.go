// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package frontend

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/choysum-dev/choysum/internal/testing/coverage"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsengine/quickjsengine"
	"github.com/choysum-dev/choysum/pkg/jsengine/scripts/choysumtest"
	"github.com/choysum-dev/choysum/pkg/jsexecutor"
	"github.com/choysum-dev/choysum/pkg/scope"
)

type stubQJSEngine struct {
	loadErr error
}

func (s stubQJSEngine) Load([]*jsengine.JsScript) error { return s.loadErr }
func (s stubQJSEngine) Execute(context.Context, *jsengine.JsRequest) (*jsengine.JsResponse, error) {
	return nil, nil
}
func (s stubQJSEngine) Close() error { return nil }

type stubJsExecutor struct {
	startErr error
	stopErr  error
}

func (s stubJsExecutor) Execute(context.Context, *jsengine.JsRequest) (*jsengine.JsResponse, error) {
	return nil, nil
}
func (s stubJsExecutor) GetJsScripts() []*jsengine.JsScript    { return nil }
func (s stubJsExecutor) SetJsScripts([]*jsengine.JsScript)     {}
func (s stubJsExecutor) Reload(...*jsengine.JsScript) error    { return nil }
func (s stubJsExecutor) AppendJsScripts(...*jsengine.JsScript) {}
func (s stubJsExecutor) Start() error                          { return s.startErr }
func (s stubJsExecutor) Stop() error                           { return s.stopErr }

func TestFrontendTestsNeedVueBranches(t *testing.T) {
	if frontendTestsNeedVue(nil) {
		t.Fatal("nil files")
	}
	prev := osReadFileQJS
	t.Cleanup(func() { osReadFileQJS = prev })
	osReadFileQJS = func(string) ([]byte, error) { return nil, os.ErrNotExist }
	if frontendTestsNeedVue([]string{"missing.ts"}) {
		t.Fatal("read error should skip")
	}
	osReadFileQJS = func(string) ([]byte, error) { return []byte(`import { mount } from '@choysum/test-utils'`), nil }
	if !frontendTestsNeedVue([]string{"a.ts"}) {
		t.Fatal("test-utils")
	}
	osReadFileQJS = func(string) ([]byte, error) { return []byte(`from '@vue/test-utils'`), nil }
	if !frontendTestsNeedVue([]string{"b.ts"}) {
		t.Fatal("vtu")
	}
	osReadFileQJS = func(string) ([]byte, error) { return []byte(`import X from './X.vue'`), nil }
	if !frontendTestsNeedVue([]string{"c.ts"}) {
		t.Fatal("vue import")
	}
	osReadFileQJS = func(string) ([]byte, error) { return []byte(`export const n = 1`), nil }
	if frontendTestsNeedVue([]string{"d.ts"}) {
		t.Fatal("plain ts")
	}
}

func TestWriteQJSTapAndJUnitHelpers(t *testing.T) {
	writeQJSTap(nil, &qjsRunReport{})
	writeQJSTap(os.Stdout, nil)

	var buf bytes.Buffer
	writeQJSTap(&buf, &qjsRunReport{
		Total: 3,
		Cases: []qjsCaseReport{
			{Name: "ok-case", OK: true},
			{Name: "fail-default", OK: false},
			{Name: "fail-msg", OK: false, Error: &struct {
				Message string `json:"message"`
				Stack   string `json:"stack"`
			}{Message: "boom", Stack: "stack"}},
		},
	})
	out := buf.String()
	if !strings.Contains(out, "ok 1 - ok-case") || !strings.Contains(out, "not ok 2") || !strings.Contains(out, "boom") {
		t.Fatalf("tap = %s", out)
	}

	if err := writeQJSJUnitIfNeeded("app", nil, "x.xml"); err != nil {
		t.Fatal(err)
	}
	if err := writeQJSJUnitIfNeeded("app", &qjsRunReport{}, ""); err != nil {
		t.Fatal(err)
	}

	junit := filepath.Join(t.TempDir(), "nested", "out.xml")
	report := &qjsRunReport{
		Total: 2, Failed: 1,
		Cases: []qjsCaseReport{
			{Name: "pass", OK: true, DurationMs: 10},
			{Name: "fail", OK: false, DurationMs: 20, Error: &struct {
				Message string `json:"message"`
				Stack   string `json:"stack"`
			}{Message: "nope", Stack: "trace"}},
			{Name: "fail2", OK: false}, // default message
		},
	}
	if err := writeQJSJUnitIfNeeded("app", report, junit); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(junit)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "nope") || !strings.Contains(string(raw), "fail2") {
		t.Fatalf("junit=%s", raw)
	}

	prevMkdir := osMkdirAllQJS
	osMkdirAllQJS = func(string, os.FileMode) error { return os.ErrPermission }
	t.Cleanup(func() { osMkdirAllQJS = prevMkdir })
	if err := writeQJSJUnitIfNeeded("app", report, filepath.Join(t.TempDir(), "d", "x.xml")); err == nil || !strings.Contains(err.Error(), "mkdir junit") {
		t.Fatalf("mkdir: %v", err)
	}
	osMkdirAllQJS = prevMkdir

	prevWrite := osWriteFileQJS
	osWriteFileQJS = func(string, []byte, os.FileMode) error { return os.ErrPermission }
	t.Cleanup(func() { osWriteFileQJS = prevWrite })
	if err := writeQJSJUnitIfNeeded("app", report, filepath.Join(t.TempDir(), "w.xml")); err == nil || !strings.Contains(err.Error(), "write junit") {
		t.Fatalf("write: %v", err)
	}
	osWriteFileQJS = prevWrite

	prevMarshal := xmlMarshalIndentQJS
	xmlMarshalIndentQJS = func(any, string, string) ([]byte, error) { return nil, os.ErrInvalid }
	t.Cleanup(func() { xmlMarshalIndentQJS = prevMarshal })
	if err := writeQJSJUnitIfNeeded("app", report, filepath.Join(t.TempDir(), "m.xml")); err == nil || !strings.Contains(err.Error(), "marshal junit") {
		t.Fatalf("marshal: %v", err)
	}
}

func TestEvalChoysumTestRunBranches(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := evalChoysumTestRun(ctx, stubQJSEngine{}, ""); err == nil {
		t.Fatal("expected canceled ctx")
	}

	if _, err := evalChoysumTestRun(context.Background(), stubQJSEngine{}, ""); err == nil || !strings.Contains(err.Error(), "unexpected engine type") {
		t.Fatalf("type: %v", err)
	}

	prevMarshal := jsonMarshalQJS
	jsonMarshalQJS = func(any) ([]byte, error) { return nil, os.ErrInvalid }
	t.Cleanup(func() { jsonMarshalQJS = prevMarshal })
	engine, err := quickjsengine.NewFactory()()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = engine.Close() })
	if _, err := evalChoysumTestRun(context.Background(), engine, "x"); err == nil {
		t.Fatal("expected marshal error")
	}
	jsonMarshalQJS = prevMarshal

	if err := engine.Load([]*jsengine.JsScript{
		{FileName: "scripts/choysumtest/choysumtest.js", Content: choysumtest.ChoysumTestScript},
	}); err != nil {
		t.Fatal(err)
	}
	// Force exception: overwrite runner to throw.
	qjs := engine.(*quickjsengine.QuickjsEngine)
	v := qjs.Ctx.Eval(`globalThis.__choysum_test_run__ = async () => { throw new Error('run-boom'); }`)
	if v.IsException() {
		t.Fatal(qjs.Ctx.Exception())
	}
	v.Free()
	if _, err := evalChoysumTestRun(context.Background(), engine, ""); err == nil || !strings.Contains(err.Error(), "fe-qjs: run:") {
		t.Fatalf("exception: %v", err)
	}

	v = qjs.Ctx.Eval(`globalThis.__choysum_test_run__ = async () => 'not-json'`)
	if v.IsException() {
		t.Fatal(qjs.Ctx.Exception())
	}
	v.Free()
	if _, err := evalChoysumTestRun(nil, engine, ""); err == nil || !strings.Contains(err.Error(), "parse report") {
		t.Fatalf("parse: %v", err)
	}

	v = qjs.Ctx.Eval(`globalThis.__choysum_test_run__ = async () => undefined`)
	if v.IsException() {
		t.Fatal(qjs.Ctx.Exception())
	}
	v.Free()
	if _, err := evalChoysumTestRun(context.Background(), engine, ""); err == nil || !strings.Contains(err.Error(), "no report") {
		t.Fatalf("undefined report: %v", err)
	}

	v = qjs.Ctx.Eval(`globalThis.__choysum_test_run__ = async () => null`)
	if v.IsException() {
		t.Fatal(qjs.Ctx.Exception())
	}
	v.Free()
	if _, err := evalChoysumTestRun(context.Background(), engine, ""); err == nil || !strings.Contains(err.Error(), "no report") {
		t.Fatalf("null report: %v", err)
	}
}

func TestEvalChoysumTestRun_TimeoutInterrupts(t *testing.T) {
	engine, err := quickjsengine.NewFactory()()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = engine.Close() })
	qjs := engine.(*quickjsengine.QuickjsEngine)
	v := qjs.Ctx.Eval(`globalThis.__choysum_test_run__ = async () => { while (true) {} }`)
	if v.IsException() {
		t.Fatal(qjs.Ctx.Exception())
	}
	v.Free()

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err = evalChoysumTestRun(ctx, engine, "")
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("expected timeout/interrupt error")
	}
	if elapsed > 3*time.Second {
		t.Fatalf("interrupt too slow: %v err=%v", elapsed, err)
	}
}

func TestNewFrontendCompilerExecutorOK(t *testing.T) {
	ex, err := newFrontendCompilerExecutor(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	_ = ex.Stop()

	prev := newCompilerExecutorFn
	t.Cleanup(func() { newCompilerExecutorFn = prev })
	newCompilerExecutorFn = func(scope.Scope) (jsexecutor.JsExecutor, error) {
		return nil, os.ErrInvalid
	}
	if _, err := newFrontendCompilerExecutor(context.Background()); err == nil || !strings.Contains(err.Error(), "NewCompilerExecutor") {
		t.Fatalf("create: %v", err)
	}
	newCompilerExecutorFn = func(scope.Scope) (jsexecutor.JsExecutor, error) {
		return stubJsExecutor{startErr: os.ErrPermission}, nil
	}
	if _, err := newFrontendCompilerExecutor(context.Background()); err == nil || !strings.Contains(err.Error(), "Start compiler") {
		t.Fatalf("start: %v", err)
	}
}

func TestRunFrontendQJS_ErrorSeams(t *testing.T) {
	repo := feQjsRepoRoot(t)
	fixture := feQjsFixtureDir(t)
	work := t.TempDir()
	for _, name := range []string{"math.ts", "math.test.ts"} {
		raw, err := os.ReadFile(filepath.Join(fixture, name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(work, name), raw, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	testFile := filepath.Join(work, "math.test.ts")

	t.Run("cancelled ctx", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if _, err := RunFrontendQJS(ctx, QJSRunOptions{RepoRoot: repo, TestFiles: []string{testFile}}); err == nil {
			t.Fatal("expected cancel")
		}
	})

	t.Run("nil ctx defaults", func(t *testing.T) {
		prevGetwd := osGetwdQJS
		osGetwdQJS = func() (string, error) { return repo, nil }
		t.Cleanup(func() { osGetwdQJS = prevGetwd })
		failed, err := RunFrontendQJS(nil, QJSRunOptions{
			TestFiles:  []string{testFile},
			WorkingDir: work,
			TmpRoot:    t.TempDir(),
			Keep:       false,
		})
		if err != nil || failed {
			t.Fatalf("nil ctx run: failed=%v err=%v", failed, err)
		}
	})

	t.Run("discover error", func(t *testing.T) {
		prev := discoverFrontendTestsQJS
		discoverFrontendTestsQJS = func(string, string) ([]string, error) { return nil, os.ErrPermission }
		t.Cleanup(func() { discoverFrontendTestsQJS = prev })
		if _, err := RunFrontendQJS(context.Background(), QJSRunOptions{RepoRoot: repo, App: "x", TmpRoot: t.TempDir()}); err == nil {
			t.Fatal("discover")
		}
	})

	t.Run("resolve tmp", func(t *testing.T) {
		prev := resolveTestingTmpDirQJS
		resolveTestingTmpDirQJS = func(context.Context, string, string, string) (string, error) {
			return "", os.ErrInvalid
		}
		t.Cleanup(func() { resolveTestingTmpDirQJS = prev })
		if _, err := RunFrontendQJS(context.Background(), QJSRunOptions{RepoRoot: repo, TestFiles: []string{testFile}, TmpRoot: t.TempDir()}); err == nil || !strings.Contains(err.Error(), "resolve tmp") {
			t.Fatalf("%v", err)
		}
	})

	t.Run("mkdir runDir", func(t *testing.T) {
		prev := osMkdirAllQJS
		osMkdirAllQJS = func(string, os.FileMode) error { return os.ErrPermission }
		t.Cleanup(func() { osMkdirAllQJS = prev })
		if _, err := RunFrontendQJS(context.Background(), QJSRunOptions{RepoRoot: repo, TestFiles: []string{testFile}, TmpRoot: t.TempDir()}); err == nil || !strings.Contains(err.Error(), "mkdir") {
			t.Fatalf("%v", err)
		}
	})

	t.Run("write entry", func(t *testing.T) {
		prev := writeFrontendTestsEntryQJS
		writeFrontendTestsEntryQJS = func(string, []string) error { return os.ErrInvalid }
		t.Cleanup(func() { writeFrontendTestsEntryQJS = prev })
		if _, err := RunFrontendQJS(context.Background(), QJSRunOptions{RepoRoot: repo, TestFiles: []string{testFile}, TmpRoot: t.TempDir()}); err == nil {
			t.Fatal("entry")
		}
	})

	t.Run("compiler fail", func(t *testing.T) {
		prev := newFrontendCompilerExecutorQ
		newFrontendCompilerExecutorQ = func(context.Context) (jsexecutor.JsExecutor, error) {
			return nil, os.ErrInvalid
		}
		t.Cleanup(func() { newFrontendCompilerExecutorQ = prev })
		if _, err := RunFrontendQJS(context.Background(), QJSRunOptions{
			RepoRoot: repo, TestFiles: []string{testFile}, WorkingDir: work, TmpRoot: t.TempDir(), ForceVue: true,
		}); err == nil {
			t.Fatal("compiler")
		}
	})

	t.Run("bundle fail", func(t *testing.T) {
		prev := buildFrontendUnitBundleQJS
		buildFrontendUnitBundleQJS = func(BundleOptions) (*BundleResult, error) { return nil, os.ErrInvalid }
		t.Cleanup(func() { buildFrontendUnitBundleQJS = prev })
		if _, err := RunFrontendQJS(context.Background(), QJSRunOptions{
			RepoRoot: repo, TestFiles: []string{testFile}, WorkingDir: work, TmpRoot: t.TempDir(),
		}); err == nil || !strings.Contains(err.Error(), "bundle") {
			t.Fatalf("%v", err)
		}
	})

	passingBundle := func(opts BundleOptions) (*BundleResult, error) {
		js := `
globalThis.__choysum_test_run__ = async () => ({
  total: 1, passed: 1, failed: 0,
  cases: [{ name: 'ok', ok: true, durationMs: 1 }],
  coverageJSON: null
});
`
		if err := os.WriteFile(opts.Outfile, []byte(js), 0o644); err != nil {
			return nil, err
		}
		return &BundleResult{JS: js, JSPath: opts.Outfile}, nil
	}

	t.Run("bootstrap timers fail", func(t *testing.T) {
		prevBuild := buildFrontendUnitBundleQJS
		buildFrontendUnitBundleQJS = passingBundle
		prevBoot := bootstrapTimersQJS
		bootstrapTimersQJS = func(*quickjsengine.QuickjsEngine) bool { return false }
		t.Cleanup(func() {
			buildFrontendUnitBundleQJS = prevBuild
			bootstrapTimersQJS = prevBoot
		})
		if _, err := RunFrontendQJS(context.Background(), QJSRunOptions{
			RepoRoot: repo, TestFiles: []string{testFile}, WorkingDir: work, TmpRoot: t.TempDir(), Keep: true,
		}); err == nil || !strings.Contains(err.Error(), "BootstrapTimers") {
			t.Fatalf("%v", err)
		}
	})

	t.Run("engine create fail", func(t *testing.T) {
		prevBuild := buildFrontendUnitBundleQJS
		buildFrontendUnitBundleQJS = passingBundle
		prevEng := newQuickJSEngineQJS
		newQuickJSEngineQJS = func() (jsengine.JsEngine, error) { return nil, os.ErrInvalid }
		t.Cleanup(func() {
			buildFrontendUnitBundleQJS = prevBuild
			newQuickJSEngineQJS = prevEng
		})
		if _, err := RunFrontendQJS(context.Background(), QJSRunOptions{
			RepoRoot: repo, TestFiles: []string{testFile}, WorkingDir: work, TmpRoot: t.TempDir(),
		}); err == nil || !strings.Contains(err.Error(), "engine") {
			t.Fatalf("%v", err)
		}
	})

	t.Run("prepare vue host fail", func(t *testing.T) {
		prevBuild := buildFrontendUnitBundleQJS
		buildFrontendUnitBundleQJS = passingBundle
		prevComp := newFrontendCompilerExecutorQ
		newFrontendCompilerExecutorQ = func(context.Context) (jsexecutor.JsExecutor, error) {
			return stubJsExecutor{}, nil
		}
		prevPrep := prepareVueHostEngineQJS
		prepareVueHostEngineQJS = func(jsengine.JsEngine) error { return os.ErrInvalid }
		t.Cleanup(func() {
			buildFrontendUnitBundleQJS = prevBuild
			newFrontendCompilerExecutorQ = prevComp
			prepareVueHostEngineQJS = prevPrep
		})
		if _, err := RunFrontendQJS(context.Background(), QJSRunOptions{
			RepoRoot: repo, TestFiles: []string{testFile}, WorkingDir: work, TmpRoot: t.TempDir(), ForceVue: true,
		}); err == nil {
			t.Fatal("prepare")
		}
	})

	t.Run("load fail", func(t *testing.T) {
		prevBuild := buildFrontendUnitBundleQJS
		buildFrontendUnitBundleQJS = passingBundle
		prevEng := newQuickJSEngineQJS
		newQuickJSEngineQJS = func() (jsengine.JsEngine, error) { return stubQJSEngine{loadErr: os.ErrInvalid}, nil }
		t.Cleanup(func() {
			buildFrontendUnitBundleQJS = prevBuild
			newQuickJSEngineQJS = prevEng
		})
		if _, err := RunFrontendQJS(context.Background(), QJSRunOptions{
			RepoRoot: repo, TestFiles: []string{testFile}, WorkingDir: work, TmpRoot: t.TempDir(),
		}); err == nil || !(strings.Contains(err.Error(), "load") || strings.Contains(err.Error(), "InstallMinimalConsole")) {
			t.Fatalf("%v", err)
		}
	})

	failingBundle := func(opts BundleOptions) (*BundleResult, error) {
		js := `
globalThis.__choysum_test_run__ = async () => ({
  total: 1, passed: 0, failed: 1,
  cases: [{ name: 'nope', ok: false, durationMs: 1, error: { message: 'x', stack: 'y' } }],
  coverageJSON: null
});
`
		if err := os.WriteFile(opts.Outfile, []byte(js), 0o644); err != nil {
			return nil, err
		}
		return &BundleResult{JS: js, JSPath: opts.Outfile}, nil
	}

	t.Run("failed tests", func(t *testing.T) {
		prevBuild := buildFrontendUnitBundleQJS
		buildFrontendUnitBundleQJS = failingBundle
		t.Cleanup(func() { buildFrontendUnitBundleQJS = prevBuild })
		failed, err := RunFrontendQJS(context.Background(), QJSRunOptions{
			RepoRoot: repo, App: "fail-app", TestFiles: []string{testFile}, WorkingDir: work, TmpRoot: t.TempDir(), Keep: true,
			JUnitPath: filepath.Join(t.TempDir(), "fail.xml"),
		})
		if err != nil {
			t.Fatal(err)
		}
		if !failed {
			t.Fatal("expected failed")
		}
	})

	covBundle := func(opts BundleOptions) (*BundleResult, error) {
		cov := `{"/tmp/x.ts":{"path":"/tmp/x.ts","s":{"0":1},"f":{},"b":{},"statementMap":{"0":{"start":{"line":1,"column":0},"end":{"line":1,"column":1}}},"fnMap":{},"branchMap":{}}}`
		js := `
globalThis.__choysum_test_run__ = async () => ({
  total: 1, passed: 1, failed: 0,
  cases: [{ name: 'ok', ok: true, durationMs: 1 }],
  coverageJSON: ` + strconvQuote(cov) + `
});
`
		if err := os.WriteFile(opts.Outfile, []byte(js), 0o644); err != nil {
			return nil, err
		}
		return &BundleResult{JS: js, JSPath: opts.Outfile}, nil
	}

	t.Run("coverage preflight fail", func(t *testing.T) {
		prevBuild := buildFrontendUnitBundleQJS
		buildFrontendUnitBundleQJS = covBundle
		prevPF := preflightCoverageQJS
		preflightCoverageQJS = func(string) error { return os.ErrInvalid }
		t.Cleanup(func() {
			buildFrontendUnitBundleQJS = prevBuild
			preflightCoverageQJS = prevPF
		})
		if _, err := RunFrontendQJS(context.Background(), QJSRunOptions{
			RepoRoot: repo, TestFiles: []string{testFile}, WorkingDir: work, TmpRoot: t.TempDir(), Coverage: true,
		}); err == nil {
			t.Fatal("preflight")
		}
	})

	t.Run("instrument fail", func(t *testing.T) {
		prevBuild := buildFrontendUnitBundleQJS
		buildFrontendUnitBundleQJS = covBundle
		prevPF := preflightCoverageQJS
		preflightCoverageQJS = func(string) error { return nil }
		prevInst := instrumentJSFileQJS
		instrumentJSFileQJS = func(string) error { return os.ErrInvalid }
		t.Cleanup(func() {
			buildFrontendUnitBundleQJS = prevBuild
			preflightCoverageQJS = prevPF
			instrumentJSFileQJS = prevInst
		})
		if _, err := RunFrontendQJS(context.Background(), QJSRunOptions{
			RepoRoot: repo, TestFiles: []string{testFile}, WorkingDir: work, TmpRoot: t.TempDir(), Coverage: true,
		}); err == nil || !strings.Contains(err.Error(), "instrument") {
			t.Fatalf("%v", err)
		}
	})

	t.Run("read instrumented fail", func(t *testing.T) {
		prevBuild := buildFrontendUnitBundleQJS
		buildFrontendUnitBundleQJS = covBundle
		prevPF := preflightCoverageQJS
		preflightCoverageQJS = func(string) error { return nil }
		prevInst := instrumentJSFileQJS
		instrumentJSFileQJS = func(string) error { return nil }
		prevRead := osReadFileQJS
		osReadFileQJS = func(string) ([]byte, error) { return nil, os.ErrNotExist }
		t.Cleanup(func() {
			buildFrontendUnitBundleQJS = prevBuild
			preflightCoverageQJS = prevPF
			instrumentJSFileQJS = prevInst
			osReadFileQJS = prevRead
		})
		if _, err := RunFrontendQJS(context.Background(), QJSRunOptions{
			RepoRoot: repo, TestFiles: []string{testFile}, WorkingDir: work, TmpRoot: t.TempDir(), Coverage: true,
		}); err == nil || !strings.Contains(err.Error(), "read instrumented") {
			t.Fatalf("%v", err)
		}
	})

	t.Run("coverage report path", func(t *testing.T) {
		prevBuild := buildFrontendUnitBundleQJS
		buildFrontendUnitBundleQJS = covBundle
		prevPF := preflightCoverageQJS
		preflightCoverageQJS = func(string) error { return nil }
		prevInst := instrumentJSFileQJS
		instrumentJSFileQJS = func(string) error { return nil }
		prevRead := osReadFileQJS
		osReadFileQJS = func(path string) ([]byte, error) {
			return []byte(`globalThis.__choysum_test_run__ = async () => ({ total:1,passed:1,failed:0,cases:[{name:'ok',ok:true,durationMs:1}], coverageJSON: ` + strconvQuote(`{"a":{"path":"a","s":{"0":1},"f":{},"b":{},"statementMap":{},"fnMap":{},"branchMap":{}}}`) + `});`), nil
		}
		prevWJSON := writeCoverageJSONQJS
		writeCoverageJSONQJS = func(string, string, string, string, string) error { return nil }
		prevLcov := writeLcovQJS
		writeLcovQJS = func(context.Context, coverage.ReportOptions) error { return nil }
		prevCheck := checkCoverageQJS
		checkCoverageQJS = func(context.Context, coverage.CheckOptions) error { return nil }
		t.Cleanup(func() {
			buildFrontendUnitBundleQJS = prevBuild
			preflightCoverageQJS = prevPF
			instrumentJSFileQJS = prevInst
			osReadFileQJS = prevRead
			writeCoverageJSONQJS = prevWJSON
			writeLcovQJS = prevLcov
			checkCoverageQJS = prevCheck
		})
		failed, err := RunFrontendQJS(context.Background(), QJSRunOptions{
			RepoRoot: repo, App: "cov", TestFiles: []string{testFile}, WorkingDir: work, TmpRoot: t.TempDir(),
			Coverage: true, CoverageReport: true, CoverageCheck: true, Keep: true,
		})
		if err != nil || failed {
			t.Fatalf("cov ok path: failed=%v err=%v", failed, err)
		}
	})

	t.Run("coverage write json fail", func(t *testing.T) {
		prevBuild := buildFrontendUnitBundleQJS
		buildFrontendUnitBundleQJS = covBundle
		prevPF := preflightCoverageQJS
		preflightCoverageQJS = func(string) error { return nil }
		prevInst := instrumentJSFileQJS
		instrumentJSFileQJS = func(string) error { return nil }
		prevRead := osReadFileQJS
		osReadFileQJS = func(string) ([]byte, error) {
			return []byte(`globalThis.__choysum_test_run__ = async () => ({ total:1,passed:1,failed:0,cases:[{name:'ok',ok:true,durationMs:1}], coverageJSON: "{\"x\":1}" });`), nil
		}
		prevWJSON := writeCoverageJSONQJS
		writeCoverageJSONQJS = func(string, string, string, string, string) error { return os.ErrInvalid }
		t.Cleanup(func() {
			buildFrontendUnitBundleQJS = prevBuild
			preflightCoverageQJS = prevPF
			instrumentJSFileQJS = prevInst
			osReadFileQJS = prevRead
			writeCoverageJSONQJS = prevWJSON
		})
		if _, err := RunFrontendQJS(context.Background(), QJSRunOptions{
			RepoRoot: repo, TestFiles: []string{testFile}, WorkingDir: work, TmpRoot: t.TempDir(), Coverage: true,
		}); err == nil || !strings.Contains(err.Error(), "write coverage json") {
			t.Fatalf("%v", err)
		}
	})

	t.Run("write lcov fail", func(t *testing.T) {
		prevBuild := buildFrontendUnitBundleQJS
		buildFrontendUnitBundleQJS = covBundle
		prevPF := preflightCoverageQJS
		preflightCoverageQJS = func(string) error { return nil }
		prevInst := instrumentJSFileQJS
		instrumentJSFileQJS = func(string) error { return nil }
		prevRead := osReadFileQJS
		osReadFileQJS = func(string) ([]byte, error) {
			return []byte(`globalThis.__choysum_test_run__ = async () => ({ total:1,passed:1,failed:0,cases:[{name:'ok',ok:true,durationMs:1}], coverageJSON: "{\"x\":1}" });`), nil
		}
		prevWJSON := writeCoverageJSONQJS
		writeCoverageJSONQJS = func(string, string, string, string, string) error { return nil }
		prevLcov := writeLcovQJS
		writeLcovQJS = func(context.Context, coverage.ReportOptions) error { return os.ErrInvalid }
		t.Cleanup(func() {
			buildFrontendUnitBundleQJS = prevBuild
			preflightCoverageQJS = prevPF
			instrumentJSFileQJS = prevInst
			osReadFileQJS = prevRead
			writeCoverageJSONQJS = prevWJSON
			writeLcovQJS = prevLcov
		})
		if _, err := RunFrontendQJS(context.Background(), QJSRunOptions{
			RepoRoot: repo, TestFiles: []string{testFile}, WorkingDir: work, TmpRoot: t.TempDir(),
			Coverage: true, CoverageReport: true,
		}); err == nil || !strings.Contains(err.Error(), "WriteLcov") {
			t.Fatalf("%v", err)
		}
	})

	t.Run("check coverage fail", func(t *testing.T) {
		prevBuild := buildFrontendUnitBundleQJS
		buildFrontendUnitBundleQJS = covBundle
		prevPF := preflightCoverageQJS
		preflightCoverageQJS = func(string) error { return nil }
		prevInst := instrumentJSFileQJS
		instrumentJSFileQJS = func(string) error { return nil }
		prevRead := osReadFileQJS
		osReadFileQJS = func(string) ([]byte, error) {
			return []byte(`globalThis.__choysum_test_run__ = async () => ({ total:1,passed:1,failed:0,cases:[{name:'ok',ok:true,durationMs:1}], coverageJSON: "{\"x\":1}" });`), nil
		}
		prevWJSON := writeCoverageJSONQJS
		writeCoverageJSONQJS = func(string, string, string, string, string) error { return nil }
		prevCheck := checkCoverageQJS
		checkCoverageQJS = func(context.Context, coverage.CheckOptions) error { return errors.New("below threshold") }
		t.Cleanup(func() {
			buildFrontendUnitBundleQJS = prevBuild
			preflightCoverageQJS = prevPF
			instrumentJSFileQJS = prevInst
			osReadFileQJS = prevRead
			writeCoverageJSONQJS = prevWJSON
			checkCoverageQJS = prevCheck
		})
		if _, err := RunFrontendQJS(context.Background(), QJSRunOptions{
			RepoRoot: repo, TestFiles: []string{testFile}, WorkingDir: work, TmpRoot: t.TempDir(),
			Coverage: true, CoverageCheck: true, CoverageLines: 90,
		}); err == nil || !strings.Contains(err.Error(), "below threshold") {
			t.Fatalf("%v", err)
		}
	})

	t.Run("junit write fail after run", func(t *testing.T) {
		prevBuild := buildFrontendUnitBundleQJS
		buildFrontendUnitBundleQJS = passingBundle
		prevWrite := osWriteFileQJS
		osWriteFileQJS = func(string, []byte, os.FileMode) error { return os.ErrPermission }
		t.Cleanup(func() {
			buildFrontendUnitBundleQJS = prevBuild
			osWriteFileQJS = prevWrite
		})
		if _, err := RunFrontendQJS(context.Background(), QJSRunOptions{
			RepoRoot: repo, TestFiles: []string{testFile}, WorkingDir: work, TmpRoot: t.TempDir(),
			JUnitPath: filepath.Join(t.TempDir(), "j.xml"),
		}); err == nil || !strings.Contains(err.Error(), "write junit") {
			t.Fatalf("%v", err)
		}
	})

	t.Run("empty working dir defaults", func(t *testing.T) {
		prevBuild := buildFrontendUnitBundleQJS
		buildFrontendUnitBundleQJS = passingBundle
		t.Cleanup(func() { buildFrontendUnitBundleQJS = prevBuild })
		failed, err := RunFrontendQJS(context.Background(), QJSRunOptions{
			RepoRoot: repo, App: "wd", TestFiles: []string{testFile}, WorkingDir: "", TmpRoot: t.TempDir(), Keep: true,
		})
		if err != nil || failed {
			t.Fatalf("empty workingDir: failed=%v err=%v", failed, err)
		}
	})

	t.Run("eval run error", func(t *testing.T) {
		prevBuild := buildFrontendUnitBundleQJS
		buildFrontendUnitBundleQJS = func(opts BundleOptions) (*BundleResult, error) {
			js := `globalThis.__choysum_test_run__ = async () => { throw new Error('eval-fail'); };`
			if err := os.WriteFile(opts.Outfile, []byte(js), 0o644); err != nil {
				return nil, err
			}
			return &BundleResult{JS: js, JSPath: opts.Outfile}, nil
		}
		t.Cleanup(func() { buildFrontendUnitBundleQJS = prevBuild })
		if _, err := RunFrontendQJS(context.Background(), QJSRunOptions{
			RepoRoot: repo, TestFiles: []string{testFile}, WorkingDir: work, TmpRoot: t.TempDir(),
		}); err == nil || !strings.Contains(err.Error(), "fe-qjs: run:") {
			t.Fatalf("%v", err)
		}
	})

	t.Run("vue prepare success path", func(t *testing.T) {
		prevBuild := buildFrontendUnitBundleQJS
		buildFrontendUnitBundleQJS = passingBundle
		prevComp := newFrontendCompilerExecutorQ
		newFrontendCompilerExecutorQ = func(context.Context) (jsexecutor.JsExecutor, error) {
			return stubJsExecutor{}, nil
		}
		prevPrep := prepareVueHostEngineQJS
		prepareVueHostEngineQJS = func(jsengine.JsEngine) error { return nil }
		t.Cleanup(func() {
			buildFrontendUnitBundleQJS = prevBuild
			newFrontendCompilerExecutorQ = prevComp
			prepareVueHostEngineQJS = prevPrep
		})
		failed, err := RunFrontendQJS(context.Background(), QJSRunOptions{
			RepoRoot: repo, TestFiles: []string{testFile}, WorkingDir: work, TmpRoot: t.TempDir(), ForceVue: true, Keep: true,
		})
		if err != nil || failed {
			t.Fatalf("vue ok: failed=%v err=%v", failed, err)
		}
	})
}

func strconvQuote(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
