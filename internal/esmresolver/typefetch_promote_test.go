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
