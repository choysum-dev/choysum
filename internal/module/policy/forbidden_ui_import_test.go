// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package policy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsKitHostModuleEmptyModulesPath(t *testing.T) {
	if isKitHostModule("", "web") {
		t.Fatal("web without modulesPath must not be kit host")
	}
	if isKitHostModule("   ", "web") {
		t.Fatal("blank modulesPath must not make web a kit host")
	}
	if !isKitHostModule("", "choy_ui") {
		t.Fatal("choy_ui remains a kit host even without modulesPath")
	}
}

func TestClassifyForbiddenUiImport(t *testing.T) {
	cases := []struct {
		spec string
		want string
	}{
		{"reka-ui", "reka-ui"},
		{"reka-ui/Dialog", "reka-ui"},
		{"@unovis/vue", "@unovis"},
		{"@unovis/ts", "@unovis"},
		{"ui/button", "ui/*"},
		{"../components/ui/button", "ui/*"},
		{"../components/vendor/ui/button", "ui/*"},
		{"@/choy_ui/web/components/ui/button", "ui/*"},
		{"@/choy_ui/web/components/vendor/ui/button", "ui/*"},
		{"@/web/web/components/ui/button", "ui/*"},
		{"@/web/web/components/vendor/ui/button", "ui/*"},
		{"internal/DataTable", "internal/*"},
		{"../components/internal/DataTable", "internal/*"},
		{"@/choy_ui/web/components/internal/DatePicker", "internal/*"},
		{"@/choy_ui/web/lib/utils", "choy_ui-deep"},
		{"@choysum-dev/choy_ui/web/components/vendor/ui/button", "ui/*"},
		{"@choysum-dev/choy_ui", "choy_ui-deep"},
		{"choy_ui", "choy_ui-deep"},
		{"vue", ""},
		{"@/web/web/components/view/OFormView", ""},
		{"element-plus", ""},
	}
	for _, tc := range cases {
		if got := classifyForbiddenUiImport(tc.spec); got != tc.want {
			t.Fatalf("classifyForbiddenUiImport(%q)=%q want %q", tc.spec, got, tc.want)
		}
	}
}

func TestAssertNoForbiddenUiImports_RejectsDomainReka(t *testing.T) {
	modulesPath := t.TempDir()
	webDir := filepath.Join(modulesPath, "partner", "web")
	if err := os.MkdirAll(webDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modulesPath, "partner", "package.json"), []byte(`{
  "name": "@choysum-dev/partner",
  "choysum": { "moduleName": "partner", "application": "partner" }
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	src := "import { DialogRoot } from 'reka-ui';\nexport const x = DialogRoot;\n"
	if err := os.WriteFile(filepath.Join(webDir, "leak.ts"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	err := AssertNoForbiddenUiImports(modulesPath, "partner")
	if err == nil || !strings.Contains(err.Error(), "reka-ui") {
		t.Fatalf("expected reka-ui ban, got %v", err)
	}
}

func TestAssertNoForbiddenUiImports_RejectsDynamicImport(t *testing.T) {
	modulesPath := t.TempDir()
	webDir := filepath.Join(modulesPath, "partner", "web")
	if err := os.MkdirAll(webDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modulesPath, "partner", "package.json"), []byte(`{
  "name": "@choysum-dev/partner",
  "choysum": { "moduleName": "partner", "application": "partner" }
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	src := "export async function load() {\n  return import('@unovis/vue');\n}\n"
	if err := os.WriteFile(filepath.Join(webDir, "dyn.ts"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	err := AssertNoForbiddenUiImports(modulesPath, "partner")
	if err == nil || !strings.Contains(err.Error(), "@unovis") {
		t.Fatalf("expected @unovis dynamic-import ban, got %v", err)
	}
}

func TestAssertNoForbiddenUiImports_RejectsRuntimeReExport(t *testing.T) {
	modulesPath := t.TempDir()
	webDir := filepath.Join(modulesPath, "partner", "web")
	if err := os.MkdirAll(webDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modulesPath, "partner", "package.json"), []byte(`{
  "name": "@choysum-dev/partner",
  "choysum": { "moduleName": "partner", "application": "partner" }
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	src := "export { DialogRoot } from 'reka-ui';\n"
	if err := os.WriteFile(filepath.Join(webDir, "reexport.ts"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	err := AssertNoForbiddenUiImports(modulesPath, "partner")
	if err == nil || !strings.Contains(err.Error(), "reka-ui") {
		t.Fatalf("expected reka-ui re-export ban, got %v", err)
	}
}

func TestAssertNoForbiddenUiImports_RejectsWildcardReExport(t *testing.T) {
	modulesPath := t.TempDir()
	webDir := filepath.Join(modulesPath, "partner", "web")
	if err := os.MkdirAll(webDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modulesPath, "partner", "package.json"), []byte(`{
  "name": "@choysum-dev/partner",
  "choysum": { "moduleName": "partner", "application": "partner" }
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	src := "export * from 'reka-ui';\n"
	if err := os.WriteFile(filepath.Join(webDir, "star.ts"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	err := AssertNoForbiddenUiImports(modulesPath, "partner")
	if err == nil || !strings.Contains(err.Error(), "reka-ui") {
		t.Fatalf("expected reka-ui wildcard re-export ban, got %v", err)
	}
}

func TestAssertNoForbiddenUiImports_AllowsEmptyWebSource(t *testing.T) {
	modulesPath := t.TempDir()
	webDir := filepath.Join(modulesPath, "partner", "web")
	if err := os.MkdirAll(webDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modulesPath, "partner", "package.json"), []byte(`{
  "name": "@choysum-dev/partner",
  "choysum": { "moduleName": "partner", "application": "partner" }
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(webDir, "empty.ts"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := AssertNoForbiddenUiImports(modulesPath, "partner"); err != nil {
		t.Fatalf("empty web source must pass: %v", err)
	}
}

func TestAssertNoForbiddenUiImports_VueLineMatchesFile(t *testing.T) {
	modulesPath := t.TempDir()
	webDir := filepath.Join(modulesPath, "auth", "web", "pages")
	if err := os.MkdirAll(webDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modulesPath, "auth", "package.json"), []byte(`{
  "name": "@choysum-dev/auth",
  "choysum": { "moduleName": "auth", "application": "auth" }
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	vue := `<template><div /></template>
<script setup lang="ts">
import Button from '@/choy_ui/web/components/vendor/ui/button/Button.vue';
</script>
`
	path := filepath.Join(webDir, "LeakLine.vue")
	if err := os.WriteFile(path, []byte(vue), 0o644); err != nil {
		t.Fatal(err)
	}
	err := AssertNoForbiddenUiImports(modulesPath, "auth")
	if err == nil {
		t.Fatal("expected violation")
	}
	// Import is on file line 3 (1-based).
	if !strings.Contains(err.Error(), "LeakLine.vue:3:") {
		t.Fatalf("expected file-relative line 3 in error, got %v", err)
	}
}

func TestAssertNoForbiddenUiImports_MissingModuleRootErrors(t *testing.T) {
	modulesPath := t.TempDir()
	err := AssertNoForbiddenUiImports(modulesPath, "does-not-exist")
	if err == nil || !strings.Contains(err.Error(), "module root does not exist") {
		t.Fatalf("expected missing module root error, got %v", err)
	}
}

func TestAssertNoForbiddenUiImports_RejectsVueScriptDeepPath(t *testing.T) {
	modulesPath := t.TempDir()
	webDir := filepath.Join(modulesPath, "auth", "web", "pages")
	if err := os.MkdirAll(webDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modulesPath, "auth", "package.json"), []byte(`{
  "name": "@choysum-dev/auth",
  "choysum": { "moduleName": "auth", "application": "auth" }
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	vue := `<template><div /></template>
<script setup lang="ts">
import Button from '@/choy_ui/web/components/vendor/ui/button/Button.vue';
</script>
`
	if err := os.WriteFile(filepath.Join(webDir, "Leak.vue"), []byte(vue), 0o644); err != nil {
		t.Fatal(err)
	}
	err := AssertNoForbiddenUiImports(modulesPath, "auth")
	if err == nil || !strings.Contains(err.Error(), "ui/*") {
		t.Fatalf("expected ui/* ban from vue script, got %v", err)
	}
}

func TestAssertNoForbiddenUiImports_AllowsChoyUIModule(t *testing.T) {
	modulesPath := t.TempDir()
	webDir := filepath.Join(modulesPath, "choy_ui", "web", "components", "vendor", "ui")
	if err := os.MkdirAll(webDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modulesPath, "choy_ui", "package.json"), []byte(`{
  "name": "@choysum-dev/choy_ui",
  "choysum": { "moduleName": "choy_ui", "application": "choy_ui" }
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	src := "import { DialogRoot } from 'reka-ui';\nexport const x = DialogRoot;\n"
	if err := os.WriteFile(filepath.Join(webDir, "Dialog.ts"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := AssertNoForbiddenUiImports(modulesPath, "choy_ui"); err != nil {
		t.Fatalf("choy_ui must be exempt: %v", err)
	}
}

func TestAssertNoForbiddenUiImports_AllowsCleanDomainWeb(t *testing.T) {
	modulesPath := t.TempDir()
	webDir := filepath.Join(modulesPath, "partner", "web")
	if err := os.MkdirAll(webDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modulesPath, "partner", "package.json"), []byte(`{
  "name": "@choysum-dev/partner",
  "choysum": { "moduleName": "partner", "application": "partner" }
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	src := "import { ref } from 'vue';\nexport const n = ref(1);\n"
	if err := os.WriteFile(filepath.Join(webDir, "ok.ts"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := AssertNoForbiddenUiImports(modulesPath, "partner"); err != nil {
		t.Fatalf("clean web should pass: %v", err)
	}
}

func TestAssertNoForbiddenUiImports_RealModulesClean(t *testing.T) {
	modulesPath := findRepoModulesDirForPolicy(t)
	entries, err := os.ReadDir(modulesPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if !entry.IsDir() || shouldSkipModulesDirEntry(entry.Name()) {
			continue
		}
		name := entry.Name()
		web := filepath.Join(modulesPath, name, "web")
		if st, err := os.Stat(web); err != nil || !st.IsDir() {
			continue
		}
		if err := AssertNoForbiddenUiImports(modulesPath, name); err != nil {
			t.Errorf("module %s: %v", name, err)
		}
	}
}

func findRepoModulesDirForPolicy(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := wd
	for i := 0; i < 10; i++ {
		candidate := filepath.Join(dir, "modules", "partner", "package.json")
		if _, err := os.Stat(candidate); err == nil {
			return filepath.Join(dir, "modules")
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatal("modules/ not found from test cwd")
	return ""
}

func TestIsModuleWebSource(t *testing.T) {
	root := "/modules/partner"
	if !IsModuleWebSource(root, "/modules/partner/web/pages/X.vue") {
		t.Fatal("expected web source")
	}
	if IsModuleWebSource(root, "/modules/partner/service/models/x.ts") {
		t.Fatal("service must not count as web")
	}
}
