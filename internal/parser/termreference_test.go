// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package parser

import (
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/pkg/meta"
)

func TestParseResourceTitleExprTermReferenceBinding(t *testing.T) {
	source := `
const { _lt } = createTranslate('base', { scope: 'web/views/CountryListView' });
`
	bindings := ParseTranslateBindings(source)
	if len(bindings) != 1 || !bindings["_lt"].ReferenceOutput {
		t.Fatalf("unexpected bindings: %#v", bindings)
	}

	title, titleText, ok := ParseResourceTitleExpr("_lt('Country')", "base", bindings)
	if !ok || title != "Country" || titleText == nil || titleText.Src != "Country" {
		t.Fatalf("ParseResourceTitleExpr() = (%q, %#v, %v)", title, titleText, ok)
	}
}

func TestCloneTermReferenceWithSrc(t *testing.T) {
	base := NewTermReferenceFromParts("base", "web/views/CountryListView", "Country", "literal")
	clone := CloneTermReferenceWithSrc(&base, "Create Country")
	if clone == nil || clone.Src != "Create Country" || clone.Scope != base.Scope {
		t.Fatalf("CloneTermReferenceWithSrc() = %#v", clone)
	}
}

func NewTermReferenceFromParts(module, scope, src, kind string) meta.TermReference {
	return meta.NewTermReference(module, scope, src, kind)
}

func TestDeriveOwnerModuleFromSourcePath(t *testing.T) {
	if got := DeriveOwnerModuleFromSourcePath("modules/auth/web/a.ts"); got != "auth" {
		t.Fatalf("got %q", got)
	}
	if got := DeriveOwnerModuleFromSourcePath("/abs/modules/web/service/x.ts"); got != "web" {
		t.Fatalf("got %q", got)
	}
	if got := DeriveOwnerModuleFromSourcePath("src/pkg/here.ts"); got != "" {
		t.Fatalf("got %q", got)
	}
}

func TestParseTermReferenceCall(t *testing.T) {
	binding := TranslateBinding{Module: "auth", DefaultScope: "web/page", ReferenceOutput: true}
	ref, ok := ParseTermReferenceCall(`_lt('Hello')`, "auth", binding)
	if !ok || ref == nil || ref.Src != "Hello" || ref.Scope != "web/page" {
		t.Fatalf("basic = %#v ok=%v", ref, ok)
	}
	ref, ok = ParseTermReferenceCall(`_lt('Hi', { scope: 'explicit@loc' })`, "auth", binding)
	if !ok || ref.Scope != "explicit@loc" {
		t.Fatalf("explicit scope = %#v ok=%v", ref, ok)
	}
	ref, ok = ParseTermReferenceCall(`_lt('Hi', { path: 'web/a', location: 'title' })`, "auth", binding)
	if !ok || ref.Scope != "web/a@title" {
		t.Fatalf("path@location = %#v ok=%v", ref, ok)
	}
	if _, ok := ParseTermReferenceCall(`_t('Hi')`, "auth", TranslateBinding{ReferenceOutput: false}); ok {
		t.Fatal("text _t should fail")
	}
	ref, ok = ParseTermReferenceCall(`_lt('Bare')`, "auth", TranslateBinding{Module: "auth", DefaultScope: "web/page"})
	if !ok || ref == nil || ref.Src != "Bare" {
		t.Fatalf("bare _lt without ReferenceOutput binding = %#v ok=%v", ref, ok)
	}
}

func TestParseResourceTitleExprStringLiteral(t *testing.T) {
	title, ref, ok := ParseResourceTitleExpr(`"Plain Title"`, "auth", nil)
	if !ok || title != "Plain Title" || ref != nil {
		t.Fatalf("got title=%q ref=%#v ok=%v", title, ref, ok)
	}
}

func TestCloneTermReferenceWithSrcNilEmpty(t *testing.T) {
	if CloneTermReferenceWithSrc(nil, "x") != nil {
		t.Fatal("nil reference")
	}
	base := NewTermReferenceFromParts("auth", "a@b", "Hello", "literal")
	if CloneTermReferenceWithSrc(&base, "  ") != nil {
		t.Fatal("empty src")
	}
}

func TestParseCreateTranslateOptionsAndBalancedArgs(t *testing.T) {
	scope := parseCreateTranslateOptions(`path: "web/a", location: "title"`)
	if scope != "web/a@title" {
		t.Fatalf("path/location = %q", scope)
	}
	scope = parseCreateTranslateOptions(`scope: 'explicit'`)
	if scope != "explicit" {
		t.Fatalf("scope = %q", scope)
	}
	scope = parseCreateTranslateOptions("")
	if scope != "" {
		t.Fatal("empty options")
	}
	scope = parseCreateTranslateOptions(`path: ` + "`web/b`" + `, location: ` + "`loc`")
	if scope != "web/b@loc" {
		t.Fatalf("backtick path = %q", scope)
	}

	if _, ok := parseBalancedCallArguments("foo(", -1); ok {
		t.Fatal("bad index")
	}
	args, ok := parseBalancedCallArguments(`createTranslate('auth', { scope: "a)b" })`, strings.Index(`createTranslate('auth', { scope: "a)b" })`, "("))
	if !ok || !strings.Contains(args, "auth") {
		t.Fatalf("balanced args = %q ok=%v", args, ok)
	}
	if _, ok := parseBalancedCallArguments("createTranslate('unclosed'", strings.Index("createTranslate('unclosed'", "(")); ok {
		t.Fatal("unclosed should fail")
	}

	bindings := ParseTranslateBindings(`
const { _lt: aliased } = createTranslate("auth", { path: "p", location: "l" });
const { _t } = createTranslate('web');
`)
	if !bindings["aliased"].ReferenceOutput || bindings["aliased"].DefaultScope != "p@l" {
		t.Fatalf("aliased = %#v", bindings["aliased"])
	}
	if bindings["_t"].Module != "web" || bindings["_t"].ReferenceOutput {
		t.Fatalf("_t = %#v", bindings["_t"])
	}

	dual := ParseTranslateBindings(`const { _t, _lt } = createTranslate('auth', { scope: 'shared' });`)
	if dual["_t"].Module != "auth" || dual["_t"].ReferenceOutput {
		t.Fatalf("dual _t = %#v", dual["_t"])
	}
	if dual["_lt"].Module != "auth" || !dual["_lt"].ReferenceOutput || dual["_lt"].DefaultScope != "shared" {
		t.Fatalf("dual _lt = %#v", dual["_lt"])
	}

	title, refMeta, ok := ParseResourceTitleExpr(`_lt('Hello', { path: 'web/x', location: 't' })`, "auth", map[string]TranslateBinding{
		"_lt": {Module: "auth", ReferenceOutput: true},
	})
	if !ok || title != "Hello" || refMeta == nil || refMeta.Scope != "web/x@t" {
		t.Fatalf("title expr = %q %#v ok=%v", title, refMeta, ok)
	}
	if _, _, ok := ParseResourceTitleExpr("", "auth", nil); ok {
		t.Fatal("empty expr")
	}
	if lit, ok := parseTextCallLiteral("`tick`"); !ok || lit != "tick" {
		t.Fatalf("backtick literal = %q ok=%v", lit, ok)
	}
}
