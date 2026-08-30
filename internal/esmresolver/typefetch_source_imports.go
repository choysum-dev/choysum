// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package esmresolver

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	tsast "github.com/buke/typescript-go-internal/pkg/ast"
	tscore "github.com/buke/typescript-go-internal/pkg/core"
	tsparser "github.com/buke/typescript-go-internal/pkg/parser"
)

type typeFetchDependencyTarget struct {
	name    string
	version string
}

// collectModuleSourceImportSpecifiers walks a module tree and returns bare
// package import specifiers used by TypeScript sources (including subpaths
// such as "echarts/core").
func collectModuleSourceImportSpecifiers(moduleDir string) ([]string, error) {
	moduleDir = strings.TrimSpace(moduleDir)
	if moduleDir == "" {
		return nil, nil
	}
	seen := make(map[string]struct{})
	var out []string
	err := filepath.WalkDir(moduleDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		name := d.Name()
		if d.IsDir() {
			if shouldSkipTypeFetchSourceDir(name) {
				return filepath.SkipDir
			}
			return nil
		}
		if !isTypeFetchSourceFile(name) {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		for _, spec := range collectSourceBareImportSpecifiers(string(data), path) {
			if _, ok := seen[spec]; ok {
				continue
			}
			seen[spec] = struct{}{}
			out = append(out, spec)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(out)
	return out, nil
}

func shouldSkipTypeFetchSourceDir(name string) bool {
	switch name {
	case "node_modules", "dist", "build", "coverage", "tmp", ".git", ".choysum", ".turbo", ".vite", ".cache":
		return true
	}
	return false
}

func isTypeFetchSourceFile(name string) bool {
	lower := strings.ToLower(name)
	switch {
	case strings.HasSuffix(lower, ".ts"), strings.HasSuffix(lower, ".tsx"), strings.HasSuffix(lower, ".mts"), strings.HasSuffix(lower, ".cts"):
		return !strings.HasSuffix(lower, ".d.ts") && !strings.HasSuffix(lower, ".d.mts") && !strings.HasSuffix(lower, ".d.cts")
	default:
		return false
	}
}

func collectSourceBareImportSpecifiers(content string, fileName string) []string {
	if strings.TrimSpace(content) == "" {
		return nil
	}
	fileName = strings.TrimSpace(fileName)
	if fileName == "" || !filepath.IsAbs(fileName) {
		base := filepath.Base(fileName)
		if base == "" || base == "." || base == string(filepath.Separator) {
			base = "source.ts"
		}
		fileName = "/virtual/" + filepath.ToSlash(base)
	}
	scriptKind := tscore.GetScriptKindFromFileName(fileName)
	source := tsparser.ParseSourceFile(tsast.SourceFileParseOptions{
		FileName: fileName,
	}, content, scriptKind)
	if source == nil || source.Statements == nil {
		return nil
	}

	seen := make(map[string]struct{})
	var out []string
	add := func(raw string) {
		spec := strings.TrimSpace(raw)
		if spec == "" {
			return
		}
		if !isBarePackageImportSpecifier(spec) {
			return
		}
		if _, ok := seen[spec]; ok {
			return
		}
		seen[spec] = struct{}{}
		out = append(out, spec)
	}

	var walk func(node *tsast.Node)
	walk = func(node *tsast.Node) {
		if node == nil {
			return
		}
		switch node.Kind {
		case tsast.KindImportDeclaration, tsast.KindJSImportDeclaration:
			if decl := node.AsImportDeclaration(); decl != nil && decl.ModuleSpecifier != nil {
				add(decl.ModuleSpecifier.Text())
			}
		case tsast.KindExportDeclaration:
			if decl := node.AsExportDeclaration(); decl != nil && decl.ModuleSpecifier != nil {
				add(decl.ModuleSpecifier.Text())
			}
		case tsast.KindCallExpression:
			call := node.AsCallExpression()
			if call != nil && call.Expression != nil && call.Expression.Kind == tsast.KindImportKeyword && call.Arguments != nil && len(call.Arguments.Nodes) > 0 {
				arg := call.Arguments.Nodes[0]
				if arg != nil && (arg.Kind == tsast.KindStringLiteral || arg.Kind == tsast.KindNoSubstitutionTemplateLiteral) {
					add(arg.Text())
				}
			}
		case tsast.KindImportType:
			importType := node.AsImportTypeNode()
			if importType != nil && importType.Argument != nil && importType.Argument.Kind == tsast.KindLiteralType {
				if lit := importType.Argument.AsLiteralTypeNode(); lit != nil && lit.Literal != nil {
					if lit.Literal.Kind == tsast.KindStringLiteral || lit.Literal.Kind == tsast.KindNoSubstitutionTemplateLiteral {
						add(lit.Literal.Text())
					}
				}
			}
		}
		node.ForEachChild(func(child *tsast.Node) bool {
			walk(child)
			return false
		})
	}
	for _, stmt := range source.Statements.Nodes {
		walk(stmt)
	}
	return out
}

func isBarePackageImportSpecifier(spec string) bool {
	spec = strings.TrimSpace(spec)
	if spec == "" || strings.Contains(spec, "..") {
		return false
	}
	switch {
	case strings.HasPrefix(spec, "./"), strings.HasPrefix(spec, "../"), strings.HasPrefix(spec, "/"):
		return false
	case strings.HasPrefix(spec, "node:"), strings.HasPrefix(spec, "data:"), strings.HasPrefix(spec, "http://"), strings.HasPrefix(spec, "https://"):
		return false
	case strings.HasPrefix(spec, "@/"), strings.HasPrefix(spec, "~/"), strings.HasPrefix(spec, "#"):
		return false
	case isAssetLikeImportSpecifier(spec):
		return false
	default:
		return true
	}
}

func isAssetLikeImportSpecifier(spec string) bool {
	lower := strings.ToLower(strings.TrimSpace(spec))
	for _, ext := range []string{
		".css", ".scss", ".sass", ".less",
		".svg", ".png", ".jpg", ".jpeg", ".gif", ".webp", ".ico",
		".woff", ".woff2", ".ttf", ".eot",
		".mp3", ".mp4", ".wasm",
	} {
		if strings.HasSuffix(lower, ext) {
			return true
		}
	}
	return false
}

// shouldSkipTypeFetchSubpathSpecifier skips subpaths that are intentionally
// covered by ambient stubs (locales/plugins) or otherwise lack stable .d.ts
// on the ESM upstream.
func shouldSkipTypeFetchSubpathSpecifier(spec string) bool {
	rootPkg, subpath, ok := splitBarePackageSpecifier(spec)
	if !ok || subpath == "" {
		return false
	}
	switch rootPkg {
	case "dayjs":
		return strings.HasPrefix(subpath, "locale/") || strings.HasPrefix(subpath, "plugin/")
	case "element-plus":
		return strings.HasPrefix(subpath, "es/locale/lang/")
	default:
		return false
	}
}

// splitBarePackageSpecifier splits "echarts/core" → ("echarts","core") and
// "@scope/pkg/sub" → ("@scope/pkg","sub"). Bare roots return subpath "".
func splitBarePackageSpecifier(specifier string) (rootPkg string, subpath string, ok bool) {
	specifier = strings.TrimSpace(specifier)
	if specifier == "" || !isBarePackageImportSpecifier(specifier) {
		return "", "", false
	}
	if strings.HasPrefix(specifier, "@") {
		parts := strings.Split(specifier, "/")
		if len(parts) < 2 || strings.TrimSpace(parts[0]) == "@" || strings.TrimSpace(parts[1]) == "" {
			return "", "", false
		}
		rootPkg = parts[0] + "/" + parts[1]
		if len(parts) == 2 {
			return rootPkg, "", true
		}
		return rootPkg, strings.Join(parts[2:], "/"), true
	}
	base, rest, found := strings.Cut(specifier, "/")
	base = strings.TrimSpace(base)
	if base == "" {
		return "", "", false
	}
	if !found {
		return base, "", true
	}
	return base, rest, true
}

func subpathTypeFetchTargets(deps map[string]string, imports []string) []typeFetchDependencyTarget {
	if len(deps) == 0 || len(imports) == 0 {
		return nil
	}
	seen := make(map[string]struct{})
	var targets []typeFetchDependencyTarget
	for _, spec := range imports {
		if isAssetLikeImportSpecifier(spec) || shouldSkipTypeFetchSubpathSpecifier(spec) {
			continue
		}
		rootPkg, subpath, ok := splitBarePackageSpecifier(spec)
		if !ok || subpath == "" {
			continue
		}
		verRange, ok := deps[rootPkg]
		if !ok {
			continue
		}
		if strings.HasPrefix(verRange, "workspace:") || strings.HasPrefix(verRange, "file:") || strings.HasPrefix(verRange, "link:") {
			continue
		}
		version := strings.TrimLeft(verRange, "^~=> ")
		if version == "" || version == "*" {
			continue
		}
		if _, exists := seen[spec]; exists {
			continue
		}
		seen[spec] = struct{}{}
		targets = append(targets, typeFetchDependencyTarget{name: spec, version: version})
	}
	sort.Slice(targets, func(i, j int) bool {
		return targets[i].name < targets[j].name
	})
	return targets
}

func mergeTypeFetchTargets(base []typeFetchDependencyTarget, extra []typeFetchDependencyTarget) []typeFetchDependencyTarget {
	if len(extra) == 0 {
		return base
	}
	seen := make(map[string]struct{}, len(base)+len(extra))
	out := make([]typeFetchDependencyTarget, 0, len(base)+len(extra))
	for _, target := range base {
		key := target.name + "@" + target.version
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, target)
	}
	for _, target := range extra {
		key := target.name + "@" + target.version
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, target)
	}
	return out
}

// typeFetchDiscoverSpec builds the esm.sh discovery path for a package or
// package subpath: "echarts" → "echarts@6.1.0", "echarts/core" → "echarts@6.1.0/core".
func typeFetchDiscoverSpec(specifier, version string) string {
	specifier = strings.TrimSpace(specifier)
	version = strings.TrimSpace(version)
	if specifier == "" {
		return ""
	}
	rootPkg, subpath, ok := splitBarePackageSpecifier(specifier)
	if !ok {
		if version == "" {
			return specifier
		}
		return specifier + "@" + version
	}
	if version == "" {
		if subpath == "" {
			return rootPkg
		}
		return rootPkg + "/" + subpath
	}
	spec := rootPkg + "@" + version
	if subpath != "" {
		spec += "/" + subpath
	}
	return spec
}

// isValidTsconfigPathsMappingKey reports whether a TypeFetchResult.Package
// should become a compilerOptions.paths key (bare package or package subpath).
func isValidTsconfigPathsMappingKey(pkg string) bool {
	pkg = strings.TrimSpace(pkg)
	if pkg == "" {
		return false
	}
	if isStaleGeneratedTsconfigPathsKey(pkg) {
		return false
	}
	if strings.HasPrefix(pkg, "@") {
		parts := strings.Split(pkg[1:], "/")
		return len(parts) >= 2 && parts[0] != "" && parts[1] != ""
	}
	return true
}

// isStaleGeneratedTsconfigPathsKey reports path keys that older type-fetch
// runs wrote for every cached .d.ts (versioned specifiers and declaration
// filenames). Those entries make the IDE open thousands of cache files.
// Aliases such as "@/*" and bare packages such as "@vicons/material" stay.
func isStaleGeneratedTsconfigPathsKey(pkg string) bool {
	pkg = strings.TrimSpace(pkg)
	if pkg == "" {
		return false
	}
	if strings.Contains(pkg, "://") {
		return true
	}
	lower := strings.ToLower(pkg)
	if strings.Contains(lower, ".d.ts") || strings.Contains(lower, ".d.mts") || strings.Contains(lower, ".d.cts") {
		return true
	}
	if strings.HasPrefix(pkg, "@") {
		return strings.Contains(pkg[1:], "@")
	}
	return strings.Contains(pkg, "@")
}
