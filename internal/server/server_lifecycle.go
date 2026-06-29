// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package server

import (
	"github.com/choysum-dev/choysum/internal/server/transport"
	xfmt "golang.org/x/exp/errors/fmt"
	"google.golang.org/grpc"
)

func (s *GRPCWebServer) serve() error {
	if err := s.startHotreloadLifecycle(); err != nil {
		return xfmt.Errorf("failed to start hotreload lifecycle: %v", err)
	}
	defer s.stopHotreloadLifecycle()

	if err := s.start(false); err != nil {
		return xfmt.Errorf("failed to start server: %v", err)
	}
	defer s.stop(false)

	for {
		select {
		case <-s.runtimeScope.Context().Done():
			return nil
		case file := <-s.hotreloadQueue():
			if err := s.handleQueuedWatchEvent(file); err != nil {
				return err
			}
		}
	}
}

func (s *GRPCWebServer) grpcServerOptions() transport.GRPCServerOptions {
	serverOpts := []grpc.ServerOption{}

	s.ensureTelemetry(&serverOpts)

	unaryInterceptors := s.baseUnaryInterceptors()
	serverOpts, unaryInterceptors = s.applyAuthGRPCInterceptors(serverOpts, unaryInterceptors)
	unaryInterceptors = append(unaryInterceptors, s.taskRuntimeUnaryInterceptors()...)

	return transport.GRPCServerOptions{
		MaxRecvMsgSize:    flagMaxCallRecvMsgSize,
		MaxSendMsgSize:    flagMaxCallSendMsgSize,
		ServerOptions:     serverOpts,
		UnaryInterceptors: unaryInterceptors,
	}
}

func (s *GRPCWebServer) start(reload bool) error {
	return s.runStartupLifecycle(reload, s.resolvedRuntimeOptions()).errorValue()
}

func (s *GRPCWebServer) stop(reload bool) error {
	s.ready.Store(false)
	s.stopHTTPRuntime()
	if err := s.clearWatchRegistrations(); err != nil {
		return err
	}

	s.stopTaskRuntime()

	if err := s.unregisterAllServices(); err != nil {
		return err
	}

	if err := s.stopJSExecutor(reload); err != nil {
		return err
	}

	// if !reload && s.compileExecutor != nil {
	// 	if err := s.compileExecutor.Stop(); err != nil {
	// 		return xfmt.Errorf("Failed to stop compile executor: %w", err)
	// 	}
	// }

	if err := s.stopAuthRuntime(); err != nil {
		return err
	}

	s.stopGRPCTransport()
	s.stopTelemetryRuntime()

	if reload {
		s.runtimeScope.Logger().Info("server restarting in application mode")
	} else {
		s.runtimeScope.Logger().Info("server stopped")
	}
	return nil
}
