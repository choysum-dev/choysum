// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package vueparser

import (
	"strings"

	tsast "github.com/buke/typescript-go-internal/v7/pkg/ast"
	tscore "github.com/buke/typescript-go-internal/v7/pkg/core"
	"github.com/choysum-dev/choysum/internal/parser"
)

type vueComponentTSGoResult struct {
	extendsProperty    *parser.PropertyNode
	componentPropertys []*parser.PropertyNode
	rawExtends         string
}

func parseVueComponentWithTSGo(pathAlias map[string]string, path string, scriptContent string) (*vueComponentTSGoResult, error) {
	if strings.TrimSpace(scriptContent) == "" {
		return &vueComponentTSGoResult{}, nil
	}

	ctx, err := parseTSGoCtxWithKind(pathAlias, path, scriptContent, tscore.ScriptKindTS, true)
	if err != nil {
		return nil, err
	}

	for _, stmt := range ctx.Source.Statements.Nodes {
		if stmt == nil || stmt.Kind != tsast.KindExportAssignment {
			continue
		}

		expr := stmt.AsExportAssignment().Expression
		if expr == nil || expr.Kind != tsast.KindCallExpression {
			continue
		}
		callExpr := expr.AsCallExpression()
		if callExpr.Expression == nil || !isDefineComponentCall(ctx, callExpr.Expression) {
			continue
		}
		if callExpr.Arguments == nil || len(callExpr.Arguments.Nodes) == 0 {
			continue
		}

		options := callExpr.Arguments.Nodes[0]
		if options == nil || options.Kind != tsast.KindObjectLiteralExpression {
			continue
		}

		result := &vueComponentTSGoResult{
			componentPropertys: make([]*parser.PropertyNode, 0),
		}

		for _, propertyNode := range options.AsObjectLiteralExpression().Properties.Nodes {
			if propertyNode == nil || propertyNode.Kind != tsast.KindPropertyAssignment {
				continue
			}

			propertyAssign := propertyNode.AsPropertyAssignment()
			propertyName := tsgoPropertyName(ctx, propertyAssign.Name())
			switch propertyName {
			case "extends":
				extendsProperty, rawExtends := parseVueExtendsProperty(ctx, propertyNode, propertyAssign.Initializer)
				if extendsProperty != nil {
					result.extendsProperty = extendsProperty
					result.rawExtends = rawExtends
				}
			case "components":
				result.componentPropertys = parseVueComponentsProperties(ctx, propertyAssign.Initializer)
			}
		}

		return result, nil
	}

	return &vueComponentTSGoResult{}, nil
}

func isDefineComponentCall(ctx *tsParseCtx, expr *tsast.Node) bool {
	return isDefineComponentCallWithOptions(ctx, expr, false)
}

func isDefineComponentCallWithOptions(ctx *tsParseCtx, expr *tsast.Node, allowNamespace bool) bool {
	if ctx == nil || expr == nil {
		return false
	}
	if expr.Kind == tsast.KindIdentifier {
		imp, ok := ctx.Imports[expr.Text()]
		if !ok {
			return false
		}
		// Use the original (unresolved) module specifier to avoid false
		// negatives when tsconfig paths remap "vue" to a .d.ts file.
		originalSpec := strings.Trim(imp.ModuleSpecText, `"'`)
		return originalSpec == "vue" && imp.ReferenceIdent == "defineComponent"
	}
	if allowNamespace && expr.Kind == tsast.KindPropertyAccessExpression {
		propertyAccess := expr.AsPropertyAccessExpression()
		if propertyAccess == nil || propertyAccess.Expression == nil || propertyAccess.Name() == nil {
			return false
		}
		if strings.TrimSpace(propertyAccess.Name().Text()) != "defineComponent" {
			return false
		}
		if propertyAccess.Expression.Kind != tsast.KindIdentifier {
			return false
		}
		imp, ok := ctx.Imports[propertyAccess.Expression.Text()]
		if !ok {
			return false
		}
		originalSpec := strings.Trim(imp.ModuleSpecText, `"'`)
		return originalSpec == "vue"
	}
	return false
}

func parseVueExtendsProperty(ctx *tsParseCtx, propertyNode *tsast.Node, valueNode *tsast.Node) (*parser.PropertyNode, string) {
	if ctx == nil || propertyNode == nil || valueNode == nil {
		return nil, ""
	}

	referenceIdent := ""
	moduleSpecPath := ""
	switch valueNode.Kind {
	case tsast.KindIdentifier:
		moduleSpecPath, referenceIdent = ctx.ConvertReferenceWithModuleSpec(valueNode.Text())
	case tsast.KindPropertyAccessExpression:
		pa := valueNode.AsPropertyAccessExpression()
		moduleSpecPath, _ = ctx.ConvertReferenceWithModuleSpec(pa.Expression.Text())
		referenceIdent = pa.Name().Text()
	default:
		return nil, ""
	}

	if referenceIdent != "default" {
		return nil, ""
	}

	return buildTSGoPropertyNode(ctx, propertyNode, referenceIdent, moduleSpecPath), moduleSpecPath
}

func parseVueComponentsProperties(ctx *tsParseCtx, valueNode *tsast.Node) []*parser.PropertyNode {
	components := make([]*parser.PropertyNode, 0)
	if ctx == nil || valueNode == nil || valueNode.Kind != tsast.KindObjectLiteralExpression {
		return components
	}

	for _, propertyNode := range valueNode.AsObjectLiteralExpression().Properties.Nodes {
		if propertyNode == nil {
			continue
		}

		var referenceText string
		switch propertyNode.Kind {
		case tsast.KindShorthandPropertyAssignment:
			referenceText = strings.TrimSpace(ctx.NodeText(propertyNode.AsShorthandPropertyAssignment().Name()))
			if referenceText == "" {
				referenceText = propertyNode.AsShorthandPropertyAssignment().Name().Text()
			}
		case tsast.KindPropertyAssignment:
			initializer := propertyNode.AsPropertyAssignment().Initializer
			if initializer == nil || initializer.Kind != tsast.KindIdentifier {
				continue
			}
			referenceText = initializer.Text()
		default:
			continue
		}

		if referenceText == "" {
			continue
		}
		moduleSpecPath, referenceIdent := ctx.ConvertReferenceWithModuleSpec(referenceText)
		components = append(components, buildTSGoPropertyNode(ctx, propertyNode, referenceIdent, moduleSpecPath))
	}

	return components
}

func buildTSGoPropertyNode(ctx *tsParseCtx, node *tsast.Node, referenceIdent string, moduleSpecPath string) *parser.PropertyNode {
	if ctx == nil || node == nil {
		return nil
	}

	name, valueKind, valueText, valueStart, valueEnd := tsgoExtractPropertyStableInfo(ctx, node)
	line, col := ctx.LineColumn(node.Pos())
	return &parser.PropertyNode{
		ReferenceIdent: referenceIdent,
		ModuleSpecPath: moduleSpecPath,
		Text:           ctx.NodeText(node),
		Start:          node.Pos(),
		End:            node.End(),
		Line:           line,
		Column:         col,
		Name:           name,
		ValueKind:      valueKind,
		ValueText:      valueText,
		ValueStart:     valueStart,
		ValueEnd:       valueEnd,
	}
}

func tsgoExtractPropertyStableInfo(ctx *tsParseCtx, propertyNode *tsast.Node) (name, valueKind, valueText string, valueStart, valueEnd int) {
	if ctx == nil || propertyNode == nil {
		return "", "", "", 0, 0
	}

	switch propertyNode.Kind {
	case tsast.KindPropertyAssignment:
		propertyAssign := propertyNode.AsPropertyAssignment()
		name = tsgoPropertyName(ctx, propertyAssign.Name())
		if propertyAssign.Initializer != nil {
			valueKind = propertyAssign.Initializer.Kind.String()
			valueText = ctx.NodeText(propertyAssign.Initializer)
			valueStart = propertyAssign.Initializer.Pos()
			valueEnd = propertyAssign.Initializer.End()
		}
	case tsast.KindShorthandPropertyAssignment:
		name = ""
		valueKind = tsast.KindIdentifier.String()
		valueText = tsgoPropertyName(ctx, propertyNode.AsShorthandPropertyAssignment().Name())
		valueStart = propertyNode.Pos()
		valueEnd = propertyNode.End()
	}

	return
}

func tsgoPropertyName(ctx *tsParseCtx, nameNode *tsast.Node) string {
	if ctx == nil || nameNode == nil {
		return ""
	}
	name := strings.TrimSpace(ctx.NodeText(nameNode))
	if name == "" {
		name = strings.TrimSpace(nameNode.Text())
	}
	if value, err := parseJSStringLiteral(name); err == nil {
		name = value
	}
	return strings.TrimSpace(name)
}
