// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package choysume2e

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestSourcePathReturnsExistingFile(t *testing.T) {
	p, err := SourcePath()
	if err != nil {
		t.Fatalf("SourcePath: %v", err)
	}
	if !strings.HasSuffix(p, "choysume2e.js") {
		t.Fatalf("unexpected path: %q", p)
	}
	st, err := os.Stat(p)
	if err != nil || st.IsDir() {
		t.Fatalf("expected file at %q: %v", p, err)
	}
	if ChoysumE2EScript == "" {
		t.Fatal("embedded ChoysumE2EScript empty")
	}
}

func TestPlaywrightShimPathReturnsExistingFile(t *testing.T) {
	p, err := PlaywrightShimPath()
	if err != nil {
		t.Fatalf("PlaywrightShimPath: %v", err)
	}
	if !strings.HasSuffix(p, "choysume2e_pw_shim.mjs") {
		t.Fatalf("unexpected path: %q", p)
	}
	st, err := os.Stat(p)
	if err != nil || st.IsDir() {
		t.Fatalf("expected file at %q: %v", p, err)
	}
	if PlaywrightShimScript == "" {
		t.Fatal("embedded PlaywrightShimScript empty")
	}
}

func TestPackageFileAndMaterialize(t *testing.T) {
	p, err := packageFile("choysume2e.js")
	if err != nil {
		t.Fatalf("packageFile: %v", err)
	}
	if filepath.Base(p) != "choysume2e.js" {
		t.Fatalf("base=%q", filepath.Base(p))
	}
	if _, err := packageFile("no-such-file.xyz"); err == nil {
		t.Fatal("expected missing package file error")
	}

	// Force materialize path by writing into a fresh temp dir via materializeFile.
	got, err := materializeFile("probe.js", "export const x = 1;\n")
	if err != nil {
		t.Fatalf("materializeFile: %v", err)
	}
	raw, err := os.ReadFile(got)
	if err != nil || string(raw) != "export const x = 1;\n" {
		t.Fatalf("materialized content: %q err=%v", raw, err)
	}
	// Second call hits existing-file branch.
	got2, err := materializeFile("probe.js", "ignored")
	if err != nil || got2 != got {
		t.Fatalf("reuse: got2=%q err=%v", got2, err)
	}
}

func TestMaterializeFileMkdirTempError(t *testing.T) {
	materializeMu.Lock()
	oldDir := materializedDir
	materializedDir = ""
	materializeMu.Unlock()
	oldMkdir := osMkdirTemp
	osMkdirTemp = func(dir, pattern string) (string, error) { return "", errors.New("mkdirtemp boom") }
	defer func() {
		osMkdirTemp = oldMkdir
		materializeMu.Lock()
		materializedDir = oldDir
		materializeMu.Unlock()
	}()
	if _, err := materializeFile("x.js", "y"); err == nil || !strings.Contains(err.Error(), "mkdirtemp") {
		t.Fatalf("got %v", err)
	}
}

func TestPackageFileCallerFail(t *testing.T) {
	old := runtimeCaller
	runtimeCaller = func(skip int) (uintptr, string, int, bool) { return 0, "", 0, false }
	defer func() { runtimeCaller = old }()
	if _, err := packageFile("choysume2e.js"); err == nil || !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("got %v", err)
	}
}

func TestMaterializeFileWriteError(t *testing.T) {
	materializeMu.Lock()
	old := materializedDir
	// Point at a non-existent directory so WriteFile fails (parent missing).
	materializedDir = filepath.Join(t.TempDir(), "missing-parent", "child")
	materializeMu.Unlock()
	defer func() {
		materializeMu.Lock()
		materializedDir = old
		materializeMu.Unlock()
	}()
	if _, err := materializeFile("boom.js", "x"); err == nil {
		t.Fatal("expected write error")
	}
}

func TestSourcePathAndShimMaterializeFallback(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	pkgDir := filepath.Dir(thisFile)

	// Rename package sources so packageFile fails and SourcePath/PlaywrightShimPath materialize.
	type pair struct{ name, bak string }
	pairs := []pair{
		{"choysume2e.js", ""},
		{"choysume2e_pw_shim.mjs", ""},
	}
	t.Cleanup(func() {
		for _, p := range pairs {
			if p.bak == "" {
				continue
			}
			_ = os.Rename(p.bak, filepath.Join(pkgDir, p.name))
		}
	})
	for i := range pairs {
		src := filepath.Join(pkgDir, pairs[i].name)
		bak := src + ".bak_covtest"
		if err := os.Rename(src, bak); err != nil {
			t.Fatalf("rename %s: %v", pairs[i].name, err)
		}
		pairs[i].bak = bak
	}

	materializeMu.Lock()
	materializedDir = ""
	materializeMu.Unlock()

	jsPath, err := SourcePath()
	if err != nil {
		t.Fatalf("SourcePath materialize: %v", err)
	}
	if !strings.Contains(jsPath, "choysume2e.js") {
		t.Fatalf("path=%q", jsPath)
	}
	shimPath, err := PlaywrightShimPath()
	if err != nil {
		t.Fatalf("PlaywrightShimPath materialize: %v", err)
	}
	if !strings.Contains(shimPath, "choysume2e_pw_shim.mjs") {
		t.Fatalf("shim=%q", shimPath)
	}
}
