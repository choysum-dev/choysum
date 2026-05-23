// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package jwtauth

import (
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/choysum-dev/choysum/pkg/auth"
	"github.com/choysum-dev/choysum/pkg/auth/autherrors"
)

func signedTokenForClaimsTest(t *testing.T, claims *Claims) string {
	t.Helper()
	privateKey, _ := loadTestKeys(t)
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenString, err := token.SignedString(privateKey)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return tokenString
}

func TestFromToken(t *testing.T) {
	_, publicKey := loadTestKeys(t)
	keyFunc := func(*jwt.Token) (interface{}, error) {
		return publicKey, nil
	}

	t.Run("parses valid token", func(t *testing.T) {
		claims := NewClaims("user-1", "token-1", auth.AccessToken, map[string]interface{}{"role": "admin"}, time.Now().Add(time.Hour))
		tokenString := signedTokenForClaimsTest(t, claims)

		parsed, err := FromToken(tokenString, keyFunc)
		if err != nil {
			t.Fatalf("FromToken() error = %v", err)
		}
		if parsed.Subject != "user-1" || parsed.ID != "token-1" || parsed.Type != auth.AccessToken {
			t.Fatalf("unexpected parsed claims: %#v", parsed)
		}
		if parsed.Metadata["role"] != "admin" {
			t.Fatalf("unexpected metadata: %#v", parsed.Metadata)
		}
	})

	t.Run("returns token expired error", func(t *testing.T) {
		claims := NewClaims("user-1", "token-1", auth.AccessToken, nil, time.Now().Add(-time.Hour))
		tokenString := signedTokenForClaimsTest(t, claims)

		_, err := FromToken(tokenString, keyFunc)
		if err == nil || !autherrors.IsAuthError(err, autherrors.ErrTokenExpired) {
			t.Fatalf("expected token expired error, got %v", err)
		}
	})

	t.Run("returns invalid access token on signature error", func(t *testing.T) {
		otherKey, err := rsa.GenerateKey(rand.Reader, 1024)
		if err != nil {
			t.Fatalf("generate signing key: %v", err)
		}
		claims := NewClaims("user-1", "token-1", auth.AccessToken, nil, time.Now().Add(time.Hour))
		token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
		tokenString, err := token.SignedString(otherKey)
		if err != nil {
			t.Fatalf("sign token with other key: %v", err)
		}

		_, err = FromToken(tokenString, keyFunc)
		if err == nil || !autherrors.IsAuthError(err, autherrors.ErrInvalidAccessToken) {
			t.Fatalf("expected invalid access token error, got %v", err)
		}
	})

	t.Run("returns invalid access token on malformed token", func(t *testing.T) {
		_, err := FromToken("not-a-jwt", keyFunc)
		if err == nil || !autherrors.IsAuthError(err, autherrors.ErrInvalidAccessToken) {
			t.Fatalf("expected malformed token error, got %v", err)
		}
	})

	t.Run("wraps other parsing errors", func(t *testing.T) {
		claims := NewClaims("user-1", "token-1", auth.AccessToken, nil, time.Now().Add(time.Hour))
		tokenString := signedTokenForClaimsTest(t, claims)

		_, err := FromToken(tokenString, func(*jwt.Token) (interface{}, error) {
			return nil, errors.New("key func boom")
		})
		if err == nil || !autherrors.IsAuthError(err, autherrors.ErrTokenParsingFailed) {
			t.Fatalf("expected wrapped parsing error, got %v", err)
		}
	})
}
