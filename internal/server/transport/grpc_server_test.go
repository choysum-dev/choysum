// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package transport

import (
	"context"
	"testing"

	"google.golang.org/grpc"
)

func TestNewGRPCServerWithInterceptors(t *testing.T) {
	t.Parallel()
	unary := grpc.UnaryServerInterceptor(func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		return handler(ctx, req)
	})
	stream := grpc.StreamServerInterceptor(func(srv any, ss grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		return handler(srv, ss)
	})
	srv := NewGRPCServer(GRPCServerOptions{
		MaxRecvMsgSize:     1024,
		MaxSendMsgSize:     2048,
		UnaryInterceptors:  []grpc.UnaryServerInterceptor{unary},
		StreamInterceptors: []grpc.StreamServerInterceptor{stream},
	})
	if srv == nil {
		t.Fatal("expected non-nil server")
	}
	srv.Stop()
}

func TestNewGRPCServerWithoutInterceptors(t *testing.T) {
	t.Parallel()
	srv := NewGRPCServer(GRPCServerOptions{})
	if srv == nil {
		t.Fatal("expected non-nil server")
	}
	srv.Stop()
}
