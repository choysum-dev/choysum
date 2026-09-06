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
		{File: "/a.vue", Code: 2322, Message: "Type '{ onClick: () => void; }' is not assignable to type 'NonNullable<...>'"},
		{File: "/a.vue", Code: 2339, Message: "Property 'default' does not exist on type '{}'."},
		{File: "/a.vue", Code: 1000, Message: "real error"},
	}
	got := suppressVueTemplateParityNoise(diags)
	if len(got) != 2 {
		t.Fatalf("got %d %#v", len(got), got)
	}
	if got[0].File != "/a.ts" || got[1].Code != 1000 {
		t.Fatalf("%#v", got)
	}
}
