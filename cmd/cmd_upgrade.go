// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cmd

import (
	"context"
	"os"
	"os/signal"
	"strings"

	clicompat "github.com/choysum-dev/choysum/internal/cli/compat"
	clioutput "github.com/choysum-dev/choysum/internal/cli/output"
	cliruntime "github.com/choysum-dev/choysum/internal/cli/runtime"
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
			runtimeVersion := ""
			if cmd != nil && cmd.Root() != nil {
				runtimeVersion = strings.TrimSpace(cmd.Root().Version)
			}
			resolvedCompat, err := clicompat.ResolveCLICompatVersion(cliCompatVersion, runtimeVersion, strings.TrimSpace(os.Getenv(clicompat.CLICompatVersionEnv)))
			if err != nil {
				env.Logger().Error("module compatibility version resolution failed", "error", err)
				os.Exit(1)
			}
			runtimeOptions := cliruntime.OptionsFromScope(env)
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
			defer stop()

			// Create a transaction-bound module manager for the upgrade batch.
			txRoot := env.WithContext(ctx)
			currentInput := ""
			resolvedIndexURL := ""
			resolveIndexURL := func() (string, error) {
				if strings.TrimSpace(resolvedIndexURL) != "" {
					return resolvedIndexURL, nil
				}
				indexURL, err := resolveModuleCatalogIndexURL(runtimeOptions)
				if err != nil {
					return "", err
				}
				resolvedIndexURL = indexURL
				return resolvedIndexURL, nil
			}
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
							indexURL, indexErr := resolveIndexURL()
							if indexErr != nil {
								return indexErr
							}
							compatibleVersion, compatErr := clicompat.ResolveCompatibleRegistryLatestVersion(tx.Context(), txScope, indexURL, parsed.ModuleName, resolvedCompat.Version)
							if compatErr != nil {
								return compatErr
							}
							upgradeInput = strings.TrimSpace(parsed.ModuleName) + "@" + compatibleVersion
						}
					case internalorigin.InputKindLocal:
						if err := runtimeOptions.Validate(); err != nil {
							return err
						}
						registryBacked, bindErr := clicompat.HasRegistryOriginBinding(txScope, runtimeOptions.DefaultChoysumPath, parsed.LocalName)
						if bindErr != nil {
							return bindErr
						}
						if registryBacked {
							if strings.TrimSpace(resolvedCompat.Version) == "" {
								return xfmt.Errorf("ERR_CLI_COMPAT_VERSION_UNRESOLVED: Cannot resolve a CLI compatibility version in development mode. Provide '--cli-compat-version' or set 'CHOYSUM_CLI_COMPAT_VERSION'.")
							}
							indexURL, indexErr := resolveIndexURL()
							if indexErr != nil {
								return indexErr
							}
							compatibleVersion, compatErr := clicompat.ResolveCompatibleRegistryLatestVersion(tx.Context(), txScope, indexURL, parsed.LocalName, resolvedCompat.Version)
							if compatErr != nil {
								return compatErr
							}
							upgradeInput = parsed.LocalName + "@" + compatibleVersion
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
				attrs = append(attrs, clioutput.ModuleCommandFailureAttrs("upgrade")...)
				attrs = append(attrs, clioutput.CurrentOrRequestedAttr("input", "inputs", currentInput, args)...)
				env.Logger().Error("module upgrade failed", attrs...)
				os.Exit(1)
			}
		},
	}
	cmd.Flags().BoolVar(&withDemo, "with-demo", false, "Load demo data declared by package.json")
	cmd.Flags().StringVar(&cliCompatVersion, "cli-compat-version", "", "override CLI compatibility version for module compatibility checks")
	return cmd
}
