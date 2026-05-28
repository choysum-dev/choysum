// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cmd

import (
	"context"
	"os"
	"os/signal"
	"strings"

	"github.com/choysum-dev/choysum/internal/module/lifecycle"
	internalorigin "github.com/choysum-dev/choysum/internal/module/origin"
	"github.com/choysum-dev/choysum/pkg/jsexecutor"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/spf13/cobra"
	xfmt "golang.org/x/exp/errors/fmt"
)

func newInstallCmd(envGetter func() scope.Scope) *cobra.Command {
	var withDemo bool
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install Choysum Module",
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

			coordinator := internalorigin.NewCoordinator(env)

			// Create a transaction-bound module manager for each module install.
			// Run each module install in its own transaction to avoid keeping a
			// long-lived transaction open across multiple modules (which amplifies SQLite
			// lock contention during install bursts).
			for _, input := range args {
				parsed, err := internalorigin.ParseInput(strings.TrimSpace(input))
				if err != nil {
					env.Logger().Error("module input invalid", "input", input, "error", err)
					os.Exit(1)
				}

				moduleName := strings.TrimSpace(parsed.LocalName)
				if parsed.Kind == internalorigin.InputKindRegistry {
					resolved, fetchErr := coordinator.Fetch(ctx, parsed.CanonicalRef())
					if fetchErr != nil {
						env.Logger().Error("module source resolution failed", "input", input, "error", fetchErr)
						os.Exit(1)
					}
					if resolved == nil || strings.TrimSpace(resolved.Name) == "" {
						env.Logger().Error("module source invalid", "input", input, "reason", "resolved module is empty")
						os.Exit(1)
					}
					moduleName = strings.TrimSpace(resolved.Name)
				}

				txRoot := env.WithContext(ctx)
				if err := txRoot.Transactor().Required(ctx, func(txScope scope.Scope, tx scope.Transaction) error {
					compilerExecutor, err := jsexecutor.NewCompilerExecutor(txScope)
					if err != nil {
						return xfmt.Errorf("Error creating compiler executor: %w", err)
					}
					if err := compilerExecutor.Start(); err != nil {
						return xfmt.Errorf("Error starting compiler executor: %w", err)
					}
					defer compilerExecutor.Stop()

					if parsed.Kind == internalorigin.InputKindLocal && !meta.IsCoreModule(moduleName) {
						if _, peekErr := coordinator.Peek(ctx, moduleName); peekErr != nil {
							if strings.Contains(peekErr.Error(), "not found in addons path") {
								peekErr = xfmt.Errorf("module %s not found in addons path; run `choysum module fetch <registry>/<module>@<version>` or `choysum install <registry>/<module>@<version>`", moduleName)
							}
							return peekErr
						}
					}

					txScope.Logger().Debug("module install started", "module", moduleName)
					moduleLifecycle := lifecycle.NewService(txScope, compilerExecutor)
					if err := moduleLifecycle.Install(tx.Context(), lifecycle.InstallRequest{Name: moduleName, WithDemo: withDemo}); err != nil {
						return xfmt.Errorf("error installing module %s: %w", moduleName, err)
					}
					txScope.Logger().Debug("module installed", "module", moduleName)
					return nil
				}); err != nil {
					if parsed.Kind == internalorigin.InputKindLocal && !meta.IsCoreModule(moduleName) && strings.Contains(err.Error(), "not found in addons path") {
						err = xfmt.Errorf("module %s not found in addons path; run `choysum module fetch <registry>/<module>@<version>` or `choysum install <registry>/<module>@<version>`", moduleName)
					}
					attrs := []any{"error", err}
					attrs = append(attrs, moduleCommandFailureAttrs("install")...)
					attrs = append(attrs, moduleInstallFailureAttrs(input, moduleName)...)
					env.Logger().Error("module install failed", attrs...)
					os.Exit(1)
				}
			}
		},
	}
	cmd.Flags().BoolVar(&withDemo, "with-demo", false, "Load demo data declared by manifest")
	return cmd
}
