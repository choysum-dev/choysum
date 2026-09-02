// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package parser

import (
	"path/filepath"
	"testing"
)

func TestParseSideEffectImport_TsParserPath(t *testing.T) {
	path := "/virtual/modules/partner/service/models/partner.ts"
	content := `import '@/auth/service/models/role';`
	_, ctx := mustParseTSGoCtx(t, path, content)
	imp := ctx.imports[SideEffectImportKey]
	if imp == nil || imp.IsTypeOnly {
		t.Fatalf("side-effect import = %#v", imp)
	}
	if got := imp.ModuleSpecPath; got != "/virtual/modules/test/auth/service/models/role" {
		t.Fatalf("ModuleSpecPath = %q", got)
	}
}

func TestParseEmptyNamedImport_TsParserPath(t *testing.T) {
	path := "/virtual/modules/partner/service/models/partner.ts"
	content := `import {} from '@/core/service/api/dial';`
	_, ctx := mustParseTSGoCtx(t, path, content)
	imp := ctx.imports[SideEffectImportKey]
	if imp == nil || imp.IsTypeOnly {
		t.Fatalf("empty named import = %#v", imp)
	}
	if got := imp.ModuleSpecPath; got != "/virtual/modules/test/core/service/api/dial" {
		t.Fatalf("ModuleSpecPath = %q", got)
	}
}

func TestParseExportTypeOnly_TsParserPath(t *testing.T) {
	path := "/virtual/modules/partner/service/models/partner.ts"
	content := `export type { Role } from '@/auth/service/models/role';`
	_, ctx := mustParseTSGoCtx(t, path, content)
	exp := ctx.exports["Role"]
	if exp == nil {
		t.Fatal("expected Role export")
	}
	if !exp.IsTypeOnly {
		t.Fatalf("export IsTypeOnly=false, want true: %#v", exp)
	}
}

func TestParseExportAssignmentPropertyAccess_TsParserPath(t *testing.T) {
	path := "/virtual/modules/partner/service/models/partner.ts"
	content := `
import dial from '@/core/service/api/dial';
export default dial.create;
`
	_, ctx := mustParseTSGoCtx(t, path, content)
	exp := ctx.exports["default"]
	if exp == nil {
		t.Fatal("expected default export")
	}
	if exp.ReferenceIdent != "create" {
		t.Fatalf("ReferenceIdent = %q, want create", exp.ReferenceIdent)
	}
}

func TestParseNamespaceExport_TsParserPath(t *testing.T) {
	path := "/virtual/modules/partner/service/models/partner.ts"
	content := `export * as auth from '@/auth/service/models/role';`
	_, ctx := mustParseTSGoCtx(t, path, content)
	exp := ctx.exports["auth"]
	if exp == nil || exp.IsTypeOnly {
		t.Fatalf("namespace export = %#v", exp)
	}
}

func TestParseExportAssignmentFallback_TsParserPath(t *testing.T) {
	path := "/virtual/modules/partner/service/models/partner.ts"
	content := `export default (1 as unknown);`
	_, ctx := mustParseTSGoCtx(t, path, content)
	exp := ctx.exports["default"]
	if exp == nil {
		t.Fatal("expected default export")
	}
	if exp.ModuleSpecPath == "" {
		t.Fatalf("default export = %#v", exp)
	}
}

func TestExportBindingIsTypeOnly_SpecifierBranch(t *testing.T) {
	path := "/virtual/modules/partner/service/models/partner.ts"
	content := `export { type Role } from '@/auth/service/models/role';`
	_, ctx := mustParseTSGoCtx(t, path, content)
	exp := ctx.exports["Role"]
	if exp == nil || !exp.IsTypeOnly {
		t.Fatalf("inline export type = %#v", exp)
	}
}

func TestParseExportTypeWildcard_TsParserPath(t *testing.T) {
	path := "/virtual/modules/partner/service/models/partner.ts"
	content := `export type * from '@/auth/service/models/role';`
	_, ctx := mustParseTSGoCtx(t, path, content)
	wildcard := ctx.exports["*"]
	if wildcard == nil || len(wildcard.Wildcard) != 1 {
		t.Fatalf("wildcard export = %#v", wildcard)
	}
	if !wildcard.IsTypeOnly || !wildcard.Wildcard[0].IsTypeOnly {
		t.Fatalf("export type wildcard IsTypeOnly=false: %#v", wildcard)
	}
}

func TestParseDynamicImport_TemplateLiteral(t *testing.T) {
	path := "/virtual/modules/auth/service/tests/observability.test.ts"
	content := "await import(`@/meta/service/models/ui_resource`);\n"
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
	want := filepath.Join("/virtual/modules/meta/service/models/ui_resource")
	if got := p.DynamicImports[0].ModuleSpecPath; got != want {
		t.Fatalf("ModuleSpecPath = %q, want %q", got, want)
	}
}
