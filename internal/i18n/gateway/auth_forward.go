// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package i18ngateway

import (
	"context"
	"strconv"
	"strings"

	"github.com/choysum-dev/choysum/pkg/auth"
	"google.golang.org/grpc/metadata"
)

func accessTokenFromHTTP(ctx context.Context, authorizationHeader string) string {
	if token, ok := auth.AccessTokenFromContext(ctx); ok {
		t := strings.TrimSpace(token)
		if t != "" {
			return t
		}
	}
	authz := strings.TrimSpace(authorizationHeader)
	const prefix = "Bearer "
	if len(authz) > len(prefix) && strings.EqualFold(authz[:len(prefix)], prefix) {
		return strings.TrimSpace(authz[len(prefix):])
	}
	return ""
}

// requireTermsAuth accepts either a trusted Identity in context or a Bearer token.
// /web/ is HTTP-auth excluded, so IdentityFromContext is often empty and the
// Authorization header is the primary signal for PO export.
func requireTermsAuth(ctx context.Context, authorizationHeader string) (accessToken string, ok bool) {
	token := accessTokenFromHTTP(ctx, authorizationHeader)
	if id := auth.IdentityFromContext(ctx); id != nil && id.IsValid() {
		return token, true
	}
	if token == "" {
		return "", false
	}
	return token, true
}

// outgoingContextForUserRPC forwards the caller's identity (D1) for user-scoped RPCs.
func outgoingContextForUserRPC(ctx context.Context, accessToken string) context.Context {
	md := metadata.MD{}
	if in, ok := metadata.FromIncomingContext(ctx); ok {
		md = cloneMD(in)
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
	if len(md) == 0 {
		return ctx
	}
	return metadata.NewOutgoingContext(ctx, md)
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
