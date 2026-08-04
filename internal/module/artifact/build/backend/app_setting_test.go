// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package backendbuilder

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	module "github.com/choysum-dev/choysum/internal/module/artifact/result"
	"github.com/choysum-dev/choysum/internal/parser"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// appSettingTestDBSeq keeps shared-cache in-memory DSNs unique across -count=N reruns.
var appSettingTestDBSeq atomic.Uint64

func appSettingMemoryDSN(t *testing.T, prefix string) string {
	t.Helper()
	name := strings.ReplaceAll(t.Name(), "/", "_")
	return fmt.Sprintf("file:%s-%s-%d?mode=memory&cache=shared", prefix, name, appSettingTestDBSeq.Add(1))
}

func newAppSettingTestBuilder(t *testing.T, mod *meta.Module) (*ModuleBuilder, *gorm.DB) {
	t.Helper()
	resetAppSettingScheduledAppsForTest()
	db, err := gorm.Open(sqlite.Open(appSettingMemoryDSN(t, "app-setting")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&meta.Application{}, &meta.Module{}, &meta.Model{},
		&meta.Field{}, &meta.Service{}, &meta.Decorator{}, &meta.Argument{},
		&meta.Parameter{}, &meta.TypeParameter{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	testScope := newBuilderTestScope()
	testScope.session = &scope.Session{DB: db}
	buildPlugin := &stubEsbPlugin{name: "build"}
	return &ModuleBuilder{
		runtimeScope: testScope,
		module:       mod,
		buildPlugin:  buildPlugin,
		entryPoint:   "/virtual/entry.ts",
	}, db
}

func TestIsGeneratedAppSettingPath(t *testing.T) {
	if !isGeneratedAppSettingPath("/mods/partner/service/models/__generated__/app_setting.ts") {
		t.Fatal("expected generated path match")
	}
	if isGeneratedAppSettingPath("/mods/partner/service/models/app_setting.ts") {
		t.Fatal("expected handwritten path mismatch")
	}
}

func TestDecideAppSettingPlan_NeedInject(t *testing.T) {
	mod := &meta.Module{
		Name:              "partner",
		Path:              "/virtual/modules/partner",
		ApplicationStr:    "partner",
		ServiceEntryPoint: "service/index.ts",
	}
	builder, _ := newAppSettingTestBuilder(t, mod)
	plan, err := builder.decideAppSettingPlan(nil)
	if err != nil {
		t.Fatalf("decide: %v", err)
	}
	if !plan.NeedInject || plan.SupersedeVirtual {
		t.Fatalf("expected NeedInject only, got %+v", plan)
	}
}

func TestDecideAppSettingPlan_SkipsCoreAndEmptyService(t *testing.T) {
	builder, _ := newAppSettingTestBuilder(t, &meta.Module{
		Name: "core", Path: "/virtual/modules/core", ApplicationStr: "core", ServiceEntryPoint: "service/index.ts",
	})
	plan, err := builder.decideAppSettingPlan(nil)
	if err != nil || plan.NeedInject {
		t.Fatalf("core should skip, got plan=%+v err=%v", plan, err)
	}

	builder.module = &meta.Module{Name: "web", Path: "/virtual/modules/web", ApplicationStr: "web"}
	plan, err = builder.decideAppSettingPlan(nil)
	if err != nil || plan.NeedInject {
		t.Fatalf("empty ServiceEntryPoint should skip, got plan=%+v err=%v", plan, err)
	}
}

func TestDecideAppSettingPlan_DBVirtualReinjectsForOwner(t *testing.T) {
	owner := &meta.Module{
		Name: "partner", Path: "/virtual/modules/partner",
		ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
	}
	builder, db := newAppSettingTestBuilder(t, owner)
	if err := db.Create(&meta.Model{
		BaseModel:   meta.BaseModel{Id: sql.NullString{String: "fd1", Valid: true}},
		Name:        "AppSetting",
		Path:        "/virtual/modules/partner/service/models/__generated__/app_setting.ts",
		Application: "partner",
	}).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}
	plan, err := builder.decideAppSettingPlan(nil)
	if err != nil {
		t.Fatalf("decide: %v", err)
	}
	if !plan.NeedInject || plan.SupersedeVirtual {
		t.Fatalf("expected NeedInject re-inject for owning module, got %+v", plan)
	}

	// Sibling module sharing the application must not inject a second store path.
	builder.module = &meta.Module{
		Name: "partner_bank", Path: "/virtual/modules/partner_bank",
		ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
	}
	plan, err = builder.decideAppSettingPlan(nil)
	if err != nil {
		t.Fatalf("decide sibling: %v", err)
	}
	if plan.NeedInject || plan.SupersedeVirtual {
		t.Fatalf("expected sibling skip when owner virtual exists, got %+v", plan)
	}
}

func TestDecideAppSettingPlan_DBHandwrittenSkipsInject(t *testing.T) {
	mod := &meta.Module{
		Name: "partner_bank", Path: "/virtual/modules/partner_bank",
		ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
	}
	builder, db := newAppSettingTestBuilder(t, mod)
	if err := db.Create(&meta.Model{
		BaseModel:   meta.BaseModel{Id: sql.NullString{String: "hand", Valid: true}},
		Name:        "AppSetting",
		Path:        "/virtual/modules/partner/service/models/app_setting.ts",
		Application: "partner",
	}).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}
	plan, err := builder.decideAppSettingPlan(nil)
	if err != nil {
		t.Fatalf("decide: %v", err)
	}
	if plan.NeedInject || plan.SupersedeVirtual {
		t.Fatalf("expected skip when DB has handwritten AppSetting, got %+v", plan)
	}
}

func TestDecideAppSettingPlan_SupersedeVirtual(t *testing.T) {
	mod := &meta.Module{
		Name: "partner_bank", Path: "/virtual/modules/partner_bank",
		ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
	}
	builder, db := newAppSettingTestBuilder(t, mod)
	if err := db.Create(&meta.Model{
		BaseModel:   meta.BaseModel{Id: sql.NullString{String: "virt", Valid: true}},
		Name:        "AppSetting",
		Path:        "/virtual/modules/partner/service/models/__generated__/app_setting.ts",
		Application: "partner",
		ModuleId:    sql.NullString{String: "mod-partner", Valid: true},
	}).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}
	handPath := filepath.Join(mod.Path, "service/models/app_setting.ts")
	prebuild := []*parser.ParserResult{{
		Path:  handPath,
		Model: &meta.Model{Name: "AppSetting", Path: handPath, Application: "partner"},
	}}
	plan, err := builder.decideAppSettingPlan(prebuild)
	if err != nil {
		t.Fatalf("decide: %v", err)
	}
	if !plan.SupersedeVirtual || plan.NeedInject {
		t.Fatalf("expected SupersedeVirtual, got %+v", plan)
	}
}

func TestDecideAppSettingPlan_DuplicateHandwritten(t *testing.T) {
	mod := &meta.Module{
		Name: "partner_commercial", Path: "/virtual/modules/partner_commercial",
		ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
	}
	builder, db := newAppSettingTestBuilder(t, mod)
	if err := db.Create(&meta.Model{
		BaseModel:   meta.BaseModel{Id: sql.NullString{String: "hand", Valid: true}},
		Name:        "AppSetting",
		Path:        "/virtual/modules/partner_bank/service/models/app_setting.ts",
		Application: "partner",
		ModuleId:    sql.NullString{String: "mod-bank", Valid: true},
	}).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}
	handPath := filepath.Join(mod.Path, "service/models/app_setting.ts")
	prebuild := []*parser.ParserResult{{
		Path:  handPath,
		Model: &meta.Model{Name: "AppSetting", Path: handPath, Application: "partner"},
	}}
	_, err := builder.decideAppSettingPlan(prebuild)
	if err == nil || !strings.Contains(err.Error(), appSettingDuplicateCode) {
		t.Fatalf("expected APP_SETTING_DUPLICATE, got %v", err)
	}
}

func TestDecideAppSettingPlan_ProcessDedupOneInject(t *testing.T) {
	resetAppSettingScheduledAppsForTest()
	modA := &meta.Module{
		Name: "partner", Path: "/virtual/modules/partner",
		ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
	}
	modB := &meta.Module{
		Name: "partner_bank", Path: "/virtual/modules/partner_bank",
		ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
	}
	builderA, _ := newAppSettingTestBuilder(t, modA)
	planA, err := builderA.decideAppSettingPlan(nil)
	if err != nil || !planA.NeedInject {
		t.Fatalf("first module should inject, plan=%+v err=%v", planA, err)
	}
	builderB := &ModuleBuilder{runtimeScope: builderA.runtimeScope, module: modB, buildPlugin: &stubEsbPlugin{name: "build"}}
	planB, err := builderB.decideAppSettingPlan(nil)
	if err != nil {
		t.Fatalf("second decide: %v", err)
	}
	if planB.NeedInject {
		t.Fatalf("second module same app must not inject again, got %+v", planB)
	}
}

func TestApplyAppSettingInject_SetsEntryImportAndPath(t *testing.T) {
	mod := &meta.Module{
		Name: "partner", Path: "/virtual/modules/partner",
		ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
	}
	builder, _ := newAppSettingTestBuilder(t, mod)
	stub := builder.buildPlugin.(*stubEsbPlugin)
	if err := builder.applyAppSettingInject(AppSettingPlan{NeedInject: true}); err != nil {
		t.Fatalf("inject: %v", err)
	}
	want := appSettingGeneratedPath(mod.Path)
	if builder.appSettingInjectPath != want {
		t.Fatalf("inject path = %q, want %q", builder.appSettingInjectPath, want)
	}
	found := false
	for _, imp := range stub.entryImports {
		if imp == want {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected entry import %q in %#v", want, stub.entryImports)
	}
	src, ok := stub.virtualSources[want]
	if !ok || src == "" {
		t.Fatalf("expected virtual source registered for %q, got %#v", want, stub.virtualSources)
	}
	if !strings.Contains(src, `@Model('AppSetting', { application: "partner", softDelete: false })`) {
		t.Fatalf("unexpected virtual source: %s", src)
	}
	if !strings.Contains(src, "AppSettingBaseModel") {
		t.Fatalf("expected AppSettingBaseModel import, got: %s", src)
	}
}

func TestVirtualAppSettingSource_QuotesLiterals(t *testing.T) {
	src := virtualAppSettingSource(`/tmp/mod"quote`, "app'name\\x")
	if !strings.Contains(src, `from "/tmp/mod\"quote/core/service/index.ts"`) {
		t.Fatalf("expected quoted core import, got:\n%s", src)
	}
	if !strings.Contains(src, `application: "app'name\\x"`) {
		t.Fatalf("expected quoted application literal, got:\n%s", src)
	}
	if !strings.Contains(src, "softDelete: false") {
		t.Fatalf("expected softDelete: false in C2 template, got:\n%s", src)
	}
}

func TestEnsureAppSettingVirtualImports_SkipsHandwritten(t *testing.T) {
	owner := &meta.Module{
		Name: "partner", Path: "/virtual/modules/partner",
		ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
	}
	builder, db := newAppSettingTestBuilder(t, owner)
	if err := db.Create(&meta.Model{
		BaseModel:   meta.BaseModel{Id: sql.NullString{String: "hand", Valid: true}},
		Name:        "AppSetting",
		Path:        "/virtual/modules/partner/service/models/app_setting.ts",
		Application: "partner",
	}).Error; err != nil {
		t.Fatalf("seed handwritten: %v", err)
	}
	base := &meta.Module{
		Name: "base", Path: "/virtual/modules/base",
		ApplicationStr: "base", ServiceEntryPoint: "service/index.ts",
	}
	if err := builder.EnsureAppSettingVirtualImports([]*meta.Module{owner, base}); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	wantBase := appSettingGeneratedPath(base.Path)
	if len(builder.appSettingInjectPaths) != 1 || builder.appSettingInjectPaths[0] != wantBase {
		t.Fatalf("expected only base C2 path, got %#v", builder.appSettingInjectPaths)
	}
	stub := builder.buildPlugin.(*stubEsbPlugin)
	if _, ok := stub.virtualSources[appSettingGeneratedPath(owner.Path)]; ok {
		t.Fatal("handwritten app must not register C2 virtual source")
	}
	if _, ok := stub.virtualSources[wantBase]; !ok {
		t.Fatalf("expected base virtual source, got %#v", stub.virtualSources)
	}
}

func TestEnsureAppSettingVirtualImports_PrefersMetaVirtualPath(t *testing.T) {
	// Owner candidate is a sibling path; meta already points at the primary module.
	sibling := &meta.Module{
		Name: "partner_bank", Path: "/virtual/modules/partner_bank",
		ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
	}
	builder, db := newAppSettingTestBuilder(t, sibling)
	metaPath := "/virtual/modules/partner/service/models/__generated__/app_setting.ts"
	if err := db.Create(&meta.Model{
		BaseModel:   meta.BaseModel{Id: sql.NullString{String: "virt", Valid: true}},
		Name:        "AppSetting",
		Path:        metaPath,
		Application: "partner",
	}).Error; err != nil {
		t.Fatalf("seed virt: %v", err)
	}
	if err := builder.EnsureAppSettingVirtualImports([]*meta.Module{sibling}); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	if len(builder.appSettingInjectPaths) != 1 || builder.appSettingInjectPaths[0] != metaPath {
		t.Fatalf("expected meta virt path, got %#v", builder.appSettingInjectPaths)
	}
}

func TestAppSettingPatchCoverage_Branches(t *testing.T) {
	if src := virtualAppSettingSource("/virtual/modules", "  "); !strings.Contains(src, `application: "application"`) {
		t.Fatalf("empty application should default, got:\n%s", src)
	}

	(*ModuleBuilder)(nil).rememberAppSettingInjectPath("/x")
	builder, _ := newAppSettingTestBuilder(t, &meta.Module{
		Name: "partner", Path: "/virtual/modules/partner",
		ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
	})
	builder.rememberAppSettingInjectPath("   ")
	if builder.appSettingInjectPath != "" || len(builder.appSettingInjectPaths) != 0 {
		t.Fatalf("empty path must not remember, got path=%q paths=%#v", builder.appSettingInjectPath, builder.appSettingInjectPaths)
	}
	builder.rememberAppSettingInjectPath("/a")
	builder.rememberAppSettingInjectPath("/a")
	if len(builder.appSettingInjectPaths) != 1 {
		t.Fatalf("dedupe remember: %#v", builder.appSettingInjectPaths)
	}

	// Only appSettingInjectPath (no slice) must still merge into buildOptions.
	builder.appSettingInjectPaths = nil
	builder.appSettingInjectPath = "/only-single"
	stub := builder.buildPlugin.(*stubEsbPlugin)
	stub.SetEntryPointImports(nil)
	_ = builder.buildOptions(false)
	found := false
	for _, imp := range stub.entryImports {
		if imp == "/only-single" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected single inject path in entry imports, got %#v", stub.entryImports)
	}

	if err := (*ModuleBuilder)(nil).EnsureAppSettingVirtualImports(nil); err != nil {
		t.Fatalf("nil ensure: %v", err)
	}
	if err := builder.EnsureAppSettingVirtualImports([]*meta.Module{
		nil,
		{Name: "core", Path: "/virtual/modules/core", ApplicationStr: "core", ServiceEntryPoint: "service/index.ts"},
		{Name: "noentry", Path: "/virtual/modules/x", ApplicationStr: "x", ServiceEntryPoint: ""},
		{Name: "nopath", Path: "", ApplicationStr: "y", ServiceEntryPoint: "service/index.ts"},
		{Name: "emptyapp", Path: "/virtual/modules/z", ApplicationStr: "  ", ServiceEntryPoint: "service/index.ts"},
	}); err != nil {
		t.Fatalf("skip-only ensure: %v", err)
	}
	// Clear paths then ensure only skipped apps → early return at empty inject paths.
	builder.appSettingInjectPaths = nil
	builder.appSettingInjectPath = ""
	if err := builder.EnsureAppSettingVirtualImports([]*meta.Module{
		{Name: "core", Path: "/m/core", ApplicationStr: "core", ServiceEntryPoint: "service/index.ts"},
	}); err != nil {
		t.Fatalf("empty ensure: %v", err)
	}
	if len(builder.appSettingInjectPaths) != 0 {
		t.Fatalf("expected no inject paths, got %#v", builder.appSettingInjectPaths)
	}

	// Handwritten-only ensure also returns with empty inject paths.
	builder, db := newAppSettingTestBuilder(t, &meta.Module{
		Name: "partner", Path: "/virtual/modules/partner",
		ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
	})
	if err := db.Create(&meta.Model{
		BaseModel:   meta.BaseModel{Id: sql.NullString{String: "hand2", Valid: true}},
		Name:        "AppSetting",
		Path:        "/virtual/modules/partner/service/models/app_setting.ts",
		Application: "partner",
	}).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := builder.EnsureAppSettingVirtualImports([]*meta.Module{builder.module}); err != nil {
		t.Fatalf("handwritten-only ensure: %v", err)
	}
	if len(builder.appSettingInjectPaths) != 0 {
		t.Fatalf("handwritten-only should skip C2, got %#v", builder.appSettingInjectPaths)
	}

	// dbLoad error from Ensure.
	builder, db = newAppSettingTestBuilder(t, &meta.Module{
		Name: "base", Path: "/virtual/modules/base",
		ApplicationStr: "base", ServiceEntryPoint: "service/index.ts",
	})
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	_ = sqlDB.Close()
	if err := builder.EnsureAppSettingVirtualImports([]*meta.Module{builder.module}); err == nil || !strings.Contains(err.Error(), "load AppSetting") {
		t.Fatalf("expected ensure load error, got %v", err)
	}

	// Empty modulesPath + prebuild plugin registration / entry imports.
	builder, _ = newAppSettingTestBuilder(t, &meta.Module{
		Name: "base", Path: "/virtual/modules/base",
		ApplicationStr: "base", ServiceEntryPoint: "service/index.ts",
	})
	pre := &stubEsbPlugin{name: "prebuild"}
	builder.prebuildPlugin = pre
	builder.runtimeScope.(*builderTestScope).cfg.ModulesPath = ""
	if err := builder.EnsureAppSettingVirtualImports([]*meta.Module{builder.module}); err != nil {
		t.Fatalf("ensure empty modulesPath: %v", err)
	}
	want := appSettingGeneratedPath(builder.module.Path)
	if _, ok := pre.virtualSources[want]; !ok {
		t.Fatalf("expected prebuild virtual source, got %#v", pre.virtualSources)
	}
	if len(pre.entryImports) == 0 {
		t.Fatal("expected prebuild entry imports")
	}

	// applyAppSettingInject also wires prebuild plugin.
	builder, _ = newAppSettingTestBuilder(t, &meta.Module{
		Name: "meta", Path: "/virtual/modules/meta",
		ApplicationStr: "meta", ServiceEntryPoint: "service/index.ts",
	})
	pre = &stubEsbPlugin{name: "prebuild"}
	builder.prebuildPlugin = pre
	if err := builder.applyAppSettingInject(AppSettingPlan{NeedInject: true}); err != nil {
		t.Fatalf("apply with prebuild: %v", err)
	}
	wantMeta := appSettingGeneratedPath(builder.module.Path)
	if _, ok := pre.virtualSources[wantMeta]; !ok {
		t.Fatalf("apply should register on prebuild plugin: %#v", pre.virtualSources)
	}
	if len(pre.entryImports) == 0 {
		t.Fatal("apply should set prebuild entry imports")
	}
}

func TestEnsureAppSettingVirtualImports_SurvivesBuildOptionsReplace(t *testing.T) {
	// Multi-app bundles Decide/Inject against a core representative; Ensure must
	// still get every app's AppSetting into WithEntryPointImports.
	core := &meta.Module{
		Name: "core", Path: "/virtual/modules/core",
		ApplicationStr: "core", ServiceEntryPoint: "service/index.ts",
	}
	builder, _ := newAppSettingTestBuilder(t, core)
	owners := []*meta.Module{
		{Name: "base", Path: "/virtual/modules/base", ApplicationStr: "base", ServiceEntryPoint: "service/index.ts"},
		{Name: "task", Path: "/virtual/modules/task", ApplicationStr: "task", ServiceEntryPoint: "service/index.ts"},
		{Name: "base_dup", Path: "/virtual/modules/base2", ApplicationStr: "base", ServiceEntryPoint: "service/index.ts"},
	}
	if err := builder.EnsureAppSettingVirtualImports(owners); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	wantBase := appSettingGeneratedPath(owners[0].Path)
	wantTask := appSettingGeneratedPath(owners[1].Path)
	if len(builder.appSettingInjectPaths) != 2 {
		t.Fatalf("expected 2 inject paths (dedupe app), got %#v", builder.appSettingInjectPaths)
	}

	stub := builder.buildPlugin.(*stubEsbPlugin)
	// Simulate a wipe of whatever Ensure put on the plugin; buildOptions must restore.
	stub.SetEntryPointImports([]string{"/wiped"})
	_ = builder.buildOptions(false)

	foundBase, foundTask := false, false
	for _, imp := range stub.entryImports {
		if imp == wantBase {
			foundBase = true
		}
		if imp == wantTask {
			foundTask = true
		}
	}
	if !foundBase || !foundTask {
		t.Fatalf("expected inject paths in entry imports after buildOptions, got %#v", stub.entryImports)
	}
	if _, ok := stub.virtualSources[wantBase]; !ok {
		t.Fatalf("missing virtual source for base: %#v", stub.virtualSources)
	}
	if _, ok := stub.virtualSources[wantTask]; !ok {
		t.Fatalf("missing virtual source for task: %#v", stub.virtualSources)
	}
}

func TestReleaseAppSettingSchedule_AllowsRetryAfterFailure(t *testing.T) {
	mod := &meta.Module{
		Name: "partner", Path: "/virtual/modules/partner",
		ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
	}
	builder, _ := newAppSettingTestBuilder(t, mod)
	plan, err := builder.decideAppSettingPlan(nil)
	if err != nil || !plan.NeedInject || plan.scheduledApp != "partner" {
		t.Fatalf("expected NeedInject claim, got %+v err=%v", plan, err)
	}
	builder.appSettingPlan = plan
	builder.releaseAppSettingSchedule()

	plan2, err := builder.decideAppSettingPlan(nil)
	if err != nil || !plan2.NeedInject {
		t.Fatalf("expected retry inject after release, got %+v err=%v", plan2, err)
	}
}

func TestValidateAppSetting_DuplicateHandAndVirtual(t *testing.T) {
	mod := &meta.Module{Name: "partner", Path: "/virtual/modules/partner", ApplicationStr: "partner"}
	builder, _ := newAppSettingTestBuilder(t, mod)
	hand := filepath.Join(mod.Path, "service/models/app_setting.ts")
	virt := appSettingGeneratedPath(mod.Path)
	buildResult := module.WithParserResults(&module.BuildResult{Module: mod}, []*parser.ParserResult{
		{Path: hand, Model: &meta.Model{Name: "AppSetting", Path: hand}},
		{Path: virt, Model: &meta.Model{Name: "AppSetting", Path: virt}},
	})
	err := builder.validateAppSetting(buildResult)
	if err == nil || !strings.Contains(err.Error(), appSettingDuplicateCode) {
		t.Fatalf("expected duplicate error, got %v", err)
	}
}

func TestSupersedeVirtualAppSettings_DeletesGeneratedRows(t *testing.T) {
	mod := &meta.Module{
		Name: "partner_bank", Path: "/virtual/modules/partner_bank",
		ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
	}
	builder, db := newAppSettingTestBuilder(t, mod)
	builder.appSettingPlan = AppSettingPlan{SupersedeVirtual: true}
	if err := db.Create(&meta.Model{
		BaseModel:   meta.BaseModel{Id: sql.NullString{String: "virt", Valid: true}},
		Name:        "AppSetting",
		Path:        "/virtual/modules/partner/service/models/__generated__/app_setting.ts",
		Application: "partner",
	}).Error; err != nil {
		t.Fatalf("seed virt: %v", err)
	}
	if err := db.Create(&meta.Model{
		BaseModel:   meta.BaseModel{Id: sql.NullString{String: "hand", Valid: true}},
		Name:        "AppSetting",
		Path:        "/virtual/modules/partner_bank/service/models/app_setting.ts",
		Application: "partner",
	}).Error; err != nil {
		t.Fatalf("seed hand: %v", err)
	}
	if err := db.Create(&meta.Model{
		BaseModel:   meta.BaseModel{Id: sql.NullString{String: "other", Valid: true}},
		Name:        "Partner",
		Path:        "/virtual/modules/partner/service/models/partner.ts",
		Application: "partner",
	}).Error; err != nil {
		t.Fatalf("seed other: %v", err)
	}
	if err := builder.supersedeVirtualAppSettings(); err != nil {
		t.Fatalf("supersede: %v", err)
	}
	var count int64
	if err := db.Model(&meta.Model{}).Where("name = ?", "AppSetting").Count(&count).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected handwritten AppSetting kept, count=%d", count)
	}
	var virtLeft int64
	if err := db.Model(&meta.Model{}).Where("id = ?", "virt").Count(&virtLeft).Error; err != nil {
		t.Fatalf("count virt: %v", err)
	}
	if virtLeft != 0 {
		t.Fatalf("expected virtual AppSetting deleted, count=%d", virtLeft)
	}
	if err := db.Model(&meta.Model{}).Where("name = ?", "Partner").Count(&count).Error; err != nil {
		t.Fatalf("count partner: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected unrelated model kept, count=%d", count)
	}
}
