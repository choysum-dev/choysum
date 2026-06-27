// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package origin

import "context"

type bootstrapRegistryFallbackContextKey struct{}

// WithBootstrapRegistryFallback marks ctx so local-origin install resolution can
// use registry fallback even when the global fallback toggle is disabled.
func WithBootstrapRegistryFallback(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, bootstrapRegistryFallbackContextKey{}, true)
}

func bootstrapRegistryFallbackEnabled(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	enabled, _ := ctx.Value(bootstrapRegistryFallbackContextKey{}).(bool)
	return enabled
}
