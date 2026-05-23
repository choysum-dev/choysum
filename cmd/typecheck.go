// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cmd

import (
	"os"
	"strings"

	pkgtypecheck "github.com/choysum-dev/choysum/internal/testing/typecheck"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/spf13/cobra"
	xfmt "golang.org/x/exp/errors/fmt"
)

func newTypecheckCmd(envGetter func() scope.Scope, runtimeOptionsGetter func() cliRuntimeOptions) *cobra.Command {
	var all bool
	var keep bool

	cmd := &cobra.Command{
		Use:   "typecheck <app>",
		Short: "Type-check addons (service + web)",
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
			runtimeOptions, err := requireCliRuntimeOptions(runtimeOptionsGetter)
			if err != nil {
				return xfmt.Errorf("typecheck: invalid runtime options: %w", err)
			}

			target := "all"
			if !all {
				target = strings.TrimSpace(args[0])
			}

			repoRoot, _ := os.Getwd()
			if strings.TrimSpace(repoRoot) == "" {
				return xfmt.Errorf("typecheck: cannot determine repo root")
			}

			opts := pkgtypecheck.RunOptions{
				AddonsPath: runtimeOptions.addonsPath,
				NpmPath:    runtimeOptions.npmPath,
				RepoRoot:   repoRoot,
				TmpPath:    runtimeOptions.tmpPath,
				Target:     target,
				Keep:       keep,
				Stdout:     os.Stdout,
				Stderr:     os.Stderr,
			}
			return pkgtypecheck.Run(cmd.Context(), opts)
		},
	}

	cmd.Flags().BoolVar(&all, "all", false, "type-check all apps")
	cmd.Flags().BoolVar(&keep, "keep", false, "keep intermediate artifacts for debugging")

	return cmd
}
