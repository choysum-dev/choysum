// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package auth

import "context"

// TokenType defines the token kind.
type TokenType string

const (
	AccessToken  TokenType = "access"  // Access token for API requests.
	RefreshToken TokenType = "refresh" // Refresh token for obtaining a new access token.
)

// TokenPair holds an access token and its refresh token.
type TokenPair struct {
	AccessToken      string `json:"accessToken"`
	RefreshToken     string `json:"refreshToken"`
	ExpiresAt        int64  `json:"expiresAt"`
	RefreshExpiresAt int64  `json:"refreshExpiresAt"`
}

// Authenticator defines the authentication service contract.
type Authenticator interface {
	ValidateToken(ctx context.Context, token string, tokenType TokenType, checkRevoked bool) (Identity, error)
	CreateTokens(ctx context.Context, userID string, metadata map[string]interface{}) (*TokenPair, error)
	RefreshTokens(ctx context.Context, refreshToken string, metadata map[string]interface{}) (*TokenPair, error)
	RevokeToken(ctx context.Context, token string, reason string) error
	RevokeAllUserTokens(ctx context.Context, userID string, exceptTokenID string, reason string) (int, error)
	Close() error
}
