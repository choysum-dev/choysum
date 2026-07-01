// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package moddeps

import (
	"os"
	"path/filepath"
	"reflect"
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
