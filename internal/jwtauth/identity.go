// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package jwtauth

import (
	"time"

	"github.com/choysum-dev/choysum/pkg/auth"
)

// Identity implements auth.Identity for JWT-backed identities.
type Identity struct {
	userID    string                 // User ID.
	tokenID   string                 // Token ID.
	metadata  map[string]interface{} // Metadata.
	expiresAt time.Time              // Expiration time.
	issuedAt  time.Time              // Issue time.
	tokenType auth.TokenType         // Token type.
	issuer    string                 // Issuer.
}

// NewIdentity creates a new JWT identity.
func NewIdentity(
	userID string,
	tokenID string,
	metadata map[string]interface{},
	expiresAt time.Time,
	issuedAt time.Time,
	tokenType auth.TokenType,
	issuer string,
) *Identity {
	return &Identity{
		userID:    userID,
		tokenID:   tokenID,
		metadata:  metadata,
		expiresAt: expiresAt,
		issuedAt:  issuedAt,
		tokenType: tokenType,
		issuer:    issuer,
	}
}

// GetUserID returns the user ID.
func (i *Identity) GetUserID() string {
	return i.userID
}

// GetTokenID returns the token ID.
func (i *Identity) GetTokenID() string {
	return i.tokenID
}

// GetMetadata returns the identity metadata.
func (i *Identity) GetMetadata() map[string]interface{} {
	return i.metadata
}

// IsValid reports whether the identity is still valid.
func (i *Identity) IsValid() bool {
	if i == nil || i.userID == "" || i.tokenID == "" {
		return false
	}

	// Check whether the token has expired.
	return time.Now().Before(i.expiresAt)
}

// GetExpiresAt returns the expiration time.
func (i *Identity) GetExpiresAt() time.Time {
	return i.expiresAt
}

// GetIssuedAt returns the issue time.
func (i *Identity) GetIssuedAt() time.Time {
	return i.issuedAt
}

// GetTokenType returns the token type.
func (i *Identity) GetTokenType() auth.TokenType {
	return i.tokenType
}

// GetIssuer returns the issuer.
func (i *Identity) GetIssuer() string {
	return i.issuer
}
