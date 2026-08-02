// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package backendbuilder

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

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
	// scheduledApp is set when this builder claimed the process-wide NeedInject slot.
	// Cleared via releaseFieldDefaultSchedule on failure or after Persist/Bundle.
	scheduledApp string
}

const (
	fieldDefaultGeneratedRelPath = "service/models/__generated__/field_default.ts"
	fieldDefaultModelName        = "FieldDefault"
	fieldDefaultDuplicateCode    = "FIELD_DEFAULT_DUPLICATE"
)

// fieldDefaultScheduledApps dedupes NeedInject across modules sharing one Application
// within a single process (install / upgrade). Entries must be released when the
// claiming build/persist does not complete successfully.
var fieldDefaultScheduledApps sync.Map

type virtualSourceRegistrar interface {
	RegisterVirtualSource(path string, contents string)
}

// virtualFieldDefaultSource builds C2 thin-class source with absolute imports so
// esbuild can resolve them even when the pseudo __generated__ directory is not on disk
// (path aliases like @/core/service fail for virtual OnLoad paths).
//
// application must be baked into @Model options: FieldDefault is often first loaded on
// the build pass (after prebuild), so injectModelApplication runs too late to rewrite
// the JS that actually lands in dist/bundles/index.js. Without application, the
// runtime pool key becomes "application.FieldDefault" and lookup `{app}.FieldDefault` misses.
func virtualFieldDefaultSource(modulesPath string, application string) string {
	modulesPath = filepath.ToSlash(filepath.Clean(strings.TrimSpace(modulesPath)))
	application = strings.TrimSpace(application)
	if application == "" {
		application = "application"
	}
	coreService := filepath.ToSlash(filepath.Join(modulesPath, "core/service/index.ts"))
	baseModel := filepath.ToSlash(filepath.Join(modulesPath, "core/service/orm/model/field_default_base_model.ts"))
	// strconv.Quote produces valid TS/JS string literals (backslash, quotes, newlines).
	return fmt.Sprintf(`import { Model } from %s
import FieldDefaultBaseModel from %s

@Model('FieldDefault', { application: %s })
export default class FieldDefault extends FieldDefaultBaseModel {}
`, strconv.Quote(coreService), strconv.Quote(baseModel), strconv.Quote(application))
}

func isGeneratedFieldDefaultPath(path string) bool {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return false
	}
	normalized := filepath.ToSlash(filepath.Clean(trimmed))
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

func (b *ModuleBuilder) releaseFieldDefaultSchedule() {
	if b == nil {
		return
	}
	app := strings.TrimSpace(b.fieldDefaultPlan.scheduledApp)
	if app == "" {
		return
	}
	fieldDefaultScheduledApps.Delete(app)
	b.fieldDefaultPlan.scheduledApp = ""
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
	if len(existingHand) > 0 {
		// Handwritten store already owns this application.
		return plan, nil
	}
	if len(local) > 0 {
		// Already present in this prebuild graph (handwritten or virtual).
		return plan, nil
	}
	if len(existingVirt) > 0 {
		// Virtual rows in DB are metadata only — sources are not on disk (D12).
		// Re-inject when this module owns the virtual path (rebuild / hot-reload / bundle).
		// Other modules sharing the application must not inject a second store path.
		if fieldDefaultSameModule(existingVirt, mod) {
			return FieldDefaultPlan{NeedInject: true, scheduledApp: app}, nil
		}
		return plan, nil
	}
	owner, loaded := fieldDefaultScheduledApps.LoadOrStore(app, mod.Name)
	if loaded {
		// Same module reclaim (rebuild/hot-reload): allow re-inject.
		if ownerName, ok := owner.(string); ok && ownerName == mod.Name {
			return FieldDefaultPlan{NeedInject: true, scheduledApp: app}, nil
		}
		return plan, nil
	}
	return FieldDefaultPlan{NeedInject: true, scheduledApp: app}, nil
}

func (b *ModuleBuilder) rememberFieldDefaultInjectPath(path string) {
	path = strings.TrimSpace(path)
	if b == nil || path == "" {
		return
	}
	b.fieldDefaultInjectPath = path
	for _, existing := range b.fieldDefaultInjectPaths {
		if existing == path {
			return
		}
	}
	b.fieldDefaultInjectPaths = append(b.fieldDefaultInjectPaths, path)
}

func (b *ModuleBuilder) applyFieldDefaultInject(plan FieldDefaultPlan) error {
	if b == nil || !plan.NeedInject || b.module == nil {
		return nil
	}
	if strings.TrimSpace(b.module.Path) == "" {
		return xfmt.Errorf("FieldDefault inject requires a non-empty module path")
	}
	path := fieldDefaultGeneratedPath(b.module.Path)
	b.rememberFieldDefaultInjectPath(path)

	imports := append(b.entryPointImports(), b.fieldDefaultInjectPaths...)
	if setter, ok := b.buildPlugin.(interface{ SetEntryPointImports([]string) }); ok {
		setter.SetEntryPointImports(imports)
	}
	if setter, ok := b.prebuildPlugin.(interface{ SetEntryPointImports([]string) }); ok {
		setter.SetEntryPointImports(imports)
	}
	modulesPath := strings.TrimSpace(b.resolvedRuntimeOptions().modulesPath)
	if modulesPath == "" {
		modulesPath = filepath.Dir(b.module.Path)
	}
	source := virtualFieldDefaultSource(modulesPath, b.module.ApplicationStr)
	if registrar, ok := b.buildPlugin.(virtualSourceRegistrar); ok {
		registrar.RegisterVirtualSource(path, source)
	}
	if registrar, ok := b.prebuildPlugin.(virtualSourceRegistrar); ok {
		registrar.RegisterVirtualSource(path, source)
	}
	return nil
}

// EnsureFieldDefaultVirtualImports registers C2 FieldDefault virtual sources for each
// distinct non-core application represented by modules and records those paths so
// buildOptions merges them into WithEntryPointImports (which replaces any prior
// plugin SetEntryPointImports). Used by the multi-app dist/bundles builder, which
// otherwise only Decide/Injects against a single representative module (often core).
//
// Handwritten FieldDefault ownership (same precedence as decideFieldDefaultPlan) skips
// C2 registration so the bundle does not load two stores for one application.
func (b *ModuleBuilder) EnsureFieldDefaultVirtualImports(modules []*meta.Module) error {
	if b == nil {
		return nil
	}
	modulesPath := strings.TrimSpace(b.resolvedRuntimeOptions().modulesPath)
	seenApp := make(map[string]struct{})
	for _, mod := range modules {
		if mod == nil {
			continue
		}
		app := strings.TrimSpace(mod.ApplicationStr)
		if app == "" || app == "core" || strings.TrimSpace(mod.ServiceEntryPoint) == "" {
			continue
		}
		if _, ok := seenApp[app]; ok {
			continue
		}
		if strings.TrimSpace(mod.Path) == "" {
			continue
		}
		seenApp[app] = struct{}{}

		existing, err := b.dbLoadFieldDefaults(app)
		if err != nil {
			return xfmt.Errorf("load FieldDefault models for application %q: %w", app, err)
		}
		if len(handwrittenFieldDefaults(existing)) > 0 {
			// Handwritten store already owns this application — do not inject C2.
			continue
		}

		path := fieldDefaultGeneratedPath(mod.Path)
		if virt := generatedFieldDefaults(existing); len(virt) > 0 {
			// Prefer the canonical meta path so rebuilds match Persist / sameModule checks.
			if p := strings.TrimSpace(virt[0].Path); p != "" {
				path = filepath.ToSlash(filepath.Clean(p))
			}
		}
		b.rememberFieldDefaultInjectPath(path)
		if modulesPath == "" {
			modulesPath = filepath.Dir(mod.Path)
		}
		source := virtualFieldDefaultSource(modulesPath, app)
		if registrar, ok := b.buildPlugin.(virtualSourceRegistrar); ok {
			registrar.RegisterVirtualSource(path, source)
		}
		if registrar, ok := b.prebuildPlugin.(virtualSourceRegistrar); ok {
			registrar.RegisterVirtualSource(path, source)
		}
	}
	if len(b.fieldDefaultInjectPaths) == 0 {
		return nil
	}
	imports := append(b.entryPointImports(), b.fieldDefaultInjectPaths...)
	if setter, ok := b.buildPlugin.(interface{ SetEntryPointImports([]string) }); ok {
		setter.SetEntryPointImports(imports)
	}
	if setter, ok := b.prebuildPlugin.(interface{ SetEntryPointImports([]string) }); ok {
		setter.SetEntryPointImports(imports)
	}
	return nil
}

func (b *ModuleBuilder) planAndInjectFieldDefault(prebuildResult *module.BuildResult) error {
	plan, err := b.decideFieldDefaultPlan(module.ParserResults(prebuildResult))
	if err != nil {
		return err
	}
	b.fieldDefaultPlan = plan
	if err := b.applyFieldDefaultInject(plan); err != nil {
		b.releaseFieldDefaultSchedule()
		return xfmt.Errorf("error injecting FieldDefault: %w", err)
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
	return xfmt.Errorf("%s: application %q build produced multiple FieldDefault models", fieldDefaultDuplicateCode, app)
}

func (b *ModuleBuilder) supersedeVirtualFieldDefaults() error {
	if b == nil || !b.fieldDefaultPlan.SupersedeVirtual || b.module == nil {
		return nil
	}
	if b.runtimeScope == nil || b.runtimeScope.Session() == nil || b.runtimeScope.Session().DB == nil {
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

	root := b.runtimeScope.Session().DB
	// Fresh Unscoped handle per statement — avoid GORM clause/table pollution.
	db := func() *gorm.DB { return root.Session(&gorm.Session{NewDB: true}).Unscoped() }

	// SQLite migrations disable FK constraints; delete dependents explicitly
	// (same order as module uninstaller). Pluck IDs first for stable IN lists.
	var serviceIDs []string
	if err := db().Model(&meta.Service{}).Where("model_id IN ?", ids).Pluck("id", &serviceIDs).Error; err != nil {
		return xfmt.Errorf("load superseded FieldDefault services: %w", err)
	}
	var fieldIDs []string
	if err := db().Model(&meta.Field{}).Where("model_id IN ?", ids).Pluck("id", &fieldIDs).Error; err != nil {
		return xfmt.Errorf("load superseded FieldDefault fields: %w", err)
	}
	decoratorQ := db().Model(&meta.Decorator{}).Where("model_id IN ?", ids)
	if len(serviceIDs) > 0 {
		decoratorQ = decoratorQ.Or("service_id IN ?", serviceIDs)
	}
	if len(fieldIDs) > 0 {
		decoratorQ = decoratorQ.Or("field_id IN ?", fieldIDs)
	}
	var decoratorIDs []string
	if err := decoratorQ.Pluck("id", &decoratorIDs).Error; err != nil {
		return xfmt.Errorf("load superseded FieldDefault decorators: %w", err)
	}

	if len(decoratorIDs) > 0 {
		if result := db().Where("decorator_id IN ?", decoratorIDs).Delete(&meta.Argument{}); result.Error != nil {
			return xfmt.Errorf("delete superseded FieldDefault decorator arguments: %w", result.Error)
		}
		if result := db().Where("id IN ?", decoratorIDs).Delete(&meta.Decorator{}); result.Error != nil {
			return xfmt.Errorf("delete superseded FieldDefault decorators: %w", result.Error)
		}
	}
	if len(serviceIDs) > 0 {
		if result := db().Where("service_id IN ?", serviceIDs).Delete(&meta.TypeParameter{}); result.Error != nil {
			return xfmt.Errorf("delete superseded FieldDefault type parameters: %w", result.Error)
		}
		if result := db().Where("service_id IN ?", serviceIDs).Delete(&meta.Parameter{}); result.Error != nil {
			return xfmt.Errorf("delete superseded FieldDefault parameters: %w", result.Error)
		}
		if result := db().Where("id IN ?", serviceIDs).Delete(&meta.Service{}); result.Error != nil {
			return xfmt.Errorf("delete superseded FieldDefault services: %w", result.Error)
		}
	}
	if len(fieldIDs) > 0 {
		if result := db().Where("id IN ?", fieldIDs).Delete(&meta.Field{}); result.Error != nil {
			return xfmt.Errorf("delete superseded FieldDefault fields: %w", result.Error)
		}
	}
	if result := db().Where("id IN ?", ids).Delete(&meta.Model{}); result.Error != nil {
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
