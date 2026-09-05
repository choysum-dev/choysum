// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package typecheck

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/buke/typescript-go-internal/v7/pkg/collections"
	"github.com/buke/typescript-go-internal/v7/pkg/core"
)

// BuildCompilerOptions builds compiler options aligned with the historical
// typecheck temporary tsconfig (service / strict / bundler).
func BuildCompilerOptions(modulesPath, repoRoot string) (*core.CompilerOptions, error) {
	modulesPath = filepath.Clean(modulesPath)
	repoRoot = filepath.Clean(repoRoot)

	paths := resolveModulePaths(modulesPath)
	pathsMap := collections.NewOrderedMapWithSizeHint[string, []string](len(paths) + 1)
	for alias, targets := range paths {
		pathsMap.Set(alias, targets)
	}
	if _, ok := pathsMap.Get("@/*"); !ok {
		pathsMap.Set("@/*", []string{filepath.ToSlash(filepath.Join(modulesPath, "*"))})
	}

	typeRoots := resolveTypeRoots(repoRoot)
	types := resolveCompilerTypes(typeRoots)

	opts := &core.CompilerOptions{
		Target:                           core.ScriptTargetES2020,
		Module:                           core.ModuleKindESNext,
		ModuleResolution:                 core.ModuleResolutionKindBundler,
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
		PathsBasePath:                    filepath.ToSlash(modulesPath),
		TypeRoots:                        typeRoots,
		Types:                            types,
		ForceConsistentCasingInFileNames: core.TSTrue,
	}
	return opts, nil
}

func resolveModulePaths(modulesRoot string) map[string][]string {
	out := make(map[string][]string)
	tsconfigPath := filepath.Join(modulesRoot, "tsconfig.json")
	data, err := os.ReadFile(tsconfigPath)
	if err != nil {
		return out
	}
	var tsconfig struct {
		CompilerOptions struct {
			Paths map[string][]string `json:"paths"`
		} `json:"compilerOptions"`
	}
	if err := json.Unmarshal(data, &tsconfig); err != nil {
		return out
	}
	for alias, targets := range tsconfig.CompilerOptions.Paths {
		var absTargets []string
		for _, t := range targets {
			if filepath.IsAbs(t) {
				absTargets = append(absTargets, filepath.ToSlash(t))
			} else {
				absTargets = append(absTargets, filepath.ToSlash(filepath.Join(modulesRoot, t)))
			}
		}
		out[alias] = absTargets
	}
	return out
}

func resolveTypeRoots(repoRoot string) []string {
	var roots []string
	localTypes := filepath.Join(repoRoot, "node_modules", "@types")
	if _, err := os.Stat(localTypes); err == nil {
		roots = append(roots, filepath.ToSlash(localTypes))
	}
	if globalRoot := resolveGlobalNpmRootBestEffort(); globalRoot != "" {
		globalTypes := filepath.Join(globalRoot, "@types")
		if _, err := os.Stat(globalTypes); err == nil {
			roots = append(roots, filepath.ToSlash(globalTypes))
		}
	}
	return roots
}

func resolveGlobalNpmRootBestEffort() string {
	if override := strings.TrimSpace(os.Getenv("CHOYSUM_NPM_GLOBAL_ROOT")); override != "" {
		return override
	}
	out, err := exec.Command("npm", "root", "-g").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func resolveCompilerTypes(typeRoots []string) []string {
	for _, root := range typeRoots {
		if _, err := os.Stat(filepath.Join(root, "node")); err == nil {
			return []string{"node"}
		}
	}
	return nil
}

// ResolveModulePathsForTest exposes path resolution for unit tests.
func ResolveModulePathsForTest(modulesRoot string) map[string][]string {
	return resolveModulePaths(modulesRoot)
}
