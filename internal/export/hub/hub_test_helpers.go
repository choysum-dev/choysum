// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package hub

import (
	"context"
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
