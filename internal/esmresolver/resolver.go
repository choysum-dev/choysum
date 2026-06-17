// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

// Package esmresolver provides an esbuild plugin that resolves bare imports
// via a configurable ESM upstream (default: esm.sh) with a global cache.
package esmresolver

import (
	"github.com/evanw/esbuild/pkg/api"
)

// Resolver is an esbuild plugin that intercepts bare imports and resolves them
// through an ESM CDN with local caching.
type Resolver struct {
	upstream string
}

// Option configures a Resolver.
type Option func(*Resolver)

// WithUpstream overrides the default ESM upstream URL.
func WithUpstream(url string) Option {
	return func(r *Resolver) {
		if url != "" {
			r.upstream = url
		}
	}
}

// New creates a Resolver with the given options.
func New(opts ...Option) *Resolver {
	r := &Resolver{
		upstream: "https://esm.sh",
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// Plugin returns an esbuild plugin for this resolver. In Phase 0 it is a
// no-op plugin; OnResolve/OnLoad will be wired in Phase 1.
func (r *Resolver) Plugin() api.Plugin {
	return api.Plugin{
		Name: "choysum-esm-resolver",
		Setup: func(build api.PluginBuild) {
			// Phase 0: no-op placeholder. OnResolve/OnLoad are wired in Phase 1.
		},
	}
}
