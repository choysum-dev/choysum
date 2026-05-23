// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package server

import (
	"github.com/choysum-dev/choysum/internal/server/transport"
	xfmt "golang.org/x/exp/errors/fmt"
	"google.golang.org/grpc"
)

func (s *GRPCWebServer) initGRPCClientPool() error {
	opts := s.resolvedRuntimeOptions()
	maxCached := opts.grpcClientMaxCachedConn
	pool, err := transport.NewGRPCClientPool(transport.GRPCClientPoolOptions{
		Registry:      s.registry,
		Address:       s.address.Addr,
		MaxCachedConn: maxCached,
		EnableTLS:     opts.enabledTLS,
		TLSCaFile:     opts.tlsCaFile,
		TLSServerName: opts.tlsServerName,
		TLSCertFile:   opts.tlsCertFile,
		TLSKeyFile:    opts.tlsKeyFile,
	})
	if err != nil {
		return xfmt.Errorf("failed to create grpc client pool: %w", err)
	}
	s.grpcClientPool = pool
	return nil
}

func (s *GRPCWebServer) startGRPCTransport(opts runtimeOptions) error {
	if err := s.initGRPCClientPool(); err != nil {
		return err
	}
	s.assembleGRPCServer(opts)
	return nil
}

func (s *GRPCWebServer) assembleGRPCServer(opts runtimeOptions) {
	s.server = transport.NewGRPCServer(s.grpcServerOptions())
}

func (s *GRPCWebServer) stopGRPCTransport() {
	if s.server != nil {
		s.server.GracefulStop()
		s.server = nil
	}

	if s.grpcClientPool != nil {
		s.grpcClientPool.CloseAll()
		s.grpcClientPool = nil
	}
}

func (s *GRPCWebServer) baseUnaryInterceptors() []grpc.UnaryServerInterceptor {
	if s.grpcClientPool == nil {
		return nil
	}
	return []grpc.UnaryServerInterceptor{
		s.grpcClientPool.UnaryServerInterceptor(),
	}
}
