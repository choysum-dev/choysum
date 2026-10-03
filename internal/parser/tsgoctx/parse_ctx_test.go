// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package tsgoctx

import (
	"path/filepath"
	"testing"

	tsast "github.com/buke/typescript-go-internal/v7/pkg/ast"
	tscore "github.com/buke/typescript-go-internal/v7/pkg/core"
	"github.com/choysum-dev/choysum/internal/parser"
)

func TestParseTSGoCtxParsesImportsAndExports(t *testing.T) {
	path := filepath.Join(t.TempDir(), "modules", "auth", "web", "views", "sample.ts")
	content := `
import DefaultView, { named as alias, plain } from './base.ts'
import * as toolkit from '@/shared/toolkit.ts'

export { alias as exportedAlias }
export * from './wild.ts'
export * from '@/shared/extra.ts'
export * as namespaceExport from './namespaced.ts'
export default toolkit.Widget
export const ready = true
`

	ctx, err := Parse(map[string]string{"@": filepath.Join(filepath.Dir(filepath.Dir(filepath.Dir(path))), "")}, path, content)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if got := ctx.CurrentModuleSpecPath(); got != path[:len(path)-len(filepath.Ext(path))] {
		t.Fatalf("CurrentModuleSpecPath() = %q", got)
	}
	if ctx.Imports["DefaultView"] == nil || ctx.Imports["DefaultView"].ReferenceIdent != "default" {
		t.Fatalf("unexpected default import: %#v", ctx.Imports["DefaultView"])
	}
	if ctx.Imports["DefaultView"].IsTypeOnly {
		t.Fatalf("expected value default import, IsTypeOnly=true")
	}
	if ctx.Imports["alias"] == nil || ctx.Imports["alias"].ReferenceIdent != "named" {
		t.Fatalf("unexpected aliased import: %#v", ctx.Imports["alias"])
	}
	if ctx.Imports["plain"] == nil || ctx.Imports["plain"].ReferenceIdent != "plain" {
		t.Fatalf("unexpected named import: %#v", ctx.Imports["plain"])
	}
	if ctx.Imports["toolkit"] == nil || ctx.Imports["toolkit"].ReferenceIdent != "*" {
		t.Fatalf("unexpected namespace import: %#v", ctx.Imports["toolkit"])
	}

	if ctx.Exports["exportedAlias"] == nil || ctx.Exports["exportedAlias"].ReferenceIdent != "named" {
		t.Fatalf("unexpected re-export alias: %#v", ctx.Exports["exportedAlias"])
	}
	if ctx.Exports["default"] == nil || ctx.Exports["default"].ReferenceIdent != "Widget" {
		t.Fatalf("unexpected default property-access export: %#v", ctx.Exports["default"])
	}
	if ctx.Exports["default"].ModuleSpecPath != ctx.Imports["toolkit"].ModuleSpecPath {
		t.Fatalf("unexpected default export module spec: %#v", ctx.Exports["default"])
	}
	if ctx.Exports["ready"] == nil || ctx.Exports["ready"].ReferenceIdent != "ready" {
		t.Fatalf("unexpected named export: %#v", ctx.Exports["ready"])
	}
	if ctx.Exports["namespaceExport"] == nil {
		t.Fatalf("expected namespace export declaration")
	}
	if wildcard := ctx.Exports["*"]; wildcard == nil || len(wildcard.Wildcard) != 2 {
		t.Fatalf("unexpected wildcard exports: %#v", wildcard)
	}
}

func TestParseTSGoCtxExportsDestructuredBindings(t *testing.T) {
	path := filepath.Join(t.TempDir(), "modules", "web", "web", "components", "vendor", "ui", "sidebar", "utils.ts")
	content := `
export const [useSidebar, provideSidebarContext] = [null, null]
export const { useCommand, provideCommandContext } = { useCommand: null, provideCommandContext: null }
export const [keep, , also] = [1, 2, 3]
export const { nested: { deep } } = { nested: { deep: 1 } }
export function namedFn() {}
export {}
`
	ctx, err := Parse(nil, path, content)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	for _, name := range []string{
		"useSidebar", "provideSidebarContext", "useCommand", "provideCommandContext",
		"keep", "also", "deep", "namedFn",
	} {
		if ctx.Exports[name] == nil {
			t.Fatalf("expected destructured/named export %q, got %#v", name, ctx.Exports)
		}
	}
	if got := ExportDeclarationName(nil); got != "" {
		t.Fatalf("ExportDeclarationName(nil) = %q, want empty", got)
	}
	if got := ExportDeclarationNames(nil); got != nil {
		t.Fatalf("ExportDeclarationNames(nil) = %#v, want nil", got)
	}

	// Cover Name()-based lookup, non-variable exports, holes, and nested patterns.
	var exportEmpty *tsast.Node
	var exportFn *tsast.Node
	var exportArrayHole *tsast.Node
	var exportNested *tsast.Node
	for _, stmt := range ctx.Source.Statements.Nodes {
		if stmt == nil {
			continue
		}
		switch stmt.Kind {
		case tsast.KindExportDeclaration:
			exportEmpty = stmt
		case tsast.KindFunctionDeclaration:
			if HasModifier(stmt, tsast.KindExportKeyword) {
				exportFn = stmt
			}
		case tsast.KindVariableStatement:
			names := ExportDeclarationNames(stmt)
			if len(names) == 2 && names[0] == "keep" && names[1] == "also" {
				exportArrayHole = stmt
			}
			if len(names) == 1 && names[0] == "deep" {
				exportNested = stmt
			}
		}
	}
	if exportEmpty == nil || exportFn == nil || exportArrayHole == nil || exportNested == nil {
		t.Fatalf("expected fixture statements, got empty=%v fn=%v hole=%v nested=%v",
			exportEmpty != nil, exportFn != nil, exportArrayHole != nil, exportNested != nil)
	}
	if got := ExportDeclarationNames(exportEmpty); got != nil {
		t.Fatalf("ExportDeclarationNames(export {}) = %#v, want nil", got)
	}
	if got := ExportDeclarationName(exportFn); got != "namedFn" {
		t.Fatalf("ExportDeclarationName(function) = %q, want namedFn", got)
	}
	if got := ExportDeclarationNames(exportArrayHole); len(got) != 2 || got[0] != "keep" || got[1] != "also" {
		t.Fatalf("ExportDeclarationNames(array holes) = %#v", got)
	}
	if got := ExportDeclarationNames(exportNested); len(got) != 1 || got[0] != "deep" {
		t.Fatalf("ExportDeclarationNames(nested) = %#v", got)
	}

	// Direct edge branches that source fixtures cannot easily hit.
	if got := bindingNameIdents(nil); got != nil {
		t.Fatalf("bindingNameIdents(nil) = %#v", got)
	}
	if got := bindingNameIdents(exportFn); got != nil {
		t.Fatalf("bindingNameIdents(non-binding) = %#v", got)
	}
	factory := tsast.NewNodeFactory(tsast.NodeFactoryHooks{})
	emptyIdent := factory.NewIdentifier("")
	if got := bindingNameIdents(emptyIdent); got != nil {
		t.Fatalf("bindingNameIdents(empty ident) = %#v", got)
	}
	emptyPattern := factory.NewBindingPattern(tsast.KindArrayBindingPattern, nil)
	if got := bindingNameIdents(emptyPattern); got != nil {
		t.Fatalf("bindingNameIdents(nil elements) = %#v", got)
	}
	// Nil list entries and non-binding elements (continue branches).
	withHole := factory.NewBindingPattern(tsast.KindArrayBindingPattern, &tsast.NodeList{
		Nodes: []*tsast.Node{
			nil,
			factory.NewOmittedExpression(),
			factory.NewBindingElement(nil, nil, factory.NewIdentifier("kept"), nil),
		},
	})
	if got := bindingNameIdents(withHole); len(got) != 1 || got[0] != "kept" {
		t.Fatalf("bindingNameIdents(nil+omitted+kept) = %#v", got)
	}
	// Nil variable declarations in the list.
	varList := exportArrayHole.AsVariableStatement().DeclarationList.AsVariableDeclarationList()
	orig := varList.Declarations.Nodes
	varList.Declarations.Nodes = append([]*tsast.Node{nil}, orig...)
	if got := ExportDeclarationNames(exportArrayHole); len(got) != 2 {
		t.Fatalf("ExportDeclarationNames(with nil decl) = %#v", got)
	}
	varList.Declarations.Nodes = orig
}

func TestParseTSGoCtxHandlesDefaultDeclarationsAndMergeHelpers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "component.vue")
	content := `
export default class NamedView {}
export const enabled = true
`

	ctx, err := ParseWithKind(nil, path, content, tscore.ScriptKindTS, true)
	if err != nil {
		t.Fatalf("ParseWithKind() error = %v", err)
	}
	if got := ctx.CurrentModuleSpecPath(); got != path {
		t.Fatalf("CurrentModuleSpecPath() for vue = %q, want %q", got, path)
	}
	if ctx.Exports["default"] == nil || ctx.Exports["default"].ReferenceIdent != "NamedView" {
		t.Fatalf("unexpected default declaration export: %#v", ctx.Exports["default"])
	}
	if ctx.Exports["enabled"] == nil {
		t.Fatalf("expected named variable export")
	}

	dstImports := map[string]*parser.Import{"keep": {ReferenceIdent: "keep"}}
	MergeImports(dstImports, map[string]*parser.Import{"next": {ReferenceIdent: "next"}})
	if dstImports["keep"] == nil || dstImports["next"] == nil {
		t.Fatalf("unexpected merged imports: %#v", dstImports)
	}

	dstExports := map[string]*parser.Export{
		"*": {Wildcard: []*parser.Export{{ReferenceIdent: "*", ModuleSpecPath: "one"}}},
	}
	MergeExports(dstExports, map[string]*parser.Export{
		"*": {Wildcard: []*parser.Export{{ReferenceIdent: "*", ModuleSpecPath: "two"}}},
	})
	if len(dstExports["*"].Wildcard) != 2 {
		t.Fatalf("unexpected merged wildcard exports: %#v", dstExports["*"])
	}
	MergeExports(nil, dstExports)
	MergeImports(nil, dstImports)

	if got := ExportDeclarationName(nil); got != "" {
		t.Fatalf("ExportDeclarationName(nil) = %q, want empty", got)
	}
}

func TestParseImportIsTypeOnly_ParseCtxPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "modules", "auth", "service", "user.ts")
	content := `
import type TypeOnlyDefault from '@/base/service/models/language'
import type { TypeNamed } from '@/meta/service/models/model'
import { type InlineType, InlineValue } from '@/document/service/models/attachment_object'
import ValueDefault from '@/partner/service/models/partner'
`
	ctx, err := Parse(map[string]string{"@": filepath.Join(filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(path)))), "")}, path, content)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	assertTypeOnly := func(local string, want bool) {
		imp := ctx.Imports[local]
		if imp == nil {
			t.Fatalf("missing import binding %q", local)
		}
		if imp.IsTypeOnly != want {
			t.Fatalf("Imports[%q].IsTypeOnly=%v want %v", local, imp.IsTypeOnly, want)
		}
	}

	assertTypeOnly("TypeOnlyDefault", true)
	assertTypeOnly("TypeNamed", true)
	assertTypeOnly("InlineType", true)
	assertTypeOnly("InlineValue", false)
	assertTypeOnly("ValueDefault", false)
}

func TestParseSideEffectAndExportPaths_ParseCtxPath(t *testing.T) {
	modulesPath := filepath.Join(t.TempDir(), "modules")
	path := filepath.Join(modulesPath, "partner", "service", "models", "partner.ts")
	content := `
import '@/auth/service/models/role';
import {} from '@/core/service/api/dial';
export type { Role } from '@/auth/service/models/role';
export * as auth from '@/auth/service/models/role';
export default local;
const local = {};
`
	ctx, err := Parse(map[string]string{"@/*": filepath.Join(modulesPath, "*")}, path, content)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	var sideEffect *parser.Import
	for key, imp := range ctx.Imports {
		if parser.IsSideEffectImportMapKey(key) {
			sideEffect = imp
			break
		}
	}
	if sideEffect == nil || sideEffect.IsTypeOnly {
		t.Fatalf("side-effect import = %#v", sideEffect)
	}
	if exp := ctx.Exports["Role"]; exp == nil || !exp.IsTypeOnly {
		t.Fatalf("export type Role = %#v", exp)
	}
	if exp := ctx.Exports["auth"]; exp == nil || exp.IsTypeOnly {
		t.Fatalf("namespace export auth = %#v", exp)
	}
	if exp := ctx.Exports["default"]; exp == nil || exp.IsTypeOnly {
		t.Fatalf("default export = %#v", exp)
	}
}

func TestParseExportAssignmentFallback_ParseCtxPath(t *testing.T) {
	modulesPath := filepath.Join(t.TempDir(), "modules")
	path := filepath.Join(modulesPath, "partner", "service", "models", "partner.ts")
	content := `export default (1 as unknown);`
	ctx, err := Parse(map[string]string{"@/*": filepath.Join(modulesPath, "*")}, path, content)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if exp := ctx.Exports["default"]; exp == nil {
		t.Fatal("expected default export")
	}
}

func TestParseExportTypeWildcard_ParseCtxPath(t *testing.T) {
	modulesPath := filepath.Join(t.TempDir(), "modules")
	path := filepath.Join(modulesPath, "partner", "service", "models", "partner.ts")
	content := `export type * from '@/auth/service/models/role';`
	ctx, err := Parse(map[string]string{"@/*": filepath.Join(modulesPath, "*")}, path, content)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	wildcard := ctx.Exports["*"]
	if wildcard == nil || len(wildcard.Wildcard) != 1 {
		t.Fatalf("wildcard export = %#v", wildcard)
	}
	if !wildcard.IsTypeOnly || !wildcard.Wildcard[0].IsTypeOnly {
		t.Fatalf("export type wildcard IsTypeOnly=false: %#v", wildcard)
	}
}

func TestTSGoCtxConvertReferenceWithModuleSpecAndNodeTextGuards(t *testing.T) {
	path := filepath.Join(t.TempDir(), "views", "child.ts")
	content := `
import DefaultView from './base.ts'

const localView = DefaultView

export default localView
`

	ctx, err := Parse(nil, path, content)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if moduleSpec, referenceIdent := ctx.ConvertReferenceWithModuleSpec("DefaultView"); referenceIdent != "default" || moduleSpec != filepath.Join(filepath.Dir(path), "base") {
		t.Fatalf("unexpected imported reference conversion: %q %q", moduleSpec, referenceIdent)
	}
	if moduleSpec, referenceIdent := ctx.ConvertReferenceWithModuleSpec("localView"); referenceIdent != "default" || moduleSpec != ctx.CurrentModuleSpecPath() {
		t.Fatalf("unexpected default export conversion: %q %q", moduleSpec, referenceIdent)
	}
	if moduleSpec, referenceIdent := ctx.ConvertReferenceWithModuleSpec("missingRef"); referenceIdent != "missingRef" || moduleSpec != ctx.CurrentModuleSpecPath() {
		t.Fatalf("unexpected fallback conversion: %q %q", moduleSpec, referenceIdent)
	}

	identifierNode := findFirstNodeInSourceFile(ctx.Source, tsast.KindIdentifier)
	if identifierNode == nil || ctx.NodeText(identifierNode) == "" {
		t.Fatalf("expected NodeText() to return source text for valid identifier node")
	}
	if got := ctx.NodeText(&tsast.Node{}); got != "" {
		t.Fatalf("NodeText(invalid) = %q, want empty", got)
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
