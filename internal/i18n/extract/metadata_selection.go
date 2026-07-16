// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package extract

import (
	"fmt"
	"path/filepath"
	"strings"

	tsast "github.com/buke/typescript-go-internal/pkg/ast"
	tscore "github.com/buke/typescript-go-internal/pkg/core"
	"github.com/choysum-dev/choysum/internal/parser/tsgoctx"
)

// CollectMetadataSelection extracts selection_label terms from @Field({ selection: [...] }).
func CollectMetadataSelection(opts CollectOptions, content string) ([]TermOccurrence, []ExtractIssue) {
	ctx, issues := parseExtractScript(opts, content)
	if ctx == nil {
		return nil, issues
	}
	c := &selectionCollector{
		opts:      opts,
		ctx:       ctx,
		scopePath: ScopePathFromRelPath(opts.RelPath),
	}
	for _, stmt := range ctx.Source.Statements.Nodes {
		c.walk(stmt)
	}
	return c.terms, append(issues, c.issues...)
}

type selectionCollector struct {
	opts      CollectOptions
	ctx       *tsgoctx.ParseCtx
	scopePath string
	terms     []TermOccurrence
	issues    []ExtractIssue
}

func (c *selectionCollector) walk(node *tsast.Node) {
	if node == nil {
		return
	}
	if node.Kind == tsast.KindPropertyDeclaration {
		c.handleProperty(node)
	}
	node.ForEachChild(func(child *tsast.Node) bool {
		c.walk(child)
		return false
	})
}

func (c *selectionCollector) handleProperty(node *tsast.Node) {
	prop := node.AsPropertyDeclaration()
	if prop == nil {
		return
	}
	fieldName := ""
	if prop.Name() != nil {
		fieldName = strings.TrimSpace(c.ctx.NodeText(prop.Name()))
	}
	if fieldName == "" {
		return
	}
	mods := prop.Modifiers()
	if mods == nil {
		return
	}
	for _, mod := range mods.Nodes {
		if mod == nil || mod.Kind != tsast.KindDecorator {
			continue
		}
		dec := mod.AsDecorator()
		if dec == nil || dec.Expression == nil {
			continue
		}
		call := dec.Expression
		if call.Kind != tsast.KindCallExpression {
			continue
		}
		callExpr := call.AsCallExpression()
		if callExpr == nil || !isIdentifierNamed(c.ctx, callExpr.Expression, "Field") {
			continue
		}
		if callExpr.Arguments == nil || len(callExpr.Arguments.Nodes) == 0 {
			continue
		}
		optsNode := callExpr.Arguments.Nodes[0]
		selection := objectProp(c.ctx, optsNode, "selection")
		if selection == nil || selection.Kind != tsast.KindArrayLiteralExpression {
			continue
		}
		arr := selection.AsArrayLiteralExpression()
		if arr == nil {
			continue
		}
		for _, el := range arr.Elements.Nodes {
			c.collectSelectionItem(fieldName, el)
		}
	}
}

func (c *selectionCollector) collectSelectionItem(fieldName string, el *tsast.Node) {
	if el == nil || el.Kind != tsast.KindObjectLiteralExpression {
		return
	}
	value, valueOK := stringLiteralValue(c.ctx, objectProp(c.ctx, el, "value"))
	labelNode := objectProp(c.ctx, el, "label")
	line, col := 1, 1
	if labelNode != nil {
		line, col = c.ctx.LineColumn(labelNode.Pos())
	}
	label, labelOK := stringLiteralValue(c.ctx, labelNode)
	if !labelOK {
		if labelNode != nil {
			c.issues = append(c.issues, ExtractIssue{
				Severity:   IssueSeverityWarn,
				Code:       IssueNonLiteralMsgid,
				Message:    fmt.Sprintf("non-literal selection label on %s; skipped", fieldName),
				SourcePath: c.opts.RelPath,
				Line:       line,
				Col:        col,
			})
		}
		return
	}
	if !valueOK || value == "" {
		value = "unknown"
	}
	location := fieldName + "." + value
	c.terms = append(c.terms, TermOccurrence{
		Module:     c.opts.ModuleName,
		Scope:      FormatScope(c.scopePath, location),
		Src:        label,
		Kind:       KindSelectionLabel,
		SourcePath: c.opts.RelPath,
		Line:       line,
		Col:        col,
	})
}

func objectProp(ctx *tsgoctx.ParseCtx, node *tsast.Node, name string) *tsast.Node {
	if node == nil || node.Kind != tsast.KindObjectLiteralExpression {
		return nil
	}
	obj := node.AsObjectLiteralExpression()
	if obj == nil {
		return nil
	}
	for _, prop := range obj.Properties.Nodes {
		if prop == nil || prop.Kind != tsast.KindPropertyAssignment {
			continue
		}
		asgn := prop.AsPropertyAssignment()
		if asgn == nil || asgn.Name() == nil {
			continue
		}
		key := strings.Trim(strings.TrimSpace(ctx.NodeText(asgn.Name())), `"'`)
		if key == name {
			return asgn.Initializer
		}
	}
	return nil
}

func parseExtractScript(opts CollectOptions, content string) (*tsgoctx.ParseCtx, []ExtractIssue) {
	fakePath := opts.RelPath
	if fakePath == "" {
		fakePath = "anonymous.ts"
	}
	if !filepath.IsAbs(fakePath) {
		fakePath = filepath.Join("/extract", filepath.FromSlash(fakePath))
	}
	forceTS := strings.EqualFold(filepath.Ext(fakePath), ".vue")
	var ctx *tsgoctx.ParseCtx
	var err error
	if forceTS {
		ctx, err = tsgoctx.ParseWithKind(opts.PathAlias, fakePath, content, tscore.ScriptKindTS, true)
	} else {
		ctx, err = tsgoctx.Parse(opts.PathAlias, fakePath, content)
	}
	if err != nil {
		return nil, []ExtractIssue{{
			Severity:   IssueSeverityWarn,
			Code:       "parse_error",
			Message:    err.Error(),
			SourcePath: opts.RelPath,
			Line:       1,
			Col:        1,
		}}
	}
	return ctx, nil
}
