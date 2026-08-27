// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package hub

import (
	"context"
	"testing"

	"github.com/choysum-dev/choysum/internal/export/proto/exportpb"
	recordreader "github.com/choysum-dev/choysum/internal/export/reader/record"
	"github.com/choysum-dev/choysum/pkg/auth"
	exportpkg "github.com/choysum-dev/choysum/pkg/export"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestDescribeFields(t *testing.T) {
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
	resp, err := h.DescribeFields(ctx, &exportpb.DescribeFieldsRequest{Model: "partner.Partner"})
	if err != nil {
		t.Fatalf("DescribeFields: %v", err)
	}
	if len(resp.GetFields()) < 2 {
		t.Fatalf("fields = %#v", resp.GetFields())
	}
	if len(resp.GetDefaultFields()) == 0 {
		t.Fatalf("default_fields empty")
	}
}

func TestDescribeFieldsRequiresModel(t *testing.T) {
	h := New(Deps{RuntimeScope: newHubTestScope(t)})
	ctx := authCtx(t)
	_, err := h.DescribeFields(ctx, &exportpb.DescribeFieldsRequest{})
	if err == nil || status.Code(err) != codes.InvalidArgument {
		t.Fatalf("DescribeFields() err = %v", err)
	}
}

func TestRun_Partner_WithIds(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	seedPartnerModelMeta(t, runtimeScope.Session().DB)
	ctx := authCtx(t)

	var captured exportpkg.Spec
	h := New(Deps{
		RuntimeScope: runtimeScope,
		Run: func(_ context.Context, _ scope.Scope, spec exportpkg.Spec) (importpkg.Report, error) {
			captured = spec
			return importpkg.Report{Stats: importpkg.Stats{Ok: 1}}, nil
		},
	})

	_, err := h.Run(ctx, &exportpb.ExportRunRequest{
		Model:     "partner.Partner",
		CompanyId: "cmp_test",
		Ids:       []string{"p1", "p2"},
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(captured.Ids) != 2 || captured.Ids[0] != "p1" {
		t.Fatalf("ids = %#v", captured.Ids)
	}
	if captured.Domain != "" {
		t.Fatalf("domain = %q, want empty when ids set", captured.Domain)
	}
	if captured.Model != "partner.Partner" {
		t.Fatalf("model = %q", captured.Model)
	}
}

func TestRun_DomainWhenNoIds(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	seedPartnerModelMeta(t, runtimeScope.Session().DB)
	ctx := authCtx(t)

	var captured exportpkg.Spec
	h := New(Deps{
		RuntimeScope: runtimeScope,
		Run: func(_ context.Context, _ scope.Scope, spec exportpkg.Spec) (importpkg.Report, error) {
			captured = spec
			return importpkg.Report{Stats: importpkg.Stats{Ok: 3}}, nil
		},
	})

	domain := `{"And":[["IsActive","=",true]]}`
	_, err := h.Run(ctx, &exportpb.ExportRunRequest{
		Model:     "partner.Partner",
		CompanyId: "cmp_test",
		Domain:    domain,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(captured.Ids) != 0 {
		t.Fatalf("ids = %#v, want none", captured.Ids)
	}
	if captured.Domain != domain {
		t.Fatalf("domain = %q", captured.Domain)
	}
}

func TestToRecordSpecPreviewLimit(t *testing.T) {
	spec, err := toRecordSpec(&exportpb.ExportRunRequest{Model: "partner.Partner"}, true)
	if err != nil {
		t.Fatalf("toRecordSpec: %v", err)
	}
	if spec.Limit != previewRowLimit {
		t.Fatalf("limit = %d, want %d", spec.Limit, previewRowLimit)
	}
}

func TestDefaultExportFieldsPartner(t *testing.T) {
	fields, err := recordreader.DefaultExportFields("partner.Partner")
	if err != nil {
		t.Fatalf("DefaultExportFields: %v", err)
	}
	if len(fields) < 5 {
		t.Fatalf("fields = %#v", fields)
	}
}

func TestRunAsyncRequiresRunRequest(t *testing.T) {
	h := New(Deps{RuntimeScope: newHubTestScope(t), JSExecutor: stubJSExecutor{}})
	ctx := auth.ContextWithIdentity(context.Background(), stubIdentity{})
	_, err := h.RunAsync(ctx, &exportpb.ExportRunAsyncRequest{})
	if err == nil || status.Code(err) != codes.InvalidArgument {
		t.Fatalf("RunAsync err = %v", err)
	}
}
