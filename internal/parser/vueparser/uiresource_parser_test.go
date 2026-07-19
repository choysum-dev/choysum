// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package vueparser

import (
	"path/filepath"
	"strings"
	"testing"

	tsast "github.com/buke/typescript-go-internal/pkg/ast"
	tscore "github.com/buke/typescript-go-internal/pkg/core"
	tsparser "github.com/buke/typescript-go-internal/pkg/parser"
	tspath "github.com/buke/typescript-go-internal/pkg/tspath"
	"github.com/choysum-dev/choysum/internal/parser"
)

func TestCollectUiResourceDeclsParsesRoutesMenusActionsAndModelActions(t *testing.T) {
	sourcePath := filepath.Join(t.TempDir(), "ui_resources.ts")
	source := `
defineRoute('auth.route.user_list', {
	sequence: 7,
	requires: [{ model: 'auth.User', method: 'Browse' }],
	actions: ['auth.action.user_create'],
	path: '/users',
	defaultRoles: ['admin', "manager"],
	override: true,
	meta: { pageTitle: 'Users', requiresAuth: true },
})

defineMenu('auth.menu.management', {
	title: 'Management',
	parentMenu: 'auth.menu.root',
})

defineAction('Auth.Action.BadCase', {
	title: 'Legacy Action',
	parentMenu: 'auth.menu.root',
})

defineRoute('auth.route.public_page', {
	actions: ['auth.action.should_fail'],
	meta: { requiresAuth: false },
})

defineModelActions('auth.User', {
	entityTitle: 'User',
	titles: { create: 'Create User', edit: 'Edit User' },
	exclude: ['delete', 'copy'],
})
`

	decls, issues := collectUiResourceDecls(sourcePath, source)
	if len(decls) != 5 {
		t.Fatalf("collectUiResourceDecls() decl count = %d, want 5", len(decls))
	}

	declByID := make(map[string]*parser.UiResourceDecl, len(decls))
	for _, decl := range decls {
		declByID[decl.ID] = decl
	}

	route := declByID["auth.route.user_list"]
	if route == nil {
		t.Fatalf("expected route declaration, got %#v", declByID)
	}
	if route.Type != parser.UiResourceTypeRoute || route.Title != "Users" || route.Sequence != 7 {
		t.Fatalf("unexpected route declaration: %#v", route)
	}
	if len(route.Requires) != 1 || route.Requires[0] != "rpc:/auth.User/Browse" {
		t.Fatalf("unexpected route requires: %#v", route.Requires)
	}
	if len(route.Actions) != 1 || route.Actions[0] != "auth.action.user_create" {
		t.Fatalf("unexpected route actions: %#v", route.Actions)
	}
	if route.Path != "/users" || !route.Override {
		t.Fatalf("unexpected route path/override: %#v", route)
	}
	if len(route.DefaultRoles) != 2 || route.DefaultRoles[0] != "admin" || route.DefaultRoles[1] != "manager" {
		t.Fatalf("unexpected route default roles: %#v", route.DefaultRoles)
	}

	menu := declByID["auth.menu.management"]
	if menu == nil || menu.Type != parser.UiResourceTypeMenu || menu.ParentMenu != "auth.menu.root" {
		t.Fatalf("unexpected menu declaration: %#v", menu)
	}

	action := declByID["Auth.Action.BadCase"]
	if action == nil || action.Type != parser.UiResourceTypeAction || action.Title != "Legacy Action" {
		t.Fatalf("unexpected action declaration: %#v", action)
	}

	createAction := declByID["auth.action.user_create"]
	if createAction == nil || createAction.Title != "Create User" {
		t.Fatalf("unexpected create model action: %#v", createAction)
	}
	if len(createAction.Requires) != 1 || createAction.Requires[0] != "rpc:/auth.User/Create" {
		t.Fatalf("unexpected create model action requires: %#v", createAction.Requires)
	}

	editAction := declByID["auth.action.user_edit"]
	if editAction == nil || editAction.Title != "Edit User" {
		t.Fatalf("unexpected edit model action: %#v", editAction)
	}
	if len(editAction.Requires) != 1 || editAction.Requires[0] != "rpc:/auth.User/Update" {
		t.Fatalf("unexpected edit model action requires: %#v", editAction.Requires)
	}
	if declByID["auth.action.user_delete"] != nil || declByID["auth.action.user_copy"] != nil {
		t.Fatalf("exclude should skip delete/copy model actions: %#v", declByID)
	}

	issueCodes := make(map[parser.UiResourceIssueCode]int, len(issues))
	for _, issue := range issues {
		issueCodes[issue.Code]++
	}
	if issueCodes[parser.UiResourceIssueCodeDeclIDNamingSuggested] != 1 {
		t.Fatalf("expected naming warning, got %#v", issues)
	}
	if issueCodes[parser.UiResourceIssueCodeParentMenuOnlyForMenu] != 1 {
		t.Fatalf("expected parentMenu fatal issue, got %#v", issues)
	}
	if issueCodes[parser.UiResourceIssueCodePublicRouteHasActions] != 1 {
		t.Fatalf("expected public route fatal issue, got %#v", issues)
	}
}

func TestCollectUiResourceDeclsParsesModelActionEntityTitleTermReference(t *testing.T) {
	sourcePath := filepath.Join("modules", "base", "web", "views", "CountryListView.vue")
	source := `
import { createTranslate } from '@/web/web/i18n';
import { defineModelActions } from '@/core/web/resource';

const { _t: _tRef } = createTranslate('base', { output: 'reference', scope: 'web/views/CountryListView' });
const countryActions = defineModelActions('base.Country', { entityTitle: _tRef('Country') });
`

	decls, issues := collectUiResourceDecls(sourcePath, source)
	if len(issues) > 0 {
		t.Fatalf("collectUiResourceDecls() issues = %#v", issues)
	}

	declByID := make(map[string]*parser.UiResourceDecl, len(decls))
	for _, decl := range decls {
		declByID[decl.ID] = decl
	}

	createAction := declByID["base.action.country_create"]
	if createAction == nil {
		t.Fatalf("expected create action decl, got %#v", declByID)
	}
	if createAction.Title != "Create Country" {
		t.Fatalf("unexpected synthesized title: %q", createAction.Title)
	}
	if createAction.TitleText == nil || createAction.TitleText.Src != "Create Country" {
		t.Fatalf("unexpected titleText: %#v", createAction.TitleText)
	}
	if createAction.TitleText.Module != "base" || createAction.TitleText.Scope != "web/views/CountryListView" {
		t.Fatalf("unexpected titleText metadata: %#v", createAction.TitleText)
	}
}

func TestCollectUiResourceDeclsMetaResourceContractOverridesLegacyFields(t *testing.T) {
	sourcePath := filepath.Join(t.TempDir(), "ui_resources_meta_contract.ts")
	source := `
defineRoute('auth.route.user_list', {
	title: 'Legacy Users',
	sequence: 7,
	requires: [{ model: 'auth.User', method: 'Browse' }],
	defaultRoles: ['legacy.user'],
	actions: ['auth.action.legacy'],
	path: '/legacy/users',
	meta: {
		pageTitle: 'Users',
		resource: {
			title: 'Contract Users',
			sequence: 30,
			requires: [{ model: 'auth.User' }, { model: 'auth.User' }, { model: 'auth.User', method: 'Browse' }],
			defaultRoles: ['base.user', ' base.user ', 'admin'],
			actions: ['auth.action.user_export', ' auth.action.user_export '],
			path: '/auth/users',
			override: true,
		},
	},
})

defineMenu('auth.menu.user_list', {
	title: 'Legacy Menu',
	meta: {
		resource: {
			title: 'Users Menu',
			path: '/auth/users',
			parentMenu: 'auth.menu.root',
			defaultRoles: ['base.user', 'base.user'],
			requires: [{ model: 'auth.User' }, { model: 'auth.User' }],
		},
	},
})
`

	decls, issues := collectUiResourceDecls(sourcePath, source)
	if len(issues) != 0 {
		t.Fatalf("collectUiResourceDecls() issues = %#v", issues)
	}
	if len(decls) != 2 {
		t.Fatalf("collectUiResourceDecls() decl count = %d, want 2", len(decls))
	}

	declByID := make(map[string]*parser.UiResourceDecl, len(decls))
	for _, decl := range decls {
		declByID[decl.ID] = decl
	}

	route := declByID["auth.route.user_list"]
	if route == nil {
		t.Fatalf("expected route declaration, got %#v", declByID)
	}
	if route.Title != "Contract Users" || route.Sequence != 30 || route.Path != "/auth/users" || !route.Override {
		t.Fatalf("unexpected route contract projection: %#v", route)
	}
	if !equalTrimmedStringsForTest(route.Requires, []string{"rpc:/auth.User/*", "rpc:/auth.User/Browse"}) {
		t.Fatalf("unexpected route requires: %#v", route.Requires)
	}
	if !equalTrimmedStringsForTest(route.DefaultRoles, []string{"base.user", "admin"}) {
		t.Fatalf("unexpected route defaultRoles: %#v", route.DefaultRoles)
	}
	if !equalTrimmedStringsForTest(route.Actions, []string{"auth.action.user_export"}) {
		t.Fatalf("unexpected route actions: %#v", route.Actions)
	}

	menu := declByID["auth.menu.user_list"]
	if menu == nil {
		t.Fatalf("expected menu declaration, got %#v", declByID)
	}
	if menu.Title != "Users Menu" || menu.Path != "/auth/users" || menu.ParentMenu != "auth.menu.root" {
		t.Fatalf("unexpected menu contract projection: %#v", menu)
	}
	if !equalTrimmedStringsForTest(menu.DefaultRoles, []string{"base.user"}) {
		t.Fatalf("unexpected menu defaultRoles: %#v", menu.DefaultRoles)
	}
	if !equalTrimmedStringsForTest(menu.Requires, []string{"rpc:/auth.User/*"}) {
		t.Fatalf("unexpected menu requires: %#v", menu.Requires)
	}
}

func TestCollectUiResourceDeclsReportsLiteralValidationIssues(t *testing.T) {
	sourcePath := filepath.Join(t.TempDir(), "invalid_ui_resources.ts")
	source := `
const dynamicAction = 'auth.action.dynamic'
const someModel = 'auth.User'
const roleList = ['admin']
const entityTitle = 'User'
const modelRef = 'auth.User'

defineRoute('auth.route.invalid', {
	sequence: -3,
	actions: ['auth.action.good', dynamicAction],
	requires: [{ model: someModel }],
	defaultRoles: roleList,
	meta: { pageTitle: 'Fallback title' },
})

defineModelActions(modelRef, {})
defineModelActions('authonly', {})
defineModelActions('.User', {})
defineModelActions('auth.User', {
	entityTitle: entityTitle,
	titles: ['bad'],
})
`

	decls, issues := collectUiResourceDecls(sourcePath, source)
	if len(decls) != 5 {
		t.Fatalf("collectUiResourceDecls() decl count = %d, want 5", len(decls))
	}

	var invalidRoute *parser.UiResourceDecl
	for _, decl := range decls {
		if decl.ID == "auth.route.invalid" {
			invalidRoute = decl
			break
		}
	}
	if invalidRoute == nil {
		t.Fatalf("expected invalid route declaration, got %#v", decls)
	}
	if invalidRoute.Sequence != -3 || invalidRoute.Title != "Fallback title" {
		t.Fatalf("unexpected invalid route fields: %#v", invalidRoute)
	}

	issueCodes := make(map[parser.UiResourceIssueCode]int, len(issues))
	for _, issue := range issues {
		issueCodes[issue.Code]++
	}
	if issueCodes[parser.UiResourceIssueCodeRouteActionEntryNotLiteral] != 1 {
		t.Fatalf("expected route action entry fatal issue, got %#v", issues)
	}
	if issueCodes[parser.UiResourceIssueCodeDeclRequiresNotLiteral] != 1 {
		t.Fatalf("expected requires fatal issue, got %#v", issues)
	}
	if issueCodes[parser.UiResourceIssueCodeDeclDefaultRolesNotLiteral] != 1 {
		t.Fatalf("expected defaultRoles fatal issue, got %#v", issues)
	}
	if issueCodes[parser.UiResourceIssueCodeModelIDNotLiteral] != 1 {
		t.Fatalf("expected model id literal issue, got %#v", issues)
	}
	if issueCodes[parser.UiResourceIssueCodeModelIDInvalidFormat] != 1 {
		t.Fatalf("expected model id format issue, got %#v", issues)
	}
	if issueCodes[parser.UiResourceIssueCodeModelIDEmptySegment] != 1 {
		t.Fatalf("expected model id empty segment issue, got %#v", issues)
	}
	if issueCodes[parser.UiResourceIssueCodeModelActionEntityTitleNotLiteral] != 1 {
		t.Fatalf("expected entityTitle literal issue, got %#v", issues)
	}
	if issueCodes[parser.UiResourceIssueCodeModelActionTitlesInvalid] != 1 {
		t.Fatalf("expected titles invalid issue, got %#v", issues)
	}
}

func TestUiResourceHelperFunctions(t *testing.T) {
	if titles, _, err := parseStrictModelActionTitlesProperty(`titles: { create: 'Create', edit: "Edit" }`, "base", nil); err != nil || titles["create"] != "Create" || titles["edit"] != "Edit" {
		t.Fatalf("parseStrictModelActionTitlesProperty() = %#v, %v", titles, err)
	}
	if _, _, err := parseStrictModelActionTitlesProperty(`titles: { create: foo }`, "base", nil); err == nil {
		t.Fatal("expected parseStrictModelActionTitlesProperty() to reject non-literal value")
	}

	if values, err := parseStrictStringArrayProperty(`actions: ['a', "b"]`); err != nil || len(values) != 2 || values[0] != "a" || values[1] != "b" {
		t.Fatalf("parseStrictStringArrayProperty() = %#v, %v", values, err)
	}
	if values, err := parseStrictStringArrayProperty(`actions: ['a', ' a ', 'b', '']`); err != nil || len(values) != 2 || values[0] != "a" || values[1] != "b" {
		t.Fatalf("parseStrictStringArrayProperty() normalized = %#v, %v", values, err)
	}
	if _, err := parseStrictStringArrayProperty(`actions: ['a', dynamicAction]`); err == nil {
		t.Fatal("expected parseStrictStringArrayProperty() to reject dynamic entries")
	}

	if requires, err := parseStrictRequiresProperty(`requires: [{ model: 'auth.User' }, { model: 'auth.User', method: 'Browse' }]`); err != nil || len(requires) != 2 || requires[0] != "rpc:/auth.User/*" || requires[1] != "rpc:/auth.User/Browse" {
		t.Fatalf("parseStrictRequiresProperty() = %#v, %v", requires, err)
	}
	if requires, err := parseStrictRequiresProperty(`requires: [{ model: 'auth.User' }, { model: 'auth.User' }, { model: 'auth.User', method: 'Browse' }]`); err != nil || len(requires) != 2 || requires[0] != "rpc:/auth.User/*" || requires[1] != "rpc:/auth.User/Browse" {
		t.Fatalf("parseStrictRequiresProperty() dedupe = %#v, %v", requires, err)
	}
	if _, err := parseStrictRequiresProperty(`requires: [{ model: ref }]`); err == nil {
		t.Fatal("expected parseStrictRequiresProperty() to reject non-literal model")
	}

	parts, err := splitTopLevelJSList(`{ model: 'auth.User', method: 'Browse' }, ['x', 'y'], 'tail'`)
	if err != nil || len(parts) != 3 {
		t.Fatalf("splitTopLevelJSList() = %#v, %v", parts, err)
	}
	if _, err := splitTopLevelJSList(`'unterminated`); err == nil {
		t.Fatal("expected splitTopLevelJSList() to reject unterminated strings")
	}

	if got := parseFirstQuotedString(`title: "Users"`); got != "Users" {
		t.Fatalf("parseFirstQuotedString() = %q", got)
	}
	quoted := parseQuotedStrings(`['alpha', "beta"]`)
	if len(quoted) != 2 || quoted[0] != "alpha" || quoted[1] != "beta" {
		t.Fatalf("parseQuotedStrings() = %#v", quoted)
	}
	if value, ok := parseNumericFromText(`sequence: -12`); !ok || value != -12 {
		t.Fatalf("parseNumericFromText() = %d, %v", value, ok)
	}
	if value, err := parseJSStringLiteral(`'line\nfeed'`); err != nil || value != "line\nfeed" {
		t.Fatalf("parseJSStringLiteral() = %q, %v", value, err)
	}
	if got := parseNamedQuotedString(`meta: { pageTitle: 'Users' }`, "pageTitle"); got != "Users" {
		t.Fatalf("parseNamedQuotedString() = %q", got)
	}

	if got := resolveModelActionTitle("delete", "User", map[string]string{}); got != "Delete User" {
		t.Fatalf("resolveModelActionTitle() = %q", got)
	}
	if got := resolveModelActionTitle("edit", "User", map[string]string{"edit": "Edit User"}); got != "Edit User" {
		t.Fatalf("resolveModelActionTitle() override = %q", got)
	}
	if requires := requiresForModelAction("auth", "User", "copy"); requires != nil {
		t.Fatalf("requiresForModelAction(copy) = %#v, want nil", requires)
	}
	if requires := requiresForModelAction("auth", "User", "create"); len(requires) != 1 || requires[0] != "rpc:/auth.User/Create" {
		t.Fatalf("requiresForModelAction(create) = %#v", requires)
	}

	if got := uiTypeByFactory("defineMenu"); got != parser.UiResourceTypeMenu {
		t.Fatalf("uiTypeByFactory() = %q", got)
	}
	if !recommendedSegment("user_list") || recommendedSegment("UserList") {
		t.Fatalf("recommendedSegment() returned unexpected values")
	}
	if issue := buildUiResourceIDNamingIssue("defineRoute", "auth.route.user_list", "ui.ts", 1, 1); issue != nil {
		t.Fatalf("expected recommended route id to have no warning, got %#v", issue)
	}
	if issue := buildUiResourceIDNamingIssue("defineAction", "Auth.Action.BadCase", "ui.ts", 2, 3); issue == nil || issue.Code != parser.UiResourceIssueCodeDeclIDNamingSuggested {
		t.Fatalf("expected naming warning, got %#v", issue)
	}

	if line, column := lineColumnForOffset("a\nxyz", 4); line != 2 || column != 3 {
		t.Fatalf("lineColumnForOffset() = %d:%d, want 2:3", line, column)
	}
	if !isASCIISpace('\n') || isASCIISpace('x') {
		t.Fatalf("isASCIISpace() returned unexpected values")
	}
}

func equalTrimmedStringsForTest(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if strings.TrimSpace(a[i]) != strings.TrimSpace(b[i]) {
			return false
		}
	}
	return true
}

func TestUiParseCtxNodeHelpers(t *testing.T) {
	source := `
const literalID = 'auth.route.sample'
const shorthand = 'ignored'
const direct = {
	"title": 'Hello',
	shorthand,
	exclude: ['edit', 'delete'],
}
const wrapped = ({ path: '/users' })
const ignored = helper({ title: 'Ignored' })
`

	ctx := newUIParseCtxForTest(t, source)
	stringNode := findFirstNodeInSourceFile(ctx.sourceFile, tsast.KindStringLiteral)
	if got := ctx.parseStringLiteral(stringNode); got != "auth.route.sample" {
		t.Fatalf("parseStringLiteral() = %q", got)
	}

	objectLiterals := findNodesInSourceFileByKind(ctx.sourceFile, tsast.KindObjectLiteralExpression)
	if len(objectLiterals) < 3 {
		t.Fatalf("expected at least 3 object literals, got %d", len(objectLiterals))
	}
	directObject := objectLiterals[0]
	wrappedObject := objectLiterals[1]

	var titleProperty *tsast.Node
	var shorthandProperty *tsast.Node
	for _, property := range directObject.AsObjectLiteralExpression().Properties.Nodes {
		switch ctx.literalPropertyName(property) {
		case "title":
			titleProperty = property
		case "shorthand":
			shorthandProperty = property
		}
	}
	if titleProperty == nil || shorthandProperty == nil {
		t.Fatalf("expected title and shorthand properties, got %#v", directObject.AsObjectLiteralExpression().Properties.Nodes)
	}
	if got := ctx.literalPropertyName(titleProperty); got != "title" {
		t.Fatalf("literalPropertyName(title) = %q", got)
	}
	if got := ctx.literalPropertyName(shorthandProperty); got != "shorthand" {
		t.Fatalf("literalPropertyName(shorthand) = %q", got)
	}
	if excludes := ctx.parseExcludeMap(directObject); !excludes["edit"] || !excludes["delete"] {
		t.Fatalf("parseExcludeMap() = %#v", excludes)
	}
	if got := ctx.nodeText(titleProperty); got == "" {
		t.Fatal("nodeText() returned empty text for title property")
	}

	if got := ctx.toObjectLiteral(directObject); got != directObject {
		t.Fatalf("toObjectLiteral(direct) = %#v, want same node", got)
	}
	if got := ctx.toObjectLiteral(wrappedObject.Parent); got != wrappedObject {
		t.Fatalf("toObjectLiteral(wrapped parent) = %#v, want wrapped object", got)
	}
	callNode := findFirstNodeInSourceFile(ctx.sourceFile, tsast.KindCallExpression)
	if got := ctx.toObjectLiteral(callNode); got != nil {
		t.Fatalf("toObjectLiteral(call) = %#v, want nil", got)
	}
	if ctx.nodeText(&tsast.Node{}) != "" {
		t.Fatal("nodeText() should return empty for invalid positions")
	}
	if line, column := lineColumnForOffset(source, -1); line != 1 || column != 1 {
		t.Fatalf("lineColumnForOffset(-1) = %d:%d", line, column)
	}
}

func newUIParseCtxForTest(t *testing.T, source string) *uiParseCtx {
	t.Helper()
	path := filepath.Join(t.TempDir(), "sample.ts")
	sourceFile := tsparser.ParseSourceFile(tsast.SourceFileParseOptions{
		FileName: path,
		Path:     tspath.ToPath(path, "", true),
	}, source, tscore.ScriptKindTS)
	return &uiParseCtx{
		sourcePath: path,
		source:     source,
		sourceFile: sourceFile,
	}
}

func findFirstNodeInSourceFile(sourceFile *tsast.SourceFile, kind tsast.Kind) *tsast.Node {
	for _, stmt := range sourceFile.Statements.Nodes {
		if found := findFirstNodeByKind(stmt, kind); found != nil {
			return found
		}
	}
	return nil
}

func findNodesInSourceFileByKind(sourceFile *tsast.SourceFile, kind tsast.Kind) []*tsast.Node {
	result := make([]*tsast.Node, 0)
	for _, stmt := range sourceFile.Statements.Nodes {
		appendNodesByKind(stmt, kind, &result)
	}
	return result
}

func findFirstNodeByKind(node *tsast.Node, kind tsast.Kind) *tsast.Node {
	if node == nil {
		return nil
	}
	if node.Kind == kind {
		return node
	}
	var found *tsast.Node
	node.ForEachChild(func(child *tsast.Node) bool {
		found = findFirstNodeByKind(child, kind)
		return found != nil
	})
	return found
}

func appendNodesByKind(node *tsast.Node, kind tsast.Kind, result *[]*tsast.Node) {
	if node == nil {
		return
	}
	if node.Kind == kind {
		*result = append(*result, node)
	}
	node.ForEachChild(func(child *tsast.Node) bool {
		appendNodesByKind(child, kind, result)
		return false
	})
}
