// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package server

import (
	"context"
	"net/http"
	"time"

	"github.com/choysum-dev/choysum/internal/server/transport"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func (s *GRPCWebServer) startHTTPServer(h http.Handler) error {
	opts := s.resolvedRuntimeOptions()
	serverHandle, err := transport.StartHTTPServer(h, transport.HTTPServerOptions{
		Address:          s.address.Addr,
		ExistingListener: s.listener,
		HasProxy:         s.proxy != nil,
		EnableTLS:        opts.enabledTLS,
		TLSCertFile:      opts.tlsCertFile,
		TLSKeyFile:       opts.tlsKeyFile,
		Logger:           s.runtimeScope.Logger(),
	})
	if err != nil {
		return err
	}
	s.listener = serverHandle.Listener
	s.httpServer = serverHandle.Server
	return nil
}

func (s *GRPCWebServer) stopHTTPRuntime() {
	if s.httpServer == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := s.httpServer.Shutdown(ctx); err != nil {
		// Best-effort: on failure, force close.
		_ = s.httpServer.Close()
	}
	s.httpServer = nil
	// Listener is closed by Shutdown; ensure we re-bind on next start.
	s.listener = nil
}

func (s *GRPCWebServer) startTransportIngress(opts runtimeOptions) error {
	s.syncGRPCWebProxy(opts)
	return s.startHTTPServer(h2c.NewHandler(s.newProtocolRouter(), &http2.Server{}))
}

func (s *GRPCWebServer) newGRPCWebProxy() *grpcweb.WrappedGrpcServer {
	opts := s.resolvedRuntimeOptions()
	return transport.NewGRPCWebProxy(transport.GRPCWebProxyOptions{
		Registry:       s.registry,
		Address:        s.address.Addr,
		EnableTLS:      opts.enabledTLS,
		TLSCaFile:      opts.tlsCaFile,
		TLSServerName:  opts.tlsServerName,
		Logger:         s.runtimeScope.Logger(),
		MaxRecvMsgSize: flagMaxCallRecvMsgSize,
		MaxSendMsgSize: flagMaxCallSendMsgSize,
	})
}

func (s *GRPCWebServer) syncGRPCWebProxy(opts runtimeOptions) {
	s.proxy = nil
	if opts.enableGrpcWebProxy {
		s.proxy = s.newGRPCWebProxy()
	}
}

func (s *GRPCWebServer) newProtocolRouter() http.Handler {
	return transport.NewProtocolRouter(s.protocolRouterOptions())
}

func (s *GRPCWebServer) protocolRouterOptions() transport.ProtocolRouterOptions {
	opts := s.resolvedRuntimeOptions()
	return transport.ProtocolRouterOptions{
		RuntimeScope:    s.runtimeScope,
		Mux:             s.mux,
		GRPCServer:      s.server,
		GRPCWebProxy:    s.proxy,
		GRPCClientPool:  s.grpcClientPool,
		Authenticator:   s.authenticator,
		CSPEnabled:      opts.cspEnabled,
		CSRFEnabled:     opts.csrfEnabled,
		AuthEnabled:     opts.authEnabled,
		HTTPAuthEnabled: opts.httpAuthEnabled,
		EnableGzip:      opts.enableGzip,
	}
}
