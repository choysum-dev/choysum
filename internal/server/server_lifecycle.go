// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package server

import (
	"log/slog"

	"github.com/choysum-dev/choysum/internal/server/middleware/accesslog"
	"github.com/choysum-dev/choysum/internal/server/transport"
	"github.com/choysum-dev/choysum/pkg/scope"
	xfmt "golang.org/x/exp/errors/fmt"
	"google.golang.org/grpc"
)

func (s *GRPCWebServer) accessLogConfig() (accesslog.Options, *slog.Logger) {
	opts := accesslog.Options{SlowMs: 1000}
	var logger *slog.Logger
	if s != nil && s.runtimeScope != nil {
		logger = s.runtimeScope.Logger()
		if logCfg := scope.LogConfigFromScope(s.runtimeScope); logCfg != nil {
			opts.Access = logCfg.Access
			// Honor AccessSlowMs as loaded: NewDefaultLogConfig / YAML omit keep 1000;
			// explicit 0 (or negative) disables slow-WARN per config.sample.yaml.
			if logCfg.AccessSlowMs < 0 {
				opts.SlowMs = 0
			} else {
				opts.SlowMs = logCfg.AccessSlowMs
			}
		}
	}
	return opts, logger
}

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
				// Never tear down the server on a watch reload failure — the
				// operator can fix the module and save again. LEASE_CONFLICT is
				// already soft-failed with a defer message in the dispatch path;
				// other upgrade/build errors still reach here after ✗ logging.
				if isModuleLeaseConflict(err) {
					if s.runtimeScope != nil {
						s.runtimeScope.Logger().Warn(
							"watch reload deferred: module management lease busy",
							"event", file,
							"error", err,
						)
					}
					continue
				}
				if s.runtimeScope != nil {
					s.runtimeScope.Logger().Error(
						"watch reload failed; server keeps running",
						"event", file,
						"error", err,
					)
				}
				continue
			}
		}
	}
}

func (s *GRPCWebServer) grpcServerOptions() transport.GRPCServerOptions {
	serverOpts := []grpc.ServerOption{}

	s.ensureTelemetry(&serverOpts)

	accessOpts, accessLogger := s.accessLogConfig()
	unaryInterceptors := s.baseUnaryInterceptors()

	// Auth first, then prepend access so access stays outermost and still sees
	// unauthenticated failures (auth would otherwise short-circuit them).
	var authStream grpc.StreamServerInterceptor
	unaryInterceptors, authStream = s.applyAuthGRPCInterceptors(unaryInterceptors)
	unaryInterceptors = append(
		[]grpc.UnaryServerInterceptor{accesslog.UnaryInterceptor(accessOpts, accessLogger)},
		unaryInterceptors...,
	)
	streamInterceptors := []grpc.StreamServerInterceptor{
		accesslog.StreamInterceptor(accessOpts, accessLogger),
	}
	if authStream != nil {
		streamInterceptors = append(streamInterceptors, authStream)
	}
	unaryInterceptors = append(unaryInterceptors, s.taskRuntimeUnaryInterceptors()...)

	return transport.GRPCServerOptions{
		MaxRecvMsgSize:     flagMaxCallRecvMsgSize,
		MaxSendMsgSize:     flagMaxCallSendMsgSize,
		ServerOptions:      serverOpts,
		UnaryInterceptors:  unaryInterceptors,
		StreamInterceptors: streamInterceptors,
	}
}

func (s *GRPCWebServer) start(reload bool) error {
	return s.runStartupLifecycle(reload, s.resolvedRuntimeOptions()).errorValue()
}

func (s *GRPCWebServer) stop(reload bool) error {
	s.ready.Store(false)

	// Stop producers first (bounded wait for in-flight task workers), then drain transports.
	s.stopTaskRuntime()
	s.stopHTTPRuntime(reload)
	s.stopGRPCTransport(reload)

	if err := s.clearWatchRegistrations(); err != nil {
		return err
	}

	if err := s.unregisterAllServices(); err != nil {
		return err
	}

	if err := s.stopJSExecutor(reload); err != nil {
		return err
	}

	if err := s.stopAuthRuntime(); err != nil {
		return err
	}

	s.stopTelemetryRuntime()

	if reload {
		s.runtimeScope.Logger().Info("server restarting")
	} else {
		s.runtimeScope.Logger().Info("server stopped")
	}
	return nil
}
