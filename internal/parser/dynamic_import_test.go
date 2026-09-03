// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package parser

import (
	"testing"

	tsast "github.com/buke/typescript-go-internal/v7/pkg/ast"
)

func TestParseDynamicImports_TsParserPath(t *testing.T) {
	path := "/virtual/modules/auth/service/tests/observability.test.ts"
	content := `
const mod = await import('@/meta/service/models/ui_resource');
type Company = typeof import('@/base/service/models/company');
const local = import('./local_probe');
`
	_, ctx := mustParseTSGoCtx(t, path, content)

	if len(ctx.dynamicImports) != 2 {
		t.Fatalf("expected 2 dynamic imports, got %d: %#v", len(ctx.dynamicImports), ctx.dynamicImports)
	}

	assertDynamic := func(index int, wantSpec string, wantTypeOnly bool) {
		imp := ctx.dynamicImports[index]
		if imp == nil {
			t.Fatalf("dynamicImports[%d] is nil", index)
		}
		if !imp.IsDynamic {
			t.Fatalf("dynamicImports[%d].IsDynamic=false", index)
		}
		if imp.IsTypeOnly != wantTypeOnly {
			t.Fatalf("dynamicImports[%d].IsTypeOnly=%v want %v", index, imp.IsTypeOnly, wantTypeOnly)
		}
		if imp.ModuleSpecText != wantSpec {
			t.Fatalf("dynamicImports[%d].ModuleSpecText=%q want %q", index, imp.ModuleSpecText, wantSpec)
		}
	}

	assertDynamic(0, "@/meta/service/models/ui_resource", false)
	assertDynamic(1, "./local_probe", false)
}

func TestImportModuleSpecifierFromExpression_NonLiteral(t *testing.T) {
	if got := ImportModuleSpecifierFromExpression(nil); got != "" {
		t.Fatalf("nil arg = %q", got)
	}
}

func TestImportCallIsTypeOnly(t *testing.T) {
	_, ctx := mustParseTSGoCtx(t, "/virtual/modules/auth/service/tests/dynamic.ts", `const m = await import('./x');`)
	call := findFirstImportCallNode(ctx)
	if call == nil {
		t.Fatal("expected import() call node")
	}
	if ImportCallIsTypeOnly(call) {
		t.Fatal("runtime import() should not be type-only")
	}
	if ImportCallIsTypeOnly(nil) {
		t.Fatal("nil call should not be type-only")
	}

	typeQuery := &tsast.Node{Kind: tsast.KindTypeQuery}
	if !ImportCallIsTypeOnly(&tsast.Node{Kind: tsast.KindCallExpression, Parent: typeQuery}) {
		t.Fatal("TypeQuery parent should mark type-only")
	}
	importType := &tsast.Node{Kind: tsast.KindImportType}
	if !ImportCallIsTypeOnly(&tsast.Node{Kind: tsast.KindCallExpression, Parent: importType}) {
		t.Fatal("ImportType parent should mark type-only")
	}
}

func findFirstImportCallNode(ctx *tsgoImportExportCtx) *tsast.Node {
	if ctx == nil || ctx.source == nil {
		return nil
	}
	var found *tsast.Node
	var walk func(*tsast.Node)
	walk = func(node *tsast.Node) {
		if node == nil || found != nil {
			return
		}
		if node.Kind == tsast.KindCallExpression {
			call := node.AsCallExpression()
			if call != nil && call.Expression != nil && call.Expression.Kind == tsast.KindImportKeyword {
				found = node
				return
			}
		}
		node.ForEachChild(func(child *tsast.Node) bool {
			walk(child)
			return false
		})
	}
	for _, stmt := range ctx.source.Statements.Nodes {
		walk(stmt)
	}
	return found
}

func TestCollectDynamicImports_NilSource(t *testing.T) {
	c := &tsgoImportExportCtx{}
	c.collectDynamicImports()
}

func TestCollectDynamicImports_NilReceiver(t *testing.T) {
	var c *tsgoImportExportCtx
	c.collectDynamicImports()
}

func TestCollectDynamicImports_SkipsNilChildNodes(t *testing.T) {
	path := "/virtual/modules/auth/service/tests/dynamic.ts"
	content := `await import('./a'); await import('./b');`
	_, ctx := mustParseTSGoCtx(t, path, content)
	if len(ctx.dynamicImports) != 2 {
		t.Fatalf("dynamicImports len = %d, want 2", len(ctx.dynamicImports))
	}
}

func TestImportModuleSpecifierFromExpression_DefaultBranch(t *testing.T) {
	path := "/virtual/modules/auth/service/tests/dynamic.ts"
	content := `const key = 'meta'; await import(key);`
	p := &TsParser{
		Path:      path,
		Content:   content,
		PathAlias: map[string]string{"@/*": "/virtual/modules/*"},
	}
	if err := p.ParseImport(nil); err != nil {
		t.Fatalf("ParseImport() error = %v", err)
	}
	if len(p.DynamicImports) != 0 {
		t.Fatalf("non-literal dynamic import len = %d, want 0", len(p.DynamicImports))
	}
}

func TestCollectDynamicImports_NilStatementNode(t *testing.T) {
	path := "/virtual/modules/auth/service/tests/dynamic.ts"
	content := `await import('./x');`
	_, ctx := mustParseTSGoCtx(t, path, content)
	ctx.source.Statements.Nodes = append([]*tsast.Node{nil}, ctx.source.Statements.Nodes...)
	ctx.dynamicImports = nil
	ctx.collectDynamicImports()
	if len(ctx.dynamicImports) != 1 {
		t.Fatalf("dynamicImports len = %d, want 1", len(ctx.dynamicImports))
	}
}

func TestCollectDynamicImports_WalkNilNode(t *testing.T) {
	c := &tsgoImportExportCtx{}
	c.walkDynamicImports(nil)
	c.collectDynamicImports()
}

func TestParseDynamicImports_SkipsImportWithoutArguments(t *testing.T) {
	path := "/virtual/modules/auth/service/tests/dynamic.ts"
	content := `const f = import();`
	p := &TsParser{Path: path, Content: content}
	if err := p.ParseImport(nil); err != nil {
		t.Fatalf("ParseImport() error = %v", err)
	}
	if len(p.DynamicImports) != 0 {
		t.Fatalf("import() without args len = %d, want 0", len(p.DynamicImports))
	}
}

func TestParseDynamicImports_ParserResult(t *testing.T) {
	path := "/virtual/modules/auth/service/tests/observability.test.ts"
	content := `await import('@/meta/service/models/ui_resource');`
	p := &TsParser{
		Path:      path,
		Content:   content,
		PathAlias: map[string]string{"@/*": "/virtual/modules/*"},
	}
	if err := p.ParseImport(nil); err != nil {
		t.Fatalf("ParseImport() error = %v", err)
	}
	if len(p.DynamicImports) != 1 {
		t.Fatalf("DynamicImports len = %d, want 1", len(p.DynamicImports))
	}
	if got := p.DynamicImports[0].ModuleSpecPath; got != "/virtual/modules/meta/service/models/ui_resource" {
		t.Fatalf("ModuleSpecPath = %q", got)
	}
}
