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
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/spf13/cobra"
	xfmt "golang.org/x/exp/errors/fmt"
)

func newUpgradeCmd(envGetter func() scope.Scope) *cobra.Command {
	var withDemo bool
	var cliCompatVersion string
	cmd := &cobra.Command{
		Use:   "upgrade <module|module@version> [<module|module@version>...]",
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
			resolvedCompat, err := resolveCLICompatVersionForCommand(cmd, cliCompatVersion)
			if err != nil {
				env.Logger().Error("module compatibility version resolution failed", "error", err)
				os.Exit(1)
			}
			runtimeOptions := cliRuntimeOptionsFromScope(env)
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

					parsed, err := internalorigin.ParseInput(moduleInput)
					if err != nil {
						return xfmt.Errorf("error parsing module input %s: %w", moduleInput, err)
					}

					upgradeInput := moduleInput
					switch parsed.Kind {
					case internalorigin.InputKindRegistry:
						if strings.EqualFold(strings.TrimSpace(parsed.Version), "latest") {
							if strings.TrimSpace(resolvedCompat.Version) == "" {
								return xfmt.Errorf("ERR_CLI_COMPAT_VERSION_UNRESOLVED: Cannot resolve a CLI compatibility version in development mode. Provide '--cli-compat-version' or set 'CHOYSUM_CLI_COMPAT_VERSION'.")
							}
							compatibleVersion, compatErr := resolveCompatibleRegistryLatestVersion(tx.Context(), txScope, runtimeOptions, parsed.ModuleName, resolvedCompat.Version)
							if compatErr != nil {
								return compatErr
							}
							upgradeInput = strings.TrimSpace(parsed.ModuleName) + "@" + compatibleVersion
						}
					case internalorigin.InputKindLocal:
						registryBacked, bindErr := hasRegistryOriginBinding(txScope, runtimeOptions, parsed.LocalName)
						if bindErr != nil {
							return bindErr
						}
						if registryBacked {
							if strings.TrimSpace(resolvedCompat.Version) == "" {
								return xfmt.Errorf("ERR_CLI_COMPAT_VERSION_UNRESOLVED: Cannot resolve a CLI compatibility version in development mode. Provide '--cli-compat-version' or set 'CHOYSUM_CLI_COMPAT_VERSION'.")
							}
							resolvedInput, _, resolveErr := resolveRegistryBackedUpgradeInput(tx.Context(), txScope, runtimeOptions, parsed.LocalName, resolvedCompat.Version)
							if resolveErr != nil {
								return resolveErr
							}
							upgradeInput = resolvedInput
						}
					}

					txScope.Logger().Debug("module upgrade started", "input", upgradeInput)
					if err := moduleLifecycle.Upgrade(tx.Context(), lifecycle.UpgradeRequest{Input: upgradeInput, WithDemo: withDemo}); err != nil {
						return xfmt.Errorf("error upgrading module %s: %w", moduleInput, err)
					}
					txScope.Logger().Debug("module upgraded", "input", upgradeInput)
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
	cmd.Flags().BoolVar(&withDemo, "with-demo", false, "Load demo data declared by package.json")
	cmd.Flags().StringVar(&cliCompatVersion, "cli-compat-version", "", "override CLI compatibility version for module compatibility checks")
	return cmd
}
