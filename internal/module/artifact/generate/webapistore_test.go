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
	if metadata.StorageKind == nil || *metadata.StorageKind != "column" || metadata.ComputedKind == nil || *metadata.ComputedKind != "runtime" {
		t.Fatalf("expected resolved contract fields, got %#v", metadata)
	}
	if metadata.RelatedPath == nil || *metadata.RelatedPath != "CurrencyId.Symbol" || metadata.Searchable == nil || !*metadata.Searchable {
		t.Fatalf("expected related/searchable fields, got %#v", metadata)
	}
	if metadata.RunAs == nil || *metadata.RunAs != "system" || metadata.ShouldCreateColumn == nil || !*metadata.ShouldCreateColumn {
		t.Fatalf("expected migration/runAs fields, got %#v", metadata)
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
	if !strings.Contains(string(storeContent), "labelText") || !strings.Contains(string(storeContent), referenceKey) {
		t.Fatalf("expected selection term reference in generated store content: %s", string(storeContent))
	}
	if _, err := os.Stat(filepath.Join(webStoreDir, "stores", "index.ts")); err != nil {
		t.Fatalf("expected stores/index.ts: %v", err)
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
