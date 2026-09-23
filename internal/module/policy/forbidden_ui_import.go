// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package policy

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/choysum-dev/choysum/internal/parser"
)

const choyUIModuleName = "choy_ui"

// ForbiddenUiImportViolation is one forbidden kit/primitive import in a domain web tree.
type ForbiddenUiImportViolation struct {
	SourcePath string
	Line       int
	Column     int
	SpecText   string
	Rule       string
}

// ForbiddenUiImportScanInput configures a web-tree forbidden-import scan for one module.
type ForbiddenUiImportScanInput struct {
	ModulesPath string
	ModuleName  string
	ModuleRoot  string
	PathAlias   map[string]string
}

// CheckForbiddenUiImports reports kit/primitive imports that domain modules must not use.
func CheckForbiddenUiImports(input ForbiddenUiImportScanInput, parserResults []*parser.ParserResult) []ForbiddenUiImportViolation {
	moduleName := strings.TrimSpace(input.ModuleName)
	moduleRoot := strings.TrimSpace(input.ModuleRoot)
	if moduleName == "" || moduleRoot == "" {
		return nil
	}
	if isChoyUIKitModule(moduleName) {
		return nil
	}

	var violations []ForbiddenUiImportViolation
	for _, result := range parserResults {
		if result == nil || !IsModuleWebSource(moduleRoot, result.Path) {
			continue
		}
		for _, imp := range result.Imports {
			if imp == nil {
				continue
			}
			violations = appendForbiddenUiSpec(violations, result.Path, imp.ModuleSpecText, imp.Line, imp.Column)
		}
		for _, imp := range result.DynamicImports {
			if imp == nil {
				continue
			}
			violations = appendForbiddenUiSpec(violations, result.Path, imp.ModuleSpecText, imp.Line, imp.Column)
		}
	}

	sort.Slice(violations, func(i, j int) bool {
		if violations[i].SourcePath != violations[j].SourcePath {
			return violations[i].SourcePath < violations[j].SourcePath
		}
		if violations[i].Line != violations[j].Line {
			return violations[i].Line < violations[j].Line
		}
		if violations[i].Column != violations[j].Column {
			return violations[i].Column < violations[j].Column
		}
		return violations[i].SpecText < violations[j].SpecText
	})
	return violations
}

// IsModuleWebSource reports whether path is under moduleRoot/web/.
func IsModuleWebSource(moduleRoot, path string) bool {
	moduleRoot = strings.TrimSpace(moduleRoot)
	path = strings.TrimSpace(path)
	if moduleRoot == "" || path == "" {
		return false
	}
	rel, err := filepath.Rel(filepath.Clean(moduleRoot), filepath.Clean(path))
	if err != nil {
		return false
	}
	rel = filepath.ToSlash(rel)
	if rel == "." || strings.HasPrefix(rel, "../") || rel == ".." {
		return false
	}
	return strings.HasPrefix(rel, "web/") || rel == "web"
}

func isChoyUIKitModule(moduleName string) bool {
	return strings.TrimSpace(moduleName) == choyUIModuleName
}

func appendForbiddenUiSpec(
	violations []ForbiddenUiImportViolation,
	sourcePath, specText string,
	line, column int,
) []ForbiddenUiImportViolation {
	spec := strings.TrimSpace(strings.Trim(specText, `"'`))
	if spec == "" {
		return violations
	}
	rule := classifyForbiddenUiImport(spec)
	if rule == "" {
		return violations
	}
	return append(violations, ForbiddenUiImportViolation{
		SourcePath: sourcePath,
		Line:       line,
		Column:     column,
		SpecText:   spec,
		Rule:       rule,
	})
}

// classifyForbiddenUiImport returns a rule id when the import specifier is banned for domain modules.
func classifyForbiddenUiImport(spec string) string {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return ""
	}
	slash := filepath.ToSlash(spec)
	lower := strings.ToLower(slash)

	if lower == "reka-ui" || strings.HasPrefix(lower, "reka-ui/") {
		return "reka-ui"
	}
	if lower == "@unovis" || strings.HasPrefix(lower, "@unovis/") {
		return "@unovis"
	}

	// Bare / alias imports that target L2 ui/* or L3 internal/* trees.
	if isForbiddenUIPath(lower) {
		return "ui/*"
	}
	if isForbiddenInternalPath(lower) {
		return "internal/*"
	}
	if isForbiddenChoyDeepPath(lower) {
		return "choy_ui-deep"
	}
	return ""
}

func isForbiddenUIPath(lower string) bool {
	switch {
	case lower == "ui" || strings.HasPrefix(lower, "ui/"):
		return true
	// Current L2 vendor tree: components/vendor/ui/**
	case strings.Contains(lower, "/components/vendor/ui/") || strings.HasSuffix(lower, "/components/vendor/ui"):
		return true
	// Legacy / mistaken layouts still banned so domain cannot sneak past the rename.
	case strings.Contains(lower, "/components/ui/") || strings.HasSuffix(lower, "/components/ui"):
		return true
	case strings.Contains(lower, "/web/components/ui/") || strings.HasSuffix(lower, "/web/components/ui"):
		return true
	default:
		return false
	}
}

func isForbiddenInternalPath(lower string) bool {
	switch {
	case lower == "internal" || strings.HasPrefix(lower, "internal/"):
		// Relative "internal/..." from domain web is still a kit L3 leak risk.
		return true
	case strings.Contains(lower, "/components/internal/") || strings.HasSuffix(lower, "/components/internal"):
		return true
	case strings.Contains(lower, "/web/components/internal/") || strings.HasSuffix(lower, "/web/components/internal"):
		return true
	default:
		return false
	}
}

func isForbiddenChoyDeepPath(lower string) bool {
	// Domain modules must not deep-import choy_ui component trees (vendor/ui, internal, lib).
	// Public Choy* barrels land later (@/web); isolation gallery stays inside choy_ui.
	markers := []string{
		"@/choy_ui/",
		"@choysum-dev/choy_ui/",
		"/choy_ui/web/components/",
		"/choy_ui/web/lib/",
	}
	for _, m := range markers {
		if strings.Contains(lower, m) {
			return true
		}
	}
	if strings.HasPrefix(lower, "choy_ui/") && (strings.Contains(lower, "/components/") || strings.Contains(lower, "/lib/")) {
		return true
	}
	return false
}

// FormatForbiddenUiImportError builds a CI-readable multi-line error for violations.
func FormatForbiddenUiImportError(violations []ForbiddenUiImportViolation) error {
	if len(violations) == 0 {
		return nil
	}
	var b strings.Builder
	b.WriteString("forbidden Choy UI kit imports in domain web sources (use public Choy* APIs only):\n")
	for _, v := range violations {
		loc := v.SourcePath
		if v.Line > 0 {
			loc = fmt.Sprintf("%s:%d:%d", v.SourcePath, v.Line, v.Column)
		}
		fmt.Fprintf(&b, "  - %s imports %q (rule %s)\n", loc, v.SpecText, v.Rule)
	}
	return fmt.Errorf("%s", strings.TrimSuffix(b.String(), "\n"))
}

// AssertNoForbiddenUiImports is the disk entry used by typecheck and unit tests.
func AssertNoForbiddenUiImports(modulesPath, moduleName string) error {
	return CheckForbiddenUiImportsOnDisk(modulesPath, moduleName, nil)
}
