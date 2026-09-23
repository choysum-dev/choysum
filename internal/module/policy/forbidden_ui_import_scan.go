// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package policy

import (
	"bytes"
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

// parseWebImportFile parses one web import source; tests may override.
var parseWebImportFile = ParseServiceSourceFile

// Script open tags may include quoted attributes that contain '>' (e.g. unknown=">").
// Closing </script> must be followed by spaces (including CR) and either another tag
// ('<') or a newline/EOF so mid-line string literals like "</script>" do not truncate
// the block, while CRLF SFCs and compact one-line SFCs still match.
var vueScriptBlockRe = regexp.MustCompile(`(?ims)<script\b(?:[^>"']|"[^"]*"|'[^']*')*>([\s\S]*?)</script>(?:[\t \r]*(?:<|\n|$))`)
var vueHTMLCommentRe = regexp.MustCompile(`(?s)<!--.*?-->`)

// vueScriptInCommentRe detects HTML comments that embed a <script> sample (docs).
var vueScriptInCommentRe = regexp.MustCompile(`(?i)<script\b`)

// ScanForbiddenUiImportsOnDisk walks moduleRoot/web and applies forbidden kit import rules.
func ScanForbiddenUiImportsOnDisk(input ForbiddenUiImportScanInput) ([]ForbiddenUiImportViolation, error) {
	modulesPath := strings.TrimSpace(input.ModulesPath)
	moduleName := strings.TrimSpace(input.ModuleName)
	moduleRoot := strings.TrimSpace(input.ModuleRoot)
	if modulesPath == "" || moduleName == "" || moduleRoot == "" {
		return nil, nil
	}
	if isKitHostModule(moduleName) {
		return nil, nil
	}

	rootSt, err := statPath(moduleRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, xfmt.Errorf("module root does not exist: %s", moduleRoot)
		}
		return nil, xfmt.Errorf("stat module root: %w", err)
	}
	if !rootSt.IsDir() {
		return nil, xfmt.Errorf("module root is not a directory: %s", moduleRoot)
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
			if strings.TrimSpace(src.Content) == "" {
				// Empty / whitespace-only sources have no imports; ParseServiceSourceFile
				// rejects empty content, and TypeScript still accepts empty files.
				continue
			}
			virtualPath := webImportVirtualPath(path, i)
			result, err := parseWebImportFile(input.PathAlias, virtualPath, []byte(src.Content))
			if err != nil {
				return xfmt.Errorf("%s: %w", path, err)
			}
			if result == nil {
				continue
			}
			adjustParserResultLines(result, src.LineOffset)
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
	if isKitHostModule(moduleName) {
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
	case "node_modules", "dist", ".choysum", "tmp", ".git", "public", "coverage":
		return true
	default:
		return false
	}
}

func isWebImportSource(path string) bool {
	name := strings.ToLower(filepath.Base(path))
	switch {
	case isTypeScriptDeclarationFile(name):
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

func isTypeScriptDeclarationFile(name string) bool {
	return strings.HasSuffix(name, ".d.ts") ||
		strings.HasSuffix(name, ".d.mts") ||
		strings.HasSuffix(name, ".d.cts")
}

// webImportSource is one parse unit extracted from a web source file.
type webImportSource struct {
	Content    string
	LineOffset int // newlines before Content's first line in the on-disk file
}

func webImportSources(path string, content []byte) []webImportSource {
	if strings.HasSuffix(strings.ToLower(path), ".vue") {
		// Blank HTML comments (keep length/newlines) so commented-out <script>
		// samples in docs do not become false positives.
		masked := maskVueHTMLComments(content)
		matches := vueScriptBlockRe.FindAllSubmatchIndex(masked, -1)
		if len(matches) == 0 {
			return nil
		}
		out := make([]webImportSource, 0, len(matches))
		for _, m := range matches {
			start, end := m[2], m[3]
			raw := content[start:end]
			trimmed := bytes.TrimSpace(raw)
			if len(trimmed) == 0 {
				continue
			}
			// Account for TrimSpace so reported lines match the kept script text.
			trimLead := bytes.Index(raw, trimmed)
			lineOffset := bytes.Count(content[:start+trimLead], []byte{'\n'})
			out = append(out, webImportSource{
				Content:    string(trimmed),
				LineOffset: lineOffset,
			})
		}
		return out
	}
	return []webImportSource{{Content: string(content), LineOffset: 0}}
}

// maskVueHTMLComments blanks HTML comments that embed a <script> sample.
// Length and newlines are preserved so reported line numbers stay correct.
// Only script-bearing comments are blanked; masking every comment lets a stray
// "<!--" inside script source reach a later "-->" and hide real imports.
func maskVueHTMLComments(content []byte) []byte {
	return vueHTMLCommentRe.ReplaceAllFunc(content, func(match []byte) []byte {
		if !vueScriptInCommentRe.Match(match) {
			return match
		}
		out := make([]byte, len(match))
		for i, b := range match {
			if b == '\n' || b == '\r' {
				out[i] = b
			} else {
				out[i] = ' '
			}
		}
		return out
	})
}

// webImportVirtualPath picks a typescript-go-safe path while preserving JSX when needed.
// The original extension is kept in the stem so Foo.vue and Foo.ts do not collide.
func webImportVirtualPath(path string, index int) string {
	ext := strings.ToLower(filepath.Ext(path))
	base := path // keep original extension in the stem
	suffix := ".webimport"
	if index > 0 {
		suffix = ".webimport." + itoa(index)
	}
	switch ext {
	case ".tsx", ".jsx":
		return base + suffix + ext
	default:
		// Force .ts for .vue and other non-JSX sources: Vue paths make typescript-go panic.
		return base + suffix + ".ts"
	}
}

func adjustParserResultLines(result *parser.ParserResult, lineOffset int) {
	if result == nil || lineOffset == 0 {
		return
	}
	for _, imp := range result.Imports {
		if imp != nil && imp.Line > 0 {
			imp.Line += lineOffset
		}
	}
	for _, imp := range result.DynamicImports {
		if imp != nil && imp.Line > 0 {
			imp.Line += lineOffset
		}
	}
	for _, exp := range result.Exports {
		if exp == nil {
			continue
		}
		if exp.Line > 0 {
			exp.Line += lineOffset
		}
		for _, wild := range exp.Wildcard {
			if wild != nil && wild.Line > 0 {
				wild.Line += lineOffset
			}
		}
	}
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
