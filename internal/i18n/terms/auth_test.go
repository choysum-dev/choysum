// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package terms

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/auth"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/scope"
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

func TestOutgoingContextForUserRPCMergesOutgoingMetadata(t *testing.T) {
	base := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("x-custom", "keep"))
	out := OutgoingContextForUserRPC(base, "tok")
	md, ok := metadata.FromOutgoingContext(out)
	if !ok {
		t.Fatal("missing outgoing metadata")
	}
	if got := md.Get("x-custom"); len(got) != 1 || got[0] != "keep" {
		t.Fatalf("x-custom = %#v", got)
	}
	if got := md.Get("authorization"); len(got) != 1 || got[0] != "Bearer tok" {
		t.Fatalf("authorization = %#v", got)
	}
}

func TestOutgoingContextForInternalRPC(t *testing.T) {
	rs := newTermsAuthTestScope(t)
	out := OutgoingContextForInternalRPC(context.Background(), rs)
	md, ok := metadata.FromOutgoingContext(out)
	if !ok {
		t.Fatal("missing outgoing metadata")
	}
	if got := md.Get(InternalKeyHeader); len(got) != 1 || got[0] != "test-internal-key" {
		t.Fatalf("internal key = %#v", got)
	}
	if got := md.Get("x-choysum-depth"); len(got) != 1 || got[0] != "1" {
		t.Fatalf("depth = %#v", got)
	}
}

func TestOutgoingContextForInternalRPCEmptyKey(t *testing.T) {
	rs := &termsAuthTestScope{cfg: &config.Config{
		Auth:   &config.AuthConfig{InternalKey: ""},
		Server: &config.ServerConfig{Environment: "development"},
	}}
	out := OutgoingContextForInternalRPC(context.Background(), rs)
	md, _ := metadata.FromOutgoingContext(out)
	if len(md.Get(InternalKeyHeader)) != 0 {
		t.Fatalf("internal key = %#v", md.Get(InternalKeyHeader))
	}
}

func TestOutgoingContextForInternalRPCNilScope(t *testing.T) {
	out := OutgoingContextForInternalRPC(context.Background(), nil)
	md, _ := metadata.FromOutgoingContext(out)
	if len(md.Get(InternalKeyHeader)) != 0 {
		t.Fatalf("internal key = %#v, want omitted without scope", md.Get(InternalKeyHeader))
	}
}

func TestOutgoingContextForInternalRPCSkipsKeyInProduction(t *testing.T) {
	rs := newTermsAuthTestScope(t)
	rs.cfg.Server.Environment = "production"
	out := OutgoingContextForInternalRPC(context.Background(), rs)
	md, _ := metadata.FromOutgoingContext(out)
	if len(md.Get(InternalKeyHeader)) != 0 {
		t.Fatalf("internal key = %#v, want omitted in production", md.Get(InternalKeyHeader))
	}
}

func TestOutgoingContextForInternalRPCPreservesExistingKey(t *testing.T) {
	in := metadata.NewIncomingContext(context.Background(), metadata.Pairs(InternalKeyHeader, "existing"))
	out := OutgoingContextForInternalRPC(in, newTermsAuthTestScope(t))
	md, _ := metadata.FromOutgoingContext(out)
	if got := md.Get(InternalKeyHeader); len(got) != 1 || got[0] != "existing" {
		t.Fatalf("internal key = %#v", got)
	}
}

func TestHasOutgoingInternalKey(t *testing.T) {
	if HasOutgoingInternalKey(context.Background()) {
		t.Fatal("expected false without outgoing metadata")
	}
	ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs(InternalKeyHeader, " key "))
	if !HasOutgoingInternalKey(ctx) {
		t.Fatal("expected true with internal key")
	}
}

func TestFirstMDValueEmpty(t *testing.T) {
	if firstMDValue(nil) != "" {
		t.Fatal("expected empty")
	}
}

type termsAuthTestScope struct {
	cfg *config.Config
}

func newTermsAuthTestScope(t *testing.T) *termsAuthTestScope {
	t.Helper()
	return &termsAuthTestScope{cfg: &config.Config{
		Auth:   &config.AuthConfig{InternalKey: "test-internal-key"},
		Server: &config.ServerConfig{Environment: "development"},
	}}
}

func (s *termsAuthTestScope) Run(fn func(scope.Scope) error) error { return fn(s) }
func (s *termsAuthTestScope) Transactor() scope.Transactor         { panic("unused") }
func (s *termsAuthTestScope) Session() *scope.Session              { return nil }
func (s *termsAuthTestScope) WithContext(context.Context) scope.Scope {
	return s
}
func (s *termsAuthTestScope) Context() context.Context { return context.Background() }
func (s *termsAuthTestScope) Logger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
func (s *termsAuthTestScope) Config() *config.Config { return s.cfg }
func (s *termsAuthTestScope) FactoryInput() scope.FactoryInput {
	return scopetest.FactoryInputFromConfig(s.cfg)
}
