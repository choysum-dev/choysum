// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"context"
	"time"

	module "github.com/choysum-dev/choysum/internal/module/artifact/result"
	"github.com/choysum-dev/choysum/internal/module/evolution/hooks"
	"github.com/choysum-dev/choysum/internal/module/evolution/scripts"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsexecutor"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	xfmt "golang.org/x/exp/errors/fmt"
)

// migrationScriptRunner is the scripts.Runner surface used during upgrade prepare/finalize.
type migrationScriptRunner interface {
	Validate(ctx context.Context, fromVersion string, toVersion string, reuseExecutorScripts bool) error
	RunPhase(ctx context.Context, opts scripts.RunOptions) error
}

func reuseExecutorScriptsEnabled(moduleManager *ModuleManager) bool {
	return moduleManager != nil && moduleManager.jsExecutor != nil
}

// Overridable factories for unit tests.
var (
	newMigrationScriptRunner = func(
		runtimeScope scope.Scope,
		jsExecutor jsexecutor.ScriptExecutor,
		module *meta.Module,
		opts ...scripts.RunnerOption,
	) migrationScriptRunner {
		return scripts.NewRunner(runtimeScope, jsExecutor, module, opts...)
	}
	upgradeHooksNewRunner             = hooks.NewRunner
	upgradeHooksScriptFromBuildResult = hooks.ScriptFromBuildResult
	resolveUpgradeModuleFromOriginFn  = func(m *ModuleManager, ctx context.Context, name string) (*meta.Module, error) {
		return m.resolveUpgradeModuleFromOrigin(ctx, name)
	}
)

func moduleManagerJSExecutor(moduleManager *ModuleManager) jsexecutor.ScriptExecutor {
	if moduleManager == nil {
		return nil
	}
	return moduleManager.jsExecutor
}

// runUpgradePrepareMigrationScripts validates and runs PhasePre, or skips when versions match.
func (m *moduleUpgrader) runUpgradePrepareMigrationScripts(
	ctx context.Context,
	runner migrationScriptRunner,
	moduleName string,
	fromVersion string,
	toVersion string,
	reuseExec bool,
) error {
	if runner == nil {
		return nil
	}
	if shouldSkipSameVersionMigrationScripts(fromVersion, toVersion) {
		m.logUpgradeStep(moduleName, "scripts.validate", time.Now(), "from_version", fromVersion, "to_version", toVersion, "skipped", true, "reason", "same_version")
		m.logUpgradeStep(moduleName, moduleStepScripts(scripts.PhasePre), time.Now(), "from_version", fromVersion, "to_version", toVersion, "skipped", true, "reason", "same_version")
		return nil
	}
	validateStarted := time.Now()
	if err := runner.Validate(ctx, fromVersion, toVersion, reuseExec); err != nil {
		return xfmt.Errorf("error validating migrations for module %s: %w", moduleName, err)
	}
	m.logUpgradeStep(moduleName, "scripts.validate", validateStarted, "from_version", fromVersion, "to_version", toVersion)
	preStarted := time.Now()
	if err := runner.RunPhase(ctx, scripts.RunOptions{
		Phase:                scripts.PhasePre,
		FromVersion:          fromVersion,
		ToVersion:            toVersion,
		ReuseExecutorScripts: reuseExec,
	}); err != nil {
		return xfmt.Errorf("error running pre migrations for module %s: %w", moduleName, err)
	}
	m.logUpgradeStep(moduleName, moduleStepScripts(scripts.PhasePre), preStarted, "from_version", fromVersion, "to_version", toVersion)
	return nil
}

// runUpgradeFinalizeMigrationScripts runs PhasePost, or skips when versions match.
func (m *moduleUpgrader) runUpgradeFinalizeMigrationScripts(
	ctx context.Context,
	runner migrationScriptRunner,
	moduleName string,
	fromVersion string,
	toVersion string,
	reuseExec bool,
) error {
	if runner == nil {
		return nil
	}
	if shouldSkipSameVersionMigrationScripts(fromVersion, toVersion) {
		m.logUpgradeStep(moduleName, moduleStepScripts(scripts.PhasePost), time.Now(), "from_version", fromVersion, "to_version", toVersion, "skipped", true, "reason", "same_version")
		return nil
	}
	postStarted := time.Now()
	if err := runner.RunPhase(ctx, scripts.RunOptions{
		Phase:                scripts.PhasePost,
		FromVersion:          fromVersion,
		ToVersion:            toVersion,
		ReuseExecutorScripts: reuseExec,
	}); err != nil {
		return xfmt.Errorf("error running post migrations for module %s: %w", moduleName, err)
	}
	m.logUpgradeStep(moduleName, moduleStepScripts(scripts.PhasePost), postStarted, "from_version", fromVersion, "to_version", toVersion)
	return nil
}

func (m *moduleUpgrader) runUpgradeHookPhase(
	phase hooks.Phase,
	mod *meta.Module,
	fromVersion string,
	buildResult *module.BuildResult,
	reuseExec bool,
	phaseLabel string,
) error {
	if m == nil || mod == nil {
		return nil
	}
	hookRunner, err := upgradeHooksNewRunner(m.runtimeScope, moduleManagerJSExecutor(m.moduleManager), mod)
	if err != nil {
		return xfmt.Errorf("error preparing hooks for module %s: %w", mod.Name, err)
	}
	if hookRunner == nil {
		return nil
	}
	var hookScripts []*jsengine.JsScript
	if buildResult != nil {
		script, scriptErr := upgradeHooksScriptFromBuildResult(buildResult)
		if scriptErr != nil {
			return xfmt.Errorf("error preparing %s hook script: %w", phaseLabel, scriptErr)
		}
		if script != nil {
			hookScripts = append(hookScripts, script)
		}
	}
	hookStarted := time.Now()
	if err := hookRunner.RunPhase(m.runtimeScope.Context(), phase, hooks.RunOptions{
		FromVersion:          fromVersion,
		Scripts:              hookScripts,
		ReuseExecutorScripts: reuseExec,
	}); err != nil {
		return xfmt.Errorf("error running %s hook for module %s: %w", phaseLabel, mod.Name, err)
	}
	m.logUpgradeStep(mod.Name, moduleStepHook(phase), hookStarted, "from_version", fromVersion, "to_version", mod.Version)
	return nil
}

func (m *moduleUpgrader) runUpgradePrepareMigrations(ctx context.Context, target *meta.Module, fromVersion string, reuseExec bool) error {
	if m == nil || target == nil {
		return nil
	}
	runner := newMigrationScriptRunner(m.runtimeScope, moduleManagerJSExecutor(m.moduleManager), target, scripts.WithIntentBag(m.schemaIntents()))
	return m.runUpgradePrepareMigrationScripts(ctx, runner, target.Name, fromVersion, target.Version, reuseExec)
}

func (m *moduleUpgrader) runUpgradeFinalizeMigrations(ctx context.Context, target *meta.Module, fromVersion string, reuseExec bool) error {
	if m == nil || target == nil {
		return nil
	}
	runner := newMigrationScriptRunner(m.runtimeScope, moduleManagerJSExecutor(m.moduleManager), target, scripts.WithIntentBag(m.schemaIntents()))
	return m.runUpgradeFinalizeMigrationScripts(ctx, runner, target.Name, fromVersion, target.Version, reuseExec)
}
