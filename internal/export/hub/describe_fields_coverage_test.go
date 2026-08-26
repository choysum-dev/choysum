// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package hub

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/choysum-dev/choysum/internal/export/proto/exportpb"
	exportpkg "github.com/choysum-dev/choysum/pkg/export"
	"github.com/choysum-dev/choysum/pkg/meta"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestDescribeFieldsNilRequestDirect(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	_, err := describeFields(runtimeScope.Context(), runtimeScope, nil)
	if err == nil || status.Code(err) != codes.InvalidArgument {
		t.Fatalf("err = %v", err)
	}
}

func TestDescribeFieldsMissingModel(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	_, err := describeFields(runtimeScope.Context(), runtimeScope, &exportpb.DescribeFieldsRequest{})
	if err == nil || status.Code(err) != codes.InvalidArgument {
		t.Fatalf("err = %v", err)
	}
}

func TestDescribeFieldsUnknownModel(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	_, err := describeFields(runtimeScope.Context(), runtimeScope, &exportpb.DescribeFieldsRequest{Model: "missing.App"})
	if err == nil || status.Code(err) != codes.InvalidArgument {
		t.Fatalf("err = %v", err)
	}
}

func TestExportFieldNodeSkipsUnsupportedFields(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	db := runtimeScope.Session().DB
	if node, err := exportFieldNode(db, meta.Field{Name: "Id", FieldType: "varchar"}); err != nil || node != nil {
		t.Fatalf("node=%#v err=%v", node, err)
	}
	if node, err := exportFieldNode(db, meta.Field{Name: "Lines", FieldType: "One2Many"}); err != nil || node != nil {
		t.Fatalf("node=%#v err=%v", node, err)
	}
}

func TestExportFieldNodeManyToOneChildren(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	seedPartnerModelMeta(t, runtimeScope.Session().DB)
	db := runtimeScope.Session().DB
	if err := db.AutoMigrate(&meta.Model{}, &meta.Field{}); err != nil {
		t.Fatal(err)
	}
	company := &meta.Model{Name: "Company", Application: "base", Path: "/tmp", ModelTable: "base_company"}
	if err := db.Create(company).Error; err != nil {
		t.Fatal(err)
	}
	partner := &meta.Model{}
	if err := db.Where("application = ? AND name = ?", "partner", "Partner").First(partner).Error; err != nil {
		t.Fatal(err)
	}
	for _, field := range []meta.Field{
		{Name: "Code", FieldType: "varchar", ModelId: company.Id, FieldString: "Code"},
		{Name: "Name", FieldType: "varchar", ModelId: company.Id, FieldString: "Name"},
	} {
		if err := db.Create(&field).Error; err != nil {
			t.Fatal(err)
		}
	}
	m2o := meta.Field{
		Name:          "CompanyId",
		FieldType:     "ManyToOneRef",
		RelationModel: "base.Company",
		ModelId:       partner.Id,
		FieldString:   "Company",
	}
	node, err := exportFieldNode(db, m2o)
	if err != nil {
		t.Fatalf("exportFieldNode: %v", err)
	}
	if len(node.GetChildren()) != 2 {
		t.Fatalf("children = %#v", node.GetChildren())
	}
}

func TestExportFieldNodeRelationFromResolvedSpec(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	db := runtimeScope.Session().DB
	field := &meta.Field{Name: "CurrencyId", FieldType: "ManyToOneRef", FieldString: "Currency"}
	spec := &meta.FieldResolvedSpec{
		Structural: meta.FieldStructuralSpec{
			Relation: map[string]any{"targetModel": "base.Currency"},
		},
	}
	if err := field.SetResolvedSpec(spec); err != nil {
		t.Fatal(err)
	}
	node, err := exportFieldNode(db, *field)
	if err != nil || node == nil || node.GetPath() != "CurrencyId" {
		t.Fatalf("node=%#v err=%v", node, err)
	}
}

func TestFieldLabelFallback(t *testing.T) {
	if got := fieldLabel(&meta.Field{Name: "Code"}); got != "Code" {
		t.Fatalf("label = %q", got)
	}
	if got := fieldLabel(nil); got != "" {
		t.Fatalf("label = %q", got)
	}
}

func TestShouldSkipExportFieldRelationName(t *testing.T) {
	field := &meta.Field{Name: "Contacts", Relation: "One2Many"}
	if !shouldSkipExportField(field) {
		t.Fatal("expected skip for One2Many relation")
	}
}

func TestImportwriterFieldRelationTargetErrors(t *testing.T) {
	if _, err := importwriterFieldRelationTarget(&meta.Field{Name: "X"}); err == nil {
		t.Fatal("expected missing target")
	}
	if _, err := importwriterFieldRelationTarget(&meta.Field{Name: "X", RelationModel: "Currency"}); err == nil {
		t.Fatal("expected app.Model target")
	}
}

func TestDescribeFieldsUnavailableSession(t *testing.T) {
	_, err := describeFields(context.Background(), nilDBScope{ctx: context.Background()}, &exportpb.DescribeFieldsRequest{Model: "base.Country"})
	if err == nil || status.Code(err) != codes.Unavailable {
		t.Fatalf("err = %v", err)
	}
}

func TestDescribeFieldsDefaultFieldsUnsupportedModel(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	db := runtimeScope.Session().DB
	if err := db.AutoMigrate(&meta.Model{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&meta.Model{Name: "Currency", Application: "base", Path: "/tmp", ModelTable: "base_currency"}).Error; err != nil {
		t.Fatal(err)
	}
	_, err := describeFields(runtimeScope.Context(), runtimeScope, &exportpb.DescribeFieldsRequest{Model: "base.Currency"})
	if err == nil || status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("err = %v", err)
	}
}

func TestExportFieldNodeInvalidRelationTargetInDB(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	db := runtimeScope.Session().DB
	field := meta.Field{
		Name:          "CompanyId",
		FieldType:     "ManyToOneRef",
		RelationModel: "missing.Company",
		FieldString:   "Company",
	}
	node, err := exportFieldNode(db, field)
	if err != nil || node == nil || len(node.GetChildren()) != 0 {
		t.Fatalf("node=%#v err=%v", node, err)
	}
}

func TestExportFieldNodeResolvedSpecBadRelationType(t *testing.T) {
	field := &meta.Field{Name: "CurrencyId", FieldType: "ManyToOneRef"}
	_ = field.SetResolvedSpec(&meta.FieldResolvedSpec{
		Structural: meta.FieldStructuralSpec{
			Relation: map[string]any{"targetModel": 123},
		},
	})
	if _, err := importwriterFieldRelationTarget(field); err == nil {
		t.Fatal("expected relation target error")
	}
}

func TestDescribeFieldsJSONRoundTripDefault(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	seedPartnerModelFields(t, runtimeScope.Session().DB)
	h := New(Deps{RuntimeScope: runtimeScope})
	ctx := authCtx(t)
	resp, err := h.DescribeFields(ctx, &exportpb.DescribeFieldsRequest{Model: "partner.Partner"})
	if err != nil {
		t.Fatalf("DescribeFields: %v", err)
	}
	raw, err := json.Marshal(resp.GetDefaultFields())
	if err != nil || len(raw) == 0 {
		t.Fatalf("marshal defaults: %v raw=%s", err, raw)
	}
}

func TestShouldSkipExportFieldEdgeCases(t *testing.T) {
	if !shouldSkipExportField(nil) {
		t.Fatal("nil field should skip")
	}
	if !shouldSkipExportField(&meta.Field{Name: "Id"}) {
		t.Fatal("Id should skip")
	}
	if !shouldSkipExportField(&meta.Field{Name: "Payload", FieldType: "jsonb"}) {
		t.Fatal("jsonb should skip")
	}
}

func TestImportwriterFieldIsManyToOneNil(t *testing.T) {
	if importwriterFieldIsManyToOne(nil) {
		t.Fatal("nil field is not m2o")
	}
}

func TestImportwriterFieldIsManyToOneRelationName(t *testing.T) {
	if !importwriterFieldIsManyToOne(&meta.Field{Name: "CompanyId", Relation: "ManyToOne"}) {
		t.Fatal("expected ManyToOne relation")
	}
}

func TestExportFieldNodeSkipsNonCodeNameChildren(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	db := runtimeScope.Session().DB
	if err := db.AutoMigrate(&meta.Model{}, &meta.Field{}); err != nil {
		t.Fatal(err)
	}
	company := &meta.Model{Name: "Company", Application: "base", Path: "/tmp", ModelTable: "base_company"}
	if err := db.Create(company).Error; err != nil {
		t.Fatal(err)
	}
	for _, field := range []meta.Field{
		{Name: "Extra", FieldType: "varchar", ModelId: company.Id, FieldString: "Extra"},
	} {
		if err := db.Create(&field).Error; err != nil {
			t.Fatal(err)
		}
	}
	node, err := exportFieldNode(db, meta.Field{
		Name:          "CompanyId",
		FieldType:     "ManyToOneRef",
		RelationModel: "base.Company",
		FieldString:   "Company",
	})
	if err != nil || node == nil || len(node.GetChildren()) != 0 {
		t.Fatalf("node=%#v err=%v", node, err)
	}
}

func TestShouldSkipExportFieldEmptyName(t *testing.T) {
	if !shouldSkipExportField(&meta.Field{Name: "   "}) {
		t.Fatal("expected empty name to skip")
	}
}

func TestExportFieldNodeBasicField(t *testing.T) {
	field := meta.Field{Name: "Name", FieldType: "varchar", FieldString: "Name"}
	node, err := exportFieldNode(newHubTestScope(t).Session().DB, field)
	if err != nil || node == nil || node.GetPath() != "Name" {
		t.Fatalf("node=%#v err=%v", node, err)
	}
}

func TestHubDescribeFieldsUnsupportedExportModel(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	db := runtimeScope.Session().DB
	if err := db.AutoMigrate(&meta.Model{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&meta.Model{Name: "Currency", Application: "base", Path: "/tmp", ModelTable: "base_currency"}).Error; err != nil {
		t.Fatal(err)
	}
	h := New(Deps{RuntimeScope: runtimeScope})
	ctx := authCtx(t)
	_, err := h.DescribeFields(ctx, &exportpb.DescribeFieldsRequest{Model: "base.Currency"})
	if err == nil || status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("err = %v", err)
	}
}

func TestDescribeFieldsSkipsIdFieldInTree(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	seedPartnerModelFields(t, runtimeScope.Session().DB)
	db := runtimeScope.Session().DB
	partnerModel := &meta.Model{}
	if err := db.Where("application = ? AND name = ?", "partner", "Partner").First(partnerModel).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&meta.Field{Name: "Id", FieldType: "varchar", ModelId: partnerModel.Id, FieldString: "Id"}).Error; err != nil {
		t.Fatal(err)
	}
	resp, err := describeFields(runtimeScope.Context(), runtimeScope, &exportpb.DescribeFieldsRequest{Model: "partner.Partner"})
	if err != nil {
		t.Fatalf("describeFields: %v", err)
	}
	for _, node := range resp.GetFields() {
		if node.GetPath() == "Id" {
			t.Fatal("Id field should be skipped")
		}
	}
}

func TestExportFieldNodeUsesFieldNameWhenLabelMissing(t *testing.T) {
	node, err := exportFieldNode(newHubTestScope(t).Session().DB, meta.Field{Name: "Code", FieldType: "varchar"})
	if err != nil || node == nil || node.GetLabel() != "Code" {
		t.Fatalf("node=%#v err=%v", node, err)
	}
}

func TestDescribeFieldsDefaultExportFieldsInternalError(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	seedCountryModelMeta(t, runtimeScope.Session().DB)
	prev := describeDefaultExportFields
	describeDefaultExportFields = func(string) ([]string, error) {
		return nil, exportpkg.Errorf(exportpkg.CodeInvalidSpec, "bad defaults")
	}
	t.Cleanup(func() { describeDefaultExportFields = prev })
	_, err := describeFields(runtimeScope.Context(), runtimeScope, &exportpb.DescribeFieldsRequest{Model: "base.Country"})
	if err == nil || status.Code(err) != codes.Internal {
		t.Fatalf("err = %v", err)
	}
}

func TestDescribeFieldsListFieldsError(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	seedPartnerModelMeta(t, runtimeScope.Session().DB)
	if err := runtimeScope.Session().DB.Exec("DROP TABLE meta_field").Error; err != nil {
		t.Fatal(err)
	}
	_, err := describeFields(runtimeScope.Context(), runtimeScope, &exportpb.DescribeFieldsRequest{Model: "partner.Partner"})
	if err == nil || status.Code(err) != codes.Internal {
		t.Fatalf("err = %v", err)
	}
}

func TestExportFieldNodeListFieldsErrorOnRelation(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	db := runtimeScope.Session().DB
	if err := db.AutoMigrate(&meta.Model{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&meta.Model{Name: "Company", Application: "base", Path: "/tmp", ModelTable: "base_company"}).Error; err != nil {
		t.Fatal(err)
	}
	node, err := exportFieldNode(db, meta.Field{
		Name:          "CompanyId",
		FieldType:     "ManyToOneRef",
		RelationModel: "base.Company",
		FieldString:   "Company",
	})
	if err != nil || node == nil || len(node.GetChildren()) != 0 {
		t.Fatalf("node=%#v err=%v", node, err)
	}
}

func TestShouldSkipExportFieldBinaryAndMany2Many(t *testing.T) {
	if !shouldSkipExportField(&meta.Field{Name: "Attachment", FieldType: "binary"}) {
		t.Fatal("binary should skip")
	}
	if !shouldSkipExportField(&meta.Field{Name: "Tags", Relation: "Many2Many"}) {
		t.Fatal("Many2Many relation should skip")
	}
}

func TestImportwriterFieldRelationTargetResolvedSpecMissingTarget(t *testing.T) {
	field := &meta.Field{Name: "CurrencyId", FieldType: "ManyToOneRef"}
	_ = field.SetResolvedSpec(&meta.FieldResolvedSpec{
		Structural: meta.FieldStructuralSpec{Relation: map[string]any{"targetModel": ""}},
	})
	if _, err := importwriterFieldRelationTarget(field); err == nil {
		t.Fatal("expected missing target")
	}
}
