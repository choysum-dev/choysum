// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	cliruntime "github.com/choysum-dev/choysum/internal/cli/runtime"
	testingpathing "github.com/choysum-dev/choysum/internal/testing/tmpdir"
	pkgtypecheck "github.com/choysum-dev/choysum/internal/testing/typecheck"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/spf13/cobra"
	xfmt "golang.org/x/exp/errors/fmt"
)

func newTypecheckCmd(envGetter func() scope.Scope, runtimeOptionsGetter func() cliruntime.Options) *cobra.Command {
	var all bool
	var keep bool

	cmd := &cobra.Command{
		Use:          "typecheck <app>",
		Short:        "Type-check modules (service + web)",
		SilenceUsage: true,
		Args: func(cmd *cobra.Command, args []string) error {
			if all {
				if len(args) != 0 {
					return xfmt.Errorf("typecheck: --all cannot be used with an app argument")
				}
				return nil
			}
			if len(args) != 1 {
				return xfmt.Errorf("typecheck: requires exactly 1 app argument (or use --all)")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			baseScope := envGetter()
			if baseScope == nil {
				return xfmt.Errorf("typecheck: invalid scope")
			}
			runtimeOptions, err := cliruntime.RequireOptionsForCommand("typecheck", runtimeOptionsGetter)
			if err != nil {
				return err
			}

			target := "all"
			if !all {
				target = strings.TrimSpace(args[0])
			}

			repoRoot, _ := os.Getwd()
			if strings.TrimSpace(repoRoot) == "" {
				return xfmt.Errorf("typecheck: cannot determine repo root")
			}

			ctx := cmd.Context()
			boundCtx, testTmp, runHome, err := testingpathing.BindCLITestRuntimePaths(ctx, repoRoot)
			if err != nil {
				return err
			}
			ctx = boundCtx
			if keep {
				fmt.Fprintf(cmd.ErrOrStderr(), "choysum test typecheck: kept CLI test tmp root: %s\n", testTmp)
			}

			// Point type-fetch rewrite/search at the per-run home (pkg → durable
			// CHOYSUM_TEST_TMP cache). Repo-relative ../../.choysum paths only
			// hit ~/.choysum when the checkout lives under $HOME/<name>.
			prevChoysumHome, hadChoysumHome := os.LookupEnv("CHOYSUM_HOME")
			if strings.TrimSpace(runHome) != "" {
				if err := os.Setenv("CHOYSUM_HOME", runHome); err != nil {
					return xfmt.Errorf("typecheck: set CHOYSUM_HOME: %w", err)
				}
				defer func() {
					if hadChoysumHome {
						_ = os.Setenv("CHOYSUM_HOME", prevChoysumHome)
						return
					}
					_ = os.Unsetenv("CHOYSUM_HOME")
				}()
			}

			opts := pkgtypecheck.RunOptions{
				ModulesPath: runtimeOptions.ModulesPath,
				NpmPath:     filepath.Join(runtimeOptions.ModulesPath, "node_modules"),
				RepoRoot:    repoRoot,
				TmpPath:     testTmp,
				Target:      target,
				Keep:        keep,
				Stdout:      cmd.OutOrStdout(),
				Stderr:      cmd.ErrOrStderr(),
			}
			return pkgtypecheck.Run(ctx, opts)
		},
	}

	cmd.Flags().BoolVar(&all, "all", false, "type-check all apps")
	cmd.Flags().BoolVar(&keep, "keep", false, "keep intermediate artifacts under the CLI test tmp root (CHOYSUM_TEST_TMP or <os.TempDir>/choysum-testing); prints absolute paths on stderr")

	return cmd
}
