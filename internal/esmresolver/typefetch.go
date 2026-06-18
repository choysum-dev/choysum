// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package esmresolver

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	tsast "github.com/buke/typescript-go-internal/pkg/ast"
	tscore "github.com/buke/typescript-go-internal/pkg/core"
	tsparser "github.com/buke/typescript-go-internal/pkg/parser"
)

// TypeFetchResult holds the outcome of a type fetch operation.
type TypeFetchResult struct {
	Package    string
	Version    string
	CachedPath string
	FromCache  bool
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
		client = &http.Client{Timeout: 30 * time.Second}
	}

	// Check cache first.
	cacheFile := typesCachePath(typesDir, pkg, version)
	if _, err := os.Stat(cacheFile); err == nil {
		return &TypeFetchResult{Package: pkg, Version: version, CachedPath: cacheFile, FromCache: true}, nil, nil
	}

	// Step 1: HEAD request to discover the types URL.
	spec := pkg + "@" + version
	discoverURL := fmt.Sprintf("%s/%s?dts", strings.TrimRight(upstream, "/"), spec)

	req, err := http.NewRequest(http.MethodHead, discoverURL, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("create discover request: %w", err)
	}
	resp, err := client.Do(req)
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
	mainResult, transitive, err := fetchTypeRecursive(client, typesDir, typesURL, pkg, version, nil)
	if err != nil {
		return nil, nil, err
	}
	return mainResult, transitive, nil
}

// fetchTypeRecursive downloads a .d.ts file, parses its imports/references,
// and recursively fetches transitive type dependencies.
func fetchTypeRecursive(client *http.Client, typesDir, typesURL, rootPkg, rootVersion string, visited map[string]bool) (*TypeFetchResult, []TypeFetchResult, error) {
	if visited == nil {
		visited = make(map[string]bool)
	}
	// Normalize URL to avoid redundant fetches.
	normalized := strings.TrimRight(typesURL, "/")
	if visited[normalized] {
		return nil, nil, nil
	}
	visited[normalized] = true

	// Derive a cache key from the URL.
	cacheFile := typeCachePathForURL(typesDir, normalized)

	// Check cache.
	var content []byte
	if data, err := os.ReadFile(cacheFile); err == nil {
		content = data
	} else {
		req, err := http.NewRequest(http.MethodGet, normalized, nil)
		if err != nil {
			return nil, nil, fmt.Errorf("create download request for %s: %w", normalized, err)
		}
		resp, err := client.Do(req)
		if err != nil {
			return nil, nil, fmt.Errorf("download types from %s: %w", normalized, err)
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, nil, fmt.Errorf("read types body from %s: %w", normalized, err)
		}
		if resp.StatusCode != http.StatusOK {
			return nil, nil, fmt.Errorf("download types from %s: http %d", normalized, resp.StatusCode)
		}
		content = body

		// Write to cache atomically.
		if err := os.MkdirAll(filepath.Dir(cacheFile), 0755); err != nil {
			return nil, nil, fmt.Errorf("create types cache dir: %w", err)
		}
		tmpFile := cacheFile + ".tmp"
		if err := os.WriteFile(tmpFile, content, 0644); err != nil {
			return nil, nil, fmt.Errorf("write types tmp: %w", err)
		}
		if err := os.Rename(tmpFile, cacheFile); err != nil {
			return nil, nil, fmt.Errorf("rename types tmp: %w", err)
		}
	}

	// Parse imports and recursively fetch.
	imports := parseDTSImports(string(content))
	var allTransitive []TypeFetchResult

	for _, importPath := range imports {
		resolvedURL, err := resolveTypeImport(normalized, importPath)
		if err != nil {
			continue
		}
		// Derive package name from the resolved URL path.
		pkgName := typePkgNameFromURL(resolvedURL)
		_, subTransitive, err := fetchTypeRecursive(client, typesDir, resolvedURL, pkgName, "", visited)
		if err != nil {
			continue
		}
		allTransitive = append(allTransitive, subTransitive...)
	}

	result := &TypeFetchResult{
		Package:    rootPkg,
		Version:    rootVersion,
		CachedPath: cacheFile,
		FromCache:  false,
	}
	return result, allTransitive, nil
}

// parseDTSImports uses the TypeScript AST parser to extract module specifiers
// from import declarations and /// <reference> directives in .d.ts content.
func parseDTSImports(content string) []string {
	var paths []string
	seen := make(map[string]bool)

	// Use an absolute virtual path so the parser accepts it.
	fname := "/virtual/type.d.ts"
	scriptKind := tscore.GetScriptKindFromFileName(fname)
	source := tsparser.ParseSourceFile(tsast.SourceFileParseOptions{
		FileName: fname,
	}, content, scriptKind)

	if source == nil {
		return paths
	}

	// Walk statements looking for import declarations.
	for _, stmt := range source.Statements.Nodes {
		if stmt == nil {
			continue
		}
		if stmt.Kind == tsast.KindImportDeclaration || stmt.Kind == tsast.KindJSImportDeclaration {
			decl := stmt.AsImportDeclaration()
			if decl != nil && decl.ModuleSpecifier != nil {
				p := strings.Trim(decl.ModuleSpecifier.Text(), `"'`)
				if p != "" && !seen[p] && !strings.HasPrefix(p, "node:") {
					seen[p] = true
					paths = append(paths, p)
				}
			}
		}
		// Also catch import foo = require("...") — KindImportEqualsDeclaration.
		if stmt.Kind == tsast.KindImportEqualsDeclaration {
			decl := stmt.AsImportEqualsDeclaration()
			if decl != nil && decl.ModuleReference != nil {
				if ref := decl.ModuleReference.AsExternalModuleReference(); ref != nil && ref.Expression != nil {
					p := strings.Trim(ref.Expression.Text(), `"'`)
					if p != "" && !seen[p] {
						seen[p] = true
						paths = append(paths, p)
					}
				}
			}
		}
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
				if !seen[p] {
					seen[p] = true
					paths = append(paths, p)
				}
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
	// Absolute URLs pass through.
	if strings.HasPrefix(importPath, "http://") || strings.HasPrefix(importPath, "https://") {
		return importPath, nil
	}
	// Relative paths: resolve against the base URL's directory.
	base := strings.TrimRight(baseURL, "/")
	// Remove the filename from the base URL to get the directory.
	if idx := strings.LastIndex(base, "/"); idx > 0 && idx > len("https://") {
		base = base[:idx]
	}
	// Handle ../ and ./
	for strings.HasPrefix(importPath, "../") {
		if idx := strings.LastIndex(base, "/"); idx > len("https://") {
			base = base[:idx]
		}
		importPath = importPath[3:]
	}
	importPath = strings.TrimPrefix(importPath, "./")
	return base + "/" + importPath, nil
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
	pkgPath := filepath.Join(moduleDir, "package.json")
	pkg, err := ReadPackageJSON(pkgPath)
	if err != nil {
		return nil, err
	}

	deps := pkg.CollectDependencies()
	if len(deps) == 0 {
		return nil, nil
	}

	var results []TypeFetchResult
	for name, verRange := range deps {
		version := strings.TrimLeft(verRange, "^~=> ")
		if version == "" || version == "*" {
			continue
		}

		result, transitive, err := FetchTypeDefinition(client, upstream, typesDir, name, version)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[esm-type-fetch] warn: %s@%s: %v\n", name, version, err)
			continue
		}
		if result != nil {
			results = append(results, *result)
		}
		results = append(results, transitive...)
	}
	return results, nil
}

func typesCachePath(typesDir, pkg, version string) string {
	return filepath.Join(typesDir, fmt.Sprintf("%s@%s.d.ts", pkg, version))
}

// UpdateTsconfigPaths reads the tsconfig at the given path, adds or updates
// "paths" entries for each fetched type definition, and writes it back.
// The paths map package names (e.g. "vue") to their cached .d.ts file,
// relative to the directory containing the tsconfig.
func UpdateTsconfigPaths(tsconfigPath string, results []TypeFetchResult) error {
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
	if err := json.Unmarshal(data, &tsconfig); err != nil {
		return fmt.Errorf("parse tsconfig: %w", err)
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

	for _, r := range results {
		if r.CachedPath == "" {
			continue
		}
		// Compute relative path from tsconfig dir to the cached .d.ts file.
		relPath, err := filepath.Rel(tsconfigDir, r.CachedPath)
		if err != nil {
			continue
		}
		relPath = filepath.ToSlash(relPath)

		// Add a paths entry for the bare specifier → cached type.
		key := r.Package
		paths[key] = []string{relPath}
	}

	// Write back with indentation.
	out, err := json.MarshalIndent(tsconfig, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal tsconfig: %w", err)
	}
	out = append(out, '\n')

	if err := os.WriteFile(tsconfigPath, out, 0644); err != nil {
		return fmt.Errorf("write tsconfig: %w", err)
	}
	return nil
}
