// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package typecheck

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/buke/typescript-go-internal/v7/pkg/collections"
	"github.com/buke/typescript-go-internal/v7/pkg/core"
	"github.com/tailscale/hujson"
)

// BuildCompilerOptions builds compiler options aligned with the historical
// typecheck temporary tsconfig (service + web TS/TSX / strict / bundler).
//
// Relative path aliases are resolved against tsconfig compilerOptions.baseUrl
// when set (default: modules root). CompilerOptions.BaseUrl itself is not set:
// typescript-go rejects that option (TS5102).
func BuildCompilerOptions(modulesPath, repoRoot string) (*core.CompilerOptions, error) {
	modulesPath = filepath.Clean(modulesPath)
	repoRoot = filepath.Clean(repoRoot)

	paths, pathsBase, err := resolveModulePaths(modulesPath, repoRoot)
	if err != nil {
		return nil, err
	}
	pathsMap := collections.NewOrderedMapWithSizeHint[string, []string](len(paths) + 1)
	aliases := make([]string, 0, len(paths))
	for alias := range paths {
		aliases = append(aliases, alias)
	}
	slices.Sort(aliases)
	for _, alias := range aliases {
		pathsMap.Set(alias, paths[alias])
	}
	if _, ok := pathsMap.Get("@/*"); !ok {
		pathsMap.Set("@/*", []string{filepath.ToSlash(filepath.Join(pathsBase, "*"))})
	}

	typeRoots := resolveTypeRoots(modulesPath, repoRoot)
	types := resolveCompilerTypes(modulesPath, typeRoots)

	opts := &core.CompilerOptions{
		Target:                           core.ScriptTargetES2020,
		Module:                           core.ModuleKindESNext,
		ModuleResolution:                 core.ModuleResolutionKindBundler,
		Jsx:                              core.JsxEmitPreserve,
		Strict:                           core.TSTrue,
		StrictNullChecks:                 core.TSTrue,
		StrictPropertyInitialization:     core.TSFalse,
		ExperimentalDecorators:           core.TSTrue,
		AllowJs:                          core.TSTrue,
		AllowArbitraryExtensions:         core.TSTrue,
		SkipLibCheck:                     core.TSTrue,
		NoEmit:                           core.TSTrue,
		Lib:                              []string{"es2020", "dom", "dom.iterable"},
		Paths:                            pathsMap,
		PathsBasePath:                    filepath.ToSlash(pathsBase),
		TypeRoots:                        typeRoots,
		Types:                            types,
		ForceConsistentCasingInFileNames: core.TSTrue,
	}
	return opts, nil
}

func resolveModulePaths(modulesRoot, repoRoot string) (map[string][]string, string, error) {
	_ = repoRoot // reserved; typecheck must not read node_modules (use type-fetch paths only).
	out := make(map[string][]string)
	pathsBase := modulesRoot
	tsconfigPath := filepath.Join(modulesRoot, "tsconfig.json")
	data, err := readFile(tsconfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			return out, pathsBase, nil
		}
		return nil, "", fmt.Errorf("typecheck: read %s: %w", tsconfigPath, err)
	}
	var tsconfig struct {
		CompilerOptions struct {
			BaseURL string              `json:"baseUrl"`
			Paths   map[string][]string `json:"paths"`
		} `json:"compilerOptions"`
	}
	// tsconfig.json is JSONC (comments / trailing commas); standardize first.
	hv, err := hujson.Parse(data)
	if err != nil {
		return nil, "", fmt.Errorf("typecheck: parse %s: %w", tsconfigPath, err)
	}
	hv.Standardize()
	data = hv.Pack()
	if err := json.Unmarshal(data, &tsconfig); err != nil {
		return nil, "", fmt.Errorf("typecheck: parse %s: %w", tsconfigPath, err)
	}
	if base := strings.TrimSpace(tsconfig.CompilerOptions.BaseURL); base != "" {
		if filepath.IsAbs(base) {
			pathsBase = filepath.Clean(base)
		} else {
			pathsBase = filepath.Clean(filepath.Join(modulesRoot, base))
		}
	}
	for alias, targets := range tsconfig.CompilerOptions.Paths {
		var absTargets []string
		for _, t := range targets {
			abs := t
			if !filepath.IsAbs(t) {
				abs = filepath.ToSlash(filepath.Join(pathsBase, t))
			} else {
				abs = filepath.ToSlash(t)
			}
			abs = rewriteChoysumTypesPath(abs)
			if !typePathExists(abs) {
				continue
			}
			absTargets = append(absTargets, abs)
		}
		if len(absTargets) == 0 {
			continue
		}
		out[alias] = absTargets
	}
	return out, pathsBase, nil
}

// rewriteChoysumTypesPath remaps repo-relative ../../.choysum/pkg/types/... entries
// that point at a missing tree onto $CHOYSUM_HOME/pkg/types (default ~/.choysum).
func rewriteChoysumTypesPath(abs string) string {
	abs = filepath.ToSlash(abs)
	const marker = "/.choysum/pkg/types/"
	idx := strings.Index(abs, marker)
	if idx < 0 {
		return abs
	}
	if typePathExists(abs) {
		return abs
	}
	home := choysumHomeDir()
	if home == "" {
		return abs
	}
	suffix := abs[idx+len(marker):]
	alt := filepath.ToSlash(filepath.Join(home, "pkg", "types", filepath.FromSlash(suffix)))
	if typePathExists(alt) {
		return alt
	}
	return abs
}

func choysumHomeDir() string {
	if v := strings.TrimSpace(os.Getenv("CHOYSUM_HOME")); v != "" {
		return filepath.Clean(v)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".choysum")
}

func typePathExists(path string) bool {
	path = strings.TrimSpace(path)
	if path == "" {
		return false
	}
	// Glob entries (type-fetch wildcards) are treated as present; TS resolves them.
	if strings.ContainsAny(path, "*?[") {
		return true
	}
	st, err := os.Stat(filepath.FromSlash(path))
	return err == nil && !st.IsDir()
}

// hasResolvableVueTypes reports whether Vue package types are available via
// modules/tsconfig paths (type-fetch under ~/.choysum/pkg/types). Does not
// consult node_modules.
func hasResolvableVueTypes(modulesPath, repoRoot string) bool {
	paths, _, err := resolveModulePaths(modulesPath, repoRoot)
	if err != nil {
		return false
	}
	targets, ok := paths["vue"]
	if !ok {
		return false
	}
	for _, t := range targets {
		if typePathExists(t) {
			return true
		}
	}
	return false
}

func resolveTypeRoots(modulesPath, repoRoot string) []string {
	var roots []string
	seen := map[string]struct{}{}
	add := func(p string) {
		p = filepath.ToSlash(filepath.Clean(p))
		if p == "" {
			return
		}
		if _, ok := seen[p]; ok {
			return
		}
		if st, err := os.Stat(filepath.FromSlash(p)); err != nil || !st.IsDir() {
			return
		}
		seen[p] = struct{}{}
		roots = append(roots, p)
	}

	// Prefer modules/tsconfig.json typeRoots (type-fetch under ~/.choysum).
	for _, raw := range readTsconfigTypeRoots(modulesPath) {
		abs := raw
		if !filepath.IsAbs(raw) {
			abs = filepath.Join(modulesPath, raw)
		}
		abs = rewriteChoysumTypesPath(filepath.ToSlash(abs))
		// typeRoots entries are directories; rewriteChoysumTypesPath may leave a
		// missing file path — also try directory existence via choysum home.
		if !typePathExists(abs) && !dirExists(abs) {
			if alt := rewriteChoysumTypesDir(abs); alt != "" {
				abs = alt
			}
		}
		add(abs)
	}

	// Opportunistic only — typecheck must not require node_modules.
	add(filepath.Join(repoRoot, "node_modules", "@types"))
	if globalRoot := strings.TrimSpace(os.Getenv("CHOYSUM_NPM_GLOBAL_ROOT")); globalRoot != "" {
		add(filepath.Join(globalRoot, "@types"))
	}
	return roots
}

func readTsconfigTypeRoots(modulesRoot string) []string {
	tsconfigPath := filepath.Join(modulesRoot, "tsconfig.json")
	data, err := readFile(tsconfigPath)
	if err != nil {
		return nil
	}
	var tsconfig struct {
		CompilerOptions struct {
			TypeRoots []string `json:"typeRoots"`
			Types     []string `json:"types"`
		} `json:"compilerOptions"`
	}
	hv, err := hujson.Parse(data)
	if err != nil {
		return nil
	}
	hv.Standardize()
	if err := json.Unmarshal(hv.Pack(), &tsconfig); err != nil {
		return nil
	}
	return tsconfig.CompilerOptions.TypeRoots
}

func readTsconfigTypes(modulesRoot string) []string {
	tsconfigPath := filepath.Join(modulesRoot, "tsconfig.json")
	data, err := readFile(tsconfigPath)
	if err != nil {
		return nil
	}
	var tsconfig struct {
		CompilerOptions struct {
			Types []string `json:"types"`
		} `json:"compilerOptions"`
	}
	hv, err := hujson.Parse(data)
	if err != nil {
		return nil
	}
	hv.Standardize()
	if err := json.Unmarshal(hv.Pack(), &tsconfig); err != nil {
		return nil
	}
	return tsconfig.CompilerOptions.Types
}

func dirExists(path string) bool {
	st, err := os.Stat(filepath.FromSlash(path))
	return err == nil && st.IsDir()
}

// rewriteChoysumTypesDir remaps missing /.choysum/pkg/types/... directories onto
// $CHOYSUM_HOME/pkg/types (same marker logic as rewriteChoysumTypesPath).
func rewriteChoysumTypesDir(abs string) string {
	abs = filepath.ToSlash(abs)
	const marker = "/.choysum/pkg/types/"
	idx := strings.Index(abs, marker)
	if idx < 0 {
		const markerExact = "/.choysum/pkg/types"
		if !strings.HasSuffix(abs, markerExact) {
			return ""
		}
		idx = strings.Index(abs, markerExact)
		if idx < 0 {
			return ""
		}
		home := choysumHomeDir()
		if home == "" {
			return ""
		}
		alt := filepath.ToSlash(filepath.Join(home, "pkg", "types"))
		if dirExists(alt) {
			return alt
		}
		return ""
	}
	if dirExists(abs) {
		return abs
	}
	home := choysumHomeDir()
	if home == "" {
		return ""
	}
	suffix := abs[idx+len(marker):]
	alt := filepath.ToSlash(filepath.Join(home, "pkg", "types", filepath.FromSlash(suffix)))
	if dirExists(alt) {
		return alt
	}
	return ""
}

func resolveCompilerTypes(modulesPath string, typeRoots []string) []string {
	if configured := readTsconfigTypes(modulesPath); len(configured) > 0 {
		var out []string
		for _, name := range configured {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			for _, root := range typeRoots {
				if dirExists(filepath.Join(root, name)) {
					out = append(out, name)
					break
				}
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	for _, root := range typeRoots {
		if dirExists(filepath.Join(root, "node")) {
			return []string{"node"}
		}
	}
	return nil
}

// ResolveModulePathsForTest exposes path resolution for unit tests.
func ResolveModulePathsForTest(modulesRoot, repoRoot string) (map[string][]string, string, error) {
	return resolveModulePaths(modulesRoot, repoRoot)
}

// Test hooks for hard-to-trigger filesystem failures.
var readFile = os.ReadFile
