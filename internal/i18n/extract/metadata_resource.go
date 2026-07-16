// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package extract

import (
	"fmt"
	"strings"

	tsast "github.com/buke/typescript-go-internal/pkg/ast"
	"github.com/choysum-dev/choysum/internal/parser/tsgoctx"
)

var resourceDefineFns = map[string]string{
	"defineMenu":   KindMenu,
	"defineRoute":  KindRoute,
	"defineAction": KindAction,
}

// CollectMetadataResources extracts menu/route/action title terms from define* calls.
func CollectMetadataResources(opts CollectOptions, content string) ([]TermOccurrence, []ExtractIssue) {
	ctx, issues := parseExtractScript(opts, content)
	if ctx == nil {
		return nil, issues
	}
	c := &resourceCollector{
		opts:      opts,
		ctx:       ctx,
		scopePath: ScopePathFromRelPath(opts.RelPath),
	}
	for _, stmt := range ctx.Source.Statements.Nodes {
		c.walk(stmt)
	}
	return c.terms, append(issues, c.issues...)
}

type resourceCollector struct {
	opts      CollectOptions
	ctx       *tsgoctx.ParseCtx
	scopePath string
	terms     []TermOccurrence
	issues    []ExtractIssue
}

func (c *resourceCollector) walk(node *tsast.Node) {
	if node == nil {
		return
	}
	if node.Kind == tsast.KindCallExpression {
		c.handleCall(node)
	}
	node.ForEachChild(func(child *tsast.Node) bool {
		c.walk(child)
		return false
	})
}

func (c *resourceCollector) handleCall(node *tsast.Node) {
	callExpr := node.AsCallExpression()
	if callExpr == nil || callExpr.Expression == nil {
		return
	}
	fnName := strings.TrimSpace(c.ctx.NodeText(callExpr.Expression))
	if kind, ok := resourceDefineFns[fnName]; ok {
		c.collectDefineTitle(callExpr, kind)
		return
	}
	if fnName == "defineModelActions" {
		c.collectModelActionTitles(callExpr)
	}
}

func (c *resourceCollector) collectDefineTitle(callExpr *tsast.CallExpression, kind string) {
	if callExpr.Arguments == nil || len(callExpr.Arguments.Nodes) < 2 {
		return
	}
	id, idOK := stringLiteralValue(c.ctx, callExpr.Arguments.Nodes[0])
	if !idOK || id == "" {
		return
	}
	optsNode := callExpr.Arguments.Nodes[1]
	titleNode := objectProp(c.ctx, optsNode, "title")
	if titleNode == nil {
		return
	}
	line, col := c.ctx.LineColumn(titleNode.Pos())
	title, ok := stringLiteralValue(c.ctx, titleNode)
	if !ok {
		c.issues = append(c.issues, ExtractIssue{
			Severity:   IssueSeverityWarn,
			Code:       IssueNonLiteralMsgid,
			Message:    fmt.Sprintf("non-literal %s title for %s; skipped", kind, id),
			SourcePath: c.opts.RelPath,
			Line:       line,
			Col:        col,
		})
		return
	}
	c.terms = append(c.terms, TermOccurrence{
		Module:     c.opts.ModuleName,
		Scope:      FormatScope(c.scopePath, id),
		Src:        title,
		Kind:       kind,
		SourcePath: c.opts.RelPath,
		Line:       line,
		Col:        col,
	})
}

func (c *resourceCollector) collectModelActionTitles(callExpr *tsast.CallExpression) {
	if callExpr.Arguments == nil || len(callExpr.Arguments.Nodes) < 2 {
		return
	}
	optsNode := callExpr.Arguments.Nodes[1]
	titlesNode := objectProp(c.ctx, optsNode, "titles")
	if titlesNode == nil || titlesNode.Kind != tsast.KindObjectLiteralExpression {
		return
	}
	obj := titlesNode.AsObjectLiteralExpression()
	if obj == nil {
		return
	}
	for _, prop := range obj.Properties.Nodes {
		if prop == nil || prop.Kind != tsast.KindPropertyAssignment {
			continue
		}
		asgn := prop.AsPropertyAssignment()
		if asgn == nil || asgn.Name() == nil || asgn.Initializer == nil {
			continue
		}
		actionKey := strings.Trim(strings.TrimSpace(c.ctx.NodeText(asgn.Name())), `"'`)
		line, col := c.ctx.LineColumn(asgn.Initializer.Pos())
		title, ok := stringLiteralValue(c.ctx, asgn.Initializer)
		if !ok {
			c.issues = append(c.issues, ExtractIssue{
				Severity:   IssueSeverityWarn,
				Code:       IssueNonLiteralMsgid,
				Message:    fmt.Sprintf("non-literal model action title %s; skipped", actionKey),
				SourcePath: c.opts.RelPath,
				Line:       line,
				Col:        col,
			})
			continue
		}
		location := actionKey
		if model, mok := stringLiteralValue(c.ctx, callExpr.Arguments.Nodes[0]); mok && model != "" {
			location = model + "." + actionKey
		}
		c.terms = append(c.terms, TermOccurrence{
			Module:     c.opts.ModuleName,
			Scope:      FormatScope(c.scopePath, location),
			Src:        title,
			Kind:       KindAction,
			SourcePath: c.opts.RelPath,
			Line:       line,
			Col:        col,
		})
	}
}
