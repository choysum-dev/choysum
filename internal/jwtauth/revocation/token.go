// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package revocation

import (
	"time"

	"github.com/choysum-dev/choysum/pkg/auth"
)

// Token describes a revoked token record.
type Token interface {
	// GetTokenID returns the token ID.
	GetTokenID() string

	// GetUserID returns the user ID.
	GetUserID() string

	// GetTokenType returns the token type.
	GetTokenType() auth.TokenType

	// GetRevokedAt returns the revocation time.
	GetRevokedAt() time.Time

	// GetExpiresAt returns the expiration time.
	GetExpiresAt() time.Time

	// GetReason returns the revocation reason.
	GetReason() string
}

// StandardToken is the default Token implementation.
type StandardToken struct {
	TokenID   string         // Token ID.
	UserID    string         // User ID.
	TokenType auth.TokenType // Token type.
	RevokedAt time.Time      // Revocation time.
	ExpiresAt time.Time      // Expiration time.
	Reason    string         // Revocation reason.
}

// GetTokenID returns the token ID.
func (t *StandardToken) GetTokenID() string {
	return t.TokenID
}

// GetUserID returns the user ID.
func (t *StandardToken) GetUserID() string {
	return t.UserID
}

// GetTokenType returns the token type.
func (t *StandardToken) GetTokenType() auth.TokenType {
	return t.TokenType
}

// GetRevokedAt returns the revocation time.
func (t *StandardToken) GetRevokedAt() time.Time {
	return t.RevokedAt
}

// GetExpiresAt returns the expiration time.
func (t *StandardToken) GetExpiresAt() time.Time {
	return t.ExpiresAt
}

// GetReason returns the revocation reason.
func (t *StandardToken) GetReason() string {
	return t.Reason
}

// NewToken creates a standard revoked token record.
func NewToken(tokenID string, userID string, tokenType auth.TokenType, expiresAt time.Time, reason string) Token {
	return &StandardToken{
		TokenID:   tokenID,
		UserID:    userID,
		TokenType: tokenType,
		RevokedAt: time.Now(),
		ExpiresAt: expiresAt,
		Reason:    reason,
	}
}
