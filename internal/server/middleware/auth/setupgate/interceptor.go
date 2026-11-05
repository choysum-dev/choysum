// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package setupgate

import (
	"context"

	"github.com/choysum-dev/choysum/internal/provision"
	middleware "github.com/choysum-dev/choysum/internal/server/middleware/auth"
	"github.com/choysum-dev/choysum/pkg/scope"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// sessionEstablishingMethods create or renew end-user sessions and must wait for Ready.
var sessionEstablishingMethods = []string{
	"auth.User/Login",
	"auth.User/Register",
	"auth.User/RefreshTokens",
}

// setupWindowMethods serve anonymous first-run setup data and are only open while setup is pending.
var setupWindowMethods = []string{
	"base.Company/GetInitialSetupOptions",
}

// UnaryInterceptor rejects session-establishing RPCs until setup is Ready and
// setup-window RPCs outside SetupPending.
// CompleteInitialSetup and other methods are not gated here.
// Depth metadata is ignored: clients can set x-choysum-depth and must not bypass the gate.
func UnaryInterceptor(runtimeScope scope.Scope) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if err := enforceSetupPhase(runtimeScope, info.FullMethod); err != nil {
			return nil, err
		}
		return handler(ctx, req)
	}
}

// StreamInterceptor mirrors UnaryInterceptor for streaming RPCs.
func StreamInterceptor(runtimeScope scope.Scope) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if err := enforceSetupPhase(runtimeScope, info.FullMethod); err != nil {
			return err
		}
		return handler(srv, ss)
	}
}

func enforceSetupPhase(runtimeScope scope.Scope, fullMethod string) error {
	sessionMethod := middleware.IsMethodExcluded(fullMethod, sessionEstablishingMethods)
	windowMethod := middleware.IsMethodExcluded(fullMethod, setupWindowMethods)
	if !sessionMethod && !windowMethod {
		return nil
	}

	phase, err := provision.Resolve(runtimeScope)
	if err != nil {
		if runtimeScope != nil && runtimeScope.Logger() != nil {
			runtimeScope.Logger().Error("setup gate: resolve setup state failed", "method", fullMethod, "error", err)
		}
		return status.Error(codes.FailedPrecondition, "initial setup state unavailable")
	}
	if sessionMethod && phase != provision.PhaseReady {
		return status.Error(codes.FailedPrecondition, "initial setup is not complete")
	}
	if windowMethod && phase != provision.PhaseSetupPending {
		return status.Error(codes.FailedPrecondition, "initial setup is not pending")
	}
	return nil
}
