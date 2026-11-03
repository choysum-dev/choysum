// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package server

import (
	"github.com/choysum-dev/choysum/internal/server/middleware/auth/grpcauth"
	"github.com/choysum-dev/choysum/internal/server/middleware/auth/setupgate"
	"github.com/choysum-dev/choysum/pkg/auth"
	xfmt "golang.org/x/exp/errors/fmt"
	"google.golang.org/grpc"
)

type authRuntimeSetupResult struct {
	Enabled    bool
	Configured bool
	Degraded   bool
	Err        error
}

func (s *GRPCWebServer) startAuthRuntime(opts runtimeOptions) authRuntimeSetupResult {
	result := authRuntimeSetupResult{Enabled: opts.authEnabled}
	s.authenticator = nil
	if !opts.authEnabled {
		return result
	}

	authenticator, err := auth.NewAuthenticator(s.runtimeScope)
	if err != nil {
		result.Degraded = true
		result.Err = xfmt.Errorf("failed to initialize authenticator: %w", err)
		return result
	}

	s.authenticator = authenticator
	result.Configured = true
	return result
}

func (r authRuntimeSetupResult) logFields() []any {
	fields := []any{
		"startup_auth_enabled", r.Enabled,
		"startup_auth_configured", r.Configured,
		"startup_auth_degraded", r.Degraded,
	}
	if r.Err != nil {
		fields = append(fields, "startup_auth_error", r.Err)
	}
	return fields
}

func (s *GRPCWebServer) applyAuthGRPCInterceptors(unaryInterceptors []grpc.UnaryServerInterceptor) (unary []grpc.UnaryServerInterceptor, stream grpc.StreamServerInterceptor) {
	unary = unaryInterceptors
	setupUnary := setupgate.UnaryInterceptor(s.runtimeScope)
	setupStream := setupgate.StreamInterceptor(s.runtimeScope)

	if s.authenticator == nil {
		// Still gate session-establishing RPCs when auth runtime is unavailable.
		unary = append([]grpc.UnaryServerInterceptor{setupUnary}, unary...)
		return unary, setupStream
	}
	unary = append(
		[]grpc.UnaryServerInterceptor{
			grpcauth.AuthInterceptorFromConfig(s.runtimeScope, s.authenticator),
			setupUnary,
		},
		unary...,
	)
	authStream := grpcauth.StreamInterceptorFromConfig(s.runtimeScope, s.authenticator)
	return unary, chainStreamServer(authStream, setupStream)
}

// chainStreamServer runs interceptors in order (outer first). Nil entries are skipped.
func chainStreamServer(interceptors ...grpc.StreamServerInterceptor) grpc.StreamServerInterceptor {
	filtered := make([]grpc.StreamServerInterceptor, 0, len(interceptors))
	for _, interceptor := range interceptors {
		if interceptor != nil {
			filtered = append(filtered, interceptor)
		}
	}
	if len(filtered) == 0 {
		return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
			return handler(srv, ss)
		}
	}
	if len(filtered) == 1 {
		return filtered[0]
	}
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		chain := handler
		for i := len(filtered) - 1; i >= 0; i-- {
			interceptor := filtered[i]
			next := chain
			chain = func(srv interface{}, ss grpc.ServerStream) error {
				return interceptor(srv, ss, info, next)
			}
		}
		return chain(srv, ss)
	}
}

func (s *GRPCWebServer) setupAuthInterceptors(unaryInterceptors *[]grpc.UnaryServerInterceptor) (grpc.StreamServerInterceptor, authRuntimeSetupResult) {
	result := s.startAuthRuntime(s.resolvedRuntimeOptions())
	var stream grpc.StreamServerInterceptor
	*unaryInterceptors, stream = s.applyAuthGRPCInterceptors(*unaryInterceptors)
	return stream, result
}

func (s *GRPCWebServer) stopAuthRuntime() error {
	if s.authenticator == nil {
		return nil
	}
	if err := s.authenticator.Close(); err != nil {
		return xfmt.Errorf("Failed to close authenticator: %w", err)
	}
	s.authenticator = nil
	return nil
}
