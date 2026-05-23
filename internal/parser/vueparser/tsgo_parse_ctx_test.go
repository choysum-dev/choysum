// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package vueparser

import (
	"path/filepath"
	"testing"

	tsast "github.com/buke/typescript-go-internal/pkg/ast"
	tscore "github.com/buke/typescript-go-internal/pkg/core"
	"github.com/choysum-dev/choysum/internal/parser"
)

func TestParseTSGoCtxParsesImportsAndExports(t *testing.T) {
	path := filepath.Join(t.TempDir(), "addons", "auth", "web", "views", "sample.ts")
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

	ctx, err := parseTSGoCtx(map[string]string{"@": filepath.Join(filepath.Dir(filepath.Dir(filepath.Dir(path))), "")}, path, content)
	if err != nil {
		t.Fatalf("parseTSGoCtx() error = %v", err)
	}

	if got := ctx.currentModuleSpecPath(); got != path[:len(path)-len(filepath.Ext(path))] {
		t.Fatalf("currentModuleSpecPath() = %q", got)
	}
	if ctx.imports["DefaultView"] == nil || ctx.imports["DefaultView"].ReferenceIdent != "default" {
		t.Fatalf("unexpected default import: %#v", ctx.imports["DefaultView"])
	}
	if ctx.imports["alias"] == nil || ctx.imports["alias"].ReferenceIdent != "named" {
		t.Fatalf("unexpected aliased import: %#v", ctx.imports["alias"])
	}
	if ctx.imports["plain"] == nil || ctx.imports["plain"].ReferenceIdent != "plain" {
		t.Fatalf("unexpected named import: %#v", ctx.imports["plain"])
	}
	if ctx.imports["toolkit"] == nil || ctx.imports["toolkit"].ReferenceIdent != "*" {
		t.Fatalf("unexpected namespace import: %#v", ctx.imports["toolkit"])
	}

	if ctx.exports["exportedAlias"] == nil || ctx.exports["exportedAlias"].ReferenceIdent != "named" {
		t.Fatalf("unexpected re-export alias: %#v", ctx.exports["exportedAlias"])
	}
	if ctx.exports["default"] == nil || ctx.exports["default"].ReferenceIdent != "Widget" {
		t.Fatalf("unexpected default property-access export: %#v", ctx.exports["default"])
	}
	if ctx.exports["default"].ModuleSpecPath != ctx.imports["toolkit"].ModuleSpecPath {
		t.Fatalf("unexpected default export module spec: %#v", ctx.exports["default"])
	}
	if ctx.exports["ready"] == nil || ctx.exports["ready"].ReferenceIdent != "ready" {
		t.Fatalf("unexpected named export: %#v", ctx.exports["ready"])
	}
	if ctx.exports["namespaceExport"] == nil {
		t.Fatalf("expected namespace export declaration")
	}
	if wildcard := ctx.exports["*"]; wildcard == nil || len(wildcard.Wildcard) != 2 {
		t.Fatalf("unexpected wildcard exports: %#v", wildcard)
	}
}

func TestParseTSGoCtxHandlesDefaultDeclarationsAndMergeHelpers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "component.vue")
	content := `
export default class NamedView {}
export const enabled = true
`

	ctx, err := parseTSGoCtxWithKind(nil, path, content, tscore.ScriptKindTS, true)
	if err != nil {
		t.Fatalf("parseTSGoCtxWithKind() error = %v", err)
	}
	if got := ctx.currentModuleSpecPath(); got != path {
		t.Fatalf("currentModuleSpecPath() for vue = %q, want %q", got, path)
	}
	if ctx.exports["default"] == nil || ctx.exports["default"].ReferenceIdent != "NamedView" {
		t.Fatalf("unexpected default declaration export: %#v", ctx.exports["default"])
	}
	if ctx.exports["enabled"] == nil {
		t.Fatalf("expected named variable export")
	}

	dstImports := map[string]*parser.Import{"keep": {ReferenceIdent: "keep"}}
	mergeImports(dstImports, map[string]*parser.Import{"next": {ReferenceIdent: "next"}})
	if dstImports["keep"] == nil || dstImports["next"] == nil {
		t.Fatalf("unexpected merged imports: %#v", dstImports)
	}

	dstExports := map[string]*parser.Export{
		"*": {Wildcard: []*parser.Export{{ReferenceIdent: "*", ModuleSpecPath: "one"}}},
	}
	mergeExports(dstExports, map[string]*parser.Export{
		"*": {Wildcard: []*parser.Export{{ReferenceIdent: "*", ModuleSpecPath: "two"}}},
	})
	if len(dstExports["*"].Wildcard) != 2 {
		t.Fatalf("unexpected merged wildcard exports: %#v", dstExports["*"])
	}
	mergeExports(nil, dstExports)
	mergeImports(nil, dstImports)

	if got := exportDeclarationName(nil); got != "" {
		t.Fatalf("exportDeclarationName(nil) = %q, want empty", got)
	}
}

func TestTSGoCtxConvertReferenceWithModuleSpecAndNodeTextGuards(t *testing.T) {
	path := filepath.Join(t.TempDir(), "views", "child.ts")
	content := `
import DefaultView from './base.ts'

const localView = DefaultView

export default localView
`

	ctx, err := parseTSGoCtx(nil, path, content)
	if err != nil {
		t.Fatalf("parseTSGoCtx() error = %v", err)
	}

	if moduleSpec, referenceIdent := ctx.convertReferenceWithModuleSpec("DefaultView"); referenceIdent != "default" || moduleSpec != filepath.Join(filepath.Dir(path), "base") {
		t.Fatalf("unexpected imported reference conversion: %q %q", moduleSpec, referenceIdent)
	}
	if moduleSpec, referenceIdent := ctx.convertReferenceWithModuleSpec("localView"); referenceIdent != "default" || moduleSpec != ctx.currentModuleSpecPath() {
		t.Fatalf("unexpected default export conversion: %q %q", moduleSpec, referenceIdent)
	}
	if moduleSpec, referenceIdent := ctx.convertReferenceWithModuleSpec("missingRef"); referenceIdent != "missingRef" || moduleSpec != ctx.currentModuleSpecPath() {
		t.Fatalf("unexpected fallback conversion: %q %q", moduleSpec, referenceIdent)
	}

	identifierNode := findFirstNodeInSourceFile(ctx.source, tsast.KindIdentifier)
	if identifierNode == nil || ctx.nodeText(identifierNode) == "" {
		t.Fatalf("expected nodeText() to return source text for valid identifier node")
	}
	if got := ctx.nodeText(&tsast.Node{}); got != "" {
		t.Fatalf("nodeText(invalid) = %q, want empty", got)
	}
}
