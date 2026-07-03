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
	logutil "github.com/choysum-dev/choysum/internal/logger"
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
			baseCtx := context.Background()
			if cmd != nil && cmd.Context() != nil {
				baseCtx = cmd.Context()
			}
			ctx, stop := signal.NotifyContext(baseCtx, os.Interrupt)
			defer stop()

			ctx = logutil.WithStderrProgressLine(ctx)

			runtimeOptionsValidated := false

			type upgradePlan struct {
				requestedInput string
				resolvedInput  string
			}
			exitUpgradeError := func(currentInput string, runErr error) {
				attrs := []any{"error", runErr}
				attrs = append(attrs, clioutput.ModuleCommandFailureAttrs("upgrade")...)
				attrs = append(attrs, clioutput.CurrentOrRequestedAttr("input", "inputs", currentInput, args)...)
				env.Logger().Error("module upgrade failed", attrs...)
				os.Exit(1)
			}

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
			plans := make([]upgradePlan, 0, len(args))
			for _, input := range args {
				moduleInput := strings.TrimSpace(input)
				currentInput = moduleInput
				if moduleInput == "" {
					exitUpgradeError(currentInput, xfmt.Errorf("module name is empty"))
				}

				parsed, parseErr := internalorigin.ParseInput(moduleInput)
				if parseErr != nil {
					exitUpgradeError(currentInput, xfmt.Errorf("error parsing module input %s: %w", moduleInput, parseErr))
				}

				resolvedInput := moduleInput
				switch parsed.Kind {
				case internalorigin.InputKindRegistry:
					if strings.EqualFold(strings.TrimSpace(parsed.Version), "latest") {
						if strings.TrimSpace(resolvedCompat.Version) == "" {
							exitUpgradeError(currentInput, xfmt.Errorf("ERR_CLI_COMPAT_VERSION_UNRESOLVED: Cannot resolve a CLI compatibility version in development mode. Provide '--cli-compat-version' or set 'CHOYSUM_CLI_COMPAT_VERSION'."))
						}
						indexURL, indexErr := resolveIndexURL()
						if indexErr != nil {
							exitUpgradeError(currentInput, indexErr)
						}
						compatibleVersion, compatErr := clicompat.ResolveCompatibleRegistryLatestVersion(ctx, env, indexURL, parsed.ModuleName, resolvedCompat.Version)
						if compatErr != nil {
							exitUpgradeError(currentInput, compatErr)
						}
						resolvedInput = strings.TrimSpace(parsed.ModuleName) + "@" + compatibleVersion
					}
				case internalorigin.InputKindLocal:
					if !runtimeOptionsValidated {
						if err := runtimeOptions.Validate(); err != nil {
							exitUpgradeError(currentInput, err)
						}
						runtimeOptionsValidated = true
					}
					registryBacked, bindErr := clicompat.HasRegistryOriginBinding(env, runtimeOptions.DefaultChoysumPath, parsed.LocalName)
					if bindErr != nil {
						exitUpgradeError(currentInput, bindErr)
					}
					if registryBacked {
						if strings.TrimSpace(resolvedCompat.Version) == "" {
							exitUpgradeError(currentInput, xfmt.Errorf("ERR_CLI_COMPAT_VERSION_UNRESOLVED: Cannot resolve a CLI compatibility version in development mode. Provide '--cli-compat-version' or set 'CHOYSUM_CLI_COMPAT_VERSION'."))
						}
						indexURL, indexErr := resolveIndexURL()
						if indexErr != nil {
							exitUpgradeError(currentInput, indexErr)
						}
						compatibleVersion, compatErr := clicompat.ResolveCompatibleRegistryLatestVersion(ctx, env, indexURL, parsed.LocalName, resolvedCompat.Version)
						if compatErr != nil {
							exitUpgradeError(currentInput, compatErr)
						}
						resolvedInput = parsed.LocalName + "@" + compatibleVersion
					}
				}
				plans = append(plans, upgradePlan{requestedInput: moduleInput, resolvedInput: resolvedInput})
			}

			upgradeScope := env.WithContext(ctx)
			compilerExecutor, err := jsexecutor.NewCompilerExecutor(upgradeScope)
			if err != nil {
				exitUpgradeError("", xfmt.Errorf("Error creating compiler executor: %w", err))
			}
			if err := compilerExecutor.Start(); err != nil {
				exitUpgradeError("", xfmt.Errorf("Error starting compiler executor: %w", err))
			}
			defer compilerExecutor.Stop()

			moduleLifecycle := lifecycle.NewService(upgradeScope, compilerExecutor)
			for _, plan := range plans {
				currentInput = plan.requestedInput
				upgradeScope.Logger().Debug("module upgrade started", "input", plan.resolvedInput)
				if err := moduleLifecycle.Upgrade(ctx, lifecycle.UpgradeRequest{Input: plan.resolvedInput, WithDemo: withDemo}); err != nil {
					_ = compilerExecutor.Stop()
					exitUpgradeError(currentInput, xfmt.Errorf("error upgrading module %s: %w", plan.requestedInput, err))
				}
				upgradeScope.Logger().Debug("module upgraded", "input", plan.resolvedInput)
			}
		},
	}
	cmd.Flags().BoolVar(&withDemo, "with-demo", false, "Load demo data declared by package.json")
	cmd.Flags().StringVar(&cliCompatVersion, "cli-compat-version", "", "override CLI compatibility version for module compatibility checks")
	return cmd
}
