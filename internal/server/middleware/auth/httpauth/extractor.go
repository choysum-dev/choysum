// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package httpauth

import (
	"net/http"

	middleware "github.com/choysum-dev/choysum/internal/server/middleware/auth"
	xfmt "golang.org/x/exp/errors/fmt"
)

// TokenExtractor extracts a token from an HTTP request.
type TokenExtractor interface {
	// Extract attempts to extract a token from the request.
	Extract(r *http.Request) (string, error)
}

// HeaderTokenExtractor extracts a token from request headers.
type HeaderTokenExtractor struct {
	// AuthHeader is the header name, defaulting to "Authorization".
	AuthHeader string
}

// Extract extracts a token from request headers.
func (e *HeaderTokenExtractor) Extract(r *http.Request) (string, error) {
	header := "Authorization"
	if e.AuthHeader != "" {
		header = e.AuthHeader
	}

	authHeader := r.Header.Get(header)
	if authHeader == "" {
		return "", xfmt.Errorf("header %s is empty", header)
	}

	return middleware.ExtractBearerToken(authHeader), nil
}

// CookieTokenExtractor extracts a token from cookies.
type CookieTokenExtractor struct {
	// Name is the cookie name, defaulting to "auth_token".
	Name string
}

// Extract extracts a token from cookies.
func (e *CookieTokenExtractor) Extract(r *http.Request) (string, error) {
	cookieName := "auth_token"
	if e.Name != "" {
		cookieName = e.Name
	}

	cookie, err := r.Cookie(cookieName)
	if err != nil {
		return "", xfmt.Errorf("cookie %s not found: %w", cookieName, err)
	}

	return cookie.Value, nil
}

// QueryTokenExtractor extracts a token from URL query parameters.
type QueryTokenExtractor struct {
	// ParamName is the query parameter name, defaulting to "token".
	ParamName string
}

// Extract extracts a token from URL query parameters.
func (e *QueryTokenExtractor) Extract(r *http.Request) (string, error) {
	paramName := "token"
	if e.ParamName != "" {
		paramName = e.ParamName
	}

	token := r.URL.Query().Get(paramName)
	if token == "" {
		return "", xfmt.Errorf("URL query parameter %s is empty", paramName)
	}

	return token, nil
}
