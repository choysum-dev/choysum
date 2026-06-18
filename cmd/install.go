// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cmd

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/choysum-dev/choysum/internal/esmresolver"
	"github.com/choysum-dev/choysum/internal/module/lifecycle"
	internalorigin "github.com/choysum-dev/choysum/internal/module/origin"
	"github.com/choysum-dev/choysum/pkg/config"
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
					compatibleVersion, compatErr := resolveCompatibleRegistryLatestVersion(ctx, env, runtimeOptions, parsed.ModuleName, resolvedCompat.Version)
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
							if strings.Contains(peekErr.Error(), "not found in modules path") {
								peekErr = xfmt.Errorf("module %s not found in modules path; run `choysum module fetch <module>@<version>` or `choysum install <module>@<version>`", moduleName)
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
					if parsed.Kind == internalorigin.InputKindLocal && !meta.IsCoreModule(moduleName) && strings.Contains(err.Error(), "not found in modules path") {
						err = xfmt.Errorf("module %s not found in modules path; run `choysum module fetch <module>@<version>` or `choysum install <module>@<version>`", moduleName)
					}
					attrs := []any{"error", err}
					attrs = append(attrs, moduleCommandFailureAttrs("install")...)
					attrs = append(attrs, moduleInstallFailureAttrs(input, moduleName)...)
					env.Logger().Error("module install failed", attrs...)
					os.Exit(1)
				}
			}

			// Auto-trigger type fetch after successful install for IDE support.
			runTypeFetchAfterInstall(env)
		},
	}
	cmd.Flags().BoolVar(&withDemo, "with-demo", false, "Load demo data declared by package.json")
	cmd.Flags().StringVar(&cliCompatVersion, "cli-compat-version", "", "override CLI compatibility version for module compatibility checks")
	return cmd
}

// runTypeFetchAfterInstall triggers a best-effort type fetch for all modules
// after a successful install. Failures are logged but do not fail the install.
func runTypeFetchAfterInstall(env scope.Scope) {
	runtimeOpts, ok := scope.PathsRuntimeOptionsFromScope(env)
	if !ok {
		return
	}
	modulesPath := runtimeOpts.ModulesPath
	if modulesPath == "" {
		return
	}
	defaultPath := runtimeOpts.DefaultChoysumPath
	if defaultPath == "" {
		defaultPath = ".choysum"
	}
	typesDir := filepath.Join(defaultPath, "pkg", "types")
	upstream := runtimeOpts.ESMUpstreamURL
	if upstream == "" {
		upstream = config.DefaultESMUpstreamURL
	}

	client := &http.Client{Timeout: 30 * time.Second}

	entries, err := os.ReadDir(modulesPath)
	if err != nil {
		env.Logger().Warn("type-fetch: read modules dir failed", "error", err)
		return
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		moduleDir := filepath.Join(modulesPath, entry.Name())
		if _, err := os.Stat(filepath.Join(moduleDir, "package.json")); err != nil {
			continue
		}
		results, err := esmresolver.FetchTypesForModule(client, upstream, typesDir, moduleDir)
		if err != nil {
			env.Logger().Warn("type-fetch: skipped module", "module", entry.Name(), "error", err)
			continue
		}
		for _, r := range results {
			if !r.FromCache {
				env.Logger().Info("type-fetch: downloaded", "module", entry.Name(), "package", r.Package+"@"+r.Version)

			}
		}
	}
}
