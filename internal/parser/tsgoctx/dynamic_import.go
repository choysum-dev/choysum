// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package tsgoctx

import (
	"strings"

	tsast "github.com/buke/typescript-go-internal/pkg/ast"
	"github.com/choysum-dev/choysum/internal/parser"
)

func (c *ParseCtx) collectDynamicImports() {
	if c == nil || c.Source == nil {
		return
	}

	var walk func(node *tsast.Node)
	walk = func(node *tsast.Node) {
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
				spec := parser.ImportModuleSpecifierFromExpression(call.Arguments.Nodes[0])
				if spec != "" {
					c.appendDynamicImport(node, spec, call.Arguments.Nodes[0], parser.ImportCallIsTypeOnly(node))
				}
			}
		}
		node.ForEachChild(func(child *tsast.Node) bool {
			walk(child)
			return false
		})
	}

	for _, stmt := range c.Source.Statements.Nodes {
		if stmt == nil {
			continue
		}
		walk(stmt)
	}
}

func (c *ParseCtx) appendDynamicImport(callNode *tsast.Node, moduleSpecifier string, specifierNode *tsast.Node, isTypeOnly bool) {
	moduleSpecPath := c.ResolveModuleSpec(moduleSpecifier)
	line, col := c.LineColumn(callNode.Pos())
	moduleSpecText := moduleSpecifier

	c.DynamicImports = append(c.DynamicImports, &parser.Import{
		ReferenceIdent:  "default",
		ModuleSpecPath:  moduleSpecPath,
		Text:            strings.TrimSpace(c.NodeText(callNode)),
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
