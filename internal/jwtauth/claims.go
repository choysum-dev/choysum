// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package jwtauth

import (
	"strings"
	"time"

	"github.com/choysum-dev/choysum/pkg/auth"
	"github.com/choysum-dev/choysum/pkg/auth/autherrors"
	"github.com/golang-jwt/jwt/v5"
)

// Claims represents the JWT claims used by the authenticator.
type Claims struct {
	jwt.RegisteredClaims
	Type     auth.TokenType         `json:"typ,omitempty"`  // Token type: access or refresh.
	Metadata map[string]interface{} `json:"meta,omitempty"` // Custom metadata.
}

// NewClaims creates a new JWT claims payload.
func NewClaims(
	userID string,
	tokenID string,
	tokenType auth.TokenType,
	metadata map[string]interface{},
	expiresAt time.Time,
) *Claims {
	now := time.Now()

	claims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,  // User ID.
			ID:        tokenID, // Token ID.
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			Issuer:    "choysum", // Issuer.
			NotBefore: jwt.NewNumericDate(now),
		},
		Type:     tokenType,
		Metadata: metadata,
	}

	return claims
}

// ToIdentity converts Claims into an Identity.
func (c *Claims) ToIdentity() *Identity {
	expiresAt := time.Now()
	if c.ExpiresAt != nil {
		expiresAt = c.ExpiresAt.Time
	}

	issuedAt := time.Now()
	if c.IssuedAt != nil {
		issuedAt = c.IssuedAt.Time
	}

	return NewIdentity(
		c.Subject,
		c.ID,
		c.Metadata,
		expiresAt,
		issuedAt,
		c.Type,
		c.Issuer,
	)
}

// FromToken parses Claims from a JWT token.
func FromToken(tokenString string, keyFunc jwt.Keyfunc) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, keyFunc)

	// Handle parsing errors.
	if err != nil {
		// Check for specific error types.
		if strings.Contains(err.Error(), "token is expired") {
			return nil, autherrors.NewAuthError(autherrors.ErrTokenExpired, "token has expired")
		}

		// Check for signature validation errors.
		if strings.Contains(err.Error(), "signature is invalid") {
			return nil, autherrors.NewAuthError(autherrors.ErrInvalidAccessToken, "token signature is invalid")
		}

		// Check for malformed tokens.
		if strings.Contains(err.Error(), "malformed") {
			return nil, autherrors.NewAuthError(autherrors.ErrInvalidAccessToken, "token format is invalid")
		}

		// Wrap all other parsing errors.
		return nil, autherrors.WrapAuthError(err, autherrors.ErrTokenParsingFailed, "failed to parse token")
	}

	// Validate the token.
	if !token.Valid {
		return nil, autherrors.NewAuthError(autherrors.ErrInvalidAccessToken, "token is invalid")
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, autherrors.NewAuthError(autherrors.ErrInvalidTokenClaims, "invalid token claims format")
	}

	return claims, nil
}
