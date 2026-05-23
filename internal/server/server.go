// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package server

import (
	"context"
	"net"
	"net/http"
	"sync/atomic"

	"github.com/choysum-dev/choysum/internal/server/transport"
	"github.com/choysum-dev/choysum/pkg/auth"
	"github.com/choysum-dev/choysum/pkg/jsexecutor"
	"github.com/choysum-dev/choysum/pkg/registry"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/choysum-dev/choysum/pkg/trace"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/resolver"
)

var (
	flagMaxCallRecvMsgSize int = 1024 * 1024 * 32 // 32MB,  https://github.com/grpc/grpc-go/blob/v1.8.2/server.go#L54
	flagMaxCallSendMsgSize int = 1024 * 1024 * 32 // 32MB,
)

type GRPCWebServer struct {
	runtimeScope   scope.Scope
	runtimeOptions runtimeOptions
	address        *resolver.Address
	listener       net.Listener
	httpServer     *http.Server
	registry       registry.Registry          // Grpc Service registry
	authenticator  auth.Authenticator         // Server authenticator.
	jsExecutor     jsexecutor.JsExecutor      // Runtime JS executor.
	proxy          *grpcweb.WrappedGrpcServer // gRPC-Web proxy
	server         *grpc.Server               // gRPC server
	mux            *http.ServeMux             // HTTP server

	telemetry trace.Telemetry

	grpcClientPool *transport.GRPCClientPool

	hotreload       hotreloadState
	runState        runState
	taskRuntime     taskRuntimeState
	registration    registrationState
	runtimeRecovery runtimeRecoveryState
	modeSwitch      modeSwitchState

	ready atomic.Bool
}

func (s *GRPCWebServer) Serve(ctx context.Context, serviceNames ...string) error {
	if ctx != nil && s.runtimeScope != nil {
		s.runtimeScope = s.runtimeScope.WithContext(ctx)
	}

	if err := s.planServe(serviceNames); err != nil {
		return err
	}

	return s.serve()
}
