// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package hub

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/choysum-dev/choysum/internal/export/proto/exportpb"
	"github.com/choysum-dev/choysum/internal/export/registry"
	"github.com/choysum-dev/choysum/pkg/auth"
	exportpkg "github.com/choysum-dev/choysum/pkg/export"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestCheckModelExportAccessAuthRPCError(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	seedCountryModelMeta(t, runtimeScope.Session().DB)
	ctx := authCtxWithServer(t, errAuthServer{})
	err := checkModelExportAccess(ctx, runtimeScope, "base.Country", "")
	if err == nil || status.Code(err) != codes.PermissionDenied {
		t.Fatalf("err = %v", err)
	}
}

func TestCheckModelExportAccessDenied(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	seedCountryModelMeta(t, runtimeScope.Session().DB)
	ctx := authCtxWithServer(t, denyAuthServer{})
	err := checkModelExportAccess(ctx, runtimeScope, "base.Country", "")
	if err == nil || status.Code(err) != codes.PermissionDenied {
		t.Fatalf("err = %v", err)
	}
}

func TestCheckModelExportAccessPartnerRequiresCompany(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	seedPartnerModelMeta(t, runtimeScope.Session().DB)
	ctx := authCtx(t)
	err := checkModelExportAccess(ctx, runtimeScope, "partner.Partner", "")
	if err == nil || status.Code(err) != codes.InvalidArgument {
		t.Fatalf("err = %v", err)
	}
}

func TestCheckModelExportAccessMissingModelLookup(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	ctx := authCtx(t)
	err := checkModelExportAccess(ctx, runtimeScope, "missing.App", "cmp-1")
	if err == nil || status.Code(err) != codes.InvalidArgument {
		t.Fatalf("err = %v", err)
	}
}

func TestEnsureIdentityInvalid(t *testing.T) {
	if err := ensureIdentity(auth.ContextWithIdentity(context.Background(), invalidIdentity{})); err == nil {
		t.Fatal("expected invalid identity error")
	}
}

func TestActiveCompanyIDBranches(t *testing.T) {
	if got := activeCompanyID(context.Background()); got != "" {
		t.Fatalf("empty ctx = %q", got)
	}
	ctx := auth.ContextWithIdentity(context.Background(), nonStringMetaIdentity{})
	if got := activeCompanyID(ctx); got != "" {
		t.Fatalf("non-string metadata = %q", got)
	}
	ctx = auth.ContextWithIdentity(context.Background(), nilMetaIdentity{})
	if got := activeCompanyID(ctx); got != "" {
		t.Fatalf("nil metadata = %q", got)
	}
}

func TestModelCompanyFieldRequiredInvalidModel(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	ctx := runtimeScope.Context()
	if _, err := modelCompanyFieldRequired(ctx, runtimeScope, "invalid"); err == nil {
		t.Fatal("expected invalid model error")
	}
}

func TestModelCompanyFieldRequiredNilScope(t *testing.T) {
	required, err := modelCompanyFieldRequired(context.Background(), nil, "partner.Partner")
	if err != nil || required {
		t.Fatalf("required=%v err=%v", required, err)
	}
}

func TestSplitModelFullNameInvalid(t *testing.T) {
	if _, _, err := splitModelFullName("invalid"); err == nil {
		t.Fatal("expected invalid model")
	}
}

func TestRunExportValidateSpecFailure(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	seedCountryModelMeta(t, runtimeScope.Session().DB)
	ctx := authCtx(t)
	prev := validateExportSpec
	validateExportSpec = func(exportpkg.Spec) error { return errors.New("bad spec") }
	t.Cleanup(func() { validateExportSpec = prev })
	_, err := runExport(ctx, Deps{
		RuntimeScope: runtimeScope,
		Run: func(context.Context, scope.Scope, exportpkg.Spec) (importpkg.Report, error) {
			return importpkg.Report{}, nil
		},
	}, &exportpb.ExportRunRequest{Model: "base.Country"}, false)
	if err == nil || status.Code(err) != codes.InvalidArgument {
		t.Fatalf("err = %v", err)
	}
}

func TestRunExportReturnsReportWhenRunHasMessages(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	seedCountryModelMeta(t, runtimeScope.Session().DB)
	ctx := authCtx(t)
	resp, err := runExport(ctx, Deps{
		RuntimeScope: runtimeScope,
		Run: func(context.Context, scope.Scope, exportpkg.Spec) (importpkg.Report, error) {
			return importpkg.Report{
				Stats:    importpkg.Stats{Error: 1},
				Messages: []importpkg.Message{{Text: "row failed"}},
			}, errors.New("run failed")
		},
	}, &exportpb.ExportRunRequest{Model: "base.Country"}, false)
	if err != nil {
		t.Fatalf("expected report response, got err=%v", err)
	}
	if resp.GetReport().GetStats().GetError() != 1 {
		t.Fatalf("report = %#v", resp.GetReport())
	}
}

func TestRunExportInternalErrorWithoutMessages(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	seedCountryModelMeta(t, runtimeScope.Session().DB)
	ctx := authCtx(t)
	_, err := runExport(ctx, Deps{
		RuntimeScope: runtimeScope,
		Run: func(context.Context, scope.Scope, exportpkg.Spec) (importpkg.Report, error) {
			return importpkg.Report{}, errors.New("boom")
		},
	}, &exportpb.ExportRunRequest{Model: "base.Country"}, false)
	if err == nil || status.Code(err) != codes.Internal {
		t.Fatalf("err = %v", err)
	}
}

func TestRunExportNoExecutorAvailable(t *testing.T) {
	ctx := authCtx(t)
	_, err := runExport(ctx, Deps{RuntimeScope: newHubTestScope(t)}, &exportpb.ExportRunRequest{Model: "base.Country"}, false)
	if err == nil || status.Code(err) != codes.Unavailable {
		t.Fatalf("err = %v", err)
	}
}

func TestRunExportUsesActiveCompany(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	seedPartnerModelMeta(t, runtimeScope.Session().DB)
	ctx := authCtx(t)
	var captured exportpkg.Spec
	_, err := runExport(ctx, Deps{
		RuntimeScope: runtimeScope,
		Run: func(_ context.Context, _ scope.Scope, spec exportpkg.Spec) (importpkg.Report, error) {
			captured = spec
			return importpkg.Report{Stats: importpkg.Stats{Ok: 1}}, nil
		},
	}, &exportpb.ExportRunRequest{Model: "partner.Partner"}, false)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if captured.Options.CompanyID != "cmp_test" {
		t.Fatalf("company_id = %q", captured.Options.CompanyID)
	}
}

func TestRunExportToRecordSpecError(t *testing.T) {
	ctx := authCtx(t)
	_, err := runExport(ctx, Deps{
		RuntimeScope: newHubTestScope(t),
		Run: func(context.Context, scope.Scope, exportpkg.Spec) (importpkg.Report, error) {
			return importpkg.Report{}, nil
		},
	}, nil, false)
	if err == nil || status.Code(err) != codes.InvalidArgument {
		t.Fatalf("err = %v", err)
	}
}

func TestAttachInlineCSVOmitsOversizedPayload(t *testing.T) {
	resp := &exportpb.ExportRunResponse{}
	large := make([]byte, maxInlineCSVBytes+1)
	attachInlineCSV(resp, large)
	if len(resp.GetCsvData()) != 0 {
		t.Fatalf("csv_data len = %d, want 0", len(resp.GetCsvData()))
	}
	attachInlineCSV(resp, []byte("a,b\n1,2\n"))
	if string(resp.GetCsvData()) != "a,b\n1,2\n" {
		t.Fatalf("csv_data = %q", resp.GetCsvData())
	}
	attachInlineCSV(nil, []byte("x"))
}

func TestJsExecutorAdapter(t *testing.T) {
	adapter := jsExecutorAdapter{inner: stubJSExecutor{}}
	if err := adapter.Load(nil); err != nil {
		t.Fatal(err)
	}
	resp, err := adapter.Execute(context.Background(), &jsengine.JsRequest{})
	if err != nil || resp == nil {
		t.Fatalf("Execute = %v err=%v", resp, err)
	}
	if err := adapter.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestHubPreviewAndDescribeFieldsAuth(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	seedCountryModelMeta(t, runtimeScope.Session().DB)
	ctx := authCtxWithServer(t, denyAuthServer{})
	h := New(Deps{RuntimeScope: runtimeScope})
	if _, err := h.DescribeFields(ctx, &exportpb.DescribeFieldsRequest{Model: "base.Country"}); err == nil {
		t.Fatal("expected denied DescribeFields")
	}
	if _, err := h.Preview(ctx, &exportpb.ExportRunRequest{Model: "base.Country"}); err == nil {
		t.Fatal("expected denied Preview")
	}
}

func TestHubDescribeFieldsUnauthenticated(t *testing.T) {
	h := New(Deps{RuntimeScope: newHubTestScope(t)})
	if _, err := h.DescribeFields(context.Background(), &exportpb.DescribeFieldsRequest{Model: "base.Country"}); err == nil {
		t.Fatal("expected unauthenticated")
	}
}

func TestHubDescribeFieldsNilRequest(t *testing.T) {
	h := New(Deps{RuntimeScope: newHubTestScope(t)})
	ctx := authCtx(t)
	if _, err := h.DescribeFields(ctx, nil); err == nil || status.Code(err) != codes.InvalidArgument {
		t.Fatalf("err = %v", err)
	}
}

func TestHubDescribeFieldsUnavailableScope(t *testing.T) {
	h := New(Deps{})
	ctx := authCtx(t)
	if _, err := h.DescribeFields(ctx, &exportpb.DescribeFieldsRequest{Model: "base.Country"}); err == nil || status.Code(err) != codes.Unavailable {
		t.Fatalf("err = %v", err)
	}
}

func TestRunAsyncUnauthenticated(t *testing.T) {
	h := New(Deps{})
	if _, err := h.RunAsync(context.Background(), &exportpb.ExportRunAsyncRequest{}); err == nil {
		t.Fatal("expected unauthenticated")
	}
}

func TestModeFromProtoTemplate(t *testing.T) {
	mode, err := modeFromProto(exportpb.ExportMode_EXPORT_MODE_TEMPLATE)
	if err != nil || mode != exportpkg.ModeTemplate {
		t.Fatalf("mode = %q err=%v", mode, err)
	}
}

func TestModeFromProtoUnsupported(t *testing.T) {
	if _, err := modeFromProto(exportpb.ExportMode(99)); err == nil {
		t.Fatal("expected unsupported mode")
	}
}

func TestToRecordSpecTemplateSkipsPreviewLimit(t *testing.T) {
	spec, err := toRecordSpec(&exportpb.ExportRunRequest{
		Model: "base.Country",
		Mode:  exportpb.ExportMode_EXPORT_MODE_TEMPLATE,
	}, true)
	if err != nil {
		t.Fatal(err)
	}
	if spec.Limit != 0 {
		t.Fatalf("limit = %d", spec.Limit)
	}
}

func TestModelCompanyFieldRequiredEmptyCompanyField(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	db := runtimeScope.Session().DB
	if err := db.AutoMigrate(&meta.Model{}); err != nil {
		t.Fatal(err)
	}
	empty := "   "
	if err := db.Create(&meta.Model{
		Name: "NoCompany", Application: "test", Path: "/tmp", ModelTable: "test_nocompany", CompanyField: &empty,
	}).Error; err != nil {
		t.Fatal(err)
	}
	required, err := modelCompanyFieldRequired(runtimeScope.Context(), runtimeScope, "test.NoCompany")
	if err != nil || required {
		t.Fatalf("required=%v err=%v", required, err)
	}
}

type nilDBScope struct {
	ctx context.Context
}

func (s nilDBScope) Run(fn func(scope.Scope) error) error { return fn(s) }
func (s nilDBScope) Session() *scope.Session              { return &scope.Session{DB: nil} }
func (s nilDBScope) Transactor() scope.Transactor         { return nil }
func (s nilDBScope) WithContext(ctx context.Context) scope.Scope {
	next := s
	next.ctx = ctx
	return next
}
func (s nilDBScope) Context() context.Context {
	if s.ctx != nil {
		return s.ctx
	}
	return context.Background()
}
func (s nilDBScope) Logger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func TestModelCompanyFieldRequiredNilDBSession(t *testing.T) {
	ctx := context.Background()
	stubScope := nilDBScope{ctx: ctx}
	required, err := modelCompanyFieldRequired(ctx, stubScope, "partner.Partner")
	if err != nil || required {
		t.Fatalf("required=%v err=%v", required, err)
	}
}

func TestToRecordSpecEmptyModel(t *testing.T) {
	_, err := toRecordSpec(&exportpb.ExportRunRequest{}, false)
	if err == nil || status.Code(err) != codes.InvalidArgument {
		t.Fatalf("err = %v", err)
	}
}

func TestHubDescribeFieldsEmptyModel(t *testing.T) {
	h := New(Deps{RuntimeScope: newHubTestScope(t)})
	ctx := authCtx(t)
	_, err := h.DescribeFields(ctx, &exportpb.DescribeFieldsRequest{Model: "  "})
	if err == nil || status.Code(err) != codes.InvalidArgument {
		t.Fatalf("err = %v", err)
	}
}

func TestCheckModelExportAccessEmptyModel(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	ctx := authCtx(t)
	err := checkModelExportAccess(ctx, runtimeScope, "  ", "")
	if err == nil || status.Code(err) != codes.InvalidArgument {
		t.Fatalf("err = %v", err)
	}
}

func TestActiveCompanyIDCompanyIdFallback(t *testing.T) {
	ctx := auth.ContextWithIdentity(context.Background(), companyMetaIdentity{})
	if got := activeCompanyID(ctx); got != "cmp-fallback" {
		t.Fatalf("companyId fallback = %q", got)
	}
}

func TestRunExportWithExplicitCompanyID(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	seedPartnerModelMeta(t, runtimeScope.Session().DB)
	ctx := authCtx(t)
	var captured exportpkg.Spec
	_, err := runExport(ctx, Deps{
		RuntimeScope: runtimeScope,
		JSExecutor:   stubJSExecutor{},
		Run: func(_ context.Context, _ scope.Scope, spec exportpkg.Spec) (importpkg.Report, error) {
			captured = spec
			return importpkg.Report{Stats: importpkg.Stats{Ok: 1}}, nil
		},
	}, &exportpb.ExportRunRequest{Model: "partner.Partner", CompanyId: "cmp-explicit"}, false)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if captured.Options.CompanyID != "cmp-explicit" {
		t.Fatalf("company_id = %q", captured.Options.CompanyID)
	}
}

func TestRunExportWithJSExecutorAndInlineCSV(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	seedCountryModelMeta(t, runtimeScope.Session().DB)
	ctx := authCtx(t)
	_, err := runExport(ctx, Deps{
		RuntimeScope: runtimeScope,
		JSExecutor:   stubJSExecutor{},
		Run: func(context.Context, scope.Scope, exportpkg.Spec) (importpkg.Report, error) {
			return importpkg.Report{Stats: importpkg.Stats{Ok: 1}}, nil
		},
	}, &exportpb.ExportRunRequest{Model: "base.Country"}, false)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
}

func TestRunExportUnauthenticated(t *testing.T) {
	_, err := runExport(context.Background(), Deps{
		RuntimeScope: newHubTestScope(t),
		Run: func(context.Context, scope.Scope, exportpkg.Spec) (importpkg.Report, error) {
			return importpkg.Report{}, nil
		},
	}, &exportpb.ExportRunRequest{Model: "base.Country"}, false)
	if err == nil || status.Code(err) != codes.Unauthenticated {
		t.Fatalf("err = %v", err)
	}
}

func TestRunExportUnavailableRuntimeScope(t *testing.T) {
	ctx := authCtx(t)
	_, err := runExport(ctx, Deps{JSExecutor: stubJSExecutor{}}, &exportpb.ExportRunRequest{Model: "base.Country"}, false)
	if err == nil || status.Code(err) != codes.Unavailable {
		t.Fatalf("err = %v", err)
	}
}

func TestRunExportWithJSExecutorOnlyRunPath(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	seedCountryModelMeta(t, runtimeScope.Session().DB)
	ctx := authCtx(t)
	prev := runExportWithResultFn
	runExportWithResultFn = func(_ context.Context, _ scope.Scope, _ exportpkg.Spec) (importpkg.Report, registry.Result, error) {
		return importpkg.Report{Stats: importpkg.Stats{Ok: 2}}, registry.Result{CSVBytes: []byte("Name\nA\n")}, nil
	}
	t.Cleanup(func() { runExportWithResultFn = prev })
	resp, err := runExport(ctx, Deps{
		RuntimeScope: runtimeScope,
		JSExecutor:   stubJSExecutor{},
	}, &exportpb.ExportRunRequest{Model: "base.Country"}, false)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if string(resp.GetCsvData()) != "Name\nA\n" {
		t.Fatalf("csv_data = %q", resp.GetCsvData())
	}
}

func TestToRecordSpecWhitespaceModel(t *testing.T) {
	_, err := toRecordSpec(&exportpb.ExportRunRequest{Model: "   "}, false)
	if err == nil || status.Code(err) != codes.InvalidArgument {
		t.Fatalf("err = %v", err)
	}
}

func TestToRecordSpecUnspecifiedMode(t *testing.T) {
	spec, err := toRecordSpec(&exportpb.ExportRunRequest{
		Model: "base.Country",
		Mode:  exportpb.ExportMode_EXPORT_MODE_UNSPECIFIED,
	}, false)
	if err != nil || spec.Mode != exportpkg.ModeData {
		t.Fatalf("spec = %#v err=%v", spec, err)
	}
}
