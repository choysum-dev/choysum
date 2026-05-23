// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package httpauth

import (
	middleware "github.com/choysum-dev/choysum/internal/server/middleware/auth"
)

// Option configures an AuthHandler.
type Option func(*AuthHandler)

// WithTokenExtractors sets token extractors.
func WithTokenExtractors(extractors ...TokenExtractor) Option {
	return func(h *AuthHandler) {
		h.extractors = extractors
	}
}

// WithExcludePaths sets excluded paths.
func WithExcludePaths(paths ...string) Option {
	return func(h *AuthHandler) {
		h.excludePaths = append(h.excludePaths, paths...)
	}
}

// WithPathRegexExclude sets regex-based path exclusions.
func WithPathRegexExclude(patterns ...string) Option {
	return func(h *AuthHandler) {
		h.excludeRegex = append(h.excludeRegex, middleware.CompileRegexPatterns(patterns)...)
	}
}

// WithErrorHandler sets a custom error handler.
func WithErrorHandler(handler ErrorHandler) Option {
	return func(h *AuthHandler) {
		h.errorHandler = handler
	}
}

// WithResponseFormat sets the response format.
func WithResponseFormat(format middleware.ResponseFormat) Option {
	return func(h *AuthHandler) {
		h.respFormat = format
	}
}
