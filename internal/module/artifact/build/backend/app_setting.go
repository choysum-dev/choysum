// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package backendbuilder

import (
	"fmt"
	"strings"

	"github.com/choysum-dev/choysum/internal/module/artifact/build/injectappmodel"
	module "github.com/choysum-dev/choysum/internal/module/artifact/result"
	"github.com/choysum-dev/choysum/internal/parser"
	"github.com/choysum-dev/choysum/pkg/meta"
)

const (
	appSettingGeneratedRelPath = "service/models/__generated__/app_setting.ts"
	appSettingModelName        = "AppSetting"
	appSettingDuplicateCode    = "APP_SETTING_DUPLICATE"
)

// appSettingScheduledApps is the process-wide NeedInject dedup map (tests).
var appSettingScheduledApps = injectappmodel.ScheduledApps("AppSetting")

// AppSettingPlan is the Decide output for C2 AppSetting inject (legacy test shape).
type AppSettingPlan struct {
	NeedInject       bool
	SupersedeVirtual bool // maps to injectappmodel.Plan.SupersedeInject
	scheduledApp     string
}

func appSettingPlanFrom(p injectappmodel.Plan) AppSettingPlan {
	return AppSettingPlan{
		NeedInject:       p.NeedInject,
		SupersedeVirtual: p.SupersedeInject,
		scheduledApp:     p.ScheduledApp,
	}
}

func (p AppSettingPlan) toInject() injectappmodel.Plan {
	return injectappmodel.Plan{
		NeedInject:      p.NeedInject,
		SupersedeInject: p.SupersedeVirtual,
		ScheduledApp:    p.scheduledApp,
	}
}

func (b *ModuleBuilder) syncAppSettingFromSession() {
	if b == nil || b.injectSession == nil {
		return
	}
	b.appSettingPlan = appSettingPlanFrom(b.injectSession.Plan("AppSetting"))
	b.appSettingInjectPaths = b.injectSession.InjectPaths("AppSetting")
	b.appSettingInjectPath = b.injectSession.LastInjectPath("AppSetting")
}

func (b *ModuleBuilder) rememberAppSettingInjectPath(path string) {
	path = strings.TrimSpace(path)
	if b == nil || path == "" {
		return
	}
	sess := b.ensureInjectSession()
	sess.RememberPathForTest("AppSetting", path)
	b.syncAppSettingFromSession()
}

func (b *ModuleBuilder) releaseAppSettingSchedule() {
	if b == nil {
		return
	}
	sess := b.ensureInjectSession()
	plan := sess.Plan("AppSetting")
	app := strings.TrimSpace(plan.ScheduledApp)
	if app != "" {
		injectappmodel.ReleaseSchedule("AppSetting", app)
	}
	plan.ScheduledApp = ""
	sess.SetPlan("AppSetting", plan)
	b.appSettingPlan = appSettingPlanFrom(plan)
}

func virtualAppSettingSource(modulesPath string, application string) string {
	return injectappmodel.GeneratedSourceForTest("AppSetting", modulesPath, application)
}

func isGeneratedAppSettingPath(path string) bool {
	return injectappmodel.IsGeneratedPathForTest("AppSetting", path)
}

func appSettingGeneratedPath(modulePath string) string {
	return injectappmodel.GeneratedPathForTest("AppSetting", modulePath)
}

func appSettingsIn(results []*parser.ParserResult, modulePath string) []*meta.Model {
	return injectappmodel.ModelsInForTest("AppSetting", results, modulePath)
}

func handwrittenAppSettings(models []*meta.Model) []*meta.Model {
	return injectappmodel.HandwrittenForTest("AppSetting", models)
}

func generatedAppSettings(models []*meta.Model) []*meta.Model {
	return injectappmodel.GeneratedForTest("AppSetting", models)
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
	absFalse := false
	return meta.ListDeclarations(b.runtimeScope.Session().DB, meta.DeclarationQuery{
		Application: app,
		Name:        appSettingModelName,
		Abstract:    &absFalse,
	})
}

func (b *ModuleBuilder) decideAppSettingPlan(prebuildResults []*parser.ParserResult) (AppSettingPlan, error) {
	if b == nil {
		return AppSettingPlan{}, nil
	}
	sess := b.ensureInjectSession()
	plan, err := injectappmodel.DecideOne(sess, "AppSetting", prebuildResults)
	if err != nil {
		return AppSettingPlan{}, err
	}
	out := appSettingPlanFrom(plan)
	b.appSettingPlan = out
	return out, nil
}

func (b *ModuleBuilder) applyAppSettingInject(plan AppSettingPlan) error {
	sess := b.ensureInjectSession()
	sess.SetPlan("AppSetting", plan.toInject())
	if err := injectappmodel.ApplyInjectOne(sess, "AppSetting"); err != nil {
		return err
	}
	b.syncAppSettingFromSession()
	return nil
}

// EnsureAppSettingVirtualImports registers C2 AppSetting inject sources for each
// distinct non-core application (legacy name; prefer BundleInjectAppModels).
func (b *ModuleBuilder) EnsureAppSettingVirtualImports(modules []*meta.Module) error {
	if b == nil {
		return nil
	}
	sess := b.ensureInjectSession()
	sess.ClearInjectPaths("AppSetting")
	if err := injectappmodel.BundleOne(sess, "AppSetting", modules); err != nil {
		return err
	}
	b.syncAppSettingFromSession()
	if len(b.appSettingInjectPaths) == 0 {
		return nil
	}
	imports := append(b.entryPointImports(), b.appSettingInjectPaths...)
	injectHost{b}.SetEntryPointImports(imports)
	return nil
}

func (b *ModuleBuilder) planAndInjectAppSetting(prebuildResult *module.BuildResult) error {
	sess := b.ensureInjectSession()
	if err := injectappmodel.DecideAndInjectOne(sess, "AppSetting", module.ParserResults(prebuildResult)); err != nil {
		b.releaseAppSettingSchedule()
		return err
	}
	b.syncAppSettingFromSession()
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
	return fmt.Errorf("APP_SETTING_DUPLICATE: application %q build produced multiple AppSetting models", app)
}

func (b *ModuleBuilder) supersedeVirtualAppSettings() error {
	if b == nil {
		return nil
	}
	sess := b.ensureInjectSession()
	sess.SetPlan("AppSetting", b.appSettingPlan.toInject())
	return injectappmodel.SupersedeOne(sess, "AppSetting")
}

func resetAppSettingScheduledAppsForTest() {
	injectappmodel.ResetScheduledForTest()
}
