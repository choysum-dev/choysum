// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package noderuntime

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
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

func TestEnsureGlobalModuleLinksAt(t *testing.T) {
	localRoot := filepath.Join(t.TempDir(), "local", "node_modules")
	globalRoot := filepath.Join(t.TempDir(), "global", "node_modules")
	if err := os.MkdirAll(filepath.Join(globalRoot, "left-pad"), 0o755); err != nil {
		t.Fatalf("mkdir global left-pad: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(globalRoot, "@scoped", "pkg"), 0o755); err != nil {
		t.Fatalf("mkdir global @scoped/pkg: %v", err)
	}

	cleanup, err := EnsureGlobalModuleLinksAt(localRoot, globalRoot, []string{"left-pad", "@scoped/pkg"})
	if err != nil {
		t.Fatalf("EnsureGlobalModuleLinksAt error: %v", err)
	}

	leftPadLink := filepath.Join(localRoot, "left-pad")
	if st, statErr := os.Lstat(leftPadLink); statErr != nil {
		t.Fatalf("expected left-pad symlink, lstat err=%v", statErr)
	} else if st.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("expected left-pad symlink, mode=%v", st.Mode())
	}
	scopedLink := filepath.Join(localRoot, "@scoped", "pkg")
	if st, statErr := os.Lstat(scopedLink); statErr != nil {
		t.Fatalf("expected @scoped/pkg symlink, lstat err=%v", statErr)
	} else if st.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("expected @scoped/pkg symlink, mode=%v", st.Mode())
	}

	cleanup()

	if _, err := os.Lstat(leftPadLink); !os.IsNotExist(err) {
		t.Fatalf("expected left-pad link removed after cleanup, err=%v", err)
	}
	if _, err := os.Lstat(scopedLink); !os.IsNotExist(err) {
		t.Fatalf("expected @scoped/pkg link removed after cleanup, err=%v", err)
	}
	if st, err := os.Stat(localRoot); err == nil && st.IsDir() {
		entries, readErr := os.ReadDir(localRoot)
		if readErr != nil {
			t.Fatalf("read local root: %v", readErr)
		}
		if len(entries) != 0 {
			t.Fatalf("expected local root empty after cleanup, found %d entries", len(entries))
		}
	}
}

func TestEnsureGlobalModuleLinksAtCleansUpOnSymlinkFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink permission semantics differ on windows")
	}

	localRoot := filepath.Join(t.TempDir(), "local", "node_modules")
	globalRoot := filepath.Join(t.TempDir(), "global", "node_modules")
	if err := os.MkdirAll(filepath.Join(globalRoot, "left-pad"), 0o755); err != nil {
		t.Fatalf("mkdir global left-pad: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(globalRoot, "@scoped", "pkg"), 0o755); err != nil {
		t.Fatalf("mkdir global @scoped/pkg: %v", err)
	}
	if err := os.MkdirAll(localRoot, 0o755); err != nil {
		t.Fatalf("mkdir local root: %v", err)
	}

	blockedScopeDir := filepath.Join(localRoot, "@scoped")
	if err := os.MkdirAll(blockedScopeDir, 0o755); err != nil {
		t.Fatalf("mkdir local @scoped dir: %v", err)
	}
	if err := os.Chmod(blockedScopeDir, 0o555); err != nil {
		t.Fatalf("chmod local @scoped dir readonly: %v", err)
	}
	defer func() {
		_ = os.Chmod(blockedScopeDir, 0o755)
	}()

	cleanup, err := EnsureGlobalModuleLinksAt(localRoot, globalRoot, []string{"left-pad", "@scoped/pkg"})
	if err == nil {
		cleanup()
		t.Fatal("expected symlink failure for readonly scoped directory")
	}
	if _, statErr := os.Lstat(filepath.Join(localRoot, "left-pad")); !os.IsNotExist(statErr) {
		t.Fatalf("expected rollback to remove previously created left-pad link, lstat err=%v", statErr)
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
