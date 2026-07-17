// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package extract

import (
	"fmt"
	"path/filepath"
	"strings"

	tsast "github.com/buke/typescript-go-internal/pkg/ast"
	tscore "github.com/buke/typescript-go-internal/pkg/core"
	"github.com/choysum-dev/choysum/internal/parser"
	"github.com/choysum-dev/choysum/internal/parser/tsgoctx"
)

// CollectOptions configures AST collection for one source file.
type CollectOptions struct {
	// ModuleName is the owning module (CLI / CollectModule sets this).
	ModuleName string
	// RelPath is module-relative path with extension (e.g. web/pages/Login.ts).
	RelPath string
	// PathAlias is optional tsconfig paths map for import resolution.
	PathAlias map[string]string
}

// CollectScript extracts `_t` / `_lt` / `_tr` literal terms from TS/TSX (or Vue script) source.
func CollectScript(opts CollectOptions, content string) ([]TermOccurrence, []ExtractIssue) {
	terms, issues, _ := collectScript(opts, content)
	return terms, issues
}

func collectScript(opts CollectOptions, content string) ([]TermOccurrence, []ExtractIssue, map[string]string) {
	fakePath := opts.RelPath
	if fakePath == "" {
		fakePath = "anonymous.ts"
	}
	if !filepath.IsAbs(fakePath) {
		fakePath = filepath.Join("/extract", filepath.FromSlash(fakePath))
	}

	// Vue script content is TypeScript even when RelPath ends in .vue.
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
		}}, nil
	}

	c := &scriptCollector{
		opts:                   opts,
		ctx:                    ctx,
		scopePath:              ScopePathFromRelPath(opts.RelPath),
		translateIdents:        map[string]bool{},
		translateDefaultScopes: map[string]string{},
	}
	for _, stmt := range ctx.Source.Statements.Nodes {
		c.walk(stmt, "")
	}
	return c.terms, c.issues, c.translateDefaultScopes
}

type scriptCollector struct {
	opts                   CollectOptions
	ctx                    *tsgoctx.ParseCtx
	scopePath              string
	scopeStack             []string
	enclosing              []string
	translateIdents        map[string]bool
	translateDefaultScopes map[string]string
	terms                  []TermOccurrence
	issues                 []ExtractIssue
}

func (c *scriptCollector) walk(node *tsast.Node, pendingLocation string) {
	if node == nil {
		return
	}

	switch node.Kind {
	case tsast.KindCallExpression:
		c.handleCall(node, pendingLocation)
		return
	case tsast.KindVariableDeclaration:
		c.handleVariableDeclaration(node)
		return
	case tsast.KindFunctionDeclaration:
		c.handleNamedFunction(node)
		return
	case tsast.KindMethodDeclaration, tsast.KindMethodSignature:
		c.handleNamedMethod(node)
		return
	case tsast.KindFunctionExpression, tsast.KindArrowFunction:
		c.walkChildren(node, pendingLocation)
		return
	}

	c.walkChildren(node, pendingLocation)
}

func (c *scriptCollector) walkChildren(node *tsast.Node, pendingLocation string) {
	node.ForEachChild(func(child *tsast.Node) bool {
		c.walk(child, pendingLocation)
		return false
	})
}

func (c *scriptCollector) handleNamedFunction(node *tsast.Node) {
	fn := node.AsFunctionDeclaration()
	name := ""
	if fn != nil && fn.Name() != nil {
		name = strings.TrimSpace(c.ctx.NodeText(fn.Name()))
	}
	if name != "" {
		c.enclosing = append(c.enclosing, name)
		defer func() { c.enclosing = c.enclosing[:len(c.enclosing)-1] }()
	}
	c.walkChildren(node, "")
}

func (c *scriptCollector) handleNamedMethod(node *tsast.Node) {
	name := ""
	if node.Name() != nil {
		name = strings.TrimSpace(c.ctx.NodeText(node.Name()))
	}
	if name != "" {
		c.enclosing = append(c.enclosing, name)
		defer func() { c.enclosing = c.enclosing[:len(c.enclosing)-1] }()
	}
	c.walkChildren(node, "")
}

func (c *scriptCollector) handleVariableDeclaration(node *tsast.Node) {
	nameNode := node.Name()
	init := node.Initializer()

	if init != nil && init.Kind == tsast.KindCallExpression {
		if c.tryRegisterCreateTranslate(init, nameNode) {
			c.walkChildren(init, "")
			return
		}
	}

	bindingName := bindingIdentName(c.ctx, nameNode)
	if init != nil && init.Kind == tsast.KindCallExpression && bindingName != "" {
		c.handleCall(init, bindingName)
		return
	}

	if bindingName != "" && init != nil && (init.Kind == tsast.KindArrowFunction || init.Kind == tsast.KindFunctionExpression) {
		c.enclosing = append(c.enclosing, bindingName)
		defer func() { c.enclosing = c.enclosing[:len(c.enclosing)-1] }()
		c.walk(init, "")
		return
	}

	c.walkChildren(node, "")
}

func (c *scriptCollector) tryRegisterCreateTranslate(call *tsast.Node, nameNode *tsast.Node) bool {
	callExpr := call.AsCallExpression()
	if callExpr == nil {
		return false
	}
	factoryName := strings.TrimSpace(c.ctx.NodeText(callExpr.Expression))
	if factoryName != "createTranslate" {
		return false
	}

	moduleArg := ""
	defaultScope := ""
	args := callExpr.Arguments
	if args != nil && len(args.Nodes) > 0 {
		if lit, ok := stringLiteralValue(c.ctx, args.Nodes[0]); ok {
			moduleArg = lit
		}
		if len(args.Nodes) > 1 {
			defaultScope, _ = parseTranslateOptions(c.ctx, args.Nodes[1])
		}
	}
	if moduleArg != "" && c.opts.ModuleName != "" && moduleArg != c.opts.ModuleName {
		line, col := c.ctx.LineColumn(call.Pos())
		c.issues = append(c.issues, ExtractIssue{
			Severity:   IssueSeverityWarn,
			Code:       IssueModuleMismatch,
			Message:    fmt.Sprintf("%s(%q) does not match module %q", factoryName, moduleArg, c.opts.ModuleName),
			SourcePath: c.opts.RelPath,
			Line:       line,
			Col:        col,
		})
	}

	registerTranslateBindings(c.translateIdents, c.translateDefaultScopes, defaultScope, c.ctx, nameNode)
	return true
}

func registerTranslateBindings(
	idents map[string]bool,
	defaultScopes map[string]string,
	defaultScope string,
	ctx *tsgoctx.ParseCtx,
	nameNode *tsast.Node,
) {
	if nameNode == nil {
		return
	}
	switch nameNode.Kind {
	case tsast.KindIdentifier:
		// const t = createTranslate('m') — not enough without property access.
		return
	case tsast.KindObjectBindingPattern:
		pattern := nameNode.AsBindingPattern()
		if pattern == nil || pattern.Elements.Nodes == nil {
			return
		}
		for _, el := range pattern.Elements.Nodes {
			if el == nil || el.Kind != tsast.KindBindingElement {
				continue
			}
			propName := ""
			if pn := el.PropertyName(); pn != nil {
				propName = strings.TrimSpace(ctx.NodeText(pn))
			} else if el.Name() != nil {
				propName = strings.TrimSpace(ctx.NodeText(el.Name()))
			}
			localName := propName
			if el.Name() != nil && el.Name().Kind == tsast.KindIdentifier {
				localName = strings.TrimSpace(ctx.NodeText(el.Name()))
			}
			if isTranslateIdentifier(propName) || isTranslateIdentifier(localName) {
				if localName != "" {
					idents[localName] = true
					if defaultScope != "" {
						defaultScopes[localName] = defaultScope
					}
				}
			}
		}
	}
}

func (c *scriptCollector) handleCall(node *tsast.Node, pendingLocation string) {
	callExpr := node.AsCallExpression()
	if callExpr == nil {
		c.walkChildren(node, pendingLocation)
		return
	}

	if isIdentifierNamed(c.ctx, callExpr.Expression, "withI18nScope") {
		c.handleWithI18nScope(node, callExpr)
		return
	}

	if c.isTranslateCallee(callExpr.Expression) {
		c.collectTranslateCall(node, callExpr, pendingLocation)
		// Still walk nested args for nested calls (rare).
		if callExpr.Arguments != nil {
			for _, arg := range callExpr.Arguments.Nodes {
				c.walk(arg, "")
			}
		}
		return
	}

	c.walkChildren(node, pendingLocation)
}

func (c *scriptCollector) handleWithI18nScope(node *tsast.Node, callExpr *tsast.CallExpression) {
	manual := ""
	var body *tsast.Node
	if callExpr.Arguments != nil {
		if len(callExpr.Arguments.Nodes) > 0 {
			if lit, ok := stringLiteralValue(c.ctx, callExpr.Arguments.Nodes[0]); ok {
				manual = lit
			}
		}
		if len(callExpr.Arguments.Nodes) > 1 {
			body = callExpr.Arguments.Nodes[1]
		}
	}
	if manual != "" {
		c.scopeStack = append(c.scopeStack, manual)
		defer func() { c.scopeStack = c.scopeStack[:len(c.scopeStack)-1] }()
	}
	if body != nil {
		c.walk(body, "")
	}
	// Walk expression / other children except already handled args.
	if callExpr.Expression != nil {
		c.walk(callExpr.Expression, "")
	}
}

func (c *scriptCollector) isTranslateCallee(expr *tsast.Node) bool {
	if expr == nil {
		return false
	}
	if expr.Kind == tsast.KindIdentifier {
		name := strings.TrimSpace(c.ctx.NodeText(expr))
		if isTranslateIdentifier(name) {
			return true
		}
		return c.translateIdents[name]
	}
	return false
}

func isTranslateIdentifier(name string) bool {
	return name == "_t" || name == "_lt" || name == "_tr"
}

func (c *scriptCollector) collectTranslateCall(node *tsast.Node, callExpr *tsast.CallExpression, pendingLocation string) {
	line, col := c.ctx.LineColumn(node.Pos())
	args := callExpr.Arguments
	if args == nil || len(args.Nodes) == 0 {
		c.issues = append(c.issues, ExtractIssue{
			Severity:   IssueSeverityWarn,
			Code:       IssueNonLiteralMsgid,
			Message:    "translate call missing msgid argument",
			SourcePath: c.opts.RelPath,
			Line:       line,
			Col:        col,
		})
		return
	}

	src, ok := stringLiteralValue(c.ctx, args.Nodes[0])
	if !ok {
		c.issues = append(c.issues, ExtractIssue{
			Severity:   IssueSeverityWarn,
			Code:       IssueNonLiteralMsgid,
			Message:    "non-literal msgid; skipped",
			SourcePath: c.opts.RelPath,
			Line:       line,
			Col:        col,
		})
		return
	}

	manualScope := ""
	kind := KindLiteral
	if len(args.Nodes) > 1 {
		manualScope, kind = parseTranslateOptions(c.ctx, args.Nodes[1])
	}
	if manualScope == "" && callExpr.Expression != nil && callExpr.Expression.Kind == tsast.KindIdentifier {
		calleeName := strings.TrimSpace(c.ctx.NodeText(callExpr.Expression))
		manualScope = c.translateDefaultScopes[calleeName]
	}

	location := pendingLocation
	if location == "" && len(c.enclosing) > 0 {
		location = c.enclosing[len(c.enclosing)-1]
	}

	scope := ResolveI18nScope(manualScope, c.scopeStack, c.scopePath, location)
	c.terms = append(c.terms, TermOccurrence{
		Module:     c.opts.ModuleName,
		Scope:      scope,
		Src:        src,
		Kind:       kind,
		SourcePath: c.opts.RelPath,
		Line:       line,
		Col:        col,
	})
}

func parseTranslateOptions(ctx *tsgoctx.ParseCtx, node *tsast.Node) (scope string, kind string) {
	kind = KindLiteral
	if node == nil || node.Kind != tsast.KindObjectLiteralExpression {
		return "", kind
	}
	obj := node.AsObjectLiteralExpression()
	if obj == nil || obj.Properties.Nodes == nil {
		return "", kind
	}
	for _, prop := range obj.Properties.Nodes {
		if prop == nil || prop.Kind != tsast.KindPropertyAssignment {
			continue
		}
		pa := prop.AsPropertyAssignment()
		if pa == nil || pa.Name() == nil {
			continue
		}
		key := strings.TrimSpace(ctx.NodeText(pa.Name()))
		val, ok := stringLiteralValue(ctx, prop.Initializer())
		if !ok {
			continue
		}
		switch key {
		case "scope":
			scope = val
		case "kind":
			kind = val
		}
	}
	return scope, kind
}

func stringLiteralValue(ctx *tsgoctx.ParseCtx, node *tsast.Node) (string, bool) {
	if node == nil {
		return "", false
	}
	if node.Kind != tsast.KindStringLiteral && node.Kind != tsast.KindNoSubstitutionTemplateLiteral {
		return "", false
	}
	text := strings.TrimSpace(ctx.NodeText(node))
	if node.Kind == tsast.KindNoSubstitutionTemplateLiteral {
		if len(text) >= 2 && text[0] == '`' && text[len(text)-1] == '`' {
			return text[1 : len(text)-1], true
		}
		return "", false
	}
	v, err := parser.ParseJSStringLiteral(text)
	if err != nil {
		return "", false
	}
	return v, true
}

func isIdentifierNamed(ctx *tsgoctx.ParseCtx, node *tsast.Node, name string) bool {
	if node == nil || node.Kind != tsast.KindIdentifier {
		return false
	}
	return strings.TrimSpace(ctx.NodeText(node)) == name
}

func bindingIdentName(ctx *tsgoctx.ParseCtx, nameNode *tsast.Node) string {
	if nameNode == nil || nameNode.Kind != tsast.KindIdentifier {
		return ""
	}
	return strings.TrimSpace(ctx.NodeText(nameNode))
}
