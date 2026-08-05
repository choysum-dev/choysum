// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package backendbuilder

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/internal/module/artifact/build/injectappmodel"
	module "github.com/choysum-dev/choysum/internal/module/artifact/result"
	"github.com/choysum-dev/choysum/internal/parser"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newInjectTestBuilder(t *testing.T, mod *meta.Module, results []*parser.ParserResult) (*ModuleBuilder, *stubEsbPlugin, *gorm.DB) {
	t.Helper()
	injectappmodel.ResetScheduledForTest()
	t.Cleanup(injectappmodel.ResetScheduledForTest)

	dsn := filepath.Join(t.TempDir(), "inject-cov.sqlite")
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := meta.EnsureDualStoreTables(db); err != nil {
		t.Fatal(err)
	}
	_ = db.AutoMigrate(&meta.Application{}, &meta.Module{})

	buildPlugin := &stubEsbPlugin{name: "build", parserResults: results, virtualSources: map[string]string{}}
	prebuildPlugin := &stubEsbPlugin{name: "prebuild", parserResults: results, virtualSources: map[string]string{}}
	testScope := newBuilderTestScope()
	testScope.session = &scope.Session{DB: db}
	builder := &ModuleBuilder{
		runtimeScope:   testScope,
		module:         mod,
		buildPlugin:    buildPlugin,
		prebuildPlugin: prebuildPlugin,
		entryPoint:     "", // skip esbuild; use plugin parser results
	}
	return builder, buildPlugin, db
}

func TestInjectHost_NilGuards(t *testing.T) {
	h := injectHost{}
	if h.Module() != nil || h.SessionDB() != nil || h.EntryPointImports() != nil {
		t.Fatal("nil builder host accessors")
	}
	if h.ModulesPath() != "" {
		t.Fatal("nil ModulesPath")
	}
	h.SetEntryPointImports([]string{"x"})
	h.RegisterVirtualSource("p", "c")

	b := &ModuleBuilder{}
	h2 := injectHost{b: b}
	if h2.SessionDB() != nil {
		t.Fatal("nil scope SessionDB")
	}
	_ = h2.ModulesPath()
	_ = h2.EntryPointImports()
	h2.SetEntryPointImports([]string{"y"})
	h2.RegisterVirtualSource("p2", "c2")

	if (*ModuleBuilder)(nil).ensureInjectSession() != nil {
		t.Fatal("nil ensureInjectSession")
	}
	(*ModuleBuilder)(nil).releaseInjectSchedules()
	b.releaseInjectSchedules()
}

func TestInjectAppModelsWrappers(t *testing.T) {
	if err := (*ModuleBuilder)(nil).injectAppModels(nil); err != nil {
		t.Fatal(err)
	}
	if err := (*ModuleBuilder)(nil).supersedeInjectAppModels(); err != nil {
		t.Fatal(err)
	}
	if err := (*ModuleBuilder)(nil).validateInjectAppModels(nil); err != nil {
		t.Fatal(err)
	}
	if err := (*ModuleBuilder)(nil).BundleInjectAppModels(nil); err != nil {
		t.Fatal(err)
	}

	mod := &meta.Module{
		Name: "partner", Path: "/virtual/modules/partner",
		ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
	}
	a := "/virtual/modules/partner/service/models/a.ts"
	bPath := "/virtual/modules/partner/service/models/b.ts"
	builder, buildPlugin, _ := newInjectTestBuilder(t, mod, nil)

	pre := module.WithParserResults(&module.BuildResult{Module: mod}, []*parser.ParserResult{
		{Path: a, Model: &meta.Model{Name: "FieldDefault", Path: a}},
		{Path: bPath, Model: &meta.Model{Name: "FieldDefault", Path: bPath}},
	})
	if err := builder.injectAppModels(pre); err == nil || !strings.Contains(err.Error(), "FIELD_DEFAULT_DUPLICATE") {
		t.Fatalf("expected inject error with release, got %v", err)
	}

	if err := builder.injectAppModels(nil); err != nil {
		t.Fatal(err)
	}
	builder.ensureInjectSession().SetPlan("AppSetting", injectappmodel.Plan{SupersedeInject: true})
	if err := builder.supersedeInjectAppModels(); err != nil {
		t.Fatal(err)
	}

	dupBuild := module.WithParserResults(&module.BuildResult{Module: mod}, []*parser.ParserResult{
		{Path: a, Model: &meta.Model{Name: "FieldDefault", Path: a}},
		{Path: bPath, Model: &meta.Model{Name: "FieldDefault", Path: bPath}},
	})
	if err := builder.validateInjectAppModels(dupBuild); err == nil {
		t.Fatal("expected validate duplicate error")
	}

	base := &meta.Module{
		Name: "base", Path: "/virtual/modules/base",
		ApplicationStr: "base", ServiceEntryPoint: "service/index.ts",
	}
	if err := builder.BundleInjectAppModels([]*meta.Module{base}); err != nil {
		t.Fatal(err)
	}
	if len(builder.ensureInjectSession().AllInjectPaths()) == 0 {
		t.Fatal("expected bundle inject paths")
	}

	host := injectHost{b: builder}
	if host.Module() != mod || host.SessionDB() == nil || host.ModulesPath() == "" {
		t.Fatal("live host accessors")
	}
	_ = host.EntryPointImports()
	host.SetEntryPointImports([]string{"/imp"})
	host.RegisterVirtualSource("/v.ts", "export {}")
	if buildPlugin.virtualSources["/v.ts"] == "" {
		t.Fatal("RegisterVirtualSource")
	}
}

func TestBuildOptions_IncludesInjectPaths(t *testing.T) {
	mod := &meta.Module{
		Name: "partner", Path: "/virtual/modules/partner",
		ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
	}
	builder, _, _ := newInjectTestBuilder(t, mod, nil)
	if err := builder.injectAppModels(nil); err != nil {
		t.Fatal(err)
	}
	if len(builder.injectSession.AllInjectPaths()) == 0 {
		t.Fatal("expected inject paths")
	}
	opts := builder.buildOptions(true)
	if opts == nil || len(opts.Plugins) == 0 {
		t.Fatal("expected build options with plugins")
	}
}

func TestPersistModuleModels_SupersedeFlushKeys(t *testing.T) {
	mod := &meta.Module{
		Name: "partner", Path: "/virtual/modules/partner",
		ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
	}
	mod.Id = sql.NullString{String: "mod-persist", Valid: true}
	builder, _, _ := newInjectTestBuilder(t, mod, nil)
	sess := builder.ensureInjectSession()
	sess.SetPlan("FieldDefault", injectappmodel.Plan{SupersedeInject: true})
	sess.SetPlan("AppSetting", injectappmodel.Plan{SupersedeInject: true})
	if err := builder.persistModuleModels(mod.Id.String, []*meta.Model{{
		Name: "Partner", Path: "/virtual/modules/partner/service/models/partner.ts", Application: "partner",
	}}); err != nil {
		t.Fatal(err)
	}
}

func TestValidate_InjectAppModelsError(t *testing.T) {
	mod := &meta.Module{
		Name: "partner", Path: "/virtual/modules/partner",
		ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
	}
	builder, _, _ := newInjectTestBuilder(t, mod, nil)
	a := "/virtual/modules/partner/service/models/fd_a.ts"
	bPath := "/virtual/modules/partner/service/models/fd_b.ts"
	// Same-name models with extends so inheritance checks pass, then inject validate fails.
	parent := &meta.Model{Name: "FieldDefault", Path: a}
	child := &meta.Model{Name: "FieldDefault", Path: bPath, Extends: a}
	result := module.WithParserResults(&module.BuildResult{Module: mod}, []*parser.ParserResult{
		{Path: a, Model: parent},
		{Path: bPath, Model: child},
	})
	if err := builder.validate(result); err == nil || !strings.Contains(err.Error(), "FIELD_DEFAULT_DUPLICATE") {
		t.Fatalf("expected validate inject duplicate, got %v", err)
	}
}

func TestPersist_SupersedeInjectError(t *testing.T) {
	mod := &meta.Module{
		Name: "partner", Path: "/virtual/modules/partner",
		ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
	}
	mod.Id = sql.NullString{String: "mod-sup-err", Valid: true}
	builder, _, db := newInjectTestBuilder(t, mod, nil)
	builder.ensureInjectSession().SetPlan("FieldDefault", injectappmodel.Plan{SupersedeInject: true})
	if err := meta.DropRawModelTable(db); err != nil {
		t.Fatal(err)
	}
	result := &module.BuildResult{Module: mod}
	if err := builder.persist(result); err == nil {
		t.Fatal("expected supersede error in persist")
	}
	if err := builder.Persist(result); err == nil {
		t.Fatal("expected Persist wrapper error + release")
	}
}

func TestPersistModuleModels_EmptyModuleID(t *testing.T) {
	builder := &ModuleBuilder{}
	if err := builder.persistModuleModels("  ", nil); err != nil {
		t.Fatal(err)
	}
}

func TestBuildWithoutPersist_PrebuildError(t *testing.T) {
	mod := &meta.Module{
		Name: "partner", Path: "/virtual/modules/partner",
		ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
	}
	builder, prebuildPlugin, _ := newInjectTestBuilder(t, mod, nil)
	// Force prebuild esbuild path then fail: set entryPoint to missing file.
	builder.entryPoint = filepath.Join(t.TempDir(), "missing-entry.ts")
	builder.prebuildPlugin = prebuildPlugin
	if _, err := builder.BuildWithoutPersist(); err == nil {
		t.Fatal("expected prebuild error")
	}
}

func TestPersistModuleModels_EmptyAppSkipsSupersedeKeys(t *testing.T) {
	mod := &meta.Module{
		Name: "partner", Path: "/virtual/modules/partner",
		ApplicationStr: "  ", ServiceEntryPoint: "service/index.ts",
	}
	mod.Id = sql.NullString{String: "mod-empty-app", Valid: true}
	builder, _, _ := newInjectTestBuilder(t, mod, nil)
	sess := builder.ensureInjectSession()
	sess.SetPlan("FieldDefault", injectappmodel.Plan{SupersedeInject: true})
	if err := builder.persistModuleModels(mod.Id.String, nil); err != nil {
		t.Fatal(err)
	}
}

func TestBuildWithoutPersist_InjectError(t *testing.T) {
	mod := &meta.Module{
		Name: "partner", Path: "/virtual/modules/partner",
		ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
	}
	a := "/virtual/modules/partner/service/models/a.ts"
	bPath := "/virtual/modules/partner/service/models/b.ts"
	results := []*parser.ParserResult{
		{Path: a, Model: &meta.Model{Name: "FieldDefault", Path: a}},
		{Path: bPath, Model: &meta.Model{Name: "FieldDefault", Path: bPath}},
	}
	builder, _, _ := newInjectTestBuilder(t, mod, results)
	if _, err := builder.BuildWithoutPersist(); err == nil {
		t.Fatal("expected injectAppModels error")
	}
}

func TestBuildWithoutPersist_BuildErrorReleases(t *testing.T) {
	mod := &meta.Module{
		Name: "partner", Path: "/virtual/modules/partner",
		ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
	}
	builder, buildPlugin, _ := newInjectTestBuilder(t, mod, nil)
	buildPlugin.getParserResultsErr = fmt.Errorf("forced build parser results failure")
	if _, err := builder.BuildWithoutPersist(); err == nil || !strings.Contains(err.Error(), "forced build parser results failure") {
		t.Fatalf("expected build parser results failure, got %v", err)
	}
}

func TestBundle_InjectError(t *testing.T) {
	mod := &meta.Module{
		Name: "partner", Path: "/virtual/modules/partner",
		ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
	}
	a := "/virtual/modules/partner/service/models/a.ts"
	bPath := "/virtual/modules/partner/service/models/b.ts"
	results := []*parser.ParserResult{
		{Path: a, Model: &meta.Model{Name: "FieldDefault", Path: a}},
		{Path: bPath, Model: &meta.Model{Name: "FieldDefault", Path: bPath}},
	}
	builder, _, _ := newInjectTestBuilder(t, mod, results)
	if _, err := builder.Bundle(); err == nil {
		t.Fatal("expected Bundle injectAppModels error")
	}
}

func TestBuildWithoutPersist_UpdateAndValidateRelease(t *testing.T) {
	mod := &meta.Module{
		Name: "partner", Path: "/virtual/modules/partner",
		ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
	}

	// Validate error after inject: circular Partner inheritance.
	p1 := "/virtual/modules/partner/service/models/p1.ts"
	p2 := "/virtual/modules/partner/service/models/p2.ts"
	builder, _, _ := newInjectTestBuilder(t, mod, []*parser.ParserResult{
		{Path: p1, Model: &meta.Model{Name: "Partner", Path: p1, Extends: p2}},
		{Path: p2, Model: &meta.Model{Name: "Partner", Path: p2, Extends: p1}},
	})
	if _, err := builder.BuildWithoutPersist(); err == nil {
		t.Fatal("expected validate error with schedule release")
	}

	// updatePrebuildResult error inside BuildWithoutPersist: getNewExtends returns a tip,
	// then refreshModelExtendsProperty fails via fixedParser.
	oldPath := "/virtual/modules/partner/service/models/partner_old.ts"
	tipPath := "/virtual/modules/partner/service/models/partner_tip.ts"
	curPath := "/virtual/modules/partner/service/models/partner.ts"
	builder2, _, db := newInjectTestBuilder(t, mod, []*parser.ParserResult{{
		Path:    curPath,
		Content: "export default class Partner extends Base {}",
		Model:   &meta.Model{Name: "Partner", Path: curPath, Extends: oldPath, Application: "partner"},
		Imports: map[string]*parser.Import{},
	}})
	for _, seed := range []struct{ id, path string }{
		{"raw-old", oldPath},
		{"raw-tip", tipPath},
	} {
		if err := meta.PersistModelTreeAsRaw(db, &meta.Model{
			BaseModel:   meta.BaseModel{Id: sql.NullString{String: seed.id, Valid: true}},
			Name:        "Partner",
			Path:        seed.path,
			Application: "partner",
		}); err != nil {
			t.Fatal(err)
		}
	}
	// Tip ordering: prefer tipPath by bumping timestamps via PreferDeclarationTip.
	if err := meta.PreferDeclarationTip(db, "partner", "Partner", tipPath); err != nil {
		t.Fatal(err)
	}
	builder2.tsParser = fixedParser{err: sql.ErrConnDone}
	if _, err := builder2.BuildWithoutPersist(); err == nil {
		t.Fatal("expected updatePrebuildResult error with schedule release")
	}
}
