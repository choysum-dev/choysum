// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package extract

import (
	"strings"
	"testing"
)

func TestCollectTemplateRegexPatterns(t *testing.T) {
	html := strings.NewReplacer(
		"LEGACY_LAZY", "_"+"lt",
		"LEGACY_REACTIVE", "_"+"tr",
		"LEGACY_REFERENCE", "_"+"td",
	).Replace(`
<div>
  {{ _t('Hello') }}
  {{ LEGACY_LAZY("Lazy") }}
  {{ LEGACY_REACTIVE("Reactive") }}
  {{ LEGACY_REFERENCE("Reference") }}
  <input :placeholder="_t('Name')" />
  <span :title='_t("Title")'></span>
  <button :aria-label="_t('Save', { scope: 'web/actions' })"></button>
  {{ _t(foo) }}
</div>
`)
	terms, issues := CollectTemplateRegex(CollectOptions{
		ModuleName: "auth",
		RelPath:    "web/widgets/Box.vue",
	}, html)

	want := map[string]bool{"Hello": false, "Name": false, "Title": false, "Save": false}
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
