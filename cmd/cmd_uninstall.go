// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cmd

import (
	"context"
	"os"
	"os/signal"
	"strings"

	clioutput "github.com/choysum-dev/choysum/internal/cli/output"
	logutil "github.com/choysum-dev/choysum/internal/logger"
	"github.com/choysum-dev/choysum/internal/module/ops"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/spf13/cobra"
	xfmt "golang.org/x/exp/errors/fmt"
)

func newUninstallCmd(envGetter func() scope.Scope) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall Choysum Module",
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
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
			defer stop()

			ctx = logutil.WithStderrProgressLine(ctx)

			uninstallScope := env.WithContext(ctx)
			currentModule := ""
			for _, name := range args {
				currentModule = name
				moduleName := strings.TrimSpace(name)
				uninstallScope.Logger().Debug("module uninstall started", "module", moduleName)
				if err := runCLIModuleOp(ctx, env, cliModuleOpOptions{
					Action:     ops.ActionUninstall,
					ModuleName: moduleName,
					Origin:     "local",
				}); err != nil {
					attrs := []any{"error", xfmt.Errorf("error uninstalling module %s: %w", name, err)}
					attrs = append(attrs, clioutput.ModuleCommandFailureAttrs("uninstall")...)
					attrs = append(attrs, clioutput.CurrentOrRequestedAttr("module", "modules", currentModule, args)...)
					env.Logger().Error("module uninstall failed", attrs...)
					os.Exit(1)
				}
				uninstallScope.Logger().Debug("module uninstalled", "module", moduleName)
			}
		},
	}
	return cmd
}
