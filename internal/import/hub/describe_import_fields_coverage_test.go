// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package hub

import (
	"context"
	"errors"
	"testing"

	"github.com/choysum-dev/choysum/internal/import/proto/importpb"
	"github.com/choysum-dev/choysum/pkg/meta"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func TestDescribeImportFieldsNilRequestDirect(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	_, err := describeImportFields(runtimeScope.Context(), runtimeScope, nil)
	if err == nil || status.Code(err) != codes.InvalidArgument {
		t.Fatalf("err = %v", err)
	}
}

func TestDescribeImportFieldsMissingModelDirect(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	_, err := describeImportFields(runtimeScope.Context(), runtimeScope, &importpb.DescribeImportFieldsRequest{})
	if err == nil || status.Code(err) != codes.InvalidArgument {
		t.Fatalf("err = %v", err)
	}
}

func TestDescribeImportFieldsUnknownModelDirect(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	_, err := describeImportFields(runtimeScope.Context(), runtimeScope, &importpb.DescribeImportFieldsRequest{Model: "missing.App"})
	if err == nil || status.Code(err) != codes.InvalidArgument {
		t.Fatalf("err = %v", err)
	}
}

func TestDescribeImportFieldsUnavailableSession(t *testing.T) {
	_, err := describeImportFields(context.Background(), nilDBScope{ctx: context.Background()}, &importpb.DescribeImportFieldsRequest{Model: "base.Country"})
	if err == nil || status.Code(err) != codes.Unavailable {
		t.Fatalf("err = %v", err)
	}
}

func TestDescribeImportFieldsDefaultFieldsUnsupportedModelDirect(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	seedCountryModelMeta(t, runtimeScope.Session().DB)
	db := runtimeScope.Session().DB
	if err := db.Create(&meta.Model{Name: "Currency", Application: "base", Path: "/tmp", ModelTable: "base_currency"}).Error; err != nil {
		// Currency may already exist from other tests; ignore duplicate.
		_ = err
	}
	if err := db.Where("application = ? AND name = ?", "base", "Currency").FirstOrCreate(&meta.Model{
		Name: "Currency", Application: "base", Path: "/tmp", ModelTable: "base_currency",
	}).Error; err != nil {
		t.Fatalf("seed currency: %v", err)
	}
	_, err := describeImportFields(runtimeScope.Context(), runtimeScope, &importpb.DescribeImportFieldsRequest{Model: "base.Currency"})
	if err == nil || status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("err = %v", err)
	}
}

func TestDescribeImportFieldsDefaultFieldsInternalError(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	seedCountryModelMeta(t, runtimeScope.Session().DB)
	prev := describeDefaultImportFields
	describeDefaultImportFields = func(string) ([]string, error) {
		return nil, errors.New("bad defaults")
	}
	t.Cleanup(func() { describeDefaultImportFields = prev })
	_, err := describeImportFields(runtimeScope.Context(), runtimeScope, &importpb.DescribeImportFieldsRequest{Model: "base.Country"})
	if err == nil || status.Code(err) != codes.Internal {
		t.Fatalf("err = %v", err)
	}
}

func TestDescribeImportFieldsListFieldsError(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	seedPartnerModelMeta(t, runtimeScope.Session().DB)
	prev := listDescribeImportModelFields
	listDescribeImportModelFields = func(*gorm.DB, *meta.Model) ([]meta.Field, error) {
		return nil, errors.New("list failed")
	}
	t.Cleanup(func() { listDescribeImportModelFields = prev })
	_, err := describeImportFields(runtimeScope.Context(), runtimeScope, &importpb.DescribeImportFieldsRequest{Model: "partner.Partner"})
	if err == nil || status.Code(err) != codes.Internal {
		t.Fatalf("err = %v", err)
	}
}

func TestImportFieldNodeSkipsUnsupportedFields(t *testing.T) {
	db := newHubTestScope(t).Session().DB
	cases := []meta.Field{
		{Name: "Id", FieldType: "varchar"},
		{Name: "   ", FieldType: "varchar"},
		{Name: "Blob", FieldType: "binary"},
		{Name: "Meta", FieldType: "json"},
		{Name: "Lines", FieldType: "One2Many"},
		{Name: "Tags", FieldType: "Many2Many"},
		{Name: "Contacts", Relation: "One2Many"},
		{Name: "Links", Relation: "Many2Many"},
		{Name: "Readonly", FieldType: "varchar", IsReadonly: true},
	}
	for _, field := range cases {
		node, err := importFieldNode(db, field)
		if err != nil || node != nil {
			t.Fatalf("field=%#v node=%#v err=%v", field, node, err)
		}
	}
}

func TestImportFieldNodeBasicAndLabelFallback(t *testing.T) {
	db := newHubTestScope(t).Session().DB
	node, err := importFieldNode(db, meta.Field{Name: "Name", FieldType: "varchar", FieldString: "Display"})
	if err != nil || node == nil || node.GetPath() != "Name" || node.GetLabel() != "Display" {
		t.Fatalf("node=%#v err=%v", node, err)
	}
	node, err = importFieldNode(db, meta.Field{Name: "Code", FieldType: "varchar"})
	if err != nil || node == nil || node.GetLabel() != "Code" {
		t.Fatalf("node=%#v err=%v", node, err)
	}
	if got := importFieldLabel(nil); got != "" {
		t.Fatalf("label = %q", got)
	}
}

func TestImportFieldNodeManyToOneChildren(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	seedPartnerModelMeta(t, runtimeScope.Session().DB)
	db := runtimeScope.Session().DB
	if err := db.AutoMigrate(&meta.Model{}, &meta.Field{}); err != nil {
		t.Fatal(err)
	}
	company := &meta.Model{Name: "Company", Application: "base", Path: "/tmp", ModelTable: "base_company"}
	if err := db.Where("application = ? AND name = ?", "base", "Company").FirstOrCreate(company).Error; err != nil {
		t.Fatal(err)
	}
	partner := &meta.Model{}
	if err := db.Where("application = ? AND name = ?", "partner", "Partner").First(partner).Error; err != nil {
		t.Fatal(err)
	}
	for _, field := range []meta.Field{
		{Name: "Code", FieldType: "varchar", ModelId: company.Id, FieldString: "Code"},
		{Name: "Name", FieldType: "varchar", ModelId: company.Id, FieldString: "Name"},
		{Name: "Id", FieldType: "varchar", ModelId: company.Id},
		{Name: "Notes", FieldType: "varchar", ModelId: company.Id},
	} {
		_ = db.Where("model_id = ? AND name = ?", company.Id, field.Name).FirstOrCreate(&field).Error
	}
	m2o := meta.Field{
		Name:          "CompanyId",
		FieldType:     "ManyToOneRef",
		RelationModel: "base.Company",
		ModelId:       partner.Id,
		FieldString:   "Company",
	}
	node, err := importFieldNode(db, m2o)
	if err != nil {
		t.Fatalf("importFieldNode: %v", err)
	}
	if len(node.GetChildren()) != 2 {
		t.Fatalf("children = %#v", node.GetChildren())
	}
}

func TestImportFieldNodeSkipsReadonlyRelationChildren(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	db := runtimeScope.Session().DB
	if err := db.AutoMigrate(&meta.Model{}, &meta.Field{}); err != nil {
		t.Fatal(err)
	}
	company := &meta.Model{Name: "ImportRelSkipCo", Application: "hub", Path: "/tmp", ModelTable: "hub_rel_skip"}
	if err := db.Where("application = ? AND name = ?", "hub", "ImportRelSkipCo").FirstOrCreate(company).Error; err != nil {
		t.Fatal(err)
	}
	for _, field := range []meta.Field{
		{Name: "Code", FieldType: "varchar", ModelId: company.Id, IsReadonly: true},
		{Name: "Name", FieldType: "varchar", ModelId: company.Id, IsReadonly: true},
	} {
		_ = db.Where("model_id = ? AND name = ?", company.Id, field.Name).FirstOrCreate(&field).Error
	}
	node, err := importFieldNode(db, meta.Field{
		Name: "CompanyId", FieldType: "ManyToOneRef", RelationModel: "hub.ImportRelSkipCo", FieldString: "Company",
	})
	if err != nil || node == nil || len(node.GetChildren()) != 0 {
		t.Fatalf("node=%#v err=%v", node, err)
	}
}

func TestImportFieldNodeRelationFromResolvedSpec(t *testing.T) {
	db := newHubTestScope(t).Session().DB
	field := &meta.Field{Name: "CurrencyId", FieldType: "ManyToOneRef", FieldString: "Currency"}
	spec := &meta.FieldResolvedSpec{
		Structural: meta.FieldStructuralSpec{
			Relation: map[string]any{"targetModel": "base.Currency"},
		},
	}
	if err := field.SetResolvedSpec(spec); err != nil {
		t.Fatal(err)
	}
	node, err := importFieldNode(db, *field)
	if err != nil || node == nil || node.GetPath() != "CurrencyId" {
		t.Fatalf("node=%#v err=%v", node, err)
	}
}

func TestImportFieldNodeMissingRelationTarget(t *testing.T) {
	db := newHubTestScope(t).Session().DB
	node, err := importFieldNode(db, meta.Field{Name: "CompanyId", FieldType: "ManyToOneRef", FieldString: "Company"})
	if err != nil || node == nil || len(node.GetChildren()) != 0 {
		t.Fatalf("node=%#v err=%v", node, err)
	}
}

func TestImportFieldNodeInvalidRelationTargetFormat(t *testing.T) {
	db := newHubTestScope(t).Session().DB
	node, err := importFieldNode(db, meta.Field{
		Name: "CompanyId", FieldType: "ManyToOneRef", RelationModel: "Company", FieldString: "Company",
	})
	if err != nil || node == nil || len(node.GetChildren()) != 0 {
		t.Fatalf("node=%#v err=%v", node, err)
	}
}

func TestImportFieldNodeUnknownRelationModel(t *testing.T) {
	db := newHubTestScope(t).Session().DB
	node, err := importFieldNode(db, meta.Field{
		Name: "CompanyId", FieldType: "ManyToOneRef", RelationModel: "missing.Company", FieldString: "Company",
	})
	if err != nil || node == nil || len(node.GetChildren()) != 0 {
		t.Fatalf("node=%#v err=%v", node, err)
	}
}

func TestFieldIsManyToOneVariants(t *testing.T) {
	if fieldIsManyToOne(nil) {
		t.Fatal("nil")
	}
	if !fieldIsManyToOne(&meta.Field{FieldType: "ManyToOne"}) {
		t.Fatal("ManyToOne")
	}
	if !fieldIsManyToOne(&meta.Field{Relation: "ManyToOne"}) {
		t.Fatal("Relation ManyToOne")
	}
	if fieldIsManyToOne(&meta.Field{FieldType: "varchar"}) {
		t.Fatal("varchar")
	}
}

func TestFieldRelationTargetErrors(t *testing.T) {
	if _, err := fieldRelationTarget(&meta.Field{Name: "X"}); err == nil {
		t.Fatal("expected missing target")
	}
	if _, err := fieldRelationTarget(&meta.Field{Name: "X", RelationModel: "Currency"}); err == nil {
		t.Fatal("expected app.Model target")
	}
	got, err := fieldRelationTarget(&meta.Field{Name: "X", RelationModel: "base.Company"})
	if err != nil || got != "base.Company" {
		t.Fatalf("got=%q err=%v", got, err)
	}
}

func TestShouldSkipImportFieldNil(t *testing.T) {
	if !shouldSkipImportField(nil) {
		t.Fatal("expected nil skip")
	}
}

func TestDescribeImportFieldsSkipsBrokenNodes(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	seedPartnerModelMeta(t, runtimeScope.Session().DB)
	db := runtimeScope.Session().DB
	partner := &meta.Model{}
	if err := db.Where("application = ? AND name = ?", "partner", "Partner").First(partner).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&meta.Field{}); err != nil {
		t.Fatal(err)
	}
	_ = db.Create(&meta.Field{Name: "Name", FieldType: "varchar", ModelId: partner.Id}).Error
	_ = db.Create(&meta.Field{Name: "Id", FieldType: "varchar", ModelId: partner.Id}).Error
	resp, err := describeImportFields(runtimeScope.Context(), runtimeScope, &importpb.DescribeImportFieldsRequest{Model: "partner.Partner"})
	if err != nil {
		t.Fatalf("describeImportFields: %v", err)
	}
	for _, node := range resp.GetFields() {
		if node.GetPath() == "Id" {
			t.Fatal("Id should be skipped")
		}
	}
}

func TestHubDescribeImportFieldsAuthEdges(t *testing.T) {
	h := New(Deps{})
	if _, err := h.DescribeImportFields(context.Background(), &importpb.DescribeImportFieldsRequest{Model: "base.Country"}); err == nil {
		t.Fatal("expected unauthenticated")
	}
	ctx := authCtx(t)
	if _, err := h.DescribeImportFields(ctx, &importpb.DescribeImportFieldsRequest{Model: "base.Country"}); err == nil || status.Code(err) != codes.Unavailable {
		t.Fatalf("nil scope err = %v", err)
	}
	h = New(Deps{RuntimeScope: newHubTestScope(t)})
	if _, err := h.DescribeImportFields(ctx, nil); err == nil || status.Code(err) != codes.InvalidArgument {
		t.Fatalf("nil req err = %v", err)
	}
}

func TestHubDescribeImportFieldsAccessDenied(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	seedCountryModelMeta(t, runtimeScope.Session().DB)
	h := New(Deps{RuntimeScope: runtimeScope})
	ctx := authCtxWithServer(t, denyAuthServer{})
	if _, err := h.DescribeImportFields(ctx, &importpb.DescribeImportFieldsRequest{Model: "base.Country"}); err == nil {
		t.Fatal("expected denied")
	}
}

func TestImportFieldNodeListRelFieldsError(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	db := runtimeScope.Session().DB
	if err := db.AutoMigrate(&meta.Model{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&meta.Model{Name: "ImportListErrCo", Application: "hub", Path: "/tmp", ModelTable: "hub_list_err"}).Error; err != nil {
		t.Fatal(err)
	}
	// meta_field table intentionally absent so ListFields fails.
	node, err := importFieldNode(db, meta.Field{
		Name: "CompanyId", FieldType: "ManyToOneRef", RelationModel: "hub.ImportListErrCo", FieldString: "Company",
	})
	if err != nil || node == nil || len(node.GetChildren()) != 0 {
		t.Fatalf("node=%#v err=%v", node, err)
	}
}
