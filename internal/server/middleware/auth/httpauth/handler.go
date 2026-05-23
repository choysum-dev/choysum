// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package httpauth

import (
	"net/http"
	"regexp"
	"strings"

	middleware "github.com/choysum-dev/choysum/internal/server/middleware/auth"
	"github.com/choysum-dev/choysum/pkg/auth"
	"github.com/choysum-dev/choysum/pkg/auth/autherrors"
	"github.com/choysum-dev/choysum/pkg/scope"
)

// ErrorHandler defines the error handler function type.
type ErrorHandler func(w http.ResponseWriter, r *http.Request, err error)

// AuthHandler is the HTTP authentication handler.
type AuthHandler struct {
	authenticator auth.Authenticator
	runtimeScope  scope.Scope
	extractors    []TokenExtractor
	errorHandler  ErrorHandler
	excludePaths  []string
	excludeRegex  []*regexp.Regexp
	respFormat    middleware.ResponseFormat
}

// NewAuthHandler creates a new authentication handler.
func NewAuthHandler(runtimeScope scope.Scope, authenticator auth.Authenticator, opts ...Option) *AuthHandler {
	h := &AuthHandler{
		authenticator: authenticator,
		runtimeScope:  runtimeScope,
		extractors:    defaultExtractors(),
		excludePaths:  []string{},
		excludeRegex:  []*regexp.Regexp{},
		respFormat:    middleware.JSONResponseFormat,
	}

	// Apply the default error handler.
	h.errorHandler = h.defaultErrorHandler

	// Apply custom options.
	for _, opt := range opts {
		opt(h)
	}

	return h
}

// defaultErrorHandler is the default error handler implementation.
func (h *AuthHandler) defaultErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	middleware.FormatHTTPError(w, r, err, h.respFormat, h.runtimeScope)
}

// extractToken extracts a token from the request.
func (h *AuthHandler) extractToken(r *http.Request) (string, error) {
	for _, extractor := range h.extractors {
		if token, err := extractor.Extract(r); err == nil && token != "" {
			return token, nil
		}
	}
	// Use autherrors for the missing-token error.
	return "", autherrors.NewAuthError(autherrors.ErrMissingToken, "authentication token not found")
}

// Handler returns the middleware handler.
func (h *AuthHandler) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip authentication when the authenticator is nil.
		if h.authenticator == nil {
			next.ServeHTTP(w, r)
			return
		}

		// Check whether the path is excluded.
		if middleware.IsPathExcluded(r.URL.Path, h.excludePaths, h.excludeRegex) {
			next.ServeHTTP(w, r)
			return
		}

		// Extract the token.
		token, err := h.extractToken(r)
		if err != nil {
			h.errorHandler(w, r, err)
			return
		}

		// Validate the token and always check revocation.
		identity, err := h.authenticator.ValidateToken(r.Context(), token, auth.AccessToken, true)
		if err != nil {
			h.errorHandler(w, r, err)
			return
		}

		// Write the identity and request access token into trusted context (Go-only).
		ctx := auth.ContextWithIdentity(r.Context(), identity)
		ctx = auth.ContextWithAccessToken(ctx, token)
		r = r.WithContext(ctx)

		// Call the next handler.
		next.ServeHTTP(w, r)
	})
}

// defaultExtractors returns the built-in token extractors.
func defaultExtractors() []TokenExtractor {
	return []TokenExtractor{
		&HeaderTokenExtractor{},
		&CookieTokenExtractor{},
		&QueryTokenExtractor{},
	}
}

// AuthHandlerFunc creates a convenience HTTP auth middleware.
func AuthHandlerFunc(runtimeScope scope.Scope, authenticator auth.Authenticator) func(next http.Handler) http.Handler {
	return NewAuthHandler(runtimeScope, authenticator).Handler
}

// AuthHandlerFromConfig creates HTTP auth middleware from config.
func AuthHandlerFromConfig(runtimeScope scope.Scope, authenticator auth.Authenticator) func(next http.Handler) http.Handler {
	runtimeOpts := runtimeOptionsFromScope(runtimeScope)
	if authenticator == nil || !runtimeOpts.enabled || runtimeOpts.httpAuth == nil {
		// Return a no-op middleware.
		return func(next http.Handler) http.Handler { return next }
	}

	cfg := runtimeOpts.httpAuth
	opts := []Option{}

	// Apply configuration.
	if len(cfg.ExcludedPaths) > 0 {
		opts = append(opts, WithExcludePaths(cfg.ExcludedPaths...))
	}

	if len(cfg.ExcludedRegex) > 0 {
		opts = append(opts, WithPathRegexExclude(cfg.ExcludedRegex...))
	}

	// Configure token extractors.
	if len(cfg.TokenExtractors) > 0 {
		extractors := []TokenExtractor{}
		for _, name := range cfg.TokenExtractors {
			switch strings.ToLower(name) {
			case "header":
				extractors = append(extractors, &HeaderTokenExtractor{})
			case "cookie":
				e := &CookieTokenExtractor{}
				if cfg.CookieName != "" {
					e.Name = cfg.CookieName
				}
				extractors = append(extractors, e)
			case "query":
				e := &QueryTokenExtractor{}
				if cfg.QueryParamName != "" {
					e.ParamName = cfg.QueryParamName
				}
				extractors = append(extractors, e)
			}
		}
		if len(extractors) > 0 {
			opts = append(opts, WithTokenExtractors(extractors...))
		}
	}

	// Configure the response format.
	if cfg.ResponseFormat == "text" {
		opts = append(opts, WithResponseFormat(middleware.PlainTextResponseFormat))
	}

	return NewAuthHandler(runtimeScope, authenticator, opts...).Handler
}
