// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

// Package cmd exposes the Choysum CLI entrypoint and shared command wiring.
package cmd

import (
	"context"
	"os"
	"strings"
	"sync"

	cliruntime "github.com/choysum-dev/choysum/internal/cli/runtime"
	"github.com/choysum-dev/choysum/internal/config/snapshot"
	"github.com/choysum-dev/choysum/internal/logger"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/spf13/cobra"
	xfmt "golang.org/x/exp/errors/fmt"
)

const lightweightScopeAnnotation = "lightweightScope"

// Command owns the root Cobra command and the lazily initialized runtime scope used by subcommands.
type Command struct {
	rootCmd        *cobra.Command
	runtimeScope   scope.Scope
	runtimeOptions cliruntime.Options
}

// Execute runs the configured root Cobra command.
func (c *Command) Execute() error {
	return c.rootCmd.Execute()
}

// NewCommander constructs the Choysum root command and wires its subcommands.
func NewCommander(ctx context.Context, version string) *Command {
	rootCmd := &cobra.Command{
		Use:     "choysum",
		Short:   "Run the Choysum command-line interface",
		Version: version,
		Long: `Choysum provides commands for module lifecycle management,
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

			lightweightScope := shouldUseLightweightRuntimeScope(cmd)
			runtimeScope, runtimeOptions, err := newCommandRuntimeScope(ctx, cfgPath, lightweightScope)
			if err != nil {
				initErr = err
				return
			}
			c.runtimeScope = runtimeScope
			c.runtimeOptions = runtimeOptions
		})
		return initErr
	}

	envGetter := func() scope.Scope { return c.runtimeScope }
	cliOptionsGetter := func() cliruntime.Options { return c.runtimeOptions }

	c.rootCmd.AddCommand(
		newInstallCmd(envGetter),
		newUpgradeCmd(envGetter),
		newUninstallCmd(envGetter),
		newModuleCmd(envGetter, cliOptionsGetter),
		newRunCmd(),
		newTestCmd(envGetter, cliOptionsGetter),
		newTypeFetchCmd(envGetter),
		newI18nCmd(envGetter),
	)
	return c
}

func shouldUseLightweightRuntimeScope(cmd *cobra.Command) bool {
	for node := cmd; node != nil; node = node.Parent() {
		if node.Annotations == nil {
			continue
		}
		if value, ok := node.Annotations[lightweightScopeAnnotation]; ok && strings.EqualFold(strings.TrimSpace(value), "true") {
			return true
		}
	}
	return false
}

func newCommandRuntimeScope(ctx context.Context, cfgPath string, lightweight bool) (scope.Scope, cliruntime.Options, error) {
	cfg, err := config.LoadWithProvider(nil, cfgPath)
	if err != nil {
		return nil, cliruntime.Options{}, xfmt.Errorf("error reading config file %s: %w", cfgPath, err)
	}

	cfgOptions := cliruntime.NewScopeInputConfigOptions(snapshot.New(cfg))
	if cfgOptions == nil {
		return nil, cliruntime.Options{}, xfmt.Errorf("failed to parse config options")
	}
	runtimeOptions := cliruntime.Options{
		DefaultChoysumPath:    cfgOptions.DefaultChoysumPath,
		ModulesPath:           cfgOptions.ModulesPath,
		TmpPath:               cfgOptions.TmpPath,
		ModuleCatalogIndexURL: strings.TrimSpace(cfgOptions.ModuleCatalogIndexURL),
	}
	if err := runtimeOptions.Validate(); err != nil {
		return nil, cliruntime.Options{}, xfmt.Errorf("invalid cli runtime options: %w", err)
	}

	l := logger.NewLoggerWithWriter(cfg.Log, os.Stderr)
	input := cliruntime.NewCommandScopeInput(cfgOptions, runtimeOptions)

	var runtimeScope scope.Scope
	if lightweight {
		runtimeScope = cliruntime.NewScopeWithoutDB(ctx, input, l)
	} else {
		runtimeScope = scope.NewScope(ctx, input, l)
	}
	if runtimeScope == nil {
		return nil, cliruntime.Options{}, xfmt.Errorf("failed to initialize scope")
	}

	return runtimeScope, runtimeOptions, nil
}
