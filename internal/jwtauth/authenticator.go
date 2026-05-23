// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package jwtauth

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/choysum-dev/choysum/internal/jwtauth/revocation"
	"github.com/choysum-dev/choysum/pkg/auth"
	"github.com/choysum-dev/choysum/pkg/auth/autherrors"
	"github.com/choysum-dev/choysum/pkg/scope"
)

// JwtAuthenticator implements JWT-based authentication.
type JwtAuthenticator struct {
	runtimeScope  scope.Scope
	keyProvider   KeyProvider
	revokeStore   revocation.Store
	cache         *identityCache
	accessExpiry  time.Duration
	refreshExpiry time.Duration
	cacheEnabled  bool
}

// cachedIdentity stores identity data in the local cache.
type cachedIdentity struct {
	identity  *Identity
	expiry    time.Time
	cacheTime time.Time
}

// NewJWTAuthenticator creates a JWT authenticator.
func NewJWTAuthenticator(runtimeScope scope.Scope) (*JwtAuthenticator, error) {
	cfg := runtimeOptionsFromScope(runtimeScope).authJWT
	if cfg == nil {
		return nil, autherrors.NewAuthError(autherrors.ErrJWTConfigurationMissing, "missing JWT configuration")
	}

	// Create the key provider.
	keyProvider, err := NewFileKeyProvider(cfg)
	if err != nil {
		return nil, autherrors.WrapAuthError(err, autherrors.ErrKeyProviderInitFailed, "failed to initialize key provider")
	}

	// Create the revocation store.
	revokeStore, err := revocation.NewStore(runtimeScope)
	if err != nil {
		return nil, autherrors.WrapAuthError(err, autherrors.ErrRevocationStoreFailed, "failed to initialize revocation store")
	}

	authenticator := &JwtAuthenticator{
		runtimeScope:  runtimeScope,
		keyProvider:   keyProvider,
		revokeStore:   revokeStore,
		accessExpiry:  cfg.AccessTokenExpiry,
		refreshExpiry: cfg.RefreshTokenExpiry,
		cacheEnabled:  cfg.IdentityCache != nil && cfg.IdentityCache.Enabled,
	}

	// Initialize the identity cache.
	if cfg.IdentityCache != nil && cfg.IdentityCache.Enabled {
		cache, err := newIdentityCache(runtimeScope)
		if err != nil {
			return nil, autherrors.WrapAuthError(err, autherrors.ErrCacheInitFailed, "failed to initialize identity cache")
		}
		authenticator.cache = cache
	}

	return authenticator, nil
}

// ValidateToken validates a token and returns the associated identity.
//
// token: token to validate
// tokenType: token type (access or refresh)
// checkRevoked: whether to check revocation state in the backing store
func (a *JwtAuthenticator) ValidateToken(ctx context.Context, token string, tokenType auth.TokenType, checkRevoked bool) (auth.Identity, error) {
	if ctx == nil {
		ctx = a.runtimeScope.Context()
	}
	if token == "" {
		return nil, autherrors.NewAuthError(autherrors.ErrInvalidTokenID, "token cannot be empty")
	}

	// Check the cache.
	if a.cacheEnabled && tokenType == auth.AccessToken && !checkRevoked {
		if identity := a.getFromCache(token); identity != nil {
			return identity, nil
		}
	}

	// Parse and validate the token.
	claims, err := FromToken(token, func(t *jwt.Token) (interface{}, error) {
		return a.keyProvider.GetPublicKey()
	})

	if err != nil {
		return nil, err
	}

	// Validate the token type.
	if claims.Type != tokenType {
		return nil, autherrors.NewAuthErrorf(autherrors.ErrTokenTypeMismatch,
			"token type mismatch: need %s, got %s", tokenType, claims.Type)
	}

	// Check whether the token has been revoked only when requested.
	if checkRevoked {
		revoked, err := a.revokeStore.IsRevoked(ctx, claims.ID)
		if err != nil {
			a.runtimeScope.Logger().Warn("token revocation check failed", "error", err)
		} else if revoked {
			return nil, autherrors.NewAuthError(autherrors.ErrTokenAlreadyRevoked, "token has been revoked")
		}
	}

	// Create the identity object.
	identity := claims.ToIdentity()

	// Add the identity to the cache.
	if a.cacheEnabled && tokenType == auth.AccessToken {
		a.addToCache(token, identity)
	}

	return identity, nil
}

// CreateTokens creates an access/refresh token pair.
func (a *JwtAuthenticator) CreateTokens(ctx context.Context, userID string, metadata map[string]interface{}) (*auth.TokenPair, error) {
	if userID == "" {
		return nil, autherrors.NewAuthError(autherrors.ErrInvalidUserID, "user ID cannot be empty")
	}

	now := time.Now()

	// Set the access-token expiry.
	accessExpiry := now.Add(a.accessExpiry)
	if a.accessExpiry <= 0 {
		accessExpiry = now.Add(15 * time.Minute) // Default to 15 minutes.
	}

	// Set the refresh-token expiry.
	refreshExpiry := now.Add(a.refreshExpiry)
	if a.refreshExpiry <= 0 {
		refreshExpiry = now.Add(7 * 24 * time.Hour) // Default to 7 days.
	}

	// Create token IDs.
	accessTokenID := uuid.New().String()
	refreshTokenID := uuid.New().String()

	// Create access-token claims.
	accessClaims := NewClaims(
		userID,
		accessTokenID,
		auth.AccessToken,
		metadata,
		accessExpiry,
	)

	// Create refresh-token claims.
	refreshClaims := NewClaims(
		userID,
		refreshTokenID,
		auth.RefreshToken,
		nil, // Refresh tokens do not include sensitive metadata.
		refreshExpiry,
	)

	// Get the private key.
	privateKey, err := a.keyProvider.GetPrivateKey()
	if err != nil {
		return nil, autherrors.WrapAuthError(err, autherrors.ErrTokenSigningFailed, "failed to get private key")
	}

	// Sign the access token.
	accessToken := jwt.NewWithClaims(jwt.SigningMethodRS256, accessClaims)
	accessTokenString, err := accessToken.SignedString(privateKey)
	if err != nil {
		return nil, autherrors.WrapAuthError(err, autherrors.ErrTokenSigningFailed, "failed to sign access token")
	}

	// Sign the refresh token.
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodRS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString(privateKey)
	if err != nil {
		return nil, autherrors.WrapAuthError(err, autherrors.ErrTokenSigningFailed, "failed to sign refresh token")
	}

	// Build the token pair.
	tokenPair := &auth.TokenPair{
		AccessToken:      accessTokenString,
		RefreshToken:     refreshTokenString,
		ExpiresAt:        accessExpiry.UnixMilli(),
		RefreshExpiresAt: refreshExpiry.UnixMilli(),
	}

	return tokenPair, nil
}

// CreateAccessTokenWithTTL issues an access token with a custom TTL.
// It does not create a refresh token.
func (a *JwtAuthenticator) CreateAccessTokenWithTTL(ctx context.Context, userID string, metadata map[string]interface{}, ttl time.Duration) (string, int64, error) {
	if userID == "" {
		return "", 0, autherrors.NewAuthError(autherrors.ErrInvalidUserID, "user ID cannot be empty")
	}
	if ctx == nil {
		ctx = a.runtimeScope.Context()
	}

	now := time.Now()
	expiry := now.Add(ttl)
	if ttl <= 0 {
		expiry = now.Add(a.accessExpiry)
		if a.accessExpiry <= 0 {
			expiry = now.Add(15 * time.Minute)
		}
	}

	accessTokenID := uuid.New().String()
	accessClaims := NewClaims(
		userID,
		accessTokenID,
		auth.AccessToken,
		metadata,
		expiry,
	)

	privateKey, err := a.keyProvider.GetPrivateKey()
	if err != nil {
		return "", 0, autherrors.WrapAuthError(err, autherrors.ErrTokenSigningFailed, "failed to get private key")
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodRS256, accessClaims)
	accessTokenString, err := accessToken.SignedString(privateKey)
	if err != nil {
		return "", 0, autherrors.WrapAuthError(err, autherrors.ErrTokenSigningFailed, "failed to sign access token")
	}

	return accessTokenString, expiry.UnixMilli(), nil
}

// RefreshTokens uses a refresh token to create a new token pair.
func (a *JwtAuthenticator) RefreshTokens(ctx context.Context, refreshToken string, metadata map[string]interface{}) (*auth.TokenPair, error) {
	if ctx == nil {
		ctx = a.runtimeScope.Context()
	}
	// Validate the refresh token and check revocation at the same time.
	identity, err := a.ValidateToken(ctx, refreshToken, auth.RefreshToken, true)
	if err != nil {
		return nil, err
	}

	// Get the user ID.
	userID := identity.GetUserID()

	// Revoke the current refresh token.
	if err := a.RevokeToken(ctx, refreshToken, "refreshed into a new token"); err != nil {
		a.runtimeScope.Logger().Warn("refresh token revocation failed", "error", err)
	}

	// Create a new token pair with the latest metadata.
	return a.CreateTokens(ctx, userID, metadata)
}

// RevokeToken revokes a token.
//
// token: token to revoke
// reason: revocation reason recorded in the store
func (a *JwtAuthenticator) RevokeToken(ctx context.Context, token string, reason string) error {
	if ctx == nil {
		ctx = a.runtimeScope.Context()
	}
	if token == "" {
		return autherrors.NewAuthError(autherrors.ErrInvalidTokenID, "token cannot be empty")
	}

	// Set the default revocation reason.
	if reason == "" {
		reason = "user request"
	}

	// Parse the token without validating the signature.
	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	claims := &Claims{}
	_, _, err := parser.ParseUnverified(token, claims)
	if err != nil {
		return autherrors.WrapAuthError(err, autherrors.ErrTokenParsingFailed, "failed to parse token")
	}

	// Read the token ID, user ID, and expiry time.
	tokenID := claims.ID
	userID := claims.Subject

	// Ensure the token ID is present.
	if tokenID == "" {
		return autherrors.NewAuthError(autherrors.ErrInvalidTokenID, "token ID is empty")
	}

	// Remove the token from cache.
	if a.cacheEnabled {
		a.removeFromCache(token)
	}

	// Read the expiry time.
	var expiresAt time.Time
	if claims.ExpiresAt != nil {
		expiresAt = claims.ExpiresAt.Time
	} else {
		// If the token has no expiry, use a longer fallback lifetime.
		expiresAt = time.Now().Add(30 * 24 * time.Hour)
	}

	// Read the token type.
	tokenType := claims.Type
	if tokenType == "" {
		tokenType = auth.AccessToken
	}

	// Revoke the token.
	return a.revokeStore.RevokeToken(ctx, tokenID, userID, tokenType, expiresAt, reason)
}

// RevokeAllUserTokens revokes all tokens for a user except the specified token.
//
// userID: user ID
// exceptTokenID: token ID to keep, if any
// reason: revocation reason recorded in the store
func (a *JwtAuthenticator) RevokeAllUserTokens(ctx context.Context, userID string, exceptTokenID string, reason string) (int, error) {
	if ctx == nil {
		ctx = a.runtimeScope.Context()
	}
	if userID == "" {
		return 0, autherrors.NewAuthError(autherrors.ErrInvalidUserID, "user ID cannot be empty")
	}

	// Set the default revocation reason.
	if reason == "" {
		reason = "bulk revocation"
	}

	// Clear cached tokens for the user when caching is enabled.
	if a.cacheEnabled {
		a.clearUserCache(userID, exceptTokenID)
	}

	// Revoke the tokens.
	return a.revokeStore.RevokeAllUserTokens(ctx, userID, exceptTokenID, reason)
}

// Close closes the authenticator and releases its resources.
func (a *JwtAuthenticator) Close() error {
	var errs []error

	// Close the key provider.
	if err := a.keyProvider.Close(); err != nil {
		errs = append(errs, fmt.Errorf("failed to close key provider: %w", err))
	}

	// Close the revocation store.
	if err := a.revokeStore.Close(); err != nil {
		errs = append(errs, fmt.Errorf("failed to close revocation store: %w", err))
	}
	if a.cache != nil {
		if err := a.cache.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close identity cache: %w", err))
		}
	}

	if len(errs) > 0 {
		// Combine error messages.
		var errMsgs []string
		for _, err := range errs {
			errMsgs = append(errMsgs, err.Error())
		}
		return fmt.Errorf("errors occurred while closing authenticator: %s", strings.Join(errMsgs, "; "))
	}

	return nil
}

// Cache helpers.

// getFromCache retrieves identity data from cache.
func (a *JwtAuthenticator) getFromCache(token string) *Identity {
	if !a.cacheEnabled || a.cache == nil {
		return nil
	}

	cached, err := a.cache.get(token)
	if err != nil {
		a.runtimeScope.Logger().Warn("jwt identity cache read failed", "error", err)
		return nil
	}
	if cached == nil {
		return nil
	}

	return cached.identity
}

// addToCache stores identity data in cache.
func (a *JwtAuthenticator) addToCache(token string, identity *Identity) {
	if !a.cacheEnabled || a.cache == nil {
		return
	}

	// Calculate the cache expiry.
	cacheTTL := time.Duration(0)
	if jwtCfg := runtimeOptionsFromScope(a.runtimeScope).authJWT; jwtCfg != nil && jwtCfg.IdentityCache != nil {
		cacheTTL = jwtCfg.IdentityCache.TTL
	}
	if cacheTTL <= 0 {
		cacheTTL = 5 * time.Minute
	}

	now := time.Now()
	cacheExpiry := now.Add(cacheTTL)

	// Ensure the cache expiry does not exceed the token expiry.
	if identity.expiresAt.Before(cacheExpiry) {
		cacheExpiry = identity.expiresAt
	}
	if !cacheExpiry.After(now) {
		return
	}

	cached := &cachedIdentity{
		identity:  identity,
		expiry:    cacheExpiry,
		cacheTime: now,
	}

	if err := a.cache.set(token, cached); err != nil {
		a.runtimeScope.Logger().Warn("jwt identity cache update failed", "error", err)
	}
}

// removeFromCache removes a token from cache.
func (a *JwtAuthenticator) removeFromCache(token string) {
	if !a.cacheEnabled || a.cache == nil {
		return
	}
	if err := a.cache.delete(token); err != nil {
		a.runtimeScope.Logger().Warn("jwt identity cache entry deletion failed", "error", err)
	}
}

// clearUserCache clears cached tokens for a user.
func (a *JwtAuthenticator) clearUserCache(userID string, exceptTokenID string) {
	if !a.cacheEnabled || a.cache == nil {
		return
	}
	if err := a.cache.deleteUserTokens(userID, exceptTokenID); err != nil {
		a.runtimeScope.Logger().Warn("jwt identity cache clear failed", "error", err, "user_id", userID)
	}
}
