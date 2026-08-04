// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"time"

	internalbackendbuilder "github.com/choysum-dev/choysum/internal/module/artifact/build/backend"
	module "github.com/choysum-dev/choysum/internal/module/artifact/result"
	dataloader "github.com/choysum-dev/choysum/internal/module/evolution/data"
	"github.com/choysum-dev/choysum/internal/module/evolution/hooks"
	"github.com/choysum-dev/choysum/internal/module/evolution/schema"
	"github.com/choysum-dev/choysum/internal/module/plan"
	"github.com/choysum-dev/choysum/internal/module/policy"
	"github.com/choysum-dev/choysum/internal/task"

	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsexecutor"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/rs/xid"

	xfmt "golang.org/x/exp/errors/fmt"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type moduleInstaller struct {
	module        *meta.Module
	runtimeScope  scope.Scope
	moduleManager *ModuleManager
	ctx           *opContext

	builder module.Builder
}

func (m *moduleInstaller) restoreModuleIfSoftDeleted() error {
	var existing meta.Module
	err := m.runtimeScope.Session().Unscoped().Where("name = ?", m.module.Name).Take(&existing).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return xfmt.Errorf("error checking existing module %s: %w", m.module.Name, err)
	}

	// Ensure we update the existing row instead of inserting a new one.
	m.module.Id = existing.Id

	if existing.DeletedAt.Valid {
		if err := m.runtimeScope.Session().Unscoped().Model(&meta.Module{}).
			Where("id = ?", existing.Id.String).
			Update("deleted_at", nil).Error; err != nil {
			return xfmt.Errorf("error restoring soft-deleted module %s: %w", m.module.Name, err)
		}
	}
	return nil
}

// check external dependency is installed
func (m *moduleInstaller) checkExtDepInstalled() error {
	return policy.CheckExternalDependencies(m.module)
}

func (m *moduleInstaller) checkModuleInstalled() bool {
	if m.runtimeScope.Session().Migrator().HasTable(&meta.Module{}) {
		if result := m.runtimeScope.Session().
			Preload("Dependencies", func(db *gorm.DB) *gorm.DB { return db.Where("status = ?", meta.Installed).Order("id ASC") }).
			Preload("Dependents", func(db *gorm.DB) *gorm.DB { return db.Where("status = ?", meta.Installed).Order("id ASC") }).
			Where("name = ?", m.module.Name).Take(m.module); result.Error != nil {
			return false
		}
		if m.module.Status == meta.Installed {
			return true
		}
	} else {
		return false
	}
	return false
}

func (m *moduleInstaller) validate() error {
	if err := m.checkExtDepInstalled(); err != nil {
		return xfmt.Errorf("error checking external dependencies: %w", err)
	}

	if err := m.assertDependenciesInstalled(); err != nil {
		return xfmt.Errorf("error asserting dependencies installed: %w", err)
	}
	return nil
}

func (m *moduleInstaller) assertDependenciesInstalled() error {
	deps, err := policy.ResolveInstalledDependencies(m.moduleManager.Load, m.module)
	if err != nil {
		return err
	}
	m.module.Dependencies = deps
	return nil
}

func (m *moduleInstaller) install() error {
	prepareStarted := time.Now()

	if m.checkModuleInstalled() {
		m.runtimeScope.Logger().Debug("module operation skipped", "module", m.module.Name, "reason", "already_installed")
		return nil
	}

	if err := m.validate(); err != nil {
		return xfmt.Errorf("error validating module: %w", err)
	}
	logModuleOperationStep(m.runtimeScope, m.ctx, plan.OpInstall, m.module.Name, moduleStepPrepare, prepareStarted)

	var buildResult *module.BuildResult
	persistLater := false
	if m.builder != nil {
		if split, ok := m.builder.(module.SplitBuilder); ok {
			buildStarted := time.Now()
			result, err := split.BuildWithoutPersist()
			if err != nil {
				return xfmt.Errorf("error building module: %w", err)
			}
			buildResult = result
			persistLater = true
			logModuleOperationStep(m.runtimeScope, m.ctx, plan.OpInstall, m.module.Name, moduleStepBuild, buildStarted)
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
	err := txRoot.Transactor().Required(ctx, func(txScope scope.Scope, tx scope.Transaction) error {
		committed := m.forCommitScope(txScope)
		return committed.commitInstall(buildResult, persistLater)
	})
	LogInstallOuterTxHold(m.runtimeScope.Logger(), "module_commit", txHoldStarted, err)
	if err != nil {
		return err
	}
	return m.finalizeInstall(buildResult)
}

func (m *moduleInstaller) forCommitScope(txScope scope.Scope) *moduleInstaller {
	committed := *m
	committed.runtimeScope = txScope
	entryPoint := ""
	if m.module != nil {
		entryPoint = m.module.ServiceEntryPoint
	}
	committed.builder = internalbackendbuilder.NewModuleBuilder(
		txScope,
		m.moduleManager.jsExecutor,
		m.module,
		entryPoint,
		internalbackendbuilder.WithPublishDist(false),
	)
	return &committed
}

func (m *moduleInstaller) commitInstall(buildResult *module.BuildResult, persistLater bool) error {
	if err := m.restoreModuleIfSoftDeleted(); err != nil {
		return err
	}

	if m.builder != nil {
		if persistLater {
			if split, ok := m.builder.(module.SplitBuilder); ok {
				if err := split.Persist(buildResult); err != nil {
					return xfmt.Errorf("error persisting module: %w", err)
				}
			} else {
				return xfmt.Errorf("builder does not support Persist for module %s", m.module.Name)
			}
		} else {
			buildStarted := time.Now()
			result, err := m.builder.Build()
			if err != nil {
				return xfmt.Errorf("error building module: %w", err)
			}
			buildResult = result
			logModuleOperationStep(m.runtimeScope, m.ctx, plan.OpInstall, m.module.Name, moduleStepBuild, buildStarted)
		}
	}

	initializeStarted := time.Now()
	if hookRunner, err := hooks.NewRunner(m.runtimeScope, m.moduleManager.jsExecutor, m.module); err != nil {
		return xfmt.Errorf("error preparing hooks for module %s: %w", m.module.Name, err)
	} else if hookRunner != nil {
		var hookScripts []*jsengine.JsScript
		if buildResult != nil {
			if script, err := hooks.ScriptFromBuildResult(buildResult); err != nil {
				return xfmt.Errorf("error preparing pre_init hook script: %w", err)
			} else if script != nil {
				hookScripts = append(hookScripts, script)
			}
		}
		if err := hookRunner.RunPhase(m.runtimeScope.Context(), hooks.PhasePreInit, hooks.RunOptions{Scripts: hookScripts, ReuseExecutorScripts: m.moduleManager != nil && m.moduleManager.jsExecutor != nil}); err != nil {
			return xfmt.Errorf("error running pre_init hook for module %s: %w", m.module.Name, err)
		}
	}
	logModuleOperationStep(m.runtimeScope, m.ctx, plan.OpInstall, m.module.Name, moduleStepInitialize, initializeStarted)

	migrator, err := schema.NewMigrator(m.runtimeScope, m.module)
	if err != nil {
		return xfmt.Errorf("error preparing schema migrator: %w", err)
	}
	schemaStarted := time.Now()
	if err := migrator.Migrate(); err != nil {
		return xfmt.Errorf("error migrating module: %w", err)
	}
	logModuleOperationStep(m.runtimeScope, m.ctx, plan.OpInstall, m.module.Name, moduleStepSchema, schemaStarted)

	dataLoader := dataloader.New(m.runtimeScope)
	applyCtx := m.runtimeScope.Context()
	if applyCtx == nil {
		applyCtx = context.Background()
	}
	dataStarted := time.Now()
	if err := dataLoader.ApplyModule(applyCtx, m.module, dataloader.ApplyOptions{WithDemo: m.ctx != nil && m.ctx.withDemo}); err != nil {
		return xfmt.Errorf("error applying data for module %s: %w", m.module.Name, err)
	}
	logModuleOperationStep(m.runtimeScope, m.ctx, plan.OpInstall, m.module.Name, moduleStepData, dataStarted)

	saveStarted := time.Now()
	m.module.Status = meta.Installed
	if len(m.module.Dependencies) > 0 {
		if err := m.runtimeScope.Session().Model(m.module).Association("Dependencies").Replace(m.module.Dependencies); err != nil {
			return xfmt.Errorf("error saving module dependencies: %w", err)
		}
	}
	// Omit association trees: Persist already wrote meta_raw_* + recomputed effective
	// meta_model*. Cascading Models here re-creates declaration shells with module_id and
	// duplicates logical names (breaks UI rpc dependency checks / UNIQUE(application,name)).
	if err := m.runtimeScope.Session().
		Omit("Dependencies", "Dependents", "Models", "Components", "UiResources").
		Save(m.module).Error; err != nil {
		return xfmt.Errorf("error saving module: %w", err)
	}
	logModuleOperationStep(m.runtimeScope, m.ctx, plan.OpInstall, m.module.Name, moduleStepSave, saveStarted)

	if err := importModuleTerminology(m.runtimeScope, m.module, runtimeOptionsFromScope(m.runtimeScope).modulesPath); err != nil {
		return err
	}

	if strings.EqualFold(strings.TrimSpace(m.module.Name), "meta") {
		if err := disableLegacyModuleIndexDailySchedule(m.runtimeScope); err != nil {
			return xfmt.Errorf("error disabling legacy module index schedule: %w", err)
		}
	}
	if strings.EqualFold(strings.TrimSpace(m.module.Name), "document") {
		if err := ensureDocumentAttachmentGCSchedule(m.runtimeScope); err != nil {
			return xfmt.Errorf("error ensuring document attachment gc schedule: %w", err)
		}
	}

	return nil
}

func (m *moduleInstaller) finalizeInstall(buildResult *module.BuildResult) error {
	finalizeStarted := time.Now()
	if hookRunner, err := hooks.NewRunner(m.runtimeScope, m.moduleManager.jsExecutor, m.module); err != nil {
		return xfmt.Errorf("error preparing hooks for module %s: %w", m.module.Name, err)
	} else if hookRunner != nil {
		var hookScripts []*jsengine.JsScript
		if buildResult != nil {
			if script, err := hooks.ScriptFromBuildResult(buildResult); err != nil {
				return xfmt.Errorf("error preparing post_init hook script: %w", err)
			} else if script != nil {
				hookScripts = append(hookScripts, script)
			}
		}
		if err := hookRunner.RunPhase(m.runtimeScope.Context(), hooks.PhasePostInit, hooks.RunOptions{Scripts: hookScripts, ReuseExecutorScripts: m.moduleManager != nil && m.moduleManager.jsExecutor != nil}); err != nil {
			return xfmt.Errorf("error running post_init hook for module %s: %w", m.module.Name, err)
		}
	}
	logModuleOperationStep(m.runtimeScope, m.ctx, plan.OpInstall, m.module.Name, moduleStepFinalize, finalizeStarted)

	return nil
}

func installerScheduleDB(runtimeScope scope.Scope) (*gorm.DB, error) {
	if runtimeScope == nil {
		return nil, xfmt.Errorf("missing db session")
	}
	ctx := runtimeScope.Context()
	if db, ok := scope.DBForScope(ctx, runtimeScope); ok {
		return db, nil
	}
	if runtimeScope.Session() == nil || runtimeScope.Session().DB == nil {
		return nil, xfmt.Errorf("missing db session")
	}
	return nil, xfmt.Errorf("missing db session")
}

func disableLegacyModuleIndexDailySchedule(runtimeScope scope.Scope) error {
	db, err := installerScheduleDB(runtimeScope)
	if err != nil {
		return err
	}
	if !db.Migrator().HasTable((&task.Schedule{}).TableName()) {
		return nil
	}
	return task.WhereScheduleNameEq(db, "meta.module_index.daily_sync").Delete(&task.Schedule{}).Error
}

func ensureDocumentAttachmentGCSchedule(runtimeScope scope.Scope) error {
	db, err := installerScheduleDB(runtimeScope)
	if err != nil {
		return err
	}
	const (
		scheduleName = "document.attachment.gc"
		targetApp    = "document"
		fullMethod   = "document.AttachmentContent/RunGarbageCollection"
		cronExpr     = "*/5 * * * *"
		timezone     = "UTC"
	)
	payload := datatypes.JSON(mustJSON(map[string]any{}))
	var existing task.Schedule
	res := task.WhereScheduleNameEq(db, scheduleName).Take(&existing)
	if res.Error != nil {
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			now := time.Now().UTC()
			return db.Create(&task.Schedule{
				Id:                xid.New().String(),
				Active:            true,
				Name:              task.EncodeTranslatedScheduleName(scheduleName),
				TargetApp:         targetApp,
				FullMethod:        fullMethod,
				PayloadTemplate:   payload,
				SchedulerUserId:   "admin",
				TriggeredByUserId: "admin",
				CronExpr:          cronExpr,
				Timezone:          timezone,
				TimeoutMs:         0,
				NextRunAt:         nil,
				CreatedAt:         now,
				UpdatedAt:         now,
			}).Error
		}
		return res.Error
	}
	updates := map[string]any{
		"active":                true,
		"target_app":            targetApp,
		"full_method":           fullMethod,
		"payload_template_json": payload,
		"cron_expr":             cronExpr,
		"timezone":              timezone,
		"timeout_ms":            int64(0),
		"updated_at":            time.Now().UTC(),
	}
	return db.Model(&task.Schedule{}).Where("id = ?", existing.Id).Updates(updates).Error
}

func mustJSON(value any) []byte {
	data, _ := json.Marshal(value)
	return data
}

func newModuleInstaller(runtimeScope scope.Scope, jsExecutor jsexecutor.ScriptExecutor, module *meta.Module, moduleManager *ModuleManager, ctx *opContext) *moduleInstaller {
	installer := &moduleInstaller{
		module:        module,
		runtimeScope:  runtimeScope,
		moduleManager: moduleManager,
		ctx:           ctx,
	}

	if module.ServiceEntryPoint != "" && !filepath.IsAbs(module.ServiceEntryPoint) {
		module.ServiceEntryPoint = filepath.Join(runtimeOptionsFromScope(runtimeScope).modulesPath, module.Name, module.ServiceEntryPoint)
	}
	installer.builder = internalbackendbuilder.NewModuleBuilder(runtimeScope, jsExecutor, module, module.ServiceEntryPoint, internalbackendbuilder.WithPublishDist(false))

	return installer
}
