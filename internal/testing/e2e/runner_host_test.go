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

	path, err := cdp.ResolveChromiumPath()
	if err != nil {
		t.Skip(err.Error())
	}
	oldStart := cdpStart
	cdpStart = func(ctx context.Context, opts cdp.StartOptions) (*cdp.Session, error) {
		h := true
		return cdp.Start(ctx, cdp.StartOptions{ExecPath: path, Headless: &h})
	}
	defer func() { cdpStart = oldStart }()

	oldEng := newQJSEngine
	newQJSEngine = func() (jsengine.JsEngine, error) { return nil, errors.New("engine boom") }
	defer func() { newQJSEngine = oldEng }()

	err = runE2EHost(context.Background(), RunOptions{WorkDir: repoRoot, Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}, runDir, "http://x", runtimePath, []string{spec})
	if err == nil || !strings.Contains(err.Error(), "engine boom") {
		t.Fatalf("got %v", err)
	}
}

func TestRunE2EHostChromeSmoke(t *testing.T) {
	path, err := cdp.ResolveChromiumPath()
	if err != nil {
		t.Skip(err.Error())
	}
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
	cdpStart = func(ctx context.Context, opts cdp.StartOptions) (*cdp.Session, error) {
		h := true
		return cdp.Start(ctx, cdp.StartOptions{ExecPath: path, Headless: &h})
	}
	defer func() { cdpStart = oldStart }()

	var stdout, stderr bytes.Buffer
	err = runE2EHost(context.Background(), RunOptions{
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
