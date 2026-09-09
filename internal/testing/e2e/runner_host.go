// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/choysum-dev/choysum/internal/testing/e2e/cdp"
	"github.com/choysum-dev/choysum/internal/testing/e2e/pagehost"
	"github.com/choysum-dev/choysum/internal/testing/tap"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsengine/quickjsengine"
	"github.com/choysum-dev/choysum/pkg/jsengine/scripts/choysumtest"
	xfmt "golang.org/x/exp/errors/fmt"
)

type e2eCaseReport struct {
	Name       string `json:"name"`
	OK         bool   `json:"ok"`
	DurationMs int    `json:"durationMs"`
	Error      *struct {
		Message string `json:"message"`
		Stack   string `json:"stack"`
	} `json:"error"`
}

type e2eRunReport struct {
	Total  int             `json:"total"`
	Passed int             `json:"passed"`
	Failed int             `json:"failed"`
	Cases  []e2eCaseReport `json:"cases"`
}

var (
	cdpStart     = cdp.Start
	newQJSEngine = func() (jsengine.JsEngine, error) { return quickjsengine.NewFactory()() }
)

// runE2EHost executes QJS e2e specs via chromedp + choysumtest + @choysum/e2e.
func runE2EHost(ctx context.Context, opts RunOptions, specsDir string, baseURL string, runtimePath string, qjsSpecFiles []string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if len(qjsSpecFiles) == 0 {
		return nil
	}
	runDir := filepath.Dir(runtimePath)
	repoRoot := strings.TrimSpace(opts.WorkDir)
	if repoRoot == "" {
		wd, _ := os.Getwd()
		repoRoot = wd
	}

	bundleDir := filepath.Join(runDir, ".e2e")
	if err := os.MkdirAll(bundleDir, 0o755); err != nil {
		return xfmt.Errorf("e2e host: mkdir: %w", err)
	}
	entryPath := filepath.Join(bundleDir, "entry.js")
	if err := WriteE2EEntry(entryPath, qjsSpecFiles); err != nil {
		return err
	}
	outfile := filepath.Join(bundleDir, "e2e.bundle.js")
	bundle, err := BuildE2EBundle(E2EBundleOptions{
		RepoRoot:   repoRoot,
		EntryPath:  entryPath,
		Outfile:    outfile,
		WorkingDir: repoRoot,
		RunDir:     runDir,
	})
	if err != nil {
		return err
	}

	runtimeRaw, err := os.ReadFile(runtimePath)
	if err != nil {
		return xfmt.Errorf("e2e host: read runtime: %w", err)
	}

	// Headless by default; set CHOYSUM_E2E_HEADED=1 only for local debugging.
	session, err := cdpStart(ctx, cdp.StartOptions{})
	if err != nil {
		return err
	}
	defer session.Close()
	// Do not engine.Close() here: after failed async host calls QuickJS can still
	// hold promise objects and JS_FreeRuntime aborts the process (gc_obj_list).
	// The choysum CLI process exits after the e2e run.

	engine, err := newQJSEngine()
	if err != nil {
		return xfmt.Errorf("e2e host: engine: %w", err)
	}

	qjs, ok := engine.(*quickjsengine.QuickjsEngine)
	if !ok || qjs == nil || qjs.Ctx == nil {
		return xfmt.Errorf("e2e host: expected QuickjsEngine")
	}
	if !qjs.Ctx.BootstrapTimers() {
		return xfmt.Errorf("e2e host: BootstrapTimers failed")
	}

	host, err := pagehost.Install(engine, session, string(runtimeRaw))
	if err != nil {
		return err
	}
	defer host.Drain()

	if err := engine.Load([]*jsengine.JsScript{
		{FileName: "scripts/choysumtest/choysumtest.js", Content: choysumtest.ChoysumTestScript},
		{FileName: bundle.JSPath, Content: bundle.JS},
	}); err != nil {
		return xfmt.Errorf("e2e host: load: %w", err)
	}

	report, err := evalE2ETestRun(ctx, engine)
	if err != nil {
		return err
	}

	out := opts.Stdout
	if out == nil {
		out = os.Stdout
	}
	errOut := opts.Stderr
	if errOut == nil {
		errOut = os.Stderr
	}
	tapReport := &tap.Report{Total: report.Total}
	for _, c := range report.Cases {
		tc := tap.Case{Name: c.Name, OK: c.OK}
		if c.Error != nil {
			tc.Error = &tap.CaseError{Message: c.Error.Message, Stack: c.Error.Stack}
		}
		tapReport.Cases = append(tapReport.Cases, tc)
	}
	tap.Write(out, tapReport)

	if report.Failed > 0 {
		shotDir := filepath.Join(bundleDir, "screenshots")
		_ = os.MkdirAll(shotDir, 0o755)
		// Capture the current tab; NewPage() would navigate to about:blank first.
		_ = session.ScreenshotCurrent(filepath.Join(shotDir, "failure.png"))
		fmt.Fprintf(errOut, "# e2e-qjs failed (%d/%d) baseURL=%s specsDir=%s\n", report.Failed, report.Total, baseURL, specsDir)
		return xfmt.Errorf("e2e host: %d failed", report.Failed)
	}
	fmt.Fprintf(errOut, "# e2e-qjs ok (%d) (%s)\n", report.Total, time.Now().Format(time.RFC3339))
	return nil
}

func evalE2ETestRun(ctx context.Context, engine jsengine.JsEngine) (*e2eRunReport, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	qjs, ok := engine.(*quickjsengine.QuickjsEngine)
	if !ok || qjs == nil || qjs.Ctx == nil {
		return nil, xfmt.Errorf("e2e host: run: expected QuickjsEngine")
	}
	restore := qjs.SwapExecContext(ctx)
	defer restore()

	script := `(async () => {
  if (typeof globalThis.__choysum_test_run__ !== 'function') {
    throw new Error('__choysum_test_run__ missing');
  }
  const report = await globalThis.__choysum_test_run__();
  return JSON.stringify(report);
})()`
	// Use Go Await (not EvalAwait): host methods resolve via ctx.Schedule from
	// goroutines (waitForResponse/delay). EvalAwait's C poll does not ProcessJobs.
	val := qjs.Ctx.Eval(script)
	if val.IsException() {
		return nil, xfmt.Errorf("e2e host: run: %v", qjs.Ctx.Exception())
	}
	val = qjs.Ctx.Await(val)
	defer val.Free()
	if val.IsException() {
		return nil, xfmt.Errorf("e2e host: run: %v", qjs.Ctx.Exception())
	}
	raw := strings.TrimSpace(val.String())
	if raw == "" || raw == "null" || raw == "undefined" {
		return nil, xfmt.Errorf("e2e host: run: empty report")
	}
	var report e2eRunReport
	if err := json.Unmarshal([]byte(raw), &report); err != nil {
		return nil, xfmt.Errorf("e2e host: parse report: %w raw=%s", err, raw)
	}
	return &report, nil
}
