// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package policy

import (
	"fmt"
	"path/filepath"
	"sort"
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
	return DefaultApplicationForModuleName(moduleName), true
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
	targetApp, _ = ResolveModuleApplication(targetModule, lookup)
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
	for _, imp := range result.DynamicImports {
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
			violations = appendValueImportViolation(
				violations,
				result.Path,
				sourceApp,
				importViolationKind(imp),
				imp,
				input.ModulesPath,
				input.Lookup,
			)
		}

		for _, imp := range result.DynamicImports {
			if imp == nil || imp.IsTypeOnly {
				continue
			}
			violations = appendValueImportViolation(
				violations,
				result.Path,
				sourceApp,
				"dynamic import",
				imp,
				input.ModulesPath,
				input.Lookup,
			)
		}

		for _, exp := range result.Exports {
			if exp == nil {
				continue
			}
			collectExport := func(entry *parser.Export) {
				if entry == nil || entry.IsTypeOnly || strings.TrimSpace(entry.ModuleSpecPath) == "" {
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
		return violations[i].ModuleSpecPath < violations[j].ModuleSpecPath
	})
	return violations
}

func importViolationKind(imp *parser.Import) string {
	if imp != nil && imp.IsDynamic {
		return "dynamic import"
	}
	return "import"
}

func appendValueImportViolation(
	violations []ImportBoundaryViolation,
	sourcePath string,
	sourceApp string,
	kind string,
	imp *parser.Import,
	modulesPath string,
	lookup ModuleApplicationLookup,
) []ImportBoundaryViolation {
	if imp == nil {
		return violations
	}
	specText := strings.TrimSpace(imp.ModuleSpecText)
	if specText == "" {
		specText = imp.Text
	}
	return appendImportViolation(
		violations,
		sourcePath,
		sourceApp,
		kind,
		imp.ModuleSpecPath,
		specText,
		imp.Line,
		imp.Column,
		modulesPath,
		lookup,
	)
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

// ModuleApplicationLookupWithDefault wraps a lookup and falls back to ResolveModuleApplication.
func ModuleApplicationLookupWithDefault(base ModuleApplicationLookup) ModuleApplicationLookup {
	return func(moduleName string) (string, bool) {
		if base != nil {
			if app, ok := base(moduleName); ok {
				return app, true
			}
		}
		return ResolveModuleApplication(moduleName, nil)
	}
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
