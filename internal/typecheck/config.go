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

	paths, pathsBase, err := resolveModulePaths(modulesPath)
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

	typeRoots := resolveTypeRoots(repoRoot)
	types := resolveCompilerTypes(typeRoots)

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

func resolveModulePaths(modulesRoot string) (map[string][]string, string, error) {
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
			if filepath.IsAbs(t) {
				absTargets = append(absTargets, filepath.ToSlash(t))
			} else {
				absTargets = append(absTargets, filepath.ToSlash(filepath.Join(pathsBase, t)))
			}
		}
		out[alias] = absTargets
	}
	return out, pathsBase, nil
}

func resolveTypeRoots(repoRoot string) []string {
	var roots []string
	localTypes := filepath.Join(repoRoot, "node_modules", "@types")
	if st, err := os.Stat(localTypes); err == nil && st.IsDir() {
		roots = append(roots, filepath.ToSlash(localTypes))
	}
	// Optional explicit global types root only — never shell out to npm.
	if globalRoot := strings.TrimSpace(os.Getenv("CHOYSUM_NPM_GLOBAL_ROOT")); globalRoot != "" {
		globalTypes := filepath.Join(globalRoot, "@types")
		if st, err := os.Stat(globalTypes); err == nil && st.IsDir() {
			roots = append(roots, filepath.ToSlash(globalTypes))
		}
	}
	return roots
}

func resolveCompilerTypes(typeRoots []string) []string {
	for _, root := range typeRoots {
		if st, err := os.Stat(filepath.Join(root, "node")); err == nil && st.IsDir() {
			return []string{"node"}
		}
	}
	return nil
}

// ResolveModulePathsForTest exposes path resolution for unit tests.
func ResolveModulePathsForTest(modulesRoot string) (map[string][]string, string, error) {
	return resolveModulePaths(modulesRoot)
}

// Test hooks for hard-to-trigger filesystem failures.
var readFile = os.ReadFile
