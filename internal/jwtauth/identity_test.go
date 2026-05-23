// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package jwtauth

import (
	"errors"
	"testing"
	"time"

	"github.com/choysum-dev/choysum/pkg/auth"
)

func TestIdentityAccessorsAndValidity(t *testing.T) {
	now := time.Now()
	identity := NewIdentity(
		"user-1",
		"token-1",
		map[string]interface{}{"role": "admin"},
		now.Add(time.Minute),
		now.Add(-time.Minute),
		auth.AccessToken,
		"choysum-test",
	)

	if identity.GetUserID() != "user-1" || identity.GetTokenID() != "token-1" {
		t.Fatalf("unexpected identity ids: user=%q token=%q", identity.GetUserID(), identity.GetTokenID())
	}
	if identity.GetMetadata()["role"] != "admin" {
		t.Fatalf("unexpected identity metadata: %#v", identity.GetMetadata())
	}
	if identity.GetTokenType() != auth.AccessToken || identity.GetIssuer() != "choysum-test" {
		t.Fatalf("unexpected token type/issuer: type=%q issuer=%q", identity.GetTokenType(), identity.GetIssuer())
	}
	if identity.GetExpiresAt().Before(now) || identity.GetIssuedAt().After(now) {
		t.Fatalf("unexpected identity timestamps: expires=%v issued=%v now=%v", identity.GetExpiresAt(), identity.GetIssuedAt(), now)
	}
	if !identity.IsValid() {
		t.Fatal("expected future identity to be valid")
	}
	if (*Identity)(nil).IsValid() {
		t.Fatal("expected nil identity to be invalid")
	}
	if NewIdentity("", "token", nil, now.Add(time.Minute), now, auth.AccessToken, "choysum").IsValid() {
		t.Fatal("expected identity without user id to be invalid")
	}
	if NewIdentity("user", "", nil, now.Add(time.Minute), now, auth.AccessToken, "choysum").IsValid() {
		t.Fatal("expected identity without token id to be invalid")
	}
	if NewIdentity("user", "token", nil, now.Add(-time.Second), now, auth.AccessToken, "choysum").IsValid() {
		t.Fatal("expected expired identity to be invalid")
	}
}

func TestRefreshTokensUsesRuntimeScopeContextAndReturnsNewPair(t *testing.T) {
	authenticator, store, runtimeScope := newManualAuthenticator(t, true)
	store.revokeErr = errors.New("revoke failed")

	pair, err := authenticator.CreateTokens(nilContext(), "user-refresh", map[string]interface{}{"scope": "old"})
	if err != nil {
		t.Fatalf("CreateTokens() error = %v", err)
	}

	refreshed, err := authenticator.RefreshTokens(nilContext(), pair.RefreshToken, map[string]interface{}{"scope": "new"})
	if err != nil {
		t.Fatalf("RefreshTokens() error = %v", err)
	}
	if refreshed.AccessToken == "" || refreshed.RefreshToken == "" {
		t.Fatal("expected refreshed token pair to be populated")
	}
	if refreshed.AccessToken == pair.AccessToken || refreshed.RefreshToken == pair.RefreshToken {
		t.Fatal("expected refreshed tokens to differ from original pair")
	}
	if store.lastCtx != runtimeScope.ctx {
		t.Fatal("expected nil ctx to fall back to scope context")
	}
	if store.lastReason != "refreshed into a new token" || store.lastUserID != "user-refresh" || store.lastTokenType != auth.RefreshToken {
		t.Fatalf("unexpected revoke args: user=%q type=%q reason=%q", store.lastUserID, store.lastTokenType, store.lastReason)
	}

	identity, err := authenticator.ValidateToken(nilContext(), refreshed.AccessToken, auth.AccessToken, false)
	if err != nil {
		t.Fatalf("ValidateToken(refreshed access) error = %v", err)
	}
	if identity.GetUserID() != "user-refresh" || identity.GetMetadata()["scope"] != "new" {
		t.Fatalf("unexpected refreshed identity: user=%q metadata=%#v", identity.GetUserID(), identity.GetMetadata())
	}
	refreshIdentity, err := authenticator.ValidateToken(nilContext(), refreshed.RefreshToken, auth.RefreshToken, false)
	if err != nil {
		t.Fatalf("ValidateToken(refreshed refresh) error = %v", err)
	}
	if len(refreshIdentity.GetMetadata()) != 0 {
		t.Fatalf("expected refresh token metadata to stay empty, got %#v", refreshIdentity.GetMetadata())
	}
}

func TestFileKeyProviderAndFactoryHelpers(t *testing.T) {
	provider := &FileKeyProvider{}
	if _, err := provider.GetPublicKey(); err == nil {
		t.Fatal("expected GetPublicKey to fail before initialization")
	}

	runtimeScope := newFakeScope()
	setTestJWTKeyFiles(t, runtimeScope.cfg.Auth.JWT)
	runtimeScope.cfg.Auth.JWT.RevokeStore = "memory"
	runtimeScope.cfg.Auth.JWT.IdentityCache.Enabled = false

	created, err := createJWTAuthenticator(runtimeScope)
	if err != nil {
		t.Fatalf("createJWTAuthenticator() error = %v", err)
	}
	typed, ok := created.(*JwtAuthenticator)
	if !ok {
		t.Fatalf("expected *JwtAuthenticator, got %T", created)
	}
	t.Cleanup(func() {
		_ = typed.Close()
	})

	if _, err := typed.keyProvider.GetPublicKey(); err != nil {
		t.Fatalf("expected initialized public key provider, got %v", err)
	}
	if _, err := typed.keyProvider.GetPrivateKey(); err != nil {
		t.Fatalf("expected initialized private key provider, got %v", err)
	}
}
