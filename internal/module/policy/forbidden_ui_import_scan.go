// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package policy

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/choysum-dev/choysum/internal/parser"
	xfmt "golang.org/x/exp/errors/fmt"
)

var walkWebTree = func(root string, walkFn fs.WalkDirFunc) error {
	return filepath.WalkDir(root, walkFn)
}

var vueScriptBlockRe = regexp.MustCompile(`(?is)<script\b[^>]*>([\s\S]*?)</script>`)

// ScanForbiddenUiImportsOnDisk walks moduleRoot/web and applies forbidden kit import rules.
func ScanForbiddenUiImportsOnDisk(input ForbiddenUiImportScanInput) ([]ForbiddenUiImportViolation, error) {
	modulesPath := strings.TrimSpace(input.ModulesPath)
	moduleName := strings.TrimSpace(input.ModuleName)
	moduleRoot := strings.TrimSpace(input.ModuleRoot)
	if modulesPath == "" || moduleName == "" || moduleRoot == "" {
		return nil, nil
	}
	if isChoyUIKitModule(moduleName) {
		return nil, nil
	}

	webRoot := filepath.Join(moduleRoot, "web")
	st, err := statPath(webRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, xfmt.Errorf("stat web dir: %w", err)
	}
	if !st.IsDir() {
		return nil, nil
	}

	var parserResults []*parser.ParserResult
	walkErr := walkWebTree(webRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if shouldSkipWebScanDir(d.Name()) {
				return fs.SkipDir
			}
			return nil
		}
		if !isWebImportSource(path) {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return xfmt.Errorf("read %s: %w", path, err)
		}
		sources := webImportSources(path, content)
		for i, src := range sources {
			// Always parse as .ts: Vue paths make typescript-go panic (ScriptKind unset).
			virtualPath := strings.TrimSuffix(path, filepath.Ext(path)) + ".webimport.ts"
			if i > 0 {
				virtualPath = strings.TrimSuffix(path, filepath.Ext(path)) + ".webimport." + itoa(i) + ".ts"
			}
			result, err := ParseServiceSourceFile(input.PathAlias, virtualPath, []byte(src))
			if err != nil {
				return err
			}
			// Keep the on-disk path for violation reporting.
			result.Path = path
			parserResults = append(parserResults, result)
		}
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}

	return CheckForbiddenUiImports(ForbiddenUiImportScanInput{
		ModulesPath: modulesPath,
		ModuleName:  moduleName,
		ModuleRoot:  moduleRoot,
		PathAlias:   input.PathAlias,
	}, parserResults), nil
}

// CheckForbiddenUiImportsOnDisk is the filesystem entry used by typecheck.
func CheckForbiddenUiImportsOnDisk(modulesPath, moduleName string, pathAlias map[string]string) error {
	modulesPath = strings.TrimSpace(modulesPath)
	moduleName = strings.TrimSpace(moduleName)
	if modulesPath == "" || moduleName == "" {
		return nil
	}
	if isChoyUIKitModule(moduleName) {
		return nil
	}
	moduleRoot := filepath.Join(modulesPath, moduleName)
	if pathAlias == nil {
		pathAlias = ModulePathAliasForBoundary(modulesPath)
	}
	violations, err := ScanForbiddenUiImportsOnDisk(ForbiddenUiImportScanInput{
		ModulesPath: modulesPath,
		ModuleName:  moduleName,
		ModuleRoot:  moduleRoot,
		PathAlias:   pathAlias,
	})
	if err != nil {
		return err
	}
	return FormatForbiddenUiImportError(violations)
}

func shouldSkipWebScanDir(name string) bool {
	switch name {
	case "node_modules", "dist", ".choysum", "tmp", ".git":
		return true
	default:
		return false
	}
}

func isWebImportSource(path string) bool {
	name := strings.ToLower(filepath.Base(path))
	switch {
	case strings.HasSuffix(name, ".d.ts"):
		return false
	case strings.HasSuffix(name, ".vue"):
		return true
	case strings.HasSuffix(name, ".ts"), strings.HasSuffix(name, ".tsx"),
		strings.HasSuffix(name, ".mts"), strings.HasSuffix(name, ".cts"),
		strings.HasSuffix(name, ".js"), strings.HasSuffix(name, ".jsx"),
		strings.HasSuffix(name, ".mjs"), strings.HasSuffix(name, ".cjs"):
		return true
	default:
		return false
	}
}

func webImportSources(path string, content []byte) []string {
	if strings.HasSuffix(strings.ToLower(path), ".vue") {
		matches := vueScriptBlockRe.FindAllSubmatch(content, -1)
		if len(matches) == 0 {
			return nil
		}
		out := make([]string, 0, len(matches))
		for _, m := range matches {
			if len(m) < 2 {
				continue
			}
			src := strings.TrimSpace(string(m[1]))
			if src == "" {
				continue
			}
			out = append(out, src)
		}
		return out
	}
	return []string{string(content)}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
