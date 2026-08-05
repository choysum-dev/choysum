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
	fieldDefaultGeneratedRelPath = "service/models/__generated__/field_default.ts"
	fieldDefaultModelName        = "FieldDefault"
	fieldDefaultDuplicateCode    = "FIELD_DEFAULT_DUPLICATE"
)

// fieldDefaultScheduledApps is the process-wide NeedInject dedup map (tests).
var fieldDefaultScheduledApps = injectappmodel.ScheduledApps("FieldDefault")

// FieldDefaultPlan is the Decide output for C2 FieldDefault inject (legacy test shape).
type FieldDefaultPlan struct {
	NeedInject       bool
	SupersedeVirtual bool // maps to injectappmodel.Plan.SupersedeInject
	scheduledApp     string
}

func fieldDefaultPlanFrom(p injectappmodel.Plan) FieldDefaultPlan {
	return FieldDefaultPlan{
		NeedInject:       p.NeedInject,
		SupersedeVirtual: p.SupersedeInject,
		scheduledApp:     p.ScheduledApp,
	}
}

func (p FieldDefaultPlan) toInject() injectappmodel.Plan {
	return injectappmodel.Plan{
		NeedInject:      p.NeedInject,
		SupersedeInject: p.SupersedeVirtual,
		ScheduledApp:    p.scheduledApp,
	}
}

func (b *ModuleBuilder) syncFieldDefaultFromSession() {
	if b == nil || b.injectSession == nil {
		return
	}
	b.fieldDefaultPlan = fieldDefaultPlanFrom(b.injectSession.Plan("FieldDefault"))
	b.fieldDefaultInjectPaths = b.injectSession.InjectPaths("FieldDefault")
	b.fieldDefaultInjectPath = b.injectSession.LastInjectPath("FieldDefault")
}

func (b *ModuleBuilder) rememberFieldDefaultInjectPath(path string) {
	path = strings.TrimSpace(path)
	if b == nil || path == "" {
		return
	}
	sess := b.ensureInjectSession()
	sess.RememberPathForTest("FieldDefault", path)
	b.syncFieldDefaultFromSession()
}

func (b *ModuleBuilder) releaseFieldDefaultSchedule() {
	if b == nil {
		return
	}
	sess := b.ensureInjectSession()
	plan := sess.Plan("FieldDefault")
	app := strings.TrimSpace(plan.ScheduledApp)
	if app != "" {
		injectappmodel.ReleaseSchedule("FieldDefault", app)
	}
	plan.ScheduledApp = ""
	sess.SetPlan("FieldDefault", plan)
	b.fieldDefaultPlan = fieldDefaultPlanFrom(plan)
}

func virtualFieldDefaultSource(modulesPath string, application string) string {
	return injectappmodel.GeneratedSourceForTest("FieldDefault", modulesPath, application)
}

func isGeneratedFieldDefaultPath(path string) bool {
	return injectappmodel.IsGeneratedPathForTest("FieldDefault", path)
}

func fieldDefaultGeneratedPath(modulePath string) string {
	return injectappmodel.GeneratedPathForTest("FieldDefault", modulePath)
}

func fieldDefaultsIn(results []*parser.ParserResult, modulePath string) []*meta.Model {
	return injectappmodel.ModelsInForTest("FieldDefault", results, modulePath)
}

func handwrittenFieldDefaults(models []*meta.Model) []*meta.Model {
	return injectappmodel.HandwrittenForTest("FieldDefault", models)
}

func generatedFieldDefaults(models []*meta.Model) []*meta.Model {
	return injectappmodel.GeneratedForTest("FieldDefault", models)
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
	absFalse := false
	return meta.ListDeclarations(b.runtimeScope.Session().DB, meta.DeclarationQuery{
		Application: app,
		Name:        fieldDefaultModelName,
		Abstract:    &absFalse,
	})
}

func (b *ModuleBuilder) decideFieldDefaultPlan(prebuildResults []*parser.ParserResult) (FieldDefaultPlan, error) {
	if b == nil {
		return FieldDefaultPlan{}, nil
	}
	sess := b.ensureInjectSession()
	plan, err := injectappmodel.DecideOne(sess, "FieldDefault", prebuildResults)
	if err != nil {
		return FieldDefaultPlan{}, err
	}
	out := fieldDefaultPlanFrom(plan)
	b.fieldDefaultPlan = out
	return out, nil
}

func (b *ModuleBuilder) applyFieldDefaultInject(plan FieldDefaultPlan) error {
	sess := b.ensureInjectSession()
	sess.SetPlan("FieldDefault", plan.toInject())
	if err := injectappmodel.ApplyInjectOne(sess, "FieldDefault"); err != nil {
		return err
	}
	b.syncFieldDefaultFromSession()
	return nil
}

// EnsureFieldDefaultVirtualImports registers C2 FieldDefault inject sources for each
// distinct non-core application (legacy name; prefer BundleInjectAppModels).
func (b *ModuleBuilder) EnsureFieldDefaultVirtualImports(modules []*meta.Module) error {
	if b == nil {
		return nil
	}
	sess := b.ensureInjectSession()
	sess.ClearInjectPaths("FieldDefault")
	if err := injectappmodel.BundleOne(sess, "FieldDefault", modules); err != nil {
		return err
	}
	b.syncFieldDefaultFromSession()
	if len(b.fieldDefaultInjectPaths) == 0 {
		return nil
	}
	imports := append(b.entryPointImports(), b.fieldDefaultInjectPaths...)
	injectHost{b}.SetEntryPointImports(imports)
	return nil
}

func (b *ModuleBuilder) planAndInjectFieldDefault(prebuildResult *module.BuildResult) error {
	sess := b.ensureInjectSession()
	if err := injectappmodel.DecideAndInjectOne(sess, "FieldDefault", module.ParserResults(prebuildResult)); err != nil {
		b.releaseFieldDefaultSchedule()
		return err
	}
	b.syncFieldDefaultFromSession()
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
	return fmt.Errorf("FIELD_DEFAULT_DUPLICATE: application %q build produced multiple FieldDefault models", app)
}

func (b *ModuleBuilder) supersedeVirtualFieldDefaults() error {
	if b == nil {
		return nil
	}
	sess := b.ensureInjectSession()
	sess.SetPlan("FieldDefault", b.fieldDefaultPlan.toInject())
	return injectappmodel.SupersedeOne(sess, "FieldDefault")
}

func resetFieldDefaultScheduledAppsForTest() {
	injectappmodel.ResetScheduledForTest()
}
