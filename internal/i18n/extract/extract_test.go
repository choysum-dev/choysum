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

func TestWritePotRejectsMissingMsgctxt(t *testing.T) {
	err := WritePot(os.Stderr, "auth", []PotEntry{{Msgid: "Save"}})
	if err == nil {
		t.Fatal("expected error for missing msgctxt")
	}
}
