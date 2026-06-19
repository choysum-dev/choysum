// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package noderuntime

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestResolveGlobalNpmRootUsesOverride(t *testing.T) {
	t.Setenv("CHOYSUM_NPM_GLOBAL_ROOT", "/tmp/custom-node-modules")
	got, err := ResolveGlobalNpmRoot()
	if err != nil {
		t.Fatalf("ResolveGlobalNpmRoot returned error: %v", err)
	}
	if got != "/tmp/custom-node-modules" {
		t.Fatalf("ResolveGlobalNpmRoot returned %q, want %q", got, "/tmp/custom-node-modules")
	}
}

func TestFindExecutablePrefersPath(t *testing.T) {
	binDir := t.TempDir()
	writeExecFile(t, filepath.Join(binDir, "playwright"), "#!/bin/sh\nexit 0\n")
	t.Setenv("PATH", binDir)

	tool, binPath, found := FindExecutable("playwright")
	if !found {
		t.Fatal("expected executable to be found via PATH")
	}
	if tool != "playwright" {
		t.Fatalf("FindExecutable tool = %q, want %q", tool, "playwright")
	}
	if binPath != "" {
		t.Fatalf("FindExecutable binPath = %q, want empty", binPath)
	}
}

func TestFindExecutableFallsBackToLocalBin(t *testing.T) {
	t.Setenv("PATH", "")
	root := t.TempDir()
	binDir := filepath.Join(root, ".bin")
	playwrightPath := filepath.Join(binDir, "playwright")
	writeExecFile(t, playwrightPath, "#!/bin/sh\nexit 0\n")

	tool, gotBinDir, found := FindExecutable("playwright", root)
	if !found {
		t.Fatal("expected executable to be found via local .bin")
	}
	if tool != playwrightPath {
		t.Fatalf("FindExecutable tool = %q, want %q", tool, playwrightPath)
	}
	if gotBinDir != binDir {
		t.Fatalf("FindExecutable binDir = %q, want %q", gotBinDir, binDir)
	}
}

func TestFindExecutableHandlesNpmBinaryPathHint(t *testing.T) {
	t.Setenv("PATH", "")
	binDir := t.TempDir()
	npmPath := filepath.Join(binDir, "npm")
	npxPath := filepath.Join(binDir, "npx")
	writeExecFile(t, npmPath, "#!/bin/sh\nexit 0\n")
	writeExecFile(t, npxPath, "#!/bin/sh\nexit 0\n")

	tool, gotBinDir, found := FindExecutable("npx", npmPath)
	if !found {
		t.Fatal("expected executable to be found via npm sibling path")
	}
	if tool != npxPath {
		t.Fatalf("FindExecutable tool = %q, want %q", tool, npxPath)
	}
	if gotBinDir != binDir {
		t.Fatalf("FindExecutable binDir = %q, want %q", gotBinDir, binDir)
	}
}

func TestMissingRequiredNodeModules(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "@playwright", "test"), 0o755); err != nil {
		t.Fatalf("mkdir playwright package: %v", err)
	}

	required := []string{"@playwright/test", "@connectrpc/connect"}
	missing := MissingRequiredNodeModules(required, root)
	if !reflect.DeepEqual(missing, []string{"@connectrpc/connect"}) {
		t.Fatalf("MissingRequiredNodeModules returned %#v", missing)
	}
	if !ModuleInstalledInRoots("@playwright/test", root) {
		t.Fatal("expected @playwright/test to be detected")
	}
	if ModuleInstalledInRoots("@connectrpc/connect", root) {
		t.Fatal("expected @connectrpc/connect to be missing")
	}
}

func writeExecFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
