// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package hub

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/choysum-dev/choysum/internal/import/proto/importpb"
	"github.com/choysum-dev/choysum/pkg/auth"
	"github.com/choysum-dev/choysum/pkg/auth/grpcclient/authpb"
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
)

type errAuthServer struct {
	authpb.UnimplementedUserServer
}

func (errAuthServer) CheckMethodAccess(context.Context, *authpb.User_CheckMethodAccess_Req) (*authpb.User_CheckMethodAccess_Resp, error) {
	return nil, errors.New("auth rpc down")
}

type invalidIdentity struct{}

func (invalidIdentity) GetUserID() string  { return "" }
func (invalidIdentity) GetTokenID() string { return "" }
func (invalidIdentity) GetMetadata() map[string]interface{} {
	return nil
}
func (invalidIdentity) IsValid() bool { return false }

type nonStringMetaIdentity struct{}

func (nonStringMetaIdentity) GetUserID() string  { return "u1" }
func (nonStringMetaIdentity) GetTokenID() string { return "tok" }
func (nonStringMetaIdentity) GetMetadata() map[string]interface{} {
	return map[string]interface{}{"activeCompanyId": 123, "companyId": true}
}
func (nonStringMetaIdentity) IsValid() bool { return true }

type nilMetaIdentity struct{}

func (nilMetaIdentity) GetUserID() string  { return "u1" }
func (nilMetaIdentity) GetTokenID() string { return "tok" }
func (nilMetaIdentity) GetMetadata() map[string]interface{} {
	return nil
}
func (nilMetaIdentity) IsValid() bool { return true }

func authCtxWithServer(t *testing.T, server authpb.UserServer) context.Context {
	t.Helper()
	lis := bufconn.Listen(1024 * 1024)
	grpcServer := grpc.NewServer()
	authpb.RegisterUserServer(grpcServer, server)
	go func() { _ = grpcServer.Serve(lis) }()
	t.Cleanup(grpcServer.Stop)
	dialer := func(ctx context.Context, serviceName string) (*grpc.ClientConn, error) {
		return grpc.DialContext(ctx, "bufnet",
			grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() }),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
	}
	ctx := auth.ContextWithIdentity(context.Background(), stubIdentity{})
	return grpcclient.ContextWithServiceDialer(ctx, dialer)
}

func TestCheckModelImportAccessAuthRPCError(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	ctx := authCtxWithServer(t, errAuthServer{})
	err := checkModelImportAccess(ctx, runtimeScope, "base.Country", "")
	if err == nil {
		t.Fatal("expected auth rpc error")
	}
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("code = %v", status.Code(err))
	}
}

func TestEnsureIdentityInvalid(t *testing.T) {
	if err := ensureIdentity(auth.ContextWithIdentity(context.Background(), invalidIdentity{})); err == nil {
		t.Fatal("expected invalid identity error")
	}
}

func TestActiveCompanyIDNonStringMetadata(t *testing.T) {
	ctx := auth.ContextWithIdentity(context.Background(), nonStringMetaIdentity{})
	if got := activeCompanyID(ctx); got != "" {
		t.Fatalf("non-string metadata = %q", got)
	}
	ctx = auth.ContextWithIdentity(context.Background(), nilMetaIdentity{})
	if got := activeCompanyID(ctx); got != "" {
		t.Fatalf("nil metadata = %q", got)
	}
}

func TestCompanyFieldRequiredInvalidModelAndMissingMeta(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	ctx := runtimeScope.Context()
	if companyFieldRequired(ctx, runtimeScope, "invalid") {
		t.Fatal("invalid model name")
	}
	if companyFieldRequired(ctx, runtimeScope, "missing.App") {
		t.Fatal("missing model row")
	}
}

func TestRunImportJSExecutorPath(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	seedCountryModelMeta(t, runtimeScope.Session().DB)
	ctx := authCtx(t)
	h := New(Deps{
		RuntimeScope: runtimeScope,
		SourceReader: StubSourceReader{"src-js": []byte("Name,Code,IsActive,ZipRequired,StateRequired\nX,1,true,true,false\n")},
		JSExecutor:   stubJSExecutor{},
		Run:          nil,
	})
	resp, err := h.Run(ctx, &importpb.ImportRunRequest{TargetModel: "base.Country", SourceRef: "src-js"})
	if err != nil {
		t.Fatalf("Run with JSExecutor: %v", err)
	}
	if resp.GetReport().GetStats().GetOk() == 0 && resp.GetReport().GetStats().GetError() == 0 {
		t.Fatalf("unexpected empty report: %#v", resp.GetReport())
	}
}

func TestRunImportValidateSpecFailure(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	seedCountryModelMeta(t, runtimeScope.Session().DB)
	ctx := authCtx(t)
	_, err := runImport(ctx, Deps{
		RuntimeScope: runtimeScope,
		Run: func(context.Context, scope.Scope, importpkg.Spec) (importpkg.Report, error) {
			return importpkg.Report{}, nil
		},
	}, &importpb.ImportRunRequest{
		TargetModel: "base.Country",
		SourceRef:   "src-1",
		Policy:      importpb.ImportPolicy_IMPORT_POLICY_BEST_EFFORT,
	}, false)
	if err == nil || status.Code(err) != codes.InvalidArgument {
		t.Fatalf("ValidateSpec err = %v", err)
	}
}

func TestRunImportReturnsReportWhenRunHasMessages(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	seedCountryModelMeta(t, runtimeScope.Session().DB)
	ctx := authCtx(t)
	resp, err := runImport(ctx, Deps{
		RuntimeScope: runtimeScope,
		SourceReader: StubSourceReader{"src-msg": []byte("Name,Code,IsActive,ZipRequired,StateRequired\nX,1,true,true,false\n")},
		Run: func(context.Context, scope.Scope, importpkg.Spec) (importpkg.Report, error) {
			return importpkg.Report{
				Stats:    importpkg.Stats{Error: 1},
				Messages: []importpkg.Message{{Text: "row failed"}},
			}, errors.New("run failed")
		},
	}, &importpb.ImportRunRequest{TargetModel: "base.Country", SourceRef: "src-msg"}, false)
	if err != nil {
		t.Fatalf("expected report response, got err=%v", err)
	}
	if resp.GetReport().GetStats().GetError() != 1 {
		t.Fatalf("report = %#v", resp.GetReport())
	}
}

func TestRunImportNoExecutorAvailable(t *testing.T) {
	ctx := authCtx(t)
	_, err := runImport(ctx, Deps{RuntimeScope: newHubTestScope(t)}, &importpb.ImportRunRequest{
		TargetModel: "base.Country",
		SourceRef:   "src-1",
	}, false)
	if err == nil || status.Code(err) != codes.Unavailable {
		t.Fatalf("err = %v", err)
	}
}

func TestRunImportInternalErrorWithoutMessages(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	seedCountryModelMeta(t, runtimeScope.Session().DB)
	ctx := authCtx(t)
	_, err := runImport(ctx, Deps{
		RuntimeScope: runtimeScope,
		SourceReader: StubSourceReader{"src-boom": []byte("Name,Code,IsActive,ZipRequired,StateRequired\nX,1,true,true,false\n")},
		Run: func(context.Context, scope.Scope, importpkg.Spec) (importpkg.Report, error) {
			return importpkg.Report{}, errors.New("boom")
		},
	}, &importpb.ImportRunRequest{TargetModel: "base.Country", SourceRef: "src-boom"}, false)
	if err == nil || status.Code(err) != codes.Internal {
		t.Fatalf("err = %v", err)
	}
}

func TestCompanyFieldRequiredNoSession(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	ctx := context.Background()
	if companyFieldRequired(ctx, runtimeScope, "partner.Partner") {
		t.Fatal("expected false without session in ctx")
	}
}

func TestCheckModelImportAccessPartnerOK(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	seedPartnerModelMeta(t, runtimeScope.Session().DB)
	ctx := authCtx(t)
	if err := checkModelImportAccess(ctx, runtimeScope, "partner.Partner", "cmp-1"); err != nil {
		t.Fatalf("partner access: %v", err)
	}
}

func TestHubParseHeadersDefaultReader(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	h := New(Deps{
		RuntimeScope: runtimeScope,
		SourceReader: nil,
	})
	ctx := auth.ContextWithIdentity(context.Background(), stubIdentity{})
	_, err := h.ParseHeaders(ctx, &importpb.ParseHeadersRequest{SourceRef: "missing"})
	if err == nil {
		t.Fatal("expected read error via default reader")
	}
}

func TestReadErrorType(t *testing.T) {
	err := csvReadError("boom")
	if err.Error() != "boom" {
		t.Fatalf("error text = %q", err.Error())
	}
}

func TestCompanyFieldRequiredEmptyCompanyField(t *testing.T) {
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
	if companyFieldRequired(runtimeScope.Context(), runtimeScope, "test.NoCompany") {
		t.Fatal("whitespace company field should not require company")
	}
}

func TestJsExecutorAdapterExecuteError(t *testing.T) {
	adapter := jsExecutorAdapter{inner: stubJSExecutorErr{}}
	if _, err := adapter.Execute(context.Background(), &jsengine.JsRequest{}); err == nil {
		t.Fatal("expected execute error")
	}
}

type stubJSExecutorErr struct{}

func (stubJSExecutorErr) AppendJsScripts(...*jsengine.JsScript) {}
func (stubJSExecutorErr) Start() error                          { return nil }
func (stubJSExecutorErr) Stop() error                           { return nil }
func (stubJSExecutorErr) Execute(context.Context, *jsengine.JsRequest) (*jsengine.JsResponse, error) {
	return nil, errors.New("execute failed")
}
func (stubJSExecutorErr) GetJsScripts() []*jsengine.JsScript { return nil }
func (stubJSExecutorErr) SetJsScripts([]*jsengine.JsScript)    {}
func (stubJSExecutorErr) Reload(...*jsengine.JsScript) error   { return nil }
