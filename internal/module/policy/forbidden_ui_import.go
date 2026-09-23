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

const (
	// kitHostModuleIsolation is the module that owns L2 vendor/ui during isolation.
	kitHostModuleIsolation = "choy_ui"
	// kitHostModuleCutover is the module that owns the kit after choy_ui merges into web.
	// Keep the relative tree as web/components/vendor/ui (do not rename vendor/ui again).
	kitHostModuleCutover = "web"
)

// isKitHostModule reports modules allowed to import reka-ui / vendor/ui / kit internals.
// Isolation: only choy_ui. After the kit merges into web, return true for web instead.
func isKitHostModule(moduleName string) bool {
	switch strings.TrimSpace(moduleName) {
	case kitHostModuleIsolation:
		return true
	case kitHostModuleCutover:
		// Product web still ships Element Plus until cutover; do not exempt it yet
		// or domain-facing web code could import Reka unnoticed. Return true once
		// vendor/ui lives under modules/web and Element Plus is removed.
		return false
	default:
		return false
	}
}

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
	if isKitHostModule(moduleName) {
		return nil
	}

	var violations []ForbiddenUiImportViolation
	for _, result := range parserResults {
		if result == nil || !IsModuleWebSource(moduleRoot, result.Path) {
			continue
		}
		for _, imp := range result.Imports {
			if imp == nil || imp.IsTypeOnly {
				continue
			}
			violations = appendForbiddenUiSpec(violations, result.Path, imp.ModuleSpecText, imp.Line, imp.Column)
		}
		for _, imp := range result.DynamicImports {
			if imp == nil || imp.IsTypeOnly {
				continue
			}
			violations = appendForbiddenUiSpec(violations, result.Path, imp.ModuleSpecText, imp.Line, imp.Column)
		}
		for _, exp := range result.Exports {
			violations = appendForbiddenUiExports(violations, result.Path, exp)
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
	moduleRoot = filepath.Clean(strings.TrimSpace(moduleRoot))
	path = filepath.Clean(strings.TrimSpace(path))
	if moduleRoot == "" || path == "" || moduleRoot == "." {
		return false
	}
	prefix := moduleRoot + string(filepath.Separator)
	if !strings.HasPrefix(path, prefix) {
		return false
	}
	rel := filepath.ToSlash(strings.TrimPrefix(path, prefix))
	return strings.HasPrefix(rel, "web/") || rel == "web"
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

// appendForbiddenUiExports records runtime re-exports (export … from / export * from).
// Type-only re-exports are skipped; Export has no ModuleSpecText, so ModuleSpecPath is used.
func appendForbiddenUiExports(
	violations []ForbiddenUiImportViolation,
	sourcePath string,
	exp *parser.Export,
) []ForbiddenUiImportViolation {
	if exp == nil || exp.IsTypeOnly {
		return violations
	}
	if len(exp.Wildcard) > 0 {
		sawSpec := false
		for _, wild := range exp.Wildcard {
			if wild == nil || wild.IsTypeOnly {
				continue
			}
			if strings.TrimSpace(wild.ModuleSpecPath) == "" {
				continue
			}
			sawSpec = true
			violations = appendForbiddenUiSpec(violations, sourcePath, wild.ModuleSpecPath, wild.Line, wild.Column)
		}
		if sawSpec {
			return violations
		}
		// Some parser shapes keep the specifier only on the parent Export.
		return appendForbiddenUiSpec(violations, sourcePath, exp.ModuleSpecPath, exp.Line, exp.Column)
	}
	return appendForbiddenUiSpec(violations, sourcePath, exp.ModuleSpecPath, exp.Line, exp.Column)
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
	default:
		return false
	}
}

func isForbiddenChoyDeepPath(lower string) bool {
	// Isolation: domain must not deep-import the choy_ui kit trees.
	// After cutover, continue matching /components/vendor/ui via isForbiddenUIPath;
	// extend @/web/web/{components/vendor,components/internal,lib} bans once O*
	// deep imports are replaced by public Choy* barrels.
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
