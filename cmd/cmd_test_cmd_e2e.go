// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	cliruntime "github.com/choysum-dev/choysum/internal/cli/runtime"
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

	cmd := &cobra.Command{
		Use:   "e2e <module> [-- <playwrightArgs...>]",
		Short: "Run module-scoped system E2E (choysum run + Playwright)",
		Long: "Run module-scoped system E2E (choysum run + Playwright).\n\n" +
			"<module> refers to the module directory name under the modules path (e.g. modules/auth -> auth), not package.json's name.",
		Args: func(cmd *cobra.Command, args []string) error {
			if all {
				return nil
			}
			if len(args) < 1 {
				return fmt.Errorf("e2e: requires <module> (or use --all)")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			baseScope := envGetter()
			if baseScope == nil {
				return fmt.Errorf("scope is not initialized")
			}
			runtimeOptions, err := cliruntime.RequireOptionsForCommand("e2e", runtimeOptionsGetter)
			if err != nil {
				return err
			}

			playwrightArgs := []string{}
			moduleName := ""
			if !all {
				moduleName = args[0]
				if len(args) > 1 {
					playwrightArgs = args[1:]
				}
			} else {
				playwrightArgs = args
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
			if all {
				if testingpathing.TestingRunIDFromContext(ctx) == "" {
					ctx = testingpathing.ContextWithTestingRunID(ctx, testingpathing.NewTestingRunID())
				}
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
						TmpPath:         runtimeOptions.TmpPath,
						Module:          mod,
						Scenarios:       scenarios,
						WithDemo:        withDemo,
						Keep:            keep,
						Timeout:         timeout,
						StartupTimeout:  startupTimeout,
						Port:            port,
						Verbose:         verbose,
						RuntimeLogLevel: normalizedRuntimeLogLevel,
						PlaywrightArgs:  playwrightArgs,
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
				TmpPath:         runtimeOptions.TmpPath,
				Module:          moduleName,
				Scenarios:       scenarios,
				WithDemo:        withDemo,
				Keep:            keep,
				Timeout:         timeout,
				StartupTimeout:  startupTimeout,
				Port:            port,
				Verbose:         verbose,
				RuntimeLogLevel: normalizedRuntimeLogLevel,
				PlaywrightArgs:  playwrightArgs,
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
	cmd.Flags().BoolVar(&keep, "keep", false, "keep temp DB/config/log for debugging")
	cmd.Flags().DurationVar(&timeout, "timeout", 0, "Overall timeout for each scenario run (e.g. 5m). 0 means no timeout")
	cmd.Flags().DurationVar(&startupTimeout, "startup-timeout", 3*time.Minute, "Timeout waiting for /readyz")
	cmd.Flags().IntVar(&port, "port", 0, "Server port (default: auto pick)")
	cmd.Flags().BoolVar(&verbose, "verbose", false, "Verbose e2e runner output")
	cmd.Flags().StringVar(&runtimeLogLevel, "runtime-log-level", "", "override runtime log level during install/server run (debug|info|warn|error; default: warn, or debug when --verbose is set and this flag is omitted)")
	cmd.Flags().BoolVar(&all, "all", false, "run E2E for all runnable modules")
	return cmd
}
