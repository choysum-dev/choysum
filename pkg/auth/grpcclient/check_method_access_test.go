// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package grpcclient

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/choysum-dev/choysum/pkg/auth/grpcclient/authpb"
	grpcclient "github.com/choysum-dev/choysum/pkg/grpc/client"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

func TestCloneMetadata(t *testing.T) {
	t.Run("nil becomes empty metadata", func(t *testing.T) {
		got := cloneMetadata(nil)
		if got == nil {
			t.Fatal("expected empty metadata map")
		}
		if len(got) != 0 {
			t.Fatalf("expected empty metadata, got %#v", got)
		}
	})

	t.Run("copies slices", func(t *testing.T) {
		original := metadata.MD{
			"x-trace": {"trace-1", "trace-2"},
		}

		cloned := cloneMetadata(original)
		original["x-trace"][0] = "mutated"

		if cloned["x-trace"][0] != "trace-1" {
			t.Fatalf("expected cloned metadata to keep original value, got %#v", cloned)
		}
		cloned["x-trace"][1] = "changed-copy"
		if original["x-trace"][1] != "trace-2" {
			t.Fatalf("expected original metadata to stay unchanged, got %#v", original)
		}
	})
}

func TestCheckMethodAccess(t *testing.T) {
	t.Run("returns grpc result and forwards incoming metadata", func(t *testing.T) {
		server := &recordingUserServer{
			response: &authpb.User_CheckMethodAccess_Resp{Result: true},
		}
		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
			"x-choysum-trace", "trace-1",
			"x-choysum-trace", "trace-2",
			"x-company", "c-9",
		))
		ctx = grpcclient.ContextWithServiceDialer(ctx, newUserServiceDialer(t, server))

		allowed, err := CheckMethodAccess(ctx, "company-1", "/sales.Order/Read")
		if err != nil {
			t.Fatalf("CheckMethodAccess returned error: %v", err)
		}
		if !allowed {
			t.Fatal("expected access to be allowed")
		}
		if server.lastReq == nil {
			t.Fatal("expected server to receive a request")
		}
		if server.lastReq.GetCompanyId() != "company-1" || server.lastReq.GetServiceFullName() != "/sales.Order/Read" {
			t.Fatalf("unexpected request: %#v", server.lastReq)
		}
		if got := server.lastMD.Get("x-choysum-trace"); len(got) != 2 || got[0] != "trace-1" || got[1] != "trace-2" {
			t.Fatalf("unexpected forwarded trace metadata: %#v", server.lastMD)
		}
		if got := server.lastMD.Get("x-company"); len(got) != 1 || got[0] != "c-9" {
			t.Fatalf("unexpected forwarded company metadata: %#v", server.lastMD)
		}
	})

	t.Run("returns dial errors", func(t *testing.T) {
		_, err := CheckMethodAccess(context.Background(), "company-1", "/sales.Order/Read")
		if err == nil {
			t.Fatal("expected dial error")
		}
		var missingDialer *grpcclient.MissingServiceDialerError
		if !errors.As(err, &missingDialer) {
			t.Fatalf("expected MissingServiceDialerError, got %T (%v)", err, err)
		}
	})

	t.Run("returns rpc errors", func(t *testing.T) {
		server := &recordingUserServer{
			err: status.Error(codes.PermissionDenied, "denied"),
		}
		ctx := grpcclient.ContextWithServiceDialer(context.Background(), newUserServiceDialer(t, server))

		allowed, err := CheckMethodAccess(ctx, "company-1", "/sales.Order/Delete")
		if err == nil {
			t.Fatal("expected rpc error")
		}
		if allowed {
			t.Fatal("expected fail-closed false result on rpc error")
		}
		if status.Code(err) != codes.PermissionDenied {
			t.Fatalf("expected permission denied, got %v", err)
		}
	})
}

type recordingUserServer struct {
	authpb.UnimplementedUserServer

	response *authpb.User_CheckMethodAccess_Resp
	err      error
	lastReq  *authpb.User_CheckMethodAccess_Req
	lastMD   metadata.MD
}

func (s *recordingUserServer) CheckMethodAccess(ctx context.Context, req *authpb.User_CheckMethodAccess_Req) (*authpb.User_CheckMethodAccess_Resp, error) {
	s.lastReq = req
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		s.lastMD = md
	} else {
		s.lastMD = metadata.MD{}
	}
	if s.err != nil {
		return nil, s.err
	}
	return s.response, nil
}

func newUserServiceDialer(t *testing.T, server authpb.UserServer) grpcclient.ServiceDialer {
	t.Helper()

	lis := bufconn.Listen(1024 * 1024)
	grpcServer := grpc.NewServer()
	authpb.RegisterUserServer(grpcServer, server)
	go func() { _ = grpcServer.Serve(lis) }()
	t.Cleanup(grpcServer.Stop)

	var conn *grpc.ClientConn
	t.Cleanup(func() {
		if conn != nil {
			_ = conn.Close()
		}
	})

	return func(ctx context.Context, serviceName string) (*grpc.ClientConn, error) {
		if serviceName != "auth.User" {
			return nil, errors.New("unexpected service")
		}
		if conn != nil {
			return conn, nil
		}
		c, err := grpc.DialContext(ctx, "bufnet",
			grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() }),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		if err != nil {
			return nil, err
		}
		conn = c
		return conn, nil
	}
}
