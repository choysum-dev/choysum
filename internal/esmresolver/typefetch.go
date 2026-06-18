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
// to the type cache directory. Returns the cached file path.
func FetchTypeDefinition(client *http.Client, upstream, typesDir, pkg, version string) (*TypeFetchResult, error) {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}

	// Check cache first to avoid unnecessary network requests.
	cacheFile := typesCachePath(typesDir, pkg, version)
	if _, err := os.Stat(cacheFile); err == nil {
		return &TypeFetchResult{Package: pkg, Version: version, CachedPath: cacheFile, FromCache: true}, nil
	}

	// Step 1: HEAD request to discover the types URL via x-typescript-types header.
	spec := pkg + "@" + version
	discoverURL := fmt.Sprintf("%s/%s?dts", strings.TrimRight(upstream, "/"), spec)

	req, err := http.NewRequest(http.MethodHead, discoverURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create discover request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("discover types URL: %w", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("discover types URL: http %d", resp.StatusCode)
	}

	typesURL := resp.Header.Get("x-typescript-types")
	if typesURL == "" {
		return nil, fmt.Errorf("no x-typescript-types header for %s", spec)
	}

	// Step 2: Download the .d.ts file.
	req2, err := http.NewRequest(http.MethodGet, typesURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create download request: %w", err)
	}
	resp2, err := client.Do(req2)
	if err != nil {
		return nil, fmt.Errorf("download types: %w", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download types: http %d", resp2.StatusCode)
	}

	content, err := io.ReadAll(resp2.Body)
	if err != nil {
		return nil, fmt.Errorf("read types body: %w", err)
	}

	// Step 4: Write to cache atomically.
	if err := os.MkdirAll(filepath.Dir(cacheFile), 0755); err != nil {
		return nil, fmt.Errorf("create types cache dir: %w", err)
	}
	tmpFile := cacheFile + ".tmp"
	if err := os.WriteFile(tmpFile, content, 0644); err != nil {
		return nil, fmt.Errorf("write types tmp: %w", err)
	}
	if err := os.Rename(tmpFile, cacheFile); err != nil {
		return nil, fmt.Errorf("rename types tmp: %w", err)
	}

	return &TypeFetchResult{Package: pkg, Version: version, CachedPath: cacheFile, FromCache: false}, nil
}

// FetchTypesForModule reads the module's package.json and fetches type
// definitions for all dependencies. Returns the fetch results.
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
		// Use a simple heuristic: strip leading ^ ~ = from version range.
		version := strings.TrimLeft(verRange, "^~=> ")
		if version == "" || version == "*" {
			continue
		}

		result, err := FetchTypeDefinition(client, upstream, typesDir, name, version)
		if err != nil {
			// Log warning but continue with other deps.
			fmt.Fprintf(os.Stderr, "[esm-type-fetch] warn: %s@%s: %v\n", name, version, err)
			continue
		}
		results = append(results, *result)
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
