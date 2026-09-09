// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package frontend

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/buke/quickjs-go"
	"github.com/choysum-dev/choysum/internal/testing/coverage"
	"github.com/choysum-dev/choysum/internal/testing/tap"
	testingpathing "github.com/choysum-dev/choysum/internal/testing/tmpdir"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsengine/quickjsengine"
	"github.com/choysum-dev/choysum/pkg/jsengine/scripts/choysumtest"
	"github.com/choysum-dev/choysum/pkg/jsexecutor"
	"github.com/choysum-dev/choysum/pkg/scope"
	xfmt "golang.org/x/exp/errors/fmt"
)

// Test seams for rare OS / engine / coverage failures (overridden in unit tests).
var (
	osGetwdQJS                   = os.Getwd
	osMkdirAllQJS                = os.MkdirAll
	osRemoveAllQJS               = os.RemoveAll
	osReadFileQJS                = os.ReadFile
	osWriteFileQJS               = os.WriteFile
	jsonMarshalQJS               = json.Marshal
	xmlMarshalIndentQJS          = xml.MarshalIndent
	resolveTestingTmpDirQJS      = testingpathing.ResolveTestingTmpDirFromContext
	discoverFrontendTestsQJS     = DiscoverFrontendTests
	writeFrontendTestsEntryQJS   = WriteFrontendTestsEntry
	buildFrontendUnitBundleQJS   = BuildFrontendUnitBundle
	newFrontendCompilerExecutorQ = newFrontendCompilerExecutor
	prepareVueHostEngineQJS      = PrepareVueHostEngine
	bootstrapTimersQJS           = func(e *quickjsengine.QuickjsEngine) bool { return e.Ctx.BootstrapTimers() }
	newQuickJSEngineQJS          = func() (jsengine.JsEngine, error) { return quickjsengine.NewFactory()() }
	preflightCoverageQJS         = coverage.PreflightInstrumentationPrerequisites
	instrumentJSFileQJS          = coverage.InstrumentJSFile
	writeCoverageJSONQJS         = coverage.WriteCoverageJSONWithRunIDAndTmpRoot
	writeLcovQJS                 = coverage.WriteLcov
	checkCoverageQJS             = coverage.CheckCoverage
	newCompilerExecutorFn        = func(runtimeScope scope.Scope) (jsexecutor.JsExecutor, error) {
		return jsexecutor.NewCompilerExecutor(runtimeScope)
	}
)

// QJSRunOptions configures a QuickJS FE unit run (discover → bundle → host → coverage).
type QJSRunOptions struct {
	RepoRoot           string
	App                string
	JUnitPath          string
	Pattern            string
	Coverage           bool
	CoverageReport     bool
	CoverageCheck      bool
	CoverageReportDir  string
	CoverageLines      int
	CoverageFunctions  int
	CoverageBranches   int
	CoverageStatements int
	TmpRoot            string
	Keep               bool
	// TestFiles overrides DiscoverFrontendTests when non-empty (fixtures / go tests).
	TestFiles []string
	// WorkingDir is passed to esbuild AbsWorkingDir (defaults to RepoRoot).
	WorkingDir string
	// ForceVue forces Vue host bundling even when no .vue import is detected.
	ForceVue bool
}

type qjsCaseReport struct {
	Name       string `json:"name"`
	OK         bool   `json:"ok"`
	DurationMs int    `json:"durationMs"`
	Error      *struct {
		Message string `json:"message"`
		Stack   string `json:"stack"`
	} `json:"error"`
}

type qjsRunReport struct {
	Total        int             `json:"total"`
	Passed       int             `json:"passed"`
	Failed       int             `json:"failed"`
	Cases        []qjsCaseReport `json:"cases"`
	CoverageJSON *string         `json:"coverageJSON"`
}

// RunFrontendQJS runs FE unit tests on QuickJS + choysumtest (+ Vue host when needed).
func RunFrontendQJS(ctx context.Context, opts QJSRunOptions) (bool, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return true, err
	}

	repoRoot := strings.TrimSpace(opts.RepoRoot)
	if repoRoot == "" {
		wd, _ := osGetwdQJS()
		repoRoot = wd
	}
	app := strings.TrimSpace(opts.App)
	if app == "" {
		app = "fe"
	}

	testFiles := opts.TestFiles
	if len(testFiles) == 0 {
		discovered, err := discoverFrontendTestsQJS(repoRoot, app)
		if err != nil {
			return true, err
		}
		testFiles = discovered
	}
	if len(testFiles) == 0 {
		fmt.Fprintf(os.Stderr, "# fe-qjs %s: no tests\n", app)
		return false, nil
	}

	workspaceTmpDir, err := resolveTestingTmpDirQJS(ctx, repoRoot, testingpathing.EffectiveCLITestTmpRoot(ctx, opts.TmpRoot), "frontend")
	if err != nil {
		return true, xfmt.Errorf("fe-qjs: resolve tmp dir: %w", err)
	}
	runDir := filepath.Join(workspaceTmpDir, "fe-qjs", sanitizeFrontendAppToken(app))
	if err := osMkdirAllQJS(runDir, 0o755); err != nil {
		return true, xfmt.Errorf("fe-qjs: mkdir: %w", err)
	}
	if !opts.Keep {
		defer func() { _ = osRemoveAllQJS(runDir) }()
	} else {
		fmt.Fprintf(os.Stderr, "choysum test: kept frontend qjs dir: %s\n", runDir)
	}

	entryPath := filepath.Join(runDir, "entry.ts")
	if err := writeFrontendTestsEntryQJS(entryPath, testFiles); err != nil {
		return true, err
	}
	outJS := filepath.Join(runDir, "bundle.js")
	needVue := opts.ForceVue || frontendTestsNeedVue(testFiles)

	var executor jsexecutor.JsExecutor
	if needVue {
		ex, err := newFrontendCompilerExecutorQ(ctx)
		if err != nil {
			return true, err
		}
		executor = ex
		defer func() { _ = executor.Stop() }()
	}

	workingDir := strings.TrimSpace(opts.WorkingDir)
	if workingDir == "" {
		workingDir = repoRoot
	}

	fmt.Fprintf(os.Stderr, "# fe-qjs %s (%d files, vue=%v)\n", app, len(testFiles), needVue)
	bundle, err := buildFrontendUnitBundleQJS(BundleOptions{
		RepoRoot:   repoRoot,
		EntryPath:  entryPath,
		Outfile:    outJS,
		Sourcemap:  opts.Coverage || opts.CoverageReport,
		WorkingDir: workingDir,
		Vue:        needVue,
		JsExecutor: executor,
		CacheDir:   filepath.Join(workspaceTmpDir, "esm-cache"),
	})
	if err != nil {
		return true, xfmt.Errorf("fe-qjs: bundle: %w", err)
	}

	if opts.Coverage {
		if err := preflightCoverageQJS(repoRoot); err != nil {
			return true, err
		}
		if err := instrumentJSFileQJS(bundle.JSPath); err != nil {
			return true, xfmt.Errorf("fe-qjs: instrument: %w", err)
		}
		raw, readErr := osReadFileQJS(bundle.JSPath)
		if readErr != nil {
			return true, xfmt.Errorf("fe-qjs: read instrumented: %w", readErr)
		}
		bundle.JS = string(raw)
	}

	engine, err := newQuickJSEngineQJS()
	if err != nil {
		return true, xfmt.Errorf("fe-qjs: engine: %w", err)
	}
	defer func() { _ = engine.Close() }()

	if needVue {
		if err := prepareVueHostEngineQJS(engine); err != nil {
			return true, err
		}
	} else {
		if err := InstallMinimalConsole(engine); err != nil {
			return true, err
		}
		if qjs, ok := engine.(*quickjsengine.QuickjsEngine); ok {
			if !bootstrapTimersQJS(qjs) {
				return true, xfmt.Errorf("frontend host: BootstrapTimers failed")
			}
		}
	}

	if err := engine.Load([]*jsengine.JsScript{
		{FileName: "scripts/choysumtest/choysumtest.js", Content: choysumtest.ChoysumTestScript},
		{FileName: bundle.JSPath, Content: bundle.JS},
	}); err != nil {
		return true, xfmt.Errorf("fe-qjs: load: %w", err)
	}

	report, err := evalChoysumTestRun(ctx, engine, opts.Pattern)
	if err != nil {
		return true, err
	}

	failed := report.Failed > 0
	writeQJSTap(os.Stdout, report)

	if opts.Coverage && report.CoverageJSON != nil && strings.TrimSpace(*report.CoverageJSON) != "" {
		runID := fmt.Sprintf("fe-qjs-%s-%d", sanitizeFrontendAppToken(app), time.Now().UnixNano())
		if err := writeCoverageJSONQJS(repoRoot, app, runID, *report.CoverageJSON, workspaceTmpDir); err != nil {
			return true, xfmt.Errorf("fe-qjs: write coverage json: %w", err)
		}
		// Align with Vitest/CI: Codecov uploads $CoverageReportDir/fe/<app>/lcov.info.
		reportDir := strings.TrimSpace(opts.CoverageReportDir)
		if reportDir == "" {
			reportDir = filepath.Join(workspaceTmpDir, "coverage", "reports")
		}
		reportDir = filepath.Join(reportDir, "fe", sanitizeFrontendAppToken(app))
		if opts.CoverageReport {
			if err := writeLcovQJS(ctx, coverage.ReportOptions{
				RepoRoot:  repoRoot,
				TmpRoot:   workspaceTmpDir,
				ReportDir: reportDir,
				Reporters: []string{"lcovonly", "text"},
				RunID:     runID,
			}); err != nil {
				return true, xfmt.Errorf("fe-qjs: WriteLcov: %w", err)
			}
		}
		if opts.CoverageCheck {
			if err := checkCoverageQJS(ctx, coverage.CheckOptions{
				RepoRoot:   repoRoot,
				TmpRoot:    workspaceTmpDir,
				RunID:      runID,
				Lines:      opts.CoverageLines,
				Functions:  opts.CoverageFunctions,
				Branches:   opts.CoverageBranches,
				Statements: opts.CoverageStatements,
			}); err != nil {
				return true, err
			}
		}
	}

	if err := writeQJSJUnitIfNeeded(app, report, opts.JUnitPath); err != nil {
		return true, err
	}

	if failed {
		fmt.Fprintf(os.Stderr, "# fe-qjs %s failed (%d/%d)\n", app, report.Failed, report.Total)
		return true, nil
	}
	fmt.Fprintf(os.Stderr, "# fe-qjs %s ok\n", app)
	return false, nil
}

func runOneAppFrontendTestsQJS(
	ctx context.Context,
	repoRoot string,
	app string,
	junitPath string,
	pattern string,
	coverageEnabled bool,
	coverageReport bool,
	coverageCheck bool,
	_ bool, // feCoverageAll unused on QJS path
	coverageReportDir string,
	coverageLines int,
	coverageFunctions int,
	coverageBranches int,
	coverageStatements int,
	tmpRoot string,
	keep bool,
) (bool, error) {
	if _, err := CheckIllegalFrontendMarks(repoRoot, app, ScanModeError); err != nil {
		return true, err
	}
	return RunFrontendQJS(ctx, QJSRunOptions{
		RepoRoot:           repoRoot,
		App:                app,
		JUnitPath:          junitPath,
		Pattern:            pattern,
		Coverage:           coverageEnabled,
		CoverageReport:     coverageReport,
		CoverageCheck:      coverageCheck,
		CoverageReportDir:  coverageReportDir,
		CoverageLines:      coverageLines,
		CoverageFunctions:  coverageFunctions,
		CoverageBranches:   coverageBranches,
		CoverageStatements: coverageStatements,
		TmpRoot:            tmpRoot,
		Keep:               keep,
	})
}

func frontendTestsNeedVue(testFiles []string) bool {
	for _, path := range testFiles {
		raw, err := osReadFileQJS(path)
		if err != nil {
			continue
		}
		if reVueImport.Match(raw) || bytes.Contains(raw, []byte("@choysum/test-utils")) || bytes.Contains(raw, []byte("@vue/test-utils")) {
			return true
		}
	}
	return false
}

func newFrontendCompilerExecutor(ctx context.Context) (jsexecutor.JsExecutor, error) {
	cfg := &config.Config{
		Server: &config.ServerConfig{
			JsEngineFactory:   "quickjs",
			JsExecutorFactory: "default",
		},
	}
	runtimeScope := &spikeBuildScope{ctx: ctx, cfg: cfg}
	executor, err := newCompilerExecutorFn(runtimeScope)
	if err != nil {
		return nil, xfmt.Errorf("fe-qjs: NewCompilerExecutor: %w", err)
	}
	if err := executor.Start(); err != nil {
		return nil, xfmt.Errorf("fe-qjs: Start compiler: %w", err)
	}
	return executor, nil
}

func evalChoysumTestRun(ctx context.Context, engine jsengine.JsEngine, pattern string) (*qjsRunReport, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	qjs, ok := engine.(*quickjsengine.QuickjsEngine)
	if !ok {
		return nil, xfmt.Errorf("fe-qjs: unexpected engine type %T", engine)
	}
	// Bind ctx so the engine interrupt handler honors cancel/deadline during Eval.
	restore := qjs.SwapExecContext(ctx)
	defer restore()

	patJSON, err := jsonMarshalQJS(strings.TrimSpace(pattern))
	if err != nil {
		return nil, err
	}
	script := `(async () => {
  const r = await globalThis.__choysum_test_run__({ pattern: ` + string(patJSON) + ` });
  const encoded = JSON.stringify(r);
  if (encoded === undefined || encoded === "null") {
    throw new Error("__choysum_test_run__ returned no report");
  }
  return encoded;
})()`
	val := qjs.Ctx.Eval(script, quickjs.EvalAwait(true))
	defer val.Free()
	if val.IsException() {
		// Exception() clears and returns the pending QuickJS error (buke/quickjs-go).
		return nil, xfmt.Errorf("fe-qjs: run: %v", qjs.Ctx.Exception())
	}
	raw := strings.TrimSpace(val.String())
	if raw == "" || raw == "null" || raw == "undefined" {
		return nil, xfmt.Errorf("fe-qjs: run: __choysum_test_run__ returned no report")
	}
	var report qjsRunReport
	if err := json.Unmarshal([]byte(raw), &report); err != nil {
		return nil, xfmt.Errorf("fe-qjs: parse report: %w raw=%s", err, raw)
	}
	return &report, nil
}

func writeQJSTap(w io.Writer, report *qjsRunReport) {
	if w == nil || report == nil {
		return
	}
	tapReport := &tap.Report{Total: report.Total}
	for _, c := range report.Cases {
		tc := tap.Case{Name: c.Name, OK: c.OK}
		if c.Error != nil {
			tc.Error = &tap.CaseError{Message: c.Error.Message, Stack: c.Error.Stack}
		}
		tapReport.Cases = append(tapReport.Cases, tc)
	}
	tap.Write(w, tapReport)
}

type qjsJUnitSuites struct {
	XMLName    xml.Name        `xml:"testsuites"`
	Tests      int             `xml:"tests,attr"`
	Failures   int             `xml:"failures,attr"`
	TestSuites []qjsJUnitSuite `xml:"testsuite"`
}

type qjsJUnitSuite struct {
	Name      string         `xml:"name,attr"`
	Tests     int            `xml:"tests,attr"`
	Failures  int            `xml:"failures,attr"`
	TestCases []qjsJUnitCase `xml:"testcase"`
}

type qjsJUnitCase struct {
	ClassName string        `xml:"classname,attr"`
	Name      string        `xml:"name,attr"`
	Time      string        `xml:"time,attr"`
	Failure   *qjsJUnitFail `xml:"failure,omitempty"`
}

type qjsJUnitFail struct {
	Message string `xml:"message,attr"`
	Body    string `xml:",chardata"`
}

func writeQJSJUnitIfNeeded(app string, report *qjsRunReport, junitPath string) error {
	junitPath = strings.TrimSpace(junitPath)
	if junitPath == "" || report == nil {
		return nil
	}
	suite := qjsJUnitSuite{Name: app, Tests: report.Total, Failures: report.Failed}
	for _, c := range report.Cases {
		jc := qjsJUnitCase{
			ClassName: app,
			Name:      c.Name,
			Time:      fmt.Sprintf("%.3f", float64(c.DurationMs)/1000.0),
		}
		if !c.OK {
			msg := "failed"
			body := ""
			if c.Error != nil {
				if c.Error.Message != "" {
					msg = c.Error.Message
				}
				body = c.Error.Stack
			}
			jc.Failure = &qjsJUnitFail{Message: msg, Body: body}
		}
		suite.TestCases = append(suite.TestCases, jc)
	}
	doc := qjsJUnitSuites{
		Tests:      report.Total,
		Failures:   report.Failed,
		TestSuites: []qjsJUnitSuite{suite},
	}
	out, err := xmlMarshalIndentQJS(doc, "", "  ")
	if err != nil {
		return xfmt.Errorf("fe-qjs: marshal junit: %w", err)
	}
	if dir := filepath.Dir(junitPath); dir != "" && dir != "." {
		if err := osMkdirAllQJS(dir, 0o755); err != nil {
			return xfmt.Errorf("fe-qjs: mkdir junit: %w", err)
		}
	}
	payload := append([]byte(xml.Header), out...)
	if err := osWriteFileQJS(junitPath, payload, 0o644); err != nil {
		return xfmt.Errorf("fe-qjs: write junit: %w", err)
	}
	return nil
}
