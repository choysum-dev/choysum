// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package coverage

import (
	"path/filepath"
	"strings"
)

// matchCoverageGlob is a minimal nyc/minimatch-style matcher for coverage
// include/exclude patterns. It supports `**`, `*`, and `?`. Paths are compared
// with forward slashes; leading `./` is stripped.
func matchCoverageGlob(pattern, path string) bool {
	pattern = strings.TrimSpace(pattern)
	path = strings.TrimSpace(path)
	if pattern == "" {
		return false
	}
	pattern = strings.ReplaceAll(pattern, "\\", "/")
	path = strings.ReplaceAll(path, "\\", "/")
	path = strings.TrimPrefix(path, "./")
	pattern = strings.TrimPrefix(pattern, "./")

	// Absolute patterns: try matching against full path and path-relative forms.
	if strings.HasPrefix(pattern, "/") || (len(pattern) > 1 && pattern[1] == ':') {
		if matchGlobSegments(splitGlob(pattern), splitGlob(path)) {
			return true
		}
	}

	// Prefer matching against the path as-is and against a repo-relative suffix.
	if matchGlobSegments(splitGlob(pattern), splitGlob(path)) {
		return true
	}
	base := filepath.Base(path)
	if matchGlobSegments(splitGlob(pattern), splitGlob(base)) {
		return true
	}
	return false
}

func splitGlob(s string) []string {
	s = strings.Trim(s, "/")
	if s == "" {
		return nil
	}
	return strings.Split(s, "/")
}

func matchGlobSegments(pattern, path []string) bool {
	return matchGlobAt(pattern, path, 0, 0)
}

func matchGlobAt(pattern, path []string, pi, si int) bool {
	for pi < len(pattern) || si < len(path) {
		if pi < len(pattern) && pattern[pi] == "**" {
			if pi+1 == len(pattern) {
				return true
			}
			for k := si; k <= len(path); k++ {
				if matchGlobAt(pattern, path, pi+1, k) {
					return true
				}
			}
			return false
		}
		if pi >= len(pattern) || si >= len(path) {
			return false
		}
		if !matchGlobPart(pattern[pi], path[si]) {
			return false
		}
		pi++
		si++
	}
	return true
}

func matchGlobPart(pattern, part string) bool {
	if pattern == "*" {
		return true
	}
	i, j := 0, 0
	for i < len(pattern) && j < len(part) {
		switch pattern[i] {
		case '*':
			if i+1 == len(pattern) {
				return true
			}
			for k := j; k <= len(part); k++ {
				if matchGlobPart(pattern[i+1:], part[k:]) {
					return true
				}
			}
			return false
		case '?':
			i++
			j++
		default:
			if pattern[i] != part[j] {
				return false
			}
			i++
			j++
		}
	}
	for i < len(pattern) && pattern[i] == '*' {
		i++
	}
	return i == len(pattern) && j == len(part)
}

func pathMatchesAnyGlob(path string, globs []string) bool {
	for _, g := range globs {
		if matchCoverageGlob(g, path) {
			return true
		}
	}
	return false
}

// defaultCoverageExcludes mirrors nycCommonArgs defaults.
func defaultCoverageExcludes() []string {
	return []string{
		"**/*.test.ts",
		"**/*.d.ts",
		"**/.choysum/**",
		"**/dist/**",
		"**/node_modules/**",
	}
}

func mergeExcludeGlobs(extra []string) []string {
	out := append([]string{}, defaultCoverageExcludes()...)
	for _, e := range extra {
		e = strings.TrimSpace(e)
		if e == "" {
			continue
		}
		out = append(out, e)
	}
	return out
}

func coveragePathIncluded(path string, includes, excludes []string) bool {
	path = strings.ReplaceAll(path, "\\", "/")
	if pathMatchesAnyGlob(path, excludes) {
		return false
	}
	if len(includes) == 0 {
		return true
	}
	return pathMatchesAnyGlob(path, includes)
}
