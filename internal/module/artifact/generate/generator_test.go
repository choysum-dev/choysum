// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package generator

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

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
		&meta.IrApplication{},
		&meta.IrModule{},
		&meta.IrModel{},
		&meta.IrField{},
		&meta.IrService{},
		&meta.IrDecorator{},
		&meta.IrParameter{},
		&meta.IrTypeParameter{},
	); err != nil {
		t.Fatalf("migrate generator meta tables: %v", err)
	}
}

func seedGeneratorAppFixture(t *testing.T, runtimeScope *generatorTestScope) (*meta.IrApplication, *meta.IrModule) {
	t.Helper()
	seedGeneratorMetaTables(t, runtimeScope)

	app := &meta.IrApplication{BaseModel: meta.BaseModel{Id: sql.NullString{String: "app-fixture", Valid: true}}, Name: "crm"}
	mod := &meta.IrModule{BaseModel: meta.BaseModel{Id: sql.NullString{String: "module-fixture", Valid: true}}, Name: "crm", ApplicationStr: "crm", ApplicationId: app.Id}
	if err := runtimeScope.db.Create(app).Error; err != nil {
		t.Fatalf("create fixture app: %v", err)
	}
	if err := runtimeScope.db.Create(mod).Error; err != nil {
		t.Fatalf("create fixture module: %v", err)
	}

	model := &meta.IrModel{
		BaseModel:  meta.BaseModel{Id: sql.NullString{String: "model-fixture", Valid: true}, UpdatedAt: time.Date(2026, 4, 8, 12, 0, 0, 0, time.UTC)},
		Name:       "Partner",
		Path:       "@/crm/models/partner.ts",
		ModelTable: "crm_partner",
		ModuleId:   mod.Id,
	}
	if err := runtimeScope.db.Create(model).Error; err != nil {
		t.Fatalf("create fixture model: %v", err)
	}

	fields := []*meta.IrField{
		{BaseModel: meta.BaseModel{Id: sql.NullString{String: "field-id", Valid: true}}, Name: "Id", FieldType: "Char", TsTypeAnnotation: "string", NotNull: true, Indexed: true, ModelId: model.Id},
		{BaseModel: meta.BaseModel{Id: sql.NullString{String: "field-name", Valid: true}}, Name: "Name", FieldType: "Char", TsTypeAnnotation: "string", ModelId: model.Id},
	}
	for _, field := range fields {
		if err := runtimeScope.db.Create(field).Error; err != nil {
			t.Fatalf("create fixture field %s: %v", field.Name, err)
		}
	}

	service := &meta.IrService{
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
	param := &meta.IrParameter{BaseModel: meta.BaseModel{Id: sql.NullString{String: "param-fixture", Valid: true}}, Name: "partner_id", ProtobufType: "string", ServiceId: service.Id}
	if err := runtimeScope.db.Create(param).Error; err != nil {
		t.Fatalf("create fixture parameter: %v", err)
	}

	return app, mod
}

func testApp() *meta.IrApplication {
	return &meta.IrApplication{
		Name: "crm",
		Models: []*meta.IrModel{
			{
				Name:     "BaseModel",
				Path:     "@/base/service/models/base_model.ts",
				Services: []*meta.IrService{{Name: "Browse", AccessibilityModifier: "public", IsStatic: true}},
			},
			{
				Name: "Partner",
				Path: "@/crm/service/models/partner.ts",
				Fields: []*meta.IrField{
					{Name: "Id", FieldType: "Char", TsTypeAnnotation: "string", NotNull: true, Indexed: true},
					{Name: "CompanyId", FieldType: "ManyToOne", TsTypeAnnotation: "Company", RelationModel: "Company", Relation: "company_id"},
					{Name: "Children", FieldType: "OneToMany", TsTypeAnnotation: "Partner[]", RelationModel: "Partner", Relation: "children_ids"},
				},
				Services: []*meta.IrService{{
					Name:                  "CreatePartner",
					AccessibilityModifier: "public",
					IsStatic:              true,
					ProtobufType:          "crm.CreatePartnerReply",
					Parameters:            []*meta.IrParameter{{Name: "partner_id", ProtobufType: "string"}},
				}},
			},
		},
	}
}

func TestGeneratorHelpers(t *testing.T) {
	runtimeScope := newGeneratorScope(t)
	generator := &grpcGenerator{runtimeScope: runtimeScope}
	services := []*meta.IrService{
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
	if NewGrpcGenerator(runtimeScope, &meta.IrModule{Name: "base"}) == nil {
		t.Fatal("expected grpc generator constructor to return non-nil")
	}
}

func TestServiceDiscoveryConventionConsistencyAcrossStages(t *testing.T) {
	runtimeScope := newGeneratorScope(t)
	runtimeOpts := runtimeOptionsFromScope(runtimeScope)
	modulePath := filepath.Join(runtimeOpts.modulesPath, "crm")
	mod := &meta.IrModule{Path: modulePath, ApplicationStr: "crm"}
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

	app := &meta.IrApplication{BaseModel: meta.BaseModel{Id: sql.NullString{String: "app-1", Valid: true}}, Name: "crm"}
	mod := &meta.IrModule{BaseModel: meta.BaseModel{Id: sql.NullString{String: "module-1", Valid: true}}, Name: "crm", ApplicationStr: "crm", ApplicationId: app.Id}
	if err := runtimeScope.db.Create(app).Error; err != nil {
		t.Fatalf("create app: %v", err)
	}
	if err := runtimeScope.db.Create(mod).Error; err != nil {
		t.Fatalf("create module: %v", err)
	}

	base := &meta.IrModel{
		BaseModel:  meta.BaseModel{Id: sql.NullString{String: "model-base", Valid: true}, UpdatedAt: time.Date(2026, 4, 8, 9, 0, 0, 0, time.UTC)},
		Name:       "Partner",
		Path:       "@/crm/models/partner.ts",
		ModelTable: "crm_partner",
		ModuleId:   mod.Id,
	}
	olderSamePath := &meta.IrModel{
		BaseModel:  meta.BaseModel{Id: sql.NullString{String: "model-old", Valid: true}, UpdatedAt: time.Date(2026, 4, 8, 8, 0, 0, 0, time.UTC)},
		Name:       "Partner",
		Path:       "@/crm/models/partner.ts",
		ModelTable: "crm_partner_old",
		ModuleId:   mod.Id,
	}
	extension := &meta.IrModel{
		BaseModel:  meta.BaseModel{Id: sql.NullString{String: "model-ext", Valid: true}, UpdatedAt: time.Date(2026, 4, 8, 10, 0, 0, 0, time.UTC)},
		Name:       "Partner",
		Path:       "@/crm_ext/models/partner.ts",
		Extends:    "@/crm/models/partner.ts",
		ModelTable: "crm_partner",
		ModuleId:   mod.Id,
	}
	other := &meta.IrModel{
		BaseModel:  meta.BaseModel{Id: sql.NullString{String: "model-other", Valid: true}, UpdatedAt: time.Date(2026, 4, 8, 11, 0, 0, 0, time.UTC)},
		Name:       "Company",
		Path:       "@/crm/models/company.ts",
		ModelTable: "crm_company",
		ModuleId:   mod.Id,
	}
	synthetic := &meta.IrModel{
		BaseModel: meta.BaseModel{Id: sql.NullString{String: "model-synthetic", Valid: true}, UpdatedAt: time.Date(2026, 4, 8, 12, 0, 0, 0, time.UTC)},
		Name:      "I18n",
		Path:      "go://i18n/crm",
		ModuleId:  mod.Id,
		Abstract:  true,
		Readonly:  true,
	}
	models := []*meta.IrModel{olderSamePath, base, extension, other, synthetic}
	for _, model := range models {
		if err := runtimeScope.db.Create(model).Error; err != nil {
			t.Fatalf("create model %s: %v", model.Name, err)
		}
	}
	fields := []*meta.IrField{
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
	services := []*meta.IrService{
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
	param := &meta.IrParameter{BaseModel: meta.BaseModel{Id: sql.NullString{String: "param-1", Valid: true}}, Name: "partner_id", ProtobufType: "string", ServiceId: services[2].Id}
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

func TestGeneratorEntryPoints(t *testing.T) {
	t.Run("Generate and GenerateCtx short-circuit safely", func(t *testing.T) {
		generator := &grpcGenerator{module: &meta.IrModule{}}

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
			module:                 &meta.IrModule{},
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
		if _, ok := any(NewGrpcGenerator(newGeneratorScope(t), &meta.IrModule{Name: "base"})).(module.Generator); !ok {
			t.Fatal("expected NewGrpcGenerator to satisfy module.Generator")
		}
	})

	t.Run("GenerateCtx runs full success pipeline", func(t *testing.T) {
		runtimeScope := newGeneratorScope(t)
		_, mod := seedGeneratorAppFixture(t, runtimeScope)
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

	base := &meta.IrModel{
		BaseModel: meta.BaseModel{UpdatedAt: time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC), Id: sql.NullString{String: "base", Valid: true}},
		Name:      "Partner",
		Path:      basePath,
		Fields: []*meta.IrField{
			{Name: "Name"},
			{Name: "Contacts"},
		},
	}

	bank := &meta.IrModel{
		BaseModel: meta.BaseModel{UpdatedAt: time.Date(2026, 3, 15, 11, 0, 0, 0, time.UTC), Id: sql.NullString{String: "bank", Valid: true}},
		Name:      "Partner",
		Path:      bankPath,
		Extends:   basePath,
		Fields: []*meta.IrField{
			{Name: "BankAccounts"},
		},
	}

	commercial := &meta.IrModel{
		BaseModel: meta.BaseModel{UpdatedAt: time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC), Id: sql.NullString{String: "commercial", Valid: true}},
		Name:      "Partner",
		Path:      commercialPath,
		Extends:   basePath,
		Fields: []*meta.IrField{
			{Name: "PartnerIdentifiers"},
		},
	}

	merged := mergeSameNameModelsByExtensionChain([]*meta.IrModel{base, bank, commercial})
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

	base := &meta.IrModel{
		BaseModel: meta.BaseModel{UpdatedAt: time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC), Id: sql.NullString{String: "base", Valid: true}},
		Name:      "Partner",
		Path:      basePath,
		Fields:    []*meta.IrField{{Name: "Name", TsTypeAnnotation: "base-name", Size: 100}, {Name: "Code", TsTypeAnnotation: "base-code", Size: 40}},
	}
	bank := &meta.IrModel{
		BaseModel: meta.BaseModel{UpdatedAt: time.Date(2026, 3, 15, 11, 0, 0, 0, time.UTC), Id: sql.NullString{String: "bank", Valid: true}},
		Name:      "Partner",
		Path:      bankPath,
		Extends:   basePath,
		Fields:    []*meta.IrField{{Name: "Name", TsTypeAnnotation: "bank-name", Size: 120}},
	}
	commercial := &meta.IrModel{
		BaseModel: meta.BaseModel{UpdatedAt: time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC), Id: sql.NullString{String: "commercial", Valid: true}},
		Name:      "Partner",
		Path:      commercialPath,
		Extends:   bankPath,
		Fields:    []*meta.IrField{{Name: "Name", TsTypeAnnotation: "commercial-name", Size: 140}},
	}

	merged := mergeSameNameModelsByExtensionChain([]*meta.IrModel{commercial, base, bank})
	if merged == nil {
		t.Fatalf("expected merged model, got nil")
	}

	var mergedName *meta.IrField
	var mergedCode *meta.IrField
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

	base := &meta.IrModel{BaseModel: meta.BaseModel{UpdatedAt: time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC), Id: sql.NullString{String: "base", Valid: true}}, Name: "Partner", Path: basePath, Fields: []*meta.IrField{{Name: "Name", TsTypeAnnotation: "base-name", Size: 100}}}
	branchA := &meta.IrModel{BaseModel: meta.BaseModel{UpdatedAt: time.Date(2026, 3, 15, 11, 0, 0, 0, time.UTC), Id: sql.NullString{String: "a-older", Valid: true}}, Name: "Partner", Path: branchAPath, Extends: basePath, Fields: []*meta.IrField{{Name: "Name", TsTypeAnnotation: "branch-a", Size: 120}}}
	branchBNewer := &meta.IrModel{BaseModel: meta.BaseModel{UpdatedAt: time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC), Id: sql.NullString{String: "b-newer", Valid: true}}, Name: "Partner", Path: branchBPath, Extends: basePath, Fields: []*meta.IrField{{Name: "Name", TsTypeAnnotation: "branch-b-newer", Size: 140}}}

	mergedByUpdatedAt := mergeSameNameModelsByExtensionChain([]*meta.IrModel{base, branchA, branchBNewer})
	if mergedByUpdatedAt == nil {
		t.Fatalf("expected merged model for UpdatedAt tie-break case, got nil")
	}

	var nameByUpdatedAt *meta.IrField
	for _, f := range mergedByUpdatedAt.Fields {
		if f != nil && f.Name == "Name" {
			nameByUpdatedAt = f
			break
		}
	}
	if nameByUpdatedAt == nil || nameByUpdatedAt.TsTypeAnnotation != "branch-b-newer" || nameByUpdatedAt.Size != 140 {
		t.Fatalf("unexpected UpdatedAt tie-break result: %#v", nameByUpdatedAt)
	}

	branchASameTimeLowID := &meta.IrModel{BaseModel: meta.BaseModel{UpdatedAt: time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC), Id: sql.NullString{String: "aaa", Valid: true}}, Name: "Partner", Path: branchAPath, Extends: basePath, Fields: []*meta.IrField{{Name: "Name", TsTypeAnnotation: "branch-a-aaa", Size: 150}}}
	branchBSameTimeHighID := &meta.IrModel{BaseModel: meta.BaseModel{UpdatedAt: time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC), Id: sql.NullString{String: "zzz", Valid: true}}, Name: "Partner", Path: branchBPath, Extends: basePath, Fields: []*meta.IrField{{Name: "Name", TsTypeAnnotation: "branch-b-zzz", Size: 160}}}

	mergedByID := mergeSameNameModelsByExtensionChain([]*meta.IrModel{branchBSameTimeHighID, base, branchASameTimeLowID})
	if mergedByID == nil {
		t.Fatalf("expected merged model for Id tie-break case, got nil")
	}

	var nameByID *meta.IrField
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

	baseA := &meta.IrModel{BaseModel: meta.BaseModel{UpdatedAt: time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC), Id: sql.NullString{String: "a-base", Valid: true}}, Name: "Partner", Path: baseAPath, Fields: []*meta.IrField{{Name: "Name"}}}
	childA := &meta.IrModel{BaseModel: meta.BaseModel{UpdatedAt: time.Date(2026, 3, 15, 11, 0, 0, 0, time.UTC), Id: sql.NullString{String: "a-child", Valid: true}}, Name: "Partner", Path: childAPath, Extends: baseAPath, Fields: []*meta.IrField{{Name: "AField"}}}
	baseB := &meta.IrModel{BaseModel: meta.BaseModel{UpdatedAt: time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC), Id: sql.NullString{String: "b-base", Valid: true}}, Name: "Partner", Path: baseBPath, Fields: []*meta.IrField{{Name: "Name"}}}
	childB := &meta.IrModel{BaseModel: meta.BaseModel{UpdatedAt: time.Date(2026, 3, 15, 13, 0, 0, 0, time.UTC), Id: sql.NullString{String: "b-child", Valid: true}}, Name: "Partner", Path: childBPath, Extends: baseBPath, Fields: []*meta.IrField{{Name: "BField"}}}

	selected := selectSameNameModelsInPrimaryExtensionChain([]*meta.IrModel{baseA, childA, baseB, childB})
	merged := mergeSameNameModelsByExtensionChain(selected)
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
