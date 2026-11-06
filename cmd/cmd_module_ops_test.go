// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cmd

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/internal/module/lifecycle"
	"github.com/choysum-dev/choysum/internal/module/ops"
	"github.com/choysum-dev/choysum/internal/state/lease"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsexecutor"
	pkgmeta "github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	statepkg "github.com/choysum-dev/choysum/pkg/state"
)

func TestFormatPlanBlockers(t *testing.T) {
	err := formatPlanBlockers(nil)
	if err == nil || !strings.Contains(err.Error(), "has blockers") {
		t.Fatalf("empty blockers = %v", err)
	}
	err = formatPlanBlockers([]ops.PlanIssue{
		{Code: "A", Message: "first"},
		{Code: "B", Message: ""},
		{Message: "  "},
	})
	if err == nil || !strings.Contains(err.Error(), "first") || !strings.Contains(err.Error(), "B") {
		t.Fatalf("blockers = %v", err)
	}
}

func TestRunCLIModuleOpValidation(t *testing.T) {
	if err := runCLIModuleOp(context.Background(), nil, cliModuleOpOptions{Action: ops.ActionInstall, ModuleName: "core"}); err == nil || !strings.Contains(err.Error(), "scope") {
		t.Fatalf("nil env = %v", err)
	}
	env := &commandTestScope{ctx: context.Background()}
	if err := runCLIModuleOp(context.Background(), env, cliModuleOpOptions{ModuleName: "core"}); err == nil || !strings.Contains(err.Error(), "action") {
		t.Fatalf("empty action = %v", err)
	}
	if err := runCLIModuleOp(context.Background(), env, cliModuleOpOptions{Action: ops.ActionInstall}); err == nil || !strings.Contains(err.Error(), "module name") {
		t.Fatalf("empty module = %v", err)
	}
	if err := runCLIModuleOp(context.Background(), env, cliModuleOpOptions{Action: ops.ActionUninstall, LifecycleInput: "x"}); err == nil || !strings.Contains(err.Error(), "module name") {
		t.Fatalf("uninstall empty name = %v", err)
	}
}

func TestRunCLIModuleOpExecutorCreateError(t *testing.T) {
	prev := newCLICompilerExecutor
	t.Cleanup(func() { newCLICompilerExecutor = prev })
	newCLICompilerExecutor = func(_ scope.Scope, _ ...jsexecutor.Option) (jsexecutor.JsExecutor, error) {
		return nil, errors.New("no engine")
	}
	env := &commandTestScope{ctx: context.Background()}
	err := runCLIModuleOp(context.Background(), env, cliModuleOpOptions{
		Action:     ops.ActionUninstall,
		ModuleName: "core",
	})
	if err == nil || !strings.Contains(err.Error(), "Error creating compiler executor") {
		t.Fatalf("create err = %v", err)
	}
}

func TestRunCLIModuleOpExecutorStartError(t *testing.T) {
	prev := newCLICompilerExecutor
	t.Cleanup(func() { newCLICompilerExecutor = prev })
	newCLICompilerExecutor = func(_ scope.Scope, _ ...jsexecutor.Option) (jsexecutor.JsExecutor, error) {
		return &stubCLIExecutor{startErr: errors.New("start boom")}, nil
	}
	err := runCLIModuleOp(context.Background(), &commandTestScope{ctx: context.Background()}, cliModuleOpOptions{
		Action:     ops.ActionUninstall,
		ModuleName: "core",
	})
	if err == nil || !strings.Contains(err.Error(), "Error starting compiler executor") {
		t.Fatalf("start err = %v", err)
	}
}

func TestRunCLIModuleOpNilContextAndNilWithContext(t *testing.T) {
	prevExec := newCLICompilerExecutor
	prevHost := newCLIOpsHost
	t.Cleanup(func() {
		newCLICompilerExecutor = prevExec
		newCLIOpsHost = prevHost
	})
	newCLICompilerExecutor = func(_ scope.Scope, _ ...jsexecutor.Option) (jsexecutor.JsExecutor, error) {
		return &stubCLIExecutor{}, nil
	}
	newCLIOpsHost = func(scope.Scope, jsexecutor.JsExecutor, statepkg.LockerFactory) cliOpsHost {
		return &stubCLIOpsHost{plan: ops.PlanResult{Id: "op1"}}
	}
	env := &nilWithContextScope{commandTestScope: commandTestScope{ctx: context.Background()}}
	if err := runCLIModuleOp(nil, env, cliModuleOpOptions{Action: ops.ActionUninstall, ModuleName: "core"}); err != nil {
		t.Fatalf("nil ctx: %v", err)
	}
}

func TestRunCLIModuleOpPrepareInstallPaths(t *testing.T) {
	prevExec := newCLICompilerExecutor
	prevPrep := prepareCLIInstall
	prevHost := newCLIOpsHost
	t.Cleanup(func() {
		newCLICompilerExecutor = prevExec
		prepareCLIInstall = prevPrep
		newCLIOpsHost = prevHost
	})
	newCLICompilerExecutor = func(_ scope.Scope, _ ...jsexecutor.Option) (jsexecutor.JsExecutor, error) {
		return &stubCLIExecutor{}, nil
	}
	newCLIOpsHost = func(scope.Scope, jsexecutor.JsExecutor, statepkg.LockerFactory) cliOpsHost {
		return &stubCLIOpsHost{plan: ops.PlanResult{Id: "op1"}}
	}

	prepareCLIInstall = func(context.Context, scope.Scope, string, ...lifecycle.Option) (*lifecycle.PrefetchedInstall, error) {
		return nil, errors.New("prepare boom")
	}
	err := runCLIModuleOp(context.Background(), &commandTestScope{ctx: context.Background()}, cliModuleOpOptions{
		Action:     ops.ActionInstall,
		ModuleName: "core",
	})
	if err == nil || !strings.Contains(err.Error(), "prepare boom") {
		t.Fatalf("prepare err = %v", err)
	}

	prepareCLIInstall = func(context.Context, scope.Scope, string, ...lifecycle.Option) (*lifecycle.PrefetchedInstall, error) {
		return &lifecycle.PrefetchedInstall{RootName: "  "}, nil
	}
	err = runCLIModuleOp(context.Background(), &commandTestScope{ctx: context.Background()}, cliModuleOpOptions{
		Action:     ops.ActionInstall,
		ModuleName: "core",
	})
	if err == nil || !strings.Contains(err.Error(), "prepared install root is empty") {
		t.Fatalf("empty root = %v", err)
	}

	var sawInput string
	prepareCLIInstall = func(_ context.Context, _ scope.Scope, input string, _ ...lifecycle.Option) (*lifecycle.PrefetchedInstall, error) {
		sawInput = input
		return &lifecycle.PrefetchedInstall{
			RootName: "core",
			Modules:  map[string]*pkgmeta.Module{"core": {Name: "core"}},
		}, nil
	}
	host := &stubCLIOpsHost{plan: ops.PlanResult{Id: "op1"}}
	newCLIOpsHost = func(scope.Scope, jsexecutor.JsExecutor, statepkg.LockerFactory) cliOpsHost { return host }
	err = runCLIModuleOp(context.Background(), &commandTestScope{ctx: context.Background()}, cliModuleOpOptions{
		Action: ops.ActionInstall,
		// empty LifecycleInput → prepare uses ModuleName
		ModuleName: "core",
	})
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	if sawInput != "core" {
		t.Fatalf("prepare input = %q", sawInput)
	}
	if host.lastPlan.ModuleName != "core" {
		t.Fatalf("plan module = %#v", host.lastPlan)
	}
}

func TestRunCLIModuleOpPrefetchAndEmptyNameAfterPrefetch(t *testing.T) {
	prevExec := newCLICompilerExecutor
	prevHost := newCLIOpsHost
	t.Cleanup(func() {
		newCLICompilerExecutor = prevExec
		newCLIOpsHost = prevHost
	})
	newCLICompilerExecutor = func(_ scope.Scope, _ ...jsexecutor.Option) (jsexecutor.JsExecutor, error) {
		return &stubCLIExecutor{}, nil
	}
	newCLIOpsHost = func(s scope.Scope, _ jsexecutor.JsExecutor, lf statepkg.LockerFactory) cliOpsHost {
		if lf != nil {
			_ = lf(s) // cover lockerFactory closure in runCLIModuleOp
		}
		return &stubCLIOpsHost{plan: ops.PlanResult{Id: "op1"}}
	}

	// lifecycleInput keeps early guard open; empty RootName leaves moduleName empty.
	err := runCLIModuleOp(context.Background(), &commandTestScope{ctx: context.Background()}, cliModuleOpOptions{
		Action:         ops.ActionUpgrade,
		LifecycleInput: "core@1.0.0",
	})
	if err == nil || !strings.Contains(err.Error(), "module name is empty") {
		t.Fatalf("empty after upgrade input-only = %v", err)
	}

	host := &stubCLIOpsHost{plan: ops.PlanResult{Id: "op1"}}
	newCLIOpsHost = func(s scope.Scope, _ jsexecutor.JsExecutor, lf statepkg.LockerFactory) cliOpsHost {
		if lf != nil {
			_ = lf(s)
		}
		return host
	}
	err = runCLIModuleOp(context.Background(), &commandTestScope{ctx: context.Background()}, cliModuleOpOptions{
		Action:         ops.ActionInstall,
		LifecycleInput: "ignored@version",
		ModuleName:     "wrong",
		Prefetch: &lifecycle.PrefetchedInstall{
			RootName: "core",
			Modules:  map[string]*pkgmeta.Module{"core": {Name: "core"}},
		},
	})
	if err != nil {
		t.Fatalf("prefetch install: %v", err)
	}
	if host.lastPlan.ModuleName != "core" {
		t.Fatalf("expected RootName override, got %#v", host.lastPlan)
	}
}

func TestDefaultNewCLIOpsHost(t *testing.T) {
	env := &commandTestScope{ctx: context.Background()}
	host := defaultNewCLIOpsHost(env, &stubCLIExecutor{}, func(scope.Scope) statepkg.Locker {
		return lease.New(env)
	})
	if host == nil {
		t.Fatal("expected ops host")
	}
}

func TestRunCLIModuleOpPlanBlockersAndApplyRunErrors(t *testing.T) {
	prevExec := newCLICompilerExecutor
	prevHost := newCLIOpsHost
	t.Cleanup(func() {
		newCLICompilerExecutor = prevExec
		newCLIOpsHost = prevHost
	})
	exec := &stubCLIExecutor{}
	newCLICompilerExecutor = func(_ scope.Scope, _ ...jsexecutor.Option) (jsexecutor.JsExecutor, error) {
		return exec, nil
	}

	newCLIOpsHost = func(scope.Scope, jsexecutor.JsExecutor, statepkg.LockerFactory) cliOpsHost {
		return &stubCLIOpsHost{plan: ops.PlanResult{
			Id:       "op1",
			Blockers: []ops.PlanIssue{{Code: "X", Message: "blocked"}},
		}}
	}
	err := runCLIModuleOp(context.Background(), &commandTestScope{ctx: context.Background()}, cliModuleOpOptions{
		Action:     ops.ActionUninstall,
		ModuleName: "core",
		Executor:   exec, // shared executor path
	})
	if err == nil || !strings.Contains(err.Error(), "blocked") {
		t.Fatalf("blockers = %v", err)
	}

	newCLIOpsHost = func(scope.Scope, jsexecutor.JsExecutor, statepkg.LockerFactory) cliOpsHost {
		return &stubCLIOpsHost{plan: ops.PlanResult{Id: "op1"}, applyErr: errors.New("apply boom")}
	}
	err = runCLIModuleOp(context.Background(), &commandTestScope{ctx: context.Background()}, cliModuleOpOptions{
		Action:     ops.ActionUninstall,
		ModuleName: "core",
	})
	if err == nil || !strings.Contains(err.Error(), "apply boom") {
		t.Fatalf("apply err = %v", err)
	}

	newCLIOpsHost = func(scope.Scope, jsexecutor.JsExecutor, statepkg.LockerFactory) cliOpsHost {
		return &stubCLIOpsHost{planErr: errors.New("plan boom")}
	}
	err = runCLIModuleOp(context.Background(), &commandTestScope{ctx: context.Background()}, cliModuleOpOptions{
		Action:     ops.ActionUninstall,
		ModuleName: "core",
	})
	if err == nil || !strings.Contains(err.Error(), "plan boom") {
		t.Fatalf("plan err = %v", err)
	}

	newCLIOpsHost = func(scope.Scope, jsexecutor.JsExecutor, statepkg.LockerFactory) cliOpsHost {
		return &stubCLIOpsHost{plan: ops.PlanResult{Id: "op1"}, runErr: errors.New("run boom")}
	}
	err = runCLIModuleOp(context.Background(), &commandTestScope{ctx: context.Background()}, cliModuleOpOptions{
		Action:     ops.ActionUninstall,
		ModuleName: "core",
		WithDemo:   true,
	})
	if err == nil || !strings.Contains(err.Error(), "run boom") {
		t.Fatalf("run err = %v", err)
	}
}

func TestRunCLIModuleOpSyncAllowsEmptyModuleName(t *testing.T) {
	prevExec := newCLICompilerExecutor
	prevHost := newCLIOpsHost
	t.Cleanup(func() {
		newCLICompilerExecutor = prevExec
		newCLIOpsHost = prevHost
	})
	newCLICompilerExecutor = func(_ scope.Scope, _ ...jsexecutor.Option) (jsexecutor.JsExecutor, error) {
		return &stubCLIExecutor{}, nil
	}
	newCLIOpsHost = func(scope.Scope, jsexecutor.JsExecutor, statepkg.LockerFactory) cliOpsHost {
		return &stubCLIOpsHost{plan: ops.PlanResult{Id: "sync1"}}
	}
	if err := runCLIModuleOp(context.Background(), &commandTestScope{ctx: context.Background()}, cliModuleOpOptions{
		Action: ops.ActionSync,
	}); err != nil {
		t.Fatalf("sync: %v", err)
	}
}

type stubCLIExecutor struct {
	startErr error
	started  int
	stopped  int
}

func (e *stubCLIExecutor) Start() error {
	e.started++
	return e.startErr
}
func (e *stubCLIExecutor) Stop() error {
	e.stopped++
	return nil
}
func (e *stubCLIExecutor) Execute(context.Context, *jsengine.JsRequest) (*jsengine.JsResponse, error) {
	return &jsengine.JsResponse{}, nil
}
func (e *stubCLIExecutor) GetJsScripts() []*jsengine.JsScript { return nil }
func (e *stubCLIExecutor) SetJsScripts([]*jsengine.JsScript)  {}
func (e *stubCLIExecutor) Reload(...*jsengine.JsScript) error { return nil }
func (e *stubCLIExecutor) AppendJsScripts(...*jsengine.JsScript) {}

type stubCLIOpsHost struct {
	plan     ops.PlanResult
	planErr  error
	applyErr error
	runErr   error
	lastPlan ops.PlanRequest
}

func (h *stubCLIOpsHost) Plan(_ context.Context, req ops.PlanRequest) (ops.PlanResult, error) {
	h.lastPlan = req
	return h.plan, h.planErr
}
func (h *stubCLIOpsHost) Apply(context.Context, ops.ApplyRequest) error { return h.applyErr }
func (h *stubCLIOpsHost) Run(context.Context, ops.RunRequest) error     { return h.runErr }

type nilWithContextScope struct {
	commandTestScope
}

func (s *nilWithContextScope) WithContext(context.Context) scope.Scope { return nil }
