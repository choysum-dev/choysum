// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package parser

import (
	"context"
	"path/filepath"
	"testing"

	tsast "github.com/buke/typescript-go-internal/pkg/ast"
)

func mustParseTSGoCtx(t *testing.T, path string, content string) (*TsParser, *tsgoImportExportCtx) {
	t.Helper()
	parser := &TsParser{Path: path, Content: content, PathAlias: map[string]string{"@/*": "/virtual/modules/test/*"}}
	ctx, err := parser.parseTSGoImportExportCtx()
	if err != nil {
		t.Fatalf("parseTSGoImportExportCtx() error = %v", err)
	}
	parser.ImportsMap = ctx.imports
	parser.ExportsMap = ctx.exports
	return parser, ctx
}

func findFirstStatementByKind(ctx *tsgoImportExportCtx, kind tsast.Kind) *tsast.Node {
	if ctx == nil || ctx.source == nil {
		return nil
	}
	for _, stmt := range ctx.source.Statements.Nodes {
		if stmt != nil && stmt.Kind == kind {
			return stmt
		}
	}
	return nil
}

func findClassMemberByKind(classNode *tsast.Node, kind tsast.Kind, name string) *tsast.Node {
	if classNode == nil || classNode.Kind != tsast.KindClassDeclaration {
		return nil
	}
	classDecl := classNode.AsClassDeclaration()
	if classDecl.Members == nil {
		return nil
	}
	for _, member := range classDecl.Members.Nodes {
		if member == nil || member.Kind != kind {
			continue
		}
		if name == "" {
			return member
		}
		if got := tsgoPropertyName(nil, member.Name()); got == name {
			return member
		}
		if member.Name() != nil && member.Name().Text() == name {
			return member
		}
	}
	return nil
}

func TestParseClassNode_ResolvesExtendsWithoutPreparse(t *testing.T) {
	p := &TsParser{
		Path:      "/virtual/modules/test/service/user.ts",
		Content:   "import BaseModel from './base';\nexport default class User extends BaseModel {}\n",
		PathAlias: map[string]string{},
	}

	class, err := p.ParseClassNode(nil, nil)
	if err != nil {
		t.Fatalf("parse class failed: %v", err)
	}
	if class == nil {
		t.Fatalf("expected class")
	}
	if class.Name != "User" {
		t.Fatalf("unexpected class name: %s", class.Name)
	}
	if class.Extends == nil {
		t.Fatalf("expected extends info")
	}
	if class.Extends.ReferenceIdent != "default" {
		t.Fatalf("expected extends reference default, got %s", class.Extends.ReferenceIdent)
	}
	if class.Extends.ModuleSpecPath != "/virtual/modules/test/service/base" {
		t.Fatalf("unexpected extends module spec path: %s", class.Extends.ModuleSpecPath)
	}
}

func TestParseClassNode_DefaultExportAssignmentClass(t *testing.T) {
	p := &TsParser{
		Path: "/virtual/modules/test/service/user_assignment.ts",
		Content: "import BaseModel from './base';\n" +
			"class User extends BaseModel {}\n" +
			"export default User;\n",
		PathAlias: map[string]string{},
	}

	class, err := p.ParseClassNode(nil, nil)
	if err != nil {
		t.Fatalf("parse class failed: %v", err)
	}
	if class == nil {
		t.Fatalf("expected class")
	}
	if class.Name != "User" {
		t.Fatalf("unexpected class name: %s", class.Name)
	}
	if class.Extends == nil {
		t.Fatalf("expected extends info")
	}
	if class.Extends.ReferenceIdent != "default" {
		t.Fatalf("expected extends reference default, got %s", class.Extends.ReferenceIdent)
	}
}

func TestParseClassNode_NonDefaultExportClassReturnsNil(t *testing.T) {
	p := &TsParser{
		Path:      "/virtual/modules/test/service/compiler.ts",
		Content:   "import { PostgresQueryCompiler } from 'kysely';\nexport class ChoysumPostgresQueryCompiler extends PostgresQueryCompiler {}\n",
		PathAlias: map[string]string{},
	}

	class, err := p.ParseClassNode(nil, nil)
	if err != nil {
		t.Fatalf("parse class failed: %v", err)
	}
	if class != nil {
		t.Fatalf("expected nil class for non-default export, got %+v", class)
	}
}

func TestConvertReferenceWithModuleSpec_FallbackUsesTrimSuffix(t *testing.T) {
	p := &TsParser{
		Path:       "/virtual/modules/test/service/stats.ts",
		ImportsMap: map[string]*Import{},
		ExportsMap: map[string]*Export{},
	}

	moduleSpec, referenceIdent := p.ConvertReferenceWithModuleSpec("UnknownRef")
	if moduleSpec != "/virtual/modules/test/service/stats" {
		t.Fatalf("unexpected fallback module spec path: %s", moduleSpec)
	}
	if referenceIdent != "UnknownRef" {
		t.Fatalf("unexpected fallback reference ident: %s", referenceIdent)
	}
}

func TestConvertReferenceWithModuleSpec_DefaultExportAlias(t *testing.T) {
	p := &TsParser{
		Path:       "/virtual/modules/test/service/user.ts",
		ImportsMap: map[string]*Import{},
		ExportsMap: map[string]*Export{
			"default": {
				ReferenceIdent: "User",
				ModuleSpecPath: "/virtual/modules/test/service/user",
			},
		},
	}

	moduleSpec, referenceIdent := p.ConvertReferenceWithModuleSpec("User")
	if moduleSpec != "/virtual/modules/test/service/user" {
		t.Fatalf("unexpected module spec path: %s", moduleSpec)
	}
	if referenceIdent != "default" {
		t.Fatalf("expected default reference ident, got %s", referenceIdent)
	}
}

func TestParseTSTreeAndTSGoHelperFunctions(t *testing.T) {
	relPath := filepath.Join("internal", "parser", "sample.ts")
	tree, err := ParseTSTree(context.Background(), relPath, "export const named = 1\n")
	if err != nil {
		t.Fatalf("ParseTSTree() error = %v", err)
	}
	if tree == nil || tree.Text() == "" {
		t.Fatalf("expected parsed tree with source text, got %#v", tree)
	}

	parser, ctx := mustParseTSGoCtx(t, relPath, "export const named = 1\nexport default class User {}\n")
	if got := ctx.currentModuleSpecPath(); filepath.Base(got) != "sample" {
		t.Fatalf("currentModuleSpecPath() = %q, want basename sample", got)
	}
	if got := (&tsgoImportExportCtx{path: "/tmp/sample.vue"}).currentModuleSpecPath(); got != "/tmp/sample.vue" {
		t.Fatalf("currentModuleSpecPath(non-ts) = %q, want /tmp/sample.vue", got)
	}
	if got := ctx.resolveModuleSpec("@/shared/types.ts"); got != "/virtual/modules/test/shared/types" {
		t.Fatalf("resolveModuleSpec(alias) = %q", got)
	}
	if got := ctx.nodeText(nil); got != "" {
		t.Fatalf("nodeText(nil) = %q, want empty", got)
	}
	if line, col := ctx.lineColumn(0); line != 1 || col != 1 {
		t.Fatalf("lineColumn(0) = (%d,%d), want (1,1)", line, col)
	}

	varStmt := findFirstStatementByKind(ctx, tsast.KindVariableStatement)
	if varStmt == nil {
		t.Fatal("expected variable statement")
	}
	if got := ctx.nodeText(varStmt); got == "" {
		t.Fatal("expected nodeText(variable) to be non-empty")
	}
	if got := tsgoExportDeclarationName(varStmt); got != "named" {
		t.Fatalf("tsgoExportDeclarationName(variable) = %q, want named", got)
	}
	if !tsgoHasModifier(varStmt, tsast.KindExportKeyword) {
		t.Fatal("expected export modifier on variable statement")
	}
	if tsgoHasModifier(nil, tsast.KindExportKeyword) {
		t.Fatal("expected nil node to have no modifiers")
	}

	classStmt := findFirstStatementByKind(ctx, tsast.KindClassDeclaration)
	if classStmt == nil {
		t.Fatal("expected class declaration")
	}
	if !tsgoIsExportDefaultDeclaration(classStmt) {
		t.Fatal("expected class declaration to be export default")
	}
	if !tsgoIsExportDeclaration(classStmt) {
		t.Fatal("expected class declaration to be export declaration")
	}
	if got := tsgoExportDeclarationName(classStmt); got != "User" {
		t.Fatalf("tsgoExportDeclarationName(class) = %q, want User", got)
	}

	text, start, end, line, column := tsgoNodeStableInfo(ctx, classStmt)
	if text == "" || start >= end || line == 0 || column == 0 {
		t.Fatalf("unexpected stable info: text=%q start=%d end=%d line=%d column=%d", text, start, end, line, column)
	}
	if text, start, end, line, column := tsgoNodeStableInfo(nil, nil); text != "" || start != 0 || end != 0 || line != 0 || column != 0 {
		t.Fatalf("expected zero stable info for nil input, got %q %d %d %d %d", text, start, end, line, column)
	}

	decls := varStmt.AsVariableStatement().DeclarationList.AsVariableDeclarationList().Declarations.Nodes
	if len(decls) == 0 || decls[0] == nil {
		t.Fatal("expected variable declaration")
	}
	nameNode := decls[0].AsVariableDeclaration().Name()
	if got := tsgoPropertyName(ctx, nameNode); got != "named" {
		t.Fatalf("tsgoPropertyName(identifier) = %q, want named", got)
	}
	if got := tsgoPropertyName(nil, nameNode); got != "" {
		t.Fatalf("tsgoPropertyName(nil ctx) = %q, want empty", got)
	}

	defaultModule, defaultRef := parser.ConvertReferenceWithModuleSpec("User")
	if filepath.Base(defaultModule) != "sample" || defaultRef != "default" {
		t.Fatalf("ConvertReferenceWithModuleSpec(default alias) = (%q,%q)", defaultModule, defaultRef)
	}
}

func TestTSGoParseDecoratorObjectAndClassMembers(t *testing.T) {
	content := "import BaseModel from './base'\n" +
		"import * as decorators from '@/decorators'\n" +
		"import { Field, Column } from './field'\n" +
		"const alias = 'runtime'\n" +
		"@decorators.Model({ table: 'users', flags: [1, true, null, { deep: 'v' }], alias })\n" +
		"export default abstract class User extends decorators.Entity {\n" +
		"  @Field({ label: 'Name', tags: ['a', 2] })\n" +
		"  readonly name: string\n" +
		"  @Column()\n" +
		"  #meta: { enabled: true }\n" +
		"  values: number[]\n" +
		"  async save<T extends BaseModel>(this: User, payload: T): Promise<void> { return }\n" +
		"  syncMethod(): void {}\n" +
		"}\n"

	parser, ctx := mustParseTSGoCtx(t, "/virtual/modules/test/service/user.ts", content)
	classNode := parser.tsgoFindClassNode(ctx, "")
	if classNode == nil {
		t.Fatal("expected class node")
	}

	if got := tsgoNodeTypeName(nil); got != "" {
		t.Fatalf("tsgoNodeTypeName(nil) = %q, want empty", got)
	}
	if got := tsgoNodeTypeName(classNode); got == "" {
		t.Fatal("expected class node type name to be non-empty")
	}

	decoratorNode := classNode.Decorators()[0]
	decorator, err := parser.tsgoParseDecorator(ctx, decoratorNode)
	if err != nil {
		t.Fatalf("tsgoParseDecorator() error = %v", err)
	}
	if decorator == nil || decorator.Name != "Model" || decorator.ReferenceIdent != "" {
		t.Fatalf("unexpected class decorator: %#v", decorator)
	}
	if decorator.ModuleSpecPath != "/virtual/modules/test/decorators" {
		t.Fatalf("unexpected decorator module path: %q", decorator.ModuleSpecPath)
	}
	if len(decorator.Arguments) != 1 {
		t.Fatalf("expected one decorator argument, got %#v", decorator.Arguments)
	}
	if decorator.Arguments[0].Type != "ObjectLiteral" || len(decorator.Arguments[0].ObjectProperties) != 2 {
		t.Fatalf("unexpected decorator argument payload: %#v", decorator.Arguments[0])
	}

	classInfo, err := parser.ParseClassNode(nil, nil)
	if err != nil {
		t.Fatalf("ParseClassNode() error = %v", err)
	}
	if classInfo == nil || classInfo.Name != "User" || !classInfo.Abstract {
		t.Fatalf("unexpected class info: %#v", classInfo)
	}
	if classInfo.Extends == nil || classInfo.Extends.Name != "Entity" || classInfo.Extends.ReferenceIdent != "Entity" || classInfo.Extends.ModuleSpecPath != "/virtual/modules/test/decorators" {
		t.Fatalf("unexpected extends info: %#v", classInfo.Extends)
	}
	if len(classInfo.Decorators) != 1 || len(classInfo.MemberVars) != 3 || len(classInfo.MemberMethods) != 2 {
		t.Fatalf("unexpected class members: decorators=%d vars=%d methods=%d", len(classInfo.Decorators), len(classInfo.MemberVars), len(classInfo.MemberMethods))
	}

	var nameVar, metaVar, valuesVar *MemberVar
	for _, memberVar := range classInfo.MemberVars {
		switch memberVar.Name {
		case "name":
			nameVar = memberVar
		case "#meta":
			metaVar = memberVar
		case "values":
			valuesVar = memberVar
		}
	}
	if nameVar == nil || !nameVar.IsReadonly || nameVar.AccessibilityModifier != "public" || nameVar.TypeAnnotation != "string" {
		t.Fatalf("unexpected name var: %#v", nameVar)
	}
	if len(nameVar.Decorators) != 1 || nameVar.Decorators[0].Name != "Field" || nameVar.Decorators[0].ModuleSpecPath != "/virtual/modules/test/service/field" {
		t.Fatalf("unexpected name var decorators: %#v", nameVar.Decorators)
	}
	if metaVar == nil || metaVar.AccessibilityModifier != "private" || metaVar.TypeAnnotation != "jsonobject" || metaVar.TsTypeReference != "jsonobject" {
		t.Fatalf("unexpected private meta var: %#v", metaVar)
	}
	if valuesVar == nil || valuesVar.TypeAnnotation != "number[]" || valuesVar.TsTypeReference != "number" {
		t.Fatalf("unexpected values var: %#v", valuesVar)
	}

	method := classInfo.MemberMethods[0]
	if method.Name != "save" || method.AccessibilityModifier != "public" || method.TypeAnnotation != "Promise<void>" {
		t.Fatalf("unexpected async method: %#v", method)
	}
	if len(method.Parameters) != 2 || method.Parameters[0].Name != "this" || method.Parameters[1].Name != "payload" {
		t.Fatalf("unexpected method parameters: %#v", method.Parameters)
	}
	if len(method.TypeParameters) != 1 || method.TypeParameters[0].Name != "T" || method.TypeParameters[0].ReferenceIdent != "default" || method.TypeParameters[0].ModuleSpecPath != "/virtual/modules/test/service/base" {
		t.Fatalf("unexpected type parameters: %#v", method.TypeParameters)
	}

	metaNode := findClassMemberByKind(classNode, tsast.KindPropertyDeclaration, "#meta")
	if metaNode == nil {
		t.Fatal("expected #meta property node")
	}
	metaMember, err := parser.tsgoParseMemberVar(ctx, metaNode)
	if err != nil {
		t.Fatalf("tsgoParseMemberVar(#meta) error = %v", err)
	}
	if metaMember == nil || metaMember.Name != "#meta" {
		t.Fatalf("unexpected parsed meta member: %#v", metaMember)
	}

	saveNode := findClassMemberByKind(classNode, tsast.KindMethodDeclaration, "save")
	if saveNode == nil {
		t.Fatal("expected save method node")
	}
	parsedMethod, err := parser.tsgoParseMemberMethod(ctx, saveNode)
	if err != nil {
		t.Fatalf("tsgoParseMemberMethod(save) error = %v", err)
	}
	if parsedMethod == nil || parsedMethod.Name != "save" {
		t.Fatalf("unexpected parsed method: %#v", parsedMethod)
	}
	if syncNode := findClassMemberByKind(classNode, tsast.KindMethodDeclaration, "syncMethod"); syncNode != nil {
		parsedSync, err := parser.tsgoParseMemberMethod(ctx, syncNode)
		if err != nil {
			t.Fatalf("tsgoParseMemberMethod(syncMethod) error = %v", err)
		}
		if parsedSync == nil || parsedSync.Name != "syncMethod" {
			t.Fatalf("expected sync method to be parsed, got %#v", parsedSync)
		}
	}

	if ids := tsgoCollectReferenceIdents(classNode.AsClassDeclaration().HeritageClauses.Nodes[0].AsHeritageClause().Types.Nodes[0].AsExpressionWithTypeArguments().Expression); len(ids) != 2 || ids[0] != "decorators" || ids[1] != "Entity" {
		t.Fatalf("unexpected reference idents: %#v", ids)
	}
	if ids := tsgoCollectReferenceIdents(nil); ids != nil {
		t.Fatalf("expected nil reference idents, got %#v", ids)
	}

	objectNode := decoratorNode.AsDecorator().Expression.AsCallExpression().Arguments.Nodes[0]
	convertedMap, err := parser.tsgoConvertObjectLiteralToMap(ctx, objectNode)
	if err != nil {
		t.Fatalf("tsgoConvertObjectLiteralToMap() error = %v", err)
	}
	if convertedMap["table"] != "users" || convertedMap["alias"] != "alias" {
		t.Fatalf("unexpected converted object literal: %#v", convertedMap)
	}
	flags, ok := convertedMap["flags"].([]interface{})
	if !ok || len(flags) != 4 {
		t.Fatalf("unexpected converted flags: %#v", convertedMap["flags"])
	}
	deep, ok := flags[3].(map[string]interface{})
	if !ok || deep["deep"] != "v" {
		t.Fatalf("unexpected nested object literal: %#v", flags[3])
	}
	properties := parser.tsgoExtractObjectProperties(ctx, objectNode)
	if len(properties) != 2 || properties[0].Name != "table" || properties[1].ValueKind != "ArrayLiteral" {
		t.Fatalf("unexpected extracted object properties: %#v", properties)
	}

	if got, err := parser.tsgoConvertLiteralToInterface(ctx, objectNode.AsObjectLiteralExpression().Properties.Nodes[0].AsPropertyAssignment().Initializer); err != nil || got != "users" {
		t.Fatalf("tsgoConvertLiteralToInterface() = %#v, %v", got, err)
	}
	arrayNode := objectNode.AsObjectLiteralExpression().Properties.Nodes[1].AsPropertyAssignment().Initializer
	if got, err := parser.tsgoConvertArrayLiteralToArray(ctx, arrayNode); err != nil || len(got) != 4 {
		t.Fatalf("tsgoConvertArrayLiteralToArray() = %#v, %v", got, err)
	}

	if got := tsgoNodeTypeName(objectNode); got != "ObjectLiteral" {
		t.Fatalf("tsgoNodeTypeName(object) = %q, want ObjectLiteral", got)
	}
	if got := tsgoNodeTypeName(arrayNode); got != "ArrayLiteral" {
		t.Fatalf("tsgoNodeTypeName(array) = %q, want ArrayLiteral", got)
	}
	if got := tsgoNodeTypeName(objectNode.AsObjectLiteralExpression().Properties.Nodes[2].AsShorthandPropertyAssignment().Name()); got != "IdentExpr" {
		t.Fatalf("tsgoNodeTypeName(identifier) = %q, want IdentExpr", got)
	}
	if !tsgoIsLiteralNode(objectNode.AsObjectLiteralExpression().Properties.Nodes[0].AsPropertyAssignment().Initializer) {
		t.Fatal("expected string literal to be detected as literal")
	}
	if tsgoIsLiteralNode(objectNode) {
		t.Fatal("did not expect object literal to be treated as literal")
	}
}
