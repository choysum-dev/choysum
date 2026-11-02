// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package setupgate

import (
	"context"
	"strconv"
	"strings"

	"github.com/choysum-dev/choysum/internal/provision"
	middleware "github.com/choysum-dev/choysum/internal/server/middleware/auth"
	"github.com/choysum-dev/choysum/pkg/scope"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// sessionEstablishingMethods create or renew end-user sessions and must wait for Ready.
var sessionEstablishingMethods = []string{
	"auth.User/Login",
	"auth.User/Register",
	"auth.User/RefreshTokens",
}

// UnaryInterceptor rejects depth=0 session-establishing RPCs until setup is Ready.
// CompleteInitialSetup and other methods are not gated here.
func UnaryInterceptor(runtimeScope scope.Scope) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if err := enforceReadyForSessionMethod(ctx, runtimeScope, info.FullMethod); err != nil {
			return nil, err
		}
		return handler(ctx, req)
	}
}

// StreamInterceptor mirrors UnaryInterceptor for streaming RPCs.
func StreamInterceptor(runtimeScope scope.Scope) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if err := enforceReadyForSessionMethod(ss.Context(), runtimeScope, info.FullMethod); err != nil {
			return err
		}
		return handler(srv, ss)
	}
}

func enforceReadyForSessionMethod(ctx context.Context, runtimeScope scope.Scope, fullMethod string) error {
	if getDepthFromIncomingMetadata(ctx) != 0 {
		return nil
	}
	if !middleware.IsMethodExcluded(fullMethod, sessionEstablishingMethods) {
		return nil
	}

	phase, err := provision.Resolve(runtimeScope)
	if err != nil {
		return status.Errorf(codes.FailedPrecondition, "initial setup state unavailable")
	}
	if phase != provision.PhaseReady {
		return status.Error(codes.FailedPrecondition, "initial setup is not complete")
	}
	return nil
}

func getDepthFromIncomingMetadata(ctx context.Context) int {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return 0
	}
	vals := md.Get("x-choysum-depth")
	if len(vals) == 0 {
		return 0
	}
	v := strings.TrimSpace(vals[0])
	if v == "" {
		return 0
	}
	if n, err := strconv.Atoi(v); err == nil && n >= 0 {
		return n
	}
	return 0
}
