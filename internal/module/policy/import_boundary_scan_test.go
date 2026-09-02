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
