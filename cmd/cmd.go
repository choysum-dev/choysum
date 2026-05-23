// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

// Package cmd exposes the Choysum CLI entrypoint and shared command wiring.
package cmd

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/choysum-dev/choysum/internal/config/snapshot"
	"github.com/choysum-dev/choysum/internal/logger"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/spf13/cobra"
	xfmt "golang.org/x/exp/errors/fmt"
)

// Command owns the root Cobra command and the lazily initialized runtime scope used by subcommands.
type Command struct {
	rootCmd           *cobra.Command
	runtimeScope      scope.Scope
	cliRuntimeOptions cliRuntimeOptions
}

// Execute runs the configured root Cobra command.
func (c *Command) Execute() error {
	return c.rootCmd.Execute()
}

// NewCommander constructs the Choysum root command and wires its subcommands.
func NewCommander(ctx context.Context) *Command {
	rootCmd := &cobra.Command{
		Use:   "choysum",
		Short: "Run the Choysum command-line interface",
		Long: `Choysum provides commands for module lifecycle management, registry operations,
runtime startup, and test workflows.

Use --config to point to a workspace config file when you want to override the
built-in defaults or load a non-default workspace config.`,
	}
	c := &Command{
		rootCmd: rootCmd,
	}

	var configPath string
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "", "config file (default is ./config.yaml when present, otherwise built-in defaults)")

	var initOnce sync.Once
	var initErr error
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if cmd != nil {
			name := cmd.Name()
			if name == "run" {
				return nil
			}
		}
		initOnce.Do(func() {
			cfgPath := configPath
			if cfgPath == "" {
				var err error
				cfgPath, err = config.DefaultConfigPath()
				if err != nil {
					initErr = xfmt.Errorf("resolve default config file failed: %w", err)
					return
				}
			}

			runtimeScope, runtimeOptions, err := newCommandRuntimeScope(ctx, cfgPath)
			if err != nil {
				initErr = err
				return
			}
			c.runtimeScope = runtimeScope
			c.cliRuntimeOptions = runtimeOptions
		})
		return initErr
	}

	envGetter := func() scope.Scope { return c.runtimeScope }
	cliOptionsGetter := func() cliRuntimeOptions { return c.cliRuntimeOptions }

	c.rootCmd.AddCommand(
		newInstallCmd(envGetter),
		newUpgradeCmd(envGetter),
		newUninstallCmd(envGetter),
		newRegistryCmd(envGetter, cliOptionsGetter),
		newModuleCmd(envGetter, cliOptionsGetter),
		newRunCmd(),
		newTestCmd(envGetter, cliOptionsGetter),
	)
	return c
}

func newCommandRuntimeScope(ctx context.Context, cfgPath string) (scope.Scope, cliRuntimeOptions, error) {
	cfg, err := config.LoadWithProvider(nil, cfgPath)
	if err != nil {
		return nil, cliRuntimeOptions{}, xfmt.Errorf("error reading config file %s: %w", cfgPath, err)
	}

	cfgOptions := newScopeInputConfigOptions(snapshot.New(cfg))
	runtimeOptions := newCliRuntimeOptionsFromScopeInputOptions(cfgOptions)
	if err := runtimeOptions.Validate(); err != nil {
		return nil, cliRuntimeOptions{}, xfmt.Errorf("invalid cli runtime options: %w", err)
	}

	l := logger.NewLoggerWithWriter(cfg.Log, os.Stderr)
	runtimeScope := scope.NewScope(ctx, newCommandRuntimeScopeInput(cfgOptions, runtimeOptions), l)
	if runtimeScope == nil {
		return nil, cliRuntimeOptions{}, xfmt.Errorf("failed to initialize scope")
	}

	return runtimeScope, runtimeOptions, nil
}

func printErrorBlock(errMsg, reason, next string) {
	printCLIOutputLine("ERROR", errMsg)
	printCLIOutputLine("REASON", reason)
	printCLIOutputLine("NEXT", next)
}

func printCLIWarning(message string) {
	printCLIOutputLine("WARN", message)
}

func printCLIError(message string) {
	printCLIOutputLine("ERROR", message)
}

func printCLIOutputLine(label, message string) {
	fmt.Fprintf(os.Stderr, "%s: %s\n", label, message)
}
