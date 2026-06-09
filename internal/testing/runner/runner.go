// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package runner

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	cov "github.com/choysum-dev/choysum/internal/testing/coverage"
	testingpathing "github.com/choysum-dev/choysum/internal/testing/tmpdir"
	"github.com/choysum-dev/choysum/pkg/scope"
	xfmt "golang.org/x/exp/errors/fmt"
)

type ResolveAppsFunc func(runtimeScope scope.Scope, arg string, runBE bool, runFE bool) ([]string, error)

type HasTestsFunc func(modulesPath string, app string) (bool, error)

type TypecheckFunc func(ctx context.Context, runtimeScope scope.Scope, repoRoot string, app string) error

type BackendRunnerFunc func(
	ctx context.Context,
	runtimeScope scope.Scope,
	app string,
	repoRoot string,
	dbDialect string,
	dbFile string,
	dbDSN string,
	keep bool,
	junitPath string,
	pattern string,
	failFast bool,
	coverage bool,
) (bool, error)

type FrontendRunnerFunc func(
	ctx context.Context,
	repoRoot string,
	app string,
	junitPath string,
	pattern string,
	coverage bool,
	coverageReport bool,
	coverageCheck bool,
	feCoverageAll bool,
	coverageReportDir string,
	coverageLines int,
	coverageFunctions int,
	coverageBranches int,
	coverageStatements int,
) (bool, error)

type RunOptions struct {
	Env         scope.Scope
	ModulesPath string
	Target      string // app|all
	RepoRoot    string

	RunBE bool
	RunFE bool

	DBDialect string
	DBFile    string
	DBDSN     string
	Keep      bool

	JUnitPath     string
	FailIfNoTests bool
	WithTypecheck bool
	Pattern       string
	FailFast      bool

	Coverage           bool
	CoverageReport     bool
	CoverageCheck      bool
	FECoverageAll      bool
	CoverageInclude    []string
	CoverageExclude    []string
	CoverageReportDir  string
	CoverageReporters  []string
	CoverageLines      int
	CoverageFunctions  int
	CoverageBranches   int
	CoverageStatements int

	Stdout io.Writer
	Stderr io.Writer

	ResolveApps      ResolveAppsFunc
	HasBackendTests  HasTestsFunc
	HasFrontendTests HasTestsFunc
	Typecheck        TypecheckFunc
	RunBackend       BackendRunnerFunc
	RunFrontend      FrontendRunnerFunc
}

func Run(ctx context.Context, opts RunOptions) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if opts.Stdout == nil {
		opts.Stdout = os.Stdout
	}
	if opts.Stderr == nil {
		opts.Stderr = os.Stderr
	}
	if !opts.RunBE && !opts.RunFE {
		fmt.Fprintln(opts.Stdout, "# no tests selected")
		return nil
	}
	runtimeOpts := runtimeOptionsFromScope(opts.Env)
	if opts.Env == nil || !runtimeOpts.hasConfig {
		return xfmt.Errorf("scope is not initialized")
	}
	if strings.TrimSpace(opts.ModulesPath) == "" {
		return xfmt.Errorf("config missing modules_path")
	}
	if strings.TrimSpace(opts.Target) == "" {
		return xfmt.Errorf("missing app")
	}
	if strings.TrimSpace(opts.RepoRoot) == "" {
		opts.RepoRoot = cov.FindRepoRootFromCwd()
	}
	testingRunID := testingpathing.TestingRunIDFromContext(ctx)
	if testingRunID == "" {
		testingRunID = testingpathing.NewTestingRunID()
		ctx = testingpathing.ContextWithTestingRunID(ctx, testingRunID)
	}
	if opts.Coverage && strings.TrimSpace(opts.CoverageReportDir) == "" {
		defaultReportDir, err := cov.ResolveCoverageReportDirWithRunID(opts.RepoRoot, runtimeOpts.tmpPath, testingRunID)
		if err != nil {
			return xfmt.Errorf("resolve default coverage report dir: %w", err)
		}
		opts.CoverageReportDir = defaultReportDir
	}

	if opts.ResolveApps == nil || opts.HasBackendTests == nil || opts.HasFrontendTests == nil || opts.RunBackend == nil || opts.RunFrontend == nil {
		return xfmt.Errorf("test runner: missing required callbacks")
	}

	apps, err := opts.ResolveApps(opts.Env, opts.Target, opts.RunBE, opts.RunFE)
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if len(apps) == 0 {
		msg := "no tests found"
		if opts.FailIfNoTests {
			return xfmt.Errorf(msg)
		}
		if opts.RunBE {
			fmt.Fprintln(opts.Stdout, msg)
		}
		return nil
	}

	if opts.Coverage && opts.RunBE {
		if strings.TrimSpace(cov.CoverageRunIDFromContext(ctx)) == "" {
			ctx = cov.ContextWithCoverageRunID(ctx, testingRunID)
		}
	}

	overallFailed := false
	ranAnyBE := false
	needsJUnitAppDisambiguation := len(apps) > 1
	for _, app := range apps {
		if err := ctx.Err(); err != nil {
			return err
		}
		hasBETests := false
		hasFETests := false
		if opts.RunBE {
			b, err := opts.HasBackendTests(opts.ModulesPath, app)
			if err != nil {
				return err
			}
			hasBETests = b
		}
		if opts.RunFE {
			f, err := opts.HasFrontendTests(opts.ModulesPath, app)
			if err != nil {
				return err
			}
			hasFETests = f
		}

		if opts.WithTypecheck && (hasBETests || hasFETests) {
			if opts.Typecheck == nil {
				return xfmt.Errorf("test runner: typecheck requested but callback missing")
			}
			if err := opts.Typecheck(ctx, opts.Env, opts.RepoRoot, app); err != nil {
				overallFailed = true
				if opts.FailFast {
					return err
				}
				fmt.Fprintln(opts.Stderr, err)
				continue
			}
		}

		needsJUnitScopeDisambiguation := hasBETests && hasFETests
		if hasBETests {
			ranAnyBE = true
			backendJUnitPath := resolveJUnitReportPath(opts.JUnitPath, app, "backend", needsJUnitAppDisambiguation, needsJUnitScopeDisambiguation)
			appFailed, err := opts.RunBackend(ctx, opts.Env, app, opts.RepoRoot, opts.DBDialect, opts.DBFile, opts.DBDSN, opts.Keep, backendJUnitPath, opts.Pattern, opts.FailFast, opts.Coverage)
			if err != nil {
				return err
			}
			if appFailed {
				overallFailed = true
				if opts.FailFast {
					break
				}
			}
		}
		if hasFETests {
			frontendJUnitPath := resolveJUnitReportPath(opts.JUnitPath, app, "frontend", needsJUnitAppDisambiguation, needsJUnitScopeDisambiguation)
			feFailed, err := opts.RunFrontend(ctx, opts.RepoRoot, app, frontendJUnitPath, opts.Pattern, opts.Coverage, opts.CoverageReport, opts.CoverageCheck, opts.FECoverageAll, opts.CoverageReportDir, opts.CoverageLines, opts.CoverageFunctions, opts.CoverageBranches, opts.CoverageStatements)
			if err != nil {
				return err
			}
			if feFailed {
				overallFailed = true
				if opts.FailFast {
					break
				}
			}
		}
	}

	if opts.Coverage && ranAnyBE {
		if opts.CoverageReport {
			if err := cov.RunNycReport(ctx, opts.RepoRoot, opts.CoverageReportDir, opts.CoverageReporters, opts.CoverageInclude, opts.CoverageExclude, runtimeOpts.tmpPath); err != nil {
				return err
			}
		}
		if opts.CoverageCheck {
			if err := cov.RunNycCheckCoverageWithTmpRoot(ctx, opts.RepoRoot, opts.CoverageInclude, opts.CoverageExclude, opts.CoverageLines, opts.CoverageFunctions, opts.CoverageBranches, opts.CoverageStatements, runtimeOpts.tmpPath); err != nil {
				return err
			}
		}
	}

	if overallFailed {
		return xfmt.Errorf("tests failed")
	}
	return nil
}

func resolveJUnitReportPath(basePath string, app string, scope string, needApp bool, needScope bool) string {
	basePath = strings.TrimSpace(basePath)
	if basePath == "" {
		return ""
	}

	hasAppPlaceholder := strings.Contains(basePath, "{app}")
	hasScopePlaceholder := strings.Contains(basePath, "{scope}")
	resolved := strings.ReplaceAll(basePath, "{app}", sanitizeJUnitReportToken(app))
	resolved = strings.ReplaceAll(resolved, "{scope}", sanitizeJUnitReportToken(scope))

	var suffixes []string
	if needApp && !hasAppPlaceholder {
		suffixes = append(suffixes, sanitizeJUnitReportToken(app))
	}
	if needScope && !hasScopePlaceholder {
		suffixes = append(suffixes, sanitizeJUnitReportToken(scope))
	}
	if len(suffixes) == 0 {
		return resolved
	}
	return insertSuffixBeforeExt(resolved, strings.Join(suffixes, "."))
}

func insertSuffixBeforeExt(path string, suffix string) string {
	path = strings.TrimSpace(path)
	suffix = strings.TrimSpace(suffix)
	if path == "" || suffix == "" {
		return path
	}
	ext := filepath.Ext(path)
	if ext == "" {
		return path + "." + suffix
	}
	return strings.TrimSuffix(path, ext) + "." + suffix + ext
}

func sanitizeJUnitReportToken(raw string) string {
	replacer := strings.NewReplacer("/", "_", "\\", "_", " ", "_")
	token := strings.TrimSpace(replacer.Replace(raw))
	if token == "" {
		return "app"
	}
	return token
}
