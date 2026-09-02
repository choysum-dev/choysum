// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package policy

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/choysum-dev/choysum/internal/parser"
	"github.com/choysum-dev/choysum/pkg/meta"
)

const coreApplication = "core"

// ModuleApplicationLookup resolves a module directory name to its application key.
type ModuleApplicationLookup func(moduleName string) (application string, ok bool)

// ImportBoundaryViolation describes one cross-app value import in service source.
type ImportBoundaryViolation struct {
	SourcePath        string
	Line              int
	Column            int
	SourceApplication string
	TargetApplication string
	TargetModule      string
	ModuleSpecPath    string
	ModuleSpecText    string
	Kind              string
}

// ServiceImportBoundaryInput configures a service-tree import boundary scan.
type ServiceImportBoundaryInput struct {
	ModulesPath       string
	ModuleRoot        string
	SourceApplication string
	ParserResults     []*parser.ParserResult
	Lookup            ModuleApplicationLookup
}

var dynamicImportPattern = regexp.MustCompile(`import\s*\(\s*['"]([^'"]+)['"]\s*\)`)

// ModuleNameFromModulesPath returns the top-level module directory for an absolute path.
func ModuleNameFromModulesPath(modulesPath, absPath string) string {
	modulesPath = strings.TrimSpace(modulesPath)
	absPath = strings.TrimSpace(absPath)
	if modulesPath == "" || absPath == "" {
		return ""
	}
	rel, err := filepath.Rel(filepath.Clean(modulesPath), filepath.Clean(absPath))
	if err != nil {
		return ""
	}
	rel = filepath.ToSlash(rel)
	if rel == "." || strings.HasPrefix(rel, "../") || rel == ".." {
		return ""
	}
	parts := strings.Split(rel, "/")
	if len(parts) == 0 || strings.TrimSpace(parts[0]) == "" {
		return ""
	}
	return strings.TrimSpace(parts[0])
}

// DefaultApplicationForModuleName maps module directory names to application keys
// when package metadata is unavailable (tests / pre-install).
func DefaultApplicationForModuleName(moduleName string) string {
	moduleName = strings.TrimSpace(moduleName)
	switch moduleName {
	case "partner_bank", "partner_commercial":
		return "partner"
	case "":
		return ""
	default:
		return moduleName
	}
}

// ResolveModuleApplication resolves an application key for a module directory name.
func ResolveModuleApplication(moduleName string, lookup ModuleApplicationLookup) (string, bool) {
	moduleName = strings.TrimSpace(moduleName)
	if moduleName == "" {
		return "", false
	}
	if lookup != nil {
		if app, ok := lookup(moduleName); ok {
			app = strings.TrimSpace(app)
			if app != "" {
				return app, true
			}
		}
	}
	app := DefaultApplicationForModuleName(moduleName)
	if app == "" {
		return "", false
	}
	return app, true
}

// IsModuleServiceSource reports whether path is under moduleRoot/service/.
func IsModuleServiceSource(moduleRoot, path string) bool {
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
	return strings.HasPrefix(rel, "service/") || rel == "service"
}

func pathUnderModules(modulesPath, specPath string) bool {
	if strings.TrimSpace(modulesPath) == "" || strings.TrimSpace(specPath) == "" {
		return false
	}
	if !filepath.IsAbs(specPath) {
		return false
	}
	rel, err := filepath.Rel(filepath.Clean(modulesPath), filepath.Clean(specPath))
	if err != nil {
		return false
	}
	rel = filepath.ToSlash(rel)
	return rel != ".." && !strings.HasPrefix(rel, "../")
}

func resolveTarget(modulesPath, moduleSpecPath string, lookup ModuleApplicationLookup) (targetModule, targetApp string, ok bool) {
	moduleSpecPath = strings.TrimSpace(moduleSpecPath)
	if moduleSpecPath == "" || !pathUnderModules(modulesPath, moduleSpecPath) {
		return "", "", false
	}
	targetModule = ModuleNameFromModulesPath(modulesPath, moduleSpecPath)
	if targetModule == "" {
		return "", "", false
	}
	targetApp, ok = ResolveModuleApplication(targetModule, lookup)
	if !ok {
		return "", "", false
	}
	return targetModule, targetApp, true
}

func isAllowedTarget(sourceApp, targetApp string) bool {
	sourceApp = strings.TrimSpace(sourceApp)
	targetApp = strings.TrimSpace(targetApp)
	if sourceApp == "" || targetApp == "" {
		return true
	}
	if targetApp == coreApplication {
		return true
	}
	return sourceApp == targetApp
}

func appendImportViolation(
	violations []ImportBoundaryViolation,
	sourcePath string,
	sourceApp string,
	kind string,
	moduleSpecPath string,
	moduleSpecText string,
	line int,
	column int,
	modulesPath string,
	lookup ModuleApplicationLookup,
) []ImportBoundaryViolation {
	targetModule, targetApp, ok := resolveTarget(modulesPath, moduleSpecPath, lookup)
	if !ok {
		return violations
	}
	if isAllowedTarget(sourceApp, targetApp) {
		return violations
	}
	return append(violations, ImportBoundaryViolation{
		SourcePath:        sourcePath,
		Line:              line,
		Column:            column,
		SourceApplication: sourceApp,
		TargetApplication: targetApp,
		TargetModule:      targetModule,
		ModuleSpecPath:    moduleSpecPath,
		ModuleSpecText:    strings.TrimSpace(moduleSpecText),
		Kind:              kind,
	})
}

// CollectModuleNamesFromParserResult adds module directory names referenced by imports/exports.
func CollectModuleNamesFromParserResult(modulesPath string, result *parser.ParserResult, names map[string]struct{}) {
	if result == nil || names == nil {
		return
	}
	collect := func(specPath string) {
		name := ModuleNameFromModulesPath(modulesPath, specPath)
		if name != "" {
			names[name] = struct{}{}
		}
	}
	for _, imp := range result.Imports {
		if imp == nil {
			continue
		}
		collect(imp.ModuleSpecPath)
	}
	for _, exp := range result.Exports {
		if exp == nil {
			continue
		}
		collect(exp.ModuleSpecPath)
		for _, wildcard := range exp.Wildcard {
			if wildcard != nil {
				collect(wildcard.ModuleSpecPath)
			}
		}
	}
}

// CheckServiceImportBoundary scans parser results for cross-app value imports under service/.
func CheckServiceImportBoundary(input ServiceImportBoundaryInput) []ImportBoundaryViolation {
	sourceApp := strings.TrimSpace(input.SourceApplication)
	if sourceApp == "" || strings.TrimSpace(input.ModuleRoot) == "" {
		return nil
	}

	var violations []ImportBoundaryViolation
	for _, result := range input.ParserResults {
		if result == nil || !IsModuleServiceSource(input.ModuleRoot, result.Path) {
			continue
		}

		for _, imp := range result.Imports {
			if imp == nil || imp.IsTypeOnly {
				continue
			}
			specText := strings.TrimSpace(imp.ModuleSpecText)
			if specText == "" {
				specText = imp.Text
			}
			violations = appendImportViolation(
				violations,
				result.Path,
				sourceApp,
				"import",
				imp.ModuleSpecPath,
				specText,
				imp.Line,
				imp.Column,
				input.ModulesPath,
				input.Lookup,
			)
		}

		for _, exp := range result.Exports {
			if exp == nil {
				continue
			}
			collectExport := func(entry *parser.Export) {
				if entry == nil || strings.TrimSpace(entry.ModuleSpecPath) == "" {
					return
				}
				specText := entry.Text
				violations = appendImportViolation(
					violations,
					result.Path,
					sourceApp,
					"export",
					entry.ModuleSpecPath,
					specText,
					entry.Line,
					entry.Column,
					input.ModulesPath,
					input.Lookup,
				)
			}
			collectExport(exp)
			for _, wildcard := range exp.Wildcard {
				collectExport(wildcard)
			}
		}

		violations = appendDynamicImportViolations(
			violations,
			result,
			sourceApp,
			input.ModulesPath,
			input.Lookup,
		)
	}
	return violations
}

func appendDynamicImportViolations(
	violations []ImportBoundaryViolation,
	result *parser.ParserResult,
	sourceApp string,
	modulesPath string,
	lookup ModuleApplicationLookup,
) []ImportBoundaryViolation {
	content := result.RawContent
	if content == "" {
		content = result.Content
	}
	if strings.TrimSpace(content) == "" {
		return violations
	}

	for _, match := range dynamicImportPattern.FindAllStringSubmatchIndex(content, -1) {
		if len(match) < 4 {
			continue
		}
		fullStart := match[0]
		specStart, specEnd := match[2], match[3]
		spec := content[specStart:specEnd]
		if isTypeOnlyDynamicImport(content, fullStart) {
			continue
		}
		moduleSpecPath := resolveDynamicImportSpec(modulesPath, result.Path, spec)
		line, column := lineColumnAt(content, fullStart)
		violations = appendImportViolation(
			violations,
			result.Path,
			sourceApp,
			"dynamic import",
			moduleSpecPath,
			spec,
			line,
			column,
			modulesPath,
			lookup,
		)
	}
	return violations
}

func isTypeOnlyDynamicImport(content string, importStart int) bool {
	if importStart <= 0 {
		return false
	}
	prefix := strings.TrimSpace(content[:importStart])
	return strings.HasSuffix(prefix, "typeof")
}

func resolveDynamicImportSpec(modulesPath, sourcePath, spec string) string {
	spec = strings.Trim(strings.TrimSpace(spec), `"'`)
	if spec == "" {
		return ""
	}
	switch {
	case strings.HasPrefix(spec, "./") || strings.HasPrefix(spec, "../"):
		spec = filepath.Join(filepath.Dir(sourcePath), spec)
	case strings.HasPrefix(spec, "@/"):
		spec = filepath.Join(strings.TrimSpace(modulesPath), strings.TrimPrefix(spec, "@/"))
	}
	spec = strings.TrimSuffix(spec, ".ts")
	return filepath.Clean(spec)
}

func lineColumnAt(content string, offset int) (line int, column int) {
	if offset < 0 {
		offset = 0
	}
	if offset > len(content) {
		offset = len(content)
	}
	line = 1
	column = 1
	for i := 0; i < offset; i++ {
		if content[i] == '\n' {
			line++
			column = 1
			continue
		}
		column++
	}
	return line, column
}

// FormatImportBoundaryError formats violations as a hard-fail install/build error.
func FormatImportBoundaryError(violations []ImportBoundaryViolation) error {
	if len(violations) == 0 {
		return nil
	}
	var b strings.Builder
	b.WriteString("cross-app service import boundary violation")
	for _, v := range violations {
		spec := strings.TrimSpace(v.ModuleSpecText)
		if spec == "" {
			spec = v.ModuleSpecPath
		}
		fmt.Fprintf(
			&b,
			"\n- %s:%d:%d: %s (%s -> %s) via %q; use dial('app.Model') or createServiceByModel with import type",
			v.SourcePath,
			v.Line,
			v.Column,
			v.Kind,
			v.SourceApplication,
			v.TargetApplication,
			spec,
		)
	}
	return fmt.Errorf("%s", b.String())
}

// ModuleApplicationLookupFromMap adapts a module-name map for boundary checks.
func ModuleApplicationLookupFromMap(appByModule map[string]string) ModuleApplicationLookup {
	return func(moduleName string) (string, bool) {
		if appByModule == nil {
			return "", false
		}
		app, ok := appByModule[strings.TrimSpace(moduleName)]
		app = strings.TrimSpace(app)
		if !ok || app == "" {
			return "", false
		}
		return app, true
	}
}

// ModuleApplicationLookupFromModule builds a lookup using the module being built.
func ModuleApplicationLookupFromModule(module *meta.Module) ModuleApplicationLookup {
	if module == nil {
		return nil
	}
	app := strings.TrimSpace(module.ApplicationStr)
	name := strings.TrimSpace(module.Name)
	if app == "" || name == "" {
		return nil
	}
	return ModuleApplicationLookupFromMap(map[string]string{name: app})
}
