// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package terms

import (
	"context"
	"testing"

	"github.com/choysum-dev/choysum/pkg/auth"
	"google.golang.org/grpc/metadata"
)

func TestOutgoingContextForUserRPCIncrementsDepth(t *testing.T) {
	in := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-choysum-depth", "0"))
	out := OutgoingContextForUserRPC(in, "tok")
	md, ok := metadata.FromOutgoingContext(out)
	if !ok {
		t.Fatal("missing outgoing metadata")
	}
	if got := md.Get("x-choysum-depth"); len(got) != 1 || got[0] != "1" {
		t.Fatalf("depth = %#v, want 1", got)
	}
	if got := md.Get("authorization"); len(got) != 1 || got[0] != "Bearer tok" {
		t.Fatalf("authorization = %#v", got)
	}

	in = metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-choysum-depth", "2"))
	out = OutgoingContextForUserRPC(in, "")
	md, _ = metadata.FromOutgoingContext(out)
	if got := md.Get("x-choysum-depth"); len(got) != 1 || got[0] != "3" {
		t.Fatalf("depth = %#v, want 3", got)
	}
}

func TestOutgoingContextForUserRPCUsesContextToken(t *testing.T) {
	ctx := auth.ContextWithAccessToken(context.Background(), "ctx-token")
	out := OutgoingContextForUserRPC(ctx, "")
	md, ok := metadata.FromOutgoingContext(out)
	if !ok {
		t.Fatal("missing outgoing metadata")
	}
	if got := md.Get("authorization"); len(got) != 1 || got[0] != "Bearer ctx-token" {
		t.Fatalf("authorization = %#v", got)
	}
}

func TestOutgoingContextForUserRPCPreservesAuthorization(t *testing.T) {
	in := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer existing"))
	out := OutgoingContextForUserRPC(in, "new-token")
	md, _ := metadata.FromOutgoingContext(out)
	if got := md.Get("authorization"); len(got) != 1 || got[0] != "Bearer existing" {
		t.Fatalf("authorization = %#v", got)
	}
}

func TestOutgoingContextForUserRPCIgnoresInvalidDepth(t *testing.T) {
	in := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-choysum-depth", "bad"))
	out := OutgoingContextForUserRPC(in, "")
	md, _ := metadata.FromOutgoingContext(out)
	if got := md.Get("x-choysum-depth"); len(got) != 1 || got[0] != "1" {
		t.Fatalf("depth = %#v, want 1", got)
	}
}

func TestOutgoingContextForUserRPCDefaultsDepth(t *testing.T) {
	ctx := context.Background()
	out := OutgoingContextForUserRPC(ctx, "")
	md, ok := metadata.FromOutgoingContext(out)
	if !ok || md.Get("x-choysum-depth")[0] != "1" {
		t.Fatalf("depth = %#v", md.Get("x-choysum-depth"))
	}
}

func TestCloneMD(t *testing.T) {
	if got := cloneMD(nil); len(got) != 0 {
		t.Fatalf("clone nil = %#v", got)
	}
	src := metadata.Pairs("k", "v")
	dst := cloneMD(src)
	dst.Set("k", "changed")
	if src.Get("k")[0] == "changed" {
		t.Fatal("clone should not alias slices")
	}
}
