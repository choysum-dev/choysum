// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cmd

import (
	"context"
	"os"

	"github.com/choysum-dev/choysum/internal/cli/initialsetup"
	"github.com/choysum-dev/choysum/internal/provision"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/spf13/cobra"
	xfmt "golang.org/x/exp/errors/fmt"
)

// moduleSetupFlags holds the first-run setup flags shared by install and upgrade.
type moduleSetupFlags struct {
	mode          string
	adminPassword string
}

// addModuleSetupFlags registers --setup and --admin-password on cmd.
func addModuleSetupFlags(cmd *cobra.Command) *moduleSetupFlags {
	flags := &moduleSetupFlags{}
	cmd.Flags().StringVar(&flags.mode, "setup", "", initialsetup.SetupFlagUsage)
	cmd.Flags().StringVar(&flags.adminPassword, "admin-password", "", initialsetup.PasswordFlagUsage)
	return flags
}

// moduleSetupPlan is the validated setup input plus the workspace phase read
// before the command changed any module.
type moduleSetupPlan struct {
	opts   initialsetup.Options
	before provision.Phase
}

// prepare validates the setup flags and records the current phase; call it
// before the first module is installed or upgraded.
func (f *moduleSetupFlags) prepare(cmd *cobra.Command, env scope.Scope) (moduleSetupPlan, error) {
	opts, err := initialsetup.ResolveOptions(
		f.mode,
		f.adminPassword,
		cmd.Flags().Changed("admin-password"),
		os.Getenv(initialsetup.PasswordEnv),
		cmd.ErrOrStderr(),
	)
	if err != nil {
		return moduleSetupPlan{}, err
	}
	before, err := provision.Resolve(env)
	if err != nil {
		return moduleSetupPlan{}, xfmt.Errorf("resolve setup phase: %w", err)
	}
	return moduleSetupPlan{opts: opts, before: before}, nil
}

// finish runs first-run setup after all modules are done and exits 1 on failure.
func (p moduleSetupPlan) finish(ctx context.Context, cmd *cobra.Command, env scope.Scope) {
	deps := initialsetup.ScopeDeps(env.WithContext(ctx))
	if _, err := initialsetup.Run(ctx, deps, p.opts, p.before, cmd.OutOrStdout()); err != nil {
		env.Logger().Error("initial setup failed; workspace stays pending, finish it at /web/setup or rerun with --setup=complete", "error", err)
		os.Exit(1)
	}
}
