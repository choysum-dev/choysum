// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package parser

import (
	"strings"
)

// cleanVuePath removes query parameters from Vue file paths.
func cleanVuePath(path string) string {
	if path == "" {
		return path
	}
	if idx := strings.Index(path, "?"); idx != -1 {
		return path[:idx]
	}
	return path
}

func findVueChildrenWithSameName(parserResults []*ParserResult, name string) []*ParserResult {
	// 1. Collect components with the same name and build lookup maps.
	sameNameComponents := make([]*ParserResult, 0)
	pathToComponent := make(map[string]*ParserResult)
	extendsToChildren := make(map[string][]*ParserResult)

	for _, r := range parserResults {
		if r.VueComponent != nil && r.VueComponent.Name == name {
			sameNameComponents = append(sameNameComponents, r)
			pathToComponent[r.Path] = r
			if r.VueComponent.Extends != "" {
				extendsToChildren[r.VueComponent.Extends] = append(
					extendsToChildren[r.VueComponent.Extends], r)
			}
		}
	}

	if len(sameNameComponents) == 0 {
		return nil
	}

	// 2. Build an ordered inheritance chain.
	var sortedComponents []*ParserResult
	visited := make(map[string]bool)

	// 3. Walk from each component so every inheritance chain is covered.
	for _, r := range sameNameComponents {
		if visited[r.Path] {
			continue
		}

		// Find the root node of the inheritance chain.
		current := r
		for current.VueComponent.Extends != "" {
			if parent := pathToComponent[current.VueComponent.Extends]; parent != nil {
				current = parent
			} else {
				break
			}
		}

		// Build the inheritance chain from the root.
		for {
			if !visited[current.Path] {
				visited[current.Path] = true
				sortedComponents = append(sortedComponents, current)
			}

			// Add the next child component if present.
			children := extendsToChildren[current.Path]
			if len(children) == 0 {
				break
			}
			current = children[0]
		}
	}

	return sortedComponents
}

func FindVueComponentFinalChild(parserResults []*ParserResult, componentPath string, importPath string) string {
	// 1. Basic input checks.
	if componentPath == "" || len(parserResults) == 0 {
		return ""
	}

	// 2. Find the importer component.
	var importerComponent *ParserResult
	for _, r := range parserResults {
		if r.Path == cleanVuePath(componentPath) && r.VueComponent != nil {
			importerComponent = r
			break
		}
	}

	// 3. Avoid cyclic dependency: if the importer already extends this component,
	// keep the original path.
	if importerComponent != nil {
		cleanImportPath := cleanVuePath(importPath)
		if cleanImportPath == importerComponent.VueComponent.Extends || cleanImportPath == importerComponent.Path {
			return ""
		}
	}

	var originalComponent *ParserResult
	for _, r := range parserResults {
		if r.Path == cleanVuePath(importPath) && r.VueComponent != nil {
			originalComponent = r
			break
		}
	}
	if originalComponent == nil {
		return ""
	}

	// 4. Return the last component in the inheritance chain for the same name.
	sameNameComponents := findVueChildrenWithSameName(parserResults, originalComponent.VueComponent.Name)
	if len(sameNameComponents) > 0 {
		return sameNameComponents[len(sameNameComponents)-1].Path
	}

	return ""
}
