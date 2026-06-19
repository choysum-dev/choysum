// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package noderuntime

import (
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
