// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package parser

import "testing"

func TestParseDynamicImports_TsParserPath(t *testing.T) {
	path := "/virtual/modules/auth/service/tests/observability.test.ts"
	content := `
const mod = await import('@/meta/service/models/ui_resource');
type Company = typeof import('@/base/service/models/company');
const local = import('./local_probe');
`
	_, ctx := mustParseTSGoCtx(t, path, content)

	if len(ctx.dynamicImports) != 2 {
		t.Fatalf("expected 2 dynamic imports, got %d: %#v", len(ctx.dynamicImports), ctx.dynamicImports)
	}

	assertDynamic := func(index int, wantSpec string, wantTypeOnly bool) {
		imp := ctx.dynamicImports[index]
		if imp == nil {
			t.Fatalf("dynamicImports[%d] is nil", index)
		}
		if !imp.IsDynamic {
			t.Fatalf("dynamicImports[%d].IsDynamic=false", index)
		}
		if imp.IsTypeOnly != wantTypeOnly {
			t.Fatalf("dynamicImports[%d].IsTypeOnly=%v want %v", index, imp.IsTypeOnly, wantTypeOnly)
		}
		if imp.ModuleSpecText != wantSpec {
			t.Fatalf("dynamicImports[%d].ModuleSpecText=%q want %q", index, imp.ModuleSpecText, wantSpec)
		}
	}

	assertDynamic(0, "@/meta/service/models/ui_resource", false)
	assertDynamic(1, "./local_probe", false)
}

func TestParseDynamicImports_ParserResult(t *testing.T) {
	path := "/virtual/modules/auth/service/tests/observability.test.ts"
	content := `await import('@/meta/service/models/ui_resource');`
	p := &TsParser{
		Path:      path,
		Content:   content,
		PathAlias: map[string]string{"@/*": "/virtual/modules/*"},
	}
	if err := p.ParseImport(nil); err != nil {
		t.Fatalf("ParseImport() error = %v", err)
	}
	if len(p.DynamicImports) != 1 {
		t.Fatalf("DynamicImports len = %d, want 1", len(p.DynamicImports))
	}
	if got := p.DynamicImports[0].ModuleSpecPath; got != "/virtual/modules/meta/service/models/ui_resource" {
		t.Fatalf("ModuleSpecPath = %q", got)
	}
}
