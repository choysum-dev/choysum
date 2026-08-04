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

// AppSettingPlan is the Decide output for C2 virtual AppSetting inject.
type AppSettingPlan struct {
	NeedInject       bool
	SupersedeVirtual bool
	// scheduledApp is set when this builder claimed the process-wide NeedInject slot.
	// Cleared via releaseAppSettingSchedule on failure or after Persist/Bundle.
	scheduledApp string
}

const (
	appSettingGeneratedRelPath = "service/models/__generated__/app_setting.ts"
	appSettingModelName        = "AppSetting"
	appSettingDuplicateCode    = "APP_SETTING_DUPLICATE"
)

// appSettingScheduledApps dedupes NeedInject across modules sharing one Application
// within a single process (install / upgrade). Entries must be released when the
// claiming build/persist does not complete successfully.
var appSettingScheduledApps sync.Map

// virtualAppSettingSource builds C2 thin-class source with absolute imports so
// esbuild can resolve them even when the pseudo __generated__ directory is not on disk.
//
// softDelete: false is required so Set(key, null) hard-deletes and unique keys can be reused.
// application must be baked into @Model options (same reason as FieldDefault).
func virtualAppSettingSource(modulesPath string, application string) string {
	modulesPath = filepath.ToSlash(filepath.Clean(strings.TrimSpace(modulesPath)))
	application = strings.TrimSpace(application)
	if application == "" {
		application = "application"
	}
	coreService := filepath.ToSlash(filepath.Join(modulesPath, "core/service/index.ts"))
	baseModel := filepath.ToSlash(filepath.Join(modulesPath, "core/service/orm/model/app_setting_base_model.ts"))
	return fmt.Sprintf(`import { Model } from %s
import AppSettingBaseModel from %s

@Model('AppSetting', { application: %s, softDelete: false })
export default class AppSetting extends AppSettingBaseModel {}
`, strconv.Quote(coreService), strconv.Quote(baseModel), strconv.Quote(application))
}

func isGeneratedAppSettingPath(path string) bool {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return false
	}
	normalized := filepath.ToSlash(filepath.Clean(trimmed))
	return strings.HasSuffix(normalized, "/"+appSettingGeneratedRelPath) ||
		normalized == appSettingGeneratedRelPath
}

func appSettingGeneratedPath(modulePath string) string {
	return filepath.ToSlash(filepath.Clean(filepath.Join(strings.TrimSpace(modulePath), appSettingGeneratedRelPath)))
}

func appSettingsIn(results []*parser.ParserResult, modulePath string) []*meta.Model {
	out := make([]*meta.Model, 0)
	for _, result := range results {
		if result == nil || result.Model == nil {
			continue
		}
		m := result.Model
		if m.Name != appSettingModelName || m.Abstract {
			continue
		}
		if modulePath != "" && !pathWithinModuleRoot(result.Path, modulePath) && !pathWithinModuleRoot(m.Path, modulePath) {
			continue
		}
		out = append(out, m)
	}
	return out
}

func handwrittenAppSettings(models []*meta.Model) []*meta.Model {
	out := make([]*meta.Model, 0, len(models))
	for _, m := range models {
		if m == nil || isGeneratedAppSettingPath(m.Path) {
			continue
		}
		out = append(out, m)
	}
	return out
}

func generatedAppSettings(models []*meta.Model) []*meta.Model {
	out := make([]*meta.Model, 0, len(models))
	for _, m := range models {
		if m == nil || !isGeneratedAppSettingPath(m.Path) {
			continue
		}
		out = append(out, m)
	}
	return out
}

func appSettingSameModule(existing []*meta.Model, mod *meta.Module) bool {
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

func (b *ModuleBuilder) dbLoadAppSettings(app string) ([]*meta.Model, error) {
	app = strings.TrimSpace(app)
	if app == "" || b == nil || b.runtimeScope == nil || b.runtimeScope.Session() == nil {
		return nil, nil
	}
	var raws []*meta.RawModel
	err := b.runtimeScope.Session().
		Where("application = ? AND name = ? AND abstract = ?", app, appSettingModelName, false).
		Find(&raws).Error
	if err != nil {
		return nil, err
	}
	return meta.RawModelsAsModels(raws), nil
}

func (b *ModuleBuilder) releaseAppSettingSchedule() {
	if b == nil {
		return
	}
	app := strings.TrimSpace(b.appSettingPlan.scheduledApp)
	if app == "" {
		return
	}
	appSettingScheduledApps.Delete(app)
	b.appSettingPlan.scheduledApp = ""
}

func (b *ModuleBuilder) decideAppSettingPlan(prebuildResults []*parser.ParserResult) (AppSettingPlan, error) {
	plan := AppSettingPlan{}
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
	local := appSettingsIn(prebuildResults, mod.Path)
	localHand := handwrittenAppSettings(local)
	if len(localHand) > 1 {
		return plan, xfmt.Errorf("%s: application %q has multiple handwritten AppSetting models in module %q", appSettingDuplicateCode, app, mod.Name)
	}

	existing, err := b.dbLoadAppSettings(app)
	if err != nil {
		return plan, xfmt.Errorf("load AppSetting models for application %q: %w", app, err)
	}
	existingVirt := generatedAppSettings(existing)
	existingHand := handwrittenAppSettings(existing)

	if len(localHand) > 0 {
		if len(existingHand) > 0 && !appSettingSameModule(existingHand, mod) {
			return plan, xfmt.Errorf("%s: application %q already has a handwritten AppSetting outside module %q", appSettingDuplicateCode, app, mod.Name)
		}
		if len(existingVirt) > 0 {
			return AppSettingPlan{SupersedeVirtual: true}, nil
		}
		return plan, nil
	}

	// No local handwritten AppSetting — consider C2 inject.
	if len(existingHand) > 0 {
		return plan, nil
	}
	if len(local) > 0 {
		return plan, nil
	}
	if len(existingVirt) > 0 {
		// Virtual rows in DB are metadata only — sources are not on disk (D12).
		// Re-inject when this module owns the virtual path (rebuild / hot-reload / bundle).
		// Other modules sharing the application must not inject a second store path.
		if appSettingSameModule(existingVirt, mod) {
			owner, loaded := appSettingScheduledApps.LoadOrStore(app, mod.Name)
			if loaded {
				if ownerName, ok := owner.(string); ok && ownerName != mod.Name {
					// Another in-process builder still holds the claim; re-inject
					// without adopting release ownership (avoid deleting their key).
					return AppSettingPlan{NeedInject: true}, nil
				}
			}
			return AppSettingPlan{NeedInject: true, scheduledApp: app}, nil
		}
		return plan, nil
	}
	owner, loaded := appSettingScheduledApps.LoadOrStore(app, mod.Name)
	if loaded {
		if ownerName, ok := owner.(string); ok && ownerName == mod.Name {
			return AppSettingPlan{NeedInject: true, scheduledApp: app}, nil
		}
		return plan, nil
	}
	return AppSettingPlan{NeedInject: true, scheduledApp: app}, nil
}

func (b *ModuleBuilder) rememberAppSettingInjectPath(path string) {
	path = strings.TrimSpace(path)
	if b == nil || path == "" {
		return
	}
	b.appSettingInjectPath = path
	for _, existing := range b.appSettingInjectPaths {
		if existing == path {
			return
		}
	}
	b.appSettingInjectPaths = append(b.appSettingInjectPaths, path)
}

func (b *ModuleBuilder) applyAppSettingInject(plan AppSettingPlan) error {
	if b == nil || !plan.NeedInject || b.module == nil {
		return nil
	}
	if strings.TrimSpace(b.module.Path) == "" {
		return xfmt.Errorf("AppSetting inject requires a non-empty module path")
	}
	path := appSettingGeneratedPath(b.module.Path)
	b.rememberAppSettingInjectPath(path)

	imports := append(b.entryPointImports(), b.appSettingInjectPaths...)
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
	source := virtualAppSettingSource(modulesPath, b.module.ApplicationStr)
	if registrar, ok := b.buildPlugin.(virtualSourceRegistrar); ok {
		registrar.RegisterVirtualSource(path, source)
	}
	if registrar, ok := b.prebuildPlugin.(virtualSourceRegistrar); ok {
		registrar.RegisterVirtualSource(path, source)
	}
	return nil
}

// EnsureAppSettingVirtualImports registers C2 AppSetting virtual sources for each
// distinct non-core application represented by modules and records those paths so
// buildOptions merges them into WithEntryPointImports. Used by the multi-app
// dist/bundles builder (parallel to EnsureFieldDefaultVirtualImports).
func (b *ModuleBuilder) EnsureAppSettingVirtualImports(modules []*meta.Module) error {
	if b == nil {
		return nil
	}
	modulesPathCfg := strings.TrimSpace(b.resolvedRuntimeOptions().modulesPath)
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

		existing, err := b.dbLoadAppSettings(app)
		if err != nil {
			return xfmt.Errorf("load AppSetting models for application %q: %w", app, err)
		}
		if len(handwrittenAppSettings(existing)) > 0 {
			continue
		}

		path := appSettingGeneratedPath(mod.Path)
		if virt := generatedAppSettings(existing); len(virt) > 0 {
			if p := strings.TrimSpace(virt[0].Path); p != "" {
				path = filepath.ToSlash(filepath.Clean(p))
			}
		}
		b.rememberAppSettingInjectPath(path)
		modulesPath := modulesPathCfg
		if modulesPath == "" {
			modulesPath = filepath.Dir(mod.Path)
		}
		source := virtualAppSettingSource(modulesPath, app)
		if registrar, ok := b.buildPlugin.(virtualSourceRegistrar); ok {
			registrar.RegisterVirtualSource(path, source)
		}
		if registrar, ok := b.prebuildPlugin.(virtualSourceRegistrar); ok {
			registrar.RegisterVirtualSource(path, source)
		}
	}
	if len(b.appSettingInjectPaths) == 0 {
		return nil
	}
	imports := append(b.entryPointImports(), b.appSettingInjectPaths...)
	if setter, ok := b.buildPlugin.(interface{ SetEntryPointImports([]string) }); ok {
		setter.SetEntryPointImports(imports)
	}
	if setter, ok := b.prebuildPlugin.(interface{ SetEntryPointImports([]string) }); ok {
		setter.SetEntryPointImports(imports)
	}
	return nil
}

func (b *ModuleBuilder) planAndInjectAppSetting(prebuildResult *module.BuildResult) error {
	plan, err := b.decideAppSettingPlan(module.ParserResults(prebuildResult))
	if err != nil {
		return err
	}
	b.appSettingPlan = plan
	if err := b.applyAppSettingInject(plan); err != nil {
		b.releaseAppSettingSchedule()
		return xfmt.Errorf("error injecting AppSetting: %w", err)
	}
	return nil
}

func (b *ModuleBuilder) validateAppSetting(buildResult *module.BuildResult) error {
	if buildResult == nil || b == nil || b.module == nil {
		return nil
	}
	app := strings.TrimSpace(b.module.ApplicationStr)
	if app == "" || app == "core" {
		return nil
	}

	models := appSettingsIn(module.ParserResults(buildResult), b.module.Path)
	if len(models) <= 1 {
		return nil
	}
	return xfmt.Errorf("%s: application %q build produced multiple AppSetting models", appSettingDuplicateCode, app)
}

func (b *ModuleBuilder) supersedeVirtualAppSettings() error {
	if b == nil || !b.appSettingPlan.SupersedeVirtual || b.module == nil {
		return nil
	}
	if b.runtimeScope == nil || b.runtimeScope.Session() == nil || b.runtimeScope.Session().DB == nil {
		return nil
	}
	app := strings.TrimSpace(b.module.ApplicationStr)
	if app == "" {
		return nil
	}

	var existing []*meta.RawModel
	if err := b.runtimeScope.Session().
		Where("application = ? AND name = ? AND abstract = ?", app, appSettingModelName, false).
		Find(&existing).Error; err != nil {
		return xfmt.Errorf("load AppSetting rows for supersede: %w", err)
	}

	ids := make([]string, 0)
	for _, m := range existing {
		if m == nil || !isGeneratedAppSettingPath(m.Path) {
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
	db := func() *gorm.DB { return root.Session(&gorm.Session{NewDB: true}).Unscoped() }

	var serviceIDs []string
	if err := db().Model(&meta.RawService{}).Where("model_id IN ?", ids).Pluck("id", &serviceIDs).Error; err != nil {
		return xfmt.Errorf("load superseded AppSetting services: %w", err)
	}
	var fieldIDs []string
	if err := db().Model(&meta.RawField{}).Where("model_id IN ?", ids).Pluck("id", &fieldIDs).Error; err != nil {
		return xfmt.Errorf("load superseded AppSetting fields: %w", err)
	}
	decoratorQ := db().Model(&meta.RawDecorator{}).Where("model_id IN ?", ids)
	if len(serviceIDs) > 0 {
		decoratorQ = decoratorQ.Or("service_id IN ?", serviceIDs)
	}
	if len(fieldIDs) > 0 {
		decoratorQ = decoratorQ.Or("field_id IN ?", fieldIDs)
	}
	var decoratorIDs []string
	if err := decoratorQ.Pluck("id", &decoratorIDs).Error; err != nil {
		return xfmt.Errorf("load superseded AppSetting decorators: %w", err)
	}

	if len(decoratorIDs) > 0 {
		if result := db().Where("decorator_id IN ?", decoratorIDs).Delete(&meta.RawArgument{}); result.Error != nil {
			return xfmt.Errorf("delete superseded AppSetting decorator arguments: %w", result.Error)
		}
		if result := db().Where("id IN ?", decoratorIDs).Delete(&meta.RawDecorator{}); result.Error != nil {
			return xfmt.Errorf("delete superseded AppSetting decorators: %w", result.Error)
		}
	}
	if len(serviceIDs) > 0 {
		if result := db().Where("service_id IN ?", serviceIDs).Delete(&meta.RawTypeParameter{}); result.Error != nil {
			return xfmt.Errorf("delete superseded AppSetting type parameters: %w", result.Error)
		}
		if result := db().Where("service_id IN ?", serviceIDs).Delete(&meta.RawParameter{}); result.Error != nil {
			return xfmt.Errorf("delete superseded AppSetting parameters: %w", result.Error)
		}
		if result := db().Where("id IN ?", serviceIDs).Delete(&meta.RawService{}); result.Error != nil {
			return xfmt.Errorf("delete superseded AppSetting services: %w", result.Error)
		}
	}
	if len(fieldIDs) > 0 {
		if result := db().Where("id IN ?", fieldIDs).Delete(&meta.RawField{}); result.Error != nil {
			return xfmt.Errorf("delete superseded AppSetting fields: %w", result.Error)
		}
	}
	if result := db().Where("id IN ?", ids).Delete(&meta.RawModel{}); result.Error != nil {
		return xfmt.Errorf("delete superseded virtual AppSetting rows: %w", result.Error)
	}
	if err := meta.RecomputeKeys(root, []meta.LogicalKey{{Application: app, Name: appSettingModelName}}); err != nil {
		return xfmt.Errorf("recompute AppSetting after supersede: %w", err)
	}
	return nil
}

// resetAppSettingScheduledAppsForTest clears the process-wide inject dedup map (tests only).
func resetAppSettingScheduledAppsForTest() {
	appSettingScheduledApps.Range(func(key, _ any) bool {
		appSettingScheduledApps.Delete(key)
		return true
	})
}
