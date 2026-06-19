// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package esmresolver

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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
	fromCache := false
	var imports []string
	if data, err := os.ReadFile(cacheFile); err == nil {
		content = data
		fromCache = true

		imports = parseDTSImports(string(content))
		if hasMissingLocalCachedImports(cacheFile, imports) {
			body, err := downloadTypeContent(client, normalized)
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
		body, err := downloadTypeContent(client, normalized)
		if err != nil {
			return nil, nil, err
		}
		content = body

		if err := writeTypeCacheFile(cacheFile, content); err != nil {
			return nil, nil, err
		}
		imports = parseDTSImports(string(content))
	}

	if imports == nil {
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
	if rewritten != string(content) {
		content = []byte(rewritten)
		if err := writeTypeCacheFile(cacheFile, content); err != nil {
			return nil, nil, err
		}
	}

	var allTransitive []TypeFetchResult

	for _, imp := range resolvedImports {
		resolvedURL := imp.ResolvedURL
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
		FromCache:  fromCache,
	}
	return result, allTransitive, nil
}

type resolvedTypeImport struct {
	Original    string
	ResolvedURL string
}

func rewriteTypeImportSpecifiers(content, cacheFile, typesDir string, imports []resolvedTypeImport) string {
	if len(imports) == 0 {
		return content
	}

	out := content
	for _, imp := range imports {
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

		out = strings.ReplaceAll(out, `"`+imp.Original+`"`, `"`+rel+`"`)
		out = strings.ReplaceAll(out, `'`+imp.Original+`'`, `'`+rel+`'`)
	}

	return out
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

		// Catch re-export clauses like `export * from "./foo.d.ts"`.
		if stmt.Kind == tsast.KindExportDeclaration {
			decl := stmt.AsExportDeclaration()
			if decl != nil && decl.ModuleSpecifier != nil {
				p := strings.Trim(decl.ModuleSpecifier.Text(), `"'`)
				if p != "" && !seen[p] && !strings.HasPrefix(p, "node:") {
					seen[p] = true
					paths = append(paths, p)
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

func writeTypeCacheFile(cacheFile string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(cacheFile), 0755); err != nil {
		return fmt.Errorf("create types cache dir: %w", err)
	}
	tmpFile := cacheFile + ".tmp"
	if err := os.WriteFile(tmpFile, content, 0644); err != nil {
		return fmt.Errorf("write types tmp: %w", err)
	}
	if err := os.Rename(tmpFile, cacheFile); err != nil {
		return fmt.Errorf("rename types tmp: %w", err)
	}
	return nil
}

func downloadTypeContent(client *http.Client, rawURL string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create download request for %s: %w", rawURL, err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download types from %s: %w", rawURL, err)
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("read types body from %s: %w", rawURL, err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download types from %s: http %d", rawURL, resp.StatusCode)
	}

	return body, nil
}

func hasMissingLocalCachedImports(cacheFile string, imports []string) bool {
	baseDir := filepath.Clean(filepath.Dir(cacheFile))
	for _, importPath := range imports {
		if !isLocalCachedTypeSpecifier(importPath) {
			continue
		}

		candidate := filepath.Clean(filepath.Join(baseDir, filepath.FromSlash(importPath)))
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

	if err := os.WriteFile(tsconfigPath, out, 0644); err != nil {
		return fmt.Errorf("write tsconfig: %w", err)
	}
	return nil
}
