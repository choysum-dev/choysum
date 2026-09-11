// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	cliruntime "github.com/choysum-dev/choysum/internal/cli/runtime"
	cov "github.com/choysum-dev/choysum/internal/testing/coverage"
	pkge2e "github.com/choysum-dev/choysum/internal/testing/e2e"
	testsemantics "github.com/choysum-dev/choysum/internal/testing/semantics"
	testingpathing "github.com/choysum-dev/choysum/internal/testing/tmpdir"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/spf13/cobra"
)

var resolveE2EModules = pkge2e.ResolveE2EModules
var runE2EModule = pkge2e.RunModule

func isNoE2ESpecsError(err error) bool {
	return testsemantics.IsModuleNoE2ESpecsError(err)
}

func installChromiumBrowser() error {
	root := cov.FindRepoRootFromCwd()
	script := filepath.Join(root, "scripts", "ci", "install_chromium.py")
	if _, err := os.Stat(script); err != nil {
		return fmt.Errorf("install-browser: %w (expected %s; run from a choysum checkout)", err, script)
	}
	cmd := exec.Command("python3", script)
	cmd.Dir = root
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("install-browser: %w", err)
	}
	return nil
}

func newE2ECmd(envGetter func() scope.Scope, runtimeOptionsGetter func() cliruntime.Options) *cobra.Command {
	var scenarios []string
	var withDemo bool
	var keep bool
	var timeout time.Duration
	var startupTimeout time.Duration
	var port int
	var verbose bool
	var all bool
	var runtimeLogLevel string
	var installBrowser bool

	cmd := &cobra.Command{
		Use:          "e2e <module> [-- <specFilters...>]",
		Short:        "Run module-scoped system E2E (choysum run + QuickJS/chromedp)",
		SilenceUsage: true,
		Long: "Run module-scoped system E2E (choysum run + QuickJS + chromedp).\n\n" +
			"<module> refers to the module directory name under the modules path (e.g. modules/auth -> auth), not package.json's name.\n\n" +
			"Optional args after -- filter spec paths/names (for example: smoke.spec.ts).\n" +
			"Flag-looking args (leading '-') are ignored; use CHOYSUM_E2E_HEADED=1 for a visible browser.\n\n" +
			"Use --install-browser to download Chrome for Testing via scripts/ci/install_chromium.py.",
		Args: func(cmd *cobra.Command, args []string) error {
			if installBrowser {
				return nil
			}
			if all {
				return nil
			}
			if len(args) < 1 {
				return fmt.Errorf("e2e: requires <module> (or use --all / --install-browser)")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if installBrowser {
				return installChromiumBrowser()
			}
			baseScope := envGetter()
			if baseScope == nil {
				return fmt.Errorf("scope is not initialized")
			}
			runtimeOptions, err := cliruntime.RequireOptionsForCommand("e2e", runtimeOptionsGetter)
			if err != nil {
				return err
			}

			specFilterArgs := []string{}
			moduleName := ""
			if !all {
				moduleName = args[0]
				if len(args) > 1 {
					specFilterArgs = args[1:]
				}
			} else {
				specFilterArgs = args
			}

			resolvedRuntimeLogLevel := runtimeLogLevel
			if !cmd.Flags().Changed("runtime-log-level") && verbose {
				resolvedRuntimeLogLevel = "debug"
			}
			normalizedRuntimeLogLevel, err := cliruntime.NormalizeRuntimeLogLevelFlag(resolvedRuntimeLogLevel, "test e2e")
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}
			workspaceRoot, _ := os.Getwd()
			boundCtx, testTmp, _, err := testingpathing.BindCLITestRuntimePaths(ctx, workspaceRoot)
			if err != nil {
				return err
			}
			ctx = boundCtx
			if keep {
				fmt.Fprintf(os.Stderr, "choysum test e2e: kept CLI test tmp root: %s\n", testTmp)
			}
			if all {
				mods, err := resolveE2EModules(runtimeOptions.ModulesPath)
				if err != nil {
					return err
				}
				if len(mods) == 0 {
					return fmt.Errorf("%s", testsemantics.NoRunnableE2EModulesMessage(runtimeOptions.ModulesPath))
				}
				for _, mod := range mods {
					opts := pkge2e.RunOptions{
						ModulesPath:     runtimeOptions.ModulesPath,
						TmpPath:         testTmp,
						Module:          mod,
						Scenarios:       scenarios,
						WithDemo:        withDemo,
						Keep:            keep,
						Timeout:         timeout,
						StartupTimeout:  startupTimeout,
						Port:            port,
						Verbose:         verbose,
						RuntimeLogLevel: normalizedRuntimeLogLevel,
						SpecFilterArgs:  specFilterArgs,
						WorkDir:         "",
						Stdout:          os.Stdout,
						Stderr:          os.Stderr,
					}
					if err := runE2EModule(ctx, opts); err != nil {
						return err
					}
				}
				return nil
			}

			opts := pkge2e.RunOptions{
				ModulesPath:     runtimeOptions.ModulesPath,
				TmpPath:         testTmp,
				Module:          moduleName,
				Scenarios:       scenarios,
				WithDemo:        withDemo,
				Keep:            keep,
				Timeout:         timeout,
				StartupTimeout:  startupTimeout,
				Port:            port,
				Verbose:         verbose,
				RuntimeLogLevel: normalizedRuntimeLogLevel,
				SpecFilterArgs:  specFilterArgs,
				WorkDir:         "",
				Stdout:          os.Stdout,
				Stderr:          os.Stderr,
			}
			err = runE2EModule(ctx, opts)
			if isNoE2ESpecsError(err) {
				fmt.Fprintln(cmd.OutOrStdout(), testsemantics.NoTestsFoundMessage)
				return nil
			}
			return err
		},
	}

	cmd.Flags().StringArrayVar(&scenarios, "scenario", nil, "Scenario name (repeatable). Default: default")
	cmd.Flags().BoolVar(&withDemo, "with-demo", false, "Load demo data for the runtime dependency closure")
	cmd.Flags().BoolVar(&keep, "keep", false, "keep temp DB/config/log under the CLI test tmp root (CHOYSUM_TEST_TMP or <os.TempDir>/choysum-testing); prints absolute paths on stderr")
	cmd.Flags().DurationVar(&timeout, "timeout", 0, "Overall timeout for each scenario run (e.g. 5m). 0 means no timeout")
	cmd.Flags().DurationVar(&startupTimeout, "startup-timeout", 3*time.Minute, "Timeout waiting for /readyz")
	cmd.Flags().IntVar(&port, "port", 0, "Server port (default: auto pick)")
	cmd.Flags().BoolVar(&verbose, "verbose", false, "Verbose e2e runner output")
	cmd.Flags().StringVar(&runtimeLogLevel, "runtime-log-level", "", "override runtime log level during install/server run (debug|info|warn|error; default: warn, or debug when --verbose is set and this flag is omitted)")
	cmd.Flags().BoolVar(&all, "all", false, "run E2E for all runnable modules")
	cmd.Flags().BoolVar(&installBrowser, "install-browser", false, "download Chrome for Testing (scripts/ci/install_chromium.py) and exit")
	return cmd
}
