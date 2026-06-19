// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/evanw/esbuild/pkg/api"
)

func TestTsParserParseImportAndExport(t *testing.T) {
	parser := &TsParser{
		Path: "/virtual/modules/test/service/user.ts",
		Content: "import BaseModel from './base';\n" +
			"import * as helpers from '@/shared/helpers';\n" +
			"import { Named as Alias, Inline } from './named';\n" +
			"const Local = helpers.Factory;\n" +
			"export { Alias as ExportedAlias };\n" +
			"export { default as SharedDefault } from '@/shared/defaults';\n" +
			"export * from './wild';\n" +
			"export default Local;\n",
		PathAlias: map[string]string{
			"@/*": "/virtual/modules/test/*",
		},
	}

	if err := parser.ParseImport(nil); err != nil {
		t.Fatalf("ParseImport() error = %v", err)
	}
	if got := parser.ImportsMap["BaseModel"]; got == nil || got.ReferenceIdent != "default" || got.ModuleSpecPath != "/virtual/modules/test/service/base" {
		t.Fatalf("unexpected default import: %#v", got)
	}
	if got := parser.ImportsMap["helpers"]; got == nil || got.ReferenceIdent != "*" || got.ModuleSpecPath != "/virtual/modules/test/shared/helpers" {
		t.Fatalf("unexpected namespace import: %#v", got)
	}
	if got := parser.ImportsMap["Alias"]; got == nil || got.ReferenceIdent != "Named" || got.ModuleSpecPath != "/virtual/modules/test/service/named" {
		t.Fatalf("unexpected aliased import: %#v", got)
	}
	if got := parser.ImportsMap["Inline"]; got == nil || got.ReferenceIdent != "Inline" {
		t.Fatalf("unexpected named import: %#v", got)
	}

	if err := parser.ParseExport(nil); err != nil {
		t.Fatalf("ParseExport() error = %v", err)
	}
	if got := parser.ExportsMap["ExportedAlias"]; got == nil || got.ReferenceIdent != "Named" || got.ModuleSpecPath != "/virtual/modules/test/service/named" {
		t.Fatalf("unexpected re-exported alias: %#v", got)
	}
	if got := parser.ExportsMap["SharedDefault"]; got == nil || got.ReferenceIdent != "default" || got.ModuleSpecPath != "/virtual/modules/test/shared/defaults" {
		t.Fatalf("unexpected default re-export: %#v", got)
	}
	if got := parser.ExportsMap["default"]; got == nil || got.ReferenceIdent != "Local" || got.ModuleSpecPath != "/virtual/modules/test/service/user" {
		t.Fatalf("unexpected default export assignment: %#v", got)
	}
	if got := parser.ExportsMap["*"]; got == nil || len(got.Wildcard) != 1 || got.Wildcard[0].ModuleSpecPath != "/virtual/modules/test/service/wild" {
		t.Fatalf("unexpected wildcard exports: %#v", got)
	}
}

func TestParseTsconfigPathAlias(t *testing.T) {
	rawOptions := &api.BuildOptions{
		AbsWorkingDir: "/workspace/modules",
		TsconfigRaw:   `{"compilerOptions":{"paths":{"@/*":["src/*"],"~/*":["shared/*"],"vue":["../../.choysum/pkg/types/esm.sh_vue@3.5.35_dist_vue.d.mts.d.ts"]}}}`,
	}
	rawAliases, err := ParseTsconfigPathAlias(rawOptions)
	if err != nil {
		t.Fatalf("ParseTsconfigPathAlias(raw) error = %v", err)
	}
	if rawAliases["@/*"] != "/workspace/modules/src/*" || rawAliases["~/*"] != "/workspace/modules/shared/*" {
		t.Fatalf("unexpected raw aliases: %#v", rawAliases)
	}
	if _, ok := rawAliases["vue"]; ok {
		t.Fatalf("expected type-only bare alias to be filtered, got: %#v", rawAliases)
	}

	if _, err := ParseTsconfigPathAlias(&api.BuildOptions{TsconfigRaw: "{"}); err == nil {
		t.Fatal("expected invalid raw tsconfig to fail")
	}

	tempDir := t.TempDir()
	tsconfigPath := filepath.Join(tempDir, "tsconfig.json")
	if err := os.WriteFile(tsconfigPath, []byte(`{"compilerOptions":{"paths":{"#root/*":["pkg/*"]}}}`), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	fileAliases, err := ParseTsconfigPathAlias(&api.BuildOptions{Tsconfig: tsconfigPath})
	if err != nil {
		t.Fatalf("ParseTsconfigPathAlias(file) error = %v", err)
	}
	if fileAliases["#root/*"] != filepath.Join(tempDir, "pkg/*") {
		t.Fatalf("unexpected file aliases: %#v", fileAliases)
	}
}

func TestApplyPathAlias(t *testing.T) {
	aliases := map[string]string{
		"@/*":     "/workspace/src/*",
		"@":       "/workspace/modules",
		"#core":   "/workspace/core/index.ts",
		"plain/*": "/workspace/plain/*",
	}

	if got := ApplyPathAlias(aliases, "@/views/home.ts"); got != "/workspace/src/views/home.ts" {
		t.Fatalf("unexpected wildcard alias result: %s", got)
	}
	if got := ApplyPathAlias(aliases, "#core"); got != "/workspace/core/index.ts" {
		t.Fatalf("unexpected exact alias result: %s", got)
	}
	if got := ApplyPathAlias(aliases, "#core/sub"); got != "#core/sub" {
		t.Fatalf("exact alias should not match prefix path: %s", got)
	}
	if got := ApplyPathAlias(aliases, "other/module.ts"); got != "other/module.ts" {
		t.Fatalf("unexpected passthrough path: %s", got)
	}
	legacyAliases := map[string]string{"@": "/workspace/modules"}
	if got := ApplyPathAlias(legacyAliases, "@/core/web/component/xpath.vue"); got != "/workspace/modules/core/web/component/xpath.vue" {
		t.Fatalf("unexpected legacy @ alias result: %s", got)
	}
}

func TestFindVueComponentFinalChild(t *testing.T) {
	results := []*ParserResult{
		{
			Path: "/app/base/button.vue",
			VueComponent: &meta.IrComponent{
				Name: "AppButton",
			},
		},
		{
			Path: "/app/theme/button.vue",
			VueComponent: &meta.IrComponent{
				Name:    "AppButton",
				Extends: "/app/base/button.vue",
			},
		},
		{
			Path: "/app/brand/button.vue",
			VueComponent: &meta.IrComponent{
				Name:    "AppButton",
				Extends: "/app/theme/button.vue",
			},
		},
		{
			Path: "/app/page.vue",
			VueComponent: &meta.IrComponent{
				Name:    "PageView",
				Extends: "/app/layout.vue",
			},
		},
	}

	if got := FindVueComponentFinalChild(results, "/app/page.vue?vue&type=script", "/app/base/button.vue?vue&type=script"); got != "/app/brand/button.vue" {
		t.Fatalf("unexpected final child path: %s", got)
	}
	if got := FindVueComponentFinalChild(results, "/app/theme/button.vue", "/app/theme/button.vue"); got != "" {
		t.Fatalf("expected cycle/self import guard to return empty, got %s", got)
	}
	if got := FindVueComponentFinalChild(results, "", "/app/base/button.vue"); got != "" {
		t.Fatalf("expected empty component path to return empty, got %s", got)
	}
	if got := FindVueComponentFinalChild(results, "/app/page.vue", "/missing.vue"); got != "" {
		t.Fatalf("expected missing import component to return empty, got %s", got)
	}
}
