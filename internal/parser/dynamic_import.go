// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package parser

import (
	"strings"

	tsast "github.com/buke/typescript-go-internal/v7/pkg/ast"
)

func (c *tsgoImportExportCtx) collectDynamicImports() {
	if c == nil || c.source == nil {
		return
	}

	for _, stmt := range c.source.Statements.Nodes {
		if stmt == nil {
			continue
		}
		c.walkDynamicImports(stmt)
	}
}

func (c *tsgoImportExportCtx) walkDynamicImports(node *tsast.Node) {
	if node == nil {
		return
	}
	if node.Kind == tsast.KindCallExpression {
		call := node.AsCallExpression()
		if call != nil &&
			call.Expression != nil &&
			call.Expression.Kind == tsast.KindImportKeyword &&
			call.Arguments != nil &&
			len(call.Arguments.Nodes) > 0 {
			spec := ImportModuleSpecifierFromExpression(call.Arguments.Nodes[0])
			if spec != "" {
				c.appendDynamicImport(node, spec, call.Arguments.Nodes[0], ImportCallIsTypeOnly(node))
			}
		}
	}
	node.ForEachChild(func(child *tsast.Node) bool {
		c.walkDynamicImports(child)
		return false
	})
}

func (c *tsgoImportExportCtx) appendDynamicImport(callNode *tsast.Node, moduleSpecifier string, specifierNode *tsast.Node, isTypeOnly bool) {
	moduleSpecPath := c.resolveModuleSpec(moduleSpecifier)
	line, col := c.lineColumn(callNode.Pos())
	moduleSpecText := moduleSpecifier

	c.dynamicImports = append(c.dynamicImports, &Import{
		ReferenceIdent:  "default",
		ModuleSpecPath:  moduleSpecPath,
		Text:            strings.TrimSpace(c.nodeText(callNode)),
		Start:           callNode.Pos(),
		End:             callNode.End(),
		Line:            line,
		Column:          col,
		ModuleSpecText:  moduleSpecText,
		ModuleSpecStart: specifierNode.Pos(),
		ModuleSpecEnd:   specifierNode.End(),
		IsDynamic:       true,
		IsTypeOnly:      isTypeOnly,
	})
}

// ImportCallIsTypeOnly reports whether import() appears under typeof import(...) or import type.
func ImportCallIsTypeOnly(callNode *tsast.Node) bool {
	for node := callNode; node != nil; node = node.Parent {
		switch node.Kind {
		case tsast.KindTypeQuery, tsast.KindImportType:
			return true
		}
	}
	return false
}

// ImportModuleSpecifierFromExpression returns the module string when arg is a string literal.
func ImportModuleSpecifierFromExpression(arg *tsast.Node) string {
	if arg == nil {
		return ""
	}
	switch arg.Kind {
	case tsast.KindStringLiteral, tsast.KindNoSubstitutionTemplateLiteral:
		return strings.Trim(arg.Text(), "`\"'")
	default:
		return ""
	}
}
