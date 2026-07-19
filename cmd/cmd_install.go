// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cmd

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	clicompat "github.com/choysum-dev/choysum/internal/cli/compat"
	clioutput "github.com/choysum-dev/choysum/internal/cli/output"
	cliruntime "github.com/choysum-dev/choysum/internal/cli/runtime"
	"github.com/choysum-dev/choysum/internal/esmresolver"
	logutil "github.com/choysum-dev/choysum/internal/logger"
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
	var cliCompatVersion string
	cmd := &cobra.Command{
		Use:   "install <module|module@version> [<module|module@version>...]",
		Short: "Install Choysum Module",
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
			ensureInstallModulesTsconfig(env, runtimeOptions.ModulesPath)

			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
			defer stop()

			ctx = logutil.WithStderrProgressLine(ctx)

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
				if parsed.Kind == internalorigin.InputKindRegistry && strings.EqualFold(strings.TrimSpace(parsed.Version), "latest") {
					if strings.TrimSpace(resolvedCompat.Version) == "" {
						err = xfmt.Errorf("ERR_CLI_COMPAT_VERSION_UNRESOLVED: Cannot resolve a CLI compatibility version in development mode. Provide '--cli-compat-version' or set 'CHOYSUM_CLI_COMPAT_VERSION'.")
						env.Logger().Error("module install failed", "input", input, "error", err)
						os.Exit(1)
					}
					if err := runtimeOptions.Validate(); err != nil {
						env.Logger().Error("module install failed", "input", input, "error", err)
						os.Exit(1)
					}
					indexURL, indexErr := resolveModuleCatalogIndexURL(runtimeOptions)
					if indexErr != nil {
						env.Logger().Error("module install failed", "input", input, "error", indexErr)
						os.Exit(1)
					}
					compatibleVersion, compatErr := clicompat.ResolveCompatibleRegistryLatestVersion(ctx, env, indexURL, parsed.ModuleName, resolvedCompat.Version)
					if compatErr != nil {
						env.Logger().Error("module install failed", "input", input, "error", compatErr)
						os.Exit(1)
					}
					parsed.Version = compatibleVersion
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
				txHoldStarted := time.Now()
				err = txRoot.Transactor().Required(ctx, func(txScope scope.Scope, tx scope.Transaction) error {
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
				})
				lifecycle.LogInstallOuterTxHold(env.Logger(), "cli", txHoldStarted, err)
				if err != nil {
					if parsed.Kind == internalorigin.InputKindLocal && !meta.IsCoreModule(moduleName) {
						err = rewriteLocalInstallLookupError(moduleName, err)
					}
					attrs := []any{"error", err}
					attrs = append(attrs, clioutput.ModuleCommandFailureAttrs("install")...)
					attrs = append(attrs, clioutput.ModuleInstallFailureAttrs(input, moduleName)...)
					env.Logger().Error("module install failed", attrs...)
					os.Exit(1)
				}
			}
		},
	}
	cmd.Flags().BoolVar(&withDemo, "with-demo", false, "Load demo data declared by package.json")
	cmd.Flags().StringVar(&cliCompatVersion, "cli-compat-version", "", "override CLI compatibility version for module compatibility checks")
	return cmd
}

func rewriteLocalInstallLookupError(moduleName string, err error) error {
	if err == nil {
		return nil
	}
	if isLocalInstallMissingError(err) {
		return xfmt.Errorf("module %s not found in modules path; run `choysum module fetch <module>@<version>` or `choysum install <module>@<version>`", moduleName)
	}
	if isLocalInstallRegistryFallbackError(err) {
		return xfmt.Errorf("%w; run `choysum module fetch <module>@<version>` or `choysum install <module>@<version>`", err)
	}
	return err
}

func isLocalInstallMissingError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "not found in modules path")
}

func isLocalInstallRegistryFallbackError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "not found locally and registry fallback failed")
}

func ensureInstallModulesTsconfig(env scope.Scope, modulesPath string) {
	modulesPath = strings.TrimSpace(modulesPath)
	if modulesPath == "" {
		return
	}
	tsconfigPath := filepath.Join(modulesPath, "tsconfig.json")
	if err := esmresolver.UpdateTsconfigPaths(tsconfigPath, nil); err != nil {
		env.Logger().Warn("install: ensure modules tsconfig failed", "path", tsconfigPath, "error", err)
		return
	}
	env.Logger().Debug("install: ensured modules tsconfig", "path", tsconfigPath)
}
