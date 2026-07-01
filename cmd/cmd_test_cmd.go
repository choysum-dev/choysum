// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cmd

import (
	"context"
	"errors"
	"os"
	"strconv"
	"strings"
	"time"

	cliruntime "github.com/choysum-dev/choysum/internal/cli/runtime"
	"github.com/choysum-dev/choysum/internal/logger"
	cov "github.com/choysum-dev/choysum/internal/testing/coverage"
	pkgrunner "github.com/choysum-dev/choysum/internal/testing/runner"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/spf13/cobra"
	xfmt "golang.org/x/exp/errors/fmt"
)

var runTestRunnerWithDefaults = pkgrunner.RunWithDefaults

func newTestCmd(envGetter func() scope.Scope, runtimeOptionsGetter func() cliruntime.Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "test",
		Short: "Run test commands",
		Annotations: map[string]string{
			lightweightScopeAnnotation: "true",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return xfmt.Errorf("test: requires a subcommand (unit|typecheck|e2e)")
		},
	}
	cmd.AddCommand(
		newTestUnitCmd(envGetter, runtimeOptionsGetter),
		newTypecheckCmd(envGetter, runtimeOptionsGetter),
		newE2ECmd(envGetter, runtimeOptionsGetter),
	)
	return cmd
}

func newTestUnitCmd(envGetter func() scope.Scope, runtimeOptionsGetter func() cliruntime.Options) *cobra.Command {
	var dbDialect string
	var dbFile string
	var dbDSN string
	var keep bool
	var junitPath string
	var failIfNoTests bool
	var withTypecheck bool
	var pattern string
	var failFast bool
	var tapStdout bool
	var timeout time.Duration
	var scopeBE bool
	var scopeFE bool
	var coverage bool
	var coverageReport bool
	var coverageCheck bool
	var feCoverageAll bool
	var coverageInclude []string
	var coverageExclude []string
	var coverageReportDir string
	var coverageReporters []string
	var coverageLines int
	var coverageFunctions int
	var coverageBranches int
	var coverageStatements int
	var all bool
	var runtimeLogLevel string

	cmd := &cobra.Command{
		Use:   "unit <app>",
		Short: "Run backend TS tests in QuickJS",
		Args: func(cmd *cobra.Command, args []string) error {
			if all {
				if len(args) != 0 {
					return xfmt.Errorf("test unit: --all cannot be used with an app argument")
				}
				return nil
			}
			if len(args) != 1 {
				return xfmt.Errorf("test unit: requires exactly 1 app argument (or use --all)")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}
			if timeout > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, timeout)
				defer cancel()
			}

			// Scope selection is based on whether the flags were explicitly provided.
			// This allows `--be=false --fe=false` to mean "run none" (instead of
			// defaulting back to "run both").
			beSpecified := cmd.Flags().Changed("be")
			feSpecified := cmd.Flags().Changed("fe")
			runBE := scopeBE
			runFE := scopeFE
			if !beSpecified && !feSpecified {
				runBE = true
				runFE = true
			}

			baseScope := envGetter()
			if baseScope == nil {
				return xfmt.Errorf("scope is not initialized")
			}

			runtimeOptions, err := cliruntime.RequireOptionsForCommand("test unit", runtimeOptionsGetter)
			if err != nil {
				return err
			}
			modulesPath := strings.TrimSpace(runtimeOptions.ModulesPath)

			// Default: TAP to stdout, business logs to stderr.
			// This keeps TAP machine-parseable. Use --tap-stdout=false to revert legacy mixed output.
			if tapStdout && runBE {
				factoryInput := scope.FactoryInputFromScope(baseScope)
				if factoryInput == nil {
					return xfmt.Errorf("config is not initialized")
				}
				splitScope := cliruntime.RebuildScope(baseScope, factoryInput, baseScope.Logger())
				if splitScope == nil {
					return xfmt.Errorf("failed to initialize scope for tap stdout")
				}
				baseScope = splitScope
			}
			if runBE {
				normalizedLevel, err := cliruntime.NormalizeRuntimeLogLevelFlag(runtimeLogLevel, "test unit")
				if err != nil {
					return err
				}
				factoryInput := scope.FactoryInputFromScope(baseScope)
				if factoryInput == nil {
					return xfmt.Errorf("config is not initialized")
				}
				logCfg := cliruntime.CloneLogConfig(scope.LogConfigFromScope(baseScope))
				logCfg.Level = normalizedLevel
				quietScope := cliruntime.RebuildScope(baseScope, factoryInput, logger.NewLoggerWithWriter(logCfg, cmd.ErrOrStderr()))
				if quietScope == nil {
					return xfmt.Errorf("failed to initialize scope for runtime log level")
				}

				baseScope = quietScope
			}

			// Coverage options precedence:
			// 1) CLI flags (explicit)
			// 2) env vars (CI-friendly)
			// 3) defaults
			if coverage {
				if !cmd.Flags().Changed("coverage-include") {
					if v := strings.TrimSpace(os.Getenv("CHOYSUM_TEST_COVERAGE_INCLUDE")); v != "" {
						coverageInclude = cov.SplitCoverageGlobs(v)
					}
				}
				if !cmd.Flags().Changed("coverage-exclude") {
					if v := strings.TrimSpace(os.Getenv("CHOYSUM_TEST_COVERAGE_EXCLUDE")); v != "" {
						coverageExclude = cov.SplitCoverageGlobs(v)
					}
				}
				if !cmd.Flags().Changed("coverage-report-dir") {
					if v := strings.TrimSpace(os.Getenv("CHOYSUM_TEST_COVERAGE_REPORT_DIR")); v != "" {
						coverageReportDir = v
					}
				}
				if !cmd.Flags().Changed("coverage-reporters") {
					if v := strings.TrimSpace(os.Getenv("CHOYSUM_TEST_COVERAGE_REPORTERS")); v != "" {
						coverageReporters = cov.SplitCoverageReporters(v)
					}
				}

				if !cmd.Flags().Changed("coverage-lines") {
					if v := strings.TrimSpace(os.Getenv("CHOYSUM_TEST_COVERAGE_LINES")); v != "" {
						n, err := strconv.Atoi(v)
						if err != nil {
							return xfmt.Errorf("invalid CHOYSUM_TEST_COVERAGE_LINES=%q: %w", v, err)
						}
						coverageLines = n
					}
				}
				if !cmd.Flags().Changed("coverage-functions") {
					if v := strings.TrimSpace(os.Getenv("CHOYSUM_TEST_COVERAGE_FUNCTIONS")); v != "" {
						n, err := strconv.Atoi(v)
						if err != nil {
							return xfmt.Errorf("invalid CHOYSUM_TEST_COVERAGE_FUNCTIONS=%q: %w", v, err)
						}
						coverageFunctions = n
					}
				}
				if !cmd.Flags().Changed("coverage-branches") {
					if v := strings.TrimSpace(os.Getenv("CHOYSUM_TEST_COVERAGE_BRANCHES")); v != "" {
						n, err := strconv.Atoi(v)
						if err != nil {
							return xfmt.Errorf("invalid CHOYSUM_TEST_COVERAGE_BRANCHES=%q: %w", v, err)
						}
						coverageBranches = n
					}
				}
				if !cmd.Flags().Changed("coverage-statements") {
					if v := strings.TrimSpace(os.Getenv("CHOYSUM_TEST_COVERAGE_STATEMENTS")); v != "" {
						n, err := strconv.Atoi(v)
						if err != nil {
							return xfmt.Errorf("invalid CHOYSUM_TEST_COVERAGE_STATEMENTS=%q: %w", v, err)
						}
						coverageStatements = n
					}
				}
			}

			repoRoot := cov.FindRepoRootFromCwd()
			target := "all"
			if !all {
				target = args[0]
			}
			opts := pkgrunner.RunOptions{
				Env:                baseScope,
				ModulesPath:        modulesPath,
				Target:             target,
				RepoRoot:           repoRoot,
				RunBE:              runBE,
				RunFE:              runFE,
				DBDialect:          dbDialect,
				DBFile:             dbFile,
				DBDSN:              dbDSN,
				Keep:               keep,
				JUnitPath:          junitPath,
				FailIfNoTests:      failIfNoTests,
				WithTypecheck:      withTypecheck,
				Pattern:            pattern,
				FailFast:           failFast,
				Coverage:           coverage,
				CoverageReport:     coverageReport,
				CoverageCheck:      coverageCheck,
				FECoverageAll:      feCoverageAll,
				CoverageInclude:    coverageInclude,
				CoverageExclude:    coverageExclude,
				CoverageReportDir:  coverageReportDir,
				CoverageReporters:  coverageReporters,
				CoverageLines:      coverageLines,
				CoverageFunctions:  coverageFunctions,
				CoverageBranches:   coverageBranches,
				CoverageStatements: coverageStatements,
				Stdout:             cmd.OutOrStdout(),
				Stderr:             cmd.ErrOrStderr(),
			}

			err = runTestRunnerWithDefaults(ctx, opts)
			if err == nil {
				return nil
			}
			if errors.Is(err, context.DeadlineExceeded) {
				cmd.SilenceUsage = true
				if timeout > 0 {
					return xfmt.Errorf("test run timed out after %s: %w", timeout, err)
				}
				return xfmt.Errorf("test run timed out: %w", err)
			}
			if errors.Is(err, context.Canceled) {
				cmd.SilenceUsage = true
				return xfmt.Errorf("test run canceled: %w", err)
			}
			if strings.TrimSpace(err.Error()) == "tests failed" {
				// Detailed per-app failures are already printed by the runner; suppress
				// the trailing generic "Error: tests failed" line from Cobra.
				cmd.SilenceUsage = true
				cmd.SilenceErrors = true
			}
			return err
		},
	}

	cmd.Flags().StringVar(&dbDialect, "db", "sqlite", "database dialect for tests (sqlite|postgres)")
	cmd.Flags().StringVar(&dbFile, "db-file", "", "sqlite database file path (default: auto-generated; sqlite only)")
	cmd.Flags().StringVar(&dbDSN, "db-dsn", "", "Postgres DSN for tests (or env CHOYSUM_TEST_POSTGRES_DSN); required when --db=postgres")
	cmd.Flags().BoolVar(&keep, "keep", false, "keep intermediate artifacts for debugging")
	cmd.Flags().StringVar(&junitPath, "junit", "", "write JUnit XML report(s) to path; supports {app} and {scope} placeholders, and auto-disambiguates multi-app or mixed backend/frontend runs")
	cmd.Flags().DurationVar(&timeout, "timeout", 0, "overall timeout for the test run (e.g. 30s, 2m); 0 means no timeout")
	cmd.Flags().BoolVar(&failIfNoTests, "fail-if-no-tests", false, "exit non-zero if no tests found")
	cmd.Flags().BoolVar(&withTypecheck, "with-typecheck", false, "run TypeScript typecheck for modules/<app>/service + modules/<app>/web before tests (equivalent to 'choysum test typecheck <app>'; requires npm install)")
	cmd.Flags().StringVar(&pattern, "pattern", "", "run tests whose names match this regex")
	cmd.Flags().BoolVar(&failFast, "fail-fast", true, "stop after first failure (set --fail-fast=false to continue running remaining apps)")
	cmd.Flags().BoolVar(&tapStdout, "tap-stdout", true, "print TAP to stdout and logs to stderr (set false to revert legacy mixed output)")
	cmd.Flags().StringVar(&runtimeLogLevel, "runtime-log-level", "warn", "runtime log level during backend test setup/execution (debug|info|warn|error; default: warn)")
	cmd.Flags().BoolVar(&scopeBE, "be", false, "run backend (QuickJS) tests")
	cmd.Flags().BoolVar(&scopeFE, "fe", false, "run frontend (Vitest) tests")
	cmd.Flags().BoolVar(&coverage, "coverage", false, "enable coverage collection (Istanbul __coverage__ -> TmpPath/testing/<workspace-hash>/<run-id>/coverage/nyc_output)")
	cmd.Flags().BoolVar(&coverageReport, "coverage-report", false, "generate coverage report artifacts for Codecov and local inspection via nyc/vitest (requires node and test tooling installed)")
	cmd.Flags().BoolVar(&coverageCheck, "coverage-check", false, "fail if coverage is below thresholds (requires node + nyc installed)")
	cmd.Flags().BoolVar(&feCoverageAll, "fe-coverage-all", false, "frontend (Vitest) coverage: include all app source files, not just files hit by tests")
	cmd.Flags().StringArrayVar(&coverageInclude, "coverage-include", nil, "nyc include glob(s) used for report/check (repeatable)")
	cmd.Flags().StringArrayVar(&coverageExclude, "coverage-exclude", nil, "nyc exclude glob(s) used for report/check (repeatable)")
	cmd.Flags().StringVar(&coverageReportDir, "coverage-report-dir", "", "directory for coverage reports (default: TmpPath/testing/<workspace-hash>/<run-id>/coverage/reports; e.g. coverage, .choysum/coverage)")
	cmd.Flags().StringArrayVar(&coverageReporters, "coverage-reporters", nil, "nyc reporter(s) for --coverage-report (repeatable; default: text,html)")
	cmd.Flags().IntVar(&coverageLines, "coverage-lines", 0, "nyc check-coverage --lines (0 disables)")
	cmd.Flags().IntVar(&coverageFunctions, "coverage-functions", 0, "nyc check-coverage --functions (0 disables)")
	cmd.Flags().IntVar(&coverageBranches, "coverage-branches", 0, "nyc check-coverage --branches (0 disables)")
	cmd.Flags().IntVar(&coverageStatements, "coverage-statements", 0, "nyc check-coverage --statements (0 disables)")
	cmd.Flags().BoolVar(&all, "all", false, "run tests for all apps")

	_ = cmd.MarkFlagFilename("junit")
	_ = cmd.MarkFlagFilename("db-file")

	return cmd
}
