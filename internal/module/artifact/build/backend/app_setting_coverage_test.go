// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package backendbuilder

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/internal/esbplugins"
	module "github.com/choysum-dev/choysum/internal/module/artifact/result"
	"github.com/choysum-dev/choysum/internal/parser"
	"github.com/choysum-dev/choysum/pkg/jsexecutor"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/evanw/esbuild/pkg/api"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// bareAppSettingEsbPlugin implements EsbPlugin without entry-import / virtual-source helpers.
type bareAppSettingEsbPlugin struct{ name string }

func (p bareAppSettingEsbPlugin) DefinePlugins(_ scope.Scope, _ jsexecutor.ScriptExecutor, _ *meta.Module, _ ...esbplugins.EsbPluginOptions) []api.Plugin {
	return []api.Plugin{{Name: p.name}}
}
func (p bareAppSettingEsbPlugin) GetParserResults() ([]*parser.ParserResult, error) { return nil, nil }
func (p bareAppSettingEsbPlugin) SetParserResults([]*parser.ParserResult) error     { return nil }

// stickyAppSettingParserPlugin ignores SetParserResults so build can return fixed duplicates for validate.
type stickyAppSettingParserPlugin struct {
	results []*parser.ParserResult
}

func (p *stickyAppSettingParserPlugin) DefinePlugins(_ scope.Scope, _ jsexecutor.ScriptExecutor, _ *meta.Module, options ...esbplugins.EsbPluginOptions) []api.Plugin {
	for _, opt := range options {
		if opt != nil {
			opt(p)
		}
	}
	return []api.Plugin{{Name: "sticky", Setup: func(api.PluginBuild) {}}}
}
func (p *stickyAppSettingParserPlugin) GetParserResults() ([]*parser.ParserResult, error) {
	return p.results, nil
}
func (p *stickyAppSettingParserPlugin) SetParserResults([]*parser.ParserResult) error { return nil }
func (p *stickyAppSettingParserPlugin) SetEntryPointImports([]string)                 {}
func (p *stickyAppSettingParserPlugin) RegisterVirtualSource(string, string)          {}

func TestAppSettingHelpers_EdgeCases(t *testing.T) {
	if isGeneratedAppSettingPath("") || isGeneratedAppSettingPath("   ") {
		t.Fatal("empty path should not be generated")
	}
	if !isGeneratedAppSettingPath(appSettingGeneratedRelPath) {
		t.Fatal("relative generated path should match")
	}

	out := appSettingsIn([]*parser.ParserResult{
		nil,
		{Path: "/x", Model: nil},
		{Path: "/other/mod/service/models/app_setting.ts", Model: &meta.Model{Name: "AppSetting", Path: "/other/mod/service/models/app_setting.ts"}},
		{Path: "/m/service/models/app_setting.ts", Model: &meta.Model{Name: "AppSetting", Path: "/m/service/models/app_setting.ts", Abstract: true}},
		{Path: "/m/service/models/app_setting.ts", Model: &meta.Model{Name: "Partner", Path: "/m/service/models/partner.ts"}},
		{Path: "/m/service/models/app_setting.ts", Model: &meta.Model{Name: "AppSetting", Path: "/m/service/models/app_setting.ts"}},
	}, "/m")
	if len(out) != 1 || out[0].Name != "AppSetting" {
		t.Fatalf("appSettingsIn filter = %#v", out)
	}

	if appSettingSameModule(nil, nil) {
		t.Fatal("nil module")
	}
	mod := &meta.Module{Path: "/m", BaseModel: meta.BaseModel{Id: sql.NullString{String: "mid", Valid: true}}}
	if !appSettingSameModule([]*meta.Model{
		nil,
		{ModuleId: sql.NullString{String: "mid", Valid: true}},
	}, mod) {
		t.Fatal("expected module id match")
	}
	if !appSettingSameModule([]*meta.Model{
		{Path: "/m/service/models/app_setting.ts"},
	}, &meta.Module{Path: "/m"}) {
		t.Fatal("expected path match")
	}
	if appSettingSameModule([]*meta.Model{{Path: "/other/x.ts"}}, &meta.Module{Path: "/m"}) {
		t.Fatal("expected path mismatch")
	}

	hand := handwrittenAppSettings([]*meta.Model{nil, {Path: appSettingGeneratedPath("/m")}})
	if len(hand) != 0 {
		t.Fatalf("handwritten filter: %#v", hand)
	}
	gen := generatedAppSettings([]*meta.Model{nil, {Path: "/m/service/models/app_setting.ts"}})
	if len(gen) != 0 {
		t.Fatalf("generated filter: %#v", gen)
	}
}

func TestDbLoadAppSettings_GuardsAndError(t *testing.T) {
	if models, err := (*ModuleBuilder)(nil).dbLoadAppSettings("partner"); models != nil || err != nil {
		t.Fatalf("nil builder: %#v %v", models, err)
	}
	builder, db := newAppSettingTestBuilder(t, &meta.Module{Name: "p", Path: "/m", ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts"})
	if models, err := builder.dbLoadAppSettings(""); models != nil || err != nil {
		t.Fatalf("empty app: %#v %v", models, err)
	}
	builder.runtimeScope = nil
	if models, err := builder.dbLoadAppSettings("partner"); models != nil || err != nil {
		t.Fatalf("nil scope: %#v %v", models, err)
	}

	builder, db = newAppSettingTestBuilder(t, &meta.Module{Name: "p", Path: "/m", ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts"})
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	_ = sqlDB.Close()
	if _, err := builder.dbLoadAppSettings("partner"); err == nil {
		t.Fatal("expected find error on closed db")
	}
	if _, err := builder.decideAppSettingPlan(nil); err == nil || !strings.Contains(err.Error(), "load AppSetting") {
		t.Fatalf("expected decide load error, got %v", err)
	}
}

func TestReleaseAppSettingSchedule_NilBuilder(t *testing.T) {
	(*ModuleBuilder)(nil).releaseAppSettingSchedule()
}

func TestDecideAppSettingPlan_MoreBranches(t *testing.T) {
	if plan, err := (*ModuleBuilder)(nil).decideAppSettingPlan(nil); err != nil || plan.NeedInject {
		t.Fatalf("nil builder: %+v %v", plan, err)
	}
	builder, _ := newAppSettingTestBuilder(t, &meta.Module{
		Name: "partner", Path: "/virtual/modules/partner",
		ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
	})
	handA := "/virtual/modules/partner/service/models/fd_a.ts"
	handB := "/virtual/modules/partner/service/models/fd_b.ts"
	_, err := builder.decideAppSettingPlan([]*parser.ParserResult{
		{Path: handA, Model: &meta.Model{Name: "AppSetting", Path: handA}},
		{Path: handB, Model: &meta.Model{Name: "AppSetting", Path: handB}},
	})
	if err == nil || !strings.Contains(err.Error(), appSettingDuplicateCode) {
		t.Fatalf("expected local duplicate, got %v", err)
	}

	// Local handwritten, no DB virtual → no-op plan.
	plan, err := builder.decideAppSettingPlan([]*parser.ParserResult{
		{Path: handA, Model: &meta.Model{Name: "AppSetting", Path: handA}},
	})
	if err != nil || plan.NeedInject || plan.SupersedeVirtual {
		t.Fatalf("expected no-op for local hand only, got %+v %v", plan, err)
	}

	// Local generated (no handwritten) → skip inject.
	virt := appSettingGeneratedPath(builder.module.Path)
	plan, err = builder.decideAppSettingPlan([]*parser.ParserResult{
		{Path: virt, Model: &meta.Model{Name: "AppSetting", Path: virt}},
	})
	if err != nil || plan.NeedInject {
		t.Fatalf("local generated should skip inject, got %+v %v", plan, err)
	}

	// Same-module reclaim.
	resetAppSettingScheduledAppsForTest()
	plan, err = builder.decideAppSettingPlan(nil)
	if err != nil || !plan.NeedInject {
		t.Fatalf("first inject: %+v %v", plan, err)
	}
	plan2, err := builder.decideAppSettingPlan(nil)
	if err != nil || !plan2.NeedInject || plan2.scheduledApp != "partner" {
		t.Fatalf("reclaim: %+v %v", plan2, err)
	}
}

func TestApplyAndPlanAppSettingInject_ErrorPaths(t *testing.T) {
	if err := (*ModuleBuilder)(nil).applyAppSettingInject(AppSettingPlan{NeedInject: true}); err != nil {
		t.Fatal(err)
	}
	builder, _ := newAppSettingTestBuilder(t, &meta.Module{
		Name: "partner", ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
	})
	if err := builder.applyAppSettingInject(AppSettingPlan{NeedInject: true}); err == nil || !strings.Contains(err.Error(), "module path") {
		t.Fatalf("expected path error, got %v", err)
	}

	// Empty modulesPath in config → derive from module.Path; still register virtual source.
	builder.module.Path = "/virtual/modules/partner"
	builder.runtimeScope.(*builderTestScope).cfg.ModulesPath = ""
	if err := builder.applyAppSettingInject(AppSettingPlan{NeedInject: true}); err != nil {
		t.Fatalf("inject with empty modulesPath: %v", err)
	}
	stub := builder.buildPlugin.(*stubEsbPlugin)
	if len(stub.virtualSources) == 0 {
		t.Fatal("expected virtual source registration")
	}

	// buildPlugin without SetEntryPointImports / RegisterVirtualSource still succeeds.
	builder.buildPlugin = bareAppSettingEsbPlugin{name: "bare"}
	if err := builder.applyAppSettingInject(AppSettingPlan{NeedInject: true}); err != nil {
		t.Fatalf("bare plugin: %v", err)
	}

	builder, _ = newAppSettingTestBuilder(t, &meta.Module{
		Name: "partner", Path: "/virtual/modules/partner",
		ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
	})
	err := builder.planAndInjectAppSetting(module.WithParserResults(&module.BuildResult{Module: builder.module}, []*parser.ParserResult{
		{Path: "/virtual/modules/partner/a.ts", Model: &meta.Model{Name: "AppSetting", Path: "/virtual/modules/partner/a.ts"}},
		{Path: "/virtual/modules/partner/b.ts", Model: &meta.Model{Name: "AppSetting", Path: "/virtual/modules/partner/b.ts"}},
	}))
	if err == nil || !strings.Contains(err.Error(), appSettingDuplicateCode) {
		t.Fatalf("planAndInject decide error: %v", err)
	}

	builder.appSettingPlan = AppSettingPlan{}
	resetAppSettingScheduledAppsForTest()
	builder.module.Path = ""
	builder.module.ApplicationStr = "partner"
	builder.module.ServiceEntryPoint = "service/index.ts"
	builder.module.Name = "partner"
	err = builder.planAndInjectAppSetting(module.WithParserResults(&module.BuildResult{Module: builder.module}, nil))
	if err == nil || !strings.Contains(err.Error(), "injecting AppSetting") {
		t.Fatalf("planAndInject inject error: %v", err)
	}
	if _, loaded := appSettingScheduledApps.Load("partner"); loaded {
		t.Fatal("schedule should be released after inject failure")
	}
}

func TestValidateAppSetting_Guards(t *testing.T) {
	if err := (*ModuleBuilder)(nil).validateAppSetting(nil); err != nil {
		t.Fatal(err)
	}
	builder, _ := newAppSettingTestBuilder(t, &meta.Module{Name: "core", Path: "/m", ApplicationStr: "core"})
	if err := builder.validateAppSetting(&module.BuildResult{Module: builder.module}); err != nil {
		t.Fatal(err)
	}
	builder.module.ApplicationStr = ""
	if err := builder.validateAppSetting(&module.BuildResult{Module: builder.module}); err != nil {
		t.Fatal(err)
	}
	builder.module.ApplicationStr = "partner"
	if err := builder.validateAppSetting(module.WithParserResults(&module.BuildResult{Module: builder.module}, nil)); err != nil {
		t.Fatal(err)
	}
}

func TestSupersedeVirtualAppSettings_GuardsAndDependents(t *testing.T) {
	if err := (*ModuleBuilder)(nil).supersedeVirtualAppSettings(); err != nil {
		t.Fatal(err)
	}
	builder, db := newAppSettingTestBuilder(t, &meta.Module{
		Name: "partner_bank", Path: "/virtual/modules/partner_bank",
		ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
	})
	if err := builder.supersedeVirtualAppSettings(); err != nil {
		t.Fatal(err)
	}
	builder.appSettingPlan = AppSettingPlan{SupersedeVirtual: true}
	builder.runtimeScope = nil
	if err := builder.supersedeVirtualAppSettings(); err != nil {
		t.Fatal(err)
	}
	builder, db = newAppSettingTestBuilder(t, &meta.Module{
		Name: "partner_bank", Path: "/virtual/modules/partner_bank",
		ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
	})
	builder.appSettingPlan = AppSettingPlan{SupersedeVirtual: true}
	builder.module.ApplicationStr = ""
	if err := builder.supersedeVirtualAppSettings(); err != nil {
		t.Fatal(err)
	}
	builder.module.ApplicationStr = "partner"

	// Generated row with empty id → filtered out (no delete).
	if err := db.Create(&meta.Model{
		BaseModel:   meta.BaseModel{Id: sql.NullString{String: "   ", Valid: true}},
		Name:        "AppSetting",
		Path:        "/virtual/modules/partner/service/models/__generated__/app_setting.ts",
		Application: "partner",
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := builder.supersedeVirtualAppSettings(); err != nil {
		t.Fatal(err)
	}

	// Session with nil DB handle (guard before Find).
	builder.runtimeScope = &builderTestScope{session: &scope.Session{DB: nil}}
	builder.appSettingPlan = AppSettingPlan{SupersedeVirtual: true}
	builder.module.ApplicationStr = "partner"
	if err := builder.supersedeVirtualAppSettings(); err != nil {
		t.Fatal(err)
	}

	// Full dependent tree delete.
	builder, db = newAppSettingTestBuilder(t, &meta.Module{
		Name: "partner_bank", Path: "/virtual/modules/partner_bank",
		ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
	})
	builder.appSettingPlan = AppSettingPlan{SupersedeVirtual: true}
	virtID := "virt-dep"
	if err := db.Create(&meta.Model{
		BaseModel: meta.BaseModel{Id: sql.NullString{String: virtID, Valid: true}},
		Name:      "AppSetting", Path: "/virtual/modules/partner/service/models/__generated__/app_setting.ts", Application: "partner",
	}).Error; err != nil {
		t.Fatal(err)
	}
	fieldID := "fld1"
	svcID := "svc1"
	decModel := "dec-m"
	decField := "dec-f"
	decSvc := "dec-s"
	if err := db.Create(&meta.Field{BaseModel: meta.BaseModel{Id: sql.NullString{String: fieldID, Valid: true}}, Name: "Model", ModelId: sql.NullString{String: virtID, Valid: true}}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&meta.Service{BaseModel: meta.BaseModel{Id: sql.NullString{String: svcID, Valid: true}}, Name: "Get", ModelId: sql.NullString{String: virtID, Valid: true}}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&meta.Decorator{BaseModel: meta.BaseModel{Id: sql.NullString{String: decModel, Valid: true}}, Name: "Model", ModelId: sql.NullString{String: virtID, Valid: true}}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&meta.Decorator{BaseModel: meta.BaseModel{Id: sql.NullString{String: decField, Valid: true}}, Name: "Field", FieldId: sql.NullString{String: fieldID, Valid: true}}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&meta.Decorator{BaseModel: meta.BaseModel{Id: sql.NullString{String: decSvc, Valid: true}}, Name: "Service", ServiceId: sql.NullString{String: svcID, Valid: true}}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&meta.Argument{BaseModel: meta.BaseModel{Id: sql.NullString{String: "arg1", Valid: true}}, DecoratorId: sql.NullString{String: decModel, Valid: true}}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&meta.TypeParameter{BaseModel: meta.BaseModel{Id: sql.NullString{String: "tp1", Valid: true}}, Name: "T", ServiceId: sql.NullString{String: svcID, Valid: true}}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&meta.Parameter{BaseModel: meta.BaseModel{Id: sql.NullString{String: "p1", Valid: true}}, Name: "x", ServiceId: sql.NullString{String: svcID, Valid: true}}).Error; err != nil {
		t.Fatal(err)
	}
	if err := builder.supersedeVirtualAppSettings(); err != nil {
		t.Fatalf("supersede dependents: %v", err)
	}
	var left int64
	if err := db.Unscoped().Model(&meta.Model{}).Where("id = ?", virtID).Count(&left).Error; err != nil || left != 0 {
		t.Fatalf("virt model left=%d err=%v", left, err)
	}
}

func TestSupersedeVirtualAppSettings_ErrorBranches(t *testing.T) {
	builder, db := newAppSettingTestBuilder(t, &meta.Module{
		Name: "partner_bank", Path: "/virtual/modules/partner_bank",
		ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
	})
	builder.appSettingPlan = AppSettingPlan{SupersedeVirtual: true}
	if err := db.Create(&meta.Model{
		BaseModel: meta.BaseModel{Id: sql.NullString{String: "virt", Valid: true}},
		Name:      "AppSetting", Path: "/virtual/modules/partner/service/models/__generated__/app_setting.ts", Application: "partner",
	}).Error; err != nil {
		t.Fatal(err)
	}

	// Find error
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	_ = sqlDB.Close()
	if err := builder.supersedeVirtualAppSettings(); err == nil || !strings.Contains(err.Error(), "load AppSetting rows") {
		t.Fatalf("expected load error, got %v", err)
	}

	// Pluck / delete errors via callbacks on a fresh DB.
	cases := []struct {
		name   string
		table  string
		phase  string // query|delete
		substr string
	}{
		{name: "service pluck", table: "meta_service", phase: "query", substr: "load superseded AppSetting services"},
		{name: "field pluck", table: "meta_field", phase: "query", substr: "load superseded AppSetting fields"},
		{name: "decorator pluck", table: "meta_decorator", phase: "query", substr: "load superseded AppSetting decorators"},
		{name: "argument delete", table: "meta_argument", phase: "delete", substr: "decorator arguments"},
		{name: "decorator delete", table: "meta_decorator", phase: "delete", substr: "delete superseded AppSetting decorators"},
		{name: "type param delete", table: "meta_type_parameter", phase: "delete", substr: "type parameters"},
		{name: "parameter delete", table: "meta_parameter", phase: "delete", substr: "delete superseded AppSetting parameters"},
		{name: "service delete", table: "meta_service", phase: "delete", substr: "delete superseded AppSetting services"},
		{name: "field delete", table: "meta_field", phase: "delete", substr: "delete superseded AppSetting fields"},
		{name: "model delete", table: "meta_model", phase: "delete", substr: "delete superseded virtual AppSetting rows"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b, gdb := newAppSettingTestBuilder(t, &meta.Module{
				Name: "partner_bank", Path: "/virtual/modules/partner_bank",
				ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
			})
			b.appSettingPlan = AppSettingPlan{SupersedeVirtual: true}
			virtID := "v-" + tc.name
			if err := gdb.Create(&meta.Model{
				BaseModel: meta.BaseModel{Id: sql.NullString{String: virtID, Valid: true}},
				Name:      "AppSetting", Path: "/virtual/modules/partner/service/models/__generated__/app_setting.ts", Application: "partner",
			}).Error; err != nil {
				t.Fatal(err)
			}
			fieldID := "f-" + tc.name
			svcID := "s-" + tc.name
			decID := "d-" + tc.name
			if err := gdb.Create(&meta.Field{BaseModel: meta.BaseModel{Id: sql.NullString{String: fieldID, Valid: true}}, Name: "Model", ModelId: sql.NullString{String: virtID, Valid: true}}).Error; err != nil {
				t.Fatal(err)
			}
			if err := gdb.Create(&meta.Service{BaseModel: meta.BaseModel{Id: sql.NullString{String: svcID, Valid: true}}, Name: "Get", ModelId: sql.NullString{String: virtID, Valid: true}}).Error; err != nil {
				t.Fatal(err)
			}
			if err := gdb.Create(&meta.Decorator{BaseModel: meta.BaseModel{Id: sql.NullString{String: decID, Valid: true}}, Name: "Model", ModelId: sql.NullString{String: virtID, Valid: true}}).Error; err != nil {
				t.Fatal(err)
			}
			if err := gdb.Create(&meta.Argument{BaseModel: meta.BaseModel{Id: sql.NullString{String: "a-" + tc.name, Valid: true}}, DecoratorId: sql.NullString{String: decID, Valid: true}}).Error; err != nil {
				t.Fatal(err)
			}
			if err := gdb.Create(&meta.TypeParameter{BaseModel: meta.BaseModel{Id: sql.NullString{String: "t-" + tc.name, Valid: true}}, Name: "T", ServiceId: sql.NullString{String: svcID, Valid: true}}).Error; err != nil {
				t.Fatal(err)
			}
			if err := gdb.Create(&meta.Parameter{BaseModel: meta.BaseModel{Id: sql.NullString{String: "p-" + tc.name, Valid: true}}, Name: "x", ServiceId: sql.NullString{String: svcID, Valid: true}}).Error; err != nil {
				t.Fatal(err)
			}

			boom := errors.New("forced-" + tc.name)
			switch tc.phase {
			case "query":
				if err := gdb.Callback().Query().Before("gorm:query").Register("force-"+tc.name, func(tx *gorm.DB) {
					if tx.Statement != nil && tx.Statement.Table == tc.table {
						_ = tx.AddError(boom)
					}
				}); err != nil {
					t.Fatal(err)
				}
			case "delete":
				if err := gdb.Callback().Delete().Before("gorm:delete").Register("force-"+tc.name, func(tx *gorm.DB) {
					if tx.Statement != nil && tx.Statement.Table == tc.table {
						_ = tx.AddError(boom)
					}
				}); err != nil {
					t.Fatal(err)
				}
			}
			err := b.supersedeVirtualAppSettings()
			if err == nil || !strings.Contains(err.Error(), tc.substr) {
				t.Fatalf("expected %q in error, got %v", tc.substr, err)
			}
		})
	}
}

func TestBuildLifecycle_AppSettingInjectAndRelease(t *testing.T) {
	resetAppSettingScheduledAppsForTest()
	db, err := gorm.Open(sqlite.Open(appSettingMemoryDSN(t, "fd-lifecycle")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&meta.Application{}, &meta.Module{}, &meta.Model{}, &meta.Field{}, &meta.Service{}, &meta.Decorator{}, &meta.Argument{}, &meta.Parameter{}, &meta.TypeParameter{}); err != nil {
		t.Fatal(err)
	}
	testScope := newBuilderTestScope()
	testScope.session = &scope.Session{DB: db}
	mod := &meta.Module{
		Name: "partner", Path: "/virtual/modules/partner",
		ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
	}
	buildPlugin := &stubEsbPlugin{name: "build"}
	prebuildPlugin := &stubEsbPlugin{name: "prebuild"}
	builder := &ModuleBuilder{
		runtimeScope:   testScope,
		module:         mod,
		buildPlugin:    buildPlugin,
		prebuildPlugin: prebuildPlugin,
		entryPoint:     "",
	}

	result, err := builder.BuildWithoutPersist()
	if err != nil {
		t.Fatalf("BuildWithoutPersist: %v", err)
	}
	if builder.appSettingInjectPath == "" || !builder.appSettingPlan.NeedInject {
		t.Fatalf("expected inject plan, path=%q plan=%+v", builder.appSettingInjectPath, builder.appSettingPlan)
	}
	if _, ok := buildPlugin.virtualSources[builder.appSettingInjectPath]; !ok {
		t.Fatal("expected virtual source on build plugin")
	}
	// buildOptions(false) should include inject path via SetEntryPointImports already called;
	// also exercise append branch by calling buildOptions directly.
	_ = builder.buildOptions(false)

	if err := builder.Persist(result); err != nil {
		t.Fatalf("Persist: %v", err)
	}
	if builder.appSettingPlan.scheduledApp != "" {
		t.Fatal("expected schedule cleared after Persist")
	}

	// Bundle releases schedule via defer.
	resetAppSettingScheduledAppsForTest()
	builder2 := &ModuleBuilder{
		runtimeScope:   testScope,
		module:         &meta.Module{Name: "partner_bank", Path: "/virtual/modules/partner_bank", ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts"},
		buildPlugin:    &stubEsbPlugin{name: "build"},
		prebuildPlugin: &stubEsbPlugin{name: "prebuild"},
		entryPoint:     "",
	}
	if _, err := builder2.Bundle(); err != nil {
		t.Fatalf("Bundle: %v", err)
	}
	if builder2.appSettingPlan.scheduledApp != "" {
		t.Fatal("bundle should clear scheduledApp")
	}
}

func TestBuildWithoutPersist_ReleasesAppSettingOnFailures(t *testing.T) {
	resetAppSettingScheduledAppsForTest()
	db, err := gorm.Open(sqlite.Open(appSettingMemoryDSN(t, "fd-fail")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&meta.Application{}, &meta.Module{}, &meta.Model{}); err != nil {
		t.Fatal(err)
	}
	testScope := newBuilderTestScope()
	testScope.session = &scope.Session{DB: db}

	// prebuild failure
	builder := &ModuleBuilder{
		runtimeScope: testScope,
		module:       &meta.Module{Name: "partner", Path: "/virtual/modules/partner", ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts"},
		buildPlugin:  &stubEsbPlugin{name: "build"}, prebuildPlugin: &stubEsbPlugin{name: "prebuild"},
		entryPoint: "/no/such/prebuild-entry.ts",
	}
	if _, err := builder.BuildWithoutPersist(); err == nil {
		t.Fatal("expected prebuild failure")
	}

	// planAndInjectAppSetting failure after FieldDefault skips (handwritten FD owns app).
	// Empty Path makes AppSetting NeedInject fail in apply; FieldDefault Decide skips.
	mod := &meta.Module{Name: "partner", ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts"}
	if err := db.Create(&meta.Model{
		BaseModel:   meta.BaseModel{Id: sql.NullString{String: "fd-hand", Valid: true}},
		Name:        "FieldDefault",
		Path:        "/virtual/modules/partner/service/models/field_default.ts",
		Application: "partner",
	}).Error; err != nil {
		t.Fatal(err)
	}
	builder = &ModuleBuilder{
		runtimeScope: testScope, module: mod,
		buildPlugin: &stubEsbPlugin{name: "build"}, prebuildPlugin: &stubEsbPlugin{name: "prebuild"},
		entryPoint: "",
	}
	if _, err := builder.BuildWithoutPersist(); err == nil {
		t.Fatal("expected AppSetting inject path error")
	}
	if _, loaded := appSettingScheduledApps.Load("partner"); loaded {
		t.Fatal("claim should be released")
	}

	// updatePrebuildResult failure → release (NeedInject claims, then extends refresh fails).
	resetAppSettingScheduledAppsForTest()
	dbUpd, err := gorm.Open(sqlite.Open(appSettingMemoryDSN(t, "fd-upd-fail")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := dbUpd.AutoMigrate(&meta.Model{}); err != nil {
		t.Fatal(err)
	}
	parentPath := "/virtual/modules/partner/service/models/parent.ts"
	latestPath := "/virtual/modules/partner/service/models/latest.ts"
	childPath := "/virtual/modules/partner/service/models/child.ts"
	if err := dbUpd.Create(&meta.Model{
		BaseModel: meta.BaseModel{Id: sql.NullString{String: "aaa", Valid: true}},
		Name:      "X", Path: parentPath, Application: "partner",
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := dbUpd.Create(&meta.Model{
		BaseModel: meta.BaseModel{Id: sql.NullString{String: "zzz", Valid: true}},
		Name:      "X", Path: latestPath, Application: "partner",
	}).Error; err != nil {
		t.Fatal(err)
	}
	scopeUpd := newBuilderTestScope()
	scopeUpd.session = &scope.Session{DB: dbUpd}
	pre := &stubEsbPlugin{name: "prebuild", parserResults: []*parser.ParserResult{{
		Path:       childPath,
		RawContent: "export default class X {}",
		Model:      &meta.Model{Name: "X", Path: childPath, Extends: parentPath},
	}}}
	builder = &ModuleBuilder{
		runtimeScope: scopeUpd,
		module:       &meta.Module{Name: "partner", Path: "/virtual/modules/partner", ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts"},
		buildPlugin:  &stubEsbPlugin{name: "build"}, prebuildPlugin: pre,
		entryPoint:  "",
		tsParser:    fixedParser{err: errors.New("reparse boom")},
		tsPathAlias: map[string]string{},
	}
	if _, err := builder.BuildWithoutPersist(); err == nil {
		t.Fatal("expected update/prebuild downstream error")
	}
	if _, loaded := appSettingScheduledApps.Load("partner"); loaded {
		t.Fatal("claim should be released after update failure")
	}

	// Persist failure releases.
	resetAppSettingScheduledAppsForTest()
	db2, err := gorm.Open(sqlite.Open(appSettingMemoryDSN(t, "fd-persist-fail")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	testScope2 := newBuilderTestScope()
	testScope2.session = &scope.Session{DB: db2}
	builder = &ModuleBuilder{
		runtimeScope: testScope2,
		module:       &meta.Module{Name: "partner", Path: "/virtual/modules/partner", ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts"},
		buildPlugin:  &stubEsbPlugin{name: "build"}, prebuildPlugin: &stubEsbPlugin{name: "prebuild"},
		entryPoint:     "",
		appSettingPlan: AppSettingPlan{NeedInject: true, scheduledApp: "partner"},
	}
	appSettingScheduledApps.Store("partner", "partner")
	if err := builder.Persist(&module.BuildResult{Module: builder.module}); err == nil {
		t.Fatal("expected persist failure without migrated tables")
	}
	if _, loaded := appSettingScheduledApps.Load("partner"); loaded {
		t.Fatal("persist failure should release schedule")
	}

	// Persist fails inside supersedeVirtualAppSettings (closed DB) and still releases.
	resetAppSettingScheduledAppsForTest()
	dbSupersede, err := gorm.Open(sqlite.Open(appSettingMemoryDSN(t, "fd-persist-supersede-fail")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := dbSupersede.AutoMigrate(&meta.Model{}); err != nil {
		t.Fatal(err)
	}
	sqlDBSupersede, err := dbSupersede.DB()
	if err != nil {
		t.Fatal(err)
	}
	_ = sqlDBSupersede.Close()
	scopeSupersede := newBuilderTestScope()
	scopeSupersede.session = &scope.Session{DB: dbSupersede}
	builder = &ModuleBuilder{
		runtimeScope: scopeSupersede,
		module:       &meta.Module{Name: "partner", Path: "/virtual/modules/partner", ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts"},
		buildPlugin:  &stubEsbPlugin{name: "build"}, prebuildPlugin: &stubEsbPlugin{name: "prebuild"},
		entryPoint:     "",
		appSettingPlan: AppSettingPlan{SupersedeVirtual: true, scheduledApp: "partner"},
	}
	appSettingScheduledApps.Store("partner", "partner")
	if err := builder.Persist(&module.BuildResult{Module: builder.module}); err == nil {
		t.Fatal("expected persist failure from supersede")
	}
	if _, loaded := appSettingScheduledApps.Load("partner"); loaded {
		t.Fatal("supersede persist failure should release schedule")
	}

	// validate failure → release: claim NeedInject, then swap parser results to duplicates before validate
	// by using a build plugin that returns two AppSettings while prebuild stayed empty.
	resetAppSettingScheduledAppsForTest()
	db3, err := gorm.Open(sqlite.Open(appSettingMemoryDSN(t, "fd-validate-fail")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db3.AutoMigrate(&meta.Application{}, &meta.Module{}, &meta.Model{}); err != nil {
		t.Fatal(err)
	}
	testScope3 := newBuilderTestScope()
	testScope3.session = &scope.Session{DB: db3}
	tmpVal := t.TempDir()
	entryVal := filepath.Join(tmpVal, "entry.ts")
	if err := os.WriteFile(entryVal, []byte("export const ok = 1;\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	modulesVal := filepath.Join(tmpVal, "modules")
	if err := os.MkdirAll(modulesVal, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modulesVal, "tsconfig.json"), []byte(`{"compilerOptions":{"baseUrl":"."}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	testScope3.cfg.ModulesPath = modulesVal
	testScope3.cfg.DistPath = filepath.Join(tmpVal, "dist")
	hand := filepath.Join(modulesVal, "partner/service/models/app_setting.ts")
	virt := filepath.Join(modulesVal, "partner/service/models/__generated__/app_setting.ts")
	dupes := []*parser.ParserResult{
		{Path: hand, Model: &meta.Model{Name: "AppSetting", Path: hand, Application: "partner"}},
		{Path: virt, Model: &meta.Model{Name: "AppSetting", Path: virt, Application: "partner"}},
	}
	buildPlugVal := &stickyAppSettingParserPlugin{results: dupes}
	builder = &ModuleBuilder{
		runtimeScope: testScope3,
		module: &meta.Module{
			Name: "partner", Path: filepath.Join(modulesVal, "partner"),
			ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
		},
		buildPlugin: buildPlugVal, prebuildPlugin: &stubEsbPlugin{name: "prebuild"},
		entryPoint: entryVal, outFileName: "index.js",
	}
	if _, err := builder.BuildWithoutPersist(); err == nil || !strings.Contains(err.Error(), appSettingDuplicateCode) {
		t.Fatalf("expected validate duplicate, got %v", err)
	}
	if _, loaded := appSettingScheduledApps.Load("partner"); loaded {
		t.Fatal("validate failure should release")
	}

	// build failure → release (non-empty entry that esbuild rejects after inject).
	resetAppSettingScheduledAppsForTest()
	db4, err := gorm.Open(sqlite.Open(appSettingMemoryDSN(t, "fd-build-fail")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db4.AutoMigrate(&meta.Application{}, &meta.Module{}, &meta.Model{}); err != nil {
		t.Fatal(err)
	}
	testScope4 := newBuilderTestScope()
	testScope4.session = &scope.Session{DB: db4}
	tmp := t.TempDir()
	entry := filepath.Join(tmp, "entry.ts")
	if err := os.WriteFile(entry, []byte("export const ok = 1;\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	modulesDir := filepath.Join(tmp, "modules")
	if err := os.MkdirAll(modulesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modulesDir, "tsconfig.json"), []byte(`{"compilerOptions":{"baseUrl":"."}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	testScope4.cfg.ModulesPath = modulesDir
	testScope4.cfg.DistPath = filepath.Join(tmp, "dist")
	buildPlug := &stubEsbPlugin{name: "build", getParserResultsErr: errors.New("parser boom")}
	builder = &ModuleBuilder{
		runtimeScope: testScope4,
		module: &meta.Module{
			Name: "partner", Path: filepath.Join(modulesDir, "partner"),
			ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
		},
		buildPlugin: buildPlug, prebuildPlugin: &stubEsbPlugin{name: "prebuild"},
		entryPoint: entry, outFileName: "index.js",
	}
	if _, err := builder.BuildWithoutPersist(); err == nil {
		t.Fatal("expected build parser error")
	}
	if _, loaded := appSettingScheduledApps.Load("partner"); loaded {
		t.Fatal("build failure should release")
	}
}

func TestBundle_AppSettingFailurePaths(t *testing.T) {
	resetAppSettingScheduledAppsForTest()
	testScope := newBuilderTestScope()
	// prebuild failure (bad entry)
	builder := &ModuleBuilder{
		runtimeScope: testScope,
		module:       &meta.Module{Name: "partner", Path: "/virtual/modules/partner", ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts"},
		buildPlugin:  &stubEsbPlugin{name: "build"}, prebuildPlugin: &stubEsbPlugin{name: "prebuild"},
		entryPoint: "/no/such/entry.ts",
	}
	if _, err := builder.Bundle(); err == nil {
		t.Fatal("expected prebuild failure")
	}

	resetAppSettingScheduledAppsForTest()
	dbFD, err := gorm.Open(sqlite.Open(appSettingMemoryDSN(t, "fd-bundle-as-inject")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := dbFD.AutoMigrate(&meta.Model{}); err != nil {
		t.Fatal(err)
	}
	if err := dbFD.Create(&meta.Model{
		BaseModel:   meta.BaseModel{Id: sql.NullString{String: "fd-hand", Valid: true}},
		Name:        "FieldDefault",
		Path:        "/virtual/modules/partner/service/models/field_default.ts",
		Application: "partner",
	}).Error; err != nil {
		t.Fatal(err)
	}
	scopeFD := newBuilderTestScope()
	scopeFD.session = &scope.Session{DB: dbFD}
	builder = &ModuleBuilder{
		runtimeScope: scopeFD,
		module:       &meta.Module{Name: "partner", ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts"},
		buildPlugin:  &stubEsbPlugin{name: "build"}, prebuildPlugin: &stubEsbPlugin{name: "prebuild"},
		entryPoint: "",
	}
	bundleErr, err := builder.Bundle()
	_ = bundleErr
	if err == nil {
		t.Fatal("expected AppSetting inject failure after FieldDefault skip")
	}
	if !strings.Contains(err.Error(), "AppSetting") {
		t.Fatalf("expected AppSetting inject error, got %v", err)
	}

	db, err := gorm.Open(sqlite.Open(appSettingMemoryDSN(t, "fd-bundle-upd")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&meta.Model{}); err != nil {
		t.Fatal(err)
	}
	parentPath := "/virtual/modules/partner/service/models/parent.ts"
	latestPath := "/virtual/modules/partner/service/models/latest.ts"
	childPath := "/virtual/modules/partner/service/models/child.ts"
	if err := db.Create(&meta.Model{BaseModel: meta.BaseModel{Id: sql.NullString{String: "aaa", Valid: true}}, Name: "X", Path: parentPath}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&meta.Model{BaseModel: meta.BaseModel{Id: sql.NullString{String: "zzz", Valid: true}}, Name: "X", Path: latestPath}).Error; err != nil {
		t.Fatal(err)
	}
	scopeUpd := newBuilderTestScope()
	scopeUpd.session = &scope.Session{DB: db}
	pre := &stubEsbPlugin{name: "prebuild", parserResults: []*parser.ParserResult{{
		Path: childPath, RawContent: "export default class X {}",
		Model: &meta.Model{Name: "X", Path: childPath, Extends: parentPath},
	}}}
	builder = &ModuleBuilder{
		runtimeScope: scopeUpd,
		module:       &meta.Module{Name: "partner", Path: "/virtual/modules/partner", ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts"},
		buildPlugin:  &stubEsbPlugin{name: "build"}, prebuildPlugin: pre,
		entryPoint: "", tsParser: fixedParser{err: errors.New("reparse boom")}, tsPathAlias: map[string]string{},
	}
	if _, err := builder.Bundle(); err == nil {
		t.Fatal("expected update failure in Bundle")
	}

	// build failure in Bundle
	resetAppSettingScheduledAppsForTest()
	db2, err := gorm.Open(sqlite.Open(appSettingMemoryDSN(t, "fd-bundle-build")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db2.AutoMigrate(&meta.Model{}); err != nil {
		t.Fatal(err)
	}
	tmp := t.TempDir()
	entry := filepath.Join(tmp, "entry.ts")
	if err := os.WriteFile(entry, []byte("export const ok = 1;\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	modulesDir := filepath.Join(tmp, "modules")
	if err := os.MkdirAll(modulesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modulesDir, "tsconfig.json"), []byte(`{"compilerOptions":{"baseUrl":"."}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	scopeBuild := newBuilderTestScope()
	scopeBuild.session = &scope.Session{DB: db2}
	scopeBuild.cfg.ModulesPath = modulesDir
	scopeBuild.cfg.DistPath = filepath.Join(tmp, "dist")
	builder = &ModuleBuilder{
		runtimeScope: scopeBuild,
		module: &meta.Module{
			Name: "partner", Path: filepath.Join(modulesDir, "partner"),
			ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
		},
		buildPlugin:    &stubEsbPlugin{name: "build", getParserResultsErr: errors.New("parser boom")},
		prebuildPlugin: &stubEsbPlugin{name: "prebuild"},
		entryPoint:     entry, outFileName: "index.js",
	}
	if _, err := builder.Bundle(); err == nil {
		t.Fatal("expected build failure in Bundle")
	}
}
