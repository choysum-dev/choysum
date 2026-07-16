// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"errors"
	"slices"
	"time"

	"github.com/choysum-dev/choysum/internal/module/evolution/hooks"
	"github.com/choysum-dev/choysum/internal/module/plan"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	xfmt "golang.org/x/exp/errors/fmt"
	"gorm.io/gorm"
)

type moduleUninstaller struct {
	runtimeScope  scope.Scope
	module        *meta.IrModule
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

	var module meta.IrModule
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
	if err := db.Model(&meta.IrModule{}).Where("id = ?", moduleID).Update("status", meta.Uninstalled).Error; err != nil {
		return xfmt.Errorf("error updating module status: %w", err)
	}

	// Soft delete related meta records (soft delete won't trigger DB CASCADE).
	modelIDs := db.Model(&meta.IrModel{}).Select("id").Where("module_id = ?", moduleID)
	serviceIDs := db.Model(&meta.IrService{}).Select("id").Where("model_id IN (?)", modelIDs)
	fieldIDs := db.Model(&meta.IrField{}).Select("id").Where("model_id IN (?)", modelIDs)
	componentIDs := db.Model(&meta.IrComponent{}).Select("id").Where("module_id = ?", moduleID)
	uiResourceIDs := db.Model(&meta.IrUiResource{}).Select("id").Where("module_id = ?", moduleID)
	decoratorIDs := db.Model(&meta.IrDecorator{}).Select("id").Where(
		"model_id IN (?) OR service_id IN (?) OR field_id IN (?) OR component_id IN (?)",
		modelIDs, serviceIDs, fieldIDs, componentIDs,
	)

	if err := db.Where("decorator_id IN (?)", decoratorIDs).Delete(&meta.IrArgument{}).Error; err != nil {
		return xfmt.Errorf("error deleting decorator arguments: %w", err)
	}
	if err := db.Where("id IN (?)", decoratorIDs).Delete(&meta.IrDecorator{}).Error; err != nil {
		return xfmt.Errorf("error deleting decorators: %w", err)
	}
	if err := db.Where("service_id IN (?)", serviceIDs).Delete(&meta.IrTypeParameter{}).Error; err != nil {
		return xfmt.Errorf("error deleting type parameters: %w", err)
	}
	if err := db.Where("service_id IN (?)", serviceIDs).Delete(&meta.IrParameter{}).Error; err != nil {
		return xfmt.Errorf("error deleting parameters: %w", err)
	}
	if err := db.Where("id IN (?)", serviceIDs).Delete(&meta.IrService{}).Error; err != nil {
		return xfmt.Errorf("error deleting services: %w", err)
	}
	if err := db.Where("id IN (?)", fieldIDs).Delete(&meta.IrField{}).Error; err != nil {
		return xfmt.Errorf("error deleting fields: %w", err)
	}
	if err := db.Where("id IN (?)", modelIDs).Delete(&meta.IrModel{}).Error; err != nil {
		return xfmt.Errorf("error deleting models: %w", err)
	}
	if err := db.Where("id IN (?)", componentIDs).Delete(&meta.IrComponent{}).Error; err != nil {
		return xfmt.Errorf("error deleting components: %w", err)
	}
	if err := db.Where("menu_ui_resource_id IN (?) OR route_ui_resource_id IN (?)", uiResourceIDs, uiResourceIDs).Delete(&meta.IrUiResourceMenuRoute{}).Error; err != nil {
		return xfmt.Errorf("error deleting UI resource menu-route relations: %w", err)
	}
	if err := db.Where("route_ui_resource_id IN (?) OR action_ui_resource_id IN (?)", uiResourceIDs, uiResourceIDs).Delete(&meta.IrUiResourceRouteAction{}).Error; err != nil {
		return xfmt.Errorf("error deleting UI resource route-action relations: %w", err)
	}
	if err := db.Where("id IN (?)", uiResourceIDs).Delete(&meta.IrUiResource{}).Error; err != nil {
		return xfmt.Errorf("error deleting UI resources: %w", err)
	}

	// many2many join rows should be physically removed to avoid stale relations.
	if err := db.Exec(
		"DELETE FROM meta_ir_module_dependencies WHERE module_id = ? OR depend_module_id = ?",
		moduleID, moduleID,
	).Error; err != nil {
		return xfmt.Errorf("error deleting module dependency relations: %w", err)
	}

	return nil
}

func (m *moduleUninstaller) uninstall() error {
	// validate module
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

	// clean models from database
	cleanupStarted := time.Now()
	if err := m.cleanModels(); err != nil {
		return xfmt.Errorf("error cleaning models: %w", err)
	}
	if err := deleteModuleTerminology(m.runtimeScope, m.module); err != nil {
		return err
	}
	logModuleOperationStep(m.runtimeScope, m.ctx, plan.OpUninstall, m.module.Name, moduleStepCleanup, cleanupStarted)

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

func newModuleUninstaller(runtimeScope scope.Scope, module *meta.IrModule, moduleManager *ModuleManager, ctx *opContext) *moduleUninstaller {
	return &moduleUninstaller{
		runtimeScope:  runtimeScope,
		module:        module,
		moduleManager: moduleManager,
		ctx:           ctx,
	}
}
