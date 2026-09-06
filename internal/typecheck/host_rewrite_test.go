// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package typecheck

import (
	"strings"
	"testing"
)

func TestRewriteEsmShDeclareModules(t *testing.T) {
	in := `
declare module 'https://esm.sh/@vue/runtime-core@3.5.35/dist/runtime-core.d.ts' {
  export interface GlobalComponents {}
}
declare module 'https://esm.sh/vue@3.5.35/dist/vue.d.mts' {
  interface GlobalComponents {}
}
declare module 'https://esm.sh/dayjs@1.11.21/locale/*' {
  const locale: any;
  export = locale;
}
declare module '@other' {}
`
	got := rewriteEsmShDeclareModules(in)
	if !strings.Contains(got, "declare module '@vue/runtime-core'") {
		t.Fatalf("runtime-core: %s", got)
	}
	if !strings.Contains(got, "declare module 'vue'") {
		t.Fatalf("vue: %s", got)
	}
	if !strings.Contains(got, "declare module 'dayjs/locale/*'") {
		t.Fatalf("dayjs locale subpath: %s", got)
	}
	if strings.Contains(got, "declare module 'dayjs' {") && strings.Contains(got, "export = locale") {
		t.Fatalf("must not map locale/* onto bare dayjs: %s", got)
	}
	if strings.Contains(got, "https://esm.sh/") {
		t.Fatalf("leftover esm.sh module id: %s", got)
	}
	if rewriteEsmShDeclareModules("no rewrite") != "no rewrite" {
		t.Fatal("passthrough")
	}

	dq := `declare module "https://esm.sh/vue@3.5.35/dist/vue.d.mts" { export {} }`
	gotDQ := rewriteEsmShDeclareModules(dq)
	if !strings.Contains(gotDQ, "declare module 'vue'") {
		t.Fatalf("double-quote rewrite: %s", gotDQ)
	}
}

func TestRewriteEsmShDeclareModules_DoubleQuotes(t *testing.T) {
	in := `declare module "https://esm.sh/vue@3.5.35/" {
  export const x: number;
}
declare module "https://esm.sh/lodash@4.17.21" {
  const _: any;
  export = _;
}
declare module "https://esm.sh/dayjs@1.11.21/plugin/utc" {
  const p: any;
  export default p;
}
`
	got := rewriteEsmShDeclareModules(in)
	if !strings.Contains(got, "declare module 'vue'") {
		t.Fatalf("trailing slash sub=/ → bare vue: %s", got)
	}
	if !strings.Contains(got, "declare module 'lodash'") {
		t.Fatalf("empty sub → lodash: %s", got)
	}
	if !strings.Contains(got, "declare module 'dayjs/plugin/utc'") {
		t.Fatalf("subpath kept: %s", got)
	}
	if strings.Contains(got, "https://esm.sh/") {
		t.Fatalf("leftover: %s", got)
	}
}

func TestEsmShURLToModuleID(t *testing.T) {
	if got := esmShURLToModuleID("vue", ""); got != "vue" {
		t.Fatalf("empty sub: %q", got)
	}
	if got := esmShURLToModuleID("vue", "/"); got != "vue" {
		t.Fatalf("slash-only: %q", got)
	}
	if got := esmShURLToModuleID("vue", "/dist/vue.d.ts"); got != "vue" {
		t.Fatalf("main type path: %q", got)
	}
	if got := esmShURLToModuleID("@vue/runtime-core", "/dist/runtime-core.d.mts"); got != "@vue/runtime-core" {
		t.Fatalf("scoped main: %q", got)
	}
	if got := esmShURLToModuleID("vue", "/index.d.ts"); got != "vue" {
		t.Fatalf("index main: %q", got)
	}
	if got := esmShURLToModuleID("dayjs", "/locale/*"); got != "dayjs/locale/*" {
		t.Fatalf("wildcard sub: %q", got)
	}
	if got := esmShURLToModuleID("dayjs", "/plugin/utc.d.ts"); got != "dayjs/plugin/utc" {
		t.Fatalf("nested .d.ts strip: %q", got)
	}
}

func TestRewriteEsmShDeclareModules_PassthroughUnmatched(t *testing.T) {
	// Contains both markers so the fast-path does not bail, but the URL is not esm.sh-shaped
	// for the capture groups (no closing quote pattern the RE expects after a valid pkg).
	in := `declare module 'https://esm.sh/' { }`
	got := rewriteEsmShDeclareModules(in)
	if !strings.Contains(got, "https://esm.sh/") {
		t.Fatalf("unmatched should stay: %s", got)
	}
}

func TestIsEsmShTypeFetchPath(t *testing.T) {
	if !isEsmShTypeFetchPath("/Users/me/.choysum/pkg/types/esm.sh_vue@1.d.ts") {
		t.Fatal("home types")
	}
	if isEsmShTypeFetchPath("/Users/me/choysum/modules/auth/web/App.vue") {
		t.Fatal("app path")
	}
}

func TestFilterDiagnosticsToApp(t *testing.T) {
	diags := []Diagnostic{
		{File: "/repo/modules/auth/web/A.vue", Code: 1},
		{File: "/repo/modules/web/web/B.vue", Code: 2},
		{File: "/repo/modules/auth/service/x.ts", Code: 3},
	}
	got := filterDiagnosticsToApp(diags, "/repo/modules", "auth")
	if len(got) != 2 {
		t.Fatalf("got %d %#v", len(got), got)
	}
}

func TestSuppressVueTemplateParityNoise(t *testing.T) {
	diags := []Diagnostic{
		{File: "/a.vue", Code: 2339, Message: "Property '$el' does not exist"},
		{File: "/a.vue", Code: 7031, Message: "Binding element '$event' implicitly has an 'any' type."},
		{File: "/a.ts", Code: 2339, Message: "Property '$el' does not exist"},
		{File: "/a.vue", Code: 2322, Message: "Type '{ onClick: () => void; }' is not assignable to type 'NonNullable<VNodeProps & ...>'"},
		{File: "/a.vue", Code: 2322, Message: "Type '() => number' is not assignable to type 'NonNullable<((...args: any) => any) | undefined>'."},
		{File: "/a.vue", Code: 2339, Message: "Property 'default' does not exist on type '__VLS_Slots'."},
		{File: "/a.vue", Code: 2339, Message: "Property 'default' does not exist on type '{}'."},
		{File: "/a.vue", Code: 1000, Message: "real error"},
	}
	got := suppressVueTemplateParityNoise(diags)
	// .ts $el kept; NonNullable-without-on kept; .vue default-on-{} suppressed; real 1000 kept.
	if len(got) != 3 {
		t.Fatalf("got %d %#v", len(got), got)
	}
	if got[0].File != "/a.ts" || got[1].Code != 2322 || got[2].Code != 1000 {
		t.Fatalf("%#v", got)
	}
}

func TestIsVueTemplateParityNoise_RemainingCodes(t *testing.T) {
	keep := Diagnostic{File: "/a.vue", Code: 2339, Message: "Property 'foo' does not exist on type 'Bar'."}
	if isVueTemplateParityNoise(keep) {
		t.Fatal("2339 without $el/default/{} must not be noise")
	}
	cases := []Diagnostic{
		{Code: 2493, Message: "Tuple type '[]' of length '0' has no element at index '0'."},
		{Code: 7053, Message: `Element implicitly has an 'any' type because expression of type '""' can't be used to index type.`},
		{Code: 7053, Message: "Element implicitly has an 'any' type because expression of type 'string' can't be used to index type '__VLS_Slots'."},
		{Code: 2552, Message: "Cannot find name '__VLS_asFunctionalElement'. Did you mean '__VLS_asFunctionalComponent'?"},
		{Code: 2589, Message: "Type instantiation is excessively deep and possibly infinite. __VLS_ctx"},
		{Code: 2349, Message: "This expression is not callable. __VLS_asFunctionalComponent"},
		{Code: 18048, Message: "'__VLS_3' is possibly 'undefined'."},
		{Code: 2339, Message: "Property 'expose' does not exist on type '{ attrs: any; slots: __VLS_Slots; emit: any; } | undefined'."},
		{Code: 2339, Message: "Property '_t' does not exist on type '{}'."},
		{Code: 7006, Message: "Parameter 'props' implicitly has an 'any' type."},
		{Code: 7031, Message: "Binding element 'attrs' implicitly has an 'any' type."},
		{Code: 2558, Message: "Expected 0 type arguments, but got 1."},
	}
	for _, d := range cases {
		if !isVueTemplateParityNoise(d) {
			t.Fatalf("expected noise for %#v", d)
		}
	}
	if isVueTemplateParityNoise(Diagnostic{Code: 2493, Message: "other tuple"}) {
		t.Fatal("2493 without Tuple type '[]'")
	}
	if isVueTemplateParityNoise(Diagnostic{Code: 7053, Message: "other"}) {
		t.Fatal("7053 unrelated")
	}
	if isVueTemplateParityNoise(Diagnostic{Code: 2552, Message: "Cannot find name 'x'"}) {
		t.Fatal("2552 unrelated")
	}
	if isVueTemplateParityNoise(Diagnostic{Code: 2589, Message: "Type instantiation is excessively deep and possibly infinite."}) {
		t.Fatal("2589 without __VLS must not be noise")
	}
	if isVueTemplateParityNoise(Diagnostic{Code: 2349, Message: "This expression is not callable."}) {
		t.Fatal("2349 without __VLS must not be noise")
	}
	if isVueTemplateParityNoise(Diagnostic{Code: 9999, Message: "x"}) {
		t.Fatal("default")
	}
}
