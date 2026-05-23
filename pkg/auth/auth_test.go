// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package auth

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/auth/autherrors"
	"github.com/choysum-dev/choysum/pkg/cache"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/scope"
)

type authTestScope struct {
	ctx    context.Context
	cfg    *config.Config
	logger *slog.Logger
}

func (f *authTestScope) Run(fn func(scope.Scope) error) error { return fn(f) }
func (f *authTestScope) Transactor() scope.Transactor {
	return scopetest.NewPassthroughTransactor(f)
}
func (f *authTestScope) Session() *scope.Session { return nil }
func (f *authTestScope) WithContext(ctx context.Context) scope.Scope {
	return &authTestScope{ctx: ctx, cfg: f.cfg, logger: f.logger}
}
func (f *authTestScope) Context() context.Context { return f.ctx }
func (f *authTestScope) Logger() *slog.Logger     { return f.logger }
func (f *authTestScope) Config() *config.Config   { return f.cfg }
func (f *authTestScope) FactoryInput() scope.FactoryInput {
	return scopetest.FactoryInputFromConfig(f.cfg)
}

type authTestIdentity struct {
	userID   string
	tokenID  string
	metadata map[string]interface{}
	valid    bool
}

func (i authTestIdentity) GetUserID() string                   { return i.userID }
func (i authTestIdentity) GetTokenID() string                  { return i.tokenID }
func (i authTestIdentity) GetMetadata() map[string]interface{} { return i.metadata }
func (i authTestIdentity) IsValid() bool                       { return i.valid }

type authTestAuthenticator struct{}

func (authTestAuthenticator) ValidateToken(context.Context, string, TokenType, bool) (Identity, error) {
	return authTestIdentity{userID: "u", tokenID: "t", valid: true}, nil
}
func (authTestAuthenticator) CreateTokens(context.Context, string, map[string]interface{}) (*TokenPair, error) {
	return &TokenPair{AccessToken: "a"}, nil
}
func (authTestAuthenticator) RefreshTokens(context.Context, string, map[string]interface{}) (*TokenPair, error) {
	return &TokenPair{RefreshToken: "r"}, nil
}
func (authTestAuthenticator) RevokeToken(context.Context, string, string) error { return nil }
func (authTestAuthenticator) RevokeAllUserTokens(context.Context, string, string, string) (int, error) {
	return 1, nil
}
func (authTestAuthenticator) Close() error { return nil }

func newAuthScope() *authTestScope {
	return &authTestScope{
		ctx: context.Background(),
		cfg: &config.Config{
			Auth: config.NewDefaultAuthConfig(),
			Log:  config.NewDefaultLogConfig(),
		},
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

func withFactorySnapshot(t *testing.T) {
	t.Helper()
	mu.Lock()
	orig := make(map[string]Factory, len(factories))
	for k, v := range factories {
		orig[k] = v
	}
	factories = map[string]Factory{}
	mu.Unlock()
	t.Cleanup(func() {
		mu.Lock()
		factories = orig
		mu.Unlock()
	})
}

func withJWTIdentityCacheStoreDriverSnapshot(t *testing.T) {
	t.Helper()
	jwtIdentityCacheStoreDriversMu.Lock()
	orig := make(map[string]JWTIdentityCacheStoreFactory, len(jwtIdentityCacheStoreDrivers))
	for k, v := range jwtIdentityCacheStoreDrivers {
		orig[k] = v
	}
	jwtIdentityCacheStoreDrivers = map[string]JWTIdentityCacheStoreFactory{}
	jwtIdentityCacheStoreDriversMu.Unlock()
	t.Cleanup(func() {
		jwtIdentityCacheStoreDriversMu.Lock()
		jwtIdentityCacheStoreDrivers = orig
		jwtIdentityCacheStoreDriversMu.Unlock()
	})
}

func TestContextHelpersAndAuthenticationState(t *testing.T) {
	base := context.Background()
	validIdentity := authTestIdentity{userID: "user-1", tokenID: "token-1", metadata: map[string]interface{}{"role": "admin"}, valid: true}
	invalidIdentity := authTestIdentity{userID: "user-2", tokenID: "token-2", valid: false}

	if got := IdentityFromContext(base); got != nil {
		t.Fatalf("identity from empty context = %#v, want nil", got)
	}
	if IsAuthenticated(base) {
		t.Fatal("expected empty context to be unauthenticated")
	}

	ctx := ContextWithIdentity(base, validIdentity)
	ctx = ContextWithAccessToken(ctx, "access-token")
	ctx = ContextWithInternalKey(ctx, "internal-key")

	if got := IdentityFromContext(ctx); got == nil || got.GetUserID() != "user-1" {
		t.Fatalf("identity from context = %#v, want user-1", got)
	}
	if got, ok := AccessTokenFromContext(ctx); !ok || got != "access-token" {
		t.Fatalf("access token = %q ok=%v, want access-token", got, ok)
	}
	if got, ok := InternalKeyFromContext(ctx); !ok || got != "internal-key" {
		t.Fatalf("internal key = %q ok=%v, want internal-key", got, ok)
	}
	if !IsAuthenticated(ctx) {
		t.Fatal("expected valid identity to authenticate context")
	}

	ctxInvalid := ContextWithIdentity(base, invalidIdentity)
	if IsAuthenticated(ctxInvalid) {
		t.Fatal("expected invalid identity to fail authentication")
	}

	if got := ContextWithAccessToken(base, ""); got != base {
		t.Fatal("expected empty access token to leave context unchanged")
	}
	if got := ContextWithInternalKey(base, ""); got != base {
		t.Fatal("expected empty internal key to leave context unchanged")
	}
	if _, ok := AccessTokenFromContext(base); ok {
		t.Fatal("expected no access token in empty context")
	}
	if _, ok := InternalKeyFromContext(base); ok {
		t.Fatal("expected no internal key in empty context")
	}
}

func TestFactoryRegistrationLookupAndAuthenticatorCreation(t *testing.T) {
	withFactorySnapshot(t)
	runtimeScope := newAuthScope()
	called := 0
	Register("test-auth", func(gotScope scope.Scope) (Authenticator, error) {
		called++
		if gotScope != runtimeScope {
			t.Fatal("factory received unexpected env")
		}
		return authTestAuthenticator{}, nil
	})

	if !Exists("test-auth") {
		t.Fatal("expected registered factory to exist")
	}
	keys := Keys()
	sort.Strings(keys)
	if !reflect.DeepEqual(keys, []string{"test-auth"}) {
		t.Fatalf("keys = %#v, want only test-auth", keys)
	}

	authenticator, err := NewByName("test-auth", runtimeScope)
	if err != nil {
		t.Fatalf("NewByName returned error: %v", err)
	}
	if _, ok := authenticator.(authTestAuthenticator); !ok {
		t.Fatalf("unexpected authenticator type: %#v", authenticator)
	}
	if called != 1 {
		t.Fatalf("factory calls = %d, want 1", called)
	}

	runtimeScope.cfg.Auth.Type = "test-auth"
	authenticator, err = NewAuthenticator(runtimeScope)
	if err != nil {
		t.Fatalf("NewAuthenticator returned error: %v", err)
	}
	if authenticator == nil {
		t.Fatal("expected authenticator from registered type")
	}
	if called != 2 {
		t.Fatalf("factory calls = %d, want 2", called)
	}
}

func TestNewByNameAndNewAuthenticatorHandleMissingOrDisabledConfig(t *testing.T) {
	withFactorySnapshot(t)
	runtimeScope := newAuthScope()

	if _, err := NewByName("missing", runtimeScope); err == nil {
		t.Fatal("expected unsupported auth type error")
	}

	runtimeScope.cfg.Auth.Enabled = false
	authenticator, err := NewAuthenticator(runtimeScope)
	if err != nil {
		t.Fatalf("disabled NewAuthenticator returned error: %v", err)
	}
	if authenticator != nil {
		t.Fatalf("disabled NewAuthenticator = %#v, want nil", authenticator)
	}

	runtimeScope.cfg.Auth = nil
	authenticator, err = NewAuthenticator(runtimeScope)
	if err != nil {
		t.Fatalf("nil-config NewAuthenticator returned error: %v", err)
	}
	if authenticator != nil {
		t.Fatalf("nil-config NewAuthenticator = %#v, want nil", authenticator)
	}
}

func TestNewAuthenticatorPropagatesFactoryErrors(t *testing.T) {
	withFactorySnapshot(t)
	runtimeScope := newAuthScope()
	runtimeScope.cfg.Auth.Type = "broken"
	Register("broken", func(scope.Scope) (Authenticator, error) {
		return nil, errors.New("factory failed")
	})

	if _, err := NewAuthenticator(runtimeScope); err == nil || err.Error() != "factory failed" {
		t.Fatalf("expected factory error, got %v", err)
	}
}

func TestJWTIdentityCacheStoreDriverRegistrationAndLookup(t *testing.T) {
	withJWTIdentityCacheStoreDriverSnapshot(t)
	runtimeScope := newAuthScope()
	factoryCalled := false

	RegisterJWTIdentityCacheStoreDriver("fake", func(scope.Scope) (cache.Cache, error) {
		factoryCalled = true
		return nil, errors.New("boom")
	})

	if _, err := NewJWTIdentityCacheStore("fake", runtimeScope); err == nil || err.Error() != "boom" {
		t.Fatalf("expected fake driver error, got %v", err)
	}
	if !factoryCalled {
		t.Fatal("expected fake driver factory to be called")
	}

	if _, err := NewJWTIdentityCacheStore("missing", runtimeScope); err == nil || !autherrors.IsAuthError(err, autherrors.ErrDriverNotRegistered) {
		t.Fatalf("expected missing driver auth error, got %v", err)
	}

	if _, err := NewJWTIdentityCacheStore(" ", runtimeScope); err == nil || !autherrors.IsAuthError(err, autherrors.ErrDriverNotRegistered) {
		t.Fatalf("expected blank driver to fall back to default and fail without registration, got %v", err)
	}
}

func TestJWTIdentityCacheStoreDriverRegistrationPanics(t *testing.T) {
	withJWTIdentityCacheStoreDriverSnapshot(t)

	assertPanics := func(name string, fn func()) {
		t.Helper()
		defer func() {
			if recover() == nil {
				t.Fatalf("expected panic for %s", name)
			}
		}()
		fn()
	}

	assertPanics("empty name", func() {
		RegisterJWTIdentityCacheStoreDriver("", func(scope.Scope) (cache.Cache, error) { return nil, nil })
	})
	assertPanics("whitespace name", func() {
		RegisterJWTIdentityCacheStoreDriver("   ", func(scope.Scope) (cache.Cache, error) { return nil, nil })
	})
	assertPanics("nil factory", func() {
		RegisterJWTIdentityCacheStoreDriver("nil", nil)
	})
	assertPanics("duplicate driver", func() {
		RegisterJWTIdentityCacheStoreDriver("dup", func(scope.Scope) (cache.Cache, error) { return nil, nil })
		RegisterJWTIdentityCacheStoreDriver("dup", func(scope.Scope) (cache.Cache, error) { return nil, nil })
	})
}

func TestNewJWTIdentityCacheStoreFallsBackToDefaultMemoryName(t *testing.T) {
	withJWTIdentityCacheStoreDriverSnapshot(t)
	runtimeScope := newAuthScope()
	called := false

	RegisterJWTIdentityCacheStoreDriver("memory", func(scope.Scope) (cache.Cache, error) {
		called = true
		return nil, errors.New("memory-boom")
	})

	if _, err := NewJWTIdentityCacheStore(strings.Repeat(" ", 2), runtimeScope); err == nil || err.Error() != "memory-boom" {
		t.Fatalf("expected default memory driver error, got %v", err)
	}
	if !called {
		t.Fatal("expected memory driver factory to be called")
	}
}
