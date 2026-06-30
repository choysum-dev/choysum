// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package esmresolver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	tsast "github.com/buke/typescript-go-internal/pkg/ast"
	tscore "github.com/buke/typescript-go-internal/pkg/core"
	tsparser "github.com/buke/typescript-go-internal/pkg/parser"
	logutil "github.com/choysum-dev/choysum/internal/logger"
)

// TypeFetchResult holds the outcome of a type fetch operation.
type TypeFetchResult struct {
	Package    string
	Version    string
	CachedPath string
	FromCache  bool
}

// TypeFetchModuleStats summarizes one module fetch run.
//
// Direct* fields are counts over direct dependency targets from package.json.
// Transitive* fields are counts over recursively fetched/imported type files.
type TypeFetchModuleStats struct {
	DirectTargets     int
	DirectCached      int
	DirectFetched     int
	DirectReused      int
	DirectFailed      int
	TransitiveCached  int
	TransitiveFetched int
}

const defaultTypeFetchRequestTimeout = 30 * time.Second
const defaultTypeFetchParallelism = 16
const maxTypeFetchDownloadBytes int64 = 10 * 1024 * 1024
const defaultTypeFetchTransitiveRetryAttempts = 2

var typeFetchRequestTimeout = defaultTypeFetchRequestTimeout

type visitEntry struct {
	ch      chan struct{}
	success bool
}

type typeFetchState struct {
	requestSem chan struct{}
	visitedMu  sync.Mutex
	visited    map[string]*visitEntry
}

func writeTypeFetchProgressLine(message string) {
	message = strings.TrimSpace(message)
	if message == "" {
		return
	}
	line := logutil.NewProgressLine(os.Stderr)
	if line != nil && line.IsTTY() {
		line.Done("", message)
		return
	}
	fmt.Fprintln(os.Stderr, message)
}

func beginTypeFetchTransitiveProgress(ctx context.Context, rootPkg string, total int, topLevel bool) (func(int), func()) {
	if total < 200 || !topLevel {
		return func(int) {}, func() {}
	}
	if ctx == nil {
		ctx = context.Background()
	}
	rootPkg = strings.TrimSpace(rootPkg)
	if rootPkg == "" {
		rootPkg = "unknown"
	}
	messageFor := func(completed int) string {
		return fmt.Sprintf("%s: fetching transitive types (%d/%d)", rootPkg, completed, total)
	}

	ticker := logutil.ProgressTickerFromContext(ctx)
	ownsTicker := false
	if ticker == nil {
		progressLine := logutil.NewProgressLine(os.Stderr)
		if progressLine != nil && progressLine.IsTTY() {
			ticker = logutil.NewProgressTicker(progressLine, logutil.ProgressTickerOptions{Interval: 120 * time.Millisecond})
			ownsTicker = true
		}
	}
	if ticker != nil {
		ticker.SetMessage(messageFor(0))
		return func(completed int) {
				ticker.SetMessage(messageFor(completed))
			}, func() {
				if !ownsTicker {
					return
				}
				ticker.Clear()
				ticker.Stop()
			}
	}

	writeTypeFetchProgressLine(fmt.Sprintf("[esm-type-fetch] info: %s has %d transitive type imports", rootPkg, total))
	return func(completed int) {
		if completed > 0 && completed%200 == 0 {
			writeTypeFetchProgressLine(fmt.Sprintf("[esm-type-fetch] info: %s transitive progress %d/%d", rootPkg, completed, total))
		}
	}, func() {}
}

// TypeFetchSession shares fetch state across multiple module fetch calls.
// Reusing one session avoids redundant transitive traversals in a single run.
// Calls are serialized per session to avoid cross-call wait cycles when the
// same session is shared by multiple goroutines.
type TypeFetchSession struct {
	mu    sync.Mutex
	state *typeFetchState
}

func NewTypeFetchSession(parallelism int) *TypeFetchSession {
	return &TypeFetchSession{state: newTypeFetchState(parallelism)}
}

func (s *TypeFetchSession) FetchTypesForModule(ctx context.Context, client *http.Client, upstream, typesDir, moduleDir string) ([]TypeFetchResult, error) {
	results, _, err := s.FetchTypesForModuleWithStats(ctx, client, upstream, typesDir, moduleDir)
	return results, err
}

func (s *TypeFetchSession) FetchTypesForModuleWithStats(ctx context.Context, client *http.Client, upstream, typesDir, moduleDir string) ([]TypeFetchResult, TypeFetchModuleStats, error) {
	if s == nil {
		return fetchTypesForModuleWithStateAndStats(ctx, client, upstream, typesDir, moduleDir, nil)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state == nil {
		return fetchTypesForModuleWithStateAndStats(ctx, client, upstream, typesDir, moduleDir, nil)
	}
	return fetchTypesForModuleWithStateAndStats(ctx, client, upstream, typesDir, moduleDir, s.state)
}

func newTypeFetchState(parallelism int) *typeFetchState {
	if parallelism <= 0 {
		parallelism = defaultTypeFetchParallelism
	}

	return &typeFetchState{
		requestSem: make(chan struct{}, parallelism),
		visited:    make(map[string]*visitEntry),
	}
}

func (s *typeFetchState) acquireVisit(url string) (bool, func(bool)) {
	if s == nil {
		return true, func(bool) {}
	}

	s.visitedMu.Lock()
	entry, ok := s.visited[url]
	if ok {
		s.visitedMu.Unlock()
		// Another goroutine is already fetching this URL.
		// Skip instead of blocking, otherwise concurrent sibling
		// traversals that share transitive dependencies can deadlock:
		// goroutine A waits for B via acquireVisit, while B waits
		// for A's child goroutine via WaitGroup.
		// The other goroutine will populate the cache; subsequent
		// runs (or other top-level packages) will pick it up from
		// the cache without reaching acquireVisit.
		return false, func(bool) {}
	}

	entry = &visitEntry{ch: make(chan struct{})}
	s.visited[url] = entry
	s.visitedMu.Unlock()

	var once sync.Once
	return true, func(success bool) {
		once.Do(func() {
			s.visitedMu.Lock()
			entry.success = success
			if !success {
				delete(s.visited, url)
			}
			close(entry.ch)
			s.visitedMu.Unlock()
		})
	}
}

func (s *typeFetchState) withRequestSlot(run func() error) error {
	return s.withRequestSlotContext(context.Background(), run)
}

func (s *typeFetchState) withRequestSlotContext(ctx context.Context, run func() error) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if s == nil || s.requestSem == nil {
		return run()
	}

	select {
	case s.requestSem <- struct{}{}:
	case <-ctx.Done():
		return ctx.Err()
	}
	defer func() { <-s.requestSem }()
	return run()
}

// NewTypeFetchHTTPClient builds an HTTP client tuned for resilient type fetch.
func NewTypeFetchHTTPClient(timeout time.Duration) *http.Client {
	if timeout <= 0 {
		timeout = typeFetchRequestTimeout
	}

	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ResponseHeaderTimeout: 15 * time.Second,
	}

	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}
}

// PackageJSON represents the subset of package.json fields needed for
// dependency extraction.
type PackageJSON struct {
	Dependencies     map[string]string `json:"dependencies"`
	PeerDependencies map[string]string `json:"peerDependencies"`
}

// ReadPackageJSON reads and parses a package.json file.
func ReadPackageJSON(path string) (*PackageJSON, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read package.json: %w", err)
	}
	var pkg PackageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, fmt.Errorf("parse package.json: %w", err)
	}
	return &pkg, nil
}

// CollectDependencies extracts all dependency names from package.json.
// Merges dependencies and peerDependencies.
func (p *PackageJSON) CollectDependencies() map[string]string {
	deps := make(map[string]string)
	for name, ver := range p.Dependencies {
		deps[name] = ver
	}
	for name, ver := range p.PeerDependencies {
		if _, ok := deps[name]; !ok {
			deps[name] = ver
		}
	}
	return deps
}

// FetchTypeDefinition resolves the .d.ts URL for a package and downloads it
// to the type cache directory. Returns the cached file path and all transitive
// type dependencies that were fetched.
func FetchTypeDefinition(client *http.Client, upstream, typesDir, pkg, version string) (*TypeFetchResult, []TypeFetchResult, error) {
	if client == nil {
		client = NewTypeFetchHTTPClient(typeFetchRequestTimeout)
	}

	state := newTypeFetchState(defaultTypeFetchParallelism)
	return fetchTypeDefinitionWithState(context.Background(), client, upstream, typesDir, pkg, version, state)
}

func fetchTypeDefinitionWithState(ctx context.Context, client *http.Client, upstream, typesDir, pkg, version string, state *typeFetchState) (*TypeFetchResult, []TypeFetchResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if client == nil {
		client = NewTypeFetchHTTPClient(typeFetchRequestTimeout)
	}
	if state == nil {
		state = newTypeFetchState(defaultTypeFetchParallelism)
	}

	// Check cache first.
	cacheFile := typesCachePath(typesDir, pkg, version)
	if _, err := os.Stat(cacheFile); err == nil {
		return &TypeFetchResult{Package: pkg, Version: version, CachedPath: cacheFile, FromCache: true}, nil, nil
	}

	// Step 1: HEAD request to discover the types URL.
	spec := pkg + "@" + version
	discoverURL := fmt.Sprintf("%s/%s?dts", strings.TrimRight(upstream, "/"), spec)

	var resp *http.Response
	err := state.withRequestSlotContext(ctx, func() error {
		reqCtx, cancel := context.WithTimeout(ctx, typeFetchRequestTimeout)
		defer cancel()

		req, reqErr := http.NewRequestWithContext(reqCtx, http.MethodHead, discoverURL, nil)
		if reqErr != nil {
			return fmt.Errorf("create discover request: %w", reqErr)
		}

		var doErr error
		resp, doErr = client.Do(req)
		return doErr
	})
	if err != nil {
		return nil, nil, fmt.Errorf("discover types URL: %w", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("discover types URL: http %d", resp.StatusCode)
	}

	typesURL := resp.Header.Get("x-typescript-types")
	if typesURL == "" {
		return nil, nil, fmt.Errorf("no x-typescript-types header for %s", spec)
	}

	// Step 2: Recursively download the .d.ts file and its imports.
	mainResult, transitive, err := fetchTypeRecursive(ctx, client, typesDir, typesURL, pkg, version, state, nil)
	if err != nil {
		return nil, nil, err
	}
	return mainResult, transitive, nil
}

// fetchTypeRecursive downloads a .d.ts file, parses its imports/references,
// and recursively fetches transitive type dependencies.
func fetchTypeRecursive(ctx context.Context, client *http.Client, typesDir, typesURL, rootPkg, rootVersion string, state *typeFetchState, ancestors map[string]bool) (*TypeFetchResult, []TypeFetchResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	if state == nil {
		state = newTypeFetchState(defaultTypeFetchParallelism)
	}
	// Normalize URL to avoid redundant fetches.
	normalized := strings.TrimRight(typesURL, "/")
	if ancestors != nil && ancestors[normalized] {
		return nil, nil, nil
	}
	localAncestors := make(map[string]bool, len(ancestors)+1)
	for ancestor := range ancestors {
		localAncestors[ancestor] = true
	}
	localAncestors[normalized] = true

	shouldFetch, done := state.acquireVisit(normalized)
	if !shouldFetch {
		return nil, nil, nil
	}
	success := false
	defer func() { done(success) }()

	// Derive a cache key from the URL.
	cacheFile := typeCachePathForURL(typesDir, normalized)

	// Check cache.
	var content []byte
	fromCache := false
	var imports []string
	if data, err := os.ReadFile(cacheFile); err == nil {
		content = data
		fromCache = true

		imports = parseDTSImports(string(content))
		if hasMissingLocalCachedImports(cacheFile, imports) {
			body, err := downloadTypeContent(ctx, client, normalized, state)
			if err != nil {
				return nil, nil, fmt.Errorf("refresh corrupted cached types from %s: %w", normalized, err)
			}
			content = body
			fromCache = false

			if err := writeTypeCacheFile(cacheFile, content); err != nil {
				return nil, nil, err
			}
			imports = parseDTSImports(string(content))
		}
	} else {
		body, err := downloadTypeContent(ctx, client, normalized, state)
		if err != nil {
			return nil, nil, err
		}
		content = body

		if err := writeTypeCacheFile(cacheFile, content); err != nil {
			return nil, nil, err
		}
		imports = parseDTSImports(string(content))
	}

	// Parse imports/exports and resolve transitive type URLs.
	resolvedImports := make([]resolvedTypeImport, 0, len(imports))
	for _, importPath := range imports {
		if isLocalCachedTypeSpecifier(importPath) {
			continue
		}
		resolvedURL, err := resolveTypeImport(normalized, importPath)
		if err != nil {
			continue
		}
		resolvedImports = append(resolvedImports, resolvedTypeImport{Original: importPath, ResolvedURL: resolvedURL})
	}

	// Rewire import/export module specifiers to local cache paths so TypeScript
	// can follow the graph even when files are flattened into a shared cache dir.
	rewritten := rewriteTypeImportSpecifiers(string(content), cacheFile, typesDir, resolvedImports)
	rewritten = rewriteLocalCachedBridgeSpecifiers(rewritten)
	rewritten = rewriteTypeModuleAugmentationSpecifiers(rewritten)
	if rewritten != string(content) {
		content = []byte(rewritten)
		if err := writeTypeCacheFile(cacheFile, content); err != nil {
			return nil, nil, err
		}
	}
	if err := normalizeBridgeCachedTypeChildren(cacheFile, imports); err != nil {
		return nil, nil, err
	}

	// On cache hits, avoid repeatedly traversing massive transitive graphs.
	// The first cold fetch already materializes transitive entries; later runs
	// can safely reuse the cached root file for fast incremental behavior.
	if fromCache {
		result := &TypeFetchResult{
			Package:    rootPkg,
			Version:    rootVersion,
			CachedPath: cacheFile,
			FromCache:  true,
		}
		success = true
		return result, nil, nil
	}

	var allTransitive []TypeFetchResult
	total := len(resolvedImports)
	isTopLevelFetch := strings.TrimSpace(rootVersion) != ""
	updateTransitiveProgress, stopTransitiveProgress := beginTypeFetchTransitiveProgress(ctx, rootPkg, total, isTopLevelFetch)
	defer stopTransitiveProgress()

	// Process transitive imports concurrently so that the download semaphore
	// (withRequestSlot) is fully utilised. Without this, the recursive
	// traversal is depth-first sequential — only one download is in flight
	// at a time regardless of parallelism, turning a cold-cache run on a
	// package with thousands of transitive imports into an hour-long wait.
	if total > 0 {
		var (
			wg              sync.WaitGroup
			mu              sync.Mutex
			completed       int
			failedCount     int
			failedSample    string
			failedSampleErr error
		)
		for _, imp := range resolvedImports {
			if err := ctx.Err(); err != nil {
				return nil, nil, err
			}
			wg.Add(1)
			go func(imp resolvedTypeImport) {
				defer wg.Done()
				resolvedURL := imp.ResolvedURL
				pkgName := typePkgNameFromURL(resolvedURL)
				subTransitive, err := fetchTypeRecursiveWithRetry(ctx, client, typesDir, resolvedURL, pkgName, state, localAncestors, defaultTypeFetchTransitiveRetryAttempts)
				completedNow := 0
				mu.Lock()
				if err == nil {
					allTransitive = append(allTransitive, subTransitive...)
				} else {
					failedCount++
					if failedSample == "" {
						failedSample = resolvedURL
						failedSampleErr = err
					}
				}
				completed++
				completedNow = completed
				updateTransitiveProgress(completedNow)
				mu.Unlock()
			}(imp)
		}
		wg.Wait()

		if failedCount > 0 && isTopLevelFetch {
			message := fmt.Sprintf("[esm-type-fetch] warn: %s transitive fetch had %d/%d failures", rootPkg, failedCount, total)
			if failedSample != "" && failedSampleErr != nil {
				message = fmt.Sprintf("%s (example: %s: %v)", message, failedSample, failedSampleErr)
			}
			writeTypeFetchProgressLine(message)
		}
	}

	result := &TypeFetchResult{
		Package:    rootPkg,
		Version:    rootVersion,
		CachedPath: cacheFile,
		FromCache:  fromCache,
	}
	success = true
	return result, allTransitive, nil
}

func fetchTypeRecursiveWithRetry(ctx context.Context, client *http.Client, typesDir, resolvedURL, pkgName string, state *typeFetchState, ancestors map[string]bool, attempts int) ([]TypeFetchResult, error) {
	if attempts <= 0 {
		attempts = 1
	}

	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		_, subTransitive, err := fetchTypeRecursive(ctx, client, typesDir, resolvedURL, pkgName, "", state, ancestors)
		if err == nil {
			return subTransitive, nil
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("transitive fetch failed")
	}
	return nil, lastErr
}

type resolvedTypeImport struct {
	Original    string
	ResolvedURL string
}

type typeImportRewriteSpan struct {
	start int
	end   int
	value string
	quote byte
}

var typeModuleAugmentationURLPattern = regexp.MustCompile(`(?m)declare\s+module\s+(['"])(https?://esm\.sh/[^'"]+)(['"])`)

var typeModuleBridgePackages = map[string]struct{}{
	"moment": {},
	"pinia":  {},
	"vue":    {},
}

func bridgedBareSpecifierForTypeURL(rawURL string) string {
	pkg := esmTypeURLBarePackage(rawURL)
	if pkg == "" {
		return ""
	}
	if _, ok := typeModuleBridgePackages[pkg]; !ok {
		return ""
	}
	return pkg
}

func esmTypeURLBarePackage(rawURL string) string {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return ""
	}
	if parsed.Host != "esm.sh" {
		return ""
	}

	path := strings.Trim(parsed.Path, "/")
	if path == "" {
		return ""
	}

	segments := strings.Split(path, "/")
	if len(segments) == 0 {
		return ""
	}

	first, err := url.PathUnescape(segments[0])
	if err != nil {
		first = segments[0]
	}
	first = strings.TrimSpace(first)
	if first == "" {
		return ""
	}

	candidate := first
	if strings.HasPrefix(first, "@") {
		if strings.Contains(first, "/") {
			candidate = first
		} else if len(segments) > 1 {
			second, err := url.PathUnescape(segments[1])
			if err != nil {
				second = segments[1]
			}
			second = strings.TrimSpace(second)
			if second != "" {
				candidate = first + "/" + second
			}
		}
	}

	if at := strings.LastIndex(candidate, "@"); at > 0 {
		candidate = candidate[:at]
	}
	return strings.TrimSpace(candidate)
}

func rewriteTypeModuleAugmentationSpecifiers(content string) string {
	if strings.TrimSpace(content) == "" {
		return content
	}

	return typeModuleAugmentationURLPattern.ReplaceAllStringFunc(content, func(match string) string {
		sub := typeModuleAugmentationURLPattern.FindStringSubmatch(match)
		if len(sub) != 4 {
			return match
		}

		quote := sub[1]
		rawURL := sub[2]
		if sub[3] != quote {
			return match
		}
		bridged := bridgedBareSpecifierForTypeURL(rawURL)
		if bridged == "" {
			return match
		}

		oldValue := quote + rawURL + quote
		newValue := quote + bridged + quote
		return strings.Replace(match, oldValue, newValue, 1)
	})
}

func bridgedBareSpecifierForLocalCacheSpecifier(specifier string) string {
	trimmed := strings.TrimSpace(filepath.ToSlash(specifier))
	if trimmed == "" {
		return ""
	}
	base := filepath.Base(trimmed)
	if !strings.HasPrefix(base, "esm.sh_") {
		return ""
	}

	rest := strings.TrimPrefix(base, "esm.sh_")
	at := strings.Index(rest, "@")
	if at <= 0 {
		return ""
	}
	pkg := strings.TrimSpace(rest[:at])
	if pkg == "" {
		return ""
	}
	if _, ok := typeModuleBridgePackages[pkg]; !ok {
		return ""
	}
	return pkg
}

func rewriteLocalCachedBridgeSpecifiers(content string) string {
	spans := collectTypeImportRewriteSpans(content)
	if len(spans) == 0 {
		return content
	}

	sort.SliceStable(spans, func(i, j int) bool {
		return spans[i].start > spans[j].start
	})

	out := content
	for _, span := range spans {
		bridged := bridgedBareSpecifierForLocalCacheSpecifier(span.value)
		if bridged == "" {
			continue
		}
		if span.start < 0 || span.end > len(out) || span.start >= span.end {
			continue
		}
		quote := span.quote
		if quote == 0 {
			quote = '"'
		}
		replacement := string(quote) + bridged + string(quote)
		out = out[:span.start] + replacement + out[span.end:]
	}

	return out
}

func shouldNormalizeBridgeChildCacheFile(specifier string) bool {
	base := filepath.Base(filepath.ToSlash(strings.TrimSpace(specifier)))
	if base == "" {
		return false
	}
	for _, prefix := range []string{
		"esm.sh_moment-timezone@",
		"esm.sh_moment@",
		"esm.sh_pinia-plugin-persistedstate@",
		"esm.sh_pinia@",
		"esm.sh_vue-router@",
		"esm.sh_vue@",
	} {
		if strings.HasPrefix(base, prefix) {
			return true
		}
	}
	return false
}

func normalizeBridgeCachedTypeFile(filePath string, seen map[string]bool) error {
	cleanPath := filepath.Clean(filePath)
	if cleanPath == "" {
		return nil
	}
	if seen[cleanPath] {
		return nil
	}
	seen[cleanPath] = true

	contentBytes, err := os.ReadFile(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	content := string(contentBytes)

	rewritten := rewriteLocalCachedBridgeSpecifiers(content)
	rewritten = rewriteTypeModuleAugmentationSpecifiers(rewritten)
	if rewritten != content {
		if err := writeTypeCacheFile(cleanPath, []byte(rewritten)); err != nil {
			return err
		}
	}

	imports := parseDTSImports(rewritten)
	for _, importPath := range imports {
		if !isLocalCachedTypeSpecifier(importPath) || !shouldNormalizeBridgeChildCacheFile(importPath) {
			continue
		}
		childPath := filepath.Clean(filepath.Join(filepath.Dir(cleanPath), filepath.FromSlash(importPath)))
		if err := normalizeBridgeCachedTypeFile(childPath, seen); err != nil {
			return err
		}
	}

	return nil
}

func normalizeBridgeCachedTypeChildren(cacheFile string, imports []string) error {
	seen := map[string]bool{filepath.Clean(cacheFile): true}
	for _, importPath := range imports {
		if !isLocalCachedTypeSpecifier(importPath) || !shouldNormalizeBridgeChildCacheFile(importPath) {
			continue
		}
		childPath := filepath.Clean(filepath.Join(filepath.Dir(cacheFile), filepath.FromSlash(importPath)))
		if err := normalizeBridgeCachedTypeFile(childPath, seen); err != nil {
			return err
		}
	}
	return nil
}

func rewriteTypeImportSpecifiers(content, cacheFile, typesDir string, imports []resolvedTypeImport) string {
	if len(imports) == 0 {
		return content
	}

	rewriteMap := make(map[string]string, len(imports))
	for _, imp := range imports {
		if bridged := bridgedBareSpecifierForTypeURL(imp.ResolvedURL); bridged != "" {
			rewriteMap[imp.Original] = bridged
			continue
		}

		localPath := typeCachePathForURL(typesDir, imp.ResolvedURL)
		rel, err := filepath.Rel(filepath.Dir(cacheFile), localPath)
		if err != nil {
			continue
		}
		rel = filepath.ToSlash(rel)
		if rel == "" || rel == "." {
			continue
		}
		if !strings.HasPrefix(rel, ".") {
			rel = "./" + rel
		}
		rewriteMap[imp.Original] = rel
	}

	if len(rewriteMap) == 0 {
		return content
	}

	spans := collectTypeImportRewriteSpans(content)
	if len(spans) == 0 {
		return content
	}

	sort.SliceStable(spans, func(i, j int) bool {
		return spans[i].start > spans[j].start
	})

	out := content
	for _, span := range spans {
		rel, ok := rewriteMap[span.value]
		if !ok {
			continue
		}
		if span.start < 0 || span.end > len(out) || span.start >= span.end {
			continue
		}
		quote := span.quote
		if quote == 0 {
			quote = '"'
		}
		replacement := string(quote) + rel + string(quote)
		out = out[:span.start] + replacement + out[span.end:]
	}

	return out
}

func collectTypeImportRewriteSpans(content string) []typeImportRewriteSpan {
	spans := make([]typeImportRewriteSpan, 0)
	seen := make(map[string]bool)
	addSpan := func(spec *tsast.Expression) {
		if spec == nil {
			return
		}
		if spec.Kind != tsast.KindStringLiteral && spec.Kind != tsast.KindNoSubstitutionTemplateLiteral {
			return
		}
		start, end := spec.Pos(), spec.End()
		if start < 0 || end <= start || end > len(content) {
			return
		}

		raw := content[start:end]
		quote := byte(0)
		if len(raw) >= 2 {
			first := raw[0]
			if (first == '\'' || first == '"' || first == '`') && raw[len(raw)-1] == first {
				quote = first
			}
		}

		value := strings.TrimSpace(spec.Text())
		if value == "" && quote != 0 && len(raw) >= 2 {
			value = raw[1 : len(raw)-1]
		}
		if value == "" {
			return
		}

		key := fmt.Sprintf("%d:%d:%s", start, end, value)
		if seen[key] {
			return
		}
		seen[key] = true
		spans = append(spans, typeImportRewriteSpan{start: start, end: end, value: value, quote: quote})
	}

	collectImportTypeSpecifier := func(node *tsast.Node) {
		if node == nil || node.Kind != tsast.KindImportType {
			return
		}
		importType := node.AsImportTypeNode()
		if importType == nil || importType.Argument == nil || importType.Argument.Kind != tsast.KindLiteralType {
			return
		}
		litType := importType.Argument.AsLiteralTypeNode()
		if litType == nil || litType.Literal == nil {
			return
		}
		addSpan(litType.Literal)
	}

	fname := "/virtual/type.d.ts"
	scriptKind := tscore.GetScriptKindFromFileName(fname)
	source := tsparser.ParseSourceFile(tsast.SourceFileParseOptions{FileName: fname}, content, scriptKind)
	if source != nil && source.Statements != nil {
		var visit func(node *tsast.Node)
		visit = func(node *tsast.Node) {
			if node == nil {
				return
			}

			switch node.Kind {
			case tsast.KindImportDeclaration, tsast.KindJSImportDeclaration:
				if decl := node.AsImportDeclaration(); decl != nil {
					addSpan(decl.ModuleSpecifier)
				}
			case tsast.KindImportEqualsDeclaration:
				decl := node.AsImportEqualsDeclaration()
				if decl != nil && decl.ModuleReference != nil && decl.ModuleReference.Kind == tsast.KindExternalModuleReference {
					addSpan(decl.ModuleReference.AsExternalModuleReference().Expression)
				}
			case tsast.KindExportDeclaration:
				if decl := node.AsExportDeclaration(); decl != nil {
					addSpan(decl.ModuleSpecifier)
				}
			case tsast.KindCallExpression:
				call := node.AsCallExpression()
				if call != nil && call.Expression != nil && call.Expression.Kind == tsast.KindImportKeyword && call.Arguments != nil && len(call.Arguments.Nodes) > 0 {
					addSpan(call.Arguments.Nodes[0])
				}
			}

			collectImportTypeSpecifier(node)

			node.ForEachChild(func(child *tsast.Node) bool {
				visit(child)
				return false
			})
		}

		for _, stmt := range source.Statements.Nodes {
			visit(stmt)
		}
	}

	// /// <reference path="..." /> and /// <reference types="..." /> are
	// comment directives and not represented as AST import nodes.
	lineOffset := 0
	for _, chunk := range strings.SplitAfter(content, "\n") {
		line := strings.TrimSuffix(chunk, "\n")
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "///") {
			for _, attr := range []string{`path="`, `path='`, `types="`, `types='`} {
				idx := strings.Index(line, attr)
				if idx < 0 {
					continue
				}
				valueStartInLine := idx + len(attr)
				quote := attr[len(attr)-1]
				rest := line[valueStartInLine:]
				valueEndRel := strings.IndexByte(rest, quote)
				if valueEndRel < 0 {
					continue
				}
				value := strings.TrimSpace(rest[:valueEndRel])
				if value == "" {
					continue
				}

				start := lineOffset + valueStartInLine - 1
				end := lineOffset + valueStartInLine + valueEndRel + 1
				if start < 0 || end <= start || end > len(content) {
					continue
				}

				key := fmt.Sprintf("%d:%d:%s", start, end, value)
				if seen[key] {
					continue
				}
				seen[key] = true
				spans = append(spans, typeImportRewriteSpan{start: start, end: end, value: value, quote: quote})
			}
		}
		lineOffset += len(chunk)
	}

	return spans
}

func isLocalCachedTypeSpecifier(importPath string) bool {
	p := filepath.ToSlash(strings.TrimSpace(importPath))
	if p == "" {
		return false
	}
	base := filepath.Base(p)
	return strings.HasPrefix(base, "esm.sh_")
}

// parseDTSImports uses the TypeScript AST parser to extract module specifiers
// from import declarations and /// <reference> directives in .d.ts content.
func parseDTSImports(content string) []string {
	paths := make([]string, 0)
	seen := make(map[string]bool)
	addPath := func(raw string, skipNodeProtocol bool) {
		p := strings.Trim(raw, `"'`)
		p = strings.TrimSpace(p)
		if p == "" || seen[p] {
			return
		}
		if skipNodeProtocol && strings.HasPrefix(p, "node:") {
			return
		}
		seen[p] = true
		paths = append(paths, p)
	}

	// Use an absolute virtual path so the parser accepts it.
	fname := "/virtual/type.d.ts"
	scriptKind := tscore.GetScriptKindFromFileName(fname)
	source := tsparser.ParseSourceFile(tsast.SourceFileParseOptions{
		FileName: fname,
	}, content, scriptKind)

	if source == nil || source.Statements == nil {
		return paths
	}

	collectSpecifier := func(spec *tsast.Expression) {
		if spec == nil {
			return
		}
		if spec.Kind != tsast.KindStringLiteral && spec.Kind != tsast.KindNoSubstitutionTemplateLiteral {
			return
		}
		addPath(spec.Text(), true)
	}

	collectImportTypeSpecifier := func(node *tsast.Node) {
		if node == nil || node.Kind != tsast.KindImportType {
			return
		}
		importType := node.AsImportTypeNode()
		if importType == nil || importType.Argument == nil || importType.Argument.Kind != tsast.KindLiteralType {
			return
		}
		litType := importType.Argument.AsLiteralTypeNode()
		if litType == nil || litType.Literal == nil {
			return
		}
		if litType.Literal.Kind != tsast.KindStringLiteral && litType.Literal.Kind != tsast.KindNoSubstitutionTemplateLiteral {
			return
		}
		addPath(litType.Literal.Text(), true)
	}

	var visit func(node *tsast.Node)
	visit = func(node *tsast.Node) {
		if node == nil {
			return
		}

		switch node.Kind {
		case tsast.KindImportDeclaration, tsast.KindJSImportDeclaration:
			if decl := node.AsImportDeclaration(); decl != nil {
				collectSpecifier(decl.ModuleSpecifier)
			}
		case tsast.KindImportEqualsDeclaration:
			decl := node.AsImportEqualsDeclaration()
			if decl != nil && decl.ModuleReference != nil && decl.ModuleReference.Kind == tsast.KindExternalModuleReference {
				collectSpecifier(decl.ModuleReference.AsExternalModuleReference().Expression)
			}
		case tsast.KindExportDeclaration:
			if decl := node.AsExportDeclaration(); decl != nil {
				collectSpecifier(decl.ModuleSpecifier)
			}
		case tsast.KindCallExpression:
			call := node.AsCallExpression()
			if call != nil && call.Expression != nil && call.Expression.Kind == tsast.KindImportKeyword && call.Arguments != nil && len(call.Arguments.Nodes) > 0 {
				collectSpecifier(call.Arguments.Nodes[0])
			}
		}

		collectImportTypeSpecifier(node)

		node.ForEachChild(func(child *tsast.Node) bool {
			visit(child)
			return false
		})
	}

	for _, stmt := range source.Statements.Nodes {
		visit(stmt)
	}

	// Also parse /// <reference path="..." /> and /// <reference types="..." />
	// These are comments, not AST nodes, so we use simple string scanning.
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "///") {
			continue
		}
		for _, attr := range []string{`path="`, `path='`, `types="`, `types='`} {
			idx := strings.Index(line, attr)
			if idx < 0 {
				continue
			}
			start := idx + len(attr)
			quote := string(attr[len(attr)-1])
			if end := strings.Index(line[start:], quote); end >= 0 {
				p := line[start : start+end]
				addPath(p, false)
			}
		}
	}

	return paths
}

// resolveTypeImport resolves an import path against a base type URL.
func resolveTypeImport(baseURL, importPath string) (string, error) {
	if importPath == "" {
		return "", fmt.Errorf("empty import path")
	}
	importPath = strings.TrimSpace(importPath)

	// Absolute URLs pass through.
	if strings.HasPrefix(importPath, "http://") || strings.HasPrefix(importPath, "https://") {
		return importPath, nil
	}

	// Skip bare package/type library imports like "node".
	if !strings.HasPrefix(importPath, "./") && !strings.HasPrefix(importPath, "../") && !strings.HasPrefix(importPath, "/") {
		return "", fmt.Errorf("unsupported bare type import %q", importPath)
	}

	base, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("parse base type URL: %w", err)
	}
	ref, err := url.Parse(importPath)
	if err != nil {
		return "", fmt.Errorf("parse type import %q: %w", importPath, err)
	}
	return base.ResolveReference(ref).String(), nil
}

// typeCachePathForURL derives a cache file path from a type URL.
func typeCachePathForURL(typesDir, rawURL string) string {
	// Use a simple hash-like approach: replace special chars and truncate.
	safe := strings.NewReplacer(
		"https://", "", "http://", "", "/", "_", "?", "_", "&", "_", "=", "_",
	).Replace(rawURL)
	if len(safe) > 200 {
		safe = safe[:200]
	}
	return filepath.Join(typesDir, safe+".d.ts")
}

// typePkgNameFromURL extracts a human-readable package name from a type URL.
func typePkgNameFromURL(rawURL string) string {
	// Strip protocol and host.
	s := rawURL
	if idx := strings.Index(s, "://"); idx > 0 {
		s = s[idx+3:]
	}
	if idx := strings.Index(s, "/"); idx > 0 {
		s = s[idx+1:]
	}
	// Strip query string.
	if idx := strings.Index(s, "?"); idx > 0 {
		s = s[:idx]
	}
	return s
}

// FetchTypesForModule reads the module's package.json and fetches type
// definitions for all dependencies (including transitive imports).
// Returns all fetched type results (direct + transitive).
func FetchTypesForModule(client *http.Client, upstream, typesDir, moduleDir string) ([]TypeFetchResult, error) {
	results, _, err := FetchTypesForModuleWithStats(client, upstream, typesDir, moduleDir)
	return results, err
}

// FetchTypesForModuleWithStats fetches types and returns detailed module-level
// accounting (direct targets + transitive totals) in addition to fetch results.
func FetchTypesForModuleWithStats(client *http.Client, upstream, typesDir, moduleDir string) ([]TypeFetchResult, TypeFetchModuleStats, error) {
	return fetchTypesForModuleWithStateAndStats(context.Background(), client, upstream, typesDir, moduleDir, nil)
}

func fetchTypesForModuleWithState(ctx context.Context, client *http.Client, upstream, typesDir, moduleDir string, state *typeFetchState) ([]TypeFetchResult, error) {
	results, _, err := fetchTypesForModuleWithStateAndStats(ctx, client, upstream, typesDir, moduleDir, state)
	return results, err
}

func fetchTypesForModuleWithStateAndStats(ctx context.Context, client *http.Client, upstream, typesDir, moduleDir string, state *typeFetchState) ([]TypeFetchResult, TypeFetchModuleStats, error) {
	stats := TypeFetchModuleStats{}
	if ctx == nil {
		ctx = context.Background()
	}
	if client == nil {
		client = NewTypeFetchHTTPClient(defaultTypeFetchRequestTimeout)
	}
	if state == nil {
		state = newTypeFetchState(defaultTypeFetchParallelism)
	}

	pkgPath := filepath.Join(moduleDir, "package.json")
	pkg, err := ReadPackageJSON(pkgPath)
	if err != nil {
		return nil, stats, err
	}

	deps := pkg.CollectDependencies()
	if len(deps) == 0 {
		return nil, stats, nil
	}

	depNames := make([]string, 0, len(deps))
	for name := range deps {
		depNames = append(depNames, name)
	}
	sort.Strings(depNames)

	type dependencyFetchTarget struct {
		name    string
		version string
	}
	targets := make([]dependencyFetchTarget, 0, len(depNames))
	for _, name := range depNames {
		verRange := deps[name]
		if strings.HasPrefix(verRange, "workspace:") || strings.HasPrefix(verRange, "file:") || strings.HasPrefix(verRange, "link:") {
			continue
		}
		version := strings.TrimLeft(verRange, "^~=> ")
		if version == "" || version == "*" {
			continue
		}
		targets = append(targets, dependencyFetchTarget{name: name, version: version})
	}
	if len(targets) == 0 {
		return nil, stats, nil
	}
	stats.DirectTargets = len(targets)

	ticker := logutil.ProgressTickerFromContext(ctx)
	moduleName := strings.TrimSpace(filepath.Base(moduleDir))
	if moduleName == "" {
		moduleName = "module"
	}
	setModuleProgress := func(current int, pkg, version string) {
		if ticker == nil {
			return
		}
		message := fmt.Sprintf("[%s] fetching dependency types (%d/%d)", moduleName, current, len(targets))
		pkg = strings.TrimSpace(pkg)
		version = strings.TrimSpace(version)
		if pkg != "" {
			if version != "" {
				message = fmt.Sprintf("%s: %s@%s", message, pkg, version)
			} else {
				message = fmt.Sprintf("%s: %s", message, pkg)
			}
		}
		ticker.SetMessage(message)
	}
	setModuleProgress(0, "", "")

	var results []TypeFetchResult
	for idx, target := range targets {
		if err := ctx.Err(); err != nil {
			return nil, stats, err
		}
		name := target.name
		version := target.version

		setModuleProgress(idx+1, name, version)
		start := time.Now()
		result, transitive, err := fetchTypeDefinitionWithState(ctx, client, upstream, typesDir, name, version, state)
		if err != nil {
			stats.DirectFailed++
			if ctxErr := ctx.Err(); ctxErr != nil {
				return nil, stats, ctxErr
			}
			writeTypeFetchProgressLine(fmt.Sprintf("[esm-type-fetch] warn: [%s] %s@%s (%s): %v", moduleName, name, version, time.Since(start).Round(time.Millisecond), err))
			continue
		}
		if result != nil {
			results = append(results, *result)
			if result.FromCache {
				stats.DirectCached++
			} else {
				stats.DirectFetched++
			}
		} else {
			stats.DirectReused++
		}
		for _, tr := range transitive {
			if tr.FromCache {
				stats.TransitiveCached++
			} else {
				stats.TransitiveFetched++
			}
		}
		results = append(results, transitive...)
	}
	return results, stats, nil
}

func typesCachePath(typesDir, pkg, version string) string {
	return filepath.Join(typesDir, fmt.Sprintf("%s@%s.d.ts", pkg, version))
}

func writeTypeCacheFile(cacheFile string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(cacheFile), 0755); err != nil {
		return fmt.Errorf("create types cache dir: %w", err)
	}
	tmpFile := cacheFile + ".tmp"
	if err := os.WriteFile(tmpFile, content, 0644); err != nil {
		return fmt.Errorf("write types tmp: %w", err)
	}
	defer func() {
		_ = os.Remove(tmpFile)
	}()
	if err := os.Rename(tmpFile, cacheFile); err != nil {
		return fmt.Errorf("rename types tmp: %w", err)
	}
	return nil
}

func writeAtomicFile(filePath string, content []byte, perm os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return fmt.Errorf("create parent dir: %w", err)
	}
	tmpFile, err := os.CreateTemp(filepath.Dir(filePath), filepath.Base(filePath)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create tmp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	tmpClosed := false
	cleanup := true
	defer func() {
		if !tmpClosed {
			_ = tmpFile.Close()
		}
		if cleanup {
			_ = os.Remove(tmpPath)
		}
	}()

	if err := tmpFile.Chmod(perm); err != nil {
		return fmt.Errorf("chmod tmp file: %w", err)
	}
	if _, err := tmpFile.Write(content); err != nil {
		return fmt.Errorf("write tmp file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		tmpClosed = true
		return fmt.Errorf("close tmp file: %w", err)
	}
	tmpClosed = true
	if err := renameFileWithBackup(tmpPath, filePath); err != nil {
		return fmt.Errorf("rename tmp file: %w", err)
	}

	cleanup = false
	return nil
}

func downloadTypeContent(ctx context.Context, client *http.Client, rawURL string, state *typeFetchState) ([]byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	var resp *http.Response
	var reqCancel context.CancelFunc
	err := state.withRequestSlotContext(ctx, func() error {
		reqCtx, cancel := context.WithTimeout(ctx, typeFetchRequestTimeout)
		reqCancel = cancel

		req, reqErr := http.NewRequestWithContext(reqCtx, http.MethodGet, rawURL, nil)
		if reqErr != nil {
			return fmt.Errorf("create download request for %s: %w", rawURL, reqErr)
		}

		var doErr error
		resp, doErr = client.Do(req)
		return doErr
	})
	if err != nil {
		return nil, fmt.Errorf("download types from %s: %w", rawURL, err)
	}
	if reqCancel != nil {
		defer reqCancel()
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download types from %s: http %d", rawURL, resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxTypeFetchDownloadBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read types body from %s: %w", rawURL, err)
	}
	if int64(len(body)) > maxTypeFetchDownloadBytes {
		return nil, fmt.Errorf("download types from %s: response exceeds %d bytes", rawURL, maxTypeFetchDownloadBytes)
	}

	return body, nil
}

func hasMissingLocalCachedImports(cacheFile string, imports []string) bool {
	baseDir := filepath.Clean(filepath.Dir(cacheFile))
	for _, importPath := range imports {
		trimmed := strings.TrimSpace(importPath)
		if trimmed == "" {
			continue
		}

		// Recover from historical partial caches (for example demand-filtered runs)
		// where relative .d.ts imports were left unresolved in cached root files.
		isRelativeTypePath := strings.HasPrefix(trimmed, "./") || strings.HasPrefix(trimmed, "../")
		if !isLocalCachedTypeSpecifier(trimmed) && !isRelativeTypePath {
			continue
		}

		candidate := filepath.Clean(filepath.Join(baseDir, filepath.FromSlash(trimmed)))
		if candidate != baseDir && !strings.HasPrefix(candidate, baseDir+string(os.PathSeparator)) {
			continue
		}
		if _, err := os.Stat(candidate); err != nil {
			return true
		}
	}
	return false
}

// UpdateTsconfigPaths reads the tsconfig at the given path, adds or updates
// "paths" entries for each fetched type definition, and writes it back.
// The paths map package names (e.g. "vue") to their cached .d.ts file,
// relative to the directory containing the tsconfig.
func UpdateTsconfigPaths(tsconfigPath string, results []TypeFetchResult) error {
	if err := ensureModulesTsconfig(tsconfigPath); err != nil {
		return fmt.Errorf("ensure tsconfig: %w", err)
	}

	if len(results) == 0 {
		return nil
	}

	// Read existing tsconfig.
	data, err := os.ReadFile(tsconfigPath)
	if err != nil {
		return fmt.Errorf("read tsconfig: %w", err)
	}

	// Parse into a flexible map to preserve unknown fields.
	var tsconfig map[string]interface{}
	if len(data) == 0 || strings.TrimSpace(string(data)) == "" {
		tsconfig = make(map[string]interface{})
	} else if err := json.Unmarshal(data, &tsconfig); err != nil {
		return fmt.Errorf("parse tsconfig: %w", err)
	}
	if tsconfig == nil {
		tsconfig = make(map[string]interface{})
	}

	// Navigate to compilerOptions.paths.
	compilerOptions, ok := tsconfig["compilerOptions"].(map[string]interface{})
	if !ok {
		compilerOptions = make(map[string]interface{})
		tsconfig["compilerOptions"] = compilerOptions
	}

	paths, ok := compilerOptions["paths"].(map[string]interface{})
	if !ok {
		paths = make(map[string]interface{})
		compilerOptions["paths"] = paths
	}

	tsconfigDir := filepath.Dir(tsconfigPath)
	absDir, err := filepath.Abs(tsconfigDir)
	if err != nil {
		return fmt.Errorf("absolute tsconfig dir: %w", err)
	}
	tsconfigDir = absDir

	// Resolve the working directory once so that relative CachedPath
	// values can be absolutised without a per-item os.Getwd syscall.
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	for _, r := range results {
		if r.CachedPath == "" {
			continue
		}
		cachedPath := r.CachedPath
		if !filepath.IsAbs(cachedPath) {
			cachedPath = filepath.Join(wd, cachedPath)
		}
		// Compute relative path from tsconfig dir to the cached .d.ts file.
		relPath, err := filepath.Rel(tsconfigDir, cachedPath)
		if err != nil {
			continue
		}
		relPath = filepath.ToSlash(relPath)

		// Add a paths entry for the bare specifier to cached type.
		key := r.Package
		paths[key] = []string{relPath}
	}

	// Write back with indentation.
	out, err := json.MarshalIndent(tsconfig, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal tsconfig: %w", err)
	}
	out = append(out, '\n')

	if err := writeAtomicFile(tsconfigPath, out, 0644); err != nil {
		return fmt.Errorf("write tsconfig: %w", err)
	}
	return nil
}

func ensureModulesTsconfig(tsconfigPath string) error {
	st, err := os.Stat(tsconfigPath)
	if err == nil {
		if st.IsDir() {
			return fmt.Errorf("tsconfig path is a directory: %s", tsconfigPath)
		}
		return nil
	}
	if !os.IsNotExist(err) {
		return fmt.Errorf("stat tsconfig: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(tsconfigPath), 0o755); err != nil {
		return fmt.Errorf("create tsconfig dir: %w", err)
	}

	base := map[string]interface{}{
		"$schema": "https://json.schemastore.org/tsconfig",
		"compilerOptions": map[string]interface{}{
			"allowArbitraryExtensions":     true,
			"allowImportingTsExtensions":   true,
			"allowJs":                      true,
			"experimentalDecorators":       true,
			"lib":                          []string{"ES2020", "DOM", "DOM.Iterable"},
			"module":                       "ESNext",
			"moduleResolution":             "bundler",
			"noEmit":                       true,
			"paths":                        map[string]interface{}{"@/*": []string{"./*"}},
			"skipLibCheck":                 true,
			"strict":                       true,
			"strictPropertyInitialization": false,
			"target":                       "ES2020",
		},
		"exclude": []string{
			"**/*.test.ts",
			"**/*.test.tsx",
			"**/*.spec.ts",
			"**/*.spec.tsx",
			"**/__tests__/**",
			"**/tests/**",
			"**/e2e/**",
		},
		"display": "Recommended",
	}

	out, err := json.MarshalIndent(base, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal tsconfig: %w", err)
	}
	out = append(out, '\n')

	if err := writeAtomicFile(tsconfigPath, out, 0644); err != nil {
		return fmt.Errorf("write tsconfig: %w", err)
	}
	return nil
}
