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
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/evanw/esbuild/pkg/api"
	"golang.org/x/sync/singleflight"
)

// Metrics holds lightweight counters for resolver observability.
// All fields are safe for concurrent access via atomic operations.
type Metrics struct {
	CacheHit  atomic.Int64
	CacheMiss atomic.Int64
	Downloads atomic.Int64
	Errors    atomic.Int64
	// total download duration in milliseconds (atomic)
	DownloadDurationMs atomic.Int64
	// DownloadedPkgs tracks unique package names that were downloaded (deduplicated by package name).
	DownloadedPkgs sync.Map
}

const maxResolverDownloadBytes int64 = 50 * 1024 * 1024

const (
	vueI18nBareSpecifier = "vue-i18n"
	vueI18nProdEntryPath = "dist/vue-i18n.esm-browser.prod.js"
)

// Snapshot returns a point-in-time copy of the metrics.
func (m *Metrics) Snapshot() (hit, miss, downloads, errors int64, downloadMs int64) {
	return m.CacheHit.Load(), m.CacheMiss.Load(), m.Downloads.Load(), m.Errors.Load(), m.DownloadDurationMs.Load()
}

// SnapshotDownloadedPkgs returns a sorted list of unique package names that
// were downloaded in the current build.
func (m *Metrics) SnapshotDownloadedPkgs() []string {
	if m == nil {
		return nil
	}
	pkgs := make([]string, 0)
	m.DownloadedPkgs.Range(func(key, _ any) bool {
		if s, ok := key.(string); ok && s != "" {
			pkgs = append(pkgs, s)
		}
		return true
	})
	sort.Strings(pkgs)
	return pkgs
}

// Resolver is an esbuild plugin that intercepts bare imports and resolves them
// through an ESM CDN with local caching.
type Resolver struct {
	upstream     string
	cacheDir     string
	target       string
	offline      bool
	moduleName   string
	application  string
	client       *http.Client
	singleflight singleflight.Group
	lockfilePath string       // path to esm.lock for version pinning
	modulePath   string       // module root for deriving lockfile path
	lockfileOnce sync.Once    // protects lockfile loading
	lockfile     *EsmLockfile // cached parsed lockfile (nil if not loaded)
	lockfileErr  error        // error from last lockfile load attempt
	logger       *slog.Logger // logger for structured metrics output (optional)
	metrics      *Metrics     // resolver metrics (nil if not initialised)
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

// WithModuleName sets a module label for resolver observability logs.
func WithModuleName(name string) Option {
	return func(r *Resolver) {
		name = strings.TrimSpace(name)
		if name != "" {
			r.moduleName = name
		}
	}
}

// WithApplicationName sets an application label for resolver observability logs.
func WithApplicationName(name string) Option {
	return func(r *Resolver) {
		name = strings.TrimSpace(name)
		if name != "" {
			r.application = name
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

// WithLockfile sets the path to an esm.lock file. When set, the resolver
// uses locked versions from the file to ensure reproducible builds.
func WithLockfile(path string) Option {
	return func(r *Resolver) {
		if path != "" {
			r.lockfilePath = path
		}
	}
}

// WithModulePath sets the module root directory. The lockfile is expected
// at <modulePath>/esm.lock unless an explicit lockfile path is provided.
func WithModulePath(path string) Option {
	return func(r *Resolver) {
		if path != "" {
			r.modulePath = path
		}
	}
}

// WithLogger sets the structured logger for metrics output. When set, the
// resolver emits cache/download metrics at the end of each build.
func WithLogger(logger *slog.Logger) Option {
	return func(r *Resolver) {
		if logger != nil {
			r.logger = logger
		}
	}
}

// WithMetrics sets a shared Metrics instance for cross-build aggregation.
// If not set, a private Metrics instance is created automatically.
func WithMetrics(m *Metrics) Option {
	return func(r *Resolver) {
		if m != nil {
			r.metrics = m
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
		metrics: &Metrics{},
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

func (r *Resolver) ensureDefaults() {
	if strings.TrimSpace(r.upstream) == "" {
		r.upstream = "https://esm.sh"
	}
	if strings.TrimSpace(r.target) == "" {
		r.target = "es2020"
	}
	if r.client == nil {
		r.client = &http.Client{Timeout: 30 * time.Second}
	}
	if r.metrics == nil {
		r.metrics = &Metrics{}
	}
}

// Plugin returns an esbuild plugin that resolves bare imports via the ESM CDN.
func (r *Resolver) Plugin() api.Plugin {
	r.ensureDefaults()

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

	// isUpstreamInternalPath detects absolute-path imports produced by esm.sh
	// that reference sub-modules on the same upstream (e.g. "/pkg@ver/deno/...",
	// "/node/process.mjs" for Node.js built-in polyfills).
	// Excludes local filesystem paths (they may also start with "/").
	isUpstreamInternalPath := func(path string) bool {
		if !strings.HasPrefix(path, "/") || strings.HasPrefix(path, "//") {
			return false
		}
		// Fast-path: esm.sh-style internal paths use a well-known directory
		// prefix (e.g. "/pkg@ver/deno/..."). Check this cheap string pattern
		// before falling back to os.Stat, avoiding a syscall for every
		// absolute path that does not match the pattern.
		parts := strings.SplitN(strings.TrimPrefix(path, "/"), "/", 2)
		if len(parts) == 0 {
			return false
		}
		first := parts[0]
		if !(strings.Contains(first, "@") || first == "node" || first == "stable" || isESMVersionPrefix(first)) {
			return false
		}
		// Exclude paths that exist as local files.
		if _, err := os.Stat(path); err == nil {
			return false
		}
		return true
	}

	return api.Plugin{
		Name: "choysum-esm-resolver",
		Setup: func(build api.PluginBuild) {
			build.OnResolve(api.OnResolveOptions{Filter: ".*"}, func(args api.OnResolveArgs) (api.OnResolveResult, error) {
				// Inside choysum-esm namespace: resolve relative and absolute
				// imports against the importer URL via standard URL resolution.
				if args.Namespace == "choysum-esm" {
					return r.resolveInNamespace(args)
				}

				// Rewrite upstream-internal absolute paths (e.g. "/pkg@ver/deno/...")
				// back to full esm.sh URLs so esbuild can continue resolving.
				if isUpstreamInternalPath(args.Path) {
					esmURL := r.upstream + args.Path
					return api.OnResolveResult{
						Path:      esmURL,
						Namespace: "choysum-esm",
					}, nil
				}

				if !isBareImport(args.Path) {
					return api.OnResolveResult{}, nil
				}

				// Apply lockfile version pinning to the specifier.
				spec, lockErr := r.lockedSpecifier(args.Path)
				if lockErr != nil {
					r.metrics.Errors.Add(1)
					return api.OnResolveResult{}, r.formatError("lockfile error", args.Path, r.effectiveLockfilePath(), lockErr.Error())
				}
				spec = rewriteProductionSpecifier(spec)

				// CSS imports from any target are external.
				if args.Kind == api.ResolveCSSURLToken {
					resolvedURL := fmt.Sprintf("%s/%s?target=%s", r.upstream, spec, r.target)
					return api.OnResolveResult{
						Path:     resolvedURL,
						External: true,
					}, nil
				}

				// Map bare import to esm.sh URL.
				esmURL := fmt.Sprintf("%s/%s?target=%s", r.upstream, spec, r.target)
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
				metrics := r.metrics

				// Try cache first, verify integrity if metadata present.
				if content, ok := r.readCache(cacheFile); ok {
					if metrics != nil {
						metrics.CacheHit.Add(1)
					}
					return api.OnLoadResult{
						Contents: ptr(content),
						Loader:   loaderForURL(url),
					}, nil
				}

				if metrics != nil {
					metrics.CacheMiss.Add(1)
				}

				// Offline mode: cache miss is a hard error.
				if r.offline {
					if metrics != nil {
						metrics.Errors.Add(1)
					}
					return api.OnLoadResult{}, r.formatError("cache miss (offline)", pkg, url,
						"run 'choysum install' with network access to populate the cache")
				}

				// Download with singleflight deduplication.
				type fetchResult struct {
					content string
					err     error
				}
				v, err, _ := r.singleflight.Do(cacheKey, func() (any, error) {
					downloadStart := time.Now()
					if metrics != nil {
						metrics.Downloads.Add(1)
					}
					content, dlErr := r.downloadWithRetry(url)
					if metrics != nil {
						metrics.DownloadDurationMs.Add(time.Since(downloadStart).Milliseconds())
					}
					if dlErr != nil {
						return fetchResult{}, dlErr
					}
					// Record the package name for downstream observability.
					if metrics != nil && pkg != "" {
						metrics.DownloadedPkgs.Store(pkg, struct{}{})
					}
					if writeErr := r.writeCache(cacheFile, []byte(content)); writeErr != nil {
						if r.logger != nil {
							r.logger.Warn("esm resolver: failed to write cache", "file", cacheFile, "error", writeErr)
						}
					}
					return fetchResult{content: content}, nil
				})
				if err != nil {
					if metrics != nil {
						metrics.Errors.Add(1)
					}
					return api.OnLoadResult{}, r.formatError("download failed", pkg, url, err.Error())
				}
				result := v.(fetchResult)

				return api.OnLoadResult{
					Contents: ptr(result.content),
					Loader:   loaderForURL(url),
				}, nil
			})

			// OnEnd: emit structured metrics summary.
			build.OnEnd(func(result *api.BuildResult) (api.OnEndResult, error) {
				if r.logger != nil && r.metrics != nil {
					hit, miss, downloads, errors, downloadMs := r.metrics.Snapshot()
					attrs := []any{
						"upstream", r.upstream,
						"target", r.target,
					}
					if r.moduleName != "" {
						attrs = append(attrs, "module", r.moduleName)
					}
					if r.application != "" {
						attrs = append(attrs, "application", r.application)
					}
					attrs = append(attrs,
						"metric_scope", "cumulative",
						"cache_hit", hit,
						"cache_miss", miss,
						"downloads", downloads,
						"cumulative_download_duration_ms", downloadMs,
						"errors", errors,
					)
					if downloads > 0 || errors > 0 {
						r.logger.Info("esm resolver metrics", attrs...)
						downloadedPkgs := r.metrics.SnapshotDownloadedPkgs()
						if len(downloadedPkgs) > 0 {
							pkgs := downloadedPkgs
							limit := 20
							if len(pkgs) > limit {
								pkgs = pkgs[:limit]
							}
							r.logger.Debug("esm resolver downloaded packages",
								"packages_count", len(downloadedPkgs),
								"packages_limit", limit,
								"packages", pkgs,
							)
						}
					} else {
						r.logger.Debug("esm resolver metrics", attrs...)
					}
				}
				return api.OnEndResult{}, nil
			})
		},
	}
}

func (r *Resolver) effectiveLockfilePath() string {
	if strings.TrimSpace(r.lockfilePath) != "" {
		return r.lockfilePath
	}
	if strings.TrimSpace(r.modulePath) != "" {
		return filepath.Join(r.modulePath, "esm.lock")
	}
	return ""
}

// resolveLockfile returns the parsed lockfile if available, loading it on first
// call. Returns nil when no lockfile is configured or the lockfile does not
// exist. Returns an error when a configured lockfile cannot be parsed.
func (r *Resolver) resolveLockfile() (*EsmLockfile, error) {
	r.lockfileOnce.Do(func() {
		path := r.effectiveLockfilePath()
		if path == "" {
			return
		}
		lock, err := ReadLockfile(path)
		if err != nil {
			r.lockfileErr = err
			return
		}
		r.lockfile = lock
	})
	return r.lockfile, r.lockfileErr
}

// lockedSpecifier returns the version-pinned specifier if the lockfile has an
// entry for the given import path. Otherwise returns the original specifier.
func (r *Resolver) lockedSpecifier(specifier string) (string, error) {
	lock, err := r.resolveLockfile()
	if err != nil {
		return "", err
	}
	return LookupLockedSpec(lock, specifier), nil
}

// rewriteProductionSpecifier rewrites selected package bare specifiers to
// explicit production entry files when upstream defaults are known to resolve
// to development variants.
func rewriteProductionSpecifier(specifier string) string {
	specifier = strings.TrimSpace(specifier)
	if specifier == "" {
		return specifier
	}

	suffixStart := len(specifier)
	if i := strings.Index(specifier, "?"); i >= 0 && i < suffixStart {
		suffixStart = i
	}
	if i := strings.Index(specifier, "#"); i >= 0 && i < suffixStart {
		suffixStart = i
	}
	core := specifier[:suffixStart]
	suffix := specifier[suffixStart:]

	base := core
	subpath := ""
	if slash := strings.Index(core, "/"); slash >= 0 {
		base = core[:slash]
		subpath = core[slash+1:]
	}

	if base != vueI18nBareSpecifier && !strings.HasPrefix(base, vueI18nBareSpecifier+"@") {
		return specifier
	}
	if strings.TrimSpace(subpath) != "" {
		return specifier
	}

	return base + "/" + vueI18nProdEntryPath + suffix
}

// resolveInNamespace handles import resolution for files already inside the
// choysum-esm namespace. Relative imports are resolved against the importer URL
// via standard URL resolution. Absolute HTTP(S) URLs and fragment-only
// specifiers are handled directly.
func (r *Resolver) resolveInNamespace(args api.OnResolveArgs) (api.OnResolveResult, error) {
	// Fragment-only specifiers (e.g. "#icon"): mark as external.
	if isFragmentOnly(args.Path) {
		return api.OnResolveResult{Path: args.Path, External: true}, nil
	}

	// Data URLs: pass through as external.
	if strings.HasPrefix(args.Path, "data:") {
		return api.OnResolveResult{Path: args.Path, External: true}, nil
	}

	// Absolute local filesystem paths (e.g. /Users/..., /home/...) should not
	// go through the ESM namespace. Let esbuild resolve them normally.
	if isLocalFilesystemPath(args.Path) {
		return api.OnResolveResult{}, nil
	}

	// Already an absolute HTTP(S) URL: resolve in namespace.
	if strings.HasPrefix(args.Path, "http://") || strings.HasPrefix(args.Path, "https://") {
		resolvedPath := trimCSSWrapperSuffix(args.Path)
		// CSS URL tokens in stylesheet content should remain external.
		if args.Kind == api.ResolveCSSURLToken {
			return api.OnResolveResult{Path: resolvedPath, External: true}, nil
		}
		return api.OnResolveResult{
			Path:      resolvedPath,
			Namespace: "choysum-esm",
		}, nil
	}

	// Resolve relative/absolute path against the importer URL.
	importerURL := stripNamespace(args.Importer)
	if importerURL == "" {
		return api.OnResolveResult{}, nil
	}

	base, err := url.Parse(importerURL)
	if err != nil {
		return api.OnResolveResult{}, fmt.Errorf("esm-resolver: invalid importer URL %q: %w", importerURL, err)
	}

	ref, err := url.Parse(args.Path)
	if err != nil {
		return api.OnResolveResult{}, fmt.Errorf("esm-resolver: invalid import path %q: %w", args.Path, err)
	}

	resolved := base.ResolveReference(ref)
	resolvedURL := resolved.String()

	// If the resolved URL is a local filesystem path, don't put it in namespace.
	if isLocalFilesystemPath(resolvedURL) {
		return api.OnResolveResult{}, nil
	}

	// Re-add the target parameter if it was present in the original
	// importer URL but got dropped during URL resolution.
	if parsedImporter, err := url.Parse(importerURL); err == nil {
		importerTarget := strings.TrimSpace(parsedImporter.Query().Get("target"))
		if importerTarget != "" {
			if parsedResolved, parseErr := url.Parse(resolvedURL); parseErr == nil {
				resolvedQuery := parsedResolved.Query()
				if strings.TrimSpace(resolvedQuery.Get("target")) == "" {
					resolvedQuery.Set("target", importerTarget)
					parsedResolved.RawQuery = resolvedQuery.Encode()
					resolvedURL = parsedResolved.String()
				}
			}
		}
	}

	resolvedURL = trimCSSWrapperSuffix(resolvedURL)

	// CSS URL tokens in stylesheet content should remain external.
	if args.Kind == api.ResolveCSSURLToken {
		return api.OnResolveResult{Path: resolvedURL, External: true}, nil
	}

	return api.OnResolveResult{
		Path:      resolvedURL,
		Namespace: "choysum-esm",
	}, nil
}

func hasCSSSuffix(path string) bool {
	return strings.HasSuffix(path, ".css") ||
		strings.HasSuffix(path, ".css.js") ||
		strings.HasSuffix(path, ".css.mjs") ||
		strings.HasSuffix(path, ".css.ts")
}

// trimCSSWrapperSuffix rewrites esm.sh CSS wrapper modules to their raw CSS URL
// by removing the trailing JS/MJS/TS extension (e.g. .css.js -> .css).
func trimCSSWrapperSuffix(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}

	trimExt := func(ext string) {
		u.Path = u.Path[:len(u.Path)-len(ext)]
		if u.RawPath == "" {
			return
		}
		lowerRawPath := strings.ToLower(u.RawPath)
		if strings.HasSuffix(lowerRawPath, ext) && len(u.RawPath) >= len(ext) {
			u.RawPath = u.RawPath[:len(u.RawPath)-len(ext)]
			return
		}
		// If RawPath cannot be synchronized safely, clear it to avoid path/encoding mismatch.
		u.RawPath = ""
	}

	lowerPath := strings.ToLower(u.Path)
	switch {
	case strings.HasSuffix(lowerPath, ".css.js"):
		trimExt(".js")
	case strings.HasSuffix(lowerPath, ".css.mjs"):
		trimExt(".mjs")
	case strings.HasSuffix(lowerPath, ".css.ts"):
		trimExt(".ts")
	default:
		return rawURL
	}

	return u.String()
}

// isLocalFilesystemPath reports whether path looks like an absolute local
// filesystem path rather than a remote ESM URL or upstream-internal path.
func isLocalFilesystemPath(path string) bool {
	if filepath.VolumeName(path) != "" {
		return true
	}
	if len(path) >= 3 {
		drive := path[0]
		if ((drive >= 'a' && drive <= 'z') || (drive >= 'A' && drive <= 'Z')) && path[1] == ':' && (path[2] == '/' || path[2] == '\\') {
			return true
		}
	}
	if !strings.HasPrefix(path, "/") || strings.HasPrefix(path, "//") {
		return false
	}
	first := strings.SplitN(strings.TrimPrefix(path, "/"), "/", 2)[0]
	// Check if the file exists on disk (with or without extension).
	if _, err := os.Stat(path); err == nil {
		return true
	}
	// Common UNIX filesystem root directories indicate a local path.
	localRoots := []string{"Users", "home", "var", "tmp", "etc", "usr", "opt",
		"Applications", "Library", "System", "Volumes", "private", "dev", "proc", "sys", "root", "sbin", "bin", "srv", "mnt", "media"}
	for _, root := range localRoots {
		if first == root {
			return true
		}
	}
	return false
}

// isFragmentOnly reports whether path is a fragment-only specifier like "#icon".
func isFragmentOnly(path string) bool {
	path = strings.TrimSpace(path)
	if path == "" || !strings.HasPrefix(path, "#") {
		return false
	}
	// Must have only a fragment, no scheme, host, or path.
	parsed, err := url.Parse(path)
	if err != nil {
		return false
	}
	return parsed.Scheme == "" && parsed.Host == "" && parsed.Path == "" && parsed.Opaque == "" && parsed.RawQuery == "" && parsed.Fragment != ""
}

func isESMVersionPrefix(segment string) bool {
	if len(segment) < 2 || segment[0] != 'v' {
		return false
	}
	for i := 1; i < len(segment); i++ {
		if segment[i] < '0' || segment[i] > '9' {
			return false
		}
	}
	return true
}

// loaderForURL returns the esbuild loader appropriate for a resolved ESM URL.
func loaderForURL(rawURL string) api.Loader {
	lower := strings.ToLower(rawURL)
	// Strip query string and fragment for extension detection.
	if idx := strings.Index(lower, "?"); idx >= 0 {
		lower = lower[:idx]
	}
	if idx := strings.Index(lower, "#"); idx >= 0 {
		lower = lower[:idx]
	}
	if hasCSSSuffix(lower) {
		return api.LoaderCSS
	}
	if strings.HasSuffix(lower, ".ts") || strings.HasSuffix(lower, ".tsx") || strings.HasSuffix(lower, ".mts") {
		return api.LoaderTS
	}
	return api.LoaderJS
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
	actual := sha512Hex(content)
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
		// Fall back to the default Choysum path when no cache dir is configured.
		if defaultPath, err := config.ResolveDefaultChoysumPaths(); err == nil {
			return filepath.Join(defaultPath, "pkg", "esm")
		}
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

	content, err := io.ReadAll(io.LimitReader(resp.Body, maxResolverDownloadBytes+1))
	if err != nil {
		return "", fmt.Errorf("read body: %w", err)
	}
	if int64(len(content)) > maxResolverDownloadBytes {
		return "", fmt.Errorf("read body: response too large (>%d bytes)", maxResolverDownloadBytes)
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
	return errors.As(err, target)
}

func (r *Resolver) writeCache(cacheFile string, data []byte) error {
	dir := filepath.Dir(cacheFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	tmpFile, err := os.CreateTemp(dir, filepath.Base(cacheFile)+"-*.tmp")
	if err != nil {
		return err
	}
	integrityTmp, err := os.CreateTemp(dir, filepath.Base(cacheFile)+".integrity-*.tmp")
	if err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpFile.Name())
		return err
	}
	tmpPath := tmpFile.Name()
	integrityTmpPath := integrityTmp.Name()
	cleanupCacheTmp := true
	cleanupIntegrityTmp := true
	defer func() {
		if tmpFile != nil {
			_ = tmpFile.Close()
		}
		if integrityTmp != nil {
			_ = integrityTmp.Close()
		}
		if cleanupCacheTmp {
			_ = os.Remove(tmpPath)
		}
		if cleanupIntegrityTmp {
			_ = os.Remove(integrityTmpPath)
		}
	}()
	if _, err := tmpFile.Write(data); err != nil {
		return err
	}
	if err := tmpFile.Close(); err != nil {
		tmpFile = nil
		return err
	}
	tmpFile = nil

	hash := sha512Hex(data)
	if _, err := integrityTmp.Write([]byte(hash)); err != nil {
		return err
	}
	if err := integrityTmp.Close(); err != nil {
		integrityTmp = nil
		return err
	}
	integrityTmp = nil

	if err := os.Rename(tmpPath, cacheFile); err != nil {
		return err
	}
	cleanupCacheTmp = false

	integrityFile := cacheFile + ".integrity"
	if err := os.Rename(integrityTmpPath, integrityFile); err != nil {
		_ = os.Remove(cacheFile)
		return err
	}
	cleanupIntegrityTmp = false
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

// stripNamespace removes the "choysum-esm:" prefix from a path.
func stripNamespace(path string) string {
	if idx := strings.Index(path, "://"); idx > 0 {
		if schemeEnd := strings.LastIndex(path[:idx], ":"); schemeEnd > 0 {
			return path[schemeEnd+1:]
		}
	}
	return path
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

func sha512Hex(b []byte) string {
	h := sha512.Sum512(b)
	return hex.EncodeToString(h[:])
}

func ptr[T any](v T) *T {
	return &v
}
