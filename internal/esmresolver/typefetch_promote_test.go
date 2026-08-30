// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package esmresolver

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFindMatchingBraceAndIsOnlyComments(t *testing.T) {
	if findMatchingBrace("nope", 0) != -1 || findMatchingBrace("{", 0) != -1 {
		t.Fatal("expected unmatched")
	}
	src := "{ a: '{', b: \"}\", d: /* } */ // }\n  e: 1 }"
	end := findMatchingBrace(src, 0)
	if end != len(src)-1 {
		t.Fatalf("end=%d want %d src=%q", end, len(src)-1, src)
	}
	if !isOnlyComments("  // a\n /* b */ ") {
		t.Fatal("comment-only should be true")
	}
	if isOnlyComments("/* unclosed") || isOnlyComments("code") {
		t.Fatal("non-comment-only should be false")
	}
}

func TestPromoteAmbientModuleForPathsTarget_ExportEquals(t *testing.T) {
	content := `declare module 'https://esm.sh/fast-deep-equal@3.1.3/index.d.ts' {
    const equal: (a: any, b: any) => boolean;
    export = equal;
}
`
	got := promoteAmbientModuleForPathsTarget(content)
	if strings.Contains(got, "declare module") {
		t.Fatalf("expected ambient wrapper to be promoted, got %q", got)
	}
	if !strings.Contains(got, "declare const equal") {
		t.Fatalf("expected declare const after promotion, got %q", got)
	}
	if !strings.Contains(got, "export = equal") {
		t.Fatalf("expected export = after promotion, got %q", got)
	}
}

func TestPromoteAmbientModuleForPathsTarget_StripsAsyncOnDeclare(t *testing.T) {
	content := `declare module 'https://esm.sh/example@1.0.0/index.d.ts' {
  async function run(): Promise<void>;
  export { run };
}
`
	got := promoteAmbientModuleForPathsTarget(content)
	if strings.Contains(got, "declare async") {
		t.Fatalf("ambient declare must not keep async, got %q", got)
	}
	if !strings.Contains(got, "declare function run") {
		t.Fatalf("expected declare function after stripping async, got %q", got)
	}
}

func TestPromoteAmbientModuleForPathsTarget_IgnoresBlockCommentTextForMinIndent(t *testing.T) {
	content := `declare module 'https://esm.sh/example@1.0.0/index.d.ts' {
/*
Descriptive free text without a leading star.
*/
  function run(): void;
  export { run };
}
`
	got := promoteAmbientModuleForPathsTarget(content)
	if !strings.Contains(got, "declare function run") {
		t.Fatalf("expected top-level function to get declare despite block-comment text, got %q", got)
	}
}

func TestPromoteAmbientModuleForPathsTarget_SkipsNestedDecls(t *testing.T) {
	content := `declare module 'https://esm.sh/example@1.0.0/index.d.ts' {
  namespace Helpers {
    function inner(): void;
    const value: number;
  }
  class Box {
    method(): void;
  }
  export { Helpers, Box };
}
`
	got := promoteAmbientModuleForPathsTarget(content)
	if strings.Contains(got, "declare function inner") {
		t.Fatalf("nested function must not get declare, got %q", got)
	}
	if strings.Contains(got, "declare const value") {
		t.Fatalf("nested const must not get declare, got %q", got)
	}
	if !strings.Contains(got, "declare namespace Helpers") {
		t.Fatalf("expected top-level namespace to be declared, got %q", got)
	}
	if !strings.Contains(got, "declare class Box") {
		t.Fatalf("expected top-level class to be declared, got %q", got)
	}
}

func TestPromoteAmbientModuleForPathsTarget_SkipsAugmentationOnly(t *testing.T) {
	content := `declare module 'https://esm.sh/pinia@3.0.4/dist/pinia.d.ts' {
  interface DefineStoreOptionsBase<S, Store> {
    persist?: boolean
  }
}
`
	got := promoteAmbientModuleForPathsTarget(content)
	if got != content {
		t.Fatalf("expected augmentation-only module to remain unchanged")
	}
}

func TestPromoteAmbientModuleForPathsTarget_SkipsMultiModule(t *testing.T) {
	content := `declare module 'https://esm.sh/a@1.0.0/index.d.ts' {
  export const a: number;
}
declare module 'https://esm.sh/b@1.0.0/index.d.ts' {
  export const b: number;
}
`
	got := promoteAmbientModuleForPathsTarget(content)
	if got != content {
		t.Fatalf("expected multi-module file to remain unchanged")
	}
}

func TestPromoteAmbientModuleForPathsTarget_AlreadyModule(t *testing.T) {
	content := `export default function equal(a: any, b: any): boolean;
`
	got := promoteAmbientModuleForPathsTarget(content)
	if got != content {
		t.Fatalf("expected real module to remain unchanged")
	}
}

func TestSplitBarePackageSpecifier(t *testing.T) {
	tests := []struct {
		in      string
		root    string
		subpath string
		ok      bool
	}{
		{in: "echarts", root: "echarts", subpath: "", ok: true},
		{in: "echarts/core", root: "echarts", subpath: "core", ok: true},
		{in: "echarts/charts", root: "echarts", subpath: "charts", ok: true},
		{in: "@scope/pkg", root: "@scope/pkg", subpath: "", ok: true},
		{in: "@scope/pkg/sub", root: "@scope/pkg", subpath: "sub", ok: true},
		{in: "@scope/pkg/a/b", root: "@scope/pkg", subpath: "a/b", ok: true},
		{in: "./local", ok: false},
		{in: "@/alias", ok: false},
		{in: "node:fs", ok: false},
	}
	for _, tt := range tests {
		root, sub, ok := splitBarePackageSpecifier(tt.in)
		if ok != tt.ok || root != tt.root || sub != tt.subpath {
			t.Fatalf("splitBarePackageSpecifier(%q) = (%q,%q,%v), want (%q,%q,%v)",
				tt.in, root, sub, ok, tt.root, tt.subpath, tt.ok)
		}
	}
}

func TestIsBarePackageImportSpecifier(t *testing.T) {
	tests := []struct {
		in string
		ok bool
	}{
		{in: "vue", ok: true},
		{in: "@scope/pkg", ok: true},
		{in: "./local", ok: false},
		{in: "../up", ok: false},
		{in: "lodash/../evil", ok: false},
		{in: "@/alias", ok: false},
		{in: "~/components/Button", ok: false},
		{in: "#internal", ok: false},
		{in: "node:fs", ok: false},
	}
	for _, tt := range tests {
		if got := isBarePackageImportSpecifier(tt.in); got != tt.ok {
			t.Fatalf("isBarePackageImportSpecifier(%q) = %v, want %v", tt.in, got, tt.ok)
		}
	}
}

func TestTypeFetchDiscoverSpec(t *testing.T) {
	tests := []struct {
		specifier string
		version   string
		want      string
	}{
		{specifier: "echarts", version: "6.1.0", want: "echarts@6.1.0"},
		{specifier: "echarts/core", version: "6.1.0", want: "echarts@6.1.0/core"},
		{specifier: "@vue/test-utils", version: "2.4.11", want: "@vue/test-utils@2.4.11"},
		{specifier: "@vue/test-utils/dist/vue-test-utils.esm-browser", version: "2.4.11", want: "@vue/test-utils@2.4.11/dist/vue-test-utils.esm-browser"},
		{specifier: "", version: "1.0.0", want: ""},
		{specifier: "echarts", version: "", want: "echarts"},
		{specifier: "echarts/core", version: "", want: "echarts/core"},
		{specifier: "not a package!!", version: "1.2.3", want: "not a package!!@1.2.3"},
		{specifier: "not a package!!", version: "", want: "not a package!!"},
	}
	for _, tt := range tests {
		got := typeFetchDiscoverSpec(tt.specifier, tt.version)
		if got != tt.want {
			t.Fatalf("typeFetchDiscoverSpec(%q,%q) = %q, want %q", tt.specifier, tt.version, got, tt.want)
		}
	}
}

func TestSubpathTypeFetchTargets(t *testing.T) {
	deps := map[string]string{"echarts": "^6.1.0", "vue": "^3.5.35", "dayjs": "^1.11.21"}
	imports := []string{
		"echarts/core",
		"echarts/charts",
		"vue",
		"lodash/get",
		"echarts/core",
		"dayjs/locale/zh-cn",
		"element-plus/dist/index.css",
	}
	got := subpathTypeFetchTargets(deps, imports)
	if len(got) != 2 {
		t.Fatalf("targets = %#v, want 2 echarts subpaths", got)
	}
	if got[0].name != "echarts/charts" || got[0].version != "6.1.0" {
		t.Fatalf("first target = %#v", got[0])
	}
	if got[1].name != "echarts/core" || got[1].version != "6.1.0" {
		t.Fatalf("second target = %#v", got[1])
	}
}

func TestIsAssetLikeImportSpecifier(t *testing.T) {
	if !isAssetLikeImportSpecifier("element-plus/dist/index.css") {
		t.Fatal("expected css import to be treated as asset")
	}
	if isAssetLikeImportSpecifier("echarts/core") {
		t.Fatal("did not expect echarts/core to be treated as asset")
	}
}

func TestShouldSkipTypeFetchSubpathSpecifier(t *testing.T) {
	if !shouldSkipTypeFetchSubpathSpecifier("dayjs/locale/zh-cn") {
		t.Fatal("expected dayjs locale subpath to be skipped")
	}
	if shouldSkipTypeFetchSubpathSpecifier("echarts/core") {
		t.Fatal("did not expect echarts/core to be skipped")
	}
}

func TestIsStaleGeneratedTsconfigPathsKey(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{in: "@/*", want: false},
		{in: "vue", want: false},
		{in: "@vicons/material", want: false},
		{in: "echarts/core", want: false},
		{in: "@vicons/material@0.13.0/es/AbcFilled.d.ts", want: true},
		{in: "vue@3.5.40/dist/vue.d.mts", want: true},
		{in: "https://esm.sh/vue", want: true},
	}
	for _, tt := range tests {
		if got := isStaleGeneratedTsconfigPathsKey(tt.in); got != tt.want {
			t.Fatalf("isStaleGeneratedTsconfigPathsKey(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestIsValidTsconfigPathsMappingKey(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{in: "echarts", want: true},
		{in: "echarts/core", want: true},
		{in: "@vue/test-utils", want: true},
		{in: "@vue/test-utils/dist/x", want: true},
		{in: "echarts@6.0.0/core.d.ts", want: false},
		{in: "vue@3.5.35/dist/vue.d.mts", want: false},
		{in: "https://esm.sh/vue", want: false},
		{in: "", want: false},
	}
	for _, tt := range tests {
		if got := isValidTsconfigPathsMappingKey(tt.in); got != tt.want {
			t.Fatalf("isValidTsconfigPathsMappingKey(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestCollectSourceBareImportSpecifiers(t *testing.T) {
	content := `
import type { EChartsOption } from 'echarts';
import { use } from 'echarts/core';
import { BarChart } from "echarts/charts";
import equal from 'fast-deep-equal';
import local from './local';
import alias from '@/alias';
export { TitleComponent } from 'echarts/components';
const dyn = import('echarts/renderers');
type X = import('vue').Ref;
`
	got := collectSourceBareImportSpecifiers(content, "chart.ts")
	want := map[string]bool{
		"echarts":            true,
		"echarts/core":       true,
		"echarts/charts":     true,
		"fast-deep-equal":    true,
		"echarts/components": true,
		"echarts/renderers":  true,
		"vue":                true,
	}
	if len(got) != len(want) {
		t.Fatalf("got %#v, want keys %#v", got, want)
	}
	for _, spec := range got {
		if !want[spec] {
			t.Fatalf("unexpected specifier %q in %#v", spec, got)
		}
	}
}

func TestCollectModuleSourceImportSpecifiers(t *testing.T) {
	dir := t.TempDir()
	srcDir := filepath.Join(dir, "web", "components")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "chart.ts"), []byte("import { use } from 'echarts/core';\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "node_modules", "echarts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "node_modules", "echarts", "index.ts"), []byte("import x from 'should-not-see';\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := collectModuleSourceImportSpecifiers(dir)
	if err != nil {
		t.Fatalf("collectModuleSourceImportSpecifiers: %v", err)
	}
	if len(got) != 1 || got[0] != "echarts/core" {
		t.Fatalf("got %#v, want [echarts/core]", got)
	}
}

func TestUpdateTsconfigPaths_SkipsVersionedTransitiveKeys(t *testing.T) {
	dir := t.TempDir()
	tsconfigPath := filepath.Join(dir, "tsconfig.json")
	typesDir := filepath.Join(dir, "types")
	if err := os.MkdirAll(typesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	valid := filepath.Join(typesDir, "echarts-core.d.ts")
	junk := filepath.Join(typesDir, "echarts-versioned.d.ts")
	if err := os.WriteFile(valid, []byte("export {};\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(junk, []byte("export {};\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	results := []TypeFetchResult{
		{Package: "echarts/core", Version: "6.1.0", CachedPath: valid},
		{Package: "echarts@6.0.0/core.d.ts", Version: "", CachedPath: junk},
	}
	if err := UpdateTsconfigPaths(tsconfigPath, results); err != nil {
		t.Fatalf("UpdateTsconfigPaths: %v", err)
	}
	data, err := os.ReadFile(tsconfigPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, `"echarts/core"`) {
		t.Fatalf("expected echarts/core path, got %s", content)
	}
	if strings.Contains(content, "echarts@6.0.0/core.d.ts") {
		t.Fatalf("did not expect versioned transitive path key, got %s", content)
	}
}

func TestUpdateTsconfigPaths_PrunesStaleVersionedKeys(t *testing.T) {
	dir := t.TempDir()
	tsconfigPath := filepath.Join(dir, "tsconfig.json")
	initial := `{
  "compilerOptions": {
    "paths": {
      "@/*": ["./*"],
      "@vicons/material": ["types/vicons-index.d.ts"],
      "@vicons/material@0.13.0/es/AbcFilled.d.ts": ["types/AbcFilled.d.ts"],
      "vue@3.5.40/dist/vue.d.mts": ["types/vue.d.mts"]
    }
  }
}`
	if err := os.WriteFile(tsconfigPath, []byte(initial), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := UpdateTsconfigPaths(tsconfigPath, nil); err != nil {
		t.Fatalf("UpdateTsconfigPaths: %v", err)
	}
	data, err := os.ReadFile(tsconfigPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, `"@/*"`) {
		t.Fatalf("expected @/* alias to remain, got %s", content)
	}
	if !strings.Contains(content, `"@vicons/material"`) {
		t.Fatalf("expected bare @vicons/material to remain, got %s", content)
	}
	if strings.Contains(content, "AbcFilled.d.ts") {
		t.Fatalf("expected per-icon path key to be pruned, got %s", content)
	}
	if strings.Contains(content, "vue@3.5.40") {
		t.Fatalf("expected versioned vue path key to be pruned, got %s", content)
	}
}

func TestFetchTypeDefinition_PromotesAmbientCJSWrapper(t *testing.T) {
	typesURLPath := "/fast-deep-equal@3.1.3/index.d.ts"
	body := `declare module 'https://esm.sh/fast-deep-equal@3.1.3/index.d.ts' {
    const equal: (a: any, b: any) => boolean;
    export = equal;
}
`
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodHead && strings.HasPrefix(r.URL.Path, "/fast-deep-equal@3.1.3") && r.URL.RawQuery == "dts":
			w.Header().Set("x-typescript-types", srv.URL+typesURLPath)
			w.WriteHeader(http.StatusOK)
		case r.URL.Path == typesURLPath:
			w.Header().Set("Content-Type", "application/typescript")
			_, _ = w.Write([]byte(body))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	typesDir := t.TempDir()
	result, _, err := FetchTypeDefinition(srv.Client(), srv.URL, typesDir, "fast-deep-equal", "3.1.3")
	if err != nil {
		t.Fatalf("FetchTypeDefinition: %v", err)
	}
	if result == nil {
		t.Fatal("expected result")
	}
	data, err := os.ReadFile(result.CachedPath)
	if err != nil {
		t.Fatalf("read cache: %v", err)
	}
	content := string(data)
	if strings.Contains(content, "declare module") {
		t.Fatalf("expected promoted module, got %q", content)
	}
	if !strings.Contains(content, "declare const equal") || !strings.Contains(content, "export = equal") {
		t.Fatalf("unexpected promoted content: %q", content)
	}
}

func TestFetchTypesForModule_IncludesSubpathFromSource(t *testing.T) {
	moduleDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(moduleDir, "package.json"), []byte(`{"peerDependencies":{"echarts":"^6.1.0"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	srcDir := filepath.Join(moduleDir, "web")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "chart.ts"), []byte("import { use } from 'echarts/core';\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodHead && r.URL.Path == "/echarts@6.1.0" && r.URL.RawQuery == "dts":
			w.Header().Set("x-typescript-types", srv.URL+"/echarts@6.1.0/types/dist/all.d.ts")
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodHead && r.URL.Path == "/echarts@6.1.0/core" && r.URL.RawQuery == "dts":
			w.Header().Set("x-typescript-types", srv.URL+"/echarts@6.1.0/core.d.ts")
			w.WriteHeader(http.StatusOK)
		case r.URL.Path == "/echarts@6.1.0/types/dist/all.d.ts":
			_, _ = w.Write([]byte("export declare const init: any;\n"))
		case r.URL.Path == "/echarts@6.1.0/core.d.ts":
			_, _ = w.Write([]byte("export declare const use: (...args: any[]) => void;\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	typesDir := t.TempDir()
	results, stats, err := FetchTypesForModuleWithStats(srv.Client(), srv.URL, typesDir, moduleDir)
	if err != nil {
		t.Fatalf("FetchTypesForModuleWithStats: %v", err)
	}
	if stats.DirectTargets < 2 {
		t.Fatalf("DirectTargets = %d, want at least bare + subpath", stats.DirectTargets)
	}
	found := map[string]bool{}
	for _, r := range results {
		found[r.Package] = true
	}
	if !found["echarts"] {
		t.Fatalf("missing echarts result in %#v", results)
	}
	if !found["echarts/core"] {
		t.Fatalf("missing echarts/core result in %#v", results)
	}
}
