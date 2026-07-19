// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package extract

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractModuleWritesPotWithMsgctxt(t *testing.T) {
	root := t.TempDir()
	moduleRoot := filepath.Join(root, "auth")
	pageDir := filepath.Join(moduleRoot, "web", "pages")
	if err := os.MkdirAll(pageDir, 0o755); err != nil {
		t.Fatal(err)
	}

	ts := `const { _t } = createTranslate('auth')
const title = _t('Sign in')
withI18nScope('manual.box', () => { _t('Boxed') })
_t(x)
`
	if err := os.WriteFile(filepath.Join(pageDir, "Login.ts"), []byte(ts), 0o644); err != nil {
		t.Fatal(err)
	}

	vue := `<template><span>{{ _t('Welcome') }}</span></template>
<script setup lang="ts">
const { _t } = createTranslate('auth')
const label = _t('Page')
</script>
`
	if err := os.WriteFile(filepath.Join(pageDir, "Home.vue"), []byte(vue), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := ExtractModule(moduleRoot, "auth", nil, true)
	if err != nil {
		t.Fatalf("ExtractModule: %v", err)
	}
	if result.PotPath == "" {
		t.Fatal("expected pot path")
	}
	raw, err := os.ReadFile(result.PotPath)
	if err != nil {
		t.Fatal(err)
	}
	pot := string(raw)
	for _, want := range []string{
		`msgctxt "web/pages/Login@title"`,
		`msgid "Sign in"`,
		`msgctxt "manual.box"`,
		`msgid "Boxed"`,
		`msgctxt "web/pages/Home"`,
		`msgid "Welcome"`,
		`msgctxt "web/pages/Home@label"`,
		`msgid "Page"`,
	} {
		if !strings.Contains(pot, want) {
			t.Fatalf("pot missing %q\n%s", want, pot)
		}
	}
	if strings.Contains(pot, "msgid \"x\"") {
		t.Fatal("non-literal should not appear in pot")
	}
	foundWarn := false
	for _, issue := range result.Issues {
		if issue.Code == IssueNonLiteralMsgid {
			foundWarn = true
		}
	}
	if !foundWarn {
		t.Fatalf("expected non-literal warn, got %#v", result.Issues)
	}
}

func TestExtractModuleSkipsConventionalTestSources(t *testing.T) {
	moduleRoot := filepath.Join(t.TempDir(), "base")
	sources := map[string]string{
		"web/menu/menus.ts":               `_t('Production menu')`,
		"web/menu/menus.test.ts":          `_t('Test file TS')`,
		"web/menu/menus.test.tsx":         `_t('Test file TSX')`,
		"web/menu/menus.spec.ts":          `_t('Spec file TS')`,
		"web/menu/menus.spec.tsx":         `_t('Spec file TSX')`,
		"web/test/helper.ts":              `_t('Test directory helper')`,
		"web/tests/helper.ts":             `_t('Tests directory helper')`,
		"web/__tests__/helper.vue":        `<template>{{ _t('Vue test helper') }}</template>`,
		"web/contest/helper.ts":           `_t('Contest production')`,
		"web/testimonials/helper.tsx":     `_t('Testimonials production')`,
		"web/menu/menus.tested.ts":        `_t('Tested production')`,
		"web/menu/menus.specification.ts": `_t('Specification production')`,
	}
	for rel, content := range sources {
		path := filepath.Join(moduleRoot, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	result, err := ExtractModule(moduleRoot, "base", nil, true)
	if err != nil {
		t.Fatalf("ExtractModule: %v", err)
	}
	raw, err := os.ReadFile(result.PotPath)
	if err != nil {
		t.Fatal(err)
	}
	pot := string(raw)
	for _, want := range []string{
		"Production menu",
		"Contest production",
		"Testimonials production",
		"Tested production",
		"Specification production",
	} {
		if !strings.Contains(pot, `msgid "`+want+`"`) {
			t.Errorf("pot missing production term %q\n%s", want, pot)
		}
	}
	for _, skipped := range []string{
		"Test file TS",
		"Test file TSX",
		"Spec file TS",
		"Spec file TSX",
		"Test directory helper",
		"Tests directory helper",
		"Vue test helper",
	} {
		if strings.Contains(pot, skipped) {
			t.Errorf("pot contains skipped test term %q\n%s", skipped, pot)
		}
	}
}

func TestExtractModuleDoesNotSkipModuleRootNamedTest(t *testing.T) {
	moduleRoot := filepath.Join(t.TempDir(), "test")
	path := filepath.Join(moduleRoot, "web", "pages", "Home.ts")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`_t('From test module')`), 0o644); err != nil {
		t.Fatal(err)
	}
	// Nested skipped dir must still be ignored.
	nested := filepath.Join(moduleRoot, "test", "helper.ts")
	if err := os.MkdirAll(filepath.Dir(nested), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(nested, []byte(`_t('Nested test helper')`), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := ExtractModule(moduleRoot, "test", nil, false)
	if err != nil {
		t.Fatalf("ExtractModule: %v", err)
	}
	found := false
	for _, term := range result.Terms {
		if term.Src == "From test module" {
			found = true
		}
		if term.Src == "Nested test helper" {
			t.Fatalf("nested test/ dir should still be skipped, got %#v", result.Terms)
		}
	}
	if !found {
		t.Fatalf("expected term from module named test, got %#v", result.Terms)
	}
}

func TestWritePotRejectsMissingMsgctxt(t *testing.T) {
	err := WritePot(os.Stderr, "auth", []PotEntry{{Msgid: "Save"}})
	if err == nil {
		t.Fatal("expected error for missing msgctxt")
	}
}
