// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package backendbuilder

import (
	"path/filepath"
	"strings"
	"sync"

	internalbackendplugin "github.com/choysum-dev/choysum/internal/esbplugins/backendplugin"
	module "github.com/choysum-dev/choysum/internal/module/artifact/result"
	"github.com/choysum-dev/choysum/internal/parser"
	"github.com/choysum-dev/choysum/pkg/meta"
	xfmt "golang.org/x/exp/errors/fmt"
	"gorm.io/gorm"
)

// FieldDefaultPlan is the Decide output for C2 virtual FieldDefault inject.
type FieldDefaultPlan struct {
	NeedInject       bool
	SupersedeVirtual bool
}

const (
	fieldDefaultGeneratedRelPath = "service/models/__generated__/field_default.ts"
	fieldDefaultModelName        = "FieldDefault"
	fieldDefaultDuplicateCode    = "FIELD_DEFAULT_DUPLICATE"
)

// fieldDefaultScheduledApps dedupes NeedInject across modules sharing one Application
// within a single process (install / upgrade).
var fieldDefaultScheduledApps sync.Map

const virtualFieldDefaultSource = `import { Model } from '@/core/service'
import FieldDefaultBaseModel from '@/core/service/orm/model/field_default_base_model'

@Model('FieldDefault')
export default class FieldDefault extends FieldDefaultBaseModel {}
`

type virtualSourceRegistrar interface {
	RegisterVirtualSource(path string, contents string)
}

func isGeneratedFieldDefaultPath(path string) bool {
	normalized := filepath.ToSlash(filepath.Clean(strings.TrimSpace(path)))
	if normalized == "" {
		return false
	}
	return strings.HasSuffix(normalized, "/"+fieldDefaultGeneratedRelPath) ||
		normalized == fieldDefaultGeneratedRelPath
}

func fieldDefaultGeneratedPath(modulePath string) string {
	return filepath.ToSlash(filepath.Clean(filepath.Join(strings.TrimSpace(modulePath), fieldDefaultGeneratedRelPath)))
}

func fieldDefaultsIn(results []*parser.ParserResult, modulePath string) []*meta.Model {
	out := make([]*meta.Model, 0)
	for _, result := range results {
		if result == nil || result.Model == nil {
			continue
		}
		m := result.Model
		if m.Name != fieldDefaultModelName || m.Abstract {
			continue
		}
		if modulePath != "" && !pathWithinModuleRoot(result.Path, modulePath) && !pathWithinModuleRoot(m.Path, modulePath) {
			continue
		}
		out = append(out, m)
	}
	return out
}

func handwrittenFieldDefaults(models []*meta.Model) []*meta.Model {
	out := make([]*meta.Model, 0, len(models))
	for _, m := range models {
		if m == nil || isGeneratedFieldDefaultPath(m.Path) {
			continue
		}
		out = append(out, m)
	}
	return out
}

func generatedFieldDefaults(models []*meta.Model) []*meta.Model {
	out := make([]*meta.Model, 0, len(models))
	for _, m := range models {
		if m == nil || !isGeneratedFieldDefaultPath(m.Path) {
			continue
		}
		out = append(out, m)
	}
	return out
}

func fieldDefaultSameModule(existing []*meta.Model, mod *meta.Module) bool {
	if mod == nil {
		return false
	}
	modPath := strings.TrimSpace(mod.Path)
	modID := ""
	if mod.Id.Valid {
		modID = strings.TrimSpace(mod.Id.String)
	}
	for _, m := range existing {
		if m == nil {
			continue
		}
		if modID != "" && m.ModuleId.Valid && strings.TrimSpace(m.ModuleId.String) == modID {
			return true
		}
		if modPath != "" && pathWithinModuleRoot(m.Path, modPath) {
			return true
		}
	}
	return false
}

func (b *ModuleBuilder) dbLoadFieldDefaults(app string) ([]*meta.Model, error) {
	app = strings.TrimSpace(app)
	if app == "" || b == nil || b.runtimeScope == nil || b.runtimeScope.Session() == nil {
		return nil, nil
	}
	var models []*meta.Model
	err := b.runtimeScope.Session().
		Where("application = ? AND name = ? AND abstract = ?", app, fieldDefaultModelName, false).
		Find(&models).Error
	if err != nil {
		return nil, err
	}
	return models, nil
}

func (b *ModuleBuilder) decideFieldDefaultPlan(prebuildResults []*parser.ParserResult) (FieldDefaultPlan, error) {
	plan := FieldDefaultPlan{}
	if b == nil || b.module == nil {
		return plan, nil
	}
	mod := b.module
	if strings.TrimSpace(mod.ServiceEntryPoint) == "" ||
		strings.TrimSpace(mod.ApplicationStr) == "" ||
		strings.TrimSpace(mod.ApplicationStr) == "core" {
		return plan, nil
	}

	app := strings.TrimSpace(mod.ApplicationStr)
	local := fieldDefaultsIn(prebuildResults, mod.Path)
	localHand := handwrittenFieldDefaults(local)
	if len(localHand) > 1 {
		return plan, xfmt.Errorf("%s: application %q has multiple handwritten FieldDefault models in module %q", fieldDefaultDuplicateCode, app, mod.Name)
	}

	existing, err := b.dbLoadFieldDefaults(app)
	if err != nil {
		return plan, xfmt.Errorf("load FieldDefault models for application %q: %w", app, err)
	}
	existingVirt := generatedFieldDefaults(existing)
	existingHand := handwrittenFieldDefaults(existing)

	if len(localHand) > 0 {
		if len(existingHand) > 0 && !fieldDefaultSameModule(existingHand, mod) {
			return plan, xfmt.Errorf("%s: application %q already has a handwritten FieldDefault outside module %q", fieldDefaultDuplicateCode, app, mod.Name)
		}
		if len(existingVirt) > 0 {
			return FieldDefaultPlan{SupersedeVirtual: true}, nil
		}
		return plan, nil
	}

	// No local handwritten FieldDefault — consider C2 inject.
	if len(existing) > 0 || len(local) > 0 {
		return plan, nil
	}
	if _, loaded := fieldDefaultScheduledApps.LoadOrStore(app, mod.Name); loaded {
		return plan, nil
	}
	return FieldDefaultPlan{NeedInject: true}, nil
}

func (b *ModuleBuilder) applyFieldDefaultInject(plan FieldDefaultPlan) error {
	if b == nil || !plan.NeedInject || b.module == nil {
		return nil
	}
	path := fieldDefaultGeneratedPath(b.module.Path)
	b.fieldDefaultInjectPath = path

	imports := append(b.entryPointImports(), path)
	if setter, ok := b.buildPlugin.(interface{ SetEntryPointImports([]string) }); ok {
		setter.SetEntryPointImports(imports)
	}
	if registrar, ok := b.buildPlugin.(virtualSourceRegistrar); ok {
		registrar.RegisterVirtualSource(path, virtualFieldDefaultSource)
	} else if bp, ok := b.buildPlugin.(*internalbackendplugin.BackendPlugin); ok && bp != nil {
		bp.RegisterVirtualSource(path, virtualFieldDefaultSource)
	}
	return nil
}

func (b *ModuleBuilder) validateFieldDefault(buildResult *module.BuildResult) error {
	if buildResult == nil || b == nil || b.module == nil {
		return nil
	}
	app := strings.TrimSpace(b.module.ApplicationStr)
	if app == "" || app == "core" {
		return nil
	}

	models := fieldDefaultsIn(module.ParserResults(buildResult), b.module.Path)
	if len(models) <= 1 {
		return nil
	}
	hand := handwrittenFieldDefaults(models)
	virt := generatedFieldDefaults(models)
	if len(hand) > 1 || (len(hand) > 0 && len(virt) > 0) {
		return xfmt.Errorf("%s: application %q build produced multiple FieldDefault models", fieldDefaultDuplicateCode, app)
	}
	return nil
}

func (b *ModuleBuilder) supersedeVirtualFieldDefaults() error {
	if b == nil || !b.fieldDefaultPlan.SupersedeVirtual || b.module == nil {
		return nil
	}
	if b.runtimeScope == nil || b.runtimeScope.Session() == nil {
		return nil
	}
	app := strings.TrimSpace(b.module.ApplicationStr)
	if app == "" {
		return nil
	}

	var existing []*meta.Model
	if err := b.runtimeScope.Session().
		Where("application = ? AND name = ? AND abstract = ?", app, fieldDefaultModelName, false).
		Find(&existing).Error; err != nil {
		return xfmt.Errorf("load FieldDefault rows for supersede: %w", err)
	}

	ids := make([]string, 0)
	for _, m := range existing {
		if m == nil || !isGeneratedFieldDefaultPath(m.Path) {
			continue
		}
		if !m.Id.Valid || strings.TrimSpace(m.Id.String) == "" {
			continue
		}
		ids = append(ids, m.Id.String)
	}
	if len(ids) == 0 {
		return nil
	}
	if result := b.runtimeScope.Session().Unscoped().Where("id IN ?", ids).Delete(&meta.Model{}); result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil
		}
		return xfmt.Errorf("delete superseded virtual FieldDefault rows: %w", result.Error)
	}
	return nil
}

// resetFieldDefaultScheduledAppsForTest clears the process-wide inject dedup map (tests only).
func resetFieldDefaultScheduledAppsForTest() {
	fieldDefaultScheduledApps.Range(func(key, _ any) bool {
		fieldDefaultScheduledApps.Delete(key)
		return true
	})
}
