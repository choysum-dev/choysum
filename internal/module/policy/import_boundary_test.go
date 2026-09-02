// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package policy

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/internal/parser"
)

func testModulesPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "modules")
}

func testLookup() ModuleApplicationLookup {
	return ModuleApplicationLookupFromMap(map[string]string{
		"auth":               "auth",
		"base":               "base",
		"core":               "core",
		"meta":               "meta",
		"partner":            "partner",
		"partner_bank":       "partner",
		"partner_commercial": "partner",
	})
}

func TestCheckServiceImportBoundary_AllowsCoreSameAppAndImportType(t *testing.T) {
	modulesPath := testModulesPath(t)
	moduleRoot := filepath.Join(modulesPath, "partner")
	source := filepath.Join(moduleRoot, "service", "models", "partner.ts")
	coreSpec := filepath.Join(modulesPath, "core", "service", "mixins", "message_thread")
	partnerBankSpec := filepath.Join(modulesPath, "partner_bank", "service", "models", "bank_account")

	violations := CheckServiceImportBoundary(ServiceImportBoundaryInput{
		ModulesPath:       modulesPath,
		ModuleRoot:        moduleRoot,
		SourceApplication: "partner",
		Lookup:            testLookup(),
		ParserResults: []*parser.ParserResult{{
			Path: source,
			Imports: map[string]*parser.Import{
				"MessageThreadModel": {
					ModuleSpecPath: coreSpec,
					ModuleSpecText: "@/core/service/mixins/message_thread",
					IsTypeOnly:     false,
					Line:           1,
					Column:         1,
				},
				"BankAccountModel": {
					ModuleSpecPath: partnerBankSpec,
					ModuleSpecText: "@/partner_bank/service/models/bank_account",
					IsTypeOnly:     false,
					Line:           2,
					Column:         1,
				},
				"UserModel": {
					ModuleSpecPath: filepath.Join(modulesPath, "auth", "service", "models", "user", "user"),
					ModuleSpecText: "@/auth/service/models/user/user",
					IsTypeOnly:     true,
					Line:           3,
					Column:         1,
				},
			},
		}},
	})
	if len(violations) != 0 {
		t.Fatalf("expected no violations, got %#v", violations)
	}
}

func TestCheckServiceImportBoundary_RejectsCrossAppValueImport(t *testing.T) {
	modulesPath := testModulesPath(t)
	moduleRoot := filepath.Join(modulesPath, "partner")
	source := filepath.Join(moduleRoot, "service", "models", "partner.ts")
	authSpec := filepath.Join(modulesPath, "auth", "service", "models", "role")

	violations := CheckServiceImportBoundary(ServiceImportBoundaryInput{
		ModulesPath:       modulesPath,
		ModuleRoot:        moduleRoot,
		SourceApplication: "partner",
		Lookup:            testLookup(),
		ParserResults: []*parser.ParserResult{{
			Path: source,
			Imports: map[string]*parser.Import{
				"Role": {
					ModuleSpecPath: authSpec,
					ModuleSpecText: "@/auth/service/models/role",
					IsTypeOnly:     false,
					Line:           4,
					Column:         1,
				},
			},
		}},
	})
	if len(violations) != 1 {
		t.Fatalf("expected one violation, got %#v", violations)
	}
	v := violations[0]
	if v.SourceApplication != "partner" || v.TargetApplication != "auth" {
		t.Fatalf("unexpected apps: %#v", v)
	}
	if v.Kind != "import" {
		t.Fatalf("kind = %q, want import", v.Kind)
	}

	err := FormatImportBoundaryError(violations)
	if err == nil || !strings.Contains(err.Error(), "partner -> auth") || !strings.Contains(err.Error(), "dial('app.Model')") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckServiceImportBoundary_RejectsDynamicImport(t *testing.T) {
	modulesPath := testModulesPath(t)
	moduleRoot := filepath.Join(modulesPath, "auth")
	source := filepath.Join(moduleRoot, "service", "tests", "observability.test.ts")
	metaSpec := filepath.Join(modulesPath, "meta", "service", "models", "ui_resource")

	violations := CheckServiceImportBoundary(ServiceImportBoundaryInput{
		ModulesPath:       modulesPath,
		ModuleRoot:        moduleRoot,
		SourceApplication: "auth",
		Lookup:            testLookup(),
		ParserResults: []*parser.ParserResult{{
			Path: source,
			DynamicImports: []*parser.Import{{
				ModuleSpecPath: metaSpec,
				ModuleSpecText: "@/meta/service/models/ui_resource",
				IsDynamic:      true,
				Line:           2,
				Column:         1,
			}},
		}},
	})
	if len(violations) != 1 {
		t.Fatalf("expected one dynamic import violation, got %#v", violations)
	}
	if violations[0].Kind != "dynamic import" || violations[0].TargetApplication != "meta" {
		t.Fatalf("unexpected violation: %#v", violations[0])
	}
}

func TestCheckServiceImportBoundary_SkipsNonServiceFiles(t *testing.T) {
	modulesPath := testModulesPath(t)
	moduleRoot := filepath.Join(modulesPath, "partner")
	source := filepath.Join(moduleRoot, "web", "pages", "Partner.vue")
	authSpec := filepath.Join(modulesPath, "auth", "service", "models", "role")

	violations := CheckServiceImportBoundary(ServiceImportBoundaryInput{
		ModulesPath:       modulesPath,
		ModuleRoot:        moduleRoot,
		SourceApplication: "partner",
		Lookup:            testLookup(),
		ParserResults: []*parser.ParserResult{{
			Path: source,
			Imports: map[string]*parser.Import{
				"Role": {ModuleSpecPath: authSpec, IsTypeOnly: false, Line: 1, Column: 1},
			},
		}},
	})
	if len(violations) != 0 {
		t.Fatalf("expected web sources to be skipped, got %#v", violations)
	}
}

func TestDefaultApplicationForModuleName(t *testing.T) {
	if got := DefaultApplicationForModuleName("partner_bank"); got != "partner" {
		t.Fatalf("partner_bank app = %q, want partner", got)
	}
	if got := DefaultApplicationForModuleName("auth"); got != "auth" {
		t.Fatalf("auth app = %q, want auth", got)
	}
}

func TestIsModuleServiceSource(t *testing.T) {
	root := filepath.Join("/virtual/modules/auth")
	if !IsModuleServiceSource(root, filepath.Join(root, "service/models/user.ts")) {
		t.Fatal("expected service/models path to match")
	}
	if IsModuleServiceSource(root, filepath.Join(root, "web/index.ts")) {
		t.Fatal("expected web path to be excluded")
	}
}
