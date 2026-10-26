// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package server

import (
	"time"

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

type grpcServerStopper interface {
	GracefulStop()
	Stop()
}

func (s *GRPCWebServer) stopGRPCTransport(reload bool) {
	if s.server != nil {
		if reload {
			// Hot reload keeps process context live; wait for in-flight RPCs like before.
			s.server.GracefulStop()
		} else {
			stopGRPCServerWithTimeout(s.server, 2*time.Second, 2*time.Second)
		}
		s.server = nil
	}

	if s.grpcClientPool != nil {
		s.grpcClientPool.CloseAll()
		s.grpcClientPool = nil
	}
}

// stopGRPCServerWithTimeout waits for GracefulStop up to gracefulTimeout, then
// calls Stop asynchronously and waits up to forceWait more. If both waiters
// expire, it returns so process exit can continue (grpc-go may hold server.mu
// across handlersWG.Wait inside GracefulStop).
func stopGRPCServerWithTimeout(srv grpcServerStopper, gracefulTimeout, forceWait time.Duration) {
	if srv == nil {
		return
	}
	done := make(chan struct{})
	go func() {
		srv.GracefulStop()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(gracefulTimeout):
		go srv.Stop()
		select {
		case <-done:
		case <-time.After(forceWait):
		}
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
