// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package gateway

import (
	"context"
	"fmt"
	"strings"

	"github.com/choysum-dev/choysum/pkg/auth"
	"google.golang.org/grpc/metadata"
)

func outgoingContextForAuthRPC(ctx context.Context) context.Context {
	md := metadata.MD{}
	if in, ok := metadata.FromIncomingContext(ctx); ok {
		md = cloneOutgoingMetadata(in)
	}

	if token, ok := auth.AccessTokenFromContext(ctx); ok {
		t := strings.TrimSpace(token)
		if t != "" && len(md.Get("authorization")) == 0 {
			md.Set("authorization", "Bearer "+t)
		}
	}

	if len(md.Get("x-choysum-depth")) == 0 {
		md.Set("x-choysum-depth", "1")
	}

	if len(md) == 0 {
		return ctx
	}
	return metadata.NewOutgoingContext(ctx, md)
}

func cloneOutgoingMetadata(md metadata.MD) metadata.MD {
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

func enabledCompanyIDsFromIdentity(identity auth.Identity, activeCompanyID string) []string {
	out := []string{}
	seen := map[string]struct{}{}
	add := func(v string) {
		t := strings.TrimSpace(v)
		if t == "" {
			return
		}
		if _, exists := seen[t]; exists {
			return
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}

	if identity != nil {
		metadata := identity.GetMetadata()
		if metadata != nil {
			switch vv := metadata["enabledCompanyIds"].(type) {
			case []string:
				for _, item := range vv {
					add(item)
				}
			case []any:
				for _, item := range vv {
					add(normalizeOptionalText(item))
				}
			}
			if activeCompanyID == "" {
				if v, ok := metadata["activeCompanyId"].(string); ok {
					activeCompanyID = v
				}
			}
		}
	}

	add(activeCompanyID)
	return out
}

func normalizeOptionalText(value any) string {
	if value == nil {
		return ""
	}
	if text, ok := value.(string); ok {
		return strings.TrimSpace(text)
	}
	text := strings.TrimSpace(fmt.Sprintf("%v", value))
	if text == "<nil>" {
		return ""
	}
	return text
}

func asRecord(value any) map[string]any {
	record, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	return record
}
