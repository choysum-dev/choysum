// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package auth

import (
	"context"
)

// contextKey stores authentication-related values in context.
type contextKey int

const (
	authIdentityKey contextKey = iota
	authAccessTokenKey
	authInternalKeyKey
)

// ContextWithIdentity adds identity information to context.
func ContextWithIdentity(ctx context.Context, identity Identity) context.Context {
	return context.WithValue(ctx, authIdentityKey, identity)
}

// ContextWithAccessToken stores the *raw* access token string (without the "Bearer " prefix)
// in the context. This value is trusted and must never be exposed to JS-visible objects.
func ContextWithAccessToken(ctx context.Context, token string) context.Context {
	if token == "" {
		return ctx
	}
	return context.WithValue(ctx, authAccessTokenKey, token)
}

// AccessTokenFromContext returns the raw access token string (without the "Bearer " prefix)
// previously stored by ContextWithAccessToken.
func AccessTokenFromContext(ctx context.Context) (string, bool) {
	v := ctx.Value(authAccessTokenKey)
	token, ok := v.(string)
	if !ok || token == "" {
		return "", false
	}
	return token, true
}

// ContextWithInternalKey stores internal auth key in context for trusted internal calls.
func ContextWithInternalKey(ctx context.Context, key string) context.Context {
	if key == "" {
		return ctx
	}
	return context.WithValue(ctx, authInternalKeyKey, key)
}

// InternalKeyFromContext returns internal auth key stored in context.
func InternalKeyFromContext(ctx context.Context) (string, bool) {
	v := ctx.Value(authInternalKeyKey)
	key, ok := v.(string)
	if !ok || key == "" {
		return "", false
	}
	return key, true
}

// IdentityFromContext reads identity information from context.
func IdentityFromContext(ctx context.Context) Identity {
	if v := ctx.Value(authIdentityKey); v != nil {
		if id, ok := v.(Identity); ok {
			return id
		}
	}
	return nil
}

// IsAuthenticated reports whether context contains a valid identity.
func IsAuthenticated(ctx context.Context) bool {
	id := IdentityFromContext(ctx)
	return id != nil && id.IsValid()
}
