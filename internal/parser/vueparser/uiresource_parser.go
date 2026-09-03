// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package vueparser

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	tsast "github.com/buke/typescript-go-internal/v7/pkg/ast"
	tscore "github.com/buke/typescript-go-internal/v7/pkg/core"
	tsparser "github.com/buke/typescript-go-internal/v7/pkg/parser"
	tspath "github.com/buke/typescript-go-internal/v7/pkg/tspath"
	"github.com/choysum-dev/choysum/internal/parser"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/ettle/strcase"
)

type uiParseCtx struct {
	sourcePath  string
	source      string
	sourceFile  *tsast.SourceFile
	ownerModule string
	bindings    map[string]parser.TranslateBinding
}

func collectUiResourceDecls(sourcePath string, source string) ([]*parser.UiResourceDecl, []*parser.UiResourceDeclIssue) {
	normalized := tspath.NormalizePath(sourcePath)
	if !filepath.IsAbs(normalized) {
		absPath, err := filepath.Abs(normalized)
		if err != nil {
			return nil, nil
		}
		normalized = tspath.NormalizePath(absPath)
	}

	scriptKind := tscore.GetScriptKindFromFileName(normalized)
	if strings.EqualFold(filepath.Ext(normalized), ".vue") {
		scriptKind = tscore.ScriptKindTS
	}

	sourceFile := tsparser.ParseSourceFile(tsast.SourceFileParseOptions{
		FileName: normalized,
		Path:     tspath.ToPath(normalized, "", true),
	}, source, scriptKind)

	ctx := &uiParseCtx{
		sourcePath:  sourcePath,
		source:      source,
		sourceFile:  sourceFile,
		ownerModule: parser.DeriveOwnerModuleFromSourcePath(sourcePath),
		bindings:    parser.ParseTranslateBindings(source),
	}
	decls, issues := ctx.collectUiResourceDecls()
	return decls, issues
}

func (c *uiParseCtx) collectUiResourceDecls() ([]*parser.UiResourceDecl, []*parser.UiResourceDeclIssue) {
	if c.sourceFile == nil {
		return nil, nil
	}

	decls := make([]*parser.UiResourceDecl, 0)
	issues := make([]*parser.UiResourceDeclIssue, 0)

	var walk func(node *tsast.Node, inheritedParentMenu string)
	walk = func(node *tsast.Node, inheritedParentMenu string) {
		if node == nil {
			return
		}

		if node.Kind == tsast.KindCallExpression {
			callDecls, callIssues, menuID, recognized := c.parseUiResourceCall(node, inheritedParentMenu)
			decls = append(decls, callDecls...)
			issues = append(issues, callIssues...)

			nextParentMenu := inheritedParentMenu
			if recognized && menuID != "" {
				nextParentMenu = menuID
			}
			node.ForEachChild(func(child *tsast.Node) bool {
				walk(child, nextParentMenu)
				return false
			})
			return
		}

		node.ForEachChild(func(child *tsast.Node) bool {
			walk(child, inheritedParentMenu)
			return false
		})
	}

	for _, stmt := range c.sourceFile.Statements.Nodes {
		walk(stmt, "")
	}

	return decls, issues
}

func (c *uiParseCtx) parseUiResourceCall(call *tsast.Node, inheritedParentMenu string) ([]*parser.UiResourceDecl, []*parser.UiResourceDeclIssue, string, bool) {
	callExpr := call.AsCallExpression()
	if callExpr == nil || callExpr.Expression == nil {
		return nil, nil, "", false
	}

	fn := ""
	if callExpr.Expression.Kind == tsast.KindIdentifier {
		fn = strings.TrimSpace(callExpr.Expression.Text())
	}

	if fn != "defineRoute" && fn != "defineMenu" && fn != "defineAction" && fn != "defineModelActions" {
		return nil, nil, "", false
	}

	if callExpr.Arguments == nil || len(callExpr.Arguments.Nodes) == 0 {
		return nil, nil, "", true
	}
	args := callExpr.Arguments.Nodes

	issuePos := call.Pos()
	if callText := c.nodeText(call); callText != "" {
		if idx := strings.Index(callText, fn); idx >= 0 {
			issuePos = call.Pos() + idx
		}
	} else if callExpr.Expression != nil {
		issuePos = callExpr.Expression.Pos()
	}
	line, column := lineColumnForOffset(c.source, issuePos)
	mkIssue := func(severity parser.UiResourceIssueSeverity, code parser.UiResourceIssueCode, resourceID string, message string) *parser.UiResourceDeclIssue {
		return &parser.UiResourceDeclIssue{
			Severity:   severity,
			Code:       code,
			Factory:    fn,
			ResourceID: strings.TrimSpace(resourceID),
			Message:    message,
			SourcePath: c.sourcePath,
			Line:       line,
			Column:     column,
		}
	}

	if fn == "defineModelActions" {
		decls, issues := c.parseModelActionsDecls(args, line, column)
		return decls, issues, "", true
	}

	id := c.parseStringLiteral(args[0])
	if id == "" {
		return nil, []*parser.UiResourceDeclIssue{mkIssue(
			parser.UiResourceIssueSeverityFatal,
			parser.UiResourceIssueCodeDeclIDNotLiteral,
			"",
			"first argument must be a string literal resource id; variables or computed expressions are not supported",
		)}, "", true
	}

	decl := &parser.UiResourceDecl{
		ID:           id,
		Type:         uiTypeByFactory(fn),
		SourcePath:   c.sourcePath,
		SourceLine:   line,
		SourceColumn: column,
	}
	issues := make([]*parser.UiResourceDeclIssue, 0)
	if warning := buildUiResourceIDNamingIssue(fn, id, c.sourcePath, line, column); warning != nil {
		issues = append(issues, warning)
	}

	if len(args) > 1 {
		issues = append(issues, c.fillUiDeclFromOptions(decl, args[1], fn, line, column)...)
	}

	if decl.Type == parser.UiResourceTypeMenu && inheritedParentMenu != "" && strings.TrimSpace(decl.ParentMenu) == "" {
		decl.ParentMenu = inheritedParentMenu
	}

	if decl.Type == parser.UiResourceTypeRoute && len(args) > 1 && c.isRoutePublicByMeta(args[1]) {
		if len(decl.Actions) > 0 {
			issues = append(issues, mkIssue(
				parser.UiResourceIssueSeverityFatal,
				parser.UiResourceIssueCodePublicRouteHasActions,
				decl.ID,
				"public route cannot declare actions",
			))
		}
		return nil, issues, "", true
	}

	menuID := ""
	if decl.Type == parser.UiResourceTypeMenu {
		menuID = strings.TrimSpace(decl.ID)
	}

	return []*parser.UiResourceDecl{decl}, issues, menuID, true
}

func (c *uiParseCtx) parseModelActionsDecls(args []*tsast.Node, line int, column int) ([]*parser.UiResourceDecl, []*parser.UiResourceDeclIssue) {
	modelID := c.parseStringLiteral(args[0])
	if modelID == "" {
		return nil, []*parser.UiResourceDeclIssue{{
			Severity:   parser.UiResourceIssueSeverityFatal,
			Code:       parser.UiResourceIssueCodeModelIDNotLiteral,
			Factory:    "defineModelActions",
			Message:    "first argument must be a string literal model id in the form [application].[ModelName]",
			SourcePath: c.sourcePath,
			Line:       line,
			Column:     column,
		}}
	}

	parts := strings.Split(modelID, ".")
	if len(parts) != 2 {
		return nil, []*parser.UiResourceDeclIssue{{
			Severity:   parser.UiResourceIssueSeverityFatal,
			Code:       parser.UiResourceIssueCodeModelIDInvalidFormat,
			Factory:    "defineModelActions",
			ResourceID: modelID,
			Message:    "model id must follow [application].[ModelName]",
			SourcePath: c.sourcePath,
			Line:       line,
			Column:     column,
		}}
	}

	app := strings.TrimSpace(parts[0])
	modelName := strings.TrimSpace(parts[1])
	if app == "" || modelName == "" {
		return nil, []*parser.UiResourceDeclIssue{{
			Severity:   parser.UiResourceIssueSeverityFatal,
			Code:       parser.UiResourceIssueCodeModelIDEmptySegment,
			Factory:    "defineModelActions",
			ResourceID: modelID,
			Message:    "model id must not contain empty application or model name",
			SourcePath: c.sourcePath,
			Line:       line,
			Column:     column,
		}}
	}

	issues := make([]*parser.UiResourceDeclIssue, 0)
	entityTitle := ""
	entityTitleText := (*meta.TermReference)(nil)
	titleOverrides := map[string]string{}
	titleOverrideTexts := map[string]*meta.TermReference{}
	if len(args) > 1 {
		issues = append(issues, c.fillUiDeclFromOptions(&parser.UiResourceDecl{}, args[1], "defineModelActions", line, column)...)
		parsedEntityTitle, parsedEntityTitleText, parsedTitleOverrides, parsedTitleOverrideTexts, titleIssues := c.parseModelActionDisplayOptions(args[1], line, column)
		entityTitle = parsedEntityTitle
		entityTitleText = parsedEntityTitleText
		titleOverrides = parsedTitleOverrides
		titleOverrideTexts = parsedTitleOverrideTexts
		issues = append(issues, titleIssues...)
	}

	excludes := map[string]bool{}
	if len(args) > 1 {
		excludes = c.parseExcludeMap(args[1])
	}

	ops := []string{"create", "edit", "delete", "copy"}
	decls := make([]*parser.UiResourceDecl, 0, len(ops))
	for _, op := range ops {
		if excludes[op] {
			continue
		}
		requires := requiresForModelAction(app, modelName, op)
		id := fmt.Sprintf("%s.action.%s_%s", app, strcase.ToSnake(modelName), op)
		title, titleText := resolveModelActionTitleParts(op, entityTitle, entityTitleText, titleOverrides, titleOverrideTexts)
		decls = append(decls, &parser.UiResourceDecl{
			ID:           id,
			Type:         parser.UiResourceTypeAction,
			Title:        title,
			TitleText:    titleText,
			Requires:     requires,
			SourcePath:   c.sourcePath,
			SourceLine:   line,
			SourceColumn: column,
		})
	}

	return decls, issues
}

func (c *uiParseCtx) parseModelActionDisplayOptions(options *tsast.Node, line int, column int) (string, *meta.TermReference, map[string]string, map[string]*meta.TermReference, []*parser.UiResourceDeclIssue) {
	obj := c.toObjectLiteral(options)
	if obj == nil {
		return "", nil, map[string]string{}, map[string]*meta.TermReference{}, nil
	}

	issues := make([]*parser.UiResourceDeclIssue, 0)
	entityTitle := ""
	var entityTitleText *meta.TermReference
	titles := map[string]string{}
	titleTexts := map[string]*meta.TermReference{}
	appendFatal := func(code parser.UiResourceIssueCode, message string) {
		issues = append(issues, &parser.UiResourceDeclIssue{
			Severity:   parser.UiResourceIssueSeverityFatal,
			Code:       code,
			Factory:    "defineModelActions",
			Message:    message,
			SourcePath: c.sourcePath,
			Line:       line,
			Column:     column,
		})
	}

	for _, property := range obj.AsObjectLiteralExpression().Properties.Nodes {
		name := c.literalPropertyName(property)
		if name == "" {
			continue
		}
		propertyText := c.nodeText(property)
		switch name {
		case "entityTitle":
			valueExpr := strings.TrimSpace(parsePropertyValueExpr(propertyText))
			title, titleText, ok := parser.ParseResourceTitleExpr(valueExpr, c.ownerModule, c.bindings)
			if !ok {
				appendFatal(parser.UiResourceIssueCodeModelActionEntityTitleNotLiteral, "entityTitle must be a string literal or _lt('literal'[, opts])")
				break
			}
			entityTitle = title
			entityTitleText = titleText
		case "titles":
			parsedTitles, parsedTitleTexts, err := parseStrictModelActionTitlesProperty(propertyText, c.ownerModule, c.bindings)
			if err != nil {
				appendFatal(parser.UiResourceIssueCodeModelActionTitlesInvalid, fmt.Sprintf("titles must be an object literal like { create: 'Create User' }; %s", err.Error()))
				break
			}
			for op, title := range parsedTitles {
				titles[op] = title
			}
			for op, titleText := range parsedTitleTexts {
				titleTexts[op] = titleText
			}
		}
	}

	return entityTitle, entityTitleText, titles, titleTexts, issues
}

func parseStrictModelActionTitlesProperty(propertyText string, ownerModule string, bindings map[string]parser.TranslateBinding) (map[string]string, map[string]*meta.TermReference, error) {
	expr := parsePropertyValueExpr(propertyText)
	if expr == "" {
		return nil, nil, fmt.Errorf("value is missing")
	}
	if !strings.HasPrefix(expr, "{") || !strings.HasSuffix(expr, "}") {
		return nil, nil, fmt.Errorf("found non-object expression %q", expr)
	}

	body := strings.TrimSpace(expr[1 : len(expr)-1])
	if body == "" {
		return map[string]string{}, map[string]*meta.TermReference{}, nil
	}

	fields, err := splitTopLevelJSList(body)
	if err != nil {
		return nil, nil, err
	}

	allowed := map[string]bool{"create": true, "edit": true, "delete": true, "copy": true}
	out := make(map[string]string, len(fields))
	titleTexts := make(map[string]*meta.TermReference, len(fields))
	seen := make(map[string]bool, len(fields))
	for _, field := range fields {
		parts := strings.SplitN(field, ":", 2)
		if len(parts) != 2 {
			return nil, nil, fmt.Errorf("found non-literal field %q", strings.TrimSpace(field))
		}

		key := strings.TrimSpace(parts[0])
		if strings.HasPrefix(key, "'") || strings.HasPrefix(key, "\"") {
			unquoted, err := strconv.Unquote(key)
			if err != nil {
				return nil, nil, fmt.Errorf("invalid quoted property name %q", key)
			}
			key = unquoted
		}
		if !allowed[key] {
			return nil, nil, fmt.Errorf("unsupported action key %q", key)
		}
		if seen[key] {
			return nil, nil, fmt.Errorf("found duplicate property %q", key)
		}
		seen[key] = true

		value, titleText, ok := parser.ParseResourceTitleExpr(strings.TrimSpace(parts[1]), ownerModule, bindings)
		if !ok {
			return nil, nil, fmt.Errorf("property %q must be a string literal or _lt('literal'[, opts])", key)
		}
		out[key] = value
		if titleText != nil {
			titleTexts[key] = titleText
		}
	}

	return out, titleTexts, nil
}

func resolveModelActionTitleParts(op string, entityTitle string, entityTitleText *meta.TermReference, overrides map[string]string, overrideTexts map[string]*meta.TermReference) (string, *meta.TermReference) {
	if title, ok := overrides[op]; ok && strings.TrimSpace(title) != "" {
		if titleText, ok := overrideTexts[op]; ok && titleText != nil {
			return strings.TrimSpace(title), titleText
		}
		return strings.TrimSpace(title), nil
	}

	entityTitle = strings.TrimSpace(entityTitle)
	if entityTitle == "" {
		return "", nil
	}

	prefix := map[string]string{
		"create": "Create ",
		"edit":   "Edit ",
		"delete": "Delete ",
		"copy":   "Copy ",
	}[op]
	if prefix == "" {
		return "", nil
	}
	title := prefix + entityTitle
	if entityTitleText == nil {
		return title, nil
	}
	return title, parser.CloneTermReferenceWithSrc(entityTitleText, title)
}

func resolveModelActionTitle(op string, entityTitle string, overrides map[string]string) string {
	title, _ := resolveModelActionTitleParts(op, entityTitle, nil, overrides, nil)
	return title
}

func requiresForModelAction(app string, modelName string, op string) []string {
	method := ""
	switch op {
	case "create":
		method = "Create"
	case "edit":
		method = "Update"
	case "delete":
		method = "Delete"
	default:
		return nil
	}
	return []string{fmt.Sprintf("rpc:/%s.%s/%s", app, modelName, method)}
}

func (c *uiParseCtx) parseExcludeMap(options *tsast.Node) map[string]bool {
	result := map[string]bool{}
	if options == nil {
		return result
	}

	re := regexp.MustCompile(`exclude\s*:\s*\[([^\]]*)\]`)
	m := re.FindStringSubmatch(c.nodeText(options))
	if len(m) != 2 {
		return result
	}
	for _, v := range parseQuotedStrings(m[1]) {
		if v != "" {
			result[v] = true
		}
	}
	return result
}

func (c *uiParseCtx) fillUiDeclFromOptions(decl *parser.UiResourceDecl, options *tsast.Node, factory string, line int, column int) []*parser.UiResourceDeclIssue {
	if decl == nil || options == nil {
		return nil
	}
	obj := c.toObjectLiteral(options)
	if obj == nil {
		return nil
	}

	issues := make([]*parser.UiResourceDeclIssue, 0)
	var metaProperty *tsast.Node
	appendFatal := func(code parser.UiResourceIssueCode, message string) {
		issues = append(issues, &parser.UiResourceDeclIssue{
			Severity:   parser.UiResourceIssueSeverityFatal,
			Code:       code,
			Factory:    factory,
			ResourceID: strings.TrimSpace(decl.ID),
			Message:    message,
			SourcePath: c.sourcePath,
			Line:       line,
			Column:     column,
		})
	}

	for _, property := range obj.AsObjectLiteralExpression().Properties.Nodes {
		name := c.literalPropertyName(property)
		if name == "" {
			continue
		}
		propertyText := c.nodeText(property)
		switch name {
		case "title":
			valueExpr := strings.TrimSpace(parsePropertyValueExpr(propertyText))
			if title, titleText, ok := parser.ParseResourceTitleExpr(valueExpr, c.ownerModule, c.bindings); ok {
				decl.Title = title
				decl.TitleText = titleText
			}
		case "sequence":
			if v, ok := parseNumericFromText(propertyText); ok {
				decl.Sequence = v
			}
		case "requires":
			requires, err := parseStrictRequiresProperty(propertyText)
			if err != nil {
				appendFatal(parser.UiResourceIssueCodeDeclRequiresNotLiteral, fmt.Sprintf("requires must be an object-literal array like [{ model: 'auth.User' }] or [{ model: 'auth.User', method: 'Browse' }]; %s", err.Error()))
				break
			}
			decl.Requires = requires
		case "actions":
			if factory != "defineRoute" {
				break
			}
			actions, err := parseStrictStringArrayProperty(propertyText)
			if err != nil {
				code := parser.UiResourceIssueCodeRouteActionsNotLiteral
				if strings.Contains(err.Error(), "dynamic entries") {
					code = parser.UiResourceIssueCodeRouteActionEntryNotLiteral
				}
				appendFatal(code, fmt.Sprintf("actions must be a string-literal array; %s", err.Error()))
				break
			}
			decl.Actions = actions
		case "parentMenu":
			if factory != "defineMenu" {
				appendFatal(parser.UiResourceIssueCodeParentMenuOnlyForMenu, "parentMenu is only supported on defineMenu; use route.actions and relation inference for ROUTE/ACTION topology")
				break
			}
			decl.ParentMenu = parseFirstQuotedString(propertyText)
		case "path":
			decl.Path = parseFirstQuotedString(propertyText)
		case "defaultRoles":
			defaultRoles, err := parseStrictStringArrayProperty(propertyText)
			if err != nil {
				appendFatal(parser.UiResourceIssueCodeDeclDefaultRolesNotLiteral, fmt.Sprintf("defaultRoles must be a string-literal array; %s", err.Error()))
				break
			}
			decl.DefaultRoles = defaultRoles
		case "override":
			decl.Override = parseBooleanProperty(propertyText, decl.Override)
		case "meta":
			metaProperty = property
		}
	}

	if metaProperty != nil {
		issues = append(issues, c.fillUiDeclFromMeta(decl, metaProperty, factory, line, column)...)
	}

	return issues
}

func (c *uiParseCtx) fillUiDeclFromMeta(decl *parser.UiResourceDecl, metaProperty *tsast.Node, factory string, line int, column int) []*parser.UiResourceDeclIssue {
	if decl == nil || metaProperty == nil {
		return nil
	}
	metaValue := c.propertyValueNode(metaProperty)
	metaObj := c.toObjectLiteral(metaValue)
	if metaObj == nil {
		if decl.Title == "" {
			if v := parseNamedQuotedString(c.nodeText(metaProperty), "pageTitle"); v != "" {
				decl.Title = v
			}
		}
		return nil
	}

	issues := make([]*parser.UiResourceDeclIssue, 0)
	var resourceProperty *tsast.Node
	for _, property := range metaObj.AsObjectLiteralExpression().Properties.Nodes {
		name := c.literalPropertyName(property)
		if name == "" {
			continue
		}
		switch name {
		case "pageTitle":
			if decl.Title == "" {
				if v := parseFirstQuotedString(c.nodeText(property)); v != "" {
					decl.Title = v
				}
			}
		case "resource":
			resourceProperty = property
		}
	}

	if resourceProperty != nil {
		issues = append(issues, c.fillUiDeclFromResourceContract(decl, resourceProperty, factory, line, column)...)
	}

	return issues
}

func (c *uiParseCtx) fillUiDeclFromResourceContract(decl *parser.UiResourceDecl, resourceProperty *tsast.Node, factory string, line int, column int) []*parser.UiResourceDeclIssue {
	if decl == nil || resourceProperty == nil {
		return nil
	}
	resourceValue := c.propertyValueNode(resourceProperty)
	resourceObj := c.toObjectLiteral(resourceValue)
	if resourceObj == nil {
		return nil
	}

	issues := make([]*parser.UiResourceDeclIssue, 0)
	appendFatal := func(code parser.UiResourceIssueCode, message string) {
		issues = append(issues, &parser.UiResourceDeclIssue{
			Severity:   parser.UiResourceIssueSeverityFatal,
			Code:       code,
			Factory:    factory,
			ResourceID: strings.TrimSpace(decl.ID),
			Message:    message,
			SourcePath: c.sourcePath,
			Line:       line,
			Column:     column,
		})
	}

	for _, property := range resourceObj.AsObjectLiteralExpression().Properties.Nodes {
		name := c.literalPropertyName(property)
		if name == "" {
			continue
		}
		propertyText := c.nodeText(property)
		switch name {
		case "title":
			valueExpr := strings.TrimSpace(parsePropertyValueExpr(propertyText))
			if title, titleText, ok := parser.ParseResourceTitleExpr(valueExpr, c.ownerModule, c.bindings); ok {
				decl.Title = title
				decl.TitleText = titleText
			}
		case "sequence":
			if v, ok := parseNumericFromText(propertyText); ok {
				decl.Sequence = v
			}
		case "requires":
			requires, err := parseStrictRequiresProperty(propertyText)
			if err != nil {
				appendFatal(parser.UiResourceIssueCodeDeclRequiresNotLiteral, fmt.Sprintf("meta.resource.requires must be an object-literal array like [{ model: 'auth.User' }]; %s", err.Error()))
				break
			}
			decl.Requires = requires
		case "actions":
			if factory != "defineRoute" {
				break
			}
			actions, err := parseStrictStringArrayProperty(propertyText)
			if err != nil {
				code := parser.UiResourceIssueCodeRouteActionsNotLiteral
				if strings.Contains(err.Error(), "dynamic entries") {
					code = parser.UiResourceIssueCodeRouteActionEntryNotLiteral
				}
				appendFatal(code, fmt.Sprintf("meta.resource.actions must be a string-literal array; %s", err.Error()))
				break
			}
			decl.Actions = actions
		case "parentMenu":
			if factory != "defineMenu" {
				appendFatal(parser.UiResourceIssueCodeParentMenuOnlyForMenu, "meta.resource.parentMenu is only supported on defineMenu")
				break
			}
			decl.ParentMenu = parseFirstQuotedString(propertyText)
		case "path":
			decl.Path = parseFirstQuotedString(propertyText)
		case "defaultRoles":
			defaultRoles, err := parseStrictStringArrayProperty(propertyText)
			if err != nil {
				appendFatal(parser.UiResourceIssueCodeDeclDefaultRolesNotLiteral, fmt.Sprintf("meta.resource.defaultRoles must be a string-literal array; %s", err.Error()))
				break
			}
			decl.DefaultRoles = defaultRoles
		case "override":
			decl.Override = parseBooleanProperty(propertyText, decl.Override)
		}
	}

	return issues
}

func uiTypeByFactory(fn string) parser.UiResourceType {
	switch fn {
	case "defineRoute":
		return parser.UiResourceTypeRoute
	case "defineMenu":
		return parser.UiResourceTypeMenu
	default:
		return parser.UiResourceTypeAction
	}
}

func buildUiResourceIDNamingIssue(factory string, id string, sourcePath string, line int, column int) *parser.UiResourceDeclIssue {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil
	}

	expectedType := ""
	switch factory {
	case "defineRoute":
		expectedType = "route"
	case "defineMenu":
		expectedType = "menu"
	case "defineAction":
		expectedType = "action"
	default:
		return nil
	}

	parts := strings.Split(id, ".")
	if len(parts) == 3 {
		app := strings.TrimSpace(parts[0])
		typ := strings.TrimSpace(parts[1])
		name := strings.TrimSpace(parts[2])
		if recommendedSegment(app) && typ == expectedType && recommendedSegment(name) {
			return nil
		}
	}

	return &parser.UiResourceDeclIssue{
		Severity:   parser.UiResourceIssueSeverityWarning,
		Code:       parser.UiResourceIssueCodeDeclIDNamingSuggested,
		Factory:    factory,
		ResourceID: id,
		Message:    fmt.Sprintf("resource id %q does not follow the recommended naming form [application].%s.[name] with lowercase snake_case segments", id, expectedType),
		SourcePath: sourcePath,
		Line:       line,
		Column:     column,
	}
}

func recommendedSegment(segment string) bool {
	if segment == "" {
		return false
	}
	matched, _ := regexp.MatchString(`^[a-z][a-z0-9_]*$`, segment)
	return matched
}

func (c *uiParseCtx) parseStringLiteral(node *tsast.Node) string {
	if node == nil {
		return ""
	}
	text := strings.TrimSpace(c.nodeText(node))
	value, err := parseJSStringLiteral(text)
	if err != nil {
		return ""
	}
	return value
}

func parseNumericFromText(text string) (int, bool) {
	if text == "" {
		return 0, false
	}
	re := regexp.MustCompile(`:\s*(-?\d+)`)
	m := re.FindStringSubmatch(text)
	if len(m) != 2 {
		return 0, false
	}
	v, err := strconv.Atoi(strings.TrimSpace(m[1]))
	if err != nil {
		return 0, false
	}
	return v, true
}

func (c *uiParseCtx) literalPropertyName(property *tsast.Node) string {
	if property == nil {
		return ""
	}

	var nameNode *tsast.Node
	switch property.Kind {
	case tsast.KindPropertyAssignment:
		nameNode = property.AsPropertyAssignment().Name()
	case tsast.KindShorthandPropertyAssignment:
		nameNode = property.AsShorthandPropertyAssignment().Name()
	default:
		return ""
	}

	if nameNode == nil {
		return ""
	}
	name := strings.TrimSpace(c.nodeText(nameNode))
	if name == "" {
		name = strings.TrimSpace(nameNode.Text())
	}
	if name == "" {
		return ""
	}

	if value, err := parseJSStringLiteral(name); err == nil {
		return strings.TrimSpace(value)
	}
	return strings.TrimSpace(name)
}

func parseQuotedStrings(text string) []string {
	if text == "" {
		return nil
	}
	re := regexp.MustCompile(`['"]([^'"]+)['"]`)
	matches := re.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return nil
	}
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		if len(m) == 2 {
			out = append(out, m[1])
		}
	}
	return out
}

func parseFirstQuotedString(text string) string {
	v := parseQuotedStrings(text)
	if len(v) == 0 {
		return ""
	}
	return v[0]
}

func parseStrictStringArrayProperty(propertyText string) ([]string, error) {
	expr := parsePropertyValueExpr(propertyText)
	if expr == "" {
		return nil, fmt.Errorf("value is missing")
	}
	if !strings.HasPrefix(expr, "[") || !strings.HasSuffix(expr, "]") {
		return nil, fmt.Errorf("found non-array expression %q", expr)
	}

	inner := strings.TrimSpace(expr[1 : len(expr)-1])
	if inner == "" {
		return nil, nil
	}

	stringLiteralRe := regexp.MustCompile(`"[^"\\]*(?:\\.[^"\\]*)*"|'[^'\\]*(?:\\.[^'\\]*)*'`)
	reduced := stringLiteralRe.ReplaceAllString(inner, "")
	reduced = strings.ReplaceAll(reduced, ",", "")
	reduced = strings.TrimSpace(reduced)
	if reduced != "" {
		return nil, fmt.Errorf("found dynamic entries %q", reduced)
	}

	return normalizeStringLiterals(parseQuotedStrings(expr)), nil
}

func parseStrictRequiresProperty(propertyText string) ([]string, error) {
	expr := parsePropertyValueExpr(propertyText)
	if expr == "" {
		return nil, fmt.Errorf("value is missing")
	}
	if !strings.HasPrefix(expr, "[") || !strings.HasSuffix(expr, "]") {
		return nil, fmt.Errorf("found non-array expression %q", expr)
	}

	inner := strings.TrimSpace(expr[1 : len(expr)-1])
	if inner == "" {
		return nil, nil
	}

	items, err := splitTopLevelJSList(inner)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, nil
	}

	requires := make([]string, 0, len(items))
	seen := make(map[string]bool, len(items))
	for _, item := range items {
		requireValue, err := parseRequireObjectLiteral(item)
		if err != nil {
			return nil, err
		}
		if seen[requireValue] {
			continue
		}
		seen[requireValue] = true
		requires = append(requires, requireValue)
	}
	return requires, nil
}

func normalizeStringLiterals(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	seen := make(map[string]bool, len(values))
	for _, raw := range values {
		value := strings.TrimSpace(raw)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func parseBooleanProperty(propertyText string, fallback bool) bool {
	expr := strings.TrimSpace(parsePropertyValueExpr(propertyText))
	switch expr {
	case "true":
		return true
	case "false":
		return false
	default:
		return fallback
	}
}

func parsePropertyValueExpr(propertyText string) string {
	idx := strings.Index(propertyText, ":")
	if idx < 0 {
		return ""
	}
	return strings.TrimSpace(propertyText[idx+1:])
}

func (c *uiParseCtx) propertyValueNode(property *tsast.Node) *tsast.Node {
	if property == nil {
		return nil
	}
	switch property.Kind {
	case tsast.KindPropertyAssignment:
		return property.AsPropertyAssignment().Initializer
	default:
		return nil
	}
}

func splitTopLevelJSList(text string) ([]string, error) {
	items := make([]string, 0)
	start := 0
	depthBrace := 0
	depthBracket := 0
	inQuote := byte(0)
	escaped := false
	lastWasTopLevelComma := false

	for i := 0; i < len(text); i++ {
		ch := text[i]
		if inQuote != 0 {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == inQuote {
				inQuote = 0
			}
			continue
		}

		switch ch {
		case '\'', '"':
			inQuote = ch
		case '{':
			depthBrace++
		case '}':
			depthBrace--
			if depthBrace < 0 {
				return nil, fmt.Errorf("found unmatched closing brace")
			}
		case '[':
			depthBracket++
		case ']':
			depthBracket--
			if depthBracket < 0 {
				return nil, fmt.Errorf("found unmatched closing bracket")
			}
		case ',':
			if depthBrace == 0 && depthBracket == 0 {
				item := strings.TrimSpace(text[start:i])
				if item == "" {
					return nil, fmt.Errorf("found empty array entry")
				}
				items = append(items, item)
				start = i + 1
				lastWasTopLevelComma = true
			}
		default:
			if !isASCIISpace(ch) {
				lastWasTopLevelComma = false
			}
		}
	}

	if inQuote != 0 {
		return nil, fmt.Errorf("found unterminated string literal")
	}
	if depthBrace != 0 || depthBracket != 0 {
		return nil, fmt.Errorf("found unbalanced object or array literal")
	}

	last := strings.TrimSpace(text[start:])
	if last == "" {
		if len(items) > 0 && lastWasTopLevelComma {
			return items, nil
		}
		return nil, fmt.Errorf("found trailing comma or empty array entry")
	}
	items = append(items, last)
	return items, nil
}

func isASCIISpace(ch byte) bool {
	switch ch {
	case ' ', '\t', '\n', '\r':
		return true
	default:
		return false
	}
}

func parseRequireObjectLiteral(item string) (string, error) {
	item = strings.TrimSpace(item)
	if !strings.HasPrefix(item, "{") || !strings.HasSuffix(item, "}") {
		return "", fmt.Errorf("found non-object entry %q", item)
	}
	body := strings.TrimSpace(item[1 : len(item)-1])
	if body == "" {
		return "", fmt.Errorf("found empty object entry")
	}

	fields, err := splitTopLevelJSList(body)
	if err != nil {
		return "", err
	}

	kind := "rpc"
	model := ""
	method := "*"
	hasMethod := false
	seen := map[string]bool{}
	for _, field := range fields {
		parts := strings.SplitN(field, ":", 2)
		if len(parts) != 2 {
			return "", fmt.Errorf("found non-literal field %q", strings.TrimSpace(field))
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if strings.HasPrefix(key, "'") || strings.HasPrefix(key, "\"") {
			unquoted, err := strconv.Unquote(key)
			if err != nil {
				return "", fmt.Errorf("invalid quoted property name %q", key)
			}
			key = unquoted
		}
		if key == "" {
			return "", fmt.Errorf("found empty property name")
		}
		if seen[key] {
			return "", fmt.Errorf("found duplicate property %q", key)
		}
		seen[key] = true

		literal, err := parseJSStringLiteral(value)
		if err != nil {
			return "", fmt.Errorf("property %q must be a string literal", key)
		}

		switch key {
		case "kind":
			kind = literal
		case "model":
			model = literal
		case "method":
			hasMethod = true
			method = literal
		default:
			return "", fmt.Errorf("unknown property %q", key)
		}
	}

	if kind == "" {
		return "", fmt.Errorf("property %q must not be empty", "kind")
	}
	if kind != "rpc" {
		return "", fmt.Errorf("property %q must be %q", "kind", "rpc")
	}
	if strings.TrimSpace(model) == "" {
		return "", fmt.Errorf("missing required property %q", "model")
	}
	if hasMethod && strings.TrimSpace(method) == "" {
		return "", fmt.Errorf("property %q must not be empty", "method")
	}

	return fmt.Sprintf("rpc:/%s/%s", strings.TrimSpace(model), strings.TrimSpace(method)), nil
}

func parseJSStringLiteral(text string) (string, error) {
	return parser.ParseJSStringLiteral(text)
}

func parseNamedQuotedString(text string, name string) string {
	re := regexp.MustCompile(name + `\s*:\s*['"]([^'"]+)['"]`)
	m := re.FindStringSubmatch(text)
	if len(m) != 2 {
		return ""
	}
	return m[1]
}

func (c *uiParseCtx) isRoutePublicByMeta(options *tsast.Node) bool {
	if options == nil {
		return false
	}
	text := c.nodeText(options)
	if text == "" {
		return false
	}
	re := regexp.MustCompile(`meta\s*:\s*\{[\s\S]*?requiresAuth\s*:\s*false[\s\S]*?\}`)
	return re.MatchString(text)
}

func (c *uiParseCtx) toObjectLiteral(options *tsast.Node) *tsast.Node {
	if options == nil {
		return nil
	}
	if options.Kind == tsast.KindObjectLiteralExpression {
		return options
	}
	// Keep behavior aligned with legacy parser: do not infer options from
	// call-expression arguments like fn({ ... }). Only direct/object-expression
	// wrappers should be considered.
	if options.Kind == tsast.KindCallExpression {
		return nil
	}

	var objectLiteral *tsast.Node
	options.ForEachChild(func(child *tsast.Node) bool {
		if child != nil && child.Kind == tsast.KindObjectLiteralExpression {
			objectLiteral = child
			return true
		}
		return false
	})
	return objectLiteral
}

func (c *uiParseCtx) nodeText(node *tsast.Node) string {
	if node == nil {
		return ""
	}
	start, end := node.Pos(), node.End()
	if start < 0 || end < start || end > len(c.source) {
		return ""
	}
	return c.source[start:end]
}

func lineColumnForOffset(source string, offset int) (int, int) {
	if offset < 0 {
		return 1, 1
	}
	if offset > len(source) {
		offset = len(source)
	}
	line := 1
	column := 1
	for i := 0; i < offset; i++ {
		if source[i] == '\n' {
			line++
			column = 1
			continue
		}
		column++
	}
	return line, column
}
