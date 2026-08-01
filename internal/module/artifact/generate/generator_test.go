// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package generator

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/choysum-dev/choysum/internal/esbplugins"
	"github.com/choysum-dev/choysum/internal/esbplugins/backendplugin"
	module "github.com/choysum-dev/choysum/internal/module/artifact/result"
	"github.com/choysum-dev/choysum/internal/parser/backendtsparser"
	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type generatorTestScope struct {
	ctx    context.Context
	db     *gorm.DB
	logger *slog.Logger
	cfg    *config.Config
}

func (e *generatorTestScope) Run(fn func(runtimeScope scope.Scope) error) error { return fn(e) }
func (e *generatorTestScope) Transactor() scope.Transactor {
	return scopetest.NewPassthroughTransactor(e)
}
func (e *generatorTestScope) Session() *scope.Session { return &scope.Session{DB: e.db} }
func (e *generatorTestScope) WithContext(ctx context.Context) scope.Scope {
	clone := *e
	clone.ctx = ctx
	return &clone
}
func (e *generatorTestScope) Context() context.Context { return e.ctx }
func (e *generatorTestScope) Logger() *slog.Logger     { return e.logger }
func (e *generatorTestScope) Config() *config.Config   { return e.cfg }

func (e *generatorTestScope) FactoryInput() scope.FactoryInput {
	return scopetest.FactoryInputFromConfig(e.Config())
}

func newGeneratorScope(t *testing.T) *generatorTestScope {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "generator.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	return &generatorTestScope{
		ctx:    context.Background(),
		db:     db,
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		cfg: &config.Config{
			ModulesPath:        t.TempDir(),
			DistPath:           t.TempDir(),
			DefaultChoysumPath: t.TempDir(),
			Compile:            &config.CompileConfig{BundleMode: string(config.BundleModeApplication)},
		},
	}
}

func seedGeneratorMetaTables(t *testing.T, runtimeScope *generatorTestScope) {
	t.Helper()
	if err := runtimeScope.db.AutoMigrate(
		&meta.Application{},
		&meta.Module{},
		&meta.Model{},
		&meta.Field{},
		&meta.Service{},
		&meta.Decorator{},
		&meta.Parameter{},
		&meta.TypeParameter{},
	); err != nil {
		t.Fatalf("migrate generator meta tables: %v", err)
	}
}

// seedAbstractBaseModel inserts an abstract BaseModel Model at BaseModelModuleSpec path
// with conventional services. Pass nil for the default name set; pass a non-nil empty
// slice to seed BaseModel with no services.
func seedAbstractBaseModel(t *testing.T, runtimeScope *generatorTestScope, names []string) {
	t.Helper()
	seedGeneratorMetaTables(t, runtimeScope)

	path, _ := meta.BaseModelModuleSpec(runtimeScope)
	path = esbplugins.NormalizePath(path)
	if !strings.HasSuffix(path, ".ts") {
		path = path + ".ts"
	}
	if names == nil {
		names = []string{
			"Browse", "BrowseMany", "Search", "NameSearch", "Copy", "Count",
			"Create", "CreateMany", "Update", "UpdateById", "Delete", "DeleteById",
			"DefaultGet", "FieldsGet", "GetFieldTranslations", "UpdateFieldTranslations",
			"Onchange", "ReadGroup", "ReadGroupCount",
		}
	}

	model := &meta.Model{
		BaseModel: meta.BaseModel{Id: sql.NullString{String: "abstract-base-model", Valid: true}},
		Name:      "BaseModel",
		Path:      path,
		Abstract:  true,
	}
	if err := runtimeScope.db.Create(model).Error; err != nil {
		t.Fatalf("create abstract BaseModel: %v", err)
	}
	for i, name := range names {
		svc := &meta.Service{
			BaseModel:             meta.BaseModel{Id: sql.NullString{String: "base-svc-" + strconv.Itoa(i), Valid: true}},
			Name:                  name,
			AccessibilityModifier: "public",
			IsStatic:              true,
			ModelId:               model.Id,
		}
		if err := runtimeScope.db.Create(svc).Error; err != nil {
			t.Fatalf("create BaseModel service %s: %v", name, err)
		}
	}
}

func seedGeneratorAppFixture(t *testing.T, runtimeScope *generatorTestScope) (*meta.Application, *meta.Module) {
	t.Helper()
	seedGeneratorMetaTables(t, runtimeScope)

	app := &meta.Application{BaseModel: meta.BaseModel{Id: sql.NullString{String: "app-fixture", Valid: true}}, Name: "crm"}
	mod := &meta.Module{BaseModel: meta.BaseModel{Id: sql.NullString{String: "module-fixture", Valid: true}}, Name: "crm", ApplicationStr: "crm", ApplicationId: app.Id}
	if err := runtimeScope.db.Create(app).Error; err != nil {
		t.Fatalf("create fixture app: %v", err)
	}
	if err := runtimeScope.db.Create(mod).Error; err != nil {
		t.Fatalf("create fixture module: %v", err)
	}

	model := &meta.Model{
		BaseModel:  meta.BaseModel{Id: sql.NullString{String: "model-fixture", Valid: true}, UpdatedAt: time.Date(2026, 4, 8, 12, 0, 0, 0, time.UTC)},
		Name:       "Partner",
		Path:       "@/crm/models/partner.ts",
		ModelTable: "crm_partner",
		ModuleId:   mod.Id,
	}
	if err := runtimeScope.db.Create(model).Error; err != nil {
		t.Fatalf("create fixture model: %v", err)
	}

	fields := []*meta.Field{
		{BaseModel: meta.BaseModel{Id: sql.NullString{String: "field-id", Valid: true}}, Name: "Id", FieldType: "Char", TsTypeAnnotation: "string", NotNull: true, Indexed: true, ModelId: model.Id},
		{BaseModel: meta.BaseModel{Id: sql.NullString{String: "field-name", Valid: true}}, Name: "Name", FieldType: "Char", TsTypeAnnotation: "string", ModelId: model.Id},
	}
	for _, field := range fields {
		if err := runtimeScope.db.Create(field).Error; err != nil {
			t.Fatalf("create fixture field %s: %v", field.Name, err)
		}
	}

	service := &meta.Service{
		BaseModel:             meta.BaseModel{Id: sql.NullString{String: "service-fixture", Valid: true}},
		Name:                  "CreatePartner",
		AccessibilityModifier: "public",
		IsStatic:              true,
		ProtobufType:          "google.protobuf.Empty",
		ModelId:               model.Id,
	}
	if err := runtimeScope.db.Create(service).Error; err != nil {
		t.Fatalf("create fixture service: %v", err)
	}
	param := &meta.Parameter{BaseModel: meta.BaseModel{Id: sql.NullString{String: "param-fixture", Valid: true}}, Name: "partner_id", ProtobufType: "string", ServiceId: service.Id}
	if err := runtimeScope.db.Create(param).Error; err != nil {
		t.Fatalf("create fixture parameter: %v", err)
	}

	return app, mod
}

func testApp() *meta.Application {
	return &meta.Application{
		Name: "crm",
		Models: []*meta.Model{
			{
				Name:     "BaseModel",
				Path:     "@/base/service/models/base_model.ts",
				Services: []*meta.Service{{Name: "Browse", AccessibilityModifier: "public", IsStatic: true}},
			},
			{
				Name: "Partner",
				Path: "@/crm/service/models/partner.ts",
				Fields: []*meta.Field{
					{Name: "Id", FieldType: "Char", TsTypeAnnotation: "string", NotNull: true, Indexed: true},
					{Name: "CompanyId", FieldType: "ManyToOne", TsTypeAnnotation: "Company", RelationModel: "Company", Relation: "company_id"},
					{Name: "Children", FieldType: "OneToMany", TsTypeAnnotation: "Partner[]", RelationModel: "Partner", Relation: "children_ids"},
				},
				Services: []*meta.Service{{
					Name:                  "CreatePartner",
					AccessibilityModifier: "public",
					IsStatic:              true,
					ProtobufType:          "crm.CreatePartnerReply",
					Parameters:            []*meta.Parameter{{Name: "partner_id", ProtobufType: "string"}},
				}},
			},
		},
	}
}

func TestGeneratorHelpers(t *testing.T) {
	runtimeScope := newGeneratorScope(t)
	generator := &grpcGenerator{runtimeScope: runtimeScope}
	services := []*meta.Service{
		nil,
		{Name: "browse", AccessibilityModifier: "public", IsStatic: true},
		{Name: "PartnerList", AccessibilityModifier: "public", IsStatic: false},
		{Name: "Zeta", AccessibilityModifier: "public", IsStatic: true},
		{Name: "Alpha", AccessibilityModifier: "public", IsStatic: true},
	}
	filtered := generator.filterServices(services)
	if len(filtered) != 2 || filtered[0].Name != "Alpha" || filtered[1].Name != "Zeta" {
		t.Fatalf("unexpected filtered services: %#v", filtered)
	}
	if NewGrpcGenerator(runtimeScope, &meta.Module{Name: "base"}) == nil {
		t.Fatal("expected grpc generator constructor to return non-nil")
	}
}

func TestServiceDiscoveryConventionConsistencyAcrossStages(t *testing.T) {
	runtimeScope := newGeneratorScope(t)
	runtimeOpts := runtimeOptionsFromScope(runtimeScope)
	modulePath := filepath.Join(runtimeOpts.modulesPath, "crm")
	mod := &meta.Module{Path: modulePath, ApplicationStr: "crm"}
	parserImpl := backendtsparser.NewTsParser(runtimeScope, mod)

	path := filepath.Join(modulePath, "service", "models", "partner.ts")
	content := `import { Model } from '../../core/service';

@Model('Partner')
export default class Partner {
	public static async Create(): Promise<void> {
		return
	}

	public static async Fetch(): Promise<void> {
		return
	}

	public static async helper(): Promise<void> {
		return
	}

	public async Remove(): Promise<void> {
		return
	}

	private static async Hidden(): Promise<void> {
		return
	}
}
`

	parsed, err := parserImpl.Parse(map[string]string{}, path, content)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if parsed == nil || parsed.Model == nil {
		t.Fatal("expected parser result with model")
	}

	parserServiceNames := make([]string, 0, len(parsed.Model.Services))
	for _, svc := range parsed.Model.Services {
		if svc != nil {
			parserServiceNames = append(parserServiceNames, svc.Name)
		}
	}
	expectedNames := []string{"Create", "Fetch"}
	if !reflect.DeepEqual(parserServiceNames, expectedNames) {
		t.Fatalf("unexpected parser services: got=%v want=%v", parserServiceNames, expectedNames)
	}

	plugin := backendplugin.NewBackendPlugin(runtimeScope, mod, "")
	bp, ok := plugin.(*backendplugin.BackendPlugin)
	if !ok {
		t.Fatalf("unexpected backend plugin type: %T", plugin)
	}

	bp.Wg.Add(1)
	go func() {
		defer bp.Wg.Done()
		bp.ParserResultChan <- parsed
	}()

	results, err := bp.GetParserResults()
	if err != nil {
		t.Fatalf("backend plugin get parser results failed: %v", err)
	}
	if len(results) != 1 || results[0] == nil || results[0].Model == nil {
		t.Fatalf("unexpected backend plugin results: %#v", results)
	}

	backendServiceNames := make([]string, 0, len(results[0].Model.Services))
	for _, svc := range results[0].Model.Services {
		if svc != nil {
			backendServiceNames = append(backendServiceNames, svc.Name)
		}
	}
	if !reflect.DeepEqual(backendServiceNames, expectedNames) {
		t.Fatalf("unexpected backend plugin services: got=%v want=%v", backendServiceNames, expectedNames)
	}

	g := &grpcGenerator{runtimeScope: runtimeScope}
	generatorServices := g.filterServices(results[0].Model.Services)
	generatorServiceNames := make([]string, 0, len(generatorServices))
	for _, svc := range generatorServices {
		if svc != nil {
			generatorServiceNames = append(generatorServiceNames, svc.Name)
		}
	}
	if !reflect.DeepEqual(generatorServiceNames, expectedNames) {
		t.Fatalf("unexpected generator services: got=%v want=%v", generatorServiceNames, expectedNames)
	}
}

func TestGetApplicationLoadsCanonicalModelsAndFiltersServices(t *testing.T) {
	runtimeScope := newGeneratorScope(t)
	seedGeneratorMetaTables(t, runtimeScope)

	app := &meta.Application{BaseModel: meta.BaseModel{Id: sql.NullString{String: "app-1", Valid: true}}, Name: "crm"}
	mod := &meta.Module{BaseModel: meta.BaseModel{Id: sql.NullString{String: "module-1", Valid: true}}, Name: "crm", ApplicationStr: "crm", ApplicationId: app.Id}
	if err := runtimeScope.db.Create(app).Error; err != nil {
		t.Fatalf("create app: %v", err)
	}
	if err := runtimeScope.db.Create(mod).Error; err != nil {
		t.Fatalf("create module: %v", err)
	}

	base := &meta.Model{
		BaseModel:  meta.BaseModel{Id: sql.NullString{String: "model-base", Valid: true}, UpdatedAt: time.Date(2026, 4, 8, 9, 0, 0, 0, time.UTC)},
		Name:       "Partner",
		Path:       "@/crm/models/partner.ts",
		ModelTable: "crm_partner",
		ModuleId:   mod.Id,
	}
	olderSamePath := &meta.Model{
		BaseModel:  meta.BaseModel{Id: sql.NullString{String: "model-old", Valid: true}, UpdatedAt: time.Date(2026, 4, 8, 8, 0, 0, 0, time.UTC)},
		Name:       "Partner",
		Path:       "@/crm/models/partner.ts",
		ModelTable: "crm_partner_old",
		ModuleId:   mod.Id,
	}
	extension := &meta.Model{
		BaseModel:  meta.BaseModel{Id: sql.NullString{String: "model-ext", Valid: true}, UpdatedAt: time.Date(2026, 4, 8, 10, 0, 0, 0, time.UTC)},
		Name:       "Partner",
		Path:       "@/crm_ext/models/partner.ts",
		Extends:    "@/crm/models/partner.ts",
		ModelTable: "crm_partner",
		ModuleId:   mod.Id,
	}
	other := &meta.Model{
		BaseModel:  meta.BaseModel{Id: sql.NullString{String: "model-other", Valid: true}, UpdatedAt: time.Date(2026, 4, 8, 11, 0, 0, 0, time.UTC)},
		Name:       "Company",
		Path:       "@/crm/models/company.ts",
		ModelTable: "crm_company",
		ModuleId:   mod.Id,
	}
	synthetic := &meta.Model{
		BaseModel: meta.BaseModel{Id: sql.NullString{String: "model-synthetic", Valid: true}, UpdatedAt: time.Date(2026, 4, 8, 12, 0, 0, 0, time.UTC)},
		Name:      "I18n",
		Path:      "go://i18n/crm",
		ModuleId:  mod.Id,
		Abstract:  true,
		Readonly:  true,
	}
	models := []*meta.Model{olderSamePath, base, extension, other, synthetic}
	for _, model := range models {
		if err := runtimeScope.db.Create(model).Error; err != nil {
			t.Fatalf("create model %s: %v", model.Name, err)
		}
	}
	fields := []*meta.Field{
		{BaseModel: meta.BaseModel{Id: sql.NullString{String: "field-1", Valid: true}}, Name: "Name", ModelId: base.Id},
		{BaseModel: meta.BaseModel{Id: sql.NullString{String: "field-2", Valid: true}}, Name: "Code", ModelId: olderSamePath.Id},
		{BaseModel: meta.BaseModel{Id: sql.NullString{String: "field-3", Valid: true}}, Name: "ExtField", ModelId: extension.Id},
		{BaseModel: meta.BaseModel{Id: sql.NullString{String: "field-4", Valid: true}}, Name: "Vat", ModelId: other.Id},
	}
	for _, field := range fields {
		if err := runtimeScope.db.Create(field).Error; err != nil {
			t.Fatalf("create field %s: %v", field.Name, err)
		}
	}
	services := []*meta.Service{
		{BaseModel: meta.BaseModel{Id: sql.NullString{String: "service-a", Valid: true}}, Name: "zeta", AccessibilityModifier: "public", IsStatic: true, ModelId: extension.Id},
		{BaseModel: meta.BaseModel{Id: sql.NullString{String: "service-b", Valid: true}}, Name: "Alpha", AccessibilityModifier: "public", IsStatic: true, ModelId: extension.Id},
		{BaseModel: meta.BaseModel{Id: sql.NullString{String: "service-c", Valid: true}}, Name: "Zeta", AccessibilityModifier: "public", IsStatic: true, ModelId: extension.Id},
		{BaseModel: meta.BaseModel{Id: sql.NullString{String: "service-i18n", Valid: true}}, Name: "GetTranslations", AccessibilityModifier: "public", IsStatic: true, ModelId: synthetic.Id},
	}
	for _, service := range services {
		if err := runtimeScope.db.Create(service).Error; err != nil {
			t.Fatalf("create service %s: %v", service.Name, err)
		}
	}
	param := &meta.Parameter{BaseModel: meta.BaseModel{Id: sql.NullString{String: "param-1", Valid: true}}, Name: "partner_id", ProtobufType: "string", ServiceId: services[2].Id}
	if err := runtimeScope.db.Create(param).Error; err != nil {
		t.Fatalf("create parameter: %v", err)
	}

	g := &grpcGenerator{runtimeScope: runtimeScope, module: mod}
	loaded, err := g.getApplication()
	if err != nil {
		t.Fatalf("getApplication() error = %v", err)
	}
	if loaded == nil || loaded.Name != "crm" {
		t.Fatalf("unexpected application: %#v", loaded)
	}
	if len(loaded.Models) != 2 {
		t.Fatalf("expected 2 canonical models, got %#v", loaded.Models)
	}
	if loaded.Models[0].Name != "Company" || loaded.Models[1].Name != "Partner" {
		t.Fatalf("unexpected model order: %#v", loaded.Models)
	}
	partner := loaded.Models[1]
	fieldNames := map[string]bool{}
	for _, field := range partner.Fields {
		if field != nil && field.Name != "" {
			fieldNames[field.Name] = true
		}
	}
	if !fieldNames["Name"] || !fieldNames["ExtField"] || fieldNames["Code"] {
		t.Fatalf("unexpected merged partner fields: %#v", fieldNames)
	}
	if len(partner.Services) != 2 || partner.Services[0].Name != "Alpha" || partner.Services[1].Name != "Zeta" {
		t.Fatalf("unexpected filtered services: %#v", partner.Services)
	}
	if len(partner.Services[1].Parameters) != 1 || partner.Services[1].Parameters[0].Name != "partner_id" {
		t.Fatalf("unexpected service parameters: %#v", partner.Services[1].Parameters)
	}
}

func TestGetApplication_SelectionAddMergeError(t *testing.T) {
	runtimeScope := newGeneratorScope(t)
	seedGeneratorMetaTables(t, runtimeScope)

	app := &meta.Application{BaseModel: meta.BaseModel{Id: sql.NullString{String: "app-sel", Valid: true}}, Name: "crm"}
	mod := &meta.Module{BaseModel: meta.BaseModel{Id: sql.NullString{String: "module-sel", Valid: true}}, Name: "crm", ApplicationStr: "crm", ApplicationId: app.Id}
	if err := runtimeScope.db.Create(app).Error; err != nil {
		t.Fatalf("create app: %v", err)
	}
	if err := runtimeScope.db.Create(mod).Error; err != nil {
		t.Fatalf("create module: %v", err)
	}

	basePath := "@/crm/models/partner.ts"
	extPath := "@/crm_ext/models/partner.ts"
	base := &meta.Model{
		BaseModel: meta.BaseModel{Id: sql.NullString{String: "model-sel-base", Valid: true}, UpdatedAt: time.Date(2026, 4, 8, 9, 0, 0, 0, time.UTC)},
		Name:      "Partner",
		Path:      basePath,
		ModuleId:  mod.Id,
	}
	ext := &meta.Model{
		BaseModel: meta.BaseModel{Id: sql.NullString{String: "model-sel-ext", Valid: true}, UpdatedAt: time.Date(2026, 4, 8, 10, 0, 0, 0, time.UTC)},
		Name:      "Partner",
		Path:      extPath,
		Extends:   basePath,
		ModuleId:  mod.Id,
	}
	if err := runtimeScope.db.Create(base).Error; err != nil {
		t.Fatalf("create base: %v", err)
	}
	if err := runtimeScope.db.Create(ext).Error; err != nil {
		t.Fatalf("create ext: %v", err)
	}

	baseField := &meta.Field{
		BaseModel:       meta.BaseModel{Id: sql.NullString{String: "field-sel-base", Valid: true}},
		Name:            "Kind",
		FieldType:       "selection",
		SelectionKind:   "dynamic",
		SelectionMethod: "Opts",
		ModelId:         base.Id,
	}
	_ = baseField.SetResolvedSpec(&meta.FieldResolvedSpec{
		FieldName: "Kind",
		Structural: meta.FieldStructuralSpec{
			Name:            "Kind",
			FieldType:       "selection",
			SelectionKind:   "dynamic",
			SelectionMethod: "Opts",
		},
	})
	extField := &meta.Field{
		BaseModel: meta.BaseModel{Id: sql.NullString{String: "field-sel-ext", Valid: true}},
		Name:      "Kind",
		FieldType: "selection",
		ModelId:   ext.Id,
	}
	_ = extField.SetResolvedSpec(&meta.FieldResolvedSpec{
		FieldName: "Kind",
		Structural: meta.FieldStructuralSpec{
			Name:            "Kind",
			FieldType:       "selection",
			HasSelectionAdd: true,
			SelectionAdd:    []meta.FieldSelectionItem{{Value: "vip", Label: "VIP"}},
		},
	})
	for _, field := range []*meta.Field{baseField, extField} {
		if err := runtimeScope.db.Create(field).Error; err != nil {
			t.Fatalf("create field %s: %v", field.Name, err)
		}
	}

	g := &grpcGenerator{runtimeScope: runtimeScope, module: mod}
	if _, err := g.getApplication(); err == nil || !strings.Contains(err.Error(), "inherited static selection") {
		t.Fatalf("expected getApplication selectionAdd merge error, got %v", err)
	}
}

func TestGeneratorEntryPoints(t *testing.T) {
	t.Run("Generate and GenerateCtx short-circuit safely", func(t *testing.T) {
		generator := &grpcGenerator{module: &meta.Module{}}

		results, err := generator.Generate()
		if err != nil {
			t.Fatalf("Generate() error = %v", err)
		}
		if results != nil {
			t.Fatalf("expected nil results when ApplicationId is invalid, got %#v", results)
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		results, err = generator.GenerateCtx(ctx)
		if err == nil || err != context.Canceled {
			t.Fatalf("GenerateCtx(canceled) error = %v", err)
		}
		if results != nil {
			t.Fatalf("expected nil results for canceled context, got %#v", results)
		}
	})

	t.Run("GenerateToTargetsCtx injects staging directories before GenerateCtx", func(t *testing.T) {
		ctx := context.Background()
		protoDir := t.TempDir()
		webDir := t.TempDir()
		serviceDir := t.TempDir()
		generator := &grpcGenerator{
			module:                 &meta.Module{},
			protobufGenerator:      &protobufGenerator{},
			webGrpcGenerator:       &webGrpcGenerator{},
			webServiceGenerator:    &webServiceGenerator{},
			webApiStoreGenerator:   &webApiStoreGenerator{},
			serviceClientGenerator: &serviceClientGenerator{},
		}

		results, err := generator.GenerateToTargetsCtx(ctx, protoDir, webDir, serviceDir, filepath.Join(t.TempDir(), "dist-app"))
		if err != nil {
			t.Fatalf("GenerateToTargetsCtx() error = %v", err)
		}
		if results != nil {
			t.Fatalf("expected nil results when ApplicationId is invalid, got %#v", results)
		}
		if generator.protobufGenerator.modulesProtoDir != protoDir {
			t.Fatalf("unexpected protobuf proto dir: %q", generator.protobufGenerator.modulesProtoDir)
		}
		if generator.protobufGenerator.distAppDir == "" {
			t.Fatal("expected protobuf dist app dir to be set")
		}
		if generator.webGrpcGenerator.modulesProtoDir != protoDir || generator.webGrpcGenerator.modulesWebDir != webDir {
			t.Fatalf("unexpected web grpc dirs: %#v", generator.webGrpcGenerator)
		}
		if generator.webServiceGenerator.modulesWebDir != webDir {
			t.Fatalf("unexpected web service dir: %q", generator.webServiceGenerator.modulesWebDir)
		}
		if generator.webApiStoreGenerator.modulesWebDir != webDir {
			t.Fatalf("unexpected web api store dir: %q", generator.webApiStoreGenerator.modulesWebDir)
		}
		if generator.serviceClientGenerator.modulesProtoDir != protoDir || generator.serviceClientGenerator.modulesServiceDir != serviceDir {
			t.Fatalf("unexpected service client dirs: %#v", generator.serviceClientGenerator)
		}
		if _, ok := any(NewGrpcGenerator(newGeneratorScope(t), &meta.Module{Name: "base"})).(module.Generator); !ok {
			t.Fatal("expected NewGrpcGenerator to satisfy module.Generator")
		}
	})

	t.Run("GenerateCtx runs full success pipeline", func(t *testing.T) {
		runtimeScope := newGeneratorScope(t)
		_, mod := seedGeneratorAppFixture(t, runtimeScope)
		seedAbstractBaseModel(t, runtimeScope, nil)
		protoDir := t.TempDir()
		webDir := t.TempDir()
		serviceDir := t.TempDir()
		distAppDir := filepath.Join(t.TempDir(), "apps", "crm")
		generator := &grpcGenerator{
			runtimeScope: runtimeScope,
			module:       mod,
			protobufGenerator: &protobufGenerator{
				runtimeScope:    runtimeScope,
				module:          mod,
				modulesProtoDir: protoDir,
				distAppDir:      distAppDir,
			},
			webGrpcGenerator: &webGrpcGenerator{
				runtimeScope:    runtimeScope,
				module:          mod,
				plugins:         []GrpcPlugin{fakeGrpcPlugin{name: "fake-grpc"}},
				modulesProtoDir: protoDir,
				modulesWebDir:   webDir,
			},
			webServiceGenerator:    &webServiceGenerator{runtimeScope: runtimeScope, module: mod, modulesWebDir: webDir},
			webApiStoreGenerator:   &webApiStoreGenerator{runtimeScope: runtimeScope, module: mod, modulesWebDir: webDir},
			serviceClientGenerator: &serviceClientGenerator{runtimeScope: runtimeScope, module: mod, modulesProtoDir: protoDir, modulesServiceDir: serviceDir},
		}

		results, err := generator.GenerateCtx(context.Background())
		if err != nil {
			t.Fatalf("GenerateCtx(success) error = %v", err)
		}
		if len(results) != 7 {
			t.Fatalf("expected 7 generator results, got %#v", results)
		}
		resultNames := map[string]bool{}
		for _, result := range results {
			if result != nil {
				resultNames[result.Name] = true
			}
		}
		for _, name := range []string{"protobuf", "fake-grpc", "webservice", "serviceclient", "webapistore", "generated-tsconfig"} {
			if !resultNames[name] {
				t.Fatalf("expected result %q, got %#v", name, resultNames)
			}
		}
		runtimeOpts := runtimeOptionsFromScope(runtimeScope)
		generatedRoot, err := WorkspaceGeneratedAPIRoot(runtimeOpts.modulesPath, runtimeOpts.defaultChoysumPath)
		if err != nil {
			t.Fatalf("WorkspaceGeneratedAPIRoot() error = %v", err)
		}
		for _, path := range []string{
			filepath.Join(protoDir, "crm.proto"),
			filepath.Join(webDir, "pb", "crm_pb.ts"),
			filepath.Join(webDir, "service.ts"),
			filepath.Join(webDir, "stores", "partner.ts"),
			filepath.Join(serviceDir, "service.ts"),
			filepath.Join(generatedRoot, "tsconfig.json"),
			filepath.Join(distAppDir, "assets", "crm.proto"),
		} {
			if _, err := os.Stat(path); err != nil {
				t.Fatalf("expected generated file %s: %v", path, err)
			}
		}
	})
}

func TestMergeSameNameModelsByExtensionChain_PreservesBranchedExtensionFields(t *testing.T) {
	basePath := "@/partner/service/models/partner.ts"
	bankPath := "@/partner_bank/service/models/partner.ts"
	commercialPath := "@/partner_commercial/service/models/partner.ts"

	base := &meta.Model{
		BaseModel: meta.BaseModel{UpdatedAt: time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC), Id: sql.NullString{String: "base", Valid: true}},
		Name:      "Partner",
		Path:      basePath,
		Fields: []*meta.Field{
			{Name: "Name"},
			{Name: "Contacts"},
		},
	}

	bank := &meta.Model{
		BaseModel: meta.BaseModel{UpdatedAt: time.Date(2026, 3, 15, 11, 0, 0, 0, time.UTC), Id: sql.NullString{String: "bank", Valid: true}},
		Name:      "Partner",
		Path:      bankPath,
		Extends:   basePath,
		Fields: []*meta.Field{
			{Name: "BankAccounts"},
		},
	}

	commercial := &meta.Model{
		BaseModel: meta.BaseModel{UpdatedAt: time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC), Id: sql.NullString{String: "commercial", Valid: true}},
		Name:      "Partner",
		Path:      commercialPath,
		Extends:   basePath,
		Fields: []*meta.Field{
			{Name: "PartnerIdentifiers"},
		},
	}

	merged, err := mergeSameNameModelsByExtensionChain([]*meta.Model{base, bank, commercial})

	if err != nil {

		t.Fatalf("mergeSameNameModelsByExtensionChain: %v", err)

	}
	if merged == nil {
		t.Fatalf("expected merged model, got nil")
	}

	fieldNames := map[string]bool{}
	for _, f := range merged.Fields {
		if f == nil || f.Name == "" {
			continue
		}
		fieldNames[f.Name] = true
	}

	for _, expected := range []string{"Name", "Contacts", "BankAccounts", "PartnerIdentifiers"} {
		if !fieldNames[expected] {
			t.Fatalf("expected merged fields to contain %q, got: %#v", expected, fieldNames)
		}
	}
}

func TestMergeSameNameModelsByExtensionChain_FieldConflictUsesExtensionPriority(t *testing.T) {
	basePath := "@/partner/service/models/partner.ts"
	bankPath := "@/partner_bank/service/models/partner.ts"
	commercialPath := "@/partner_commercial/service/models/partner.ts"

	base := &meta.Model{
		BaseModel: meta.BaseModel{UpdatedAt: time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC), Id: sql.NullString{String: "base", Valid: true}},
		Name:      "Partner",
		Path:      basePath,
		Fields:    []*meta.Field{{Name: "Name", TsTypeAnnotation: "base-name", Size: 100}, {Name: "Code", TsTypeAnnotation: "base-code", Size: 40}},
	}
	bank := &meta.Model{
		BaseModel: meta.BaseModel{UpdatedAt: time.Date(2026, 3, 15, 11, 0, 0, 0, time.UTC), Id: sql.NullString{String: "bank", Valid: true}},
		Name:      "Partner",
		Path:      bankPath,
		Extends:   basePath,
		Fields:    []*meta.Field{{Name: "Name", TsTypeAnnotation: "bank-name", Size: 120}},
	}
	commercial := &meta.Model{
		BaseModel: meta.BaseModel{UpdatedAt: time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC), Id: sql.NullString{String: "commercial", Valid: true}},
		Name:      "Partner",
		Path:      commercialPath,
		Extends:   bankPath,
		Fields:    []*meta.Field{{Name: "Name", TsTypeAnnotation: "commercial-name", Size: 140}},
	}

	merged, err := mergeSameNameModelsByExtensionChain([]*meta.Model{commercial, base, bank})

	if err != nil {

		t.Fatalf("mergeSameNameModelsByExtensionChain: %v", err)

	}
	if merged == nil {
		t.Fatalf("expected merged model, got nil")
	}

	var mergedName *meta.Field
	var mergedCode *meta.Field
	for _, f := range merged.Fields {
		if f == nil {
			continue
		}
		if f.Name == "Name" {
			mergedName = f
		}
		if f.Name == "Code" {
			mergedCode = f
		}
	}

	if mergedName == nil || mergedName.TsTypeAnnotation != "commercial-name" || mergedName.Size != 140 {
		t.Fatalf("unexpected merged Name field: %#v", mergedName)
	}
	if mergedCode == nil || mergedCode.TsTypeAnnotation != "base-code" || mergedCode.Size != 40 {
		t.Fatalf("unexpected merged Code field: %#v", mergedCode)
	}
}

func TestMergeSameNameModelsByExtensionChain_SameDepthBranchConflictUsesStableTieBreak(t *testing.T) {
	basePath := "@/partner/service/models/partner.ts"
	branchAPath := "@/partner_bank/service/models/partner.ts"
	branchBPath := "@/partner_commercial/service/models/partner.ts"

	base := &meta.Model{BaseModel: meta.BaseModel{UpdatedAt: time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC), Id: sql.NullString{String: "base", Valid: true}}, Name: "Partner", Path: basePath, Fields: []*meta.Field{{Name: "Name", TsTypeAnnotation: "base-name", Size: 100}}}
	branchA := &meta.Model{BaseModel: meta.BaseModel{UpdatedAt: time.Date(2026, 3, 15, 11, 0, 0, 0, time.UTC), Id: sql.NullString{String: "a-older", Valid: true}}, Name: "Partner", Path: branchAPath, Extends: basePath, Fields: []*meta.Field{{Name: "Name", TsTypeAnnotation: "branch-a", Size: 120}}}
	branchBNewer := &meta.Model{BaseModel: meta.BaseModel{UpdatedAt: time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC), Id: sql.NullString{String: "b-newer", Valid: true}}, Name: "Partner", Path: branchBPath, Extends: basePath, Fields: []*meta.Field{{Name: "Name", TsTypeAnnotation: "branch-b-newer", Size: 140}}}

	mergedByUpdatedAt, err := mergeSameNameModelsByExtensionChain([]*meta.Model{base, branchA, branchBNewer})

	if err != nil {

		t.Fatalf("mergeSameNameModelsByExtensionChain: %v", err)

	}
	if mergedByUpdatedAt == nil {
		t.Fatalf("expected merged model for UpdatedAt tie-break case, got nil")
	}

	var nameByUpdatedAt *meta.Field
	for _, f := range mergedByUpdatedAt.Fields {
		if f != nil && f.Name == "Name" {
			nameByUpdatedAt = f
			break
		}
	}
	if nameByUpdatedAt == nil || nameByUpdatedAt.TsTypeAnnotation != "branch-b-newer" || nameByUpdatedAt.Size != 140 {
		t.Fatalf("unexpected UpdatedAt tie-break result: %#v", nameByUpdatedAt)
	}

	branchASameTimeLowID := &meta.Model{BaseModel: meta.BaseModel{UpdatedAt: time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC), Id: sql.NullString{String: "aaa", Valid: true}}, Name: "Partner", Path: branchAPath, Extends: basePath, Fields: []*meta.Field{{Name: "Name", TsTypeAnnotation: "branch-a-aaa", Size: 150}}}
	branchBSameTimeHighID := &meta.Model{BaseModel: meta.BaseModel{UpdatedAt: time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC), Id: sql.NullString{String: "zzz", Valid: true}}, Name: "Partner", Path: branchBPath, Extends: basePath, Fields: []*meta.Field{{Name: "Name", TsTypeAnnotation: "branch-b-zzz", Size: 160}}}

	mergedByID, err := mergeSameNameModelsByExtensionChain([]*meta.Model{branchBSameTimeHighID, base, branchASameTimeLowID})

	if err != nil {

		t.Fatalf("mergeSameNameModelsByExtensionChain: %v", err)

	}
	if mergedByID == nil {
		t.Fatalf("expected merged model for Id tie-break case, got nil")
	}

	var nameByID *meta.Field
	for _, f := range mergedByID.Fields {
		if f != nil && f.Name == "Name" {
			nameByID = f
			break
		}
	}
	if nameByID == nil || nameByID.TsTypeAnnotation != "branch-b-zzz" || nameByID.Size != 160 {
		t.Fatalf("unexpected Id tie-break result: %#v", nameByID)
	}
}

func TestSelectSameNameModelsInPrimaryExtensionChain_ExcludesDisconnectedChains(t *testing.T) {
	baseAPath := "@/chain_a/service/models/partner.ts"
	childAPath := "@/chain_a_ext/service/models/partner.ts"
	baseBPath := "@/chain_b/service/models/partner.ts"
	childBPath := "@/chain_b_ext/service/models/partner.ts"

	baseA := &meta.Model{BaseModel: meta.BaseModel{UpdatedAt: time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC), Id: sql.NullString{String: "a-base", Valid: true}}, Name: "Partner", Path: baseAPath, Fields: []*meta.Field{{Name: "Name"}}}
	childA := &meta.Model{BaseModel: meta.BaseModel{UpdatedAt: time.Date(2026, 3, 15, 11, 0, 0, 0, time.UTC), Id: sql.NullString{String: "a-child", Valid: true}}, Name: "Partner", Path: childAPath, Extends: baseAPath, Fields: []*meta.Field{{Name: "AField"}}}
	baseB := &meta.Model{BaseModel: meta.BaseModel{UpdatedAt: time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC), Id: sql.NullString{String: "b-base", Valid: true}}, Name: "Partner", Path: baseBPath, Fields: []*meta.Field{{Name: "Name"}}}
	childB := &meta.Model{BaseModel: meta.BaseModel{UpdatedAt: time.Date(2026, 3, 15, 13, 0, 0, 0, time.UTC), Id: sql.NullString{String: "b-child", Valid: true}}, Name: "Partner", Path: childBPath, Extends: baseBPath, Fields: []*meta.Field{{Name: "BField"}}}

	selected := selectSameNameModelsInPrimaryExtensionChain([]*meta.Model{baseA, childA, baseB, childB})
	merged, err := mergeSameNameModelsByExtensionChain(selected)
	if err != nil {
		t.Fatalf("mergeSameNameModelsByExtensionChain: %v", err)
	}
	if merged == nil {
		t.Fatalf("expected merged model, got nil")
	}

	fieldNames := map[string]bool{}
	for _, f := range merged.Fields {
		if f == nil || f.Name == "" {
			continue
		}
		fieldNames[f.Name] = true
	}
	if !fieldNames["BField"] || fieldNames["AField"] {
		t.Fatalf("unexpected primary-chain field selection: %#v", fieldNames)
	}
}

func TestMergeSameNameModelsByExtensionChain_EmptyAndNilGuards(t *testing.T) {
	merged, err := mergeSameNameModelsByExtensionChain(nil)
	if err != nil || merged != nil {
		t.Fatalf("expected nil,nil for empty input, got %#v err=%v", merged, err)
	}
	merged, err = mergeSameNameModelsByExtensionChain([]*meta.Model{nil, nil})
	if err != nil || merged != nil {
		t.Fatalf("expected nil,nil for all-nil models, got %#v err=%v", merged, err)
	}
	solo := &meta.Model{Name: "Partner", Path: "/solo"}
	merged, err = mergeSameNameModelsByExtensionChain([]*meta.Model{solo})
	if err != nil || merged != solo {
		t.Fatalf("expected solo model passthrough, got %#v err=%v", merged, err)
	}
}

func TestMergeSameNameModelsByExtensionChain_SelectionAddWithoutBaseRejected(t *testing.T) {
	extField := &meta.Field{Name: "Kind", FieldType: "selection"}
	if err := extField.SetResolvedSpec(&meta.FieldResolvedSpec{
		FieldName: "Kind",
		Structural: meta.FieldStructuralSpec{
			Name:            "Kind",
			FieldType:       "selection",
			HasSelectionAdd: true,
			SelectionAdd:    []meta.FieldSelectionItem{{Value: "vip", Label: "VIP"}},
		},
	}); err != nil {
		t.Fatalf("SetResolvedSpec: %v", err)
	}
	ext := &meta.Model{
		BaseModel: meta.BaseModel{UpdatedAt: time.Date(2026, 3, 15, 11, 0, 0, 0, time.UTC), Id: sql.NullString{String: "ext", Valid: true}},
		Name:      "Partner",
		Path:      "@/partner_vip/service/models/partner.ts",
		Fields:    []*meta.Field{extField},
	}
	// First model has no Kind; extension introduces Kind via selectionAdd only.
	base := &meta.Model{
		BaseModel: meta.BaseModel{UpdatedAt: time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC), Id: sql.NullString{String: "base", Valid: true}},
		Name:      "Partner",
		Path:      "@/partner/service/models/partner.ts",
		Fields:    []*meta.Field{{Name: "Name"}},
	}
	ext.Extends = base.Path
	_, err := mergeSameNameModelsByExtensionChain([]*meta.Model{base, ext})
	if err == nil || !strings.Contains(err.Error(), "selectionAdd requires an inherited static selection") {
		t.Fatalf("expected selectionAdd-without-base rejection, got %v", err)
	}
}

func TestMergeSameNameModelsByExtensionChain_SelectionAddConflictError(t *testing.T) {
	basePath := "@/partner/service/models/partner.ts"
	extPath := "@/partner_vip/service/models/partner.ts"
	baseField := &meta.Field{Name: "Kind", FieldType: "selection", SelectionKind: "dynamic", SelectionMethod: "Opts"}
	_ = baseField.SetResolvedSpec(&meta.FieldResolvedSpec{
		FieldName: "Kind",
		Structural: meta.FieldStructuralSpec{
			Name:            "Kind",
			FieldType:       "selection",
			SelectionKind:   "dynamic",
			SelectionMethod: "Opts",
		},
	})
	extField := &meta.Field{Name: "Kind", FieldType: "selection"}
	_ = extField.SetResolvedSpec(&meta.FieldResolvedSpec{
		FieldName: "Kind",
		Structural: meta.FieldStructuralSpec{
			Name:            "Kind",
			FieldType:       "selection",
			HasSelectionAdd: true,
			SelectionAdd:    []meta.FieldSelectionItem{{Value: "vip", Label: "VIP"}},
		},
	})
	base := &meta.Model{
		BaseModel: meta.BaseModel{UpdatedAt: time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC), Id: sql.NullString{String: "base", Valid: true}},
		Name:      "Partner",
		Path:      basePath,
		Fields:    []*meta.Field{baseField},
	}
	ext := &meta.Model{
		BaseModel: meta.BaseModel{UpdatedAt: time.Date(2026, 3, 15, 11, 0, 0, 0, time.UTC), Id: sql.NullString{String: "ext", Valid: true}},
		Name:      "Partner",
		Path:      extPath,
		Extends:   basePath,
		Fields:    []*meta.Field{extField},
	}
	_, err := mergeSameNameModelsByExtensionChain([]*meta.Model{base, ext})
	if err == nil || !strings.Contains(err.Error(), "inherited static selection") {
		t.Fatalf("expected dynamic-base merge error, got %v", err)
	}
}

func TestMergeSameNameModelsByExtensionChain_SelectionAddMerges(t *testing.T) {
	basePath := "@/partner/service/models/partner.ts"
	extPath := "@/partner_vip/service/models/partner.ts"

	baseField := &meta.Field{Name: "Kind", FieldType: "selection", FieldString: "Kind"}
	if err := baseField.SetResolvedSpec(&meta.FieldResolvedSpec{
		FieldName: "Kind",
		Structural: meta.FieldStructuralSpec{
			Name:          "Kind",
			FieldType:     "selection",
			String:        "Kind",
			SelectionKind: "static",
			Selection: []meta.FieldSelectionItem{
				{Value: "company", Label: "Company"},
				{Value: "person", Label: "Person"},
			},
		},
	}); err != nil {
		t.Fatalf("base SetResolvedSpec: %v", err)
	}
	baseField.SelectionKind = "static"
	raw, _ := json.Marshal([]meta.FieldSelectionItem{
		{Value: "company", Label: "Company"},
		{Value: "person", Label: "Person"},
	})
	baseField.Selection = string(raw)

	extField := &meta.Field{Name: "Kind", FieldType: "selection"}
	if err := extField.SetResolvedSpec(&meta.FieldResolvedSpec{
		FieldName: "Kind",
		Structural: meta.FieldStructuralSpec{
			Name:            "Kind",
			FieldType:       "selection",
			HasSelectionAdd: true,
			SelectionAdd: []meta.FieldSelectionItem{
				{Value: "vip", Label: "VIP"},
			},
		},
	}); err != nil {
		t.Fatalf("ext SetResolvedSpec: %v", err)
	}

	base := &meta.Model{
		BaseModel: meta.BaseModel{UpdatedAt: time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC), Id: sql.NullString{String: "base", Valid: true}},
		Name:      "Partner",
		Path:      basePath,
		Fields:    []*meta.Field{baseField},
	}
	ext := &meta.Model{
		BaseModel: meta.BaseModel{UpdatedAt: time.Date(2026, 3, 15, 11, 0, 0, 0, time.UTC), Id: sql.NullString{String: "ext", Valid: true}},
		Name:      "Partner",
		Path:      extPath,
		Extends:   basePath,
		Fields:    []*meta.Field{extField},
	}

	merged, err := mergeSameNameModelsByExtensionChain([]*meta.Model{base, ext})
	if err != nil {
		t.Fatalf("mergeSameNameModelsByExtensionChain: %v", err)
	}
	if merged == nil || len(merged.Fields) != 1 {
		t.Fatalf("unexpected merged model: %#v", merged)
	}
	spec, err := merged.Fields[0].GetResolvedSpec()
	if err != nil || spec == nil {
		t.Fatalf("get resolved spec: %v", err)
	}
	if len(spec.Structural.Selection) != 3 || spec.Structural.Selection[2].Value != "vip" {
		t.Fatalf("expected selectionAdd merge, got %#v", spec.Structural.Selection)
	}
	if spec.Structural.HasSelectionAdd {
		t.Fatal("expected HasSelectionAdd cleared after merge")
	}
}
