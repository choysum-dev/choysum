// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package terms

import (
	"context"
	"strings"

	"github.com/choysum-dev/choysum/pkg/auth"
	"google.golang.org/grpc/metadata"
)

// OutgoingContextForUserRPC forwards the caller access token for user-scoped RPCs.
func OutgoingContextForUserRPC(ctx context.Context, accessToken string) context.Context {
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
	if len(md.Get("x-choysum-depth")) == 0 {
		md.Set("x-choysum-depth", "1")
	}
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
