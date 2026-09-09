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

type fakeE2EEngine struct{}

func (fakeE2EEngine) Load([]*jsengine.JsScript) error                          { return nil }
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

