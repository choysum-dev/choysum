// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package vueparser

import (
	"path/filepath"
	"strings"

	tsast "github.com/buke/typescript-go-internal/pkg/ast"
	tscore "github.com/buke/typescript-go-internal/pkg/core"
	tsparser "github.com/buke/typescript-go-internal/pkg/parser"
	tspath "github.com/buke/typescript-go-internal/pkg/tspath"
	"github.com/choysum-dev/choysum/internal/parser"
)

type tsParseCtx struct {
	path      string
	pathAlias map[string]string
	source    *tsast.SourceFile
	imports   map[string]*parser.Import
	exports   map[string]*parser.Export
	lineMap   []tscore.TextPos
}

func parseTSGoCtx(pathAlias map[string]string, path string, content string) (*tsParseCtx, error) {
	return parseTSGoCtxWithKind(pathAlias, path, content, 0, false)
}

func parseTSGoCtxWithKind(pathAlias map[string]string, path string, content string, forcedScriptKind tscore.ScriptKind, useForcedScriptKind bool) (*tsParseCtx, error) {
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

	ctx := &tsParseCtx{
		path:      normalized,
		pathAlias: pathAlias,
		source:    source,
		imports:   make(map[string]*parser.Import),
		exports:   make(map[string]*parser.Export),
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

func (c *tsParseCtx) currentModuleSpecPath() string {
	if filepath.Ext(c.path) == ".ts" {
		return strings.TrimSuffix(c.path, ".ts")
	}
	return c.path
}

func (c *tsParseCtx) resolveModuleSpec(moduleSpecifier string) string {
	moduleSpecifier = strings.Trim(moduleSpecifier, `"'`)
	if strings.HasPrefix(moduleSpecifier, "./") || strings.HasPrefix(moduleSpecifier, "../") {
		moduleSpecifier = filepath.Join(filepath.Dir(c.path), moduleSpecifier)
	}
	moduleSpecifier = parser.ApplyPathAlias(c.pathAlias, moduleSpecifier)
	moduleSpecifier = strings.TrimSuffix(moduleSpecifier, ".ts")
	return moduleSpecifier
}

func (c *tsParseCtx) nodeText(node *tsast.Node) string {
	if node == nil {
		return ""
	}
	start, end := node.Pos(), node.End()
	if start < 0 || end < start || end > len(c.source.Text()) {
		return ""
	}
	return c.source.Text()[start:end]
}

func (c *tsParseCtx) lineColumn(pos int) (int, int) {
	line, col := tscore.PositionToLineAndByteOffset(pos, c.lineMap)
	return line + 1, col + 1
}

func (c *tsParseCtx) parseImport(stmt *tsast.Node) {
	decl := stmt.AsImportDeclaration()
	moduleSpecifier := strings.Trim(decl.ModuleSpecifier.Text(), `"'`)
	if moduleSpecifier == "" {
		return
	}

	moduleSpecPath := c.resolveModuleSpec(moduleSpecifier)
	line, col := c.lineColumn(stmt.Pos())
	moduleSpecText := strings.TrimSpace(c.nodeText(decl.ModuleSpecifier))
	importText := strings.TrimSpace(c.nodeText(stmt))

	makeImport := func(referenceIdent string) *parser.Import {
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
		}
	}

	if decl.ImportClause == nil {
		return
	}

	importClause := decl.ImportClause.AsImportClause()
	if defaultName := importClause.Name(); defaultName != nil {
		localName := defaultName.Text()
		if localName != "" {
			c.imports[localName] = makeImport("default")
		}
	}

	namedBindings := importClause.NamedBindings
	if namedBindings == nil {
		return
	}

	if namedBindings.Kind == tsast.KindNamespaceImport {
		alias := namedBindings.AsNamespaceImport().Name().Text()
		if alias != "" {
			c.imports[alias] = makeImport("*")
		}
		return
	}

	if namedBindings.Kind == tsast.KindNamedImports {
		for _, node := range namedBindings.AsNamedImports().Elements.Nodes {
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
			c.imports[localName] = makeImport(referenceIdent)
		}
	}
}

func (c *tsParseCtx) convertReferenceWithModuleSpec(referenceIdent string) (string, string) {
	if imp, ok := c.imports[referenceIdent]; ok {
		if imp.ReferenceIdent == "*" {
			return imp.ModuleSpecPath, ""
		}
		return imp.ModuleSpecPath, imp.ReferenceIdent
	}
	if defaultExport, ok := c.exports["default"]; ok {
		if defaultExport.ReferenceIdent == referenceIdent {
			return defaultExport.ModuleSpecPath, "default"
		}
	}
	return c.currentModuleSpecPath(), referenceIdent
}

func (c *tsParseCtx) parseExport(stmt *tsast.Node) {
	line, col := c.lineColumn(stmt.Pos())
	exportText := c.nodeText(stmt)
	newExport := func(referenceIdent string, moduleSpecPath string) *parser.Export {
		return &parser.Export{
			ReferenceIdent: referenceIdent,
			ModuleSpecPath: moduleSpecPath,
			Text:           exportText,
			Start:          stmt.Pos(),
			End:            stmt.End(),
			Line:           line,
			Column:         col,
		}
	}

	if stmt.Kind == tsast.KindExportAssignment {
		expr := stmt.AsExportAssignment().Expression
		if expr != nil {
			switch expr.Kind {
			case tsast.KindIdentifier:
				moduleSpec, ref := c.convertReferenceWithModuleSpec(expr.Text())
				c.exports["default"] = newExport(ref, moduleSpec)
				return
			case tsast.KindPropertyAccessExpression:
				pa := expr.AsPropertyAccessExpression()
				moduleSpec, _ := c.convertReferenceWithModuleSpec(pa.Expression.Text())
				c.exports["default"] = newExport(pa.Name().Text(), moduleSpec)
				return
			}
		}
		c.exports["default"] = newExport("default", c.currentModuleSpecPath())
		return
	}

	if stmt.Kind == tsast.KindExportDeclaration {
		decl := stmt.AsExportDeclaration()
		moduleSpecPath := ""
		if decl.ModuleSpecifier != nil {
			moduleSpecPath = c.resolveModuleSpec(decl.ModuleSpecifier.Text())
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
					if moduleSpecPath == "" {
						resolvedModuleSpec, resolvedRef := c.convertReferenceWithModuleSpec(referenceIdent)
						c.exports[name] = newExport(resolvedRef, resolvedModuleSpec)
					} else {
						c.exports[name] = newExport(referenceIdent, moduleSpecPath)
					}
				}
				return
			}

			if decl.ExportClause.Kind == tsast.KindNamespaceExport {
				name := decl.ExportClause.AsNamespaceExport().Name().Text()
				if name != "" {
					c.exports[name] = newExport(name, moduleSpecPath)
				}
				return
			}
		}

		if moduleSpecPath != "" {
			wildcard := newExport("*", moduleSpecPath)
			if existing, ok := c.exports["*"]; ok {
				existing.Wildcard = append(existing.Wildcard, wildcard)
			} else {
				c.exports["*"] = &parser.Export{Wildcard: []*parser.Export{wildcard}}
			}
		}
		return
	}

	if hasModifier(stmt, tsast.KindExportKeyword) && hasModifier(stmt, tsast.KindDefaultKeyword) {
		referenceIdent := "default"
		if name := exportDeclarationName(stmt); name != "" {
			referenceIdent = name
		}
		c.exports["default"] = newExport(referenceIdent, c.currentModuleSpecPath())
		return
	}

	if hasModifier(stmt, tsast.KindExportKeyword) {
		if name := exportDeclarationName(stmt); name != "" {
			c.exports[name] = newExport(name, c.currentModuleSpecPath())
		}
	}
}

func hasModifier(node *tsast.Node, kind tsast.Kind) bool {
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

func exportDeclarationName(stmt *tsast.Node) string {
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

func mergeImports(dst map[string]*parser.Import, src map[string]*parser.Import) {
	if dst == nil || src == nil {
		return
	}
	for k, v := range src {
		dst[k] = v
	}
}

func mergeExports(dst map[string]*parser.Export, src map[string]*parser.Export) {
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
