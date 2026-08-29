// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package hub

import (
	"testing"

	"github.com/choysum-dev/choysum/internal/import/proto/importpb"
	"github.com/choysum-dev/choysum/pkg/meta"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestDescribeImportFields(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	seedPartnerModelMeta(t, runtimeScope.Session().DB)
	db := runtimeScope.Session().DB
	partnerModel := &meta.Model{}
	if err := db.Where("application = ? AND name = ?", "partner", "Partner").First(partnerModel).Error; err != nil {
		t.Fatalf("load partner model: %v", err)
	}
	if err := db.AutoMigrate(&meta.Field{}); err != nil {
		t.Fatalf("AutoMigrate fields: %v", err)
	}
	modelID := partnerModel.Id
	for _, field := range []meta.Field{
		{Name: "Name", FieldType: "varchar", ModelId: modelID, FieldString: "Name"},
		{Name: "Code", FieldType: "varchar", ModelId: modelID, FieldString: "Code"},
		{Name: "CompanyId", FieldType: "ManyToOneRef", RelationModel: "base.Company", ModelId: modelID, FieldString: "Company"},
	} {
		if err := db.Create(&field).Error; err != nil {
			t.Fatalf("seed field %s: %v", field.Name, err)
		}
	}

	h := New(Deps{RuntimeScope: runtimeScope})
	ctx := authCtx(t)
	resp, err := h.DescribeImportFields(ctx, &importpb.DescribeImportFieldsRequest{Model: "partner.Partner"})
	if err != nil {
		t.Fatalf("DescribeImportFields: %v", err)
	}
	if len(resp.GetFields()) < 2 {
		t.Fatalf("fields = %#v", resp.GetFields())
	}
	if len(resp.GetDefaultFields()) == 0 {
		t.Fatalf("default_fields empty")
	}
}

func TestDescribeImportFieldsRequiresModel(t *testing.T) {
	h := New(Deps{RuntimeScope: newHubTestScope(t)})
	ctx := authCtx(t)
	_, err := h.DescribeImportFields(ctx, &importpb.DescribeImportFieldsRequest{})
	if err == nil || status.Code(err) != codes.InvalidArgument {
		t.Fatalf("DescribeImportFields() err = %v", err)
	}
}

func TestDescribeImportFieldsUnsupportedModel(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	db := runtimeScope.Session().DB
	if err := db.AutoMigrate(&meta.Model{}, &meta.Field{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	model := &meta.Model{Application: "base", Name: "Currency"}
	if err := db.Create(model).Error; err != nil {
		t.Fatalf("seed currency model: %v", err)
	}
	h := New(Deps{RuntimeScope: runtimeScope})
	ctx := authCtx(t)
	_, err := h.DescribeImportFields(ctx, &importpb.DescribeImportFieldsRequest{Model: "base.Currency"})
	if err == nil || status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("DescribeImportFields() err = %v, want FailedPrecondition", err)
	}
}
