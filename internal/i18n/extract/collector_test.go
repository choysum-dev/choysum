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
	content := strings.NewReplacer(
		"LEGACY_LAZY", "_"+"lt",
		"LEGACY_REACTIVE", "_"+"tr",
		"LEGACY_REFERENCE", "_"+"td",
	).Replace(`
const { _t } = createTranslate('auth')

const saveLabel = _t('Save')
const ignoredLazy = LEGACY_LAZY('Access Error')
const ignoredReactive = LEGACY_REACTIVE('Reactive title')
const ignoredReference = LEGACY_REFERENCE('Reference title')

function submitForm() {
  return _t('Submit')
}

withI18nScope('game.rescue', () => {
  _t('Save')
})

_t('Manual', { scope: 'manual.scope' })
_t(dynamicVar)
`)

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

	for _, ignored := range []string{"Access Error", "Reactive title", "Reference title"} {
		if got := bySrc[ignored]; len(got) != 0 {
			t.Fatalf("legacy helper %q was unexpectedly extracted: %#v", ignored, got)
		}
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

func TestCollectScriptUsesCreateTranslateDefaultScope(t *testing.T) {
	content := `
const { _t: translate } = createTranslate('auth', { path: 'web/pages/Login' })
const title = translate('User Login')
const override = _t('Override', { scope: 'manual.override' })
`

	terms, issues := CollectScript(CollectOptions{
		ModuleName: "auth",
		RelPath:    "web/pages/Login.vue",
	}, content)
	if len(issues) != 0 {
		t.Fatalf("unexpected issues: %#v", issues)
	}

	bySrc := map[string]TermOccurrence{}
	for _, term := range terms {
		bySrc[term.Src] = term
	}
	if got := bySrc["User Login"].Scope; got != "web/pages/Login" {
		t.Fatalf("default scope = %q", got)
	}
	if got := bySrc["Override"].Scope; got != "manual.override" {
		t.Fatalf("override scope = %q", got)
	}
}

func TestCollectScriptReferenceUsesCreateTranslateDefaultScope(t *testing.T) {
	content := `
const { _t, _t: reference } = createTranslate('base', { output: 'reference', path: 'web/menu', location: 'menus' })
const menuTitle = _t('Master Data')
reference('Company Management')
`

	terms, issues := CollectScript(CollectOptions{
		ModuleName: "base",
		RelPath:    "web/menu/menus.ts",
	}, content)
	if len(issues) != 0 {
		t.Fatalf("unexpected issues: %#v", issues)
	}
	if len(terms) != 2 {
		t.Fatalf("expected 2 reference terms, got %#v", terms)
	}
	for _, term := range terms {
		if term.Scope != "web/menu@menus" {
			t.Fatalf("scope for %q = %q", term.Src, term.Scope)
		}
	}
}

func TestCollectScriptReferenceRequiresLiteralSourceAndScope(t *testing.T) {
	content := `
const { _t, _t: reference } = createTranslate('base', { output: 'reference' })
const valid = _t('Users', { scope: 'base.menu.users' })
reference('Settings', { scope: 'base.menu.settings' })
_t(dynamicSrc, { scope: 'base.menu.dynamic' })
_t('Dynamic scope', { scope: dynamicScope })
_t('Missing scope')
`
	terms, issues := CollectScript(CollectOptions{
		ModuleName: "base",
		RelPath:    "web/menu/menus.ts",
	}, content)

	if len(terms) != 2 {
		t.Fatalf("expected 2 literal reference terms, got %#v", terms)
	}
	if terms[0].Module != "base" || terms[0].Scope != "base.menu.users" || terms[0].Src != "Users" || terms[0].Kind != KindLiteral {
		t.Fatalf("unexpected first reference term: %#v", terms[0])
	}
	if terms[1].Scope != "base.menu.settings" || terms[1].Src != "Settings" {
		t.Fatalf("unexpected aliased reference term: %#v", terms[1])
	}

	var msgidIssues, scopeIssues int
	for _, issue := range issues {
		switch issue.Code {
		case IssueNonLiteralMsgid:
			msgidIssues++
		case IssueNonLiteralScope:
			scopeIssues++
		}
	}
	if msgidIssues != 1 || scopeIssues != 2 {
		t.Fatalf("expected one msgid and two scope warnings, got %#v", issues)
	}
}

func TestCollectScriptSynthesizesModelActionTitles(t *testing.T) {
	content := `
const { _t: _tRef } = createTranslate('base', { output: 'reference', scope: 'web/views/CountryListView' })
defineModelActions('base.Country', { entityTitle: _tRef('Country') })
defineModelActions('base.User', {
  entityTitle: _tRef('User'),
  exclude: ['copy'],
  titles: { delete: _tRef('Deactivate User') },
})
`
	terms, issues := CollectScript(CollectOptions{
		ModuleName: "base",
		RelPath:    "web/views/CountryListView.vue",
	}, content)
	if len(issues) != 0 {
		t.Fatalf("unexpected issues: %#v", issues)
	}
	bySrc := map[string][]string{}
	for _, term := range terms {
		bySrc[term.Src] = append(bySrc[term.Src], term.Scope)
	}
	for _, src := range []string{"Country", "Create Country", "Edit Country", "Delete Country", "Copy Country"} {
		if len(bySrc[src]) == 0 {
			t.Fatalf("missing synthesized/extracted term %q in %#v", src, bySrc)
		}
	}
	if len(bySrc["Copy User"]) != 0 {
		t.Fatalf("excluded copy should not emit Copy User: %#v", bySrc)
	}
	if len(bySrc["Deactivate User"]) == 0 {
		t.Fatalf("expected titles.delete override, got %#v", bySrc)
	}
	if len(bySrc["Delete User"]) != 0 {
		t.Fatalf("override should replace Delete User: %#v", bySrc)
	}
}

func TestCollectScriptFactoryModeIsNotRecognized(t *testing.T) {
	content := `
const { _t } = createTranslate('base', { mode: 'reference' })
const title = _t('Users')
`
	terms, issues := CollectScript(CollectOptions{
		ModuleName: "base",
		RelPath:    "web/menu/menus.ts",
	}, content)
	if len(issues) != 0 {
		t.Fatalf("unexpected issues: %#v", issues)
	}
	if len(terms) != 1 || terms[0].Scope != "web/menu/menus@title" {
		t.Fatalf("factory mode was unexpectedly recognized: %#v", terms)
	}
}

func TestCollectScriptCallOutputOverridesFactoryOutput(t *testing.T) {
	content := `
const { _t: textDefault } = createTranslate('base')
const { _t: referenceDefault } = createTranslate('base', { output: 'reference' })
textDefault('Left to right', { scope: 'base.Language.Direction.ltr', output: 'reference' })
referenceDefault('Interpolated %s', { output: 'text' }, 'value')
`
	terms, issues := CollectScript(CollectOptions{
		ModuleName: "base",
		RelPath:    "service/models/language.ts",
	}, content)
	if len(issues) != 0 {
		t.Fatalf("unexpected issues: %#v", issues)
	}
	if len(terms) != 2 {
		t.Fatalf("expected two terms, got %#v", terms)
	}
	if terms[0].Scope != "base.Language.Direction.ltr" || terms[0].Src != "Left to right" {
		t.Fatalf("unexpected reference override term: %#v", terms[0])
	}
	if terms[1].Scope != "service/models/language" || terms[1].Src != "Interpolated %s" {
		t.Fatalf("unexpected text override term: %#v", terms[1])
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
		case term.Src == "Welcome" && term.Scope == "web/pages/Home":
			templateWelcome = true
		case term.Src == "Click me" && term.Scope == "web/pages/Home":
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

func TestCollectVueTemplateUsesCreateTranslateDefaultScope(t *testing.T) {
	content := `<template>
  <h1>{{ _t('User Login') }}</h1>
  <input :placeholder="_t('Enter username')" />
</template>
<script setup lang="ts">
const { _t } = createTranslate('auth', { scope: 'web/pages/Login' })
</script>
`

	terms, issues := CollectVue(CollectOptions{
		ModuleName: "auth",
		RelPath:    "web/pages/Login.vue",
	}, content)
	if len(issues) != 0 {
		t.Fatalf("unexpected issues: %#v", issues)
	}
	if len(terms) != 2 {
		t.Fatalf("term count = %d, terms=%#v", len(terms), terms)
	}
	for _, term := range terms {
		if term.Scope != "web/pages/Login" {
			t.Fatalf("scope for %q = %q", term.Src, term.Scope)
		}
	}
}
