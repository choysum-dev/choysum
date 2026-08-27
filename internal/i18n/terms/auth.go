// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package terms

import (
	"context"
	"strconv"
	"strings"

	"github.com/choysum-dev/choysum/pkg/auth"
	"github.com/choysum-dev/choysum/pkg/scope"
	"google.golang.org/grpc/metadata"
)

const InternalKeyHeader = internalKeyHeader

const internalKeyHeader = "x-choysum-internal-key"

// OutgoingContextForUserRPC forwards the caller access token for user-scoped RPCs.
func OutgoingContextForUserRPC(ctx context.Context, accessToken string) context.Context {
	md := metadata.MD{}
	if in, ok := metadata.FromIncomingContext(ctx); ok {
		md = cloneMD(in)
	}
	if out, ok := metadata.FromOutgoingContext(ctx); ok {
		for k, values := range out {
			if len(md.Get(k)) == 0 {
				copied := make([]string, len(values))
				copy(copied, values)
				md[k] = copied
			}
		}
	}
	token := strings.TrimSpace(accessToken)
	if token == "" {
		if t, ok := auth.AccessTokenFromContext(ctx); ok {
			token = strings.TrimSpace(t)
		}
	}
	if token != "" && len(md.Get("authorization")) == 0 {
		md.Set("authorization", "Bearer "+token)
	}
	depth := 0
	if values := md.Get("x-choysum-depth"); len(values) > 0 {
		if parsed, err := strconv.Atoi(strings.TrimSpace(values[0])); err == nil && parsed >= 0 {
			depth = parsed
		}
	}
	md.Set("x-choysum-depth", strconv.Itoa(depth+1))
	return metadata.NewOutgoingContext(ctx, md)
}

// OutgoingContextForInternalRPC attaches the workspace internal key for CLI/service RPCs.
func OutgoingContextForInternalRPC(ctx context.Context, runtimeScope scope.Scope) context.Context {
	md := metadata.MD{}
	if in, ok := metadata.FromIncomingContext(ctx); ok {
		md = cloneMD(in)
	}
	if authOpts, ok := scope.AuthRuntimeOptionsFromScope(runtimeScope); ok {
		key := strings.TrimSpace(authOpts.InternalKey)
		env := ""
		if serverOpts, ok := scope.ServerRuntimeOptionsFromScope(runtimeScope); ok {
			env = strings.TrimSpace(serverOpts.Environment)
		}
		if key != "" && !strings.EqualFold(env, "production") && len(md.Get(internalKeyHeader)) == 0 {
			md.Set(internalKeyHeader, key)
		}
	}
	md.Set("x-choysum-depth", "1")
	return metadata.NewOutgoingContext(ctx, md)
}

// HasOutgoingInternalKey reports whether ctx carries an internal service key.
func HasOutgoingInternalKey(ctx context.Context) bool {
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		return false
	}
	return strings.TrimSpace(firstMDValue(md.Get(internalKeyHeader))) != ""
}

func firstMDValue(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func cloneMD(md metadata.MD) metadata.MD {
	if md == nil {
		return metadata.MD{}
	}
	out := metadata.MD{}
	for k, values := range md {
		copied := make([]string, len(values))
		copy(copied, values)
		out[k] = copied
	}
	return out
}
