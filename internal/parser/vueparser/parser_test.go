// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package vueparser

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/pkg/meta"
)

func TestVueParserParseRejectsUnsupportedExtension(t *testing.T) {
	runtimeScope := newVueParserTestScope()
	module := &meta.IrModule{Path: "/virtual/addons/auth", ApplicationStr: "auth"}
	p := NewVueParser(runtimeScope, module)

	_, err := p.Parse(nil, filepath.Join(t.TempDir(), "notes.md"), "# hello")
	if err == nil || !strings.Contains(err.Error(), "unsupported file type") {
		t.Fatalf("expected unsupported file type error, got %v", err)
	}
}

func TestVueParserParseTSFileCollectsImportsExportsAndUiResources(t *testing.T) {
	runtimeScope := newVueParserTestScope()
	runtimeOpts := runtimeOptionsFromScope(runtimeScope)
	module := &meta.IrModule{Path: filepath.Join(runtimeOpts.addonsPath, "auth"), ApplicationStr: "auth"}
	p := NewVueParser(runtimeScope, module)

	path := filepath.Join(runtimeOpts.addonsPath, "auth", "web", "views", "users.ts")
	content := `
import DefaultView, { helper as alias } from './base.ts'

export { alias as exportedAlias }
export default DefaultView

defineRoute('auth.route.user_list', {
	meta: { pageTitle: 'Users' },
	actions: ['auth.action.user_create'],
})
`

	result, err := p.Parse(nil, path, content)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if result == nil {
		t.Fatal("expected parser result")
	}
	if result.Path != path || result.RawContent == "" {
		t.Fatalf("unexpected parser result header: %#v", result)
	}
	if result.Imports["DefaultView"] == nil || result.Imports["DefaultView"].ReferenceIdent != "default" {
		t.Fatalf("unexpected default import: %#v", result.Imports["DefaultView"])
	}
	if result.Imports["alias"] == nil || result.Imports["alias"].ReferenceIdent != "helper" {
		t.Fatalf("unexpected named import: %#v", result.Imports["alias"])
	}
	if result.Exports["exportedAlias"] == nil || result.Exports["exportedAlias"].ReferenceIdent != "helper" {
		t.Fatalf("unexpected re-export alias: %#v", result.Exports["exportedAlias"])
	}
	if result.Exports["default"] == nil || result.Exports["default"].ReferenceIdent != "default" {
		t.Fatalf("unexpected default export: %#v", result.Exports["default"])
	}
	if len(result.UiResourceDecls) != 1 {
		t.Fatalf("expected one ui resource decl, got %#v", result.UiResourceDecls)
	}
	decl := result.UiResourceDecls[0]
	if decl.ID != "auth.route.user_list" || decl.Title != "Users" {
		t.Fatalf("unexpected ui decl: %#v", decl)
	}
	if len(decl.Actions) != 1 || decl.Actions[0] != "auth.action.user_create" {
		t.Fatalf("unexpected route actions: %#v", decl.Actions)
	}
	if len(result.UiResourceDeclIssues) != 0 {
		t.Fatalf("expected no ui decl issues, got %#v", result.UiResourceDeclIssues)
	}
}
