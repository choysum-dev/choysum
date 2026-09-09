// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package choysume2e

import (
	"os"
	"path/filepath"
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
