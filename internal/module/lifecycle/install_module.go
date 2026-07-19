// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"context"
	"strings"

	"github.com/choysum-dev/choysum/pkg/jsexecutor"
	"github.com/choysum-dev/choysum/pkg/scope"
	xfmt "golang.org/x/exp/errors/fmt"
)

// InstallModuleRequest is the unified bootstrap / CLI install entry input.
type InstallModuleRequest struct {
	// Input is a local module name or registry ref (for example "document" or "pkg@1.2.3").
	Input    string
	WithDemo bool
}

// PrepareInstall materializes the install closure outside any install commit TX.
func PrepareInstall(ctx context.Context, runtimeScope scope.Scope, input string, opts ...Option) (*PrefetchedInstall, error) {
	return PrefetchInstallModules(ctx, runtimeScope, input, opts...)
}

// InstallModule is the shared production entry for bootstrap and CLI install:
// Prepare (prefetch) then Install (per-module short commit TX + pipeline finalize).
func InstallModule(ctx context.Context, runtimeScope scope.Scope, jsExecutor jsexecutor.ScriptExecutor, req InstallModuleRequest, opts ...Option) error {
	if runtimeScope == nil {
		return xfmt.Errorf("scope is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	input := strings.TrimSpace(req.Input)
	if input == "" {
		return xfmt.Errorf("module name is empty")
	}

	prepared, err := PrepareInstall(ctx, runtimeScope, input, opts...)
	if err != nil {
		return err
	}
	if prepared == nil || strings.TrimSpace(prepared.RootName) == "" {
		return xfmt.Errorf("prepared install root is empty")
	}

	installCtx := WithPrefetchedInstallModules(ctx, prepared.Modules)
	installScope := runtimeScope.WithContext(installCtx)
	if installScope == nil {
		installScope = runtimeScope
	}
	return NewService(installScope, jsExecutor, opts...).Install(installCtx, InstallRequest{
		Name:     prepared.RootName,
		WithDemo: req.WithDemo,
	})
}
