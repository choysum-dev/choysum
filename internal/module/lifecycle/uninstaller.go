// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/choysum-dev/choysum/internal/module/evolution/hooks"
	modmeta "github.com/choysum-dev/choysum/internal/module/meta"
	"github.com/choysum-dev/choysum/internal/module/plan"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	xfmt "golang.org/x/exp/errors/fmt"
	"gorm.io/gorm"
)

type moduleUninstaller struct {
	runtimeScope  scope.Scope
	module        *meta.Module
	moduleManager *ModuleManager
	ctx           *opContext
}

func (m *moduleUninstaller) validate() error {
	if !slices.Contains([]meta.Status{meta.Installed, meta.ToUpgrade}, m.module.Status) {
		return xfmt.Errorf("module %s is not installed", m.module.Name)
	}
	return nil
}

func (m *moduleUninstaller) cleanModels() error {
	db := m.runtimeScope.Session()

	var module meta.Module
	if err := db.Unscoped().Where("name = ?", m.module.Name).Take(&module).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return xfmt.Errorf("error loading module for deletion: %w", err)
	}
	if !module.Id.Valid {
		return xfmt.Errorf("module %s has empty id", m.module.Name)
	}
	moduleID := module.Id.String

	// Mark status for audit before soft delete.
	if err := db.Model(&meta.Module{}).Where("id = ?", moduleID).Update("status", meta.Uninstalled).Error; err != nil {
		return xfmt.Errorf("error updating module status: %w", err)
	}

	componentIDs := db.Model(&meta.Component{}).Select("id").Where("module_id = ?", moduleID)
	uiResourceIDs := db.Model(&meta.UiResource{}).Select("id").Where("module_id = ?", moduleID)
	// Component decorators still live on the shared meta_decorator table.
	componentDecoratorIDs := db.Model(&meta.Decorator{}).Select("id").Where(
		"component_id IN (?)",
		componentIDs,
	)

	// Remove declaration trees then rebuild effective projections (symmetric with Replace+Flush).
	keys, err := modmeta.RemoveModuleDeclarations(db.DB, moduleID)
	if err != nil {
		return xfmt.Errorf("error removing module declarations: %w", err)
	}
	if err := modmeta.FlushEffective(db.DB, keys); err != nil {
		return xfmt.Errorf("error flushing effective models after uninstall: %w", err)
	}

	if err := db.Where("decorator_id IN (?)", componentDecoratorIDs).Delete(&meta.Argument{}).Error; err != nil {
		return xfmt.Errorf("error deleting component decorator arguments: %w", err)
	}
	if err := db.Where("id IN (?)", componentDecoratorIDs).Delete(&meta.Decorator{}).Error; err != nil {
		return xfmt.Errorf("error deleting component decorators: %w", err)
	}
	if err := db.Where("id IN (?)", componentIDs).Delete(&meta.Component{}).Error; err != nil {
		return xfmt.Errorf("error deleting components: %w", err)
	}
	if err := db.Where("menu_ui_resource_id IN (?) OR route_ui_resource_id IN (?)", uiResourceIDs, uiResourceIDs).Delete(&meta.UiResourceMenuRoute{}).Error; err != nil {
		return xfmt.Errorf("error deleting UI resource menu-route relations: %w", err)
	}
	if err := db.Where("route_ui_resource_id IN (?) OR action_ui_resource_id IN (?)", uiResourceIDs, uiResourceIDs).Delete(&meta.UiResourceRouteAction{}).Error; err != nil {
		return xfmt.Errorf("error deleting UI resource route-action relations: %w", err)
	}
	if err := db.Where("id IN (?)", uiResourceIDs).Delete(&meta.UiResource{}).Error; err != nil {
		return xfmt.Errorf("error deleting UI resources: %w", err)
	}

	// many2many join rows should be physically removed to avoid stale relations.
	if err := db.Exec(
		"DELETE FROM meta_module_dependencies WHERE module_id = ? OR depend_module_id = ?",
		moduleID, moduleID,
	).Error; err != nil {
		return xfmt.Errorf("error deleting module dependency relations: %w", err)
	}

	// Physically delete MetaModelData registry rows for this module name (xml_id mappings).
	// Soft-delete would leave (module, name) unique-index collisions that block reinstall
	// (loader scoped First misses soft-deleted rows, then Create fails). Do not cascade-
	// delete the business seed rows those mappings point at (V1).
	if db.Migrator().HasTable((&modmeta.ModelData{}).TableName()) {
		if err := db.Unscoped().Where("module = ?", m.module.Name).Delete(&modmeta.ModelData{}).Error; err != nil {
			return xfmt.Errorf("error deleting meta model data mappings: %w", err)
		}
	}

	// SF7: hard-delete web.SavedFilter rows only when a logical model has no remaining
	// live meta_model after this module's declarations were removed (IMD-safe).
	if err := purgeSavedFiltersForGoneModels(db.DB, keys); err != nil {
		return err
	}

	return nil
}

const webSavedFilterTable = "web_saved_filter"

// purgeSavedFiltersForGoneModels deletes Favorites for logical models that no longer
// have any live effective meta_model row. No-op when the table is missing. Never
// deletes by Application alone.
func purgeSavedFiltersForGoneModels(db *gorm.DB, keys []modmeta.LogicalKey) error {
	if db == nil || len(keys) == 0 {
		return nil
	}
	// Prefer HasTable so views (MySQL GetTables) and case-aliased relations are not
	// mistaken for the concrete web_saved_filter base table used by the DELETE below.
	if !db.Migrator().HasTable(webSavedFilterTable) {
		return nil
	}
	seen := map[string]struct{}{}
	for _, key := range keys {
		k := key.Normalized()
		if !k.Valid() {
			continue
		}
		id := k.Application + "\x00" + k.Name
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}

		var remaining int64
		if err := db.Model(&meta.Model{}).
			Where("application = ? AND name = ?", k.Application, k.Name).
			Count(&remaining).Error; err != nil {
			return xfmt.Errorf("error counting surviving meta models for saved filter purge: %w", err)
		}
		if remaining > 0 {
			continue
		}
		if err := db.Exec(
			"DELETE FROM "+webSavedFilterTable+" WHERE application = ? AND model_name = ?",
			k.Application, k.Name,
		).Error; err != nil {
			return xfmt.Errorf("error deleting web saved filters for %s.%s: %w", k.Application, k.Name, err)
		}
	}
	return nil
}

func (m *moduleUninstaller) uninstall() error {
	prepareStarted := time.Now()
	if err := m.validate(); err != nil {
		return xfmt.Errorf("error validating module uninstallation: %w", err)
	}

	if hookRunner, err := hooks.NewRunner(m.runtimeScope, m.moduleManager.jsExecutor, m.module); err != nil {
		return xfmt.Errorf("error preparing hooks for module %s: %w", m.module.Name, err)
	} else if hookRunner != nil {
		var hookScripts []*jsengine.JsScript
		if err := hookRunner.RunPhase(m.runtimeScope.Context(), hooks.PhasePreUninstall, hooks.RunOptions{Scripts: hookScripts}); err != nil {
			return xfmt.Errorf("error running pre_uninstall hook for module %s: %w", m.module.Name, err)
		}
	}
	logModuleOperationStep(m.runtimeScope, m.ctx, plan.OpUninstall, m.module.Name, moduleStepPrepare, prepareStarted)

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
		committed := *m
		committed.runtimeScope = txScope
		return committed.commitUninstall()
	})
	LogModuleCommitTxHold(m.runtimeScope.Logger(), "uninstall", "module_commit", txHoldStarted, err)
	if err != nil {
		return err
	}

	return m.finalizeUninstall()
}

func (m *moduleUninstaller) commitUninstall() error {
	cleanupStarted := time.Now()
	if err := m.cleanModels(); err != nil {
		return xfmt.Errorf("error cleaning models: %w", err)
	}
	if err := deleteModuleTerminology(m.runtimeScope, m.module); err != nil {
		return err
	}
	logModuleOperationStep(m.runtimeScope, m.ctx, plan.OpUninstall, m.module.Name, moduleStepCleanup, cleanupStarted)
	return nil
}

func (m *moduleUninstaller) finalizeUninstall() error {
	finalizeStarted := time.Now()
	if hookRunner, err := hooks.NewRunner(m.runtimeScope, m.moduleManager.jsExecutor, m.module); err != nil {
		return xfmt.Errorf("error preparing hooks for module %s: %w", m.module.Name, err)
	} else if hookRunner != nil {
		var hookScripts []*jsengine.JsScript
		if err := hookRunner.RunPhase(m.runtimeScope.Context(), hooks.PhasePostUninstall, hooks.RunOptions{Scripts: hookScripts}); err != nil {
			return xfmt.Errorf("error running post_uninstall hook for module %s: %w", m.module.Name, err)
		}
	}
	logModuleOperationStep(m.runtimeScope, m.ctx, plan.OpUninstall, m.module.Name, moduleStepFinalize, finalizeStarted)
	return nil
}

func newModuleUninstaller(runtimeScope scope.Scope, module *meta.Module, moduleManager *ModuleManager, ctx *opContext) *moduleUninstaller {
	return &moduleUninstaller{
		runtimeScope:  runtimeScope,
		module:        module,
		moduleManager: moduleManager,
		ctx:           ctx,
	}
}
