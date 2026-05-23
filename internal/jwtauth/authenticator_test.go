// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package jwtauth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/auth"
	"github.com/choysum-dev/choysum/pkg/auth/autherrors"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/golang-jwt/jwt/v5"
)

type testContextKey string

type fakeScope struct {
	ctx    context.Context
	cfg    *config.Config
	logger *slog.Logger
}

func (f *fakeScope) Run(fn func(scope.Scope) error) error { return fn(f) }
func (f *fakeScope) Transactor() scope.Transactor {
	return scopetest.NewPassthroughTransactor(f)
}
func (f *fakeScope) Session() *scope.Session { return nil }
func (f *fakeScope) WithContext(ctx context.Context) scope.Scope {
	return &fakeScope{ctx: ctx, cfg: f.cfg, logger: f.logger}
}
func (f *fakeScope) Context() context.Context { return f.ctx }
func (f *fakeScope) Logger() *slog.Logger     { return f.logger }
func (f *fakeScope) Config() *config.Config   { return f.cfg }
func (f *fakeScope) FactoryInput() scope.FactoryInput {
	return scopetest.FactoryInputFromConfig(f.cfg)
}

type fakeKeyProvider struct {
	privateKey    *rsa.PrivateKey
	publicKey     *rsa.PublicKey
	privateErr    error
	publicErr     error
	closeErr      error
	closeCallOnce sync.Once
}

func (f *fakeKeyProvider) GetPrivateKey() (*rsa.PrivateKey, error) {
	if f.privateErr != nil {
		return nil, f.privateErr
	}
	return f.privateKey, nil
}

func (f *fakeKeyProvider) GetPublicKey() (*rsa.PublicKey, error) {
	if f.publicErr != nil {
		return nil, f.publicErr
	}
	return f.publicKey, nil
}

func (f *fakeKeyProvider) Close() error { return f.closeErr }

type fakeRevokeStore struct {
	mu sync.Mutex

	isRevoked      bool
	isRevokedErr   error
	isRevokedCalls int

	revokeErr    error
	revokeAllErr error
	revokeAllN   int
	closeErr     error

	lastCtx       context.Context
	lastTokenID   string
	lastUserID    string
	lastTokenType auth.TokenType
	lastReason    string
	lastExceptID  string
}

func (f *fakeRevokeStore) IsRevoked(ctx context.Context, tokenID string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.isRevokedCalls++
	f.lastCtx = ctx
	f.lastTokenID = tokenID
	if f.isRevokedErr != nil {
		return false, f.isRevokedErr
	}
	return f.isRevoked, nil
}

func (f *fakeRevokeStore) RevokeToken(ctx context.Context, tokenID string, userID string, tokenType auth.TokenType, _ time.Time, reason string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.lastCtx = ctx
	f.lastTokenID = tokenID
	f.lastUserID = userID
	f.lastTokenType = tokenType
	f.lastReason = reason
	return f.revokeErr
}

func (f *fakeRevokeStore) RevokeAllUserTokens(ctx context.Context, userID string, exceptTokenID string, reason string) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.lastCtx = ctx
	f.lastUserID = userID
	f.lastExceptID = exceptTokenID
	f.lastReason = reason
	return f.revokeAllN, f.revokeAllErr
}

func (f *fakeRevokeStore) CleanExpired(context.Context) (int, error) { return 0, nil }
func (f *fakeRevokeStore) Close() error                              { return f.closeErr }

var (
	testKeysOnce sync.Once
	testPrivKey  *rsa.PrivateKey
	testKeyErr   error
)

func loadTestKeys(t *testing.T) (*rsa.PrivateKey, *rsa.PublicKey) {
	t.Helper()
	testKeysOnce.Do(func() {
		testPrivKey, testKeyErr = rsa.GenerateKey(rand.Reader, 1024)
	})
	if testKeyErr != nil {
		t.Fatalf("generate rsa key: %v", testKeyErr)
	}
	return testPrivKey, &testPrivKey.PublicKey
}

func newFakeScope() *fakeScope {
	cfg := &config.Config{
		Auth: config.NewDefaultAuthConfig(),
		Log:  config.NewDefaultLogConfig(),
	}
	cfg.Auth.JWT.RevokeStore = "memory"
	ctx := context.WithValue(context.Background(), testContextKey("source"), "env")
	return &fakeScope{
		ctx:    ctx,
		cfg:    cfg,
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

func newManualAuthenticator(t *testing.T, cacheEnabled bool) (*JwtAuthenticator, *fakeRevokeStore, *fakeScope) {
	t.Helper()

	privateKey, publicKey := loadTestKeys(t)
	runtimeScope := newFakeScope()
	runtimeScope.cfg.Auth.JWT.IdentityCache.Enabled = cacheEnabled
	runtimeScope.cfg.Auth.JWT.IdentityCache.TTL = 10 * time.Minute
	store := &fakeRevokeStore{}
	authenticator := &JwtAuthenticator{
		runtimeScope:  runtimeScope,
		keyProvider:   &fakeKeyProvider{privateKey: privateKey, publicKey: publicKey},
		revokeStore:   store,
		accessExpiry:  20 * time.Minute,
		refreshExpiry: 2 * time.Hour,
		cacheEnabled:  cacheEnabled,
	}
	if cacheEnabled {
		runtimeScope.cfg.Auth.JWT.IdentityCache.Size = 32
		cache, err := newIdentityCache(runtimeScope)
		if err != nil {
			t.Fatalf("create identity cache: %v", err)
		}
		authenticator.cache = cache
	}
	return authenticator, store, runtimeScope
}

func nilContext() context.Context {
	var ctx context.Context
	return ctx
}

func setTestJWTKeyFiles(t *testing.T, cfg *config.JWTConfig) {
	t.Helper()
	keyDir := t.TempDir()
	cfg.PrivateKeyFile = filepath.Join(keyDir, "private.pem")
	cfg.PublicKeyFile = filepath.Join(keyDir, "public.pem")
}

func TestNewJWTAuthenticatorRequiresConfig(t *testing.T) {
	runtimeScope := newFakeScope()
	runtimeScope.cfg.Auth.JWT = nil

	_, err := NewJWTAuthenticator(runtimeScope)
	if err == nil {
		t.Fatal("expected missing jwt config error")
	}
}

func TestNewJWTAuthenticatorInitializesKeysCacheAndRevocationStore(t *testing.T) {
	runtimeScope := newFakeScope()
	setTestJWTKeyFiles(t, runtimeScope.cfg.Auth.JWT)
	runtimeScope.cfg.Auth.JWT.RevokeStore = "memory"
	runtimeScope.cfg.Auth.JWT.IdentityCache.Enabled = true

	authenticator, err := NewJWTAuthenticator(runtimeScope)
	if err != nil {
		t.Fatalf("NewJWTAuthenticator returned error: %v", err)
	}
	t.Cleanup(func() {
		_ = authenticator.Close()
	})

	if authenticator.cache == nil {
		t.Fatal("expected cache to be initialized")
	}
	if _, err := authenticator.keyProvider.GetPrivateKey(); err != nil {
		t.Fatalf("private key not initialized: %v", err)
	}
}

func TestNewJWTAuthenticatorWrapsInitializationFailures(t *testing.T) {
	t.Run("wraps key provider failures", func(t *testing.T) {
		runtimeScope := newFakeScope()
		blockedPath := filepath.Join(t.TempDir(), "blocked")
		if err := os.WriteFile(blockedPath, []byte("x"), 0o600); err != nil {
			t.Fatalf("write blocker file: %v", err)
		}
		runtimeScope.cfg.Auth.JWT.PrivateKeyFile = filepath.Join(blockedPath, "private.pem")
		runtimeScope.cfg.Auth.JWT.PublicKeyFile = filepath.Join(blockedPath, "public.pem")

		_, err := NewJWTAuthenticator(runtimeScope)
		if err == nil || !autherrors.IsAuthError(err, autherrors.ErrKeyProviderInitFailed) {
			t.Fatalf("expected key provider init failure, got %v", err)
		}
	})

	t.Run("wraps revocation store failures", func(t *testing.T) {
		runtimeScope := newFakeScope()
		setTestJWTKeyFiles(t, runtimeScope.cfg.Auth.JWT)
		runtimeScope.cfg.Auth.JWT.RevokeStore = "missing-driver"

		_, err := NewJWTAuthenticator(runtimeScope)
		if err == nil || !autherrors.IsAuthError(err, autherrors.ErrRevocationStoreFailed) {
			t.Fatalf("expected revocation store init failure, got %v", err)
		}
	})

	t.Run("wraps cache init failures", func(t *testing.T) {
		runtimeScope := newFakeScope()
		setTestJWTKeyFiles(t, runtimeScope.cfg.Auth.JWT)
		runtimeScope.cfg.Auth.JWT.RevokeStore = "memory"
		runtimeScope.cfg.Auth.JWT.IdentityCache.Enabled = true
		runtimeScope.cfg.Auth.JWT.IdentityCache.Size = 0

		_, err := NewJWTAuthenticator(runtimeScope)
		if err == nil || !autherrors.IsAuthError(err, autherrors.ErrCacheInitFailed) {
			t.Fatalf("expected cache init failure, got %v", err)
		}
	})

	t.Run("wraps missing identity cache backend driver", func(t *testing.T) {
		runtimeScope := newFakeScope()
		setTestJWTKeyFiles(t, runtimeScope.cfg.Auth.JWT)
		runtimeScope.cfg.Auth.JWT.RevokeStore = "memory"
		runtimeScope.cfg.Auth.JWT.IdentityCache.Enabled = true
		runtimeScope.cfg.Auth.JWT.IdentityCache.Backend = "missing-driver"

		_, err := NewJWTAuthenticator(runtimeScope)
		if err == nil || !autherrors.IsAuthError(err, autherrors.ErrCacheInitFailed) {
			t.Fatalf("expected cache init failure for missing driver, got %v", err)
		}
	})
}

func TestCreateTokensValidateTokenAndCache(t *testing.T) {
	authenticator, store, _ := newManualAuthenticator(t, true)

	pair, err := authenticator.CreateTokens(context.Background(), "user-1", map[string]interface{}{"role": "admin"})
	if err != nil {
		t.Fatalf("CreateTokens returned error: %v", err)
	}
	if pair.AccessToken == "" || pair.RefreshToken == "" {
		t.Fatal("expected access and refresh tokens to be populated")
	}

	identity, err := authenticator.ValidateToken(context.Background(), pair.AccessToken, auth.AccessToken, false)
	if err != nil {
		t.Fatalf("ValidateToken returned error: %v", err)
	}
	if got := identity.GetUserID(); got != "user-1" {
		t.Fatalf("user id = %q, want %q", got, "user-1")
	}
	if got := identity.GetMetadata()["role"]; got != "admin" {
		t.Fatalf("metadata role = %#v, want %q", got, "admin")
	}
	if store.isRevokedCalls != 0 {
		t.Fatalf("unexpected revoke checks: %d", store.isRevokedCalls)
	}

	secondIdentity, err := authenticator.ValidateToken(context.Background(), pair.AccessToken, auth.AccessToken, false)
	if err != nil {
		t.Fatalf("second ValidateToken returned error: %v", err)
	}
	if secondIdentity.GetTokenID() != identity.GetTokenID() {
		t.Fatalf("cached token id = %q, want %q", secondIdentity.GetTokenID(), identity.GetTokenID())
	}
	if store.isRevokedCalls != 0 {
		t.Fatalf("cache miss caused unexpected revoke checks: %d", store.isRevokedCalls)
	}

	claims, err := FromToken(pair.RefreshToken, func(*jwt.Token) (interface{}, error) {
		return authenticator.keyProvider.GetPublicKey()
	})
	if err != nil {
		t.Fatalf("FromToken(refresh) returned error: %v", err)
	}
	if claims.Metadata != nil {
		t.Fatalf("refresh token metadata = %#v, want nil", claims.Metadata)
	}
}

func TestValidateTokenUsesRuntimeScopeContextAndReportsTypeOrRevocationErrors(t *testing.T) {
	authenticator, store, runtimeScope := newManualAuthenticator(t, true)
	pair, err := authenticator.CreateTokens(context.Background(), "user-2", nil)
	if err != nil {
		t.Fatalf("CreateTokens returned error: %v", err)
	}

	if _, err := authenticator.ValidateToken(nilContext(), pair.AccessToken, auth.RefreshToken, false); err == nil || !strings.Contains(err.Error(), "token type mismatch") {
		t.Fatalf("expected token type mismatch, got %v", err)
	}

	if _, err := authenticator.ValidateToken(nilContext(), pair.RefreshToken, auth.RefreshToken, true); err != nil {
		t.Fatalf("ValidateToken(refresh) returned error: %v", err)
	}
	if store.lastCtx != runtimeScope.ctx {
		t.Fatal("expected nil ctx to fall back to scope context")
	}

	store.isRevoked = true
	if _, err := authenticator.ValidateToken(context.Background(), pair.RefreshToken, auth.RefreshToken, true); err == nil || !strings.Contains(err.Error(), "token has been revoked") {
		t.Fatalf("expected revoked token error, got %v", err)
	}
}

func TestValidateTokenHandlesAdditionalErrorBranches(t *testing.T) {
	t.Run("rejects empty tokens", func(t *testing.T) {
		authenticator, _, _ := newManualAuthenticator(t, true)

		if _, err := authenticator.ValidateToken(context.Background(), "", auth.AccessToken, false); err == nil || !autherrors.IsAuthError(err, autherrors.ErrInvalidTokenID) {
			t.Fatalf("expected empty token id error, got %v", err)
		}
	})

	t.Run("returns parse errors from public key lookup", func(t *testing.T) {
		authenticator, _, _ := newManualAuthenticator(t, true)
		pair, err := authenticator.CreateTokens(context.Background(), "user-public", nil)
		if err != nil {
			t.Fatalf("CreateTokens() error = %v", err)
		}
		authenticator.keyProvider = &fakeKeyProvider{publicErr: errors.New("public key boom")}

		if _, err := authenticator.ValidateToken(context.Background(), pair.AccessToken, auth.AccessToken, false); err == nil || !strings.Contains(err.Error(), "public key boom") {
			t.Fatalf("expected public key failure, got %v", err)
		}
	})

	t.Run("continues when revocation lookup fails", func(t *testing.T) {
		authenticator, store, _ := newManualAuthenticator(t, true)
		store.isRevokedErr = errors.New("revocation lookup boom")
		pair, err := authenticator.CreateTokens(context.Background(), "user-revoke", nil)
		if err != nil {
			t.Fatalf("CreateTokens() error = %v", err)
		}

		identity, err := authenticator.ValidateToken(context.Background(), pair.AccessToken, auth.AccessToken, true)
		if err != nil {
			t.Fatalf("ValidateToken() error = %v", err)
		}
		if identity.GetUserID() != "user-revoke" {
			t.Fatalf("user id = %q, want user-revoke", identity.GetUserID())
		}
		if store.isRevokedCalls != 1 {
			t.Fatalf("revocation checks = %d, want 1", store.isRevokedCalls)
		}
		if authenticator.getFromCache(pair.AccessToken) == nil {
			t.Fatal("expected successful validation to populate access-token cache")
		}
	})
}

func TestCreateAccessTokenWithTTLAndCreateTokensValidateInputs(t *testing.T) {
	authenticator, _, _ := newManualAuthenticator(t, false)

	if _, err := authenticator.CreateTokens(context.Background(), "", nil); err == nil {
		t.Fatal("expected empty user id error for CreateTokens")
	}
	if _, _, err := authenticator.CreateAccessTokenWithTTL(context.Background(), "", nil, time.Minute); err == nil {
		t.Fatal("expected empty user id error for CreateAccessTokenWithTTL")
	}

	accessToken, expiresAt, err := authenticator.CreateAccessTokenWithTTL(context.Background(), "user-3", map[string]interface{}{"scope": "read"}, 2*time.Minute)
	if err != nil {
		t.Fatalf("CreateAccessTokenWithTTL returned error: %v", err)
	}
	if accessToken == "" || expiresAt == 0 {
		t.Fatal("expected access token and expiry")
	}

	claims, err := FromToken(accessToken, func(*jwt.Token) (interface{}, error) {
		return authenticator.keyProvider.GetPublicKey()
	})
	if err != nil {
		t.Fatalf("FromToken(access) returned error: %v", err)
	}
	if claims.Subject != "user-3" || claims.Metadata["scope"] != "read" {
		t.Fatalf("unexpected claims: subject=%q metadata=%#v", claims.Subject, claims.Metadata)
	}
	remaining := time.Until(time.UnixMilli(expiresAt))
	if remaining < time.Minute || remaining > 3*time.Minute {
		t.Fatalf("ttl remaining = %v, want around 2m", remaining)
	}

	t.Run("falls back to configured access expiry when ttl is non-positive", func(t *testing.T) {
		authenticator, _, _ := newManualAuthenticator(t, false)
		authenticator.accessExpiry = 5 * time.Minute

		accessToken, expiresAt, err := authenticator.CreateAccessTokenWithTTL(nilContext(), "user-ttl", nil, 0)
		if err != nil {
			t.Fatalf("CreateAccessTokenWithTTL() error = %v", err)
		}
		if accessToken == "" || expiresAt == 0 {
			t.Fatal("expected fallback token and expiry")
		}
		remaining := time.Until(time.UnixMilli(expiresAt))
		if remaining < 4*time.Minute || remaining > 6*time.Minute {
			t.Fatalf("fallback ttl remaining = %v, want around 5m", remaining)
		}
	})

	t.Run("falls back to 15 minutes when configured access expiry is non-positive", func(t *testing.T) {
		authenticator, _, _ := newManualAuthenticator(t, false)
		authenticator.accessExpiry = 0

		_, expiresAt, err := authenticator.CreateAccessTokenWithTTL(context.Background(), "user-default", nil, -time.Minute)
		if err != nil {
			t.Fatalf("CreateAccessTokenWithTTL() error = %v", err)
		}
		remaining := time.Until(time.UnixMilli(expiresAt))
		if remaining < 14*time.Minute || remaining > 16*time.Minute {
			t.Fatalf("default ttl remaining = %v, want around 15m", remaining)
		}
	})

	t.Run("wraps private key lookup failures", func(t *testing.T) {
		authenticator, _, _ := newManualAuthenticator(t, false)
		authenticator.keyProvider = &fakeKeyProvider{privateErr: errors.New("private key boom")}

		_, _, err := authenticator.CreateAccessTokenWithTTL(context.Background(), "user-keyerr", nil, time.Minute)
		if err == nil || !autherrors.IsAuthError(err, autherrors.ErrTokenSigningFailed) {
			t.Fatalf("expected token signing error, got %v", err)
		}
	})
}

func TestCreateTokensUsesDefaultExpiryAndWrapsSigningErrors(t *testing.T) {
	t.Run("falls back to default expiries", func(t *testing.T) {
		authenticator, _, _ := newManualAuthenticator(t, false)
		authenticator.accessExpiry = 0
		authenticator.refreshExpiry = 0

		pair, err := authenticator.CreateTokens(context.Background(), "user-default-expiry", nil)
		if err != nil {
			t.Fatalf("CreateTokens() error = %v", err)
		}

		accessRemaining := time.Until(time.UnixMilli(pair.ExpiresAt))
		if accessRemaining < 14*time.Minute || accessRemaining > 16*time.Minute {
			t.Fatalf("access ttl remaining = %v, want around 15m", accessRemaining)
		}

		refreshRemaining := time.Until(time.UnixMilli(pair.RefreshExpiresAt))
		if refreshRemaining < (7*24*time.Hour-2*time.Hour) || refreshRemaining > (7*24*time.Hour+2*time.Hour) {
			t.Fatalf("refresh ttl remaining = %v, want around 7d", refreshRemaining)
		}
	})

	t.Run("wraps private key lookup failures", func(t *testing.T) {
		authenticator, _, _ := newManualAuthenticator(t, false)
		authenticator.keyProvider = &fakeKeyProvider{privateErr: errors.New("private key boom")}

		_, err := authenticator.CreateTokens(context.Background(), "user-sign-fail", nil)
		if err == nil || !autherrors.IsAuthError(err, autherrors.ErrTokenSigningFailed) {
			t.Fatalf("expected token signing error, got %v", err)
		}
	})
}

func TestRevokeTokenRemovesCacheAndUsesDefaultReason(t *testing.T) {
	authenticator, store, runtimeScope := newManualAuthenticator(t, true)
	pair, err := authenticator.CreateTokens(context.Background(), "user-4", nil)
	if err != nil {
		t.Fatalf("CreateTokens returned error: %v", err)
	}
	identity, err := authenticator.ValidateToken(context.Background(), pair.AccessToken, auth.AccessToken, false)
	if err != nil {
		t.Fatalf("ValidateToken returned error: %v", err)
	}
	if cached := authenticator.getFromCache(pair.AccessToken); cached == nil || cached.GetTokenID() != identity.GetTokenID() {
		t.Fatal("expected access token to be cached before revoke")
	}

	if err := authenticator.RevokeToken(nilContext(), pair.AccessToken, ""); err != nil {
		t.Fatalf("RevokeToken returned error: %v", err)
	}
	if authenticator.getFromCache(pair.AccessToken) != nil {
		t.Fatal("expected revoked token to be removed from cache")
	}
	if store.lastCtx != runtimeScope.ctx {
		t.Fatal("expected nil ctx to use scope context")
	}
	if store.lastReason != "user request" {
		t.Fatalf("revoke reason = %q, want default", store.lastReason)
	}
	if store.lastUserID != "user-4" || store.lastTokenType != auth.AccessToken {
		t.Fatalf("unexpected revoke args: user=%q type=%q", store.lastUserID, store.lastTokenType)
	}

	if err := authenticator.RevokeToken(context.Background(), "", "ignored"); err == nil {
		t.Fatal("expected empty token error")
	}
	if err := authenticator.RevokeToken(context.Background(), "not-a-jwt", "ignored"); err == nil {
		t.Fatal("expected invalid token parse error")
	}
}

func TestRevokeAllUserTokensClearsUserCacheAndUsesDefaultReason(t *testing.T) {
	authenticator, store, runtimeScope := newManualAuthenticator(t, true)
	store.revokeAllN = 2
	now := time.Now().Add(time.Hour)
	authenticator.addToCache("token-a", NewIdentity("user-5", "token-a", nil, now, time.Now(), auth.AccessToken, "choysum"))
	authenticator.addToCache("token-b", NewIdentity("user-5", "token-b", nil, now, time.Now(), auth.AccessToken, "choysum"))
	authenticator.addToCache("token-keep", NewIdentity("user-5", "token-keep", nil, now, time.Now(), auth.AccessToken, "choysum"))
	authenticator.addToCache("token-other", NewIdentity("user-6", "token-other", nil, now, time.Now(), auth.AccessToken, "choysum"))

	n, err := authenticator.RevokeAllUserTokens(nilContext(), "user-5", "token-keep", "")
	if err != nil {
		t.Fatalf("RevokeAllUserTokens returned error: %v", err)
	}
	if n != 2 {
		t.Fatalf("revoke count = %d, want 2", n)
	}
	if store.lastCtx != runtimeScope.ctx || store.lastReason != "bulk revocation" {
		t.Fatalf("unexpected revoke-all args: ctx=%v reason=%q", store.lastCtx, store.lastReason)
	}
	if authenticator.getFromCache("token-a") != nil || authenticator.getFromCache("token-b") != nil {
		t.Fatal("expected revoked user cache entries to be removed")
	}
	if authenticator.getFromCache("token-keep") == nil {
		t.Fatal("expected except token cache entry to remain")
	}
	if authenticator.getFromCache("token-other") == nil {
		t.Fatal("expected other user cache entry to remain")
	}

	if _, err := authenticator.RevokeAllUserTokens(context.Background(), "", "", ""); err == nil {
		t.Fatal("expected empty user id error")
	}
}

func TestRefreshTokensUsesRuntimeScopeContextAndContinuesAfterRevokeFailure(t *testing.T) {
	authenticator, store, runtimeScope := newManualAuthenticator(t, true)
	store.revokeErr = errors.New("revoke failed")
	pair, err := authenticator.CreateTokens(context.Background(), "user-refresh", map[string]interface{}{"scope": "old"})
	if err != nil {
		t.Fatalf("CreateTokens() error = %v", err)
	}

	refreshed, err := authenticator.RefreshTokens(nilContext(), pair.RefreshToken, map[string]interface{}{"scope": "new"})
	if err != nil {
		t.Fatalf("RefreshTokens() error = %v", err)
	}
	if refreshed.AccessToken == "" || refreshed.RefreshToken == "" {
		t.Fatal("expected refreshed token pair")
	}
	if store.lastCtx != runtimeScope.ctx {
		t.Fatal("expected nil context to fall back to scope context")
	}
}

func TestCloseAggregatesErrors(t *testing.T) {
	authenticator, _, _ := newManualAuthenticator(t, false)
	authenticator.keyProvider = &fakeKeyProvider{closeErr: errors.New("key close")}
	authenticator.revokeStore = &fakeRevokeStore{closeErr: errors.New("store close")}

	err := authenticator.Close()
	if err == nil {
		t.Fatal("expected close error")
	}
	if !strings.Contains(err.Error(), "key close") || !strings.Contains(err.Error(), "store close") {
		t.Fatalf("unexpected close error: %v", err)
	}
}

func TestGetFromCacheExpiresEntries(t *testing.T) {
	authenticator, _, _ := newManualAuthenticator(t, true)
	if err := authenticator.cache.store.Set(context.Background(), jwtAccessIdentityNamespace, "expired", mustEncodeCachedIdentity(t, &cachedIdentity{
		identity:  NewIdentity("user-7", "expired", nil, time.Now().Add(time.Hour), time.Now(), auth.AccessToken, "choysum"),
		expiry:    time.Now().Add(-time.Second),
		cacheTime: time.Now().Add(-2 * time.Second),
	}), time.Minute); err != nil {
		t.Fatalf("seed expired cache entry: %v", err)
	}

	if identity := authenticator.getFromCache("expired"); identity != nil {
		t.Fatalf("expected expired cache entry to be evicted, got %#v", identity)
	}
}

func TestCacheHelpersRespectDefaultTTLAndDisabledState(t *testing.T) {
	t.Run("disabled cache helpers are no-op", func(t *testing.T) {
		authenticator, _, _ := newManualAuthenticator(t, false)
		identity := NewIdentity("user-disabled", "token-disabled", nil, time.Now().Add(time.Minute), time.Now(), auth.AccessToken, "choysum")

		authenticator.addToCache("token-disabled", identity)
		if got := authenticator.getFromCache("token-disabled"); got != nil {
			t.Fatalf("expected disabled cache lookup to return nil, got %#v", got)
		}
		authenticator.removeFromCache("token-disabled")
		authenticator.clearUserCache("user-disabled", "")
	})

	t.Run("default ttl is clamped by token expiry", func(t *testing.T) {
		authenticator, _, runtimeScope := newManualAuthenticator(t, true)
		runtimeScope.cfg.Auth.JWT.IdentityCache.TTL = 0
		expiresAt := time.Now().Add(2 * time.Minute)
		identity := NewIdentity("user-cache", "token-cache", nil, expiresAt, time.Now(), auth.AccessToken, "choysum")

		authenticator.addToCache("token-cache", identity)

		cached, found, err := authenticator.cache.peek("token-cache")
		if err != nil {
			t.Fatalf("peek token-cache: %v", err)
		}
		if !found {
			t.Fatal("expected cached entry to exist")
		}
		if cached.cacheTime.IsZero() {
			t.Fatal("expected cacheTime to be populated")
		}
		if cached.expiry.After(expiresAt.Add(100*time.Millisecond)) || cached.expiry.Before(expiresAt.Add(-100*time.Millisecond)) {
			t.Fatalf("cache expiry = %v, want close to token expiry %v", cached.expiry, expiresAt)
		}

		authenticator.removeFromCache("token-cache")
		if authenticator.getFromCache("token-cache") != nil {
			t.Fatal("expected removeFromCache to evict token")
		}
	})
}

func mustEncodeCachedIdentity(t *testing.T, cached *cachedIdentity) []byte {
	t.Helper()
	payload, err := encodeCachedIdentity(cached)
	if err != nil {
		t.Fatalf("encode cached identity: %v", err)
	}
	return payload
}

func TestCacheConcurrentAccess(t *testing.T) {
	authenticator, _, _ := newManualAuthenticator(t, true)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				token := strings.Join([]string{"worker", "token", string(rune('a' + worker))}, "-")
				identity := NewIdentity("user", token, nil, time.Now().Add(time.Hour), time.Now(), auth.AccessToken, "choysum")
				authenticator.addToCache(token, identity)
				_ = authenticator.getFromCache(token)
				authenticator.removeFromCache(token)
			}
		}(i)
	}
	wg.Wait()
}
