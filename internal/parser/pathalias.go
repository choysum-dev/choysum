// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package parser

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"

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
		aliasPattern := "^" + regexp.QuoteMeta(alias)
		if alias[len(alias)-1] == '*' {
			aliasPattern = aliasPattern[:len(aliasPattern)-2] + "(.*)"
			realPath = realPath[:len(realPath)-1] + "$1"
		}
		re := regexp.MustCompile(aliasPattern)
		if re.MatchString(path) {
			return re.ReplaceAllString(path, realPath)
		}
	}
	return path
}
