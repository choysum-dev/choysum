// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package extract

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCollectMetadataVueTemplateDirect(t *testing.T) {
	vue := `<template>
  <OVarCharField :store="store" prop="Name" label="Name" />
  <OBooleanField prop="IsActive" :label="'Active'" />
  <ODialog search-view-title="Search Companies" />
</template>
`
	terms, issues := CollectMetadataVueTemplate(CollectOptions{ModuleName: "base", RelPath: "web/views/CompanyList.vue"}, vue)
	if len(issues) != 0 {
		t.Fatalf("issues: %#v", issues)
	}
	if len(terms) < 3 {
		t.Fatalf("terms=%d %#v", len(terms), terms)
	}

	terms2, issues2 := CollectVueMetadata(CollectOptions{ModuleName: "base", RelPath: "web/views/CompanyList.vue"}, vue)
	t.Logf("CollectVueMetadata terms=%d issues=%#v", len(terms2), issues2)
	for _, term := range terms2 {
		t.Logf("  %s %s %q", term.Kind, term.Scope, term.Src)
	}
	if len(terms2) < 3 {
		t.Fatalf("CollectVueMetadata terms=%d", len(terms2))
	}
}

func TestExtractModuleMetadataFieldMenuSelection(t *testing.T) {
	root := t.TempDir()
	moduleRoot := filepath.Join(root, "base")
	viewDir := filepath.Join(moduleRoot, "web", "views")
	menuDir := filepath.Join(moduleRoot, "web", "menu")
	modelDir := filepath.Join(moduleRoot, "service", "models")
	for _, d := range []string{viewDir, menuDir, modelDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	vue := `<template>
  <OVarCharField :store="store" prop="Name" label="Name" />
  <OBooleanField prop="IsActive" :label="'Active'" />
  <ODialog search-view-title="Search Companies" />
</template>
`
	if err := os.WriteFile(filepath.Join(viewDir, "CompanyList.vue"), []byte(vue), 0o644); err != nil {
		t.Fatal(err)
	}

	menu := `import { defineMenu } from '@/core/web/resource'
export const menus = [
  defineMenu('base.menu.company', { title: 'Company Management', path: '/base/companies' }),
]
`
	if err := os.WriteFile(filepath.Join(menuDir, "menus.ts"), []byte(menu), 0o644); err != nil {
		t.Fatal(err)
	}

	model := `import { Field, Model } from '@/core/service'
@Model('BankAccount')
export default class BankAccount {
  @Field({
    type: 'selection',
    selection: [
      { value: 'checking', label: 'Checking' },
      { value: 'savings', label: 'Savings' },
    ],
  })
  AccountType?: string;
}
`
	if err := os.WriteFile(filepath.Join(modelDir, "bank_account.ts"), []byte(model), 0o644); err != nil {
		t.Fatal(err)
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
		`#. kind: field_label`,
		`msgctxt "web/views/CompanyList@Name"`,
		`msgid "Name"`,
		`msgctxt "web/views/CompanyList@IsActive"`,
		`msgid "Active"`,
		`msgctxt "web/views/CompanyList@searchViewTitle"`,
		`msgid "Search Companies"`,
		`#. kind: menu`,
		`msgctxt "web/menu/menus@base.menu.company"`,
		`msgid "Company Management"`,
		`#. kind: selection_label`,
		`msgctxt "service/models/bank_account@AccountType.checking"`,
		`msgid "Checking"`,
		`msgid "Savings"`,
	} {
		if !strings.Contains(pot, want) {
			t.Fatalf("pot missing %q\n%s", want, pot)
		}
	}
	if strings.Contains(pot, `msgid "checking"`) {
		t.Fatal("selection value must not be extracted as msgid")
	}

	off, err := ExtractModuleWithOptions(moduleRoot, "base", ExtractModuleOptions{WritePot: false, Metadata: false})
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range off.Entries {
		if e.Kind != "" && e.Kind != KindLiteral {
			t.Fatalf("--no-metadata still extracted kind=%s msgid=%s", e.Kind, e.Msgid)
		}
	}
}
