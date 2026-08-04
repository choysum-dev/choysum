// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/choysum-dev/choysum/internal/module/evolution/hooks"
	metadata "github.com/choysum-dev/choysum/internal/module/metadata"
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

	// Soft delete related meta records (soft delete won't trigger DB CASCADE).
	// Collect logical names from declaration-layer raw rows before delete.
	var victims []modelVictim
	if err := db.Model(&meta.RawModel{}).
		Select("id, application, name").
		Where("module_id = ?", moduleID).
		Find(&victims).Error; err != nil {
		return xfmt.Errorf("error listing raw models for uninstall: %w", err)
	}

	modelIDs := db.Model(&meta.RawModel{}).Select("id").Where("module_id = ?", moduleID)
	serviceIDs := db.Model(&meta.RawService{}).Select("id").Where("model_id IN (?)", modelIDs)
	fieldIDs := db.Model(&meta.RawField{}).Select("id").Where("model_id IN (?)", modelIDs)
	componentIDs := db.Model(&meta.Component{}).Select("id").Where("module_id = ?", moduleID)
	uiResourceIDs := db.Model(&meta.UiResource{}).Select("id").Where("module_id = ?", moduleID)
	rawDecoratorIDs := db.Model(&meta.RawDecorator{}).Select("id").Where(
		"model_id IN (?) OR service_id IN (?) OR field_id IN (?)",
		modelIDs, serviceIDs, fieldIDs,
	)
	// Component decorators still live on the shared meta_decorator table.
	componentDecoratorIDs := db.Model(&meta.Decorator{}).Select("id").Where(
		"component_id IN (?)",
		componentIDs,
	)

	if err := db.Where("decorator_id IN (?)", rawDecoratorIDs).Delete(&meta.RawArgument{}).Error; err != nil {
		return xfmt.Errorf("error deleting raw decorator arguments: %w", err)
	}
	if err := db.Where("id IN (?)", rawDecoratorIDs).Delete(&meta.RawDecorator{}).Error; err != nil {
		return xfmt.Errorf("error deleting raw decorators: %w", err)
	}
	if err := db.Where("decorator_id IN (?)", componentDecoratorIDs).Delete(&meta.Argument{}).Error; err != nil {
		return xfmt.Errorf("error deleting component decorator arguments: %w", err)
	}
	if err := db.Where("id IN (?)", componentDecoratorIDs).Delete(&meta.Decorator{}).Error; err != nil {
		return xfmt.Errorf("error deleting component decorators: %w", err)
	}
	if err := db.Where("service_id IN (?)", serviceIDs).Delete(&meta.RawTypeParameter{}).Error; err != nil {
		return xfmt.Errorf("error deleting raw type parameters: %w", err)
	}
	if err := db.Where("service_id IN (?)", serviceIDs).Delete(&meta.RawParameter{}).Error; err != nil {
		return xfmt.Errorf("error deleting raw parameters: %w", err)
	}
	if err := db.Where("id IN (?)", serviceIDs).Delete(&meta.RawService{}).Error; err != nil {
		return xfmt.Errorf("error deleting raw services: %w", err)
	}
	if err := db.Where("id IN (?)", fieldIDs).Delete(&meta.RawField{}).Error; err != nil {
		return xfmt.Errorf("error deleting raw fields: %w", err)
	}
	if err := db.Where("id IN (?)", modelIDs).Delete(&meta.RawModel{}).Error; err != nil {
		return xfmt.Errorf("error deleting raw models: %w", err)
	}

	// Rebuild effective projections for touched logical names (EDS-2; no tip rebind).
	keys := make([]meta.LogicalKey, 0, len(victims))
	for _, v := range victims {
		keys = append(keys, meta.LogicalKey{Application: v.Application, Name: v.Name})
	}
	if err := meta.RecomputeKeys(db.DB, keys); err != nil {
		return xfmt.Errorf("error recomputing effective models after uninstall: %w", err)
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
	if db.Migrator().HasTable((&metadata.ModelData{}).TableName()) {
		if err := db.Unscoped().Where("module = ?", m.module.Name).Delete(&metadata.ModelData{}).Error; err != nil {
			return xfmt.Errorf("error deleting meta model data mappings: %w", err)
		}
	}

	return nil
}

type modelVictim struct {
	Id          string
	Application string
	Name        string
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
