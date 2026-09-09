// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package e2e

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/internal/testing/e2e/cdp"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsengine/quickjsengine"
	"github.com/choysum-dev/choysum/pkg/jsengine/scripts/choysumtest"
)

func TestRunE2EHostEmptySpecs(t *testing.T) {
	if err := runE2EHost(context.Background(), RunOptions{}, "", "", "", nil); err != nil {
		t.Fatalf("empty specs: %v", err)
	}
	// nil ctx + empty WorkDir (cwd fallback) with empty specs still no-ops after ctx defaulting.
	if err := runE2EHost(nil, RunOptions{}, "", "", "", nil); err != nil {
		t.Fatalf("nil ctx empty: %v", err)
	}
}

func TestRunE2EHostChromeSmokeNilStdout(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))
	runDir := t.TempDir()
	runtimePath := filepath.Join(runDir, "runtime.json")
	if err := os.WriteFile(runtimePath, []byte(`{"baseURL":"http://example.test"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	spec := filepath.Join(runDir, "pass2.spec.ts")
	if err := os.WriteFile(spec, []byte(`
import { test } from '@choysum/e2e';
test('pass2', async () => {});
`), 0o644); err != nil {
		t.Fatal(err)
	}

	oldStart := cdpStart
	cdpStart = e2eStartChromiumOrSkip(t)
	defer func() { cdpStart = oldStart }()

	// Stdout and Stderr nil → os.Stdout / os.Stderr (stderr nil guard from review).
	err := runE2EHost(nil, RunOptions{
		WorkDir: repoRoot,
		Stdout:  nil,
		Stderr:  nil,
	}, runDir, "http://example.test", runtimePath, []string{spec})
	if err != nil {
		t.Fatalf("runE2EHost: %v", err)
	}
}

func TestRunE2EHostEmptyWorkDirUsesCwd(t *testing.T) {
	runDir := t.TempDir()
	runtimePath := filepath.Join(runDir, "runtime.json")
	_ = os.WriteFile(runtimePath, []byte(`{}`), 0o644)
	spec := filepath.Join(runDir, "x.spec.ts")
	_ = os.WriteFile(spec, []byte(`import { test } from '@choysum/e2e'; test('x', async () => {});`), 0o644)

	old := cdpStart
	cdpStart = func(ctx context.Context, opts cdp.StartOptions) (*cdp.Session, error) {
		return nil, errors.New("cdp boom")
	}
	defer func() { cdpStart = old }()

	// WorkDir "" → Getwd(); still reaches cdpStart after bundling from cwd as repo root.
	err := runE2EHost(context.Background(), RunOptions{WorkDir: "", Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}, runDir, "http://x", runtimePath, []string{spec})
	// May fail at bundle (cwd not repo) or cdp boom — either exercises empty WorkDir branch.
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRunE2EHostCDPStartError(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))
	runDir := t.TempDir()
	runtimePath := filepath.Join(runDir, "runtime.json")
	if err := os.WriteFile(runtimePath, []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	spec := filepath.Join(runDir, "x.spec.ts")
	if err := os.WriteFile(spec, []byte(`import { test } from '@choysum/e2e'; test('x', async () => {});`), 0o644); err != nil {
		t.Fatal(err)
	}
	old := cdpStart
	cdpStart = func(ctx context.Context, opts cdp.StartOptions) (*cdp.Session, error) {
		return nil, errors.New("cdp boom")
	}
	defer func() { cdpStart = old }()
	err := runE2EHost(context.Background(), RunOptions{WorkDir: repoRoot, Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}, runDir, "http://x", runtimePath, []string{spec})
	if err == nil || !strings.Contains(err.Error(), "cdp boom") {
		t.Fatalf("got %v", err)
	}
}

func TestRunE2EHostEngineError(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))
	runDir := t.TempDir()
	runtimePath := filepath.Join(runDir, "runtime.json")
	_ = os.WriteFile(runtimePath, []byte(`{}`), 0o644)
	spec := filepath.Join(runDir, "x.spec.ts")
	_ = os.WriteFile(spec, []byte(`import { test } from '@choysum/e2e'; test('x', async () => {});`), 0o644)

	oldStart := cdpStart
	cdpStart = func(ctx context.Context, opts cdp.StartOptions) (*cdp.Session, error) {
		return &cdp.Session{}, nil
	}
	defer func() { cdpStart = oldStart }()

	oldEng := newQJSEngine
	newQJSEngine = func() (jsengine.JsEngine, error) { return nil, errors.New("engine boom") }
	defer func() { newQJSEngine = oldEng }()

	err := runE2EHost(context.Background(), RunOptions{WorkDir: repoRoot, Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}, runDir, "http://x", runtimePath, []string{spec})
	if err == nil || !strings.Contains(err.Error(), "engine boom") {
		t.Fatalf("got %v", err)
	}
}

func TestRunE2EHostChromeSmoke(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))
	runDir := t.TempDir()
	runtimePath := filepath.Join(runDir, "runtime.json")
	if err := os.WriteFile(runtimePath, []byte(`{"baseURL":"http://example.test"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	spec := filepath.Join(runDir, "pass.spec.ts")
	// Avoid page ops; beforeEach still opens a blank page via host.newPage.
	if err := os.WriteFile(spec, []byte(`
import { test } from '@choysum/e2e';
test('pass', async () => {});
`), 0o644); err != nil {
		t.Fatal(err)
	}

	oldStart := cdpStart
	cdpStart = e2eStartChromiumOrSkip(t)
	defer func() { cdpStart = oldStart }()

	var stdout, stderr bytes.Buffer
	err := runE2EHost(context.Background(), RunOptions{
		WorkDir: repoRoot,
		Stdout:  &stdout,
		Stderr:  &stderr,
	}, runDir, "http://example.test", runtimePath, []string{spec})
	if err != nil {
		t.Fatalf("runE2EHost: %v\nstderr=%s\nstdout=%s", err, stderr.String(), stdout.String())
	}
	if !strings.Contains(stderr.String(), "e2e-qjs ok") {
		t.Fatalf("stderr=%s", stderr.String())
	}
}

func TestEvalE2ETestRunMissing(t *testing.T) {
	eng, err := newQJSEngine()
	if err != nil {
		t.Fatal(err)
	}
	_, err = evalE2ETestRun(context.Background(), eng)
	if err == nil || !strings.Contains(err.Error(), "__choysum_test_run__") {
		t.Fatalf("got %v", err)
	}
}

func TestEvalE2ETestRunErrorPaths(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := evalE2ETestRun(ctx, nil); err == nil {
		t.Fatal("expected canceled ctx error")
	}
	if _, err := evalE2ETestRun(nil, nil); err == nil || !strings.Contains(err.Error(), "expected QuickjsEngine") {
		t.Fatalf("nil engine: %v", err)
	}

	engine, err := newQJSEngine()
	if err != nil {
		t.Fatal(err)
	}
	qjs := engine.(*quickjsengine.QuickjsEngine)

	v := qjs.Ctx.Eval(`globalThis.__choysum_test_run__ = async () => undefined`)
	if v.IsException() {
		t.Fatal(qjs.Ctx.Exception())
	}
	v.Free()
	if _, err := evalE2ETestRun(context.Background(), engine); err == nil || !strings.Contains(err.Error(), "empty report") {
		t.Fatalf("empty: %v", err)
	}

	v = qjs.Ctx.Eval(`globalThis.__choysum_test_run__ = async () => 'not-json'`)
	if v.IsException() {
		t.Fatal(qjs.Ctx.Exception())
	}
	v.Free()
	if _, err := evalE2ETestRun(context.Background(), engine); err == nil || !strings.Contains(err.Error(), "parse report") {
		t.Fatalf("parse: %v", err)
	}

	v = qjs.Ctx.Eval(`globalThis.__choysum_test_run__ = async () => ({ total:1, passed:1, failed:0, cases:[{name:'ok',ok:true,durationMs:1}] })`)
	if v.IsException() {
		t.Fatal(qjs.Ctx.Exception())
	}
	v.Free()
	report, err := evalE2ETestRun(nil, engine)
	if err != nil {
		t.Fatal(err)
	}
	if report.Total != 1 || report.Passed != 1 {
		t.Fatalf("report=%+v", report)
	}
}

func TestEvalE2ETestRunWithChoysumTest(t *testing.T) {
	engine, err := newQJSEngine()
	if err != nil {
		t.Fatal(err)
	}
	if err := engine.Load([]*jsengine.JsScript{
		{FileName: "scripts/choysumtest/choysumtest.js", Content: choysumtest.ChoysumTestScript},
		{FileName: "inline.js", Content: `test('inline', () => {});`},
	}); err != nil {
		t.Fatal(err)
	}
	report, err := evalE2ETestRun(context.Background(), engine)
	if err != nil {
		t.Fatal(err)
	}
	if report.Failed != 0 || report.Passed < 1 {
		t.Fatalf("report=%+v", report)
	}
}

func TestRunE2EHostMissingRuntime(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))
	runDir := t.TempDir()
	spec := filepath.Join(runDir, "x.spec.ts")
	if err := os.WriteFile(spec, []byte(`import { test } from '@choysum/e2e'; test('x', async () => {});`), 0o644); err != nil {
		t.Fatal(err)
	}
	err := runE2EHost(context.Background(), RunOptions{WorkDir: repoRoot}, runDir, "", filepath.Join(runDir, "missing.json"), []string{spec})
	if err == nil || !strings.Contains(err.Error(), "read runtime") {
		t.Fatalf("expected read runtime error, got %v", err)
	}
}

func TestRunE2EHostFailedCase(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))
	runDir := t.TempDir()
	runtimePath := filepath.Join(runDir, "runtime.json")
	_ = os.WriteFile(runtimePath, []byte(`{}`), 0o644)
	spec := filepath.Join(runDir, "fail.spec.ts")
	_ = os.WriteFile(spec, []byte(`
import { test } from '@choysum/e2e';
test('fail', async () => { throw new Error('boom'); });
`), 0o644)

	oldStart := cdpStart
	cdpStart = e2eStartChromiumOrSkip(t)
	defer func() { cdpStart = oldStart }()

	var stderr bytes.Buffer
	err := runE2EHost(context.Background(), RunOptions{
		WorkDir: repoRoot,
		Stdout:  &bytes.Buffer{},
		Stderr:  &stderr,
	}, runDir, "http://example.test", runtimePath, []string{spec})
	if err == nil || !strings.Contains(err.Error(), "failed") {
		t.Fatalf("expected failure, got %v", err)
	}
	if !strings.Contains(stderr.String(), "e2e-qjs failed") {
		t.Fatalf("stderr=%s", stderr.String())
	}
}

func TestRunE2EHostMkdirAndEntryErrors(t *testing.T) {
	runDirParent := t.TempDir()
	blocker := filepath.Join(runDirParent, "notadir")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Dir(runtimePath) is a file → MkdirAll(.e2e) fails.
	runtimePath := filepath.Join(blocker, "runtime.json")
	err := runE2EHost(context.Background(), RunOptions{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}, "", "http://x", runtimePath, []string{"x.spec.ts"})
	if err == nil || !strings.Contains(err.Error(), "mkdir") {
		t.Fatalf("mkdir: %v", err)
	}

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))
	runDir := t.TempDir()
	runtimePath = filepath.Join(runDir, "runtime.json")
	_ = os.WriteFile(runtimePath, []byte(`{}`), 0o644)
	oldAbs := e2eFilepathAbs
	e2eFilepathAbs = func(path string) (string, error) { return "", errors.New("abs boom") }
	defer func() { e2eFilepathAbs = oldAbs }()
	err = runE2EHost(context.Background(), RunOptions{WorkDir: repoRoot, Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}, runDir, "http://x", runtimePath, []string{filepath.Join(runDir, "x.spec.ts")})
	if err == nil || !strings.Contains(err.Error(), "resolve spec path") {
		t.Fatalf("entry: %v", err)
	}
}

func TestRunE2EHostBundleAndInstallErrors(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))
	runDir := t.TempDir()
	runtimePath := filepath.Join(runDir, "runtime.json")
	_ = os.WriteFile(runtimePath, []byte(`{}`), 0o644)
	spec := filepath.Join(runDir, "bad.spec.ts")
	// Spec that cannot bundle (syntax error).
	_ = os.WriteFile(spec, []byte(`import { from '@choysum/e2e';`), 0o644)

	err := runE2EHost(context.Background(), RunOptions{
		WorkDir: repoRoot,
		Stdout:  &bytes.Buffer{},
		Stderr:  &bytes.Buffer{},
	}, runDir, "http://x", runtimePath, []string{spec})
	if err == nil {
		t.Fatal("expected bundle error")
	}
}

func TestRunE2EHostLoadError(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))
	runDir := t.TempDir()
	runtimePath := filepath.Join(runDir, "runtime.json")
	_ = os.WriteFile(runtimePath, []byte(`{`), 0o644) // invalid runtime JSON → Install fails
	spec := filepath.Join(runDir, "x.spec.ts")
	_ = os.WriteFile(spec, []byte(`import { test } from '@choysum/e2e'; test('x', async () => {});`), 0o644)

	oldStart := cdpStart
	cdpStart = e2eStartChromiumOrSkip(t)
	defer func() { cdpStart = oldStart }()

	err := runE2EHost(context.Background(), RunOptions{
		WorkDir: repoRoot,
		Stdout:  &bytes.Buffer{},
		Stderr:  &bytes.Buffer{},
	}, runDir, "http://x", runtimePath, []string{spec})
	if err == nil || !strings.Contains(err.Error(), "parse runtime json") {
		t.Fatalf("expected install/parse error, got %v", err)
	}
}

func TestEvalE2ETestRunEvalException(t *testing.T) {
	engine, err := newQJSEngine()
	if err != nil {
		t.Fatal(err)
	}
	qjs := engine.(*quickjsengine.QuickjsEngine)
	// Break Eval by making __choysum_test_run__ a non-callable that still passes typeof check...
	// Use a getter that throws when invoked via the async IIFE path:
	v := qjs.Ctx.Eval(`Object.defineProperty(globalThis, '__choysum_test_run__', {
  get() { throw new Error('getter boom'); }
})`)
	if v.IsException() {
		t.Fatal(qjs.Ctx.Exception())
	}
	v.Free()
	_, err = evalE2ETestRun(context.Background(), engine)
	if err == nil {
		t.Fatal("expected eval error")
	}
}

func TestEvalE2ETestRunAwaitException(t *testing.T) {
	engine, err := newQJSEngine()
	if err != nil {
		t.Fatal(err)
	}
	qjs := engine.(*quickjsengine.QuickjsEngine)
	v := qjs.Ctx.Eval(`globalThis.__choysum_test_run__ = async () => { throw new Error('await boom'); }`)
	if v.IsException() {
		t.Fatal(qjs.Ctx.Exception())
	}
	v.Free()
	_, err = evalE2ETestRun(context.Background(), engine)
	if err == nil || !strings.Contains(err.Error(), "e2e host: run:") {
		t.Fatalf("expected await/run error, got %v", err)
	}
}

func TestRunE2EHostFailedCaseNilStderr(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))
	runDir := t.TempDir()
	runtimePath := filepath.Join(runDir, "runtime.json")
	_ = os.WriteFile(runtimePath, []byte(`{}`), 0o644)
	spec := filepath.Join(runDir, "fail2.spec.ts")
	_ = os.WriteFile(spec, []byte(`
import { test } from '@choysum/e2e';
test('fail2', async () => { throw new Error('boom2'); });
`), 0o644)

	oldStart := cdpStart
	cdpStart = e2eStartChromiumOrSkip(t)
	defer func() { cdpStart = oldStart }()

	err := runE2EHost(context.Background(), RunOptions{
		WorkDir: repoRoot,
		Stdout:  &bytes.Buffer{},
		Stderr:  nil, // exercises errOut = os.Stderr on failure path
	}, runDir, "http://example.test", runtimePath, []string{spec})
	if err == nil || !strings.Contains(err.Error(), "failed") {
		t.Fatalf("expected failure, got %v", err)
	}
}

func TestRunE2EHostNonQuickjsEngine(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))
	runDir := t.TempDir()
	runtimePath := filepath.Join(runDir, "runtime.json")
	_ = os.WriteFile(runtimePath, []byte(`{}`), 0o644)
	spec := filepath.Join(runDir, "x.spec.ts")
	_ = os.WriteFile(spec, []byte(`import { test } from '@choysum/e2e'; test('x', async () => {});`), 0o644)

	oldStart := cdpStart
	oldEng := newQJSEngine
	cdpStart = func(ctx context.Context, opts cdp.StartOptions) (*cdp.Session, error) {
		return &cdp.Session{}, nil
	}
	newQJSEngine = func() (jsengine.JsEngine, error) { return fakeE2EEngine{}, nil }
	defer func() {
		cdpStart = oldStart
		newQJSEngine = oldEng
	}()

	err := runE2EHost(context.Background(), RunOptions{
		WorkDir: repoRoot,
		Stdout:  &bytes.Buffer{},
		Stderr:  &bytes.Buffer{},
	}, runDir, "http://x", runtimePath, []string{spec})
	if err == nil || !strings.Contains(err.Error(), "expected QuickjsEngine") {
		t.Fatalf("got %v", err)
	}
}

func TestRunE2EHostBootstrapTimersLoadEvalHooks(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))
	runDir := t.TempDir()
	runtimePath := filepath.Join(runDir, "runtime.json")
	_ = os.WriteFile(runtimePath, []byte(`{}`), 0o644)
	spec := filepath.Join(runDir, "x.spec.ts")
	_ = os.WriteFile(spec, []byte(`import { test } from '@choysum/e2e'; test('x', async () => {});`), 0o644)

	oldStart := cdpStart
	oldEng := newQJSEngine
	oldLoad := engineLoadHook
	oldEval := evalE2ETestRunHook
	oldTimers := bootstrapTimersHook
	t.Cleanup(func() {
		cdpStart = oldStart
		newQJSEngine = oldEng
		engineLoadHook = oldLoad
		evalE2ETestRunHook = oldEval
		bootstrapTimersHook = oldTimers
	})
	cdpStart = e2eStartChromiumOrSkip(t)

	bootstrapTimersHook = func(qjs *quickjsengine.QuickjsEngine) bool { return false }
	err := runE2EHost(context.Background(), RunOptions{
		WorkDir: repoRoot, Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{},
	}, runDir, "http://x", runtimePath, []string{spec})
	if err == nil || !strings.Contains(err.Error(), "BootstrapTimers failed") {
		t.Fatalf("timers: %v", err)
	}

	bootstrapTimersHook = oldTimers
	engineLoadHook = func(engine jsengine.JsEngine, scripts []*jsengine.JsScript) error {
		return errors.New("load boom")
	}
	err = runE2EHost(context.Background(), RunOptions{
		WorkDir: repoRoot, Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{},
	}, runDir, "http://x", runtimePath, []string{spec})
	if err == nil || !strings.Contains(err.Error(), "load boom") {
		t.Fatalf("load: %v", err)
	}

	engineLoadHook = oldLoad
	evalE2ETestRunHook = func(ctx context.Context, engine jsengine.JsEngine) (*e2eRunReport, error) {
		return nil, errors.New("eval boom")
	}
	err = runE2EHost(context.Background(), RunOptions{
		WorkDir: repoRoot, Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{},
	}, runDir, "http://x", runtimePath, []string{spec})
	if err == nil || !strings.Contains(err.Error(), "eval boom") {
		t.Fatalf("eval: %v", err)
	}
}

func TestEvalE2ETestRunSyncException(t *testing.T) {
	engine, err := newQJSEngine()
	if err != nil {
		t.Fatal(err)
	}
	old := e2eTestRunScript
	e2eTestRunScript = `throw new Error('sync eval boom')`
	t.Cleanup(func() { e2eTestRunScript = old })
	_, err = evalE2ETestRun(context.Background(), engine)
	if err == nil || !strings.Contains(err.Error(), "sync eval boom") {
		t.Fatalf("got %v", err)
	}
}

type fakeE2EEngine struct{}

func (fakeE2EEngine) Load([]*jsengine.JsScript) error { return nil }
func (fakeE2EEngine) Execute(context.Context, *jsengine.JsRequest) (*jsengine.JsResponse, error) {
	return nil, nil
}
func (fakeE2EEngine) Close() error { return nil }

func e2eStartChromiumOrSkip(t *testing.T) func(ctx context.Context, opts cdp.StartOptions) (*cdp.Session, error) {
	t.Helper()
	return func(ctx context.Context, opts cdp.StartOptions) (*cdp.Session, error) {
		h := true
		seen := map[string]bool{}
		var cands []string
		add := func(p string) {
			if p == "" || seen[p] {
				return
			}
			if st, err := os.Stat(p); err != nil || st.IsDir() {
				return
			}
			seen[p] = true
			cands = append(cands, p)
		}
		if p, err := cdp.ResolveChromiumPath(); err == nil {
			add(p)
		}
		for _, p := range []string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
			"/usr/bin/google-chrome",
			"/usr/bin/chromium",
			"/usr/bin/chromium-browser",
		} {
			add(p)
		}
		var last error
		for _, path := range cands {
			sess, err := cdp.Start(ctx, cdp.StartOptions{ExecPath: path, Headless: &h})
			if err == nil {
				return sess, nil
			}
			last = err
		}
		if last == nil {
			last = errors.New("chromium unavailable")
		}
		t.Skipf("chromium start failed: %v", last)
		return nil, last
	}
}
