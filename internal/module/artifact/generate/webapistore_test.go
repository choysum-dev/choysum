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
	round := "HALF_UP"
	field := &meta.IrField{
		BaseModel:                meta.BaseModel{Id: sql.NullString{String: "field-1", Valid: true}},
		Name:                     "Amount",
		FieldType:                "Decimal",
		TsTypeAnnotation:         "number",
		Relation:                 "partner_id",
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
		Selection:                "['allow','deny']",
		Round:                    &round,
	}
	metadata := convertFieldToMetadata(field)
	if metadata.Id == nil || *metadata.Id != "field-1" || metadata.Round == nil || *metadata.Round != round || metadata.Selection == nil {
		t.Fatalf("unexpected field metadata: %#v", metadata)
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

	webStoreDir := t.TempDir()
	storeResults, err := (&webApiStoreGenerator{runtimeScope: runtimeScope, module: &meta.IrModule{ApplicationStr: "crm"}, modulesWebDir: webStoreDir}).generate(context.Background(), testApp())
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
