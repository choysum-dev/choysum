// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package i18ngateway

import (
	"context"
	"testing"

	"github.com/choysum-dev/choysum/pkg/auth"
	"google.golang.org/grpc/metadata"
)

type testIdentity struct {
	valid bool
}

func (testIdentity) GetUserID() string                   { return "user-1" }
func (testIdentity) GetTokenID() string                  { return "" }
func (testIdentity) GetMetadata() map[string]interface{} { return nil }
func (i testIdentity) IsValid() bool                     { return i.valid }

func TestAccessTokenFromHTTPPrefersContext(t *testing.T) {
	ctx := auth.ContextWithAccessToken(context.Background(), "ctx-token")
	if got := accessTokenFromHTTP(ctx, "Bearer header-token"); got != "ctx-token" {
		t.Fatalf("token = %q", got)
	}
}

func TestAccessTokenFromHTTPParsesBearerHeader(t *testing.T) {
	if got := accessTokenFromHTTP(context.Background(), "Bearer header-token"); got != "header-token" {
		t.Fatalf("token = %q", got)
	}
}

func TestRequireTermsAuthIdentityOrToken(t *testing.T) {
	ctx := auth.ContextWithIdentity(context.Background(), testIdentity{valid: true})
	if token, ok := requireTermsAuth(ctx, ""); !ok {
		t.Fatal("expected identity auth")
	} else if token != "" {
		t.Fatalf("token = %q, want empty", token)
	}

	if _, ok := requireTermsAuth(context.Background(), ""); ok {
		t.Fatal("expected unauthenticated")
	}
	if token, ok := requireTermsAuth(context.Background(), "Bearer abc"); !ok || token != "abc" {
		t.Fatalf("token=%q ok=%v", token, ok)
	}
}

func TestOutgoingContextForUserRPCIncrementsDepth(t *testing.T) {
	in := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-choysum-depth", "1", "authorization", "Bearer existing"))
	out := outgoingContextForUserRPC(in, "ignored")
	md, ok := metadata.FromOutgoingContext(out)
	if !ok {
		t.Fatal("missing outgoing metadata")
	}
	if got := md.Get("x-choysum-depth"); len(got) != 1 || got[0] != "2" {
		t.Fatalf("depth = %#v", got)
	}
	if got := md.Get("authorization"); len(got) != 1 || got[0] != "Bearer existing" {
		t.Fatalf("authorization = %#v", got)
	}
}

func TestOutgoingContextForUserRPCSetsBearerToken(t *testing.T) {
	out := outgoingContextForUserRPC(context.Background(), "rpc-token")
	md, _ := metadata.FromOutgoingContext(out)
	if got := md.Get("authorization"); len(got) != 1 || got[0] != "Bearer rpc-token" {
		t.Fatalf("authorization = %#v", got)
	}
	if got := md.Get("x-choysum-depth"); len(got) != 1 || got[0] != "1" {
		t.Fatalf("depth = %#v", got)
	}
}

func TestOutgoingContextForUserRPCUsesContextToken(t *testing.T) {
	ctx := auth.ContextWithAccessToken(context.Background(), "ctx-token")
	out := outgoingContextForUserRPC(ctx, "")
	md, _ := metadata.FromOutgoingContext(out)
	if got := md.Get("authorization"); len(got) != 1 || got[0] != "Bearer ctx-token" {
		t.Fatalf("authorization = %#v", got)
	}
}

func TestOutgoingContextForUserRPCIgnoresInvalidDepth(t *testing.T) {
	in := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-choysum-depth", "bad"))
	out := outgoingContextForUserRPC(in, "tok")
	md, _ := metadata.FromOutgoingContext(out)
	if got := md.Get("x-choysum-depth"); len(got) != 1 || got[0] != "1" {
		t.Fatalf("depth = %#v", got)
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
		t.Fatal("clone should not alias values")
	}
}
