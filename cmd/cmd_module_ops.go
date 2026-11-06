// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/choysum-dev/choysum/internal/module/lifecycle"
	"github.com/choysum-dev/choysum/internal/module/ops"
	"github.com/choysum-dev/choysum/internal/module/plan"
	"github.com/choysum-dev/choysum/internal/state/lease"
	"github.com/choysum-dev/choysum/pkg/bus"
	"github.com/choysum-dev/choysum/pkg/jsexecutor"
	"github.com/choysum-dev/choysum/pkg/scope"
	statepkg "github.com/choysum-dev/choysum/pkg/state"
	xfmt "golang.org/x/exp/errors/fmt"
)

// cliModuleOpOptions drives a synchronous CLI Plan → Apply → Run.
type cliModuleOpOptions struct {
	Action          string
	ModuleName      string
	Origin          string
	LifecycleInput  string
	WithDemo        bool
	SkipWebShell    bool
	WebShellProfile plan.WebShellProfile
	// Prefetch is optional; when nil on install, PrepareInstall runs after the executor starts.
	Prefetch *lifecycle.PrefetchedInstall
}

// Overridable for tests that force executor create/start failures without a full engine.
var newCLICompilerExecutor = jsexecutor.NewCompilerExecutor

// runCLIModuleOp assembles an ops host and runs Plan → Apply → Run synchronously.
func runCLIModuleOp(ctx context.Context, env scope.Scope, opts cliModuleOpOptions) error {
	if env == nil {
		return xfmt.Errorf("scope is not initialized")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	action := strings.TrimSpace(opts.Action)
	moduleName := strings.TrimSpace(opts.ModuleName)
	lifecycleInput := strings.TrimSpace(opts.LifecycleInput)
	if action == "" {
		return xfmt.Errorf("ops action is required")
	}
	if moduleName == "" && lifecycleInput == "" && action != ops.ActionSync && action != ops.ActionLoadDemo {
		return xfmt.Errorf("module name is empty")
	}

	baseScope := env.WithContext(ctx)
	if baseScope == nil {
		baseScope = env
	}

	compilerExecutor, err := newCLICompilerExecutor(baseScope)
	if err != nil {
		return xfmt.Errorf("Error creating compiler executor: %w", err)
	}
	if err := compilerExecutor.Start(); err != nil {
		return xfmt.Errorf("Error starting compiler executor: %w", err)
	}
	defer func() { _ = compilerExecutor.Stop() }()

	runCtx := ctx
	prefetch := opts.Prefetch
	if action == ops.ActionInstall && prefetch == nil {
		prepareInput := lifecycleInput
		if prepareInput == "" {
			prepareInput = moduleName
		}
		prepared, prepareErr := lifecycle.PrepareInstall(ctx, baseScope, prepareInput)
		if prepareErr != nil {
			return prepareErr
		}
		if prepared == nil || strings.TrimSpace(prepared.RootName) == "" {
			return xfmt.Errorf("prepared install root is empty")
		}
		prefetch = prepared
		if moduleName == "" {
			moduleName = strings.TrimSpace(prepared.RootName)
		}
	}
	if prefetch != nil && prefetch.Modules != nil {
		runCtx = lifecycle.WithPrefetchedInstallModules(ctx, prefetch.Modules)
		if moduleName == "" {
			moduleName = strings.TrimSpace(prefetch.RootName)
		}
	}
	if moduleName == "" && action != ops.ActionSync && action != ops.ActionLoadDemo {
		return xfmt.Errorf("module name is empty")
	}

	opScope := env.WithContext(runCtx)
	if opScope == nil {
		opScope = env
	}

	lockerFactory := statepkg.LockerFactory(func(runtimeScope scope.Scope) statepkg.Locker {
		return lease.New(runtimeScope)
	})
	host := &ops.Ops{
		Scope:         opScope,
		Lifecycle:     lifecycle.NewService(opScope, compilerExecutor, lifecycle.WithLockerFactory(lockerFactory)),
		Resolver:      lifecycle.NewModuleManager(opScope, compilerExecutor, lifecycle.WithLockerFactory(lockerFactory)),
		LockerFactory: lockerFactory,
		Events:        bus.EnsureHost(opScope),
	}

	planRes, err := host.Plan(runCtx, ops.PlanRequest{
		ModuleName:      moduleName,
		Action:          action,
		Origin:          strings.TrimSpace(opts.Origin),
		SkipWebShell:    opts.SkipWebShell,
		WebShellProfile: string(opts.WebShellProfile),
		LifecycleInput:  lifecycleInput,
	})
	if err != nil {
		return err
	}
	if len(planRes.Blockers) > 0 {
		return formatPlanBlockers(planRes.Blockers)
	}
	if err := host.Apply(runCtx, ops.ApplyRequest{Id: planRes.Id}); err != nil {
		return err
	}
	return host.Run(runCtx, ops.RunRequest{Id: planRes.Id, WithDemo: opts.WithDemo})
}

func formatPlanBlockers(blockers []ops.PlanIssue) error {
	parts := make([]string, 0, len(blockers))
	for _, b := range blockers {
		msg := strings.TrimSpace(b.Message)
		if msg == "" {
			msg = strings.TrimSpace(b.Code)
		}
		if msg == "" {
			continue
		}
		parts = append(parts, msg)
	}
	if len(parts) == 0 {
		return fmt.Errorf("ops: plan has blockers")
	}
	return fmt.Errorf("ops: plan blocked: %s", strings.Join(parts, "; "))
}
