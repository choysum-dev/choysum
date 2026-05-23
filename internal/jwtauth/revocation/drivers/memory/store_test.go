// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package memory

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/choysum-dev/choysum/internal/jwtauth/revocation"
	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/auth"
	"github.com/choysum-dev/choysum/pkg/auth/autherrors"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/scope"
)

type memoryTestScope struct {
	ctx    context.Context
	cfg    *config.Config
	logger *slog.Logger
}

func (f *memoryTestScope) Run(fn func(scope.Scope) error) error { return fn(f) }
func (f *memoryTestScope) Transactor() scope.Transactor {
	return scopetest.NewPassthroughTransactor(f)
}
func (f *memoryTestScope) Session() *scope.Session { return nil }
func (f *memoryTestScope) WithContext(ctx context.Context) scope.Scope {
	return &memoryTestScope{ctx: ctx, cfg: f.cfg, logger: f.logger}
}
func (f *memoryTestScope) Context() context.Context { return f.ctx }
func (f *memoryTestScope) Logger() *slog.Logger     { return f.logger }
func (f *memoryTestScope) Config() *config.Config   { return f.cfg }
func (f *memoryTestScope) FactoryInput() scope.FactoryInput {
	return scopetest.FactoryInputFromConfig(f.Config())
}

func newMemoryScope() *memoryTestScope {
	return &memoryTestScope{
		ctx:    context.Background(),
		cfg:    &config.Config{Auth: config.NewDefaultAuthConfig(), Log: config.NewDefaultLogConfig()},
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

func TestMemoryStoreRevokeAndIsRevoked(t *testing.T) {
	storeIface, err := NewMemoryStore(newMemoryScope())
	if err != nil {
		t.Fatalf("NewMemoryStore error: %v", err)
	}
	store := storeIface.(*MemoryStore)
	t.Cleanup(func() { _ = store.Close() })

	if _, err := store.IsRevoked(context.Background(), ""); err == nil || !autherrors.IsAuthError(err, autherrors.ErrInvalidTokenID) {
		t.Fatalf("expected invalid token id error, got %v", err)
	}
	if revoked, err := store.IsRevoked(context.Background(), "token-1"); err != nil || revoked {
		t.Fatalf("initial revoked=%v err=%v, want false nil", revoked, err)
	}
	if err := store.RevokeToken(context.Background(), "token-1", "user-1", auth.AccessToken, time.Now().Add(time.Hour), "reason"); err != nil {
		t.Fatalf("RevokeToken error: %v", err)
	}
	if revoked, err := store.IsRevoked(context.Background(), "token-1"); err != nil || !revoked {
		t.Fatalf("revoked=%v err=%v, want true nil", revoked, err)
	}
	if err := store.RevokeToken(context.Background(), "token-1", "user-1", auth.AccessToken, time.Now().Add(time.Hour), "reason"); err == nil || !autherrors.IsAuthError(err, autherrors.ErrTokenAlreadyRevoked) {
		t.Fatalf("expected duplicate revoke error, got %v", err)
	}
	if err := store.RevokeToken(context.Background(), "", "user-1", auth.AccessToken, time.Now().Add(time.Hour), "reason"); err == nil || !autherrors.IsAuthError(err, autherrors.ErrInvalidTokenID) {
		t.Fatalf("expected invalid token id error, got %v", err)
	}
	if err := store.RevokeToken(context.Background(), "token-2", "", auth.AccessToken, time.Now().Add(time.Hour), "reason"); err == nil || !autherrors.IsAuthError(err, autherrors.ErrInvalidUserID) {
		t.Fatalf("expected invalid user id error, got %v", err)
	}
}

func TestMemoryStoreRevokeAllAndCleanExpired(t *testing.T) {
	storeIface, err := NewMemoryStore(newMemoryScope())
	if err != nil {
		t.Fatalf("NewMemoryStore error: %v", err)
	}
	store := storeIface.(*MemoryStore)
	t.Cleanup(func() { _ = store.Close() })

	store.userIndex["user-1"] = []string{"token-a", "token-b", "token-keep"}
	n, err := store.RevokeAllUserTokens(context.Background(), "user-1", "token-keep", "")
	if err != nil {
		t.Fatalf("RevokeAllUserTokens error: %v", err)
	}
	if n != 2 {
		t.Fatalf("revoke count = %d, want 2", n)
	}
	if _, ok := store.tokens["token-a"]; !ok {
		t.Fatal("expected token-a revoked")
	}
	if _, ok := store.tokens["token-b"]; !ok {
		t.Fatal("expected token-b revoked")
	}
	if _, ok := store.tokens["token-keep"]; ok {
		t.Fatal("expected except token to remain active")
	}
	if len(store.userIndex["user-1"]) != 1 || store.userIndex["user-1"][0] != "token-keep" {
		t.Fatalf("unexpected user index after revoke all: %#v", store.userIndex["user-1"])
	}

	store.tokens["expired"] = &revocation.StandardToken{TokenID: "expired", UserID: "user-2", TokenType: auth.AccessToken, RevokedAt: time.Now(), ExpiresAt: time.Now().Add(-time.Hour), Reason: "old"}
	store.userIndex["user-2"] = []string{"expired"}
	removed, err := store.CleanExpired(context.Background())
	if err != nil {
		t.Fatalf("CleanExpired error: %v", err)
	}
	if removed != 1 {
		t.Fatalf("removed = %d, want 1", removed)
	}
	if _, ok := store.tokens["expired"]; ok {
		t.Fatal("expected expired token to be removed")
	}
	if _, ok := store.userIndex["user-2"]; ok {
		t.Fatal("expected expired user index entry to be removed")
	}

	if _, err := store.RevokeAllUserTokens(context.Background(), "", "", ""); err == nil || !autherrors.IsAuthError(err, autherrors.ErrInvalidUserID) {
		t.Fatalf("expected invalid user id error, got %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close error: %v", err)
	}
}
