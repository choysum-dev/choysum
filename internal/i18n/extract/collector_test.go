// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package extract

import (
	"strings"
	"testing"
)

func TestFormatScopeAndResolveI18nScope(t *testing.T) {
	if got := FormatScope("web/pages/Login", ""); got != "web/pages/Login" {
		t.Fatalf("FormatScope path only = %q", got)
	}
	if got := FormatScope("web/pages/Login", "title"); got != "web/pages/Login@title" {
		t.Fatalf("FormatScope path@loc = %q", got)
	}
	if got := ResolveI18nScope("game.rescue", nil, "a", "b"); got != "game.rescue" {
		t.Fatalf("manual scope = %q", got)
	}
	if got := ResolveI18nScope("", []string{"game.rescue"}, "ignored", ""); got != "game.rescue" {
		t.Fatalf("stack scope = %q", got)
	}
	if got := ResolveI18nScope("", nil, "web/pages/Login", "title"); got != "web/pages/Login@title" {
		t.Fatalf("auto scope = %q", got)
	}
	if got := ScopePathFromRelPath("web/pages/Login.vue"); got != "web/pages/Login" {
		t.Fatalf("ScopePathFromRelPath = %q", got)
	}
}

func TestCollectScriptLiteralsAndScopes(t *testing.T) {
	content := `
const { _t, _lt } = createTranslate('auth')

const saveLabel = _t('Save')
const lazyTitle = _lt('Access Error')

function submitForm() {
  return _t('Submit')
}

withI18nScope('game.rescue', () => {
  _t('Save')
})

_t('Manual', { scope: 'manual.scope' })
_t(dynamicVar)
`

	terms, issues := CollectScript(CollectOptions{
		ModuleName: "auth",
		RelPath:    "web/pages/Login.ts",
	}, content)

	bySrc := map[string][]TermOccurrence{}
	for _, term := range terms {
		bySrc[term.Src] = append(bySrc[term.Src], term)
	}

	if len(bySrc["Save"]) < 2 {
		t.Fatalf("expected at least 2 Save terms, got %#v", terms)
	}
	foundBinding := false
	foundManualStack := false
	for _, term := range bySrc["Save"] {
		if term.Scope == "web/pages/Login@saveLabel" {
			foundBinding = true
		}
		if term.Scope == "game.rescue" {
			foundManualStack = true
		}
	}
	if !foundBinding {
		t.Fatalf("missing saveLabel scope in %#v", bySrc["Save"])
	}
	if !foundManualStack {
		t.Fatalf("missing withI18nScope scope in %#v", bySrc["Save"])
	}

	if got := bySrc["Access Error"]; len(got) != 1 || got[0].Scope != "web/pages/Login@lazyTitle" {
		t.Fatalf("unexpected _lt term: %#v", got)
	}
	if got := bySrc["Submit"]; len(got) != 1 || got[0].Scope != "web/pages/Login@submitForm" {
		t.Fatalf("unexpected function scope: %#v", got)
	}
	if got := bySrc["Manual"]; len(got) != 1 || got[0].Scope != "manual.scope" {
		t.Fatalf("unexpected manual options scope: %#v", got)
	}

	foundNonLiteral := false
	for _, issue := range issues {
		if issue.Code == IssueNonLiteralMsgid {
			foundNonLiteral = true
			break
		}
	}
	if !foundNonLiteral {
		t.Fatalf("expected non_literal_msgid warn, got %#v", issues)
	}
}

func TestCollectScriptCreateTranslateMismatch(t *testing.T) {
	content := `const { _t } = createTranslate('other')
_t('Hi')
`
	_, issues := CollectScript(CollectOptions{ModuleName: "auth", RelPath: "a.ts"}, content)
	found := false
	for _, issue := range issues {
		if issue.Code == IssueModuleMismatch {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected module mismatch warn, got %#v", issues)
	}
}

func TestCollectVueScriptAndTemplate(t *testing.T) {
	content := `<template>
  <div>
    <h1>{{ _t('Welcome') }}</h1>
    <button :title="_t('Click me')">ok</button>
    <span>{{ _t(label) }}</span>
  </div>
</template>
<script setup lang="ts">
const { _t } = createTranslate('auth')
const title = _t('Page Title')
</script>
`

	terms, issues := CollectVue(CollectOptions{
		ModuleName: "auth",
		RelPath:    "web/pages/Home.vue",
	}, content)

	var scriptTitle, templateWelcome, templateClick bool
	for _, term := range terms {
		switch {
		case term.Src == "Page Title" && term.Scope == "web/pages/Home@title":
			scriptTitle = true
		case term.Src == "Welcome" && term.Scope == "web/pages/Home@template":
			templateWelcome = true
		case term.Src == "Click me" && term.Scope == "web/pages/Home@template":
			templateClick = true
		}
	}
	if !scriptTitle || !templateWelcome || !templateClick {
		t.Fatalf("missing expected terms: script=%v welcome=%v click=%v terms=%#v", scriptTitle, templateWelcome, templateClick, terms)
	}

	foundTemplateNonLiteral := false
	for _, issue := range issues {
		if issue.Code == IssueNonLiteralMsgid && strings.Contains(issue.Message, "label") {
			foundTemplateNonLiteral = true
		}
	}
	if !foundTemplateNonLiteral {
		t.Fatalf("expected template non-literal warn, got %#v", issues)
	}
}
