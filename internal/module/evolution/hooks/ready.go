// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package hooks

import (
	"context"
	"fmt"
	"strings"

	"github.com/choysum-dev/choysum/internal/module/plan"
	"github.com/choysum-dev/choysum/pkg/jsexecutor"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
)

// RunReadyOptions configures application-startup ready hook execution.
type RunReadyOptions struct {
	// ApplicationNames limits modules to those whose ApplicationStr is in this set.
	// Empty means no application filter (all installed modules that declare ready).
	ApplicationNames []string
	// Modules is the candidate installed-module set (typically status=installed).
	Modules []meta.Module
	// InvokeOnly runs ready against the shared application runtime (no Bundle/Reload).
	// Default for process startup after Track A seeded entry registration.
	InvokeOnly bool
	// ReuseExecutorScripts amortizes script reloads across modules (non-invoke path).
	ReuseExecutorScripts bool
	// PreserveExecutorScripts is a fallback when InvokeOnly is false (see Runner.RunOptions).
	PreserveExecutorScripts bool
}

// RunReadyForInstalledModules runs PhaseReady for modules that declare @HookReady,
// in depends topological order among that set. Failures are returned (phase is required).
func RunReadyForInstalledModules(
	ctx context.Context,
	runtimeScope scope.Scope,
	jsExecutor jsexecutor.ScriptExecutor,
	opts RunReadyOptions,
) error {
	if runtimeScope == nil {
		return nil
	}
	appSet := map[string]bool{}
	for _, name := range opts.ApplicationNames {
		name = strings.TrimSpace(name)
		if name != "" {
			appSet[name] = true
		}
	}
	candidates := make([]meta.Module, 0, len(opts.Modules))
	for _, mod := range opts.Modules {
		mod := mod
		if strings.TrimSpace(mod.Name) == "" {
			continue
		}
		if len(appSet) > 0 {
			app := strings.TrimSpace(mod.ApplicationStr)
			if app == "" || !appSet[app] {
				continue
			}
		}
		if !moduleSourceDeclaresHookPhase(&mod, PhaseReady) {
			continue
		}
		candidates = append(candidates, mod)
	}
	if len(candidates) == 0 {
		return nil
	}
	if jsExecutor == nil {
		return fmt.Errorf("js executor is nil")
	}
	ordered, cyclic := plan.StableTopoSortModules(candidates)
	if cyclic && runtimeScope.Logger() != nil {
		runtimeScope.Logger().Warn("ready hook module order has dependency cycle; using alphabetical fallback")
	}
	for i := range ordered {
		mod := ordered[i]
		runner, err := NewRunner(runtimeScope, jsExecutor, &mod)
		if err != nil {
			return fmt.Errorf("prepare ready hooks for module %s: %w", mod.Name, err)
		}
		if runner == nil {
			continue
		}
		if err := runner.RunPhase(ctx, PhaseReady, RunOptions{
			InvokeOnly:              opts.InvokeOnly,
			ReuseExecutorScripts:    opts.ReuseExecutorScripts,
			PreserveExecutorScripts: opts.PreserveExecutorScripts,
		}); err != nil {
			return fmt.Errorf("ready hook for module %s: %w", mod.Name, err)
		}
	}
	return nil
}
