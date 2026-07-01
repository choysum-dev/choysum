// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package moddeps

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestCollectExternalModuleDependencies(t *testing.T) {
	modulesPath := filepath.Join(t.TempDir(), "modules")

	writeModulePackageJSON(t, modulesPath, "auth", `{
		"dependencies": {
			"@choysum-dev/core": "workspace:*",
			"vue": "^3.5.0"
		},
		"peerDependencies": {
			"pinia": "^3.0.0"
		},
		"choysum": {
			"depends": ["core"]
		}
	}`)
	writeModulePackageJSON(t, modulesPath, "core", `{
		"peerDependencies": {
			"@connectrpc/connect": "^2.1.1"
		}
	}`)

	t.Run("follows choysum depends closure", func(t *testing.T) {
		got, err := CollectExternalModuleDependencies(modulesPath, []string{"auth"}, true)
		if err != nil {
			t.Fatalf("CollectExternalModuleDependencies error: %v", err)
		}
		want := []string{"@connectrpc/connect", "pinia", "vue"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("CollectExternalModuleDependencies() = %#v, want %#v", got, want)
		}
	})

	t.Run("can skip depends traversal", func(t *testing.T) {
		got, err := CollectExternalModuleDependencies(modulesPath, []string{"auth"}, false)
		if err != nil {
			t.Fatalf("CollectExternalModuleDependencies error: %v", err)
		}
		want := []string{"pinia", "vue"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("CollectExternalModuleDependencies() = %#v, want %#v", got, want)
		}
	})

	t.Run("ignores missing module package", func(t *testing.T) {
		got, err := CollectExternalModuleDependencies(modulesPath, []string{"missing-module"}, true)
		if err != nil {
			t.Fatalf("CollectExternalModuleDependencies error: %v", err)
		}
		if len(got) != 0 {
			t.Fatalf("CollectExternalModuleDependencies() = %#v, want empty list", got)
		}
	})
}

func TestMergeRequiredModules(t *testing.T) {
	got := MergeRequiredModules(
		[]string{"vite", " vue", "vite"},
		[]string{"@vue/test-utils", ""},
	)
	want := []string{"@vue/test-utils", "vite", "vue"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("MergeRequiredModules() = %#v, want %#v", got, want)
	}
}

func TestMergeRequiredModules_EdgeCases(t *testing.T) {
	t.Run("no arguments returns nil", func(t *testing.T) {
		got := MergeRequiredModules()
		if got != nil {
			t.Fatalf("MergeRequiredModules() = %#v, want nil", got)
		}
	})

	t.Run("all empty slices returns nil", func(t *testing.T) {
		got := MergeRequiredModules([]string{}, []string{})
		if got != nil {
			t.Fatalf("MergeRequiredModules(empty, empty) = %#v, want nil", got)
		}
	})

	t.Run("filters whitespace-only values", func(t *testing.T) {
		got := MergeRequiredModules([]string{"  ", "\t"})
		if got != nil {
			t.Fatalf("MergeRequiredModules(whitespace) = %#v, want nil", got)
		}
	})
}

func TestCollectExternalModuleDependencies_EdgeCases(t *testing.T) {
	t.Run("empty modulesPath returns nil", func(t *testing.T) {
		got, err := CollectExternalModuleDependencies("", []string{"auth"}, true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != nil {
			t.Fatalf("expected nil for empty modulesPath, got %#v", got)
		}
	})

	t.Run("whitespace modulesPath returns nil", func(t *testing.T) {
		got, err := CollectExternalModuleDependencies("  ", []string{"auth"}, true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != nil {
			t.Fatalf("expected nil for whitespace modulesPath, got %#v", got)
		}
	})

	t.Run("empty moduleNames returns nil", func(t *testing.T) {
		modulesPath := filepath.Join(t.TempDir(), "modules")
		writeModulePackageJSON(t, modulesPath, "auth", `{}`)
		got, err := CollectExternalModuleDependencies(modulesPath, nil, true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != nil {
			t.Fatalf("expected nil for empty moduleNames, got %#v", got)
		}
	})

	t.Run("module with no external dependencies returns nil", func(t *testing.T) {
		modulesPath := filepath.Join(t.TempDir(), "modules")
		writeModulePackageJSON(t, modulesPath, "auth", `{}`)
		got, err := CollectExternalModuleDependencies(modulesPath, []string{"auth"}, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != nil {
			t.Fatalf("expected nil for module with no deps, got %#v", got)
		}
	})

	t.Run("module with only workspace dependencies returns nil", func(t *testing.T) {
		modulesPath := filepath.Join(t.TempDir(), "modules")
		writeModulePackageJSON(t, modulesPath, "auth", `{
			"dependencies": {
				"@choysum-dev/core": "workspace:*"
			},
			"peerDependencies": {
				"@choysum-dev/base": "workspace:*"
			}
		}`)
		got, err := CollectExternalModuleDependencies(modulesPath, []string{"auth"}, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != nil {
			t.Fatalf("expected nil for workspace-only deps, got %#v", got)
		}
	})

	t.Run("invalid JSON in package.json returns error", func(t *testing.T) {
		modulesPath := filepath.Join(t.TempDir(), "modules")
		moduleDir := filepath.Join(modulesPath, "auth")
		if err := os.MkdirAll(moduleDir, 0o755); err != nil {
			t.Fatalf("mkdir auth: %v", err)
		}
		if err := os.WriteFile(filepath.Join(moduleDir, "package.json"), []byte("{invalid}"), 0o644); err != nil {
			t.Fatalf("write invalid package.json: %v", err)
		}
		_, err := CollectExternalModuleDependencies(modulesPath, []string{"auth"}, false)
		if err == nil {
			t.Fatal("expected parse error for invalid JSON")
		}
		if !strings.Contains(err.Error(), "parse package.json at") {
			t.Fatalf("expected path in parse error, got %v", err)
		}
	})

	t.Run("skips empty module name in list", func(t *testing.T) {
		modulesPath := filepath.Join(t.TempDir(), "modules")
		writeModulePackageJSON(t, modulesPath, "auth", `{
			"dependencies": {"vue": "^3.5.0"}
		}`)
		got, err := CollectExternalModuleDependencies(modulesPath, []string{"", "auth", ""}, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !reflect.DeepEqual(got, []string{"vue"}) {
			t.Fatalf("expected [vue], got %#v", got)
		}
	})

	t.Run("deduplicates module names", func(t *testing.T) {
		modulesPath := filepath.Join(t.TempDir(), "modules")
		writeModulePackageJSON(t, modulesPath, "auth", `{
			"dependencies": {"vue": "^3.5.0"}
		}`)
		got, err := CollectExternalModuleDependencies(modulesPath, []string{"auth", "auth"}, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !reflect.DeepEqual(got, []string{"vue"}) {
			t.Fatalf("expected [vue] once, got %#v", got)
		}
	})
}

func writeModulePackageJSON(t *testing.T, modulesPath string, moduleName string, body string) {
	t.Helper()
	packagePath := filepath.Join(modulesPath, moduleName, "package.json")
	if err := os.MkdirAll(filepath.Dir(packagePath), 0o755); err != nil {
		t.Fatalf("mkdir module %s: %v", moduleName, err)
	}
	if err := os.WriteFile(packagePath, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s package.json: %v", moduleName, err)
	}
}
