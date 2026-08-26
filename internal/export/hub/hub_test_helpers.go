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
	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/auth"
	"github.com/choysum-dev/choysum/pkg/auth/grpcclient/authpb"
	"github.com/choysum-dev/choysum/pkg/config"
	grpcclient "github.com/choysum-dev/choysum/pkg/grpc/client"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"gorm.io/gorm"
)

type stubIdentity struct{}

func (stubIdentity) GetUserID() string  { return "test-user" }
func (stubIdentity) GetTokenID() string { return "test-token" }
func (stubIdentity) GetMetadata() map[string]interface{} {
	return map[string]interface{}{"activeCompanyId": "cmp_test"}
}
func (stubIdentity) IsValid() bool { return true }

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
		return grpc.DialContext(ctx, "bufnet",
			grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() }),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
	}
	ctx := auth.ContextWithIdentity(context.Background(), stubIdentity{})
	return grpcclient.ContextWithServiceDialer(ctx, dialer)
}

func newHubTestScope(t *testing.T) scope.Scope {
	t.Helper()
	cfg := &config.Config{
		Db: &config.DbConfig{
			Dialect: "sqlite",
			DSN:     filepath.Join(t.TempDir(), "export-hub-test.db"),
		},
	}
	return defaultscope.NewDefaultScope(
		context.Background(),
		scopetest.FactoryInputFromConfig(cfg),
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
}

func seedPartnerModelMeta(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.AutoMigrate(&meta.Model{}, &meta.Field{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	companyField := "CompanyId"
	if err := db.Create(&meta.Model{
		Name: "Partner", Application: "partner", Path: "/tmp", ModelTable: "partner_partner", CompanyField: &companyField,
	}).Error; err != nil {
		t.Fatalf("seed partner model: %v", err)
	}
}

func seedPartnerModelFields(t *testing.T, db *gorm.DB) {
	t.Helper()
	seedPartnerModelMeta(t, db)
	partnerModel := &meta.Model{}
	if err := db.Where("application = ? AND name = ?", "partner", "Partner").First(partnerModel).Error; err != nil {
		t.Fatalf("load partner model: %v", err)
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
}

func seedCountryModelMeta(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.AutoMigrate(&meta.Model{}, &meta.Field{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	if err := db.Create(&meta.Model{Name: "Country", Application: "base", Path: "/tmp", ModelTable: "base_country"}).Error; err != nil {
		t.Fatalf("seed country model: %v", err)
	}
}

type denyAuthServer struct {
	authpb.UnimplementedUserServer
}

func (denyAuthServer) CheckMethodAccess(context.Context, *authpb.User_CheckMethodAccess_Req) (*authpb.User_CheckMethodAccess_Resp, error) {
	return &authpb.User_CheckMethodAccess_Resp{Result: false}, nil
}

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

type companyMetaIdentity struct{}

func (companyMetaIdentity) GetUserID() string  { return "u1" }
func (companyMetaIdentity) GetTokenID() string { return "tok" }
func (companyMetaIdentity) GetMetadata() map[string]interface{} {
	return map[string]interface{}{"companyId": "cmp-fallback"}
}
func (companyMetaIdentity) IsValid() bool { return true }

type spacedActiveCompanyIdentity struct{}

func (spacedActiveCompanyIdentity) GetUserID() string  { return "u1" }
func (spacedActiveCompanyIdentity) GetTokenID() string { return "tok" }
func (spacedActiveCompanyIdentity) GetMetadata() map[string]interface{} {
	return map[string]interface{}{"activeCompanyId": "  cmp_trim  "}
}
func (spacedActiveCompanyIdentity) IsValid() bool { return true }

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
