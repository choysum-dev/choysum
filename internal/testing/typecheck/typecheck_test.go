// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package typecheck

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestTypecheckRequiredModules(t *testing.T) {
	got := typecheckRequiredModules(false)
	want := []string{"vue-tsc"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("typecheckRequiredModules(false) = %#v, want %#v", got, want)
	}

	got = typecheckRequiredModules(true)
	want = []string{"vite", "vue-tsc"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("typecheckRequiredModules(true) = %#v, want %#v", got, want)
	}
}

func TestTypecheckInstallCommandPinsVueToolchain(t *testing.T) {
	got := typecheckInstallCommand([]string{"vite", "vue-tsc"})
	want := "npm install -g vite vue-tsc@3.3.7 typescript@6.0.3 @types/node"
	if got != want {
		t.Fatalf("typecheckInstallCommand() = %q, want %q", got, want)
	}
}

func TestValidateTypecheckToolchainVersions(t *testing.T) {
	root := t.TempDir()
	writePackage := func(name string, version string) {
		t.Helper()
		dir := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", name, err)
		}
		content := `{"version":"` + version + `"}`
		if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(content), 0o644); err != nil {
			t.Fatalf("write %s package.json: %v", name, err)
		}
	}

	writePackage("vue-tsc", "3.3.7")
	writePackage("typescript", "6.0.3")
	if err := validateTypecheckToolchainVersions(root); err != nil {
		t.Fatalf("expected pinned toolchain to pass, got %v", err)
	}

	writePackage("typescript", "7.0.2")
	err := validateTypecheckToolchainVersions(root)
	if err == nil {
		t.Fatal("expected TypeScript version mismatch")
	}
	for _, want := range []string{
		"typescript=7.0.2 (required 6.0.3)",
		"vue-tsc@3.3.7 typescript@6.0.3 @types/node",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q in error, got %v", want, err)
		}
	}
}

func TestTypecheckApp_AllowsNpxWithoutGlobalVueTsc(t *testing.T) {
	ctx := context.Background()

	repoRoot := t.TempDir()
	modulesPath := t.TempDir()
	if err := os.MkdirAll(filepath.Join(modulesPath, "auth", "service"), 0o755); err != nil {
		t.Fatalf("mkdir auth service: %v", err)
	}
	if err := os.WriteFile(filepath.Join(modulesPath, "auth", "service", "index.ts"), []byte("export const auth = 1\n"), 0o644); err != nil {
		t.Fatalf("write auth service ts: %v", err)
	}

	binDir := filepath.Join(t.TempDir(), "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	npxPath := filepath.Join(binDir, "npx")
	if err := os.WriteFile(npxPath, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write fake npx: %v", err)
	}

	t.Setenv("PATH", binDir)
	if err := os.MkdirAll(filepath.Join(repoRoot, "node_modules", "vue-tsc"), 0o755); err != nil {
		t.Fatalf("mkdir local vue-tsc module: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "node_modules", "vue-tsc", "package.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("write local vue-tsc package.json: %v", err)
	}

	opts := RunOptions{
		ModulesPath: modulesPath,
		RepoRoot:    repoRoot,
		Stdout:      io.Discard,
		Stderr:      io.Discard,
	}

	err := TypecheckApp(ctx, opts, "auth")
	if err != nil {
		t.Fatalf("expected success with npx-resolved vue-tsc, got: %v", err)
	}
}

func TestResolveViteClientDTS(t *testing.T) {
	t.Run("prefers first existing module root", func(t *testing.T) {
		repoRoot := t.TempDir()
		firstRoot := filepath.Join(t.TempDir(), "first-node-modules")
		secondRoot := filepath.Join(t.TempDir(), "second-node-modules")
		if err := os.MkdirAll(filepath.Join(secondRoot, "vite"), 0o755); err != nil {
			t.Fatalf("mkdir second root vite dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(secondRoot, "vite", "client.d.ts"), []byte("declare interface ImportMetaEnv {}\n"), 0o644); err != nil {
			t.Fatalf("write vite client types: %v", err)
		}

		got := resolveViteClientDTS(repoRoot, firstRoot, secondRoot)
		want := filepath.Join(secondRoot, "vite", "client.d.ts")
		if got != want {
			t.Fatalf("resolveViteClientDTS() = %q, want %q", got, want)
		}
	})

	t.Run("falls back to repo root vite path when no candidates exist", func(t *testing.T) {
		repoRoot := t.TempDir()
		got := resolveViteClientDTS(repoRoot, "", strings.Repeat(" ", 2))
		want := filepath.Join(repoRoot, "node_modules", "vite", "client.d.ts")
		if got != want {
			t.Fatalf("resolveViteClientDTS() fallback = %q, want %q", got, want)
		}
	})
}
