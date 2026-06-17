// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

// Package esmresolver provides an esbuild plugin that resolves bare imports
// via a configurable ESM upstream (default: esm.sh) with a global cache.
package esmresolver

import (
	"context"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"net"
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
	offline      bool
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

// WithOffline enables offline mode. Cache misses produce a hard error instead
// of attempting a network download.
func WithOffline(offline bool) Option {
	return func(r *Resolver) {
		r.offline = offline
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
				pkg := extractPkgFromURL(url, r.upstream)
				cacheKey := sha256Hex(url)
				cacheFile := filepath.Join(r.codeCacheDir(), cacheKey[:2], cacheKey[2:])

				// Try cache first, verify integrity if metadata present.
				if content, ok := r.readCache(cacheFile); ok {
					return api.OnLoadResult{
						Contents: ptr(content),
						Loader:   api.LoaderJS,
					}, nil
				}

				// Offline mode: cache miss is a hard error.
				if r.offline {
					return api.OnLoadResult{}, r.formatError("cache miss (offline)", pkg, url,
						"run 'choysum install' with network access to populate the cache")
				}

				// Download with singleflight deduplication.
				type fetchResult struct {
					content string
					err     error
				}
				v, err, _ := r.singleflight.Do(cacheKey, func() (any, error) {
					content, dlErr := r.downloadWithRetry(url)
					if dlErr != nil {
						return fetchResult{}, dlErr
					}
					if writeErr := r.writeCache(cacheFile, []byte(content)); writeErr != nil {
						_ = writeErr
					}
					return fetchResult{content: content}, nil
				})
				if err != nil {
					return api.OnLoadResult{}, r.formatError("download failed", pkg, url, err.Error())
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

// readCache reads a cached file and verifies its integrity.
// Returns the content and true on success, or "" and false on miss/corruption.
func (r *Resolver) readCache(cacheFile string) (string, bool) {
	content, err := os.ReadFile(cacheFile)
	if err != nil {
		return "", false
	}
	integrityFile := cacheFile + ".integrity"
	expected, err := os.ReadFile(integrityFile)
	if err != nil {
		// No integrity metadata — accept the cached content as-is.
		return string(content), true
	}
	actual := sha512Hex(string(content))
	if strings.TrimSpace(string(expected)) != actual {
		// Integrity mismatch — delete dirty cache and return miss.
		_ = os.Remove(cacheFile)
		_ = os.Remove(integrityFile)
		return "", false
	}
	return string(content), true
}

func (r *Resolver) codeCacheDir() string {
	if r.cacheDir == "" {
		return filepath.Join("pkg", "esm")
	}
	return filepath.Join(r.cacheDir, "pkg", "esm")
}

// downloadWithRetry fetches a URL with exponential backoff (initial 1s, max 10s,
// up to 3 attempts). 4xx responses are not retried; 5xx and network errors are.
func (r *Resolver) downloadWithRetry(url string) (string, error) {
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			delay := time.Duration(math.Min(float64(time.Second<<(attempt-1)), float64(10*time.Second)))
			time.Sleep(delay)
		}
		content, err := r.download(url)
		if err == nil {
			return content, nil
		}
		lastErr = err
		if !isRetryable(err) {
			break
		}
	}
	return "", lastErr
}

func (r *Resolver) download(url string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
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
		return "", &httpError{code: resp.StatusCode, body: strings.TrimSpace(string(body))}
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read body: %w", err)
	}
	return string(content), nil
}

type httpError struct {
	code int
	body string
}

func (e *httpError) Error() string {
	return fmt.Sprintf("http %d: %s", e.code, e.body)
}

// isRetryable reports whether an error from download() is worth retrying.
// 4xx errors are not retried; 5xx and network errors are.
func isRetryable(err error) bool {
	if err == nil {
		return false
	}
	var httpErr *httpError
	if asHTTPErr(err, &httpErr) {
		return httpErr.code >= 500
	}
	// Network errors (DNS, connection refused, timeout, etc.) are retryable.
	return true
}

func asHTTPErr(err error, target **httpError) bool {
	if err == nil {
		return false
	}
	// Check if the error chain contains *httpError.
	for e := err; e != nil; e = unwrapErr(e) {
		if he, ok := e.(*httpError); ok {
			*target = he
			return true
		}
	}
	return false
}

func unwrapErr(err error) error {
	type unwrapper interface{ Unwrap() error }
	if u, ok := err.(unwrapper); ok {
		return u.Unwrap()
	}
	return nil
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
	if err := os.Rename(tmpFile, cacheFile); err != nil {
		return err
	}
	// Write integrity metadata alongside the cache file.
	integrityFile := cacheFile + ".integrity"
	hash := sha512Hex(string(data))
	_ = os.WriteFile(integrityFile, []byte(hash), 0644)
	return nil
}

func (r *Resolver) formatError(errorType, pkg, url, detail string) error {
	var b strings.Builder
	b.WriteString("[esm-resolver] ")
	b.WriteString(errorType)
	if pkg != "" {
		b.WriteString("\n  package: ")
		b.WriteString(pkg)
	}
	b.WriteString("\n  url: ")
	b.WriteString(url)
	if detail != "" {
		b.WriteString("\n  detail: ")
		b.WriteString(detail)
	}
	return fmt.Errorf("%s", b.String())
}

// extractPkgFromURL extracts a human-readable package identifier from an esm.sh URL.
func extractPkgFromURL(url, upstream string) string {
	prefix := strings.TrimRight(upstream, "/") + "/"
	if !strings.HasPrefix(url, prefix) {
		return ""
	}
	rest := strings.TrimPrefix(url, prefix)
	// Strip query string.
	if idx := strings.Index(rest, "?"); idx >= 0 {
		rest = rest[:idx]
	}
	return rest
}

func sha256Hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func sha512Hex(s string) string {
	h := sha512.Sum512([]byte(s))
	return hex.EncodeToString(h[:])
}

// isNetError reports whether err is a network-level error (not an HTTP response).
func isNetError(err error) bool {
	if err == nil {
		return false
	}
	// Check for common net errors without importing net package types.
	msg := err.Error()
	for _, substr := range []string{
		"connection refused",
		"no such host",
		"i/o timeout",
		"context deadline exceeded",
		"connection reset",
		"tls:",
	} {
		if strings.Contains(strings.ToLower(msg), substr) {
			return true
		}
	}
	// Also check for net.Error interface.
	type netErr interface{ Timeout() bool }
	if ne, ok := err.(netErr); ok && ne.Timeout() {
		return true
	}
	if _, ok := err.(net.Error); ok {
		return true
	}
	return false
}

func ptr[T any](v T) *T {
	return &v
}
