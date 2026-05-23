// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cmd

import (
	"context"
	"os"
	"os/signal"
	"strings"

	"github.com/choysum-dev/choysum/internal/module/lifecycle"
	"github.com/choysum-dev/choysum/pkg/jsexecutor"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/spf13/cobra"
	xfmt "golang.org/x/exp/errors/fmt"
)

func newUpgradeCmd(envGetter func() scope.Scope) *cobra.Command {
	var withDemo bool
	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade Choysum Module",
		PreRun: func(cmd *cobra.Command, args []string) {
			env := envGetter()
			if env == nil {
				printCLIError("scope is not initialized")
				os.Exit(1)
			}
			if len(args) == 0 {
				printCLIError("Please specify the module name")
				os.Exit(1)
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			env := envGetter()
			if env == nil {
				printCLIError("scope is not initialized")
				os.Exit(1)
			}
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
			defer stop()

			// Create a transaction-bound module manager for the upgrade batch.
			txRoot := env.WithContext(ctx)
			currentInput := ""
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
				for _, input := range args {
					moduleInput := strings.TrimSpace(input)
					currentInput = moduleInput
					if moduleInput == "" {
						return xfmt.Errorf("module name is empty")
					}

					txScope.Logger().Debug("module upgrade started", "input", moduleInput)
					if err := moduleLifecycle.Upgrade(tx.Context(), lifecycle.UpgradeRequest{Input: moduleInput, WithDemo: withDemo}); err != nil {
						return xfmt.Errorf("error upgrading module %s: %w", moduleInput, err)
					}
					txScope.Logger().Debug("module upgraded", "input", moduleInput)
				}
				return nil
			}); err != nil {
				attrs := []any{"error", err}
				attrs = append(attrs, moduleCommandFailureAttrs("upgrade")...)
				attrs = append(attrs, currentOrRequestedAttr("input", "inputs", currentInput, args)...)
				env.Logger().Error("module upgrade failed", attrs...)
				os.Exit(1)
			}
		},
	}
	cmd.Flags().BoolVar(&withDemo, "with-demo", false, "Load demo data declared by manifest")
	return cmd
}
