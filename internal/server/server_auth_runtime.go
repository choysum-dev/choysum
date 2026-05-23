// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package server

import (
	"github.com/choysum-dev/choysum/internal/server/middleware/auth/grpcauth"
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

func (s *GRPCWebServer) applyAuthGRPCInterceptors(serverOpts []grpc.ServerOption, unaryInterceptors []grpc.UnaryServerInterceptor) ([]grpc.ServerOption, []grpc.UnaryServerInterceptor) {
	if s.authenticator == nil {
		return serverOpts, unaryInterceptors
	}
	unaryInterceptors = append(
		[]grpc.UnaryServerInterceptor{grpcauth.AuthInterceptorFromConfig(s.runtimeScope, s.authenticator)},
		unaryInterceptors...,
	)
	serverOpts = append(serverOpts, grpc.StreamInterceptor(grpcauth.StreamInterceptorFromConfig(s.runtimeScope, s.authenticator)))
	return serverOpts, unaryInterceptors
}

func (s *GRPCWebServer) setupAuthInterceptors(serverOpts *[]grpc.ServerOption, unaryInterceptors *[]grpc.UnaryServerInterceptor) authRuntimeSetupResult {
	result := s.startAuthRuntime(s.resolvedRuntimeOptions())
	*serverOpts, *unaryInterceptors = s.applyAuthGRPCInterceptors(*serverOpts, *unaryInterceptors)
	return result
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
