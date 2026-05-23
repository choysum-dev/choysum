// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package grpcauth

// Option configures an AuthInterceptor.
type Option func(*AuthInterceptor)

// WithHeaderName sets the header used to extract tokens.
func WithHeaderName(name string) Option {
	return func(i *AuthInterceptor) {
		i.headerName = name
	}
}

// WithEntryAuthSkipMethods sets methods that may skip authentication for inbound (depth=0) requests.
func WithEntryAuthSkipMethods(methods ...string) Option {
	return func(i *AuthInterceptor) {
		i.entryAuthSkipMethods = append(i.entryAuthSkipMethods, methods...)
	}
}
