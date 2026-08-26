// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package hub

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/choysum-dev/choysum/internal/export/proto/exportpb"
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
	seedCountryModelMeta(t, runtimeScope.Session().DB)
	resp, err := describeFields(runtimeScope.Context(), runtimeScope, &exportpb.DescribeFieldsRequest{Model: "base.Country"})
	if err != nil {
		t.Fatalf("describeFields: %v", err)
	}
	if len(resp.GetDefaultFields()) == 0 {
		t.Fatal("expected country defaults")
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
