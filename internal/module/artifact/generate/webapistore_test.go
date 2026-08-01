// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package generator

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"

	"github.com/choysum-dev/choysum/internal/esbplugins"
	"github.com/choysum-dev/choysum/internal/module/artifact/staging"
	"github.com/choysum-dev/choysum/pkg/meta"
)

func TestWebApiStoreGenerate(t *testing.T) {
	runtimeScope := newGeneratorScope(t)
	seedAbstractBaseModel(t, runtimeScope, nil)
	referenceKey := meta.TermReferenceKey("demo", "demo.status.allow", "Allow", "literal")
	stringKey := meta.TermReferenceKey("demo", "demo.model.Partner.fields", "Amount", "literal")
	helpKey := meta.TermReferenceKey("demo", "demo.model.Partner.fields", "Monetary amount in company currency", "literal")
	selectionJSON := `[{"value":"allow","label":"Allow","labelText":{"key":"` + referenceKey + `","module":"demo","scope":"demo.status.allow","src":"Allow","kind":"literal"}}]`
	round := "HALF_UP"
	searchable := true
	field := &meta.IrField{
		BaseModel:                meta.BaseModel{Id: sql.NullString{String: "field-1", Valid: true}},
		Name:                     "Amount",
		FieldType:                "Decimal",
		TsTypeAnnotation:         "number",
		RelationModel:            "Partner",
		RelationFilter:           "active = true",
		RelationModelParentField: "ParentId",
		NotNull:                  true,
		Size:                     18,
		Precision:                12,
		Scale:                    4,
		ScaleField:               "currencyScale",
		IsReadonly:               true,
		Indexed:                  true,
		RelationInverseField:     "Lines",
		RelationJoinModel:        "PartnerTag",
		RelationJoinField:        "PartnerId",
		RelationInverseJoinField: "TagId",
		Selection:                selectionJSON,
		SelectionKind:            "static",
		FieldString:              "Amount",
		StringText:               `{"key":"` + stringKey + `","module":"demo","scope":"demo.model.Partner.fields","src":"Amount","kind":"literal"}`,
		FieldHelp:                "Monetary amount in company currency",
		HelpText:                 `{"key":"` + helpKey + `","module":"demo","scope":"demo.model.Partner.fields","src":"Monetary amount in company currency","kind":"literal"}`,
		Round:                    &round,
	}
	resolvedSpec := &meta.IrFieldResolvedSpec{
		FieldName: "Amount",
		Structural: meta.IrFieldStructuralSpec{
			Related: &meta.IrFieldRelatedSpec{Path: "CurrencyId.Symbol", Store: true},
		},
		Behavior: meta.IrFieldBehaviorSpec{
			Compute: &meta.IrFieldBehaviorComputeSpec{Method: "ComputeAmount", Deps: []string{"CurrencyId"}, Store: true},
		},
		Migration: meta.IrFieldMigrationDecision{StorageKind: "column", ShouldCreateColumn: true, ResolvedColumnType: "NUMERIC(12,4)", ReasonCode: "LEGACY_COLUMN"},
	}
	resolvedSpec.Resolved.Searchable = meta.IrResolvedValue[*bool]{Value: &searchable, Source: "decorator"}
	if err := field.SetResolvedSpec(resolvedSpec); err != nil {
		t.Fatalf("set resolved spec: %v", err)
	}
	metadata := convertFieldToMetadata(field)
	if metadata.Id == nil || *metadata.Id != "field-1" || metadata.Round == nil || *metadata.Round != round || metadata.Selection == nil {
		t.Fatalf("unexpected field metadata: %#v", metadata)
	}
	if metadata.String == nil || *metadata.String != `"Amount"` {
		t.Fatalf("expected quoted string msgid, got %#v", metadata.String)
	}
	if metadata.StringText == nil || !strings.Contains(*metadata.StringText, stringKey) {
		t.Fatalf("expected stringText JSON with key, got %#v", metadata.StringText)
	}
	if metadata.Help == nil || *metadata.Help != `"Monetary amount in company currency"` {
		t.Fatalf("expected quoted help msgid, got %#v", metadata.Help)
	}
	if metadata.HelpText == nil || !strings.Contains(*metadata.HelpText, helpKey) {
		t.Fatalf("expected helpText JSON with key, got %#v", metadata.HelpText)
	}
	if metadata.StorageKind == nil || *metadata.StorageKind != "column" || metadata.ComputedKind == nil || *metadata.ComputedKind != "runtime" {
		t.Fatalf("expected resolved contract fields, got %#v", metadata)
	}
	if metadata.RelatedPath == nil || *metadata.RelatedPath != "CurrencyId.Symbol" || metadata.Searchable == nil || !*metadata.Searchable {
		t.Fatalf("expected related/searchable fields, got %#v", metadata)
	}
	if metadata.ScaleField == nil || *metadata.ScaleField != `"currencyScale"` {
		t.Fatalf("expected quoted scaleField JS literal, got %#v", metadata.ScaleField)
	}
	if metadata.ShouldCreateColumn == nil || !*metadata.ShouldCreateColumn {
		t.Fatalf("expected ShouldCreateColumn=true, got %#v", metadata)
	}
	if metadata.Translate != nil {
		t.Fatalf("non-translate field must omit Translate, got %#v", metadata.Translate)
	}
	encodedMetadata, err := json.Marshal(metadata)
	if err != nil {
		t.Fatalf("marshal field metadata: %v", err)
	}
	if strings.Contains(string(encodedMetadata), `"runAs"`) {
		t.Fatalf("field metadata must omit removed runAs key, got %s", string(encodedMetadata))
	}

	relationFields, importModels := analyzeRelationFields(testApp().Models[1])
	if len(relationFields) != 2 {
		t.Fatalf("expected 2 relation fields, got %d", len(relationFields))
	}
	if len(importModels) != 1 || importModels[0] != "Company" {
		t.Fatalf("unexpected import models: %#v", importModels)
	}
	if NewWebApiStoreGenerator(runtimeScope, &meta.IrModule{Name: "base"}) == nil {
		t.Fatal("expected web api store generator constructor to return non-nil")
	}
	app := testApp()
	if len(app.Models) > 1 && len(app.Models[1].Fields) > 0 {
		app.Models[1].Fields[0].Selection = selectionJSON
		app.Models[1].Fields[0].FieldString = "Amount"
		app.Models[1].Fields[0].StringText = `{"key":"` + stringKey + `","module":"demo","scope":"demo.model.Partner.fields","src":"Amount","kind":"literal"}`
	}
	if len(app.Models) > 1 && len(app.Models[1].Fields) > 1 {
		companyField := app.Models[1].Fields[1]
		if err := companyField.SetResolvedSpec(&meta.IrFieldResolvedSpec{
			FieldName: "CompanyId",
			Behavior:  meta.IrFieldBehaviorSpec{Search: &meta.IrFieldBehaviorMethodRef{Method: "SearchCompany"}},
			Migration: meta.IrFieldMigrationDecision{StorageKind: "column", ShouldCreateColumn: true, ResolvedColumnType: "INTEGER", ReasonCode: "RELATION_DEFAULT"},
		}); err != nil {
			t.Fatalf("set resolved spec on app fixture: %v", err)
		}
	}

	webStoreDir := t.TempDir()
	storeResults, err := (&webApiStoreGenerator{runtimeScope: runtimeScope, module: &meta.IrModule{ApplicationStr: "crm"}, modulesWebDir: webStoreDir}).generate(context.Background(), app)
	if err != nil {
		t.Fatalf("web api store generate() error = %v", err)
	}
	if len(storeResults) != 1 || storeResults[0].Name != "webapistore" {
		t.Fatalf("unexpected web api store results: %#v", storeResults)
	}
	storeContent, err := os.ReadFile(filepath.Join(webStoreDir, "stores", "partner.ts"))
	if err != nil {
		t.Fatalf("read partner.ts: %v", err)
	}
	if !strings.Contains(string(storeContent), "PartnerFieldsMetadata") || !strings.Contains(string(storeContent), "CompanyId") || !strings.Contains(string(storeContent), "relationModel: 'Company'") {
		t.Fatalf("unexpected store content: %s", string(storeContent))
	}
	if strings.Contains(string(storeContent), "relation: '") {
		t.Fatalf("unexpected legacy relation key in store content: %s", string(storeContent))
	}
	if !strings.Contains(string(storeContent), "storageKind") || !strings.Contains(string(storeContent), "searchable") {
		t.Fatalf("expected resolved contract keys in store content: %s", string(storeContent))
	}
	if strings.Contains(string(storeContent), "labelText") {
		t.Fatalf("generated store must not emit selection labelText: %s", string(storeContent))
	}
	if !strings.Contains(string(storeContent), `"label":"Allow"`) {
		t.Fatalf("expected selection label msgid in generated store content: %s", string(storeContent))
	}
	if !strings.Contains(string(storeContent), "selectionKind: 'static'") {
		t.Fatalf("expected selectionKind static in generated store content: %s", string(storeContent))
	}
	if !strings.Contains(string(storeContent), `string: "Amount"`) || !strings.Contains(string(storeContent), "stringText:") || !strings.Contains(string(storeContent), stringKey) {
		t.Fatalf("expected field string/stringText in generated store content: %s", string(storeContent))
	}
	if strings.Contains(string(storeContent), "runAs:") || strings.Contains(string(storeContent), `"runAs"`) {
		t.Fatalf("generated store must omit removed runAs key: %s", string(storeContent))
	}
	if _, err := os.Stat(filepath.Join(webStoreDir, "stores", "index.ts")); err != nil {
		t.Fatalf("expected stores/index.ts: %v", err)
	}
}

func TestWebApiStoreGenerate_DynamicSelectionOmitsInlineArray(t *testing.T) {
	field := &meta.IrField{
		BaseModel:        meta.BaseModel{Id: sql.NullString{String: "field-dyn", Valid: true}},
		Name:             "Status",
		FieldType:        "selection",
		TsTypeAnnotation: "string",
		SelectionKind:    "dynamic",
		SelectionMethod:  "StatusOptions",
		Selection:        `[{"value":"should","label":"NotEmit"}]`,
	}
	metadata := convertFieldToMetadata(field)
	if metadata.SelectionKind == nil || *metadata.SelectionKind != "dynamic" {
		t.Fatalf("expected selectionKind dynamic, got %#v", metadata.SelectionKind)
	}
	if metadata.Selection != nil {
		t.Fatalf("dynamic selection must omit inline selection array, got %#v", metadata.Selection)
	}
}

func TestConvertFieldToMetadata_PrefersResolvedSelectionOverBrokenLegacy(t *testing.T) {
	referenceKey := meta.TermReferenceKey("base", "base.model.Language.fields", "Left to right", "literal")
	field := &meta.IrField{
		BaseModel:        meta.BaseModel{Id: sql.NullString{String: "field-dir", Valid: true}},
		Name:             "Direction",
		FieldType:        "selection",
		TsTypeAnnotation: "string",
		SelectionKind:    "static",
		// Legacy overwrite from raw decorator ObjectLiteral (source text).
		Selection: `[{"value":"ltr","label":" _lt('Left to right', { scope: 'base.model.Language.fields' })"}]`,
	}
	if err := field.SetResolvedSpec(&meta.IrFieldResolvedSpec{
		FieldName: "Direction",
		Structural: meta.IrFieldStructuralSpec{
			SelectionKind: "static",
			Selection: []meta.IrFieldSelectionItem{{
				Value: "ltr",
				Label: "Left to right",
				LabelText: &meta.TermReference{
					Key:    referenceKey,
					Module: "base",
					Scope:  "base.model.Language.fields",
					Src:    "Left to right",
					Kind:   "literal",
				},
			}},
		},
	}); err != nil {
		t.Fatalf("set resolved spec: %v", err)
	}

	metadata := convertFieldToMetadata(field)
	if metadata.Selection == nil {
		t.Fatal("expected selection metadata")
	}
	if !strings.Contains(*metadata.Selection, `"label":"Left to right"`) {
		t.Fatalf("expected msgid label from ResolvedSpec, got %s", *metadata.Selection)
	}
	if strings.Contains(*metadata.Selection, "_lt(") || strings.Contains(*metadata.Selection, "labelText") {
		t.Fatalf("selection must not emit _lt source text or labelText: %s", *metadata.Selection)
	}
}

func TestConvertFieldToMetadata_InfersStaticKindAndStripsLabelText(t *testing.T) {
	field := &meta.IrField{
		BaseModel:        meta.BaseModel{Id: sql.NullString{String: "field-kind", Valid: true}},
		Name:             "Status",
		FieldType:        "selection",
		TsTypeAnnotation: "string",
		// No SelectionKind — should infer static when JSON array is present.
		Selection: `[{"value":"a","label":"A","labelText":{"src":"A"}},null,{"value":"b","label":"B"}]`,
	}
	metadata := convertFieldToMetadata(field)
	if metadata.SelectionKind == nil || *metadata.SelectionKind != "static" {
		t.Fatalf("expected inferred static kind, got %#v", metadata.SelectionKind)
	}
	if metadata.Selection == nil || strings.Contains(*metadata.Selection, "labelText") {
		t.Fatalf("expected stripped selection without labelText: %#v", metadata.Selection)
	}
	if !strings.Contains(*metadata.Selection, `"label":"A"`) || !strings.Contains(*metadata.Selection, `"label":"B"`) {
		t.Fatalf("expected both labels kept: %s", *metadata.Selection)
	}
}

func TestStripSelectionLabelTextJSON_InvalidOrEmpty(t *testing.T) {
	if got := stripSelectionLabelTextJSON(""); got != "" {
		t.Fatalf("empty input: %q", got)
	}
	if got := stripSelectionLabelTextJSON("   "); got != "   " {
		t.Fatalf("whitespace input should pass through: %q", got)
	}
	raw := `not-json`
	if got := stripSelectionLabelTextJSON(raw); got != raw {
		t.Fatalf("invalid json should pass through: %q", got)
	}
}

func TestWebApiStoreGenerateEmptyApp(t *testing.T) {
	runtimeScope := newGeneratorScope(t)
	results, err := (&webApiStoreGenerator{runtimeScope: runtimeScope, module: &meta.IrModule{ApplicationStr: "crm"}, modulesWebDir: t.TempDir()}).generate(context.Background(), &meta.IrApplication{Name: "crm"})
	if err != nil {
		t.Fatalf("generate(empty app) error = %v", err)
	}
	if results != nil {
		t.Fatalf("expected nil results for app without models, got %#v", results)
	}
}

func TestWebApiStoreGenerate_NilContext(t *testing.T) {
	runtimeScope := newGeneratorScope(t)
	seedAbstractBaseModel(t, runtimeScope, nil)
	webStoreDir := t.TempDir()
	results, err := (&webApiStoreGenerator{runtimeScope: runtimeScope, module: &meta.IrModule{ApplicationStr: "crm"}, modulesWebDir: webStoreDir}).generate(nil, &meta.IrApplication{
		Name: "crm",
		Models: []*meta.IrModel{{
			Name: "Partner",
			Path: "@/crm/service/models/partner.ts",
			Services: []*meta.IrService{{
				Name:                  "CreatePartner",
				AccessibilityModifier: "public",
				IsStatic:              true,
			}},
		}},
	})
	if err != nil {
		t.Fatalf("generate(nil ctx) error = %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("unexpected results: %#v", results)
	}
}

func TestWebApiStoreGenerate_UsesWorkspaceGeneratedTargets(t *testing.T) {
	runtimeScope := newGeneratorScope(t)
	seedAbstractBaseModel(t, runtimeScope, nil)
	_, webDir, _, err := WorkspaceGeneratedAPITargets(runtimeScope.cfg.ModulesPath, "crm", runtimeScope.cfg.DefaultChoysumPath)
	if err != nil {
		t.Fatalf("WorkspaceGeneratedAPITargets() error = %v", err)
	}

	gen := &webApiStoreGenerator{runtimeScope: runtimeScope, module: &meta.IrModule{ApplicationStr: "crm"}}
	ctx := staging.WithTmpRoot(context.Background(), t.TempDir())
	results, err := gen.generate(ctx, testApp())
	if err != nil {
		t.Fatalf("generate() error = %v", err)
	}
	if len(results) != 1 || results[0].Name != "webapistore" {
		t.Fatalf("unexpected web api store results: %#v", results)
	}
	if _, err := os.Stat(filepath.Join(webDir, "stores", "partner.ts")); err != nil {
		t.Fatalf("expected partner.ts in workspace target: %v", err)
	}
	if _, err := os.Stat(filepath.Join(webDir, "stores", "index.ts")); err != nil {
		t.Fatalf("expected stores/index.ts in workspace target: %v", err)
	}
}

func TestWebApiStoreGenerate_WorkspaceTargetsRequireDefaultChoysumPath(t *testing.T) {
	runtimeScope := newGeneratorScope(t)
	seedAbstractBaseModel(t, runtimeScope, nil)
	runtimeScope.cfg.DefaultChoysumPath = ""

	_, err := (&webApiStoreGenerator{runtimeScope: runtimeScope, module: &meta.IrModule{ApplicationStr: "crm"}}).generate(context.Background(), testApp())
	if err == nil || !strings.Contains(err.Error(), "resolve workspace generated api targets") {
		t.Fatalf("expected workspace target resolution error, got %v", err)
	}
}

func TestWebApiStoreGenerate_FiltersBaseServicesFromInterface(t *testing.T) {
	runtimeScope := newGeneratorScope(t)
	seedAbstractBaseModel(t, runtimeScope, nil)

	app := &meta.IrApplication{
		Name: "crm",
		Models: []*meta.IrModel{{
			Name: "Partner",
			Path: "@/crm/service/models/partner.ts",
			Services: []*meta.IrService{
				{Name: "Search", AccessibilityModifier: "public", IsStatic: true},
				{Name: "NameSearch", AccessibilityModifier: "public", IsStatic: true},
				{Name: "Copy", AccessibilityModifier: "public", IsStatic: true},
				{Name: "CreatePartner", AccessibilityModifier: "public", IsStatic: true, Parameters: []*meta.IrParameter{{Name: "partner_id", ProtobufType: "string"}}},
			},
		}},
	}
	webStoreDir := t.TempDir()
	_, err := (&webApiStoreGenerator{runtimeScope: runtimeScope, module: &meta.IrModule{ApplicationStr: "crm"}, modulesWebDir: webStoreDir}).generate(context.Background(), app)
	if err != nil {
		t.Fatalf("generate() error = %v", err)
	}
	storeContent, err := os.ReadFile(filepath.Join(webStoreDir, "stores", "partner.ts"))
	if err != nil {
		t.Fatalf("read partner.ts: %v", err)
	}
	content := string(storeContent)
	if !strings.Contains(content, "CreatePartner:") {
		t.Fatalf("expected custom CreatePartner on PartnerStore interface, got:\n%s", content)
	}
	for _, baseName := range []string{"Search", "NameSearch", "Copy"} {
		// Interface lines look like: "  Search: (...args:"
		needle := "  " + baseName + ": (...args:"
		if strings.Contains(content, needle) {
			t.Fatalf("base service %s must not be redeclared on PartnerStore interface, got:\n%s", baseName, content)
		}
	}
}

func TestWebApiStoreGenerate_MissingBaseModelErrors(t *testing.T) {
	runtimeScope := newGeneratorScope(t)
	seedGeneratorMetaTables(t, runtimeScope)

	_, err := (&webApiStoreGenerator{runtimeScope: runtimeScope, module: &meta.IrModule{ApplicationStr: "crm"}, modulesWebDir: t.TempDir()}).generate(context.Background(), testApp())
	if err == nil || !strings.Contains(err.Error(), "BaseModel not found") {
		t.Fatalf("expected BaseModel not found error, got %v", err)
	}
}

func TestWebApiStoreGenerate_EmptyBaseModelServicesErrors(t *testing.T) {
	runtimeScope := newGeneratorScope(t)
	seedAbstractBaseModel(t, runtimeScope, []string{})

	_, err := (&webApiStoreGenerator{runtimeScope: runtimeScope, module: &meta.IrModule{ApplicationStr: "crm"}, modulesWebDir: t.TempDir()}).generate(context.Background(), testApp())
	if err == nil || !strings.Contains(err.Error(), "no conventional services") {
		t.Fatalf("expected empty conventional services error, got %v", err)
	}
}

func TestResolveBaseServiceNames_RejectsNonConventionalOnly(t *testing.T) {
	runtimeScope := newGeneratorScope(t)
	seedGeneratorMetaTables(t, runtimeScope)

	path, _ := meta.BaseModelModuleSpec(runtimeScope)
	path = esbplugins.NormalizePath(path)
	if !strings.HasSuffix(path, ".ts") {
		path = path + ".ts"
	}
	model := &meta.IrModel{
		BaseModel: meta.BaseModel{Id: sql.NullString{String: "abstract-base-model", Valid: true}},
		Name:      "BaseModel",
		Path:      path,
		Abstract:  true,
	}
	if err := runtimeScope.db.Create(model).Error; err != nil {
		t.Fatalf("create BaseModel: %v", err)
	}
	svc := &meta.IrService{
		BaseModel:             meta.BaseModel{Id: sql.NullString{String: "svc-helper", Valid: true}},
		Name:                  "helper",
		AccessibilityModifier: "public",
		IsStatic:              true,
		ModelId:               model.Id,
	}
	if err := runtimeScope.db.Create(svc).Error; err != nil {
		t.Fatalf("create helper service: %v", err)
	}

	_, err := resolveBaseServiceNames(runtimeScope)
	if err == nil || !strings.Contains(err.Error(), "no conventional services") {
		t.Fatalf("expected no conventional services error, got %v", err)
	}
}

func TestResolveBaseServiceNames_RequiresAbstract(t *testing.T) {
	runtimeScope := newGeneratorScope(t)
	seedGeneratorMetaTables(t, runtimeScope)

	path, _ := meta.BaseModelModuleSpec(runtimeScope)
	path = esbplugins.NormalizePath(path)
	if !strings.HasSuffix(path, ".ts") {
		path = path + ".ts"
	}
	model := &meta.IrModel{
		BaseModel: meta.BaseModel{Id: sql.NullString{String: "concrete-at-base-path", Valid: true}},
		Name:      "BaseModel",
		Path:      path,
		Abstract:  false,
	}
	if err := runtimeScope.db.Create(model).Error; err != nil {
		t.Fatalf("create non-abstract model: %v", err)
	}
	svc := &meta.IrService{
		BaseModel:             meta.BaseModel{Id: sql.NullString{String: "svc-search", Valid: true}},
		Name:                  "Search",
		AccessibilityModifier: "public",
		IsStatic:              true,
		ModelId:               model.Id,
	}
	if err := runtimeScope.db.Create(svc).Error; err != nil {
		t.Fatalf("create Search service: %v", err)
	}

	_, err := resolveBaseServiceNames(runtimeScope)
	if err == nil || !strings.Contains(err.Error(), "BaseModel not found") {
		t.Fatalf("expected abstract BaseModel not found error, got %v", err)
	}
}

func TestResolveBaseServiceNamesAtPath_EmptyPath(t *testing.T) {
	runtimeScope := newGeneratorScope(t)
	_, err := resolveBaseServiceNamesAtPath(runtimeScope, "  ")
	if err == nil || !strings.Contains(err.Error(), "base model module path is empty") {
		t.Fatalf("expected empty path error, got %v", err)
	}
}

func TestResolveBaseServiceNames_LoadError(t *testing.T) {
	runtimeScope := newGeneratorScope(t)
	seedGeneratorMetaTables(t, runtimeScope)
	sqlDB, err := runtimeScope.db.DB()
	if err != nil {
		t.Fatalf("db: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}

	_, err = resolveBaseServiceNames(runtimeScope)
	if err == nil || !strings.Contains(err.Error(), "load abstract BaseModel by path") {
		t.Fatalf("expected load error, got %v", err)
	}
}

func TestConventionalBaseServiceNames_SkipsNilAndNonConventional(t *testing.T) {
	names := conventionalBaseServiceNames([]*meta.IrService{
		nil,
		{Name: "helper", AccessibilityModifier: "public", IsStatic: true},
		{Name: "Search", AccessibilityModifier: "public", IsStatic: true},
		{Name: "Browse", AccessibilityModifier: "private", IsStatic: true},
	})
	if !names["Search"] {
		t.Fatalf("expected Search, got %#v", names)
	}
	if names["helper"] || names["Browse"] || len(names) != 1 {
		t.Fatalf("unexpected names: %#v", names)
	}
}

func TestConvertFieldToMetadata_TranslateContract(t *testing.T) {
	trueVal := true
	size := 200
	field := &meta.IrField{Name: "Name", FieldType: "varchar"}
	spec := &meta.IrFieldResolvedSpec{
		FieldName: "Name",
		Structural: meta.IrFieldStructuralSpec{
			Translate: &trueVal,
			StorageHints: &meta.IrFieldStructuralStorageHints{
				Size: &size,
			},
		},
	}
	if err := field.SetResolvedSpec(spec); err != nil {
		t.Fatalf("SetResolvedSpec: %v", err)
	}
	metadata := convertFieldToMetadata(field)
	if metadata.Translate == nil || !*metadata.Translate {
		t.Fatalf("expected Translate=true, got %#v", metadata.Translate)
	}
	if metadata.Size == nil || *metadata.Size != 200 {
		t.Fatalf("expected Size from storage hints, got %#v", metadata.Size)
	}

	falseVal := false
	field2 := &meta.IrField{Name: "Code", FieldType: "varchar", Size: 40}
	spec2 := &meta.IrFieldResolvedSpec{
		FieldName: "Code",
		Structural: meta.IrFieldStructuralSpec{
			Translate: &falseVal,
		},
	}
	if err := field2.SetResolvedSpec(spec2); err != nil {
		t.Fatalf("SetResolvedSpec false: %v", err)
	}
	metadata2 := convertFieldToMetadata(field2)
	if metadata2.Translate != nil {
		t.Fatalf("translate:false must omit Translate flag, got %#v", metadata2.Translate)
	}

	field3 := &meta.IrField{Name: "Title", FieldType: "varchar", Size: 80}
	zeroSize := 0
	spec3 := &meta.IrFieldResolvedSpec{
		FieldName: "Title",
		Structural: meta.IrFieldStructuralSpec{
			Translate: &trueVal,
			StorageHints: &meta.IrFieldStructuralStorageHints{
				Size: &zeroSize,
			},
		},
	}
	if err := field3.SetResolvedSpec(spec3); err != nil {
		t.Fatalf("SetResolvedSpec zero size: %v", err)
	}
	metadata3 := convertFieldToMetadata(field3)
	if metadata3.Translate == nil || !*metadata3.Translate {
		t.Fatalf("expected Translate=true for field3, got %#v", metadata3.Translate)
	}
	if metadata3.Size == nil || *metadata3.Size != 80 {
		t.Fatalf("existing Size must win over zero storage hint, got %#v", metadata3.Size)
	}
}

func TestConvertFieldToMetadata_CompanyDependentContract(t *testing.T) {
	trueVal := true
	size := 120
	field := &meta.IrField{Name: "Cost", FieldType: "float"}
	spec := &meta.IrFieldResolvedSpec{
		FieldName: "Cost",
		Structural: meta.IrFieldStructuralSpec{
			CompanyDependent: &trueVal,
			StorageHints: &meta.IrFieldStructuralStorageHints{
				Size: &size,
			},
		},
	}
	if err := field.SetResolvedSpec(spec); err != nil {
		t.Fatalf("SetResolvedSpec: %v", err)
	}
	metadata := convertFieldToMetadata(field)
	if metadata.CompanyDependent == nil || !*metadata.CompanyDependent {
		t.Fatalf("expected CompanyDependent=true, got %#v", metadata.CompanyDependent)
	}
	if metadata.Size == nil || *metadata.Size != 120 {
		t.Fatalf("expected Size from storage hints, got %#v", metadata.Size)
	}

	falseVal := false
	field2 := &meta.IrField{Name: "Note", FieldType: "varchar", Size: 40}
	spec2 := &meta.IrFieldResolvedSpec{
		FieldName: "Note",
		Structural: meta.IrFieldStructuralSpec{
			CompanyDependent: &falseVal,
		},
	}
	if err := field2.SetResolvedSpec(spec2); err != nil {
		t.Fatalf("SetResolvedSpec false: %v", err)
	}
	metadata2 := convertFieldToMetadata(field2)
	if metadata2.CompanyDependent != nil {
		t.Fatalf("companyDependent:false must omit CompanyDependent flag, got %#v", metadata2.CompanyDependent)
	}
}

func TestConvertFieldToMetadata_CopyContract(t *testing.T) {
	falseVal := false
	field := &meta.IrField{Name: "Code", FieldType: "varchar"}
	spec := &meta.IrFieldResolvedSpec{
		FieldName: "Code",
		Structural: meta.IrFieldStructuralSpec{
			Copy: &falseVal,
		},
	}
	if err := field.SetResolvedSpec(spec); err != nil {
		t.Fatalf("SetResolvedSpec: %v", err)
	}
	metadata := convertFieldToMetadata(field)
	if metadata.Copy == nil || *metadata.Copy {
		t.Fatalf("expected Copy=false, got %#v", metadata.Copy)
	}

	trueVal := true
	field2 := &meta.IrField{Name: "Name", FieldType: "varchar"}
	spec2 := &meta.IrFieldResolvedSpec{
		FieldName: "Name",
		Structural: meta.IrFieldStructuralSpec{
			Copy: &trueVal,
		},
	}
	if err := field2.SetResolvedSpec(spec2); err != nil {
		t.Fatalf("SetResolvedSpec true: %v", err)
	}
	metadata2 := convertFieldToMetadata(field2)
	if metadata2.Copy != nil {
		t.Fatalf("copy:true must omit Copy flag, got %#v", metadata2.Copy)
	}
}

func TestConvertFieldToMetadata_OmitsEmptyScaleAndCurrencyFields(t *testing.T) {
	field := &meta.IrField{
		Name:          "Amount",
		FieldType:     "Monetary",
		CurrencyField: "",
		ScaleField:    "",
	}
	metadata := convertFieldToMetadata(field)
	if metadata.CurrencyField != nil {
		t.Fatalf("expected nil CurrencyField for empty string, got %#v", metadata.CurrencyField)
	}
	if metadata.ScaleField != nil {
		t.Fatalf("expected nil ScaleField for empty string, got %#v", metadata.ScaleField)
	}
}

func TestConvertFieldToMetadata_QuotesCurrencyFieldEscapes(t *testing.T) {
	field := &meta.IrField{
		Name:          "Amount",
		FieldType:     "Monetary",
		CurrencyField: `Cur"Id\Path`,
		ScaleField:    " scale Field ",
	}
	metadata := convertFieldToMetadata(field)
	if metadata.CurrencyField == nil || *metadata.CurrencyField != `"Cur\"Id\\Path"` {
		t.Fatalf("expected escaped currencyField JS literal, got %#v", metadata.CurrencyField)
	}
	if metadata.ScaleField == nil || *metadata.ScaleField != `"scale Field"` {
		t.Fatalf("expected trimmed quoted scaleField JS literal, got %#v", metadata.ScaleField)
	}

	tpl := template.Must(template.New("field").Parse(`{{- if .CurrencyField}}currencyField: {{.CurrencyField}},{{- end}}`))
	var buf strings.Builder
	if err := tpl.Execute(&buf, metadata); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if got := buf.String(); got != `currencyField: "Cur\"Id\\Path",` {
		t.Fatalf("expected safe template emission, got %q", got)
	}
}

func TestConvertFieldToMetadata_ImageUploadLimits(t *testing.T) {
	field := &meta.IrField{
		Name:           "Avatar",
		FieldType:      "image",
		MaxUploadBytes: 2097152,
		MaxWidth:       1024,
		MaxHeight:      768,
	}
	metadata := convertFieldToMetadata(field)
	if metadata.MaxUploadBytes == nil || *metadata.MaxUploadBytes != 2097152 {
		t.Fatalf("expected MaxUploadBytes=2097152, got %#v", metadata.MaxUploadBytes)
	}
	if metadata.MaxWidth == nil || *metadata.MaxWidth != 1024 {
		t.Fatalf("expected MaxWidth=1024, got %#v", metadata.MaxWidth)
	}
	if metadata.MaxHeight == nil || *metadata.MaxHeight != 768 {
		t.Fatalf("expected MaxHeight=768, got %#v", metadata.MaxHeight)
	}

	plain := convertFieldToMetadata(&meta.IrField{Name: "Photo", FieldType: "image"})
	if plain.MaxUploadBytes != nil || plain.MaxWidth != nil || plain.MaxHeight != nil {
		t.Fatalf("zero limits must omit metadata pointers, got %#v", plain)
	}

	tpl := template.Must(template.New("field").Parse(
		`{{- if .MaxUploadBytes}}maxUploadBytes: {{.MaxUploadBytes}},{{- end}}` +
			`{{- if .MaxWidth}}maxWidth: {{.MaxWidth}},{{- end}}` +
			`{{- if .MaxHeight}}maxHeight: {{.MaxHeight}},{{- end}}`,
	))
	var buf strings.Builder
	if err := tpl.Execute(&buf, metadata); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if got := buf.String(); got != "maxUploadBytes: 2097152,maxWidth: 1024,maxHeight: 768," {
		t.Fatalf("expected template emission, got %q", got)
	}
}

func TestWebApiStoreTemplate_EmitsCopyFalse(t *testing.T) {
	falseVal := false
	tpl := template.Must(template.New("field").Parse(`{{- if ne .Copy nil}}copy: false,{{- end}}`))
	var buf strings.Builder
	if err := tpl.Execute(&buf, FieldMetadata{Copy: &falseVal}); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if got := buf.String(); got != "copy: false," {
		t.Fatalf("expected copy: false emission, got %q", got)
	}

	buf.Reset()
	if err := tpl.Execute(&buf, FieldMetadata{}); err != nil {
		t.Fatalf("execute nil: %v", err)
	}
	if got := buf.String(); got != "" {
		t.Fatalf("expected omit when Copy nil, got %q", got)
	}
}
