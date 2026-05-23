// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package vueplugin

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
)

// parseTsconfigPathAlias parses path aliases from tsconfig for module resolution.
// It supports both raw tsconfig JSON and file-based tsconfig configurations.
//
// The function handles two scenarios:
// 1. When TsconfigRaw is provided (inline JSON configuration)
// 2. When Tsconfig file path is provided (reads from file system)
//
// It extracts compilerOptions.paths from the tsconfig and converts them to
// absolute paths based on the tsconfig directory location.
//
// Returns a map where keys are alias patterns (e.g., "@/*") and values are
// absolute file system paths (e.g., "/project/src/*").
func ParseTsconfigPathAlias(buildOptions *api.BuildOptions) (map[string]string, error) {
	var tsconfigAbsDir string
	pathAlias := make(map[string]string)
	var tsconfig map[string]interface{}

	// Branch 1: Parse from raw tsconfig JSON if provided
	if buildOptions.TsconfigRaw != "" {
		// Unmarshal the inline JSON configuration
		err := json.Unmarshal([]byte(buildOptions.TsconfigRaw), &tsconfig)
		if err != nil {
			return pathAlias, err
		}

		// Determine the base directory for resolving relative paths
		if buildOptions.AbsWorkingDir != "" {
			tsconfigAbsDir = buildOptions.AbsWorkingDir
		} else {
			// Fallback: use executable directory if no working directory is specified
			exePath, _ := os.Executable()
			tsconfigAbsDir, _ = filepath.Abs(filepath.Dir(exePath))
		}

		// Branch 2: Parse from tsconfig file if file path is provided
	} else if buildOptions.Tsconfig != "" {
		// Open and read the tsconfig.json file
		file, err := os.Open(buildOptions.Tsconfig)
		if err != nil {
			return pathAlias, err
		}
		defer file.Close()

		// Decode JSON content from the file
		err = json.NewDecoder(file).Decode(&tsconfig)
		if err != nil {
			return pathAlias, err
		}

		// Use the tsconfig file's directory as the base for relative paths
		tsconfigAbsDir, _ = filepath.Abs(filepath.Dir(buildOptions.Tsconfig))
	}

	// Extract path aliases from compilerOptions.paths if present
	if compilerOptions, ok := tsconfig["compilerOptions"].(map[string]interface{}); ok {
		if paths, ok := compilerOptions["paths"].(map[string]interface{}); ok {
			// Process each alias definition
			for alias, pathMappings := range paths {
				// TypeScript paths can have multiple mappings, we use the first one
				if pathArray, ok := pathMappings.([]interface{}); ok && len(pathArray) > 0 {
					if pathStr, ok := pathArray[0].(string); ok {
						// Convert relative path to absolute path based on tsconfig directory
						pathAlias[alias] = filepath.Join(tsconfigAbsDir, pathStr)
					}
				}
			}
		}
	}

	return pathAlias, nil
}

// applyPathAlias applies path alias mapping to the given import path.
// It supports both exact aliases and wildcard patterns using '*'.
//
// For exact aliases (e.g., "@utils" -> "/src/utils"):
//   - The alias must match the entire path
//   - Replacement is direct substitution
//
// For wildcard aliases (e.g., "@/*" -> "/src/*"):
//   - The '*' captures any remaining path segments
//   - The captured content is substituted in the target path
//
// Examples:
//   - "@/components/Button" with "@/*" -> "/src/*" becomes "/src/components/Button"
//   - "@utils" with "@utils" -> "/src/utils" becomes "/src/utils"
//
// Returns the original path if no alias matches.
func ApplyPathAlias(pathAlias map[string]string, path string) string {
	// Try each alias pattern against the input path
	for alias, realPath := range pathAlias {
		// Start with exact pattern matching using regex escaping
		aliasPattern := "^" + regexp.QuoteMeta(alias)

		// Handle wildcard patterns - '*' captures remaining path segments
		if len(alias) > 0 && alias[len(alias)-1] == '*' {
			// Remove the escaped '*' and add capture group for wildcard matching
			aliasPattern = aliasPattern[:len(aliasPattern)-2] + "(.*)"
			// Replace '*' in target path with captured content
			realPath = realPath[:len(realPath)-1] + "$1"
		} else {
			// For exact aliases, ensure complete path matching
			aliasPattern = aliasPattern + "$"
		}

		// Compile and test the pattern
		re := regexp.MustCompile(aliasPattern)
		if re.MatchString(path) {
			// Apply the alias transformation
			return re.ReplaceAllString(path, realPath)
		}
	}

	// No alias matched, return original path unchanged
	return path
}

// replaceStylePaths rewrites matched @import/@use/@forward/url paths using pathAlias and baseFilePath.
// All returned paths are POSIX-style relative paths for maximum compatibility.
func replaceStylePaths(content string, baseFilePath string, pathAlias map[string]string) string {
	pathRe := regexp.MustCompile(`@(import|use|forward)\s+(?:url\()?['"]([^'"]*)['"]\)?|url\(['"]([^'"]*)['"]\)`)
	baseDir := filepath.Dir(baseFilePath)
	return pathRe.ReplaceAllStringFunc(content, func(match string) string {
		submatches := pathRe.FindStringSubmatch(match)
		var origPath string
		if submatches[2] != "" {
			origPath = submatches[2]
		} else if submatches[3] != "" {
			origPath = submatches[3]
		} else {
			return match
		}
		absPath := ApplyPathAlias(pathAlias, origPath)
		if strings.HasPrefix(absPath, "./") || strings.HasPrefix(absPath, "../") {
			absPath = filepath.Join(baseDir, absPath)
		}
		absPath, _ = filepath.Abs(absPath)
		relPath, _ := filepath.Rel(baseDir, absPath)
		relPath = strings.ReplaceAll(relPath, `\`, `/`)
		// Ensure relative path starts with './' or '../' for Sass
		if !strings.HasPrefix(relPath, ".") {
			relPath = "./" + relPath
		}
		return strings.Replace(match, origPath, relPath, 1)
	})
}

// ResolveVueStylePath applies path alias and relative path resolution for <style> blocks in Vue SFC content.
// All rewritten paths are POSIX-style relative paths.
func ResolveVueStylePath(content string, vueFilePath string, pathAlias map[string]string) string {
	styleTagRe := regexp.MustCompile(`(?s)<style[^>]*>(.*?)</style>`)
	return styleTagRe.ReplaceAllStringFunc(content, func(styleBlock string) string {
		matches := styleTagRe.FindStringSubmatch(styleBlock)
		inner := matches[1]
		newInner := replaceStylePaths(inner, vueFilePath, pathAlias)
		return strings.Replace(styleBlock, inner, newInner, 1)
	})
}

// ResolveScssPath applies path alias and relative path resolution for @import/@use/@forward/url in SCSS content.
// All rewritten paths are POSIX-style relative paths.
func ResolveScssPath(content string, scssFilePath string, pathAlias map[string]string) string {
	return replaceStylePaths(content, scssFilePath, pathAlias)
}
