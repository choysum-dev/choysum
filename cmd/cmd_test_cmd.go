// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cmd

import (
	cliruntime "github.com/choysum-dev/choysum/internal/cli/runtime"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/spf13/cobra"
	xfmt "golang.org/x/exp/errors/fmt"
)

func newTestCmd(envGetter func() scope.Scope, runtimeOptionsGetter func() cliruntime.Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "test",
		Short: "Run test commands",
		Annotations: map[string]string{
			lightweightScopeAnnotation: "true",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return xfmt.Errorf("test: requires a subcommand (unit|typecheck|e2e)")
		},
	}
	cmd.AddCommand(
		newTestUnitCmd(envGetter, runtimeOptionsGetter),
		newTypecheckCmd(envGetter, runtimeOptionsGetter),
		newE2ECmd(envGetter, runtimeOptionsGetter),
	)
	return cmd
}
