// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

// Package esmresolver provides an esbuild plugin that resolves bare imports
// via a configurable ESM upstream (default: esm.sh) with a global cache.
package esmresolver

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/evanw/esbuild/pkg/api"
	"golang.org/x/sync/singleflight"
)

// Resolver is an esbuild plugin that intercepts bare imports and resolves them
// through an ESM CDN with local caching.
type Resolver struct {
	upstream     string
	cacheDir     string
	target       string
	client       *http.Client
	singleflight singleflight.Group
}

// Option configures a Resolver.
type Option func(*Resolver)

// WithUpstream overrides the default ESM upstream URL.
func WithUpstream(url string) Option {
	return func(r *Resolver) {
		if url != "" {
			r.upstream = strings.TrimRight(url, "/")
		}
	}
}

// WithCacheDir sets the cache root directory. Code cache is stored under
// <cacheDir>/pkg/esm/.
func WithCacheDir(dir string) Option {
	return func(r *Resolver) {
		if dir != "" {
			r.cacheDir = dir
		}
	}
}

// WithTarget sets the esm.sh target parameter (e.g. "deno", "es2020").
func WithTarget(target string) Option {
	return func(r *Resolver) {
		if target != "" {
			r.target = target
		}
	}
}

// WithHTTPClient overrides the HTTP client used for upstream requests.
func WithHTTPClient(client *http.Client) Option {
	return func(r *Resolver) {
		if client != nil {
			r.client = client
		}
	}
}

// New creates a Resolver with the given options.
func New(opts ...Option) *Resolver {
	r := &Resolver{
		upstream: "https://esm.sh",
		target:   "es2020",
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// Plugin returns an esbuild plugin that resolves bare imports via the ESM CDN.
func (r *Resolver) Plugin() api.Plugin {
	isBareImport := func(path string) bool {
		if path == "" || strings.HasPrefix(path, ".") || strings.HasPrefix(path, "/") {
			return false
		}
		if strings.HasPrefix(path, "@/") {
			return false
		}
		if strings.HasPrefix(path, "data:") || strings.HasPrefix(path, "#") {
			return false
		}
		return true
	}

	return api.Plugin{
		Name: "choysum-esm-resolver",
		Setup: func(build api.PluginBuild) {
			build.OnResolve(api.OnResolveOptions{Filter: ".*"}, func(args api.OnResolveArgs) (api.OnResolveResult, error) {
				if !isBareImport(args.Path) {
					return api.OnResolveResult{}, nil
				}

				// For backend (deno target): CSS imports are external.
				if r.target == "deno" && args.Kind == api.ResolveCSSURLToken {
					resolvedURL := fmt.Sprintf("%s/%s?target=%s", r.upstream, args.Path, r.target)
					return api.OnResolveResult{
						Path:     resolvedURL,
						External: true,
					}, nil
				}

				// Map bare import to esm.sh URL.
				esmURL := fmt.Sprintf("%s/%s?target=%s", r.upstream, args.Path, r.target)
				return api.OnResolveResult{
					Path:      esmURL,
					Namespace: "choysum-esm",
				}, nil
			})

			build.OnLoad(api.OnLoadOptions{Filter: ".*", Namespace: "choysum-esm"}, func(args api.OnLoadArgs) (api.OnLoadResult, error) {
				url := args.Path
				cacheKey := sha256Hex(url)
				cacheFile := filepath.Join(r.codeCacheDir(), cacheKey[:2], cacheKey[2:])

				// Try cache first.
				if content, err := os.ReadFile(cacheFile); err == nil {
					return api.OnLoadResult{
						Contents: ptr(string(content)),
						Loader:   api.LoaderJS,
					}, nil
				}

				// Download with singleflight deduplication.
				type fetchResult struct {
					content string
					err     error
				}
				v, err, _ := r.singleflight.Do(cacheKey, func() (any, error) {
					content, dlErr := r.download(url)
					if dlErr != nil {
						return fetchResult{}, dlErr
					}
					if writeErr := r.writeCache(cacheFile, []byte(content)); writeErr != nil {
						_ = writeErr
					}
					return fetchResult{content: content}, nil
				})
				if err != nil {
					return api.OnLoadResult{}, fmt.Errorf("[esm-resolver] download failed: %w\n  url: %s\n  hint: check that the package exists on %s", err, url, r.upstream)
				}
				result := v.(fetchResult)

				return api.OnLoadResult{
					Contents: ptr(result.content),
					Loader:   api.LoaderJS,
				}, nil
			})
		},
	}
}

func (r *Resolver) codeCacheDir() string {
	if r.cacheDir == "" {
		return filepath.Join("pkg", "esm")
	}
	return filepath.Join(r.cacheDir, "pkg", "esm")
}

func (r *Resolver) download(url string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	resp, err := r.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("http get: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return "", fmt.Errorf("http %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read body: %w", err)
	}
	return string(content), nil
}

func (r *Resolver) writeCache(cacheFile string, data []byte) error {
	dir := filepath.Dir(cacheFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	tmpFile := cacheFile + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmpFile, cacheFile)
}

func sha256Hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func ptr[T any](v T) *T {
	return &v
}
