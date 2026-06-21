// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package parser

import (
	"encoding/json"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
)

func ParseTsconfigPathAlias(buildOptions *api.BuildOptions) (map[string]string, error) {
	var tsconfigAbsDir string
	pathAlias := make(map[string]string)
	var tsconfig map[string]interface{}
	if buildOptions.TsconfigRaw != "" {
		err := json.Unmarshal([]byte(buildOptions.TsconfigRaw), &tsconfig)
		if err != nil {
			return pathAlias, err
		}
		if buildOptions.AbsWorkingDir != "" {
			tsconfigAbsDir = buildOptions.AbsWorkingDir
		} else {
			exePath, _ := os.Executable()
			tsconfigAbsDir, _ = filepath.Abs(filepath.Dir(exePath))
		}
	} else if buildOptions.Tsconfig != "" {
		file, err := os.Open(buildOptions.Tsconfig)
		if err != nil {
			return pathAlias, err
		}
		defer file.Close()
		err = json.NewDecoder(file).Decode(&tsconfig)
		if err != nil {
			return pathAlias, err
		}
		tsconfigAbsDir, _ = filepath.Abs(filepath.Dir(buildOptions.Tsconfig))
	}

	if compilerOptions, ok := tsconfig["compilerOptions"].(map[string]interface{}); ok {
		if paths, ok := compilerOptions["paths"].(map[string]interface{}); ok {
			for key, value := range paths {
				if pathArray, ok := value.([]interface{}); ok && len(pathArray) > 0 {
					if pathStr, ok := pathArray[0].(string); ok {
						if shouldSkipTypeOnlyAlias(key, pathStr) {
							continue
						}
						pathAlias[key] = filepath.Join(tsconfigAbsDir, pathStr)
					}
				}
			}
		}
	}

	return pathAlias, nil
}

func ApplyPathAlias(pathAlias map[string]string, path string) string {
	for alias, realPath := range pathAlias {
		if strings.HasSuffix(alias, "*") {
			prefix := strings.TrimSuffix(alias, "*")
			if !strings.HasPrefix(path, prefix) {
				continue
			}
			realPrefix := strings.TrimSuffix(realPath, "*")
			return realPrefix + strings.TrimPrefix(path, prefix)
		}
		if path == alias {
			return realPath
		}
	}

	// Backward compatibility: some callers pass "@" (without wildcard) to
	// represent the module root and still expect "@/..." imports to be resolved.
	if rootAlias, ok := pathAlias["@"]; ok && strings.HasPrefix(path, "@/") {
		return filepath.Join(rootAlias, strings.TrimPrefix(path, "@/"))
	}

	return path
}

func shouldSkipTypeOnlyAlias(alias string, target string) bool {
	alias = strings.TrimSpace(alias)
	target = strings.TrimSpace(target)
	if alias == "" || target == "" {
		return false
	}
	if strings.Contains(alias, "*") {
		return false
	}
	trimmed := target
	if i := strings.Index(trimmed, "?"); i >= 0 {
		trimmed = trimmed[:i]
	}
	if i := strings.Index(trimmed, "#"); i >= 0 {
		trimmed = trimmed[:i]
	}
	base := strings.ToLower(path.Base(trimmed))
	if base == "" {
		return false
	}
	if strings.HasSuffix(base, ".d.ts") || strings.HasSuffix(base, ".d.mts") || strings.HasSuffix(base, ".d.cts") {
		return true
	}
	return strings.Contains(base, ".d.ts.") || strings.Contains(base, ".d.mts.") || strings.Contains(base, ".d.cts.")
}
