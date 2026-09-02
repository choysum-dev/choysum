// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package tsgoctx

import (
	"path/filepath"
	"strings"

	tsast "github.com/buke/typescript-go-internal/pkg/ast"
	tscore "github.com/buke/typescript-go-internal/pkg/core"
	tsparser "github.com/buke/typescript-go-internal/pkg/parser"
	tspath "github.com/buke/typescript-go-internal/pkg/tspath"
	"github.com/choysum-dev/choysum/internal/parser"
)

type ParseCtx struct {
	Path      string
	PathAlias map[string]string
	Source    *tsast.SourceFile
	Imports   map[string]*parser.Import
	Exports   map[string]*parser.Export
	lineMap   []tscore.TextPos
}

func Parse(pathAlias map[string]string, path string, content string) (*ParseCtx, error) {
	return ParseWithKind(pathAlias, path, content, 0, false)
}

func ParseWithKind(pathAlias map[string]string, path string, content string, forcedScriptKind tscore.ScriptKind, useForcedScriptKind bool) (*ParseCtx, error) {
	normalized := tspath.NormalizePath(path)
	if !filepath.IsAbs(normalized) {
		absPath, err := filepath.Abs(normalized)
		if err != nil {
			return nil, err
		}
		normalized = tspath.NormalizePath(absPath)
	}

	scriptKind := tscore.GetScriptKindFromFileName(normalized)
	if useForcedScriptKind {
		scriptKind = forcedScriptKind
	}

	source := tsparser.ParseSourceFile(tsast.SourceFileParseOptions{
		FileName: normalized,
		Path:     tspath.ToPath(normalized, "", true),
	}, content, scriptKind)

	ctx := &ParseCtx{
		Path:      normalized,
		PathAlias: pathAlias,
		Source:    source,
		Imports:   make(map[string]*parser.Import),
		Exports:   make(map[string]*parser.Export),
		lineMap:   source.ECMALineMap(),
	}

	for _, stmt := range source.Statements.Nodes {
		if stmt == nil {
			continue
		}
		if stmt.Kind == tsast.KindImportDeclaration || stmt.Kind == tsast.KindJSImportDeclaration {
			ctx.parseImport(stmt)
		}
	}

	for _, stmt := range source.Statements.Nodes {
		if stmt == nil {
			continue
		}
		ctx.parseExport(stmt)
	}

	return ctx, nil
}

func (c *ParseCtx) CurrentModuleSpecPath() string {
	if filepath.Ext(c.Path) == ".ts" {
		return strings.TrimSuffix(c.Path, ".ts")
	}
	return c.Path
}

func (c *ParseCtx) ResolveModuleSpec(moduleSpecifier string) string {
	moduleSpecifier = strings.Trim(moduleSpecifier, `"'`)
	if strings.HasPrefix(moduleSpecifier, "./") || strings.HasPrefix(moduleSpecifier, "../") {
		moduleSpecifier = filepath.Join(filepath.Dir(c.Path), moduleSpecifier)
	}
	moduleSpecifier = parser.ApplyPathAlias(c.PathAlias, moduleSpecifier)
	moduleSpecifier = strings.TrimSuffix(moduleSpecifier, ".ts")
	return moduleSpecifier
}

func (c *ParseCtx) NodeText(node *tsast.Node) string {
	if node == nil {
		return ""
	}
	start, end := node.Pos(), node.End()
	if start < 0 || end < start || end > len(c.Source.Text()) {
		return ""
	}
	return c.Source.Text()[start:end]
}

func (c *ParseCtx) LineColumn(pos int) (int, int) {
	line, col := tscore.PositionToLineAndByteOffset(pos, c.lineMap)
	return line + 1, col + 1
}

func (c *ParseCtx) parseImport(stmt *tsast.Node) {
	decl := stmt.AsImportDeclaration()
	moduleSpecifier := strings.Trim(decl.ModuleSpecifier.Text(), `"'`)
	if moduleSpecifier == "" {
		return
	}

	moduleSpecPath := c.ResolveModuleSpec(moduleSpecifier)
	line, col := c.LineColumn(stmt.Pos())
	moduleSpecText := strings.TrimSpace(c.NodeText(decl.ModuleSpecifier))
	importText := strings.TrimSpace(c.NodeText(stmt))

	makeImport := func(referenceIdent string, isTypeOnly bool) *parser.Import {
		return &parser.Import{
			ReferenceIdent:  referenceIdent,
			ModuleSpecPath:  moduleSpecPath,
			Text:            importText,
			Start:           stmt.Pos(),
			End:             stmt.End(),
			Line:            line,
			Column:          col,
			ModuleSpecText:  moduleSpecText,
			ModuleSpecStart: decl.ModuleSpecifier.Pos(),
			ModuleSpecEnd:   decl.ModuleSpecifier.End(),
			IsTypeOnly:      isTypeOnly,
		}
	}

	if decl.ImportClause == nil {
		c.Imports[parser.SideEffectImportMapKey(line, col)] = makeImport("*", false)
		return
	}

	importClauseNode := decl.ImportClause
	importClause := importClauseNode.AsImportClause()
	if defaultName := importClause.Name(); defaultName != nil {
		localName := defaultName.Text()
		if localName != "" {
			c.Imports[localName] = makeImport("default", parser.ImportBindingIsTypeOnly(importClauseNode, nil))
		}
	}

	namedBindings := importClause.NamedBindings
	if namedBindings == nil {
		return
	}

	if namedBindings.Kind == tsast.KindNamespaceImport {
		alias := namedBindings.AsNamespaceImport().Name().Text()
		if alias != "" {
			c.Imports[alias] = makeImport("*", parser.ImportBindingIsTypeOnly(importClauseNode, nil))
		}
		return
	}

	if namedBindings.Kind == tsast.KindNamedImports {
		elements := namedBindings.AsNamedImports().Elements.Nodes
		if len(elements) == 0 {
			c.Imports[parser.SideEffectImportMapKey(line, col)] = makeImport("*", parser.ImportBindingIsTypeOnly(importClauseNode, nil))
			return
		}
		for _, node := range elements {
			if node == nil || node.Kind != tsast.KindImportSpecifier {
				continue
			}
			spec := node.AsImportSpecifier()
			localName := spec.Name().Text()
			if localName == "" {
				continue
			}
			referenceIdent := localName
			if spec.PropertyName != nil {
				referenceIdent = spec.PropertyName.Text()
			}
			c.Imports[localName] = makeImport(referenceIdent, parser.ImportBindingIsTypeOnly(importClauseNode, node))
		}
	}
}

func (c *ParseCtx) ConvertReferenceWithModuleSpec(referenceIdent string) (string, string) {
	if imp, ok := c.Imports[referenceIdent]; ok {
		if imp.ReferenceIdent == "*" {
			return imp.ModuleSpecPath, ""
		}
		return imp.ModuleSpecPath, imp.ReferenceIdent
	}
	if defaultExport, ok := c.Exports["default"]; ok {
		if defaultExport.ReferenceIdent == referenceIdent {
			return defaultExport.ModuleSpecPath, "default"
		}
	}
	return c.CurrentModuleSpecPath(), referenceIdent
}

func (c *ParseCtx) parseExport(stmt *tsast.Node) {
	line, col := c.LineColumn(stmt.Pos())
	exportText := c.NodeText(stmt)
	newExport := func(referenceIdent string, moduleSpecPath string, isTypeOnly bool) *parser.Export {
		return &parser.Export{
			ReferenceIdent: referenceIdent,
			ModuleSpecPath: moduleSpecPath,
			Text:           exportText,
			Start:          stmt.Pos(),
			End:            stmt.End(),
			Line:           line,
			Column:         col,
			IsTypeOnly:     isTypeOnly,
		}
	}

	if stmt.Kind == tsast.KindExportAssignment {
		expr := stmt.AsExportAssignment().Expression
		if expr != nil {
			switch expr.Kind {
			case tsast.KindIdentifier:
				moduleSpec, ref := c.ConvertReferenceWithModuleSpec(expr.Text())
				c.Exports["default"] = newExport(ref, moduleSpec, false)
				return
			case tsast.KindPropertyAccessExpression:
				pa := expr.AsPropertyAccessExpression()
				moduleSpec, _ := c.ConvertReferenceWithModuleSpec(pa.Expression.Text())
				c.Exports["default"] = newExport(pa.Name().Text(), moduleSpec, false)
				return
			}
		}
		c.Exports["default"] = newExport("default", c.CurrentModuleSpecPath(), false)
		return
	}

	if stmt.Kind == tsast.KindExportDeclaration {
		decl := stmt.AsExportDeclaration()
		moduleSpecPath := ""
		if decl.ModuleSpecifier != nil {
			moduleSpecPath = c.ResolveModuleSpec(decl.ModuleSpecifier.Text())
		}

		if decl.ExportClause != nil {
			if decl.ExportClause.Kind == tsast.KindNamedExports {
				for _, node := range decl.ExportClause.AsNamedExports().Elements.Nodes {
					if node == nil || node.Kind != tsast.KindExportSpecifier {
						continue
					}
					spec := node.AsExportSpecifier()
					name := spec.Name().Text()
					if name == "" {
						continue
					}
					referenceIdent := name
					if spec.PropertyName != nil {
						referenceIdent = spec.PropertyName.Text()
					}
					isTypeOnly := parser.ExportBindingIsTypeOnly(stmt, node)
					if moduleSpecPath == "" {
						resolvedModuleSpec, resolvedRef := c.ConvertReferenceWithModuleSpec(referenceIdent)
						c.Exports[name] = newExport(resolvedRef, resolvedModuleSpec, isTypeOnly)
					} else {
						c.Exports[name] = newExport(referenceIdent, moduleSpecPath, isTypeOnly)
					}
				}
				return
			}

			if decl.ExportClause.Kind == tsast.KindNamespaceExport {
				name := decl.ExportClause.AsNamespaceExport().Name().Text()
				if name != "" {
					c.Exports[name] = newExport(name, moduleSpecPath, parser.ExportBindingIsTypeOnly(stmt, nil))
				}
				return
			}
		}

		if moduleSpecPath != "" {
			isTypeOnly := parser.ExportBindingIsTypeOnly(stmt, nil)
			wildcard := newExport("*", moduleSpecPath, isTypeOnly)
			if existing, ok := c.Exports["*"]; ok {
				existing.Wildcard = append(existing.Wildcard, wildcard)
			} else {
				c.Exports["*"] = &parser.Export{Wildcard: []*parser.Export{wildcard}, IsTypeOnly: isTypeOnly}
			}
		}
		return
	}

	if HasModifier(stmt, tsast.KindExportKeyword) && HasModifier(stmt, tsast.KindDefaultKeyword) {
		referenceIdent := "default"
		if name := ExportDeclarationName(stmt); name != "" {
			referenceIdent = name
		}
		c.Exports["default"] = newExport(referenceIdent, c.CurrentModuleSpecPath(), false)
		return
	}

	if HasModifier(stmt, tsast.KindExportKeyword) {
		if name := ExportDeclarationName(stmt); name != "" {
			c.Exports[name] = newExport(name, c.CurrentModuleSpecPath(), false)
		}
	}
}

func HasModifier(node *tsast.Node, kind tsast.Kind) bool {
	if node == nil {
		return false
	}
	mods := node.Modifiers()
	if mods == nil {
		return false
	}
	for _, modifier := range mods.Nodes {
		if modifier != nil && modifier.Kind == kind {
			return true
		}
	}
	return false
}

func ExportDeclarationName(stmt *tsast.Node) string {
	if stmt == nil {
		return ""
	}
	name := stmt.Name()
	if name != nil {
		return name.Text()
	}
	if stmt.Kind == tsast.KindVariableStatement {
		decls := stmt.AsVariableStatement().DeclarationList.AsVariableDeclarationList().Declarations.Nodes
		if len(decls) > 0 {
			decl := decls[0]
			if decl != nil {
				return decl.AsVariableDeclaration().Name().Text()
			}
		}
	}
	return ""
}

func MergeImports(dst map[string]*parser.Import, src map[string]*parser.Import) {
	if dst == nil || src == nil {
		return
	}
	for k, v := range src {
		dst[k] = v
	}
}

func MergeExports(dst map[string]*parser.Export, src map[string]*parser.Export) {
	if dst == nil || src == nil {
		return
	}
	for k, v := range src {
		if k == "*" && v != nil {
			if existing, ok := dst[k]; ok && existing != nil && len(existing.Wildcard) > 0 && len(v.Wildcard) > 0 {
				existing.Wildcard = append(existing.Wildcard, v.Wildcard...)
				continue
			}
		}
		dst[k] = v
	}
}
