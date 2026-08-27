// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package terms

import (
	"context"
	"testing"

	"github.com/choysum-dev/choysum/pkg/auth"
	"google.golang.org/grpc/metadata"
)

func TestRpcContextUsesUserToken(t *testing.T) {
	out := rpcContext(context.Background(), "direct-token")
	md, ok := metadata.FromOutgoingContext(out)
	if !ok {
		t.Fatal("missing outgoing metadata")
	}
	if got := md.Get("authorization"); len(got) != 1 || got[0] != "Bearer direct-token" {
		t.Fatalf("authorization = %#v", got)
	}
}

func TestRpcContextUsesInternalKeyWhenTokenEmpty(t *testing.T) {
	base := metadata.NewOutgoingContext(context.Background(), metadata.Pairs(InternalKeyHeader, "cli-key"))
	out := rpcContext(base, "")
	md, ok := metadata.FromOutgoingContext(out)
	if !ok {
		t.Fatal("missing outgoing metadata")
	}
	if got := md.Get(InternalKeyHeader); len(got) != 1 || got[0] != "cli-key" {
		t.Fatalf("internal key = %#v", got)
	}
}

func TestRpcContextFallsBackToContextToken(t *testing.T) {
	ctx := auth.ContextWithAccessToken(context.Background(), "ctx-token")
	out := rpcContext(ctx, "")
	md, _ := metadata.FromOutgoingContext(out)
	if got := md.Get("authorization"); len(got) != 1 || got[0] != "Bearer ctx-token" {
		t.Fatalf("authorization = %#v", got)
	}
}
