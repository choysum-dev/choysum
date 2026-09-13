// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	module "github.com/choysum-dev/choysum/internal/module/artifact/result"
	"github.com/choysum-dev/choysum/internal/module/evolution/hooks"
	"github.com/choysum-dev/choysum/internal/module/evolution/scripts"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsexecutor"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
)

type stubMigrationRunner struct {
	validateErr error
	preErr      error
	postErr     error
	validateN   int
	preN        int
	postN       int
}

func (s *stubMigrationRunner) Validate(ctx context.Context, fromVersion string, toVersion string, reuseExecutorScripts bool) error {
	s.validateN++
	return s.validateErr
}

func (s *stubMigrationRunner) RunPhase(ctx context.Context, opts scripts.RunOptions) error {
	switch opts.Phase {
	case scripts.PhasePre:
		s.preN++
		return s.preErr
	case scripts.PhasePost:
		s.postN++
		return s.postErr
	default:
		return nil
	}
}

func TestReuseExecutorScriptsEnabled(t *testing.T) {
	if reuseExecutorScriptsEnabled(nil) {
		t.Fatal("nil manager")
	}
	if reuseExecutorScriptsEnabled(&ModuleManager{}) {
		t.Fatal("nil executor")
	}
	exec := struct{ jsexecutor.ScriptExecutor }{}
	// concrete nil interface vs typed nil — use a non-nil fake via existing test helpers if needed.
	_ = exec
	m := &ModuleManager{jsExecutor: &nopScriptExecutor{}}
	if !reuseExecutorScriptsEnabled(m) {
		t.Fatal("expected true with executor")
	}
}

type nopScriptExecutor struct{}

func (n *nopScriptExecutor) Execute(ctx context.Context, request *jsengine.JsRequest) (*jsengine.JsResponse, error) {
	return &jsengine.JsResponse{Id: request.Id}, nil
}
func (n *nopScriptExecutor) GetJsScripts() []*jsengine.JsScript { return nil }
func (n *nopScriptExecutor) SetJsScripts(scripts []*jsengine.JsScript) {
}
func (n *nopScriptExecutor) Reload(scripts ...*jsengine.JsScript) error { return nil }

func TestRunUpgradePrepareMigrationScripts(t *testing.T) {
	var logBuf bytes.Buffer
	upgrader := &moduleUpgrader{
		runtimeScope: &testLogScope{
			ctx:    context.Background(),
			logger: slog.New(slog.NewJSONHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelDebug})),
		},
	}

	t.Run("nil runner", func(t *testing.T) {
		if err := upgrader.runUpgradePrepareMigrationScripts(context.Background(), nil, "partner", "1.0.0", "1.0.0", true); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("same version skips", func(t *testing.T) {
		logBuf.Reset()
		stub := &stubMigrationRunner{}
		if err := upgrader.runUpgradePrepareMigrationScripts(context.Background(), stub, "partner", "1.0.0", "v1.0.0", true); err != nil {
			t.Fatal(err)
		}
		if stub.validateN != 0 || stub.preN != 0 {
			t.Fatalf("expected skip, got validate=%d pre=%d", stub.validateN, stub.preN)
		}
		logs := logBuf.String()
		if !strings.Contains(logs, `"skipped":true`) || !strings.Contains(logs, "same_version") {
			t.Fatalf("expected skip logs, got %s", logs)
		}
	})

	t.Run("version bump runs validate and pre", func(t *testing.T) {
		stub := &stubMigrationRunner{}
		if err := upgrader.runUpgradePrepareMigrationScripts(context.Background(), stub, "partner", "1.0.0", "1.1.0", true); err != nil {
			t.Fatal(err)
		}
		if stub.validateN != 1 || stub.preN != 1 {
			t.Fatalf("validate=%d pre=%d", stub.validateN, stub.preN)
		}
	})

	t.Run("validate error", func(t *testing.T) {
		stub := &stubMigrationRunner{validateErr: errors.New("validate boom")}
		err := upgrader.runUpgradePrepareMigrationScripts(context.Background(), stub, "partner", "1.0.0", "1.1.0", false)
		if err == nil || !strings.Contains(err.Error(), "validating migrations") {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("pre error", func(t *testing.T) {
		stub := &stubMigrationRunner{preErr: errors.New("pre boom")}
		err := upgrader.runUpgradePrepareMigrationScripts(context.Background(), stub, "partner", "1.0.0", "1.1.0", false)
		if err == nil || !strings.Contains(err.Error(), "pre migrations") {
			t.Fatalf("got %v", err)
		}
	})
}

func TestRunUpgradeFinalizeMigrationScripts(t *testing.T) {
	var logBuf bytes.Buffer
	upgrader := &moduleUpgrader{
		runtimeScope: &testLogScope{
			ctx:    context.Background(),
			logger: slog.New(slog.NewJSONHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelDebug})),
		},
	}

	t.Run("nil runner", func(t *testing.T) {
		if err := upgrader.runUpgradeFinalizeMigrationScripts(context.Background(), nil, "partner", "1.0.0", "1.0.0", true); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("same version skips post", func(t *testing.T) {
		logBuf.Reset()
		stub := &stubMigrationRunner{}
		if err := upgrader.runUpgradeFinalizeMigrationScripts(context.Background(), stub, "partner", "1.0.0", "1.0.0", true); err != nil {
			t.Fatal(err)
		}
		if stub.postN != 0 {
			t.Fatalf("post=%d", stub.postN)
		}
		if !strings.Contains(logBuf.String(), `"skipped":true`) {
			t.Fatalf("logs=%s", logBuf.String())
		}
	})

	t.Run("version bump runs post", func(t *testing.T) {
		stub := &stubMigrationRunner{}
		if err := upgrader.runUpgradeFinalizeMigrationScripts(context.Background(), stub, "partner", "1.0.0", "1.1.0", true); err != nil {
			t.Fatal(err)
		}
		if stub.postN != 1 {
			t.Fatalf("post=%d", stub.postN)
		}
	})

	t.Run("post error", func(t *testing.T) {
		stub := &stubMigrationRunner{postErr: errors.New("post boom")}
		err := upgrader.runUpgradeFinalizeMigrationScripts(context.Background(), stub, "partner", "1.0.0", "1.1.0", false)
		if err == nil || !strings.Contains(err.Error(), "post migrations") {
			t.Fatalf("got %v", err)
		}
	})
}

func TestRunUpgradeHookAndMigrationWrappers(t *testing.T) {
	var logBuf bytes.Buffer
	upgrader := &moduleUpgrader{
		runtimeScope: &testLogScope{
			ctx:    context.Background(),
			logger: slog.New(slog.NewJSONHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelDebug})),
		},
		module:        &meta.Module{Name: "partner", Version: "1.0.0"},
		moduleManager: &ModuleManager{jsExecutor: &nopScriptExecutor{}},
	}

	t.Run("nil receivers", func(t *testing.T) {
		if err := (*moduleUpgrader)(nil).runUpgradeHookPhase(hooks.PhasePreUpgrade, nil, "1.0.0", nil, true, "pre_upgrade"); err != nil {
			t.Fatal(err)
		}
		if err := (*moduleUpgrader)(nil).runUpgradePrepareMigrations(context.Background(), nil, "1.0.0", true); err != nil {
			t.Fatal(err)
		}
		if err := (*moduleUpgrader)(nil).runUpgradeFinalizeMigrations(context.Background(), nil, "1.0.0", true); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("hook runner error", func(t *testing.T) {
		prev := upgradeHooksNewRunner
		t.Cleanup(func() { upgradeHooksNewRunner = prev })
		upgradeHooksNewRunner = func(scope.Scope, jsexecutor.ScriptExecutor, *meta.Module) (*hooks.Runner, error) {
			return nil, errors.New("runner boom")
		}
		err := upgrader.runUpgradeHookPhase(hooks.PhasePreUpgrade, upgrader.module, "1.0.0", nil, true, "pre_upgrade")
		if err == nil || !strings.Contains(err.Error(), "preparing hooks") {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("hook runphase error", func(t *testing.T) {
		prev := upgradeHooksNewRunner
		t.Cleanup(func() { upgradeHooksNewRunner = prev })
		upgradeHooksNewRunner = func(scope.Scope, jsexecutor.ScriptExecutor, *meta.Module) (*hooks.Runner, error) {
			return &hooks.Runner{}, nil // nil jsExecutor → RunPhase error
		}
		err := upgrader.runUpgradeHookPhase(hooks.PhasePreUpgrade, upgrader.module, "1.0.0", nil, true, "pre_upgrade")
		if err == nil || !strings.Contains(err.Error(), "pre_upgrade") {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("hook script error", func(t *testing.T) {
		prevRunner, prevScript := upgradeHooksNewRunner, upgradeHooksScriptFromBuildResult
		t.Cleanup(func() {
			upgradeHooksNewRunner = prevRunner
			upgradeHooksScriptFromBuildResult = prevScript
		})
		upgradeHooksNewRunner = func(scope.Scope, jsexecutor.ScriptExecutor, *meta.Module) (*hooks.Runner, error) {
			return &hooks.Runner{}, nil
		}
		upgradeHooksScriptFromBuildResult = func(*module.BuildResult) (*jsengine.JsScript, error) {
			return nil, errors.New("script boom")
		}
		err := upgrader.runUpgradeHookPhase(hooks.PhasePostUpgrade, upgrader.module, "1.0.0", &module.BuildResult{}, true, "post_upgrade")
		if err == nil || !strings.Contains(err.Error(), "hook script") {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("prepare migration error via factory", func(t *testing.T) {
		prev := newMigrationScriptRunner
		t.Cleanup(func() { newMigrationScriptRunner = prev })
		newMigrationScriptRunner = func(scope.Scope, jsexecutor.ScriptExecutor, *meta.Module, ...scripts.RunnerOption) migrationScriptRunner {
			return &stubMigrationRunner{validateErr: errors.New("validate boom")}
		}
		err := upgrader.runUpgradePrepareMigrations(context.Background(), &meta.Module{Name: "partner", Version: "1.1.0"}, "1.0.0", true)
		if err == nil || !strings.Contains(err.Error(), "validating migrations") {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("finalize migration error via factory", func(t *testing.T) {
		prev := newMigrationScriptRunner
		t.Cleanup(func() { newMigrationScriptRunner = prev })
		newMigrationScriptRunner = func(scope.Scope, jsexecutor.ScriptExecutor, *meta.Module, ...scripts.RunnerOption) migrationScriptRunner {
			return &stubMigrationRunner{postErr: errors.New("post boom")}
		}
		err := upgrader.runUpgradeFinalizeMigrations(context.Background(), &meta.Module{Name: "partner", Version: "1.1.0"}, "1.0.0", true)
		if err == nil || !strings.Contains(err.Error(), "post migrations") {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("finalizeUpgrade propagates migration and hook errors", func(t *testing.T) {
		prevMig := newMigrationScriptRunner
		prevHook := upgradeHooksNewRunner
		t.Cleanup(func() {
			newMigrationScriptRunner = prevMig
			upgradeHooksNewRunner = prevHook
		})
		newMigrationScriptRunner = func(scope.Scope, jsexecutor.ScriptExecutor, *meta.Module, ...scripts.RunnerOption) migrationScriptRunner {
			return &stubMigrationRunner{postErr: errors.New("post boom")}
		}
		err := upgrader.finalizeUpgrade(&meta.Module{Name: "partner", Version: "1.1.0"}, "1.0.0", nil)
		if err == nil || !strings.Contains(err.Error(), "post migrations") {
			t.Fatalf("got %v", err)
		}

		newMigrationScriptRunner = func(scope.Scope, jsexecutor.ScriptExecutor, *meta.Module, ...scripts.RunnerOption) migrationScriptRunner {
			return &stubMigrationRunner{}
		}
		upgradeHooksNewRunner = func(scope.Scope, jsexecutor.ScriptExecutor, *meta.Module) (*hooks.Runner, error) {
			return &hooks.Runner{}, nil
		}
		err = upgrader.finalizeUpgrade(&meta.Module{Name: "partner", Version: "1.1.0"}, "1.0.0", nil)
		if err == nil || !strings.Contains(err.Error(), "post_upgrade") {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("hook nil runner ok", func(t *testing.T) {
		prev := upgradeHooksNewRunner
		t.Cleanup(func() { upgradeHooksNewRunner = prev })
		upgradeHooksNewRunner = func(scope.Scope, jsexecutor.ScriptExecutor, *meta.Module) (*hooks.Runner, error) {
			return nil, nil
		}
		if err := upgrader.runUpgradeHookPhase(hooks.PhasePreUpgrade, upgrader.module, "1.0.0", nil, true, "pre_upgrade"); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("hook script append then runphase error", func(t *testing.T) {
		prevRunner, prevScript := upgradeHooksNewRunner, upgradeHooksScriptFromBuildResult
		t.Cleanup(func() {
			upgradeHooksNewRunner = prevRunner
			upgradeHooksScriptFromBuildResult = prevScript
		})
		upgradeHooksNewRunner = func(scope.Scope, jsexecutor.ScriptExecutor, *meta.Module) (*hooks.Runner, error) {
			return &hooks.Runner{}, nil
		}
		upgradeHooksScriptFromBuildResult = func(*module.BuildResult) (*jsengine.JsScript, error) {
			return &jsengine.JsScript{FileName: "index.js", Content: "export {}"}, nil
		}
		err := upgrader.runUpgradeHookPhase(hooks.PhasePostUpgrade, upgrader.module, "1.0.0", &module.BuildResult{}, true, "post_upgrade")
		if err == nil || !strings.Contains(err.Error(), "post_upgrade") {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("upgrade propagates prepare hook and migration errors", func(t *testing.T) {
		prevHook := upgradeHooksNewRunner
		prevResolve := resolveUpgradeModuleFromOriginFn
		prevMig := newMigrationScriptRunner
		t.Cleanup(func() {
			upgradeHooksNewRunner = prevHook
			resolveUpgradeModuleFromOriginFn = prevResolve
			newMigrationScriptRunner = prevMig
		})

		u := &moduleUpgrader{
			runtimeScope:  &testLogScope{ctx: context.Background(), logger: slog.New(slog.NewJSONHandler(&logBuf, nil))},
			module:        &meta.Module{Name: "partner", Version: "1.0.0", Status: meta.Installed},
			moduleManager: &ModuleManager{jsExecutor: &nopScriptExecutor{}},
			ctx:           newOpContext(),
		}
		upgradeHooksNewRunner = func(scope.Scope, jsexecutor.ScriptExecutor, *meta.Module) (*hooks.Runner, error) {
			return &hooks.Runner{}, nil
		}
		if err := u.upgrade(); err == nil || !strings.Contains(err.Error(), "pre_upgrade") {
			t.Fatalf("hook err got %v", err)
		}

		upgradeHooksNewRunner = func(scope.Scope, jsexecutor.ScriptExecutor, *meta.Module) (*hooks.Runner, error) {
			return nil, nil
		}
		resolveUpgradeModuleFromOriginFn = func(*ModuleManager, context.Context, string) (*meta.Module, error) {
			return &meta.Module{Name: "partner", Version: "1.1.0", Status: meta.ToInstall}, nil
		}
		newMigrationScriptRunner = func(scope.Scope, jsexecutor.ScriptExecutor, *meta.Module, ...scripts.RunnerOption) migrationScriptRunner {
			return &stubMigrationRunner{validateErr: errors.New("validate boom")}
		}
		if err := u.upgrade(); err == nil || !strings.Contains(err.Error(), "validating migrations") {
			t.Fatalf("migration err got %v", err)
		}
	})
}

func TestRunUninstallHookPhaseErrors(t *testing.T) {
	m := &moduleUninstaller{
		runtimeScope:  &testLogScope{ctx: context.Background()},
		module:        &meta.Module{Name: "partner"},
		moduleManager: &ModuleManager{jsExecutor: &nopScriptExecutor{}},
	}
	if err := (*moduleUninstaller)(nil).runUninstallHookPhase(nil, hooks.PhasePreUninstall, "pre_uninstall"); err != nil {
		t.Fatal(err)
	}
	if err := m.runUninstallHookPhase(nil, hooks.PhasePreUninstall, "pre_uninstall"); err != nil {
		t.Fatal(err)
	}
	err := m.runUninstallHookPhase(&hooks.Runner{}, hooks.PhasePreUninstall, "pre_uninstall")
	if err == nil || !strings.Contains(err.Error(), "pre_uninstall") {
		t.Fatalf("got %v", err)
	}

	prev := uninstallHooksNewRunner
	t.Cleanup(func() { uninstallHooksNewRunner = prev })
	uninstallHooksNewRunner = func(scope.Scope, jsexecutor.ScriptExecutor, *meta.Module) (*hooks.Runner, error) {
		return nil, errors.New("runner boom")
	}
	m.hookRunner = nil
	if err := m.finalizeUninstall(); err == nil || !strings.Contains(err.Error(), "preparing hooks") {
		t.Fatalf("got %v", err)
	}

	uninstallHooksNewRunner = hooks.NewRunner
	m.hookRunner = &hooks.Runner{} // nil executor
	if err := m.finalizeUninstall(); err == nil || !strings.Contains(err.Error(), "post_uninstall") {
		t.Fatalf("got %v", err)
	}
}

func TestModuleManagerJSExecutorAndNameOrEmpty(t *testing.T) {
	if moduleManagerJSExecutor(nil) != nil {
		t.Fatal("nil manager")
	}
	if nameOrEmpty(nil) != "" {
		t.Fatal("nil module name")
	}
	if nameOrEmpty(&meta.Module{Name: "x"}) != "x" {
		t.Fatal("name")
	}
}

func TestNewMigrationScriptRunner_ReturnsUntypedNil(t *testing.T) {
	got := newMigrationScriptRunner(nil, nil, nil)
	if got != nil {
		t.Fatalf("expected untyped nil interface, got %#v", got)
	}
}

func TestRunUpgradeHookPhase_PreOmitsToVersion(t *testing.T) {
	var logBuf bytes.Buffer
	upgrader := &moduleUpgrader{
		runtimeScope: &testLogScope{
			ctx:    context.Background(),
			logger: slog.New(slog.NewJSONHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelDebug})),
		},
		module:        &meta.Module{Name: "partner", Version: "1.0.0"},
		moduleManager: &ModuleManager{jsExecutor: &nopScriptExecutor{}},
	}
	prevRunner, prevScript := upgradeHooksNewRunner, upgradeHooksScriptFromBuildResult
	t.Cleanup(func() {
		upgradeHooksNewRunner = prevRunner
		upgradeHooksScriptFromBuildResult = prevScript
	})
	upgradeHooksNewRunner = hooks.NewRunner
	upgradeHooksScriptFromBuildResult = func(*module.BuildResult) (*jsengine.JsScript, error) {
		return &jsengine.JsScript{FileName: "index.js", Content: "export {}"}, nil
	}
	build := &module.BuildResult{}

	logBuf.Reset()
	if err := upgrader.runUpgradeHookPhase(hooks.PhasePreUpgrade, upgrader.module, "1.0.0", build, true, "pre_upgrade"); err != nil {
		t.Fatalf("pre: %v", err)
	}
	preLogs := logBuf.String()
	if !strings.Contains(preLogs, `"step":"hook.pre_upgrade"`) {
		t.Fatalf("missing pre log: %s", preLogs)
	}
	if strings.Contains(preLogs, `"to_version"`) {
		t.Fatalf("pre_upgrade must omit to_version, got %s", preLogs)
	}

	logBuf.Reset()
	target := &meta.Module{Name: "partner", Version: "1.1.0"}
	if err := upgrader.runUpgradeHookPhase(hooks.PhasePostUpgrade, target, "1.0.0", build, true, "post_upgrade"); err != nil {
		t.Fatalf("post: %v", err)
	}
	postLogs := logBuf.String()
	if !strings.Contains(postLogs, `"to_version":"1.1.0"`) {
		t.Fatalf("post_upgrade should log target version, got %s", postLogs)
	}
}

func TestUninstallPrepareHookErrorPropagation(t *testing.T) {
	prev := uninstallHooksNewRunner
	t.Cleanup(func() { uninstallHooksNewRunner = prev })

	m := &moduleUninstaller{
		runtimeScope:  &testLogScope{ctx: context.Background()},
		module:        &meta.Module{Name: "partner", Status: meta.Installed},
		moduleManager: &ModuleManager{jsExecutor: &nopScriptExecutor{}},
	}
	uninstallHooksNewRunner = func(scope.Scope, jsexecutor.ScriptExecutor, *meta.Module) (*hooks.Runner, error) {
		return nil, errors.New("runner boom")
	}
	if err := m.uninstall(); err == nil || !strings.Contains(err.Error(), "preparing hooks") {
		t.Fatalf("got %v", err)
	}

	uninstallHooksNewRunner = func(scope.Scope, jsexecutor.ScriptExecutor, *meta.Module) (*hooks.Runner, error) {
		return &hooks.Runner{}, nil
	}
	if err := m.uninstall(); err == nil || !strings.Contains(err.Error(), "pre_uninstall") {
		t.Fatalf("got %v", err)
	}
}

func TestResolveUninstallHookRunner(t *testing.T) {
	existing := &hooks.Runner{}
	m := &moduleUninstaller{
		hookRunner:    existing,
		runtimeScope:  &testLogScope{ctx: context.Background()},
		module:        &meta.Module{Name: "partner"},
		moduleManager: &ModuleManager{},
	}
	got, err := m.resolveUninstallHookRunner()
	if err != nil || got != existing {
		t.Fatalf("got %#v err=%v", got, err)
	}

	m.hookRunner = nil
	got, err = m.resolveUninstallHookRunner()
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("expected new runner")
	}

	if _, err := (*moduleUninstaller)(nil).resolveUninstallHookRunner(); err != nil {
		t.Fatal(err)
	}

	prev := uninstallHooksNewRunner
	t.Cleanup(func() { uninstallHooksNewRunner = prev })
	uninstallHooksNewRunner = func(scope.Scope, jsexecutor.ScriptExecutor, *meta.Module) (*hooks.Runner, error) {
		return nil, errors.New("runner boom")
	}
	m.hookRunner = nil
	if _, err := m.resolveUninstallHookRunner(); err == nil || !strings.Contains(err.Error(), "preparing hooks") {
		t.Fatalf("got %v", err)
	}
}
