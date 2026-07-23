// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package generator

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/internal/module/artifact/staging"
	"github.com/choysum-dev/choysum/pkg/meta"
)

func TestWebApiStoreGenerate(t *testing.T) {
	runtimeScope := newGeneratorScope(t)
	referenceKey := meta.TermReferenceKey("demo", "demo.status.allow", "Allow", "literal")
	stringKey := meta.TermReferenceKey("demo", "demo.model.Partner.fields", "Amount", "literal")
	selectionJSON := `[{"value":"allow","label":"Allow","labelText":{"key":"` + referenceKey + `","module":"demo","scope":"demo.status.allow","src":"Allow","kind":"literal"}}]`
	round := "HALF_UP"
	searchable := true
	runAs := "system"
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
		Round:                    &round,
	}
	resolvedSpec := &meta.IrFieldResolvedSpec{
		FieldName: "Amount",
		Structural: meta.IrFieldStructuralSpec{
			Related: &meta.IrFieldRelatedSpec{Path: "CurrencyId.Symbol", Store: true},
		},
		Behavior: meta.IrFieldBehaviorSpec{
			Compute: &meta.IrFieldBehaviorComputeSpec{Method: "ComputeAmount", Deps: []string{"CurrencyId"}, Store: true, RunAs: "user"},
		},
		Migration: meta.IrFieldMigrationDecision{StorageKind: "column", ShouldCreateColumn: true, ResolvedColumnType: "NUMERIC(12,4)", ReasonCode: "LEGACY_COLUMN"},
	}
	resolvedSpec.Resolved.Searchable = meta.IrResolvedValue[*bool]{Value: &searchable, Source: "decorator"}
	resolvedSpec.Resolved.RunAs = meta.IrResolvedValue[*string]{Value: &runAs, Source: "decorator"}
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
	if metadata.StorageKind == nil || *metadata.StorageKind != "column" || metadata.ComputedKind == nil || *metadata.ComputedKind != "runtime" {
		t.Fatalf("expected resolved contract fields, got %#v", metadata)
	}
	if metadata.RelatedPath == nil || *metadata.RelatedPath != "CurrencyId.Symbol" || metadata.Searchable == nil || !*metadata.Searchable {
		t.Fatalf("expected related/searchable fields, got %#v", metadata)
	}
	if metadata.RunAs == nil || *metadata.RunAs != "system" || metadata.ShouldCreateColumn == nil || !*metadata.ShouldCreateColumn {
		t.Fatalf("expected migration/runAs fields, got %#v", metadata)
	}
	if metadata.Translate != nil {
		t.Fatalf("non-translate field must omit Translate, got %#v", metadata.Translate)
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
	if _, err := os.Stat(filepath.Join(webStoreDir, "stores", "index.ts")); err != nil {
		t.Fatalf("expected stores/index.ts: %v", err)
	}
}

func TestWebApiStoreGenerate_DynamicSelectionOmitsInlineArray(t *testing.T) {
	field := &meta.IrField{
		BaseModel:       meta.BaseModel{Id: sql.NullString{String: "field-dyn", Valid: true}},
		Name:            "Status",
		FieldType:       "selection",
		TsTypeAnnotation: "string",
		SelectionKind:   "dynamic",
		SelectionMethod: "StatusOptions",
		Selection:       `[{"value":"should","label":"NotEmit"}]`,
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
	raw := `not-json`;
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

func TestWebApiStoreGenerate_UsesWorkspaceGeneratedTargets(t *testing.T) {
	runtimeScope := newGeneratorScope(t)
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
	runtimeScope.cfg.DefaultChoysumPath = ""

	_, err := (&webApiStoreGenerator{runtimeScope: runtimeScope, module: &meta.IrModule{ApplicationStr: "crm"}}).generate(context.Background(), testApp())
	if err == nil || !strings.Contains(err.Error(), "resolve workspace generated api targets") {
		t.Fatalf("expected workspace target resolution error, got %v", err)
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
