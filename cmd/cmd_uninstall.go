// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cmd

import (
	"context"
	"os"
	"os/signal"

	clioutput "github.com/choysum-dev/choysum/internal/cli/output"
	"github.com/choysum-dev/choysum/internal/module/lifecycle"
	"github.com/choysum-dev/choysum/pkg/jsexecutor"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/spf13/cobra"
	xfmt "golang.org/x/exp/errors/fmt"
)

func newUninstallCmd(envGetter func() scope.Scope) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall Choysum Module",
		PreRun: func(cmd *cobra.Command, args []string) {
			env := envGetter()
			if env == nil {
				clioutput.PrintError("scope is not initialized")
				os.Exit(1)
			}
			if len(args) == 0 {
				clioutput.PrintError("Please specify the module name")
				os.Exit(1)
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			env := envGetter()
			if env == nil {
				clioutput.PrintError("scope is not initialized")
				os.Exit(1)
			}
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
			defer stop()

			// Create a transaction-bound module manager for the uninstall batch.
			txRoot := env.WithContext(ctx)
			currentModule := ""
			if err := txRoot.Transactor().Required(ctx, func(txScope scope.Scope, tx scope.Transaction) error {
				compilerExecutor, err := jsexecutor.NewCompilerExecutor(txScope)
				if err != nil {
					return xfmt.Errorf("Error creating compiler executor: %w", err)
				}
				if err := compilerExecutor.Start(); err != nil {
					return xfmt.Errorf("Error starting compiler executor: %w", err)
				}
				defer compilerExecutor.Stop()

				moduleLifecycle := lifecycle.NewService(txScope, compilerExecutor)
				for _, name := range args {
					currentModule = name
					txScope.Logger().Debug("module uninstall started", "module", name)
					if err := moduleLifecycle.Uninstall(tx.Context(), lifecycle.UninstallRequest{Name: name}); err != nil {
						return xfmt.Errorf("error uninstalling module %s: %w", name, err)
					}
					txScope.Logger().Debug("module uninstalled", "module", name)

				}
				return nil
			}); err != nil {
				attrs := []any{"error", err}
				attrs = append(attrs, clioutput.ModuleCommandFailureAttrs("uninstall")...)
				attrs = append(attrs, clioutput.CurrentOrRequestedAttr("module", "modules", currentModule, args)...)
				env.Logger().Error("module uninstall failed", attrs...)
				os.Exit(1)
			}
		},
	}
	return cmd
}
