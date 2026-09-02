// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package policy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeModulePackageJSON(t *testing.T, modulesPath, moduleName, application string) {
	t.Helper()
	dir := filepath.Join(modulesPath, moduleName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", dir, err)
	}
	content := `{
  "name": "@test/` + moduleName + `",
  "version": "0.0.0-test",
  "choysum": {
    "moduleName": "` + moduleName + `",
    "application": "` + application + `"
  }
}`
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile package.json: %v", err)
	}
}

func TestCheckServiceImportBoundaryOnDisk_RejectsCrossAppImport(t *testing.T) {
	modulesPath := filepath.Join(t.TempDir(), "modules")
	writeModulePackageJSON(t, modulesPath, "partner", "partner")
	writeModulePackageJSON(t, modulesPath, "auth", "auth")

	serviceFile := filepath.Join(modulesPath, "partner", "service", "models", "partner.ts")
	if err := os.MkdirAll(filepath.Dir(serviceFile), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	source := "import Role from '@/auth/service/models/role';\nexport default {};\n"
	if err := os.WriteFile(serviceFile, []byte(source), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	err := CheckServiceImportBoundaryOnDisk(modulesPath, "partner", ModulePathAliasForBoundary(modulesPath))
	if err == nil || !strings.Contains(err.Error(), "cross-app service import boundary violation") {
		t.Fatalf("CheckServiceImportBoundaryOnDisk() error = %v, want boundary violation", err)
	}
	if !strings.Contains(err.Error(), "partner -> auth") {
		t.Fatalf("error = %v, want partner -> auth", err)
	}
}

func TestCheckServiceImportBoundaryOnDisk_RejectsDynamicImport(t *testing.T) {
	modulesPath := filepath.Join(t.TempDir(), "modules")
	writeModulePackageJSON(t, modulesPath, "auth", "auth")
	writeModulePackageJSON(t, modulesPath, "meta", "meta")

	serviceFile := filepath.Join(modulesPath, "auth", "service", "tests", "observability.test.ts")
	if err := os.MkdirAll(filepath.Dir(serviceFile), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	source := "await import('@/meta/service/models/ui_resource');\n"
	if err := os.WriteFile(serviceFile, []byte(source), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	err := CheckServiceImportBoundaryOnDisk(modulesPath, "auth", ModulePathAliasForBoundary(modulesPath))
	if err == nil || !strings.Contains(err.Error(), "dynamic import") {
		t.Fatalf("CheckServiceImportBoundaryOnDisk() error = %v, want dynamic import violation", err)
	}
	if !strings.Contains(err.Error(), "auth -> meta") {
		t.Fatalf("error = %v, want auth -> meta", err)
	}
}

func TestCheckServiceImportBoundaryOnDisk_AllowsCoreImport(t *testing.T) {
	modulesPath := filepath.Join(t.TempDir(), "modules")
	writeModulePackageJSON(t, modulesPath, "partner", "partner")
	writeModulePackageJSON(t, modulesPath, "core", "core")

	serviceFile := filepath.Join(modulesPath, "partner", "service", "models", "partner.ts")
	if err := os.MkdirAll(filepath.Dir(serviceFile), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	source := "import { dial } from '@/core/service/api/dial';\nexport default {};\n"
	if err := os.WriteFile(serviceFile, []byte(source), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if err := CheckServiceImportBoundaryOnDisk(modulesPath, "partner", ModulePathAliasForBoundary(modulesPath)); err != nil {
		t.Fatalf("CheckServiceImportBoundaryOnDisk() error = %v, want nil", err)
	}
}

func TestCheckServiceImportBoundaryOnDisk_AllowsExportType(t *testing.T) {
	modulesPath := filepath.Join(t.TempDir(), "modules")
	writeModulePackageJSON(t, modulesPath, "partner", "partner")
	writeModulePackageJSON(t, modulesPath, "auth", "auth")

	serviceFile := filepath.Join(modulesPath, "partner", "service", "models", "partner.ts")
	if err := os.MkdirAll(filepath.Dir(serviceFile), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	source := "export type { Role } from '@/auth/service/models/role';\n"
	if err := os.WriteFile(serviceFile, []byte(source), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if err := CheckServiceImportBoundaryOnDisk(modulesPath, "partner", ModulePathAliasForBoundary(modulesPath)); err != nil {
		t.Fatalf("CheckServiceImportBoundaryOnDisk() error = %v, want nil", err)
	}
}

func TestCheckServiceImportBoundaryOnDisk_RejectsSideEffectImport(t *testing.T) {
	modulesPath := filepath.Join(t.TempDir(), "modules")
	writeModulePackageJSON(t, modulesPath, "partner", "partner")
	writeModulePackageJSON(t, modulesPath, "auth", "auth")

	serviceFile := filepath.Join(modulesPath, "partner", "service", "models", "partner.ts")
	if err := os.MkdirAll(filepath.Dir(serviceFile), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	source := "import '@/auth/service/models/role';\n"
	if err := os.WriteFile(serviceFile, []byte(source), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	err := CheckServiceImportBoundaryOnDisk(modulesPath, "partner", ModulePathAliasForBoundary(modulesPath))
	if err == nil || !strings.Contains(err.Error(), "partner -> auth") {
		t.Fatalf("CheckServiceImportBoundaryOnDisk() error = %v, want side-effect violation", err)
	}
}

func TestCheckServiceImportBoundaryOnDisk_AllowsImportType(t *testing.T) {
	modulesPath := filepath.Join(t.TempDir(), "modules")
	writeModulePackageJSON(t, modulesPath, "partner", "partner")
	writeModulePackageJSON(t, modulesPath, "auth", "auth")

	serviceFile := filepath.Join(modulesPath, "partner", "service", "models", "partner.ts")
	if err := os.MkdirAll(filepath.Dir(serviceFile), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	source := "import type Role from '@/auth/service/models/role';\nexport default {};\n"
	if err := os.WriteFile(serviceFile, []byte(source), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if err := CheckServiceImportBoundaryOnDisk(modulesPath, "partner", ModulePathAliasForBoundary(modulesPath)); err != nil {
		t.Fatalf("CheckServiceImportBoundaryOnDisk() error = %v, want nil", err)
	}
}

func TestReadModuleApplicationFromPackageJSON(t *testing.T) {
	modulesPath := filepath.Join(t.TempDir(), "modules")
	writeModulePackageJSON(t, modulesPath, "partner_bank", "partner")

	got, err := ReadModuleApplicationFromPackageJSON(modulesPath, "partner_bank")
	if err != nil {
		t.Fatalf("ReadModuleApplicationFromPackageJSON() error = %v", err)
	}
	if got != "partner" {
		t.Fatalf("application = %q, want partner", got)
	}
}
