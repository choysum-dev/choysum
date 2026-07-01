// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package moddeps

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type modulePackageManifest struct {
	Dependencies     map[string]string `json:"dependencies"`
	PeerDependencies map[string]string `json:"peerDependencies"`
	Choysum          struct {
		Depends []string `json:"depends"`
	} `json:"choysum"`
}

// MergeRequiredModules merges multiple module lists, removes duplicates, and sorts results.
func MergeRequiredModules(moduleLists ...[]string) []string {
	seen := make(map[string]struct{})
	for _, modules := range moduleLists {
		for _, moduleName := range modules {
			moduleName = strings.TrimSpace(moduleName)
			if moduleName == "" {
				continue
			}
			seen[moduleName] = struct{}{}
		}
	}
	if len(seen) == 0 {
		return nil
	}
	out := make([]string, 0, len(seen))
	for moduleName := range seen {
		out = append(out, moduleName)
	}
	sort.Strings(out)
	return out
}

// CollectExternalModuleDependencies scans package.json files under modulesPath and collects
// non-workspace dependencies/peerDependencies for the given module names.
//
// When followDepends is true, the traversal recursively follows choysum.depends.
func CollectExternalModuleDependencies(modulesPath string, moduleNames []string, followDepends bool) ([]string, error) {
	modulesPath = strings.TrimSpace(modulesPath)
	if modulesPath == "" {
		return nil, nil
	}

	pending := append([]string{}, moduleNames...)
	visited := make(map[string]struct{}, len(moduleNames))
	required := make(map[string]struct{})

	for len(pending) > 0 {
		moduleName := strings.TrimSpace(pending[0])
		pending = pending[1:]
		if moduleName == "" {
			continue
		}
		if _, seen := visited[moduleName]; seen {
			continue
		}
		visited[moduleName] = struct{}{}

		pkgPath := filepath.Join(modulesPath, moduleName, "package.json")
		pkg, err := readModulePackageManifest(pkgPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("read module package.json for %s: %w", moduleName, err)
		}

		appendExternalDependencyNames(required, pkg.Dependencies)
		appendExternalDependencyNames(required, pkg.PeerDependencies)

		if followDepends {
			pending = append(pending, pkg.Choysum.Depends...)
		}
	}

	if len(required) == 0 {
		return nil, nil
	}
	out := make([]string, 0, len(required))
	for moduleName := range required {
		out = append(out, moduleName)
	}
	sort.Strings(out)
	return out, nil
}

func appendExternalDependencyNames(out map[string]struct{}, dependencies map[string]string) {
	for moduleName := range dependencies {
		moduleName = strings.TrimSpace(moduleName)
		if moduleName == "" || strings.HasPrefix(moduleName, "@choysum-dev/") {
			continue
		}
		out[moduleName] = struct{}{}
	}
}

func readModulePackageManifest(path string) (modulePackageManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return modulePackageManifest{}, err
	}

	var pkg modulePackageManifest
	if err := json.Unmarshal(data, &pkg); err != nil {
		return modulePackageManifest{}, fmt.Errorf("parse package.json at %s: %w", path, err)
	}
	return pkg, nil
}
