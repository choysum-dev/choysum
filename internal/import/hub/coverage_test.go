// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package hub

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"path/filepath"
	"testing"

	"github.com/choysum-dev/choysum/internal/defaultscope"
	"github.com/choysum-dev/choysum/internal/import/proto/importpb"
	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/auth"
	"github.com/choysum-dev/choysum/pkg/auth/grpcclient/authpb"
	"github.com/choysum-dev/choysum/pkg/config"
	grpcclient "github.com/choysum-dev/choysum/pkg/grpc/client"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"gorm.io/gorm"
)

type allowAuthServer struct {
	authpb.UnimplementedUserServer
}

func (allowAuthServer) CheckMethodAccess(_ context.Context, _ *authpb.User_CheckMethodAccess_Req) (*authpb.User_CheckMethodAccess_Resp, error) {
	return &authpb.User_CheckMethodAccess_Resp{Result: true}, nil
}

func authCtx(t *testing.T) context.Context {
	t.Helper()
	lis := bufconn.Listen(1024 * 1024)
	grpcServer := grpc.NewServer()
	authpb.RegisterUserServer(grpcServer, allowAuthServer{})
	go func() { _ = grpcServer.Serve(lis) }()
	t.Cleanup(grpcServer.Stop)

	dialer := func(ctx context.Context, serviceName string) (*grpc.ClientConn, error) {
		if serviceName != "auth.User" {
			return nil, errors.New("unexpected service")
		}
		return grpc.DialContext(ctx, "bufnet",
			grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() }),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
	}
	ctx := auth.ContextWithIdentity(context.Background(), stubIdentity{})
	return grpcclient.ContextWithServiceDialer(ctx, dialer)
}

func TestEnsureIdentityAndActiveCompany(t *testing.T) {
	if err := ensureIdentity(context.Background()); err == nil {
		t.Fatal("expected unauthenticated")
	}
	if err := ensureIdentity(auth.ContextWithIdentity(context.Background(), stubIdentity{})); err != nil {
		t.Fatalf("ensureIdentity: %v", err)
	}
	ctx := auth.ContextWithIdentity(context.Background(), stubIdentity{})
	if got := activeCompanyID(ctx); got != "cmp_test" {
		t.Fatalf("activeCompanyID = %q", got)
	}
}

func TestSplitModelFullName(t *testing.T) {
	app, name, err := splitModelFullName("base.Country")
	if err != nil || app != "base" || name != "Country" {
		t.Fatalf("split = %q/%q err=%v", app, name, err)
	}
	if _, _, err := splitModelFullName("invalid"); err == nil {
		t.Fatal("expected invalid model error")
	}
}

func TestCompanyFieldRequired(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	ctx := runtimeScope.Context()
	if companyFieldRequired(ctx, runtimeScope, "base.Country") {
		t.Fatal("country should not require company")
	}
	seedPartnerModelMeta(t, runtimeScope.Session().DB)
	if !companyFieldRequired(ctx, runtimeScope, "partner.Partner") {
		t.Fatal("partner should require company")
	}
	if companyFieldRequired(ctx, nil, "partner.Partner") {
		t.Fatal("nil scope")
	}
}

func TestCheckModelImportAccess(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	ctx := authCtx(t)
	if err := checkModelImportAccess(ctx, runtimeScope, "base.Country", ""); err != nil {
		t.Fatalf("country access: %v", err)
	}
	seedPartnerModelMeta(t, runtimeScope.Session().DB)
	if err := checkModelImportAccess(ctx, runtimeScope, "partner.Partner", ""); err == nil {
		t.Fatal("expected missing company_id")
	}
	if err := checkModelImportAccess(ctx, runtimeScope, "", "cmp"); err == nil {
		t.Fatal("expected missing target model")
	}
}

func TestCheckModelImportAccessDenied(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	lis := bufconn.Listen(1024 * 1024)
	grpcServer := grpc.NewServer()
	authpb.RegisterUserServer(grpcServer, denyAuthServer{})
	go func() { _ = grpcServer.Serve(lis) }()
	t.Cleanup(grpcServer.Stop)
	dialer := func(ctx context.Context, serviceName string) (*grpc.ClientConn, error) {
		return grpc.DialContext(ctx, "bufnet",
			grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() }),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
	}
	ctx := auth.ContextWithIdentity(context.Background(), stubIdentity{})
	ctx = grpcclient.ContextWithServiceDialer(ctx, dialer)
	if err := checkModelImportAccess(ctx, runtimeScope, "base.Country", ""); err == nil {
		t.Fatal("expected denied")
	}
}

type denyAuthServer struct {
	authpb.UnimplementedUserServer
}

func (denyAuthServer) CheckMethodAccess(context.Context, *authpb.User_CheckMethodAccess_Req) (*authpb.User_CheckMethodAccess_Resp, error) {
	return &authpb.User_CheckMethodAccess_Resp{Result: false}, nil
}

func TestActiveCompanyIDBranches(t *testing.T) {
	if got := activeCompanyID(context.Background()); got != "" {
		t.Fatalf("empty ctx = %q", got)
	}
	ctx := auth.ContextWithIdentity(context.Background(), companyMetaIdentity{key: "companyId", value: "cmp-co"})
	if got := activeCompanyID(ctx); got != "cmp-co" {
		t.Fatalf("companyId metadata = %q", got)
	}
}

type companyMetaIdentity struct {
	stubIdentity
	key   string
	value any
}

func (c companyMetaIdentity) GetMetadata() map[string]interface{} {
	return map[string]interface{}{c.key: c.value}
}

func TestRunImportUsesActiveCompanyAndDefaultReader(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	seedCountryModelMeta(t, runtimeScope.Session().DB)
	ctx := authCtx(t)
	h := New(Deps{
		RuntimeScope: runtimeScope,
		SourceReader: StubSourceReader{"src-run": []byte("Name,Code,IsActive,ZipRequired,StateRequired\nX,1,true,true,false\n")},
		Run: func(_ context.Context, _ scope.Scope, spec importpkg.Spec) (importpkg.Report, error) {
			if spec.Options.CompanyID != "cmp_test" {
				t.Fatalf("company_id = %q", spec.Options.CompanyID)
			}
			return importpkg.Report{Stats: importpkg.Stats{Ok: 1}}, nil
		},
	})
	resp, err := h.Run(ctx, &importpb.ImportRunRequest{TargetModel: "base.Country", SourceRef: "src-run"})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if resp.GetReport().GetStats().GetOk() != 1 {
		t.Fatalf("report = %#v", resp.GetReport())
	}
}

func TestHubPreviewAndRun(t *testing.T) {
	csvBytes := []byte("Name,Code,IsActive,ZipRequired,StateRequired\nHub,HB001,true,true,false\n")
	runtimeScope := newHubTestScope(t)
	seedCountryModelMeta(t, runtimeScope.Session().DB)
	ctx := authCtx(t)

	h := New(Deps{
		RuntimeScope: runtimeScope,
		SourceReader: StubSourceReader{"src-hub": csvBytes},
		Run: func(_ context.Context, _ scope.Scope, spec importpkg.Spec) (importpkg.Report, error) {
			if !spec.DryRun {
				return importpkg.Report{Stats: importpkg.Stats{Total: 1, Ok: 1}}, nil
			}
			return importpkg.Report{Stats: importpkg.Stats{Total: 1, Ok: 1}, DryRun: true}, nil
		},
	})

	req := &importpb.ImportRunRequest{
		TargetModel: "base.Country",
		SourceRef:   "src-hub",
		CompanyId:   "",
	}
	preview, err := h.Preview(ctx, req)
	if err != nil {
		t.Fatalf("Preview: %v", err)
	}
	if preview.GetReport().GetDryRun() != true || preview.GetReport().GetStats().GetOk() != 1 {
		t.Fatalf("preview report = %#v", preview.GetReport())
	}

	runResp, err := h.Run(ctx, req)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if runResp.GetReport().GetDryRun() || runResp.GetReport().GetStats().GetOk() != 1 {
		t.Fatalf("run report = %#v", runResp.GetReport())
	}
}

func TestHubParseHeadersErrors(t *testing.T) {
	h := New(Deps{SourceReader: StubSourceReader{}})
	if _, err := h.ParseHeaders(context.Background(), &importpb.ParseHeadersRequest{}); err == nil {
		t.Fatal("expected unauthenticated")
	}
	ctx := auth.ContextWithIdentity(context.Background(), stubIdentity{})
	if _, err := h.ParseHeaders(ctx, &importpb.ParseHeadersRequest{}); err == nil {
		t.Fatal("expected missing source_ref")
	}
	if _, err := h.ParseHeaders(ctx, &importpb.ParseHeadersRequest{SourceRef: "missing"}); err == nil {
		t.Fatal("expected read error")
	}
	if _, err := h.ParseHeaders(ctx, &importpb.ParseHeadersRequest{SourceRef: "bad"}); err == nil {
		t.Fatal("expected parse error")
	}
	hBad := New(Deps{SourceReader: StubSourceReader{"bad": []byte("\"\n")}})
	if _, err := hBad.ParseHeaders(ctx, &importpb.ParseHeadersRequest{SourceRef: "bad"}); err == nil {
		t.Fatal("expected csv parse error")
	}
}

func TestRunAsyncRequiresExecutor(t *testing.T) {
	h := New(Deps{})
	_, err := h.RunAsync(context.Background(), &importpb.ImportRunAsyncRequest{})
	if err == nil || status.Code(err) != codes.Unauthenticated {
		t.Fatalf("RunAsync err = %v", err)
	}
}

func TestRunImportErrors(t *testing.T) {
	_, err := runImport(context.Background(), Deps{}, nil, false)
	if err == nil || status.Code(err) != codes.Unauthenticated {
		t.Fatalf("nil identity err = %v", err)
	}
	ctx := authCtx(t)
	_, err = runImport(ctx, Deps{}, &importpb.ImportRunRequest{}, false)
	if err == nil {
		t.Fatal("expected unavailable without scope")
	}
}

func TestStubSourceReaderAndDocumentSourceReader(t *testing.T) {
	stub := StubSourceReader{"ok": []byte("data")}
	if _, err := stub.Read(context.Background(), "missing"); err == nil {
		t.Fatal("expected missing ref")
	}
	raw, err := stub.Read(context.Background(), "ok")
	if err != nil || string(raw) != "data" {
		t.Fatalf("stub read = %q err=%v", raw, err)
	}

	var doc DocumentSourceReader
	if _, err := doc.Read(context.Background(), "x"); err == nil {
		t.Fatal("expected nil scope error")
	}
	doc = DocumentSourceReader{RuntimeScope: newHubTestScope(t)}
	if _, err := doc.Read(context.Background(), "x"); err == nil {
		t.Fatal("expected auth error")
	}

	ctx := auth.ContextWithIdentity(context.Background(), stubIdentity{})
	runCtx := ContextWithSourceReader(ctx, stub)
	got, err := stub.Read(runCtx, "ok")
	if err != nil || string(got) != "data" {
		t.Fatalf("loader = %q err=%v", got, err)
	}
	if ContextWithSourceReader(ctx, nil) != ctx {
		t.Fatal("nil reader unchanged")
	}
}

func TestPolicyFromProtoBranches(t *testing.T) {
	if _, err := toRecordSpec(nil, false); err == nil {
		t.Fatal("nil request")
	}
	if _, err := toRecordSpec(&importpb.ImportRunRequest{TargetModel: "x"}, false); err == nil {
		t.Fatal("missing source_ref")
	}
	if _, err := policyFromProto(importpb.ImportPolicy(99)); err == nil {
		t.Fatal("unsupported policy")
	}
	if _, err := policyFromProto(importpb.ImportPolicy_IMPORT_POLICY_STOP_KEEP); err == nil {
		t.Fatal("stop_keep rejected")
	}
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

type stubJSExecutor struct{}

func (stubJSExecutor) AppendJsScripts(...*jsengine.JsScript) {}
func (stubJSExecutor) Start() error                          { return nil }
func (stubJSExecutor) Stop() error                           { return nil }
func (stubJSExecutor) Execute(context.Context, *jsengine.JsRequest) (*jsengine.JsResponse, error) {
	return &jsengine.JsResponse{}, nil
}
func (stubJSExecutor) GetJsScripts() []*jsengine.JsScript { return nil }
func (stubJSExecutor) SetJsScripts([]*jsengine.JsScript)  {}
func (stubJSExecutor) Reload(...*jsengine.JsScript) error { return nil }

func newHubTestScope(t *testing.T) scope.Scope {
	t.Helper()
	cfg := &config.Config{
		Db: &config.DbConfig{
			Dialect: "sqlite",
			DSN:     filepath.Join(t.TempDir(), "hub-test.db"),
		},
	}
	return defaultscope.NewDefaultScope(
		context.Background(),
		scopetest.FactoryInputFromConfig(cfg),
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
}

func seedCountryModelMeta(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.AutoMigrate(&meta.Model{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	if err := db.Create(&meta.Model{Name: "Country", Application: "base", Path: "/tmp", ModelTable: "base_country"}).Error; err != nil {
		t.Fatalf("seed country model: %v", err)
	}
}

func seedPartnerModelMeta(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.AutoMigrate(&meta.Model{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	companyField := "CompanyId"
	if err := db.Create(&meta.Model{
		Name: "Partner", Application: "partner", Path: "/tmp", ModelTable: "partner_partner", CompanyField: &companyField,
	}).Error; err != nil {
		t.Fatalf("seed partner model: %v", err)
	}
}
