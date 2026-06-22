// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package noderuntime

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func ResolveGlobalNpmRoot() (string, error) {
	if override := strings.TrimSpace(os.Getenv("CHOYSUM_NPM_GLOBAL_ROOT")); override != "" {
		return override, nil
	}
	out, err := exec.Command("npm", "root", "-g").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func ResolveGlobalNpmRootBestEffort() string {
	root, err := ResolveGlobalNpmRoot()
	if err != nil {
		return ""
	}
	return root
}

func EnsureGlobalModuleLinksAt(localNodeModulesRoot string, globalNodeModulesRoot string, moduleNames []string) (func(), error) {
	noop := func() {}
	localNodeModulesRoot = strings.TrimSpace(localNodeModulesRoot)
	globalNodeModulesRoot = strings.TrimSpace(globalNodeModulesRoot)
	if localNodeModulesRoot == "" || globalNodeModulesRoot == "" {
		return noop, nil
	}
	if st, err := os.Stat(globalNodeModulesRoot); err != nil || !st.IsDir() {
		return noop, nil
	}

	if err := os.MkdirAll(localNodeModulesRoot, 0o755); err != nil {
		return nil, fmt.Errorf("create local node_modules: %w", err)
	}

	createdLinks := make([]string, 0, len(moduleNames))
	cleanup := func() {
		for _, localModuleDir := range createdLinks {
			st, err := os.Lstat(localModuleDir)
			if err != nil || st.Mode()&os.ModeSymlink == 0 {
				continue
			}
			_ = os.Remove(localModuleDir)
			pruneEmptyDirs(filepath.Dir(localModuleDir), localNodeModulesRoot)
		}
		pruneEmptyDirs(localNodeModulesRoot, localNodeModulesRoot)
	}

	for _, moduleName := range moduleNames {
		moduleName = strings.TrimSpace(moduleName)
		if moduleName == "" {
			continue
		}

		globalModuleDir := filepath.Join(globalNodeModulesRoot, filepath.FromSlash(moduleName))
		if st, err := os.Stat(globalModuleDir); err != nil || !st.IsDir() {
			continue
		}

		localModuleDir := filepath.Join(localNodeModulesRoot, filepath.FromSlash(moduleName))
		if _, err := os.Lstat(localModuleDir); err == nil {
			continue
		} else if !os.IsNotExist(err) {
			return nil, fmt.Errorf("stat %s: %w", localModuleDir, err)
		}

		if err := os.MkdirAll(filepath.Dir(localModuleDir), 0o755); err != nil {
			return nil, fmt.Errorf("prepare %s: %w", localModuleDir, err)
		}
		if err := os.Symlink(globalModuleDir, localModuleDir); err != nil {
			pruneEmptyDirs(filepath.Dir(localModuleDir), localNodeModulesRoot)
			cleanup()
			return nil, fmt.Errorf("link %s -> %s: %w", localModuleDir, globalModuleDir, err)
		}
		createdLinks = append(createdLinks, localModuleDir)
	}

	return cleanup, nil
}

func pruneEmptyDirs(startDir string, stopDir string) {
	startDir = filepath.Clean(strings.TrimSpace(startDir))
	stopDir = filepath.Clean(strings.TrimSpace(stopDir))
	if startDir == "." || stopDir == "." {
		return
	}

	dir := startDir
	for {
		rel, relErr := filepath.Rel(stopDir, dir)
		if relErr != nil || strings.HasPrefix(rel, "..") {
			return
		}

		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) > 0 {
			return
		}
		if err := os.Remove(dir); err != nil {
			return
		}
		if dir == stopDir {
			return
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return
		}
		dir = parent
	}
}

func FindExecutable(tool string, localRoots ...string) (string, string, bool) {
	tool = strings.TrimSpace(tool)
	if tool == "" {
		return "", "", false
	}
	if _, err := exec.LookPath(tool); err == nil {
		return tool, "", true
	}
	return FindExecutableInRoots(tool, localRoots...)
}

func FindExecutableInRoots(tool string, localRoots ...string) (string, string, bool) {
	tool = strings.TrimSpace(tool)
	if tool == "" {
		return "", "", false
	}

	seen := make(map[string]struct{}, len(localRoots))
	for _, root := range localRoots {
		for _, binDir := range candidateBinDirs(root) {
			if _, exists := seen[binDir]; exists {
				continue
			}
			seen[binDir] = struct{}{}

			if resolved, ok := findExecutableInBinDir(binDir, tool); ok {
				return resolved, binDir, true
			}
		}
	}
	return "", "", false
}

func candidateBinDirs(root string) []string {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil
	}

	if st, err := os.Stat(root); err == nil && !st.IsDir() {
		return []string{filepath.Dir(root)}
	}

	if filepath.Base(root) == ".bin" {
		return []string{root}
	}

	return []string{filepath.Join(root, ".bin")}
}

func findExecutableInBinDir(binDir string, tool string) (string, bool) {
	if strings.TrimSpace(binDir) == "" {
		return "", false
	}

	candidate := filepath.Join(binDir, tool)
	if st, err := os.Stat(candidate); err == nil && !st.IsDir() {
		return candidate, true
	}

	candidateCmd := candidate + ".cmd"
	if st, err := os.Stat(candidateCmd); err == nil && !st.IsDir() {
		return candidateCmd, true
	}

	return "", false
}

func ModuleInstalledInRoots(moduleName string, moduleRoots ...string) bool {
	moduleName = strings.TrimSpace(moduleName)
	if moduleName == "" {
		return false
	}
	for _, root := range moduleRoots {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		moduleDir := filepath.Join(root, filepath.FromSlash(moduleName))
		if st, err := os.Stat(moduleDir); err == nil && st.IsDir() {
			return true
		}
	}
	return false
}

func MissingRequiredNodeModules(required []string, moduleRoots ...string) []string {
	missing := make([]string, 0)
	for _, moduleName := range required {
		if ModuleInstalledInRoots(moduleName, moduleRoots...) {
			continue
		}
		missing = append(missing, moduleName)
	}
	return missing
}
