// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package extract

import "testing"

func TestCollectTemplateRegexPatterns(t *testing.T) {
	html := `
<div>
  {{ _t('Hello') }}
  {{ _lt("Lazy") }}
  <input :placeholder="_t('Name')" />
  <span :title='_t("Title")'></span>
  {{ _t(foo) }}
</div>
`
	terms, issues := CollectTemplateRegex(CollectOptions{
		ModuleName: "auth",
		RelPath:    "web/widgets/Box.vue",
	}, html)

	want := map[string]bool{"Hello": false, "Lazy": false, "Name": false, "Title": false}
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
	if len(issues) == 0 {
		t.Fatal("expected non-literal issue for _t(foo)")
	}
}
