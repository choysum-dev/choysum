// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package extract

import (
	"strings"
	"testing"
)

func TestCollectTemplateRegexPatterns(t *testing.T) {
	bt := "`"
	html := strings.NewReplacer(
		"LEGACY_LAZY", "_"+"lt",
		"LEGACY_REACTIVE", "_"+"tr",
		"LEGACY_REFERENCE", "_"+"td",
		"BT(", bt, // open backtick placeholder for template literals
		")BT", bt, // close backtick placeholder
	).Replace(`
<div>
  {{ _t('Hello') }}
  {{ LEGACY_LAZY("Lazy") }}
  {{ LEGACY_REACTIVE("Reactive") }}
  {{ LEGACY_REFERENCE("Reference") }}
  <input :placeholder="_t('Name')" />
  <span :title='_t("Title")'></span>
  <button :aria-label="_t('Save', { scope: 'web/actions' })"></button>
  {{ _t(BT(Welcome Back)BT) }}
  <input :placeholder="_t(BT(Full Name)BT)" />
  <span :title='_t(BT(Page Title)BT)'></span>
  {{ _t(foo) }}
</div>
`)
	terms, issues := CollectTemplateRegex(CollectOptions{
		ModuleName: "auth",
		RelPath:    "web/widgets/Box.vue",
	}, html)

	want := map[string]bool{
		"Hello": false, "Name": false, "Title": false, "Save": false,
		"Welcome Back": false, "Full Name": false, "Page Title": false,
	}
	for _, term := range terms {
		if term.Scope != "web/widgets/Box" {
			t.Fatalf("unexpected scope %q for %q", term.Scope, term.Src)
		}
		if _, ok := want[term.Src]; ok {
			want[term.Src] = true
		}
	}
	for src, ok := range want {
		if !ok {
			t.Fatalf("missing term %q in %#v", src, terms)
		}
	}
	for _, term := range terms {
		if term.Src == "Lazy" || term.Src == "Reactive" || term.Src == "Reference" {
			t.Fatalf("legacy helper term was unexpectedly extracted: %#v", term)
		}
	}
	if len(issues) == 0 {
		t.Fatal("expected non-literal issue for _t(foo)")
	}
}

func TestCollectTemplateRegexEscapedQuotesInMsgid(t *testing.T) {
	html := `<div>{{ _t('a\', b') }}{{ _t("foo\", bar") }}</div>`
	terms, issues := CollectTemplateRegex(CollectOptions{
		ModuleName: "auth",
		RelPath:    "web/widgets/Box.vue",
	}, html)
	if len(issues) != 0 {
		t.Fatalf("unexpected issues: %#v", issues)
	}
	want := map[string]bool{"a', b": false, `foo", bar`: false}
	for _, term := range terms {
		if _, ok := want[term.Src]; ok {
			want[term.Src] = true
		}
	}
	for src, ok := range want {
		if !ok {
			t.Fatalf("missing term %q in %#v", src, terms)
		}
	}
}

func TestCollectTemplateRegexMultipleTCallsInAttribute(t *testing.T) {
	html := `<input :placeholder="isNew ? _t('Create') : _t('Edit')" />`
	terms, _ := CollectTemplateRegex(CollectOptions{
		ModuleName: "auth",
		RelPath:    "web/widgets/Box.vue",
	}, html)
	want := map[string]bool{"Create": false, "Edit": false}
	for _, term := range terms {
		if _, ok := want[term.Src]; ok {
			want[term.Src] = true
		}
	}
	for src, ok := range want {
		if !ok {
			t.Fatalf("missing term %q in %#v", src, terms)
		}
	}
}

func TestCollectTemplateRegexMultipleTCallsInMustache(t *testing.T) {
	html := `<span>{{ isNew ? _t('Create') : _t('Edit') }}</span>`
	terms, _ := CollectTemplateRegex(CollectOptions{
		ModuleName: "auth",
		RelPath:    "web/widgets/Box.vue",
	}, html)
	want := map[string]bool{"Create": false, "Edit": false}
	for _, term := range terms {
		if _, ok := want[term.Src]; ok {
			want[term.Src] = true
		}
	}
	for src, ok := range want {
		if !ok {
			t.Fatalf("missing term %q in %#v", src, terms)
		}
	}
}
