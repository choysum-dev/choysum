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
	addonsPath := t.TempDir()
	makeDir(t, filepath.Join(addonsPath, "auth", "service"))
	makeDir(t, filepath.Join(addonsPath, "web", "web"))
	makeDir(t, filepath.Join(addonsPath, "empty"))
	makeDir(t, filepath.Join(addonsPath, ".choysum", "service"))
	makeDir(t, filepath.Join(addonsPath, "tmp", "web"))
	writeFile(t, filepath.Join(addonsPath, "README.md"), "ignored")

	t.Run("returns all apps with service or web targets", func(t *testing.T) {
		apps, err := ResolveApps(addonsPath, "all")
		if err != nil {
			t.Fatalf("ResolveApps returned error: %v", err)
		}
		sort.Strings(apps)
		if !reflect.DeepEqual(apps, []string{"auth", "web"}) {
			t.Fatalf("unexpected apps: %#v", apps)
		}
	})

	t.Run("returns explicit app when target exists and has sources", func(t *testing.T) {
		apps, err := ResolveApps(addonsPath, "auth")
		if err != nil {
			t.Fatalf("ResolveApps returned error: %v", err)
		}
		if !reflect.DeepEqual(apps, []string{"auth"}) {
			t.Fatalf("unexpected apps: %#v", apps)
		}
	})

	t.Run("rejects unknown app", func(t *testing.T) {
		_, err := ResolveApps(addonsPath, "missing")
		if err == nil || !strings.Contains(err.Error(), "unknown app") {
			t.Fatalf("expected unknown app error, got %v", err)
		}
	})

	t.Run("returns empty list for app without service or web sources", func(t *testing.T) {
		apps, err := ResolveApps(addonsPath, "empty")
		if err != nil {
			t.Fatalf("ResolveApps returned error: %v", err)
		}
		if len(apps) != 0 {
			t.Fatalf("expected no apps, got %#v", apps)
		}
	})

	t.Run("requires addons path", func(t *testing.T) {
		_, err := ResolveApps("", "all")
		if err == nil || !strings.Contains(err.Error(), "addons_path is required") {
			t.Fatalf("expected addons path error, got %v", err)
		}
	})
}

func TestHasTargets(t *testing.T) {
	addonsPath := t.TempDir()
	makeDir(t, filepath.Join(addonsPath, "svc", "service"))
	makeDir(t, filepath.Join(addonsPath, "ui", "web"))
	makeDir(t, filepath.Join(addonsPath, "both", "service"))
	makeDir(t, filepath.Join(addonsPath, "both", "web"))
	makeDir(t, filepath.Join(addonsPath, "none"))

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
			got, err := HasTargets(addonsPath, tt.app)
			if err != nil {
				t.Fatalf("HasTargets returned error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("HasTargets(%q) = %v, want %v", tt.app, got, tt.want)
			}
		})
	}
}

func TestResolveNpxPath(t *testing.T) {
	t.Run("uses npx next to provided npm path", func(t *testing.T) {
		binDir := filepath.Join(t.TempDir(), "bin")
		makeDir(t, binDir)
		npmPath := filepath.Join(binDir, "npm")
		npxPath := filepath.Join(binDir, "npx")
		writeFile(t, npmPath, "#!/bin/sh\n")
		writeFile(t, npxPath, "#!/bin/sh\n")

		got, err := resolveNpxPath(npmPath)
		if err != nil {
			t.Fatalf("resolveNpxPath returned error: %v", err)
		}
		if got != npxPath {
			t.Fatalf("resolveNpxPath returned %q, want %q", got, npxPath)
		}
	})

	t.Run("falls back to PATH lookup", func(t *testing.T) {
		binDir := filepath.Join(t.TempDir(), "bin")
		makeDir(t, binDir)
		npxPath := filepath.Join(binDir, "npx")
		writeFile(t, npxPath, "#!/bin/sh\n")
		originalPath := os.Getenv("PATH")
		t.Setenv("PATH", binDir+string(os.PathListSeparator)+originalPath)

		got, err := resolveNpxPath("")
		if err != nil {
			t.Fatalf("resolveNpxPath returned error: %v", err)
		}
		if got != "npx" {
			t.Fatalf("resolveNpxPath returned %q, want npx", got)
		}
	})

	t.Run("returns helpful error when npx is missing", func(t *testing.T) {
		t.Setenv("PATH", "")

		_, err := resolveNpxPath("")
		if err == nil || !strings.Contains(err.Error(), "missing npx") {
			t.Fatalf("expected missing npx error, got %v", err)
		}
		if !strings.Contains(err.Error(), "npm install") {
			t.Fatalf("expected install hint, got %v", err)
		}
	})
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
