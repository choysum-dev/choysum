// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	internalbackendbuilder "github.com/choysum-dev/choysum/internal/module/artifact/build/backend"
	module "github.com/choysum-dev/choysum/internal/module/artifact/result"
	"github.com/choysum-dev/choysum/internal/module/evolution/hooks"
	"github.com/choysum-dev/choysum/internal/module/evolution/schema"
	"github.com/choysum-dev/choysum/internal/module/plan"
	"github.com/choysum-dev/choysum/internal/module/policy"
	"github.com/choysum-dev/choysum/internal/persistence/sqliteretry"
	"github.com/choysum-dev/choysum/internal/task"

	importpkg "github.com/choysum-dev/choysum/pkg/import"
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

func (m *moduleInstaller) install() (didInstall bool, err error) {
	prepareStarted := time.Now()

	if m.checkModuleInstalled() {
		m.runtimeScope.Logger().Debug("module operation skipped", "module", m.module.Name, "reason", "already_installed")
		return false, nil
	}

	if err := m.validate(); err != nil {
		return false, xfmt.Errorf("error validating module: %w", err)
	}
	logModuleOperationStep(m.runtimeScope, m.ctx, plan.OpInstall, m.module.Name, moduleStepPrepare, prepareStarted)

	var buildResult *module.BuildResult
	persistLater := false
	if m.builder != nil {
		if split, ok := m.builder.(module.SplitBuilder); ok {
			buildStarted := time.Now()
			result, err := split.BuildWithoutPersist()
			if err != nil {
				return false, xfmt.Errorf("error building module: %w", err)
			}
			buildResult = result
			persistLater = true
			logModuleOperationStep(m.runtimeScope, m.ctx, plan.OpInstall, m.module.Name, moduleStepBuild, buildStarted)
		}
	}

	if err := m.installAfterPrepare(buildResult, persistLater); err != nil {
		return false, err
	}
	return true, nil
}

// installAfterPrepare runs the install commit TX, then TX-external pre_init, then finalize.
// Commit holds only Persist/schema/data/save; pre_init sees already-persisted IR and must not
// extend the Required TX. If post-commit hooks fail, status is reverted to ToInstall so a
// later install retry is not skipped as already_installed.
func (m *moduleInstaller) installAfterPrepare(buildResult *module.BuildResult, persistLater bool) error {
	if m == nil {
		return xfmt.Errorf("scope is nil")
	}
	if m.runtimeScope == nil {
		return xfmt.Errorf("scope is nil")
	}
	txHoldStarted := time.Now()
	err := m.runInstallCommitTX(m.runtimeScope, m.runtimeScope.Context(), &buildResult, persistLater)
	LogInstallOuterTxHold(m.runtimeScope.Logger(), "module_commit", txHoldStarted, err)
	if err != nil {
		return err
	}
	// Commit already marked Installed. On panic before finalize succeeds, revert so retry
	// is not skipped as already_installed. Normal hook errors use wrapPostCommitHookError.
	finalized := false
	defer func() {
		if finalized {
			return
		}
		if r := recover(); r != nil {
			if markErr := m.markPostCommitHooksIncomplete(); markErr != nil {
				name := ""
				if m.module != nil {
					name = m.module.Name
				}
				if m.runtimeScope != nil && m.runtimeScope.Logger() != nil {
					m.runtimeScope.Logger().Error(
						"failed reverting module status after post-commit panic",
						"module", name,
						"error", markErr,
					)
				}
			}
			panic(r)
		}
	}()
	if err := m.runInstallPreInit(buildResult); err != nil {
		return m.wrapPostCommitHookError("error running pre_init after commit (module persisted, not finalized)", err)
	}
	if err := m.finalizeInstall(buildResult); err != nil {
		return m.wrapPostCommitHookError("error finalizing install after commit (module persisted, not finalized)", err)
	}
	finalized = true
	return nil
}

// wrapPostCommitHookError reverts status for retry and annotates that Commit already persisted.
func (m *moduleInstaller) wrapPostCommitHookError(msg string, err error) error {
	markErr := m.markPostCommitHooksIncomplete()
	if markErr != nil {
		return fmt.Errorf("%s: %w (also failed reverting status: %w)", msg, err, markErr)
	}
	return xfmt.Errorf("%s: %w", msg, err)
}

// runInstallCommitTX runs the install commit Required TX, pausing lease renew when a manager is set.
func (m *moduleInstaller) runInstallCommitTX(txRoot scope.Scope, ctx context.Context, buildResult **module.BuildResult, persistLater bool) error {
	if txRoot == nil {
		return xfmt.Errorf("scope is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if buildResult == nil {
		return xfmt.Errorf("build result slot is nil")
	}
	var committedResult *module.BuildResult
	var committedModule *meta.Module
	var origBuildModule *meta.Module
	if *buildResult != nil {
		origBuildModule = (*buildResult).Module
	}
	err := runWithLeaseRenewPaused(m.moduleManager, func() error {
		return txRoot.Transactor().Required(ctx, func(txScope scope.Scope, _ scope.Transaction) error {
			committed := m.forCommitScope(txScope)
			result, commitErr := committed.commitInstall(*buildResult, persistLater)
			if commitErr != nil {
				return commitErr
			}
			committedResult = result
			committedModule = committed.module
			return nil
		})
	})
	if err != nil {
		// Undo bindCommitBuildModule: the TX-local module copy was discarded.
		if *buildResult != nil {
			(*buildResult).Module = origBuildModule
		}
		return err
	}
	*buildResult = committedResult
	if committedModule != nil && m.module != nil {
		*m.module = *committedModule
	}
	if *buildResult != nil && m.module != nil {
		// Repoint the published result at the caller's module; the TX-local copy
		// must not escape the commit.
		(*buildResult).Module = m.module
	}
	return nil
}

func (m *moduleInstaller) forCommitScope(txScope scope.Scope) *moduleInstaller {
	committed := *m
	committed.runtimeScope = txScope
	if m.module != nil {
		// Transaction-local module copy: commitInstall mutates Status/Id before the
		// outer Required TX commits; keep the caller's module unchanged on rollback.
		modCopy := *m.module
		committed.module = &modCopy
	}
	committed.builder = internalbackendbuilder.NewModuleBuilder(
		txScope,
		installerJSExecutor(m),
		committed.module,
		installerServiceEntryPoint(&committed),
		internalbackendbuilder.WithPublishDist(false),
	)
	return &committed
}

// bindCommitBuildModule points Persist at the transaction-local module copy.
// BuildWithoutPersist runs on the outer installer and leaves buildResult.Module on
// that pointer; without rebinding, Persist would insert the outer row while
// commitInstall Saves the copy → UNIQUE(meta_module.name).
func bindCommitBuildModule(buildResult *module.BuildResult, mod *meta.Module) {
	if buildResult == nil || mod == nil {
		return
	}
	buildResult.Module = mod
}

// installerJSExecutor returns the manager JS executor when present.
func installerJSExecutor(m *moduleInstaller) jsexecutor.ScriptExecutor {
	if m == nil {
		return nil
	}
	if m.moduleManager == nil {
		return nil
	}
	return m.moduleManager.jsExecutor
}

// installerServiceEntryPoint returns the module service entry point when present.
func installerServiceEntryPoint(m *moduleInstaller) string {
	if m == nil {
		return ""
	}
	if m.module == nil {
		return ""
	}
	return m.module.ServiceEntryPoint
}

// installerReuseExecutorScripts reports whether hook RunPhase may reuse the JS executor.
func installerReuseExecutorScripts(exec jsexecutor.ScriptExecutor) bool {
	if exec == nil {
		return false
	}
	return true
}

func (m *moduleInstaller) commitInstall(buildResult *module.BuildResult, persistLater bool) (*module.BuildResult, error) {
	if m == nil || m.module == nil {
		return nil, xfmt.Errorf("install commit installer is nil")
	}
	if err := m.restoreModuleIfSoftDeleted(); err != nil {
		return nil, err
	}
	bindCommitBuildModule(buildResult, m.module)

	if m.builder != nil {
		if persistLater {
			if split, ok := m.builder.(module.SplitBuilder); ok {
				if err := split.Persist(buildResult); err != nil {
					return nil, xfmt.Errorf("error persisting module: %w", err)
				}
			} else {
				return nil, xfmt.Errorf("builder does not support Persist for module %s", m.module.Name)
			}
		} else {
			buildStarted := time.Now()
			result, err := m.builder.Build()
			if err != nil {
				return nil, xfmt.Errorf("error building module: %w", err)
			}
			buildResult = result
			logModuleOperationStep(m.runtimeScope, m.ctx, plan.OpInstall, m.module.Name, moduleStepBuild, buildStarted)
		}
	}

	migrator, err := newInstallSchemaMigrator(m.runtimeScope, m.module)
	if err != nil {
		return nil, xfmt.Errorf("error preparing schema migrator: %w", err)
	}
	schemaStarted := time.Now()
	if err := migrator.Migrate(); err != nil {
		return nil, xfmt.Errorf("error migrating module: %w", err)
	}
	logModuleOperationStep(m.runtimeScope, m.ctx, plan.OpInstall, m.module.Name, moduleStepSchema, schemaStarted)

	applyCtx := m.runtimeScope.Context()
	if applyCtx == nil {
		applyCtx = context.Background()
	}
	dataStarted := time.Now()
	if err := applyInitdata(applyCtx, m.runtimeScope, m.module, importpkg.CallerLifecycle, m.ctx != nil && m.ctx.withDemo); err != nil {
		return nil, xfmt.Errorf("error applying data for module %s: %w", m.module.Name, err)
	}
	logModuleOperationStep(m.runtimeScope, m.ctx, plan.OpInstall, m.module.Name, moduleStepData, dataStarted)

	saveStarted := time.Now()
	m.module.Status = meta.Installed
	if err := sqliteretry.WithLockRetry(func() error {
		return replaceModuleDependenciesFn(m.runtimeScope.Session(), m.module)
	}); err != nil {
		return nil, xfmt.Errorf("error saving module dependencies: %w", err)
	}
	// Omit association trees: Persist already wrote meta_raw_* + recomputed effective
	// meta_model*. Cascading Models here re-creates declaration shells with module_id and
	// duplicates logical names (breaks UI rpc dependency checks / UNIQUE(application,name)).
	if err := sqliteretry.WithLockRetry(func() error {
		return m.runtimeScope.Session().
			Omit("Dependencies", "Dependents", "Models", "Components", "UiResources").
			Save(m.module).Error
	}); err != nil {
		return nil, xfmt.Errorf("error saving module: %w", err)
	}
	logModuleOperationStep(m.runtimeScope, m.ctx, plan.OpInstall, m.module.Name, moduleStepSave, saveStarted)

	if err := importModuleTerminology(m.runtimeScope, m.module, runtimeOptionsFromScope(m.runtimeScope).modulesPath); err != nil {
		return nil, err
	}

	if strings.EqualFold(strings.TrimSpace(m.module.Name), "meta") {
		if err := disableLegacyModuleIndexDailySchedule(m.runtimeScope); err != nil {
			return nil, xfmt.Errorf("error disabling legacy module index schedule: %w", err)
		}
	}
	if strings.EqualFold(strings.TrimSpace(m.module.Name), "document") {
		if err := ensureDocumentAttachmentGCSchedule(m.runtimeScope); err != nil {
			return nil, xfmt.Errorf("error ensuring document attachment gc schedule: %w", err)
		}
	}

	return buildResult, nil
}

// runInstallPreInit runs PhasePreInit outside the Commit TX so hook JS does not extend TX hold.
// Callers must invoke this only after a successful commit (Persisted IR + schema/data/save).
func (m *moduleInstaller) runInstallPreInit(buildResult *module.BuildResult) error {
	if m == nil {
		return nil
	}
	if m.runtimeScope == nil {
		return xfmt.Errorf("scope is nil")
	}
	if m.module == nil {
		return xfmt.Errorf("module is nil")
	}
	initializeStarted := time.Now()
	if err := runInstallHookPhase(m.runtimeScope, m.ctx, plan.OpInstall, installerJSExecutor(m), m.module, hooks.PhasePreInit, buildResult, "pre_init"); err != nil {
		return err
	}
	logModuleOperationStep(m.runtimeScope, m.ctx, plan.OpInstall, m.module.Name, moduleStepInitialize, initializeStarted)
	return nil
}

// updatePostCommitIncompleteStatus flips Installed → ToInstall for retry. Overridable in tests.
var updatePostCommitIncompleteStatus = func(sess *scope.Session, mod *meta.Module) (int64, error) {
	if sess == nil {
		return 0, xfmt.Errorf("session is nil")
	}
	if mod == nil {
		return 0, xfmt.Errorf("module is nil")
	}
	name := strings.TrimSpace(mod.Name)
	query := sess.Model(&meta.Module{}).Where("status = ?", meta.Installed)
	if mod.Id.Valid && strings.TrimSpace(mod.Id.String) != "" {
		query = query.Where("id = ?", mod.Id.String)
	} else if name != "" {
		query = query.Where("name = ?", name)
	} else {
		return 0, xfmt.Errorf("module id and name are both empty; refusing unqualified status update")
	}
	res := query.Update("status", meta.ToInstall)
	return res.RowsAffected, res.Error
}

// markPostCommitHooksIncomplete reverts status to ToInstall after a post-commit hook failure
// so a subsequent install is not skipped as already_installed. Persist/schema/data stay.
// Only mutates in-memory status after the DB update succeeds.
func (m *moduleInstaller) markPostCommitHooksIncomplete() error {
	if m == nil || m.module == nil {
		return nil
	}
	if m.runtimeScope == nil || m.runtimeScope.Session() == nil {
		return xfmt.Errorf("cannot revert module %q status: runtime scope session is nil", m.module.Name)
	}
	name := strings.TrimSpace(m.module.Name)
	if name == "" && !(m.module.Id.Valid && strings.TrimSpace(m.module.Id.String) != "") {
		return xfmt.Errorf("cannot revert module status: module id and name are both empty")
	}
	var affected int64
	if err := sqliteretry.WithLockRetry(func() error {
		n, err := updatePostCommitIncompleteStatus(m.runtimeScope.Session(), m.module)
		affected = n
		return err
	}); err != nil {
		return err
	}
	if affected == 0 {
		identifier := name
		if identifier == "" && m.module.Id.Valid {
			identifier = strings.TrimSpace(m.module.Id.String)
		}
		return xfmt.Errorf("module %q was not %q; status left unchanged", identifier, meta.Installed)
	}
	m.module.Status = meta.ToInstall
	return nil
}

func (m *moduleInstaller) finalizeInstall(buildResult *module.BuildResult) error {
	finalizeStarted := time.Now()
	if m == nil {
		return nil
	}
	if err := runInstallHookPhase(m.runtimeScope, m.ctx, plan.OpInstall, installerJSExecutor(m), m.module, hooks.PhasePostInit, buildResult, "post_init"); err != nil {
		return err
	}
	name := ""
	if m.module != nil {
		name = m.module.Name
	}
	logModuleOperationStep(m.runtimeScope, m.ctx, plan.OpInstall, name, moduleStepFinalize, finalizeStarted)

	return nil
}

// runInstallHookPhase runs one install hook phase.
func runInstallHookPhase(
	runtimeScope scope.Scope,
	opCtx *opContext,
	op plan.OpType,
	jsExec jsexecutor.ScriptExecutor,
	module *meta.Module,
	phase hooks.Phase,
	buildResult *module.BuildResult,
	phaseLabel string,
) error {
	hookRunner, err := hooksNewRunner(runtimeScope, jsExec, module)
	if err != nil {
		name := ""
		if module != nil {
			name = module.Name
		}
		return xfmt.Errorf("error preparing %s hooks for module %s: %w", phaseLabel, name, err)
	}
	if hookRunner == nil {
		return nil
	}
	var hookScripts []*jsengine.JsScript
	if buildResult != nil {
		script, scriptErr := hooksScriptFromBuildResult(buildResult)
		if scriptErr != nil {
			return xfmt.Errorf("error preparing %s hook script: %w", phaseLabel, scriptErr)
		}
		if script != nil {
			hookScripts = append(hookScripts, script)
		}
	}
	name := ""
	if module != nil {
		name = module.Name
	}
	started := time.Now()
	if err := hookRunner.RunPhase(runtimeScope.Context(), phase, hooks.RunOptions{
		Scripts:              hookScripts,
		ReuseExecutorScripts: installerReuseExecutorScripts(jsExec),
	}); err != nil {
		return xfmt.Errorf("error running %s hook for module %s: %w", phaseLabel, name, err)
	}
	logModuleOperationStep(runtimeScope, opCtx, op, name, moduleStepHook(phase), started)
	return nil
}

// Overridable in tests to exercise hook preparation failure paths.
var (
	hooksNewRunner             = hooks.NewRunner
	hooksScriptFromBuildResult = hooks.ScriptFromBuildResult
	newInstallSchemaMigrator   = schema.NewMigrator
	installerScheduleDBFn      = installerScheduleDB
)

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
	db, err := installerScheduleDBFn(runtimeScope)
	if err != nil {
		return err
	}
	if !db.Migrator().HasTable((&task.Schedule{}).TableName()) {
		return nil
	}
	return task.WhereScheduleNameEq(db, "meta.module_index.daily_sync").Delete(&task.Schedule{}).Error
}

func ensureDocumentAttachmentGCSchedule(runtimeScope scope.Scope) error {
	db, err := installerScheduleDBFn(runtimeScope)
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
