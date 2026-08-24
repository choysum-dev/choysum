// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"context"
	"log/slog"
	"strings"
	"time"

	module "github.com/choysum-dev/choysum/internal/module/artifact/result"
	"github.com/choysum-dev/choysum/internal/module/artifact/staging"
	"github.com/choysum-dev/choysum/internal/module/evolution/hooks"
	"github.com/choysum-dev/choysum/internal/module/evolution/schema"
	"github.com/choysum-dev/choysum/internal/module/evolution/scripts"
	"github.com/choysum-dev/choysum/internal/module/plan"
	"github.com/choysum-dev/choysum/internal/module/policy"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	xfmt "golang.org/x/exp/errors/fmt"
)

type moduleUpgrader struct {
	runtimeScope  scope.Scope
	module        *meta.Module
	moduleManager *ModuleManager
	ctx           *opContext
}

const (
	moduleStepPrepare    = "prepare"
	moduleStepBuild      = "build"
	moduleStepInitialize = "initialize"
	moduleStepSchema     = "schema"
	moduleStepData       = "data"
	moduleStepSave       = "save"
	moduleStepCleanup    = "cleanup"
	moduleStepFinalize   = "finalize"
)

func (m *moduleUpgrader) validate() error {
	return policy.RequireInstalledForUpgrade(m.module)
}

func moduleOperationStepInfoAttrs(step string, duration time.Duration, extra ...any) []any {
	attrs := []any{"duration_ms", duration.Milliseconds()}
	step = strings.TrimSpace(step)
	if step != "" {
		attrs = append(attrs, "step", step)
	}
	if len(extra) > 0 {
		attrs = append(attrs, extra...)
	}
	return attrs
}

func moduleOperationStepMessage(op plan.OpType) string {
	switch op {
	case plan.OpInstall:
		return "module install step completed"
	case plan.OpUninstall:
		return "module uninstall step completed"
	default:
		return "module upgrade step completed"
	}
}

func logModuleOperationStep(runtimeScope scope.Scope, opCtx *opContext, op plan.OpType, moduleName string, step string, started time.Time, extra ...any) {
	if runtimeScope == nil || runtimeScope.Logger() == nil || started.IsZero() {
		return
	}
	ctx := runtimeScope.Context()
	opid := ""
	if opCtx != nil {
		opid = strings.TrimSpace(opCtx.opid)
	}
	if opid != "" {
		ctx = staging.WithOpID(ctx, opid)
	} else {
		ctx, opid = ensureOpIDInContext(ctx)
	}
	logger := moduleOpLogger(runtimeScope.Logger(), opid, op, strings.TrimSpace(moduleName))
	// Build/schema are the long segments; emit at Info for TX-boundary timing.
	level := slog.LevelDebug
	switch strings.TrimSpace(step) {
	case moduleStepBuild, moduleStepSchema:
		level = slog.LevelInfo
	}
	logger.Log(ctx, level, moduleOperationStepMessage(op), moduleOperationStepInfoAttrs(step, time.Since(started), extra...)...)
}

func (m *moduleUpgrader) logUpgradeStep(moduleName string, step string, started time.Time, extra ...any) {
	if m == nil {
		return
	}
	logModuleOperationStep(m.runtimeScope, m.ctx, plan.OpUpgrade, moduleName, step, started, extra...)
}

func (m *moduleUpgrader) upgrade() error {
	if err := m.validate(); err != nil {
		return xfmt.Errorf("error validating module %s: %w", m.module.Name, err)
	}

	fromVersion := m.module.Version
	if m.ctx != nil {
		m.ctx.setFromVersion(m.module.Name, fromVersion)
	}
	prepareStarted := time.Now()
	if hookRunner, err := hooks.NewRunner(m.runtimeScope, m.moduleManager.jsExecutor, m.module); err != nil {
		return xfmt.Errorf("error preparing hooks for module %s: %w", m.module.Name, err)
	} else if hookRunner != nil {
		if err := hookRunner.RunPhase(m.runtimeScope.Context(), hooks.PhasePreUpgrade, hooks.RunOptions{FromVersion: fromVersion}); err != nil {
			return xfmt.Errorf("error running pre_upgrade hook for module %s: %w", m.module.Name, err)
		}
	}

	// Resolve target module version from origin without mutating origin binding during upgrade.
	target, err := m.moduleManager.resolveUpgradeModuleFromOrigin(m.runtimeScope.Context(), m.module.Name)
	if err != nil {
		return xfmt.Errorf("error resolving module %s from origin: %w", m.module.Name, err)
	}
	if m.module.Id.Valid {
		target.Id = m.module.Id
	}

	installer := newModuleInstaller(m.runtimeScope, m.moduleManager.jsExecutor, target, m.moduleManager, m.ctx)
	if err := installer.validate(); err != nil {
		return xfmt.Errorf("error validating module %s: %w", target.Name, err)
	}

	if runner := scripts.NewRunner(m.runtimeScope, m.moduleManager.jsExecutor, target); runner != nil {
		if err := runner.Validate(m.runtimeScope.Context(), fromVersion, target.Version); err != nil {
			return xfmt.Errorf("error validating migrations for module %s: %w", target.Name, err)
		}
		if err := runner.RunPhase(m.runtimeScope.Context(), scripts.RunOptions{Phase: scripts.PhasePre, FromVersion: fromVersion, ToVersion: target.Version}); err != nil {
			return xfmt.Errorf("error running pre migrations for module %s: %w", target.Name, err)
		}
	}
	m.logUpgradeStep(target.Name, moduleStepPrepare, prepareStarted, "from_version", fromVersion, "to_version", target.Version)

	var buildResult *module.BuildResult
	persistLater := false
	if installer.builder != nil {
		if split, ok := installer.builder.(module.SplitBuilder); ok {
			buildStarted := time.Now()
			result, err := split.BuildWithoutPersist()
			if err != nil {
				return xfmt.Errorf("error building module %s: %w", target.Name, err)
			}
			buildResult = result
			persistLater = true
			m.logUpgradeStep(target.Name, moduleStepBuild, buildStarted, "from_version", fromVersion, "to_version", target.Version)
		}
	}

	txRoot := m.runtimeScope
	if txRoot == nil {
		return xfmt.Errorf("scope is nil")
	}
	ctx := txRoot.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	txHoldStarted := time.Now()
	err = txRoot.Transactor().Required(ctx, func(txScope scope.Scope, tx scope.Transaction) error {
		committed := installer.forCommitScope(txScope)
		upgrader := *m
		upgrader.runtimeScope = txScope
		result, commitErr := upgrader.commitUpgrade(committed, fromVersion, buildResult, persistLater)
		if commitErr != nil {
			return commitErr
		}
		buildResult = result
		return nil
	})
	LogModuleCommitTxHold(m.runtimeScope.Logger(), "upgrade", "module_commit", txHoldStarted, err)
	if err != nil {
		return err
	}

	return m.finalizeUpgrade(installer.module, fromVersion, buildResult)
}

func (m *moduleUpgrader) commitUpgrade(installer *moduleInstaller, fromVersion string, buildResult *module.BuildResult, persistLater bool) (*module.BuildResult, error) {
	if installer == nil || installer.module == nil {
		return nil, xfmt.Errorf("upgrade commit installer is nil")
	}
	target := installer.module

	if installer.builder != nil {
		if persistLater {
			if split, ok := installer.builder.(module.SplitBuilder); ok {
				if err := split.Persist(buildResult); err != nil {
					return nil, xfmt.Errorf("error persisting module %s: %w", target.Name, err)
				}
			} else {
				return nil, xfmt.Errorf("builder does not support Persist for module %s", target.Name)
			}
		} else {
			buildStarted := time.Now()
			result, err := installer.builder.Build()
			if err != nil {
				return nil, xfmt.Errorf("error building module %s: %w", target.Name, err)
			}
			buildResult = result
			m.logUpgradeStep(target.Name, moduleStepBuild, buildStarted, "from_version", fromVersion, "to_version", target.Version)
		}
	}

	migrator, err := schema.NewMigrator(m.runtimeScope, target)
	if err != nil {
		return nil, xfmt.Errorf("error preparing schema migrator for module %s: %w", target.Name, err)
	}
	schemaMigrationStarted := time.Now()
	if err := migrator.Migrate(); err != nil {
		return nil, xfmt.Errorf("error migrating module %s: %w", target.Name, err)
	}
	m.logUpgradeStep(target.Name, moduleStepSchema, schemaMigrationStarted, "from_version", fromVersion, "to_version", target.Version)

	applyCtx := m.runtimeScope.Context()
	if applyCtx == nil {
		applyCtx = context.Background()
	}
	dataApplyStarted := time.Now()
	if err := applyInitdata(applyCtx, m.runtimeScope, target, importpkg.CallerLifecycle, m.ctx != nil && m.ctx.withDemo); err != nil {
		return nil, xfmt.Errorf("error applying data for module %s: %w", target.Name, err)
	}
	m.logUpgradeStep(target.Name, moduleStepData, dataApplyStarted, "from_version", fromVersion, "to_version", target.Version)

	persistModuleStarted := time.Now()
	target.Status = meta.Installed
	if len(target.Dependencies) > 0 {
		if err := m.runtimeScope.Session().Model(target).Association("Dependencies").Replace(target.Dependencies); err != nil {
			return nil, xfmt.Errorf("error saving module dependencies: %w", err)
		}
	}
	// Omit association trees: Persist already wrote raw + effective catalogs. Cascading
	// Models here would duplicate effective rows with module_id (see install commitSave).
	if err := m.runtimeScope.Session().
		Omit("Dependencies", "Dependents", "Models", "Components", "UiResources").
		Save(target).Error; err != nil {
		return nil, xfmt.Errorf("error saving module: %w", err)
	}
	m.logUpgradeStep(target.Name, moduleStepSave, persistModuleStarted, "from_version", fromVersion, "to_version", target.Version)

	if err := importModuleTerminology(m.runtimeScope, target, runtimeOptionsFromScope(m.runtimeScope).modulesPath); err != nil {
		return nil, err
	}
	return buildResult, nil
}

func (m *moduleUpgrader) finalizeUpgrade(target *meta.Module, fromVersion string, buildResult *module.BuildResult) error {
	if target == nil {
		return xfmt.Errorf("upgrade finalize target is nil")
	}
	finalizeStarted := time.Now()
	if runner := scripts.NewRunner(m.runtimeScope, m.moduleManager.jsExecutor, target); runner != nil {
		if err := runner.RunPhase(m.runtimeScope.Context(), scripts.RunOptions{Phase: scripts.PhasePost, FromVersion: fromVersion, ToVersion: target.Version}); err != nil {
			return xfmt.Errorf("error running post migrations for module %s: %w", target.Name, err)
		}
	}

	if hookRunner, err := hooks.NewRunner(m.runtimeScope, m.moduleManager.jsExecutor, target); err != nil {
		return xfmt.Errorf("error preparing hooks for module %s: %w", target.Name, err)
	} else if hookRunner != nil {
		var hookScripts []*jsengine.JsScript
		if buildResult != nil {
			if script, err := hooks.ScriptFromBuildResult(buildResult); err != nil {
				return xfmt.Errorf("error preparing post_upgrade hook script: %w", err)
			} else if script != nil {
				hookScripts = append(hookScripts, script)
			}
		}
		if err := hookRunner.RunPhase(m.runtimeScope.Context(), hooks.PhasePostUpgrade, hooks.RunOptions{FromVersion: fromVersion, Scripts: hookScripts}); err != nil {
			return xfmt.Errorf("error running post_upgrade hook for module %s: %w", target.Name, err)
		}
	}
	m.logUpgradeStep(target.Name, moduleStepFinalize, finalizeStarted, "from_version", fromVersion, "to_version", target.Version)
	return nil
}

func newModuleUpgrader(runtimeScope scope.Scope, module *meta.Module, moduleManager *ModuleManager, ctx *opContext) *moduleUpgrader {
	return &moduleUpgrader{
		runtimeScope:  runtimeScope,
		module:        module,
		moduleManager: moduleManager,
		ctx:           ctx,
	}
}
