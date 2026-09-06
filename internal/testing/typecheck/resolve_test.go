// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package typecheck

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestResolveApps(t *testing.T) {
	modulesPath := t.TempDir()
	makeDir(t, filepath.Join(modulesPath, "auth", "service"))
	makeDir(t, filepath.Join(modulesPath, "web", "web"))
	makeDir(t, filepath.Join(modulesPath, "empty"))
	makeDir(t, filepath.Join(modulesPath, ".choysum", "service"))
	makeDir(t, filepath.Join(modulesPath, "tmp", "web"))
	writeFile(t, filepath.Join(modulesPath, "README.md"), "ignored")

	t.Run("returns all apps with service or web targets", func(t *testing.T) {
		apps, err := ResolveApps(modulesPath, "all")
		if err != nil {
			t.Fatalf("ResolveApps returned error: %v", err)
		}
		sort.Strings(apps)
		if !reflect.DeepEqual(apps, []string{"auth", "web"}) {
			t.Fatalf("unexpected apps: %#v", apps)
		}
	})

	t.Run("returns explicit app when target exists and has sources", func(t *testing.T) {
		apps, err := ResolveApps(modulesPath, "auth")
		if err != nil {
			t.Fatalf("ResolveApps returned error: %v", err)
		}
		if !reflect.DeepEqual(apps, []string{"auth"}) {
			t.Fatalf("unexpected apps: %#v", apps)
		}
	})

	t.Run("rejects unknown app", func(t *testing.T) {
		_, err := ResolveApps(modulesPath, "missing")
		if err == nil || !strings.Contains(err.Error(), "unknown app") {
			t.Fatalf("expected unknown app error, got %v", err)
		}
	})

	t.Run("returns empty list for app without service or web sources", func(t *testing.T) {
		apps, err := ResolveApps(modulesPath, "empty")
		if err != nil {
			t.Fatalf("ResolveApps returned error: %v", err)
		}
		if len(apps) != 0 {
			t.Fatalf("expected no apps, got %#v", apps)
		}
	})

	t.Run("requires modules path", func(t *testing.T) {
		_, err := ResolveApps("", "all")
		if err == nil || !strings.Contains(err.Error(), "modules_path is required") {
			t.Fatalf("expected modules path error, got %v", err)
		}
	})

	t.Run("propagates read modules dir error", func(t *testing.T) {
		missingModulesPath := filepath.Join(t.TempDir(), "missing")
		_, err := ResolveApps(missingModulesPath, "all")
		if err == nil || !strings.Contains(err.Error(), "read modules dir") {
			t.Fatalf("expected read modules dir error, got %v", err)
		}
	})
}

func TestHasTargets(t *testing.T) {
	modulesPath := t.TempDir()
	makeDir(t, filepath.Join(modulesPath, "svc", "service"))
	makeDir(t, filepath.Join(modulesPath, "ui", "web"))
	makeDir(t, filepath.Join(modulesPath, "both", "service"))
	makeDir(t, filepath.Join(modulesPath, "both", "web"))
	makeDir(t, filepath.Join(modulesPath, "none"))

	tests := []struct {
		name string
		app  string
		want bool
	}{
		{name: "service directory counts", app: "svc", want: true},
		{name: "web directory counts", app: "ui", want: true},
		{name: "both directories count", app: "both", want: true},
		{name: "missing targets returns false", app: "none", want: false},
		{name: "unknown app returns false", app: "missing", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := HasTargets(modulesPath, tt.app)
			if err != nil {
				t.Fatalf("HasTargets returned error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("HasTargets(%q) = %v, want %v", tt.app, got, tt.want)
			}
		})
	}
}

func TestTypecheckHelperFunctions(t *testing.T) {
	if got := sanitizeAppToken("crm/web app"); got != "crm_web_app" {
		t.Fatalf("sanitizeAppToken() = %q, want crm_web_app", got)
	}
	if got := sanitizeAppToken(""); got != "app" {
		t.Fatalf("sanitizeAppToken(empty) = %q, want app", got)
	}
	if got := sanitizeAppToken("   "); got != "___" {
		t.Fatalf("sanitizeAppToken(spaces) = %q, want ___", got)
	}

	for _, name := range []string{"node_modules", "dist", ".choysum", "tmp"} {
		if !shouldSkipTypecheckInputScanDir(name) {
			t.Fatalf("shouldSkipTypecheckInputScanDir(%q) = false, want true", name)
		}
	}
	if shouldSkipTypecheckInputScanDir("service") {
		t.Fatal("shouldSkipTypecheckInputScanDir(service) = true, want false")
	}

	modulesPath := t.TempDir()
	appName := "auth"
	makeDir(t, filepath.Join(modulesPath, appName, "node_modules"))
	writeFile(t, filepath.Join(modulesPath, appName, "node_modules", "ignored.ts"), "export const ignored = true\n")

	hasInputs, err := hasTypecheckInputs(modulesPath, appName)
	if err != nil {
		t.Fatalf("hasTypecheckInputs(only skipped dirs) error = %v", err)
	}
	if hasInputs {
		t.Fatal("hasTypecheckInputs() = true with only skipped dirs")
	}

	makeDir(t, filepath.Join(modulesPath, appName, "web"))
	writeFile(t, filepath.Join(modulesPath, appName, "web", "index.vue"), "<template><div/></template>\n")
	hasInputs, err = hasTypecheckInputs(modulesPath, appName)
	if err != nil {
		t.Fatalf("hasTypecheckInputs(with vue input) error = %v", err)
	}
	if !hasInputs {
		t.Fatal("hasTypecheckInputs() = false, want true when vue input exists")
	}
}

func makeDir(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", dir, err)
	}
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("WriteFile(%q): %v", path, err)
	}
}
