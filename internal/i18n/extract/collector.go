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

// CollectScript extracts `_t` literal terms from TS/TSX (or Vue script) source.
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
		referenceIdents:        map[string]bool{},
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
	referenceIdents        map[string]bool
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
	referenceOutput := false
	args := callExpr.Arguments
	if args != nil && len(args.Nodes) > 0 {
		if lit, ok := stringLiteralValue(c.ctx, args.Nodes[0]); ok {
			moduleArg = lit
		}
		if len(args.Nodes) > 1 {
			defaultScope, referenceOutput = parseCreateTranslateOptions(c.ctx, args.Nodes[1])
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

	registerTranslateBindings(c.translateIdents, c.referenceIdents, c.translateDefaultScopes, defaultScope, referenceOutput, c.ctx, nameNode)
	return true
}

func registerTranslateBindings(
	idents map[string]bool,
	referenceIdents map[string]bool,
	defaultScopes map[string]string,
	defaultScope string,
	referenceOutput bool,
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
		if pattern == nil || pattern.Elements == nil || pattern.Elements.Nodes == nil {
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
			if propName == "_t" {
				if localName != "" {
					idents[localName] = true
					if referenceOutput {
						referenceIdents[localName] = true
					}
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

	if isIdentifierNamed(c.ctx, callExpr.Expression, "defineModelActions") {
		c.collectModelActionsTerms(node, callExpr)
		c.walkChildren(node, pendingLocation)
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
	return name == "_t"
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
	calleeName := ""
	if callExpr.Expression != nil && callExpr.Expression.Kind == tsast.KindIdentifier {
		calleeName = strings.TrimSpace(c.ctx.NodeText(callExpr.Expression))
	}
	isReference := c.referenceIdents[calleeName]
	if len(args.Nodes) > 1 {
		if callOutput, ok := parseCallOutput(c.ctx, args.Nodes[1]); ok {
			isReference = callOutput == "reference"
		}
	}
	if isReference {
		if len(args.Nodes) > 1 {
			var scopeOK bool
			manualScope, scopeOK = parseReferenceScope(c.ctx, args.Nodes[1])
			if !scopeOK || strings.TrimSpace(manualScope) == "" {
				c.issues = append(c.issues, ExtractIssue{
					Severity:   IssueSeverityWarn,
					Code:       IssueNonLiteralScope,
					Message:    "reference-output _t scope must be a non-empty string literal; skipped",
					SourcePath: c.opts.RelPath,
					Line:       line,
					Col:        col,
				})
				return
			}
		} else if strings.TrimSpace(c.translateDefaultScopes[calleeName]) == "" {
			c.issues = append(c.issues, ExtractIssue{
				Severity:   IssueSeverityWarn,
				Code:       IssueNonLiteralScope,
				Message:    "reference-output _t requires an explicit or factory-default literal scope; skipped",
				SourcePath: c.opts.RelPath,
				Line:       line,
				Col:        col,
			})
			return
		}
	} else if len(args.Nodes) > 1 {
		manualScope, kind = parseTranslateOptions(c.ctx, args.Nodes[1])
	}
	if manualScope == "" && calleeName != "" {
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

// collectModelActionsTerms emits synthesized Create/Edit/Delete/Copy titles when
// defineModelActions uses entityTitle: _t('Entity') (D19). TitleText at build time
// uses those synthesized msgids under the same scope as the entity _t call.
func (c *scriptCollector) collectModelActionsTerms(node *tsast.Node, callExpr *tsast.CallExpression) {
	if callExpr.Arguments == nil || len(callExpr.Arguments.Nodes) < 2 {
		return
	}
	options := callExpr.Arguments.Nodes[1]
	if options == nil || options.Kind != tsast.KindObjectLiteralExpression {
		return
	}
	obj := options.AsObjectLiteralExpression()
	if obj == nil || obj.Properties == nil || obj.Properties.Nodes == nil {
		return
	}

	line, col := c.ctx.LineColumn(node.Pos())
	var entitySrc string
	var entityScope string
	titleOverrides := map[string]struct {
		src   string
		scope string
	}{}
	excludes := map[string]bool{}

	for _, prop := range obj.Properties.Nodes {
		if prop == nil || prop.Kind != tsast.KindPropertyAssignment {
			continue
		}
		pa := prop.AsPropertyAssignment()
		if pa == nil || pa.Name() == nil {
			continue
		}
		key := strings.TrimSpace(c.ctx.NodeText(pa.Name()))
		init := prop.Initializer()
		switch key {
		case "entityTitle":
			src, scope, ok := c.translateCallLiteral(init)
			if ok {
				entitySrc = src
				entityScope = scope
			}
		case "titles":
			if init == nil || init.Kind != tsast.KindObjectLiteralExpression {
				continue
			}
			titlesObj := init.AsObjectLiteralExpression()
			if titlesObj == nil || titlesObj.Properties == nil || titlesObj.Properties.Nodes == nil {
				continue
			}
			for _, titleProp := range titlesObj.Properties.Nodes {
				if titleProp == nil || titleProp.Kind != tsast.KindPropertyAssignment {
					continue
				}
				tpa := titleProp.AsPropertyAssignment()
				if tpa == nil || tpa.Name() == nil {
					continue
				}
				op := strings.TrimSpace(c.ctx.NodeText(tpa.Name()))
				src, scope, ok := c.translateCallLiteral(titleProp.Initializer())
				if !ok {
					continue
				}
				titleOverrides[op] = struct {
					src   string
					scope string
				}{src: src, scope: scope}
			}
		case "exclude":
			if init == nil || init.Kind != tsast.KindArrayLiteralExpression {
				continue
			}
			arr := init.AsArrayLiteralExpression()
			if arr == nil || arr.Elements == nil || arr.Elements.Nodes == nil {
				continue
			}
			for _, el := range arr.Elements.Nodes {
				if lit, ok := stringLiteralValue(c.ctx, el); ok {
					excludes[strings.TrimSpace(lit)] = true
				}
			}
		}
	}

	prefixes := []struct {
		op     string
		prefix string
	}{
		{op: "create", prefix: "Create "},
		{op: "edit", prefix: "Edit "},
		{op: "delete", prefix: "Delete "},
		{op: "copy", prefix: "Copy "},
	}
	for _, item := range prefixes {
		if excludes[item.op] {
			continue
		}
		src := ""
		scope := ""
		if override, ok := titleOverrides[item.op]; ok {
			src = override.src
			scope = override.scope
		} else if entitySrc != "" {
			src = item.prefix + entitySrc
			scope = entityScope
		}
		if src == "" {
			continue
		}
		if scope == "" {
			scope = ResolveI18nScope("", c.scopeStack, c.scopePath, "")
		}
		c.terms = append(c.terms, TermOccurrence{
			Module:     c.opts.ModuleName,
			Scope:      scope,
			Src:        src,
			Kind:       KindLiteral,
			SourcePath: c.opts.RelPath,
			Line:       line,
			Col:        col,
		})
	}
}

// translateCallLiteral returns msgid + resolved scope for a `_t('…')` / aliased call.
func (c *scriptCollector) translateCallLiteral(node *tsast.Node) (src string, scope string, ok bool) {
	if node == nil || node.Kind != tsast.KindCallExpression {
		return "", "", false
	}
	callExpr := node.AsCallExpression()
	if callExpr == nil || !c.isTranslateCallee(callExpr.Expression) {
		return "", "", false
	}
	args := callExpr.Arguments
	if args == nil || len(args.Nodes) == 0 {
		return "", "", false
	}
	src, ok = stringLiteralValue(c.ctx, args.Nodes[0])
	if !ok {
		return "", "", false
	}
	calleeName := ""
	if callExpr.Expression != nil && callExpr.Expression.Kind == tsast.KindIdentifier {
		calleeName = strings.TrimSpace(c.ctx.NodeText(callExpr.Expression))
	}
	manualScope := ""
	if len(args.Nodes) > 1 {
		if s, scopeOK := parseReferenceScope(c.ctx, args.Nodes[1]); scopeOK {
			manualScope = s
		}
	}
	if manualScope == "" && calleeName != "" {
		manualScope = c.translateDefaultScopes[calleeName]
	}
	scope = ResolveI18nScope(manualScope, c.scopeStack, c.scopePath, "")
	return src, scope, true
}

func parseReferenceScope(ctx *tsgoctx.ParseCtx, node *tsast.Node) (string, bool) {
	if node == nil || node.Kind != tsast.KindObjectLiteralExpression {
		return "", false
	}
	obj := node.AsObjectLiteralExpression()
	if obj == nil || obj.Properties == nil || obj.Properties.Nodes == nil {
		return "", false
	}
	scope := ""
	path := ""
	location := ""
	for _, prop := range obj.Properties.Nodes {
		if prop == nil || prop.Kind != tsast.KindPropertyAssignment {
			continue
		}
		pa := prop.AsPropertyAssignment()
		if pa == nil || pa.Name() == nil {
			continue
		}
		key := strings.TrimSpace(ctx.NodeText(pa.Name()))
		if key != "scope" && key != "path" && key != "location" {
			continue
		}
		value, ok := stringLiteralValue(ctx, prop.Initializer())
		if !ok {
			return "", false
		}
		switch key {
		case "scope":
			scope = value
		case "path":
			path = value
		case "location":
			location = value
		}
	}
	if strings.TrimSpace(scope) != "" {
		return scope, true
	}
	formatted := FormatScope(path, location)
	return formatted, strings.TrimSpace(formatted) != ""
}

func parseCallOutput(ctx *tsgoctx.ParseCtx, node *tsast.Node) (string, bool) {
	if node == nil || node.Kind != tsast.KindObjectLiteralExpression {
		return "", false
	}
	obj := node.AsObjectLiteralExpression()
	if obj == nil || obj.Properties == nil || obj.Properties.Nodes == nil {
		return "", false
	}
	for _, prop := range obj.Properties.Nodes {
		if prop == nil || prop.Kind != tsast.KindPropertyAssignment {
			continue
		}
		pa := prop.AsPropertyAssignment()
		if pa == nil || pa.Name() == nil || strings.TrimSpace(ctx.NodeText(pa.Name())) != "output" {
			continue
		}
		value, ok := stringLiteralValue(ctx, prop.Initializer())
		if !ok || (value != "text" && value != "reference") {
			return "", false
		}
		return value, true
	}
	return "", false
}

func parseTranslateOptions(ctx *tsgoctx.ParseCtx, node *tsast.Node) (scope string, kind string) {
	kind = KindLiteral
	path := ""
	location := ""
	if node == nil || node.Kind != tsast.KindObjectLiteralExpression {
		return "", kind
	}
	obj := node.AsObjectLiteralExpression()
	if obj == nil || obj.Properties == nil || obj.Properties.Nodes == nil {
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
		case "path":
			path = val
		case "location":
			location = val
		case "kind":
			kind = val
		}
	}
	if strings.TrimSpace(scope) == "" {
		scope = FormatScope(path, location)
	}
	return scope, kind
}

func parseCreateTranslateOptions(ctx *tsgoctx.ParseCtx, node *tsast.Node) (scope string, referenceOutput bool) {
	path := ""
	location := ""
	if node == nil || node.Kind != tsast.KindObjectLiteralExpression {
		return "", false
	}
	obj := node.AsObjectLiteralExpression()
	if obj == nil || obj.Properties == nil || obj.Properties.Nodes == nil {
		return "", false
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
		value, ok := stringLiteralValue(ctx, prop.Initializer())
		if !ok {
			continue
		}
		switch key {
		case "scope":
			scope = value
		case "path":
			path = value
		case "location":
			location = value
		case "output":
			referenceOutput = value == "reference"
		}
	}
	if strings.TrimSpace(scope) == "" {
		scope = FormatScope(path, location)
	}
	return scope, referenceOutput
}

func stringLiteralValue(ctx *tsgoctx.ParseCtx, node *tsast.Node) (string, bool) {
	if node == nil {
		return "", false
	}
	if node.Kind != tsast.KindStringLiteral && node.Kind != tsast.KindNoSubstitutionTemplateLiteral {
		return "", false
	}
	text := strings.TrimSpace(ctx.NodeText(node))
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
