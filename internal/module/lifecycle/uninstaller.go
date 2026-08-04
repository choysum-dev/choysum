// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"context"
	"errors"
	"slices"
	"strings"
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
	// Capture IMD victims before soft-delete so MetaModelData.ModelId can rebind to a surviving tip.
	var victims []modelVictim
	if err := db.Model(&meta.Model{}).
		Select("id, application, name").
		Where("module_id = ?", moduleID).
		Find(&victims).Error; err != nil {
		return xfmt.Errorf("error listing models for uninstall: %w", err)
	}

	modelIDs := db.Model(&meta.Model{}).Select("id").Where("module_id = ?", moduleID)
	serviceIDs := db.Model(&meta.Service{}).Select("id").Where("model_id IN (?)", modelIDs)
	fieldIDs := db.Model(&meta.Field{}).Select("id").Where("model_id IN (?)", modelIDs)
	componentIDs := db.Model(&meta.Component{}).Select("id").Where("module_id = ?", moduleID)
	uiResourceIDs := db.Model(&meta.UiResource{}).Select("id").Where("module_id = ?", moduleID)
	decoratorIDs := db.Model(&meta.Decorator{}).Select("id").Where(
		"model_id IN (?) OR service_id IN (?) OR field_id IN (?) OR component_id IN (?)",
		modelIDs, serviceIDs, fieldIDs, componentIDs,
	)

	if err := db.Where("decorator_id IN (?)", decoratorIDs).Delete(&meta.Argument{}).Error; err != nil {
		return xfmt.Errorf("error deleting decorator arguments: %w", err)
	}
	if err := db.Where("id IN (?)", decoratorIDs).Delete(&meta.Decorator{}).Error; err != nil {
		return xfmt.Errorf("error deleting decorators: %w", err)
	}
	if err := db.Where("service_id IN (?)", serviceIDs).Delete(&meta.TypeParameter{}).Error; err != nil {
		return xfmt.Errorf("error deleting type parameters: %w", err)
	}
	if err := db.Where("service_id IN (?)", serviceIDs).Delete(&meta.Parameter{}).Error; err != nil {
		return xfmt.Errorf("error deleting parameters: %w", err)
	}
	if err := db.Where("id IN (?)", serviceIDs).Delete(&meta.Service{}).Error; err != nil {
		return xfmt.Errorf("error deleting services: %w", err)
	}
	if err := db.Where("id IN (?)", fieldIDs).Delete(&meta.Field{}).Error; err != nil {
		return xfmt.Errorf("error deleting fields: %w", err)
	}
	if err := db.Where("id IN (?)", modelIDs).Delete(&meta.Model{}).Error; err != nil {
		return xfmt.Errorf("error deleting models: %w", err)
	}

	// IMD: other modules' MetaModelData rows may still point at soft-deleted MetaModel ids.
	// Rebind to the surviving tip for the same (application, name) when one remains.
	if err := rebindMetaModelDataTips(db.DB, victims); err != nil {
		return err
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

// rebindMetaModelDataTips updates MetaModelData.ModelId away from soft-deleted IMD rows
// onto the surviving tip for the same (application, name). No-op when the table is absent
// or when no live MetaModel remains for that logical model.
func rebindMetaModelDataTips(db *gorm.DB, victims []modelVictim) error {
	if len(victims) == 0 {
		return nil
	}
	if !db.Migrator().HasTable((&metadata.ModelData{}).TableName()) {
		return nil
	}

	type logicalModel struct {
		Application string
		Name        string
	}
	grouped := map[logicalModel][]string{}
	for _, v := range victims {
		id := strings.TrimSpace(v.Id)
		app := strings.TrimSpace(v.Application)
		name := strings.TrimSpace(v.Name)
		if id == "" || app == "" || name == "" {
			continue
		}
		key := logicalModel{Application: app, Name: name}
		grouped[key] = append(grouped[key], id)
	}

	for key, victimIDs := range grouped {
		tip, err := meta.ResolveMetaModelTip(db, key.Application, key.Name)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			return xfmt.Errorf("error resolving meta model tip for %s.%s: %w", key.Application, key.Name, err)
		}
		tipID := strings.TrimSpace(tip.Id.String)
		if tipID == "" {
			continue
		}
		if err := db.Model(&metadata.ModelData{}).
			Where("model_id IN ?", victimIDs).
			Update("model_id", tipID).Error; err != nil {
			return xfmt.Errorf("error rebinding meta model data for %s.%s: %w", key.Application, key.Name, err)
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
