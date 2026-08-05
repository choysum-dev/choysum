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

// bareEsbPlugin implements EsbPlugin without entry-import / virtual-source helpers.
type bareEsbPlugin struct{ name string }

func (p bareEsbPlugin) DefinePlugins(_ scope.Scope, _ jsexecutor.ScriptExecutor, _ *meta.Module, _ ...esbplugins.EsbPluginOptions) []api.Plugin {
	return []api.Plugin{{Name: p.name}}
}
func (p bareEsbPlugin) GetParserResults() ([]*parser.ParserResult, error) { return nil, nil }
func (p bareEsbPlugin) SetParserResults([]*parser.ParserResult) error     { return nil }

// stickyParserPlugin ignores SetParserResults so build can return fixed duplicates for validate.
type stickyParserPlugin struct {
	results []*parser.ParserResult
}

func (p *stickyParserPlugin) DefinePlugins(_ scope.Scope, _ jsexecutor.ScriptExecutor, _ *meta.Module, options ...esbplugins.EsbPluginOptions) []api.Plugin {
	for _, opt := range options {
		if opt != nil {
			opt(p)
		}
	}
	return []api.Plugin{{Name: "sticky", Setup: func(api.PluginBuild) {}}}
}
func (p *stickyParserPlugin) GetParserResults() ([]*parser.ParserResult, error) {
	return p.results, nil
}
func (p *stickyParserPlugin) SetParserResults([]*parser.ParserResult) error { return nil }
func (p *stickyParserPlugin) SetEntryPointImports([]string)                 {}
func (p *stickyParserPlugin) RegisterVirtualSource(string, string)          {}

func TestFieldDefaultHelpers_EdgeCases(t *testing.T) {
	if isGeneratedFieldDefaultPath("") || isGeneratedFieldDefaultPath("   ") {
		t.Fatal("empty path should not be generated")
	}
	if !isGeneratedFieldDefaultPath(fieldDefaultGeneratedRelPath) {
		t.Fatal("relative generated path should match")
	}

	out := fieldDefaultsIn([]*parser.ParserResult{
		nil,
		{Path: "/x", Model: nil},
		{Path: "/other/mod/service/models/field_default.ts", Model: &meta.Model{Name: "FieldDefault", Path: "/other/mod/service/models/field_default.ts"}},
		{Path: "/m/service/models/field_default.ts", Model: &meta.Model{Name: "FieldDefault", Path: "/m/service/models/field_default.ts", Abstract: true}},
		{Path: "/m/service/models/field_default.ts", Model: &meta.Model{Name: "Partner", Path: "/m/service/models/partner.ts"}},
		{Path: "/m/service/models/field_default.ts", Model: &meta.Model{Name: "FieldDefault", Path: "/m/service/models/field_default.ts"}},
	}, "/m")
	if len(out) != 1 || out[0].Name != "FieldDefault" {
		t.Fatalf("fieldDefaultsIn filter = %#v", out)
	}

	if fieldDefaultSameModule(nil, nil) {
		t.Fatal("nil module")
	}
	mod := &meta.Module{Path: "/m", BaseModel: meta.BaseModel{Id: sql.NullString{String: "mid", Valid: true}}}
	if !fieldDefaultSameModule([]*meta.Model{
		nil,
		{ModuleId: sql.NullString{String: "mid", Valid: true}},
	}, mod) {
		t.Fatal("expected module id match")
	}
	if !fieldDefaultSameModule([]*meta.Model{
		{Path: "/m/service/models/field_default.ts"},
	}, &meta.Module{Path: "/m"}) {
		t.Fatal("expected path match")
	}
	if fieldDefaultSameModule([]*meta.Model{{Path: "/other/x.ts"}}, &meta.Module{Path: "/m"}) {
		t.Fatal("expected path mismatch")
	}

	hand := handwrittenFieldDefaults([]*meta.Model{nil, {Path: fieldDefaultGeneratedPath("/m")}})
	if len(hand) != 0 {
		t.Fatalf("handwritten filter: %#v", hand)
	}
	gen := generatedFieldDefaults([]*meta.Model{nil, {Path: "/m/service/models/field_default.ts"}})
	if len(gen) != 0 {
		t.Fatalf("generated filter: %#v", gen)
	}
}

func TestDbLoadFieldDefaults_GuardsAndError(t *testing.T) {
	if models, err := (*ModuleBuilder)(nil).dbLoadFieldDefaults("partner"); models != nil || err != nil {
		t.Fatalf("nil builder: %#v %v", models, err)
	}
	builder, db := newFieldDefaultTestBuilder(t, &meta.Module{Name: "p", Path: "/m", ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts"})
	if models, err := builder.dbLoadFieldDefaults(""); models != nil || err != nil {
		t.Fatalf("empty app: %#v %v", models, err)
	}
	builder.runtimeScope = nil
	if models, err := builder.dbLoadFieldDefaults("partner"); models != nil || err != nil {
		t.Fatalf("nil scope: %#v %v", models, err)
	}

	builder, db = newFieldDefaultTestBuilder(t, &meta.Module{Name: "p", Path: "/m", ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts"})
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	_ = sqlDB.Close()
	if _, err := builder.dbLoadFieldDefaults("partner"); err == nil {
		t.Fatal("expected find error on closed db")
	}
	if _, err := builder.decideFieldDefaultPlan(nil); err == nil || !strings.Contains(err.Error(), "load FieldDefault") {
		t.Fatalf("expected decide load error, got %v", err)
	}
}

func TestReleaseFieldDefaultSchedule_NilBuilder(t *testing.T) {
	(*ModuleBuilder)(nil).releaseFieldDefaultSchedule()
}

func TestDecideFieldDefaultPlan_MoreBranches(t *testing.T) {
	if plan, err := (*ModuleBuilder)(nil).decideFieldDefaultPlan(nil); err != nil || plan.NeedInject {
		t.Fatalf("nil builder: %+v %v", plan, err)
	}
	builder, _ := newFieldDefaultTestBuilder(t, &meta.Module{
		Name: "partner", Path: "/virtual/modules/partner",
		ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
	})
	handA := "/virtual/modules/partner/service/models/fd_a.ts"
	handB := "/virtual/modules/partner/service/models/fd_b.ts"
	_, err := builder.decideFieldDefaultPlan([]*parser.ParserResult{
		{Path: handA, Model: &meta.Model{Name: "FieldDefault", Path: handA}},
		{Path: handB, Model: &meta.Model{Name: "FieldDefault", Path: handB}},
	})
	if err == nil || !strings.Contains(err.Error(), fieldDefaultDuplicateCode) {
		t.Fatalf("expected local duplicate, got %v", err)
	}

	// Local handwritten, no DB virtual → no-op plan.
	plan, err := builder.decideFieldDefaultPlan([]*parser.ParserResult{
		{Path: handA, Model: &meta.Model{Name: "FieldDefault", Path: handA}},
	})
	if err != nil || plan.NeedInject || plan.SupersedeVirtual {
		t.Fatalf("expected no-op for local hand only, got %+v %v", plan, err)
	}

	// Local generated (no handwritten) → skip inject.
	virt := fieldDefaultGeneratedPath(builder.module.Path)
	plan, err = builder.decideFieldDefaultPlan([]*parser.ParserResult{
		{Path: virt, Model: &meta.Model{Name: "FieldDefault", Path: virt}},
	})
	if err != nil || plan.NeedInject {
		t.Fatalf("local generated should skip inject, got %+v %v", plan, err)
	}

	// Same-module reclaim.
	resetFieldDefaultScheduledAppsForTest()
	plan, err = builder.decideFieldDefaultPlan(nil)
	if err != nil || !plan.NeedInject {
		t.Fatalf("first inject: %+v %v", plan, err)
	}
	plan2, err := builder.decideFieldDefaultPlan(nil)
	if err != nil || !plan2.NeedInject || plan2.scheduledApp != "partner" {
		t.Fatalf("reclaim: %+v %v", plan2, err)
	}
}

func TestApplyAndPlanInject_ErrorPaths(t *testing.T) {
	if err := (*ModuleBuilder)(nil).applyFieldDefaultInject(FieldDefaultPlan{NeedInject: true}); err != nil {
		t.Fatal(err)
	}
	builder, _ := newFieldDefaultTestBuilder(t, &meta.Module{
		Name: "partner", ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
	})
	if err := builder.applyFieldDefaultInject(FieldDefaultPlan{NeedInject: true}); err == nil || !strings.Contains(err.Error(), "module path") {
		t.Fatalf("expected path error, got %v", err)
	}

	// Empty modulesPath in config → derive from module.Path; still register virtual source.
	builder.module.Path = "/virtual/modules/partner"
	builder.runtimeScope.(*builderTestScope).cfg.ModulesPath = ""
	if err := builder.applyFieldDefaultInject(FieldDefaultPlan{NeedInject: true}); err != nil {
		t.Fatalf("inject with empty modulesPath: %v", err)
	}
	stub := builder.buildPlugin.(*stubEsbPlugin)
	if len(stub.virtualSources) == 0 {
		t.Fatal("expected virtual source registration")
	}

	// buildPlugin without SetEntryPointImports / RegisterVirtualSource still succeeds.
	builder.buildPlugin = bareEsbPlugin{name: "bare"}
	if err := builder.applyFieldDefaultInject(FieldDefaultPlan{NeedInject: true}); err != nil {
		t.Fatalf("bare plugin: %v", err)
	}

	builder, _ = newFieldDefaultTestBuilder(t, &meta.Module{
		Name: "partner", Path: "/virtual/modules/partner",
		ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
	})
	err := builder.planAndInjectFieldDefault(module.WithParserResults(&module.BuildResult{Module: builder.module}, []*parser.ParserResult{
		{Path: "/virtual/modules/partner/a.ts", Model: &meta.Model{Name: "FieldDefault", Path: "/virtual/modules/partner/a.ts"}},
		{Path: "/virtual/modules/partner/b.ts", Model: &meta.Model{Name: "FieldDefault", Path: "/virtual/modules/partner/b.ts"}},
	}))
	if err == nil || !strings.Contains(err.Error(), fieldDefaultDuplicateCode) {
		t.Fatalf("planAndInject decide error: %v", err)
	}

	builder.fieldDefaultPlan = FieldDefaultPlan{}
	resetFieldDefaultScheduledAppsForTest()
	builder.module.Path = ""
	builder.module.ApplicationStr = "partner"
	builder.module.ServiceEntryPoint = "service/index.ts"
	builder.module.Name = "partner"
	err = builder.planAndInjectFieldDefault(module.WithParserResults(&module.BuildResult{Module: builder.module}, nil))
	if err == nil || !strings.Contains(err.Error(), "injecting FieldDefault") {
		t.Fatalf("planAndInject inject error: %v", err)
	}
	if _, loaded := fieldDefaultScheduledApps.Load("partner"); loaded {
		t.Fatal("schedule should be released after inject failure")
	}
}

func TestValidateFieldDefault_Guards(t *testing.T) {
	if err := (*ModuleBuilder)(nil).validateFieldDefault(nil); err != nil {
		t.Fatal(err)
	}
	builder, _ := newFieldDefaultTestBuilder(t, &meta.Module{Name: "core", Path: "/m", ApplicationStr: "core"})
	if err := builder.validateFieldDefault(&module.BuildResult{Module: builder.module}); err != nil {
		t.Fatal(err)
	}
	builder.module.ApplicationStr = ""
	if err := builder.validateFieldDefault(&module.BuildResult{Module: builder.module}); err != nil {
		t.Fatal(err)
	}
	builder.module.ApplicationStr = "partner"
	if err := builder.validateFieldDefault(module.WithParserResults(&module.BuildResult{Module: builder.module}, nil)); err != nil {
		t.Fatal(err)
	}
}

func TestSupersedeVirtualFieldDefaults_GuardsAndDependents(t *testing.T) {
	if err := (*ModuleBuilder)(nil).supersedeVirtualFieldDefaults(); err != nil {
		t.Fatal(err)
	}
	builder, db := newFieldDefaultTestBuilder(t, &meta.Module{
		Name: "partner_bank", Path: "/virtual/modules/partner_bank",
		ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
	})
	if err := builder.supersedeVirtualFieldDefaults(); err != nil {
		t.Fatal(err)
	}
	builder.fieldDefaultPlan = FieldDefaultPlan{SupersedeVirtual: true}
	builder.runtimeScope = nil
	if err := builder.supersedeVirtualFieldDefaults(); err != nil {
		t.Fatal(err)
	}
	builder, db = newFieldDefaultTestBuilder(t, &meta.Module{
		Name: "partner_bank", Path: "/virtual/modules/partner_bank",
		ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
	})
	builder.fieldDefaultPlan = FieldDefaultPlan{SupersedeVirtual: true}
	builder.module.ApplicationStr = ""
	if err := builder.supersedeVirtualFieldDefaults(); err != nil {
		t.Fatal(err)
	}
	builder.module.ApplicationStr = "partner"

	// Generated row with empty id → filtered out (no delete).
	if err := db.Create(&meta.RawModel{
		BaseModel:   meta.BaseModel{Id: sql.NullString{String: "   ", Valid: true}},
		Name:        "FieldDefault",
		Path:        "/virtual/modules/partner/service/models/__generated__/field_default.ts",
		Application: "partner",
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := builder.supersedeVirtualFieldDefaults(); err != nil {
		t.Fatal(err)
	}

	// Session with nil DB handle (guard before Find).
	builder.runtimeScope = &builderTestScope{session: &scope.Session{DB: nil}}
	builder.fieldDefaultPlan = FieldDefaultPlan{SupersedeVirtual: true}
	builder.module.ApplicationStr = "partner"
	if err := builder.supersedeVirtualFieldDefaults(); err != nil {
		t.Fatal(err)
	}

	// Full dependent tree delete.
	builder, db = newFieldDefaultTestBuilder(t, &meta.Module{
		Name: "partner_bank", Path: "/virtual/modules/partner_bank",
		ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
	})
	builder.fieldDefaultPlan = FieldDefaultPlan{SupersedeVirtual: true}
	virtID := "virt-dep"
	if err := db.Create(&meta.RawModel{
		BaseModel: meta.BaseModel{Id: sql.NullString{String: virtID, Valid: true}},
		Name:      "FieldDefault", Path: "/virtual/modules/partner/service/models/__generated__/field_default.ts", Application: "partner",
	}).Error; err != nil {
		t.Fatal(err)
	}
	fieldID := "fld1"
	svcID := "svc1"
	decModel := "dec-m"
	decField := "dec-f"
	decSvc := "dec-s"
	if err := db.Create(&meta.RawField{BaseModel: meta.BaseModel{Id: sql.NullString{String: fieldID, Valid: true}}, Name: "Model", ModelId: sql.NullString{String: virtID, Valid: true}}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&meta.RawService{BaseModel: meta.BaseModel{Id: sql.NullString{String: svcID, Valid: true}}, Name: "Get", ModelId: sql.NullString{String: virtID, Valid: true}}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&meta.RawDecorator{BaseModel: meta.BaseModel{Id: sql.NullString{String: decModel, Valid: true}}, Name: "Model", ModelId: sql.NullString{String: virtID, Valid: true}}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&meta.RawDecorator{BaseModel: meta.BaseModel{Id: sql.NullString{String: decField, Valid: true}}, Name: "Field", FieldId: sql.NullString{String: fieldID, Valid: true}}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&meta.RawDecorator{BaseModel: meta.BaseModel{Id: sql.NullString{String: decSvc, Valid: true}}, Name: "Service", ServiceId: sql.NullString{String: svcID, Valid: true}}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&meta.RawArgument{BaseModel: meta.BaseModel{Id: sql.NullString{String: "arg1", Valid: true}}, DecoratorId: sql.NullString{String: decModel, Valid: true}}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&meta.RawTypeParameter{BaseModel: meta.BaseModel{Id: sql.NullString{String: "tp1", Valid: true}}, Name: "T", ServiceId: sql.NullString{String: svcID, Valid: true}}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&meta.RawParameter{BaseModel: meta.BaseModel{Id: sql.NullString{String: "p1", Valid: true}}, Name: "x", ServiceId: sql.NullString{String: svcID, Valid: true}}).Error; err != nil {
		t.Fatal(err)
	}
	if err := builder.supersedeVirtualFieldDefaults(); err != nil {
		t.Fatalf("supersede dependents: %v", err)
	}
	var left int64
	if err := db.Unscoped().Model(&meta.RawModel{}).Where("id = ?", virtID).Count(&left).Error; err != nil || left != 0 {
		t.Fatalf("virt model left=%d err=%v", left, err)
	}
}

func TestSupersedeVirtualFieldDefaults_RecomputeError(t *testing.T) {
	builder, db := newFieldDefaultTestBuilder(t, &meta.Module{
		Name: "partner_bank", Path: "/virtual/modules/partner_bank",
		ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
	})
	builder.fieldDefaultPlan = FieldDefaultPlan{SupersedeVirtual: true}
	if err := db.Create(&meta.RawModel{
		BaseModel: meta.BaseModel{Id: sql.NullString{String: "virt-recompute", Valid: true}},
		Name:      "FieldDefault", Path: "/virtual/modules/partner/service/models/__generated__/field_default.ts", Application: "partner",
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Migrator().DropTable(&meta.Model{}); err != nil {
		t.Fatal(err)
	}
	if err := builder.supersedeVirtualFieldDefaults(); err == nil || !strings.Contains(err.Error(), "recompute FieldDefault after supersede") {
		t.Fatalf("expected recompute error, got %v", err)
	}
}

func TestSupersedeVirtualFieldDefaults_ErrorBranches(t *testing.T) {
	builder, db := newFieldDefaultTestBuilder(t, &meta.Module{
		Name: "partner_bank", Path: "/virtual/modules/partner_bank",
		ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
	})
	builder.fieldDefaultPlan = FieldDefaultPlan{SupersedeVirtual: true}
	if err := db.Create(&meta.RawModel{
		BaseModel: meta.BaseModel{Id: sql.NullString{String: "virt", Valid: true}},
		Name:      "FieldDefault", Path: "/virtual/modules/partner/service/models/__generated__/field_default.ts", Application: "partner",
	}).Error; err != nil {
		t.Fatal(err)
	}

	// Find error
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	_ = sqlDB.Close()
	if err := builder.supersedeVirtualFieldDefaults(); err == nil || !strings.Contains(err.Error(), "load FieldDefault rows") {
		t.Fatalf("expected load error, got %v", err)
	}

	// Pluck / delete errors via callbacks on a fresh DB.
	cases := []struct {
		name   string
		table  string
		phase  string // query|delete
		substr string
	}{
		{name: "service pluck", table: "meta_raw_service", phase: "query", substr: "load declaration services"},
		{name: "field pluck", table: "meta_raw_field", phase: "query", substr: "load declaration fields"},
		{name: "decorator pluck", table: "meta_raw_decorator", phase: "query", substr: "load declaration decorators"},
		{name: "argument delete", table: "meta_raw_argument", phase: "delete", substr: "delete declaration arguments"},
		{name: "decorator delete", table: "meta_raw_decorator", phase: "delete", substr: "delete declaration decorators"},
		{name: "type param delete", table: "meta_raw_type_parameter", phase: "delete", substr: "delete declaration type parameters"},
		{name: "parameter delete", table: "meta_raw_parameter", phase: "delete", substr: "delete declaration parameters"},
		{name: "service delete", table: "meta_raw_service", phase: "delete", substr: "delete declaration services"},
		{name: "field delete", table: "meta_raw_field", phase: "delete", substr: "delete declaration fields"},
		{name: "model delete", table: "meta_raw_model", phase: "delete", substr: "delete declaration models"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b, gdb := newFieldDefaultTestBuilder(t, &meta.Module{
				Name: "partner_bank", Path: "/virtual/modules/partner_bank",
				ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
			})
			b.fieldDefaultPlan = FieldDefaultPlan{SupersedeVirtual: true}
			virtID := "v-" + tc.name
			if err := gdb.Create(&meta.RawModel{
				BaseModel: meta.BaseModel{Id: sql.NullString{String: virtID, Valid: true}},
				Name:      "FieldDefault", Path: "/virtual/modules/partner/service/models/__generated__/field_default.ts", Application: "partner",
			}).Error; err != nil {
				t.Fatal(err)
			}
			fieldID := "f-" + tc.name
			svcID := "s-" + tc.name
			decID := "d-" + tc.name
			if err := gdb.Create(&meta.RawField{BaseModel: meta.BaseModel{Id: sql.NullString{String: fieldID, Valid: true}}, Name: "Model", ModelId: sql.NullString{String: virtID, Valid: true}}).Error; err != nil {
				t.Fatal(err)
			}
			if err := gdb.Create(&meta.RawService{BaseModel: meta.BaseModel{Id: sql.NullString{String: svcID, Valid: true}}, Name: "Get", ModelId: sql.NullString{String: virtID, Valid: true}}).Error; err != nil {
				t.Fatal(err)
			}
			if err := gdb.Create(&meta.RawDecorator{BaseModel: meta.BaseModel{Id: sql.NullString{String: decID, Valid: true}}, Name: "Model", ModelId: sql.NullString{String: virtID, Valid: true}}).Error; err != nil {
				t.Fatal(err)
			}
			if err := gdb.Create(&meta.RawArgument{BaseModel: meta.BaseModel{Id: sql.NullString{String: "a-" + tc.name, Valid: true}}, DecoratorId: sql.NullString{String: decID, Valid: true}}).Error; err != nil {
				t.Fatal(err)
			}
			if err := gdb.Create(&meta.RawTypeParameter{BaseModel: meta.BaseModel{Id: sql.NullString{String: "t-" + tc.name, Valid: true}}, Name: "T", ServiceId: sql.NullString{String: svcID, Valid: true}}).Error; err != nil {
				t.Fatal(err)
			}
			if err := gdb.Create(&meta.RawParameter{BaseModel: meta.BaseModel{Id: sql.NullString{String: "p-" + tc.name, Valid: true}}, Name: "x", ServiceId: sql.NullString{String: svcID, Valid: true}}).Error; err != nil {
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
			err := b.supersedeVirtualFieldDefaults()
			if err == nil || !strings.Contains(err.Error(), tc.substr) {
				t.Fatalf("expected %q in error, got %v", tc.substr, err)
			}
		})
	}
}

func TestBuildLifecycle_FieldDefaultInjectAndRelease(t *testing.T) {
	resetFieldDefaultScheduledAppsForTest()
	db, err := gorm.Open(sqlite.Open(fieldDefaultMemoryDSN(t, "fd-lifecycle")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&meta.Application{}, &meta.Module{},
		&meta.RawModel{}, &meta.RawField{}, &meta.RawService{}, &meta.RawDecorator{}, &meta.RawArgument{},
		&meta.RawParameter{}, &meta.RawTypeParameter{},
		&meta.Model{}, &meta.Field{}, &meta.Service{}, &meta.Decorator{}, &meta.Argument{},
		&meta.Parameter{}, &meta.TypeParameter{},
	); err != nil {
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
	if builder.fieldDefaultInjectPath == "" || !builder.fieldDefaultPlan.NeedInject {
		t.Fatalf("expected inject plan, path=%q plan=%+v", builder.fieldDefaultInjectPath, builder.fieldDefaultPlan)
	}
	if _, ok := buildPlugin.virtualSources[builder.fieldDefaultInjectPath]; !ok {
		t.Fatal("expected virtual source on build plugin")
	}
	// buildOptions(false) should include inject path via SetEntryPointImports already called;
	// also exercise append branch by calling buildOptions directly.
	_ = builder.buildOptions(false)

	if err := builder.Persist(result); err != nil {
		t.Fatalf("Persist: %v", err)
	}
	if builder.fieldDefaultPlan.scheduledApp != "" {
		t.Fatal("expected schedule cleared after Persist")
	}

	// Bundle releases schedule via defer.
	resetFieldDefaultScheduledAppsForTest()
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
	if builder2.fieldDefaultPlan.scheduledApp != "" {
		t.Fatal("bundle should clear scheduledApp")
	}
}

func TestBuildWithoutPersist_ReleasesOnFailures(t *testing.T) {
	resetFieldDefaultScheduledAppsForTest()
	db, err := gorm.Open(sqlite.Open(fieldDefaultMemoryDSN(t, "fd-fail")), &gorm.Config{})
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

	// planAndInject failure (empty module path after NeedInject claim path uses empty Path).
	mod := &meta.Module{Name: "partner", ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts"}
	builder = &ModuleBuilder{
		runtimeScope: testScope, module: mod,
		buildPlugin: &stubEsbPlugin{name: "build"}, prebuildPlugin: &stubEsbPlugin{name: "prebuild"},
		entryPoint: "",
	}
	if _, err := builder.BuildWithoutPersist(); err == nil {
		t.Fatal("expected inject path error")
	}
	if _, loaded := fieldDefaultScheduledApps.Load("partner"); loaded {
		t.Fatal("claim should be released")
	}

	// updatePrebuildResult failure → release (NeedInject claims, then extends refresh fails).
	resetFieldDefaultScheduledAppsForTest()
	dbUpd, err := gorm.Open(sqlite.Open(fieldDefaultMemoryDSN(t, "fd-upd-fail")), &gorm.Config{})
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
	if _, loaded := fieldDefaultScheduledApps.Load("partner"); loaded {
		t.Fatal("claim should be released after update failure")
	}

	// Persist failure releases.
	resetFieldDefaultScheduledAppsForTest()
	db2, err := gorm.Open(sqlite.Open(fieldDefaultMemoryDSN(t, "fd-persist-fail")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	testScope2 := newBuilderTestScope()
	testScope2.session = &scope.Session{DB: db2}
	builder = &ModuleBuilder{
		runtimeScope: testScope2,
		module:       &meta.Module{Name: "partner", Path: "/virtual/modules/partner", ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts"},
		buildPlugin:  &stubEsbPlugin{name: "build"}, prebuildPlugin: &stubEsbPlugin{name: "prebuild"},
		entryPoint:       "",
		fieldDefaultPlan: FieldDefaultPlan{NeedInject: true, scheduledApp: "partner"},
	}
	fieldDefaultScheduledApps.Store("partner", "partner")
	if err := builder.Persist(&module.BuildResult{Module: builder.module}); err == nil {
		t.Fatal("expected persist failure without migrated tables")
	}
	if _, loaded := fieldDefaultScheduledApps.Load("partner"); loaded {
		t.Fatal("persist failure should release schedule")
	}

	// Persist fails inside supersedeVirtualFieldDefaults (closed DB) and still releases.
	resetFieldDefaultScheduledAppsForTest()
	dbSupersede, err := gorm.Open(sqlite.Open(fieldDefaultMemoryDSN(t, "fd-persist-supersede-fail")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := dbSupersede.AutoMigrate(&meta.RawModel{}, &meta.Model{}); err != nil {
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
		entryPoint:       "",
		fieldDefaultPlan: FieldDefaultPlan{SupersedeVirtual: true, scheduledApp: "partner"},
	}
	fieldDefaultScheduledApps.Store("partner", "partner")
	if err := builder.Persist(&module.BuildResult{Module: builder.module}); err == nil {
		t.Fatal("expected persist failure from supersede")
	}
	if _, loaded := fieldDefaultScheduledApps.Load("partner"); loaded {
		t.Fatal("supersede persist failure should release schedule")
	}

	// validate failure → release: claim NeedInject, then swap parser results to duplicates before validate
	// by using a build plugin that returns two FieldDefaults while prebuild stayed empty.
	resetFieldDefaultScheduledAppsForTest()
	db3, err := gorm.Open(sqlite.Open(fieldDefaultMemoryDSN(t, "fd-validate-fail")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db3.AutoMigrate(&meta.Application{}, &meta.Module{}); err != nil {
		t.Fatal(err)
	}
	if err := meta.EnsureDualStoreTables(db3); err != nil {
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
	hand := filepath.Join(modulesVal, "partner/service/models/field_default.ts")
	virt := filepath.Join(modulesVal, "partner/service/models/__generated__/field_default.ts")
	dupes := []*parser.ParserResult{
		{Path: hand, Model: &meta.Model{Name: "FieldDefault", Path: hand, Application: "partner"}},
		{Path: virt, Model: &meta.Model{Name: "FieldDefault", Path: virt, Application: "partner"}},
	}
	buildPlugVal := &stickyParserPlugin{results: dupes}
	builder = &ModuleBuilder{
		runtimeScope: testScope3,
		module: &meta.Module{
			Name: "partner", Path: filepath.Join(modulesVal, "partner"),
			ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
		},
		buildPlugin: buildPlugVal, prebuildPlugin: &stubEsbPlugin{name: "prebuild"},
		entryPoint: entryVal, outFileName: "index.js",
	}
	if _, err := builder.BuildWithoutPersist(); err == nil || !strings.Contains(err.Error(), fieldDefaultDuplicateCode) {
		t.Fatalf("expected validate duplicate, got %v", err)
	}
	if _, loaded := fieldDefaultScheduledApps.Load("partner"); loaded {
		t.Fatal("validate failure should release")
	}

	// build failure → release (non-empty entry that esbuild rejects after inject).
	resetFieldDefaultScheduledAppsForTest()
	db4, err := gorm.Open(sqlite.Open(fieldDefaultMemoryDSN(t, "fd-build-fail")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db4.AutoMigrate(&meta.Application{}, &meta.Module{}); err != nil {
		t.Fatal(err)
	}
	if err := meta.EnsureDualStoreTables(db4); err != nil {
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
	if _, loaded := fieldDefaultScheduledApps.Load("partner"); loaded {
		t.Fatal("build failure should release")
	}
}

func TestBundle_FailurePaths(t *testing.T) {
	resetFieldDefaultScheduledAppsForTest()
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

	resetFieldDefaultScheduledAppsForTest()
	builder = &ModuleBuilder{
		runtimeScope: testScope,
		module:       &meta.Module{Name: "partner", ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts"},
		buildPlugin:  &stubEsbPlugin{name: "build"}, prebuildPlugin: &stubEsbPlugin{name: "prebuild"},
		entryPoint: "",
	}
	if _, err := builder.Bundle(); err == nil {
		t.Fatal("expected inject failure")
	}

	db, err := gorm.Open(sqlite.Open(fieldDefaultMemoryDSN(t, "fd-bundle-upd")), &gorm.Config{})
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
	resetFieldDefaultScheduledAppsForTest()
	db2, err := gorm.Open(sqlite.Open(fieldDefaultMemoryDSN(t, "fd-bundle-build")), &gorm.Config{})
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
