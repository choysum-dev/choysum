// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package e2e

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestWriteE2EEntry(t *testing.T) {
	dir := t.TempDir()
	entry := filepath.Join(dir, "entry.js")
	spec := filepath.Join(dir, "a.spec.ts")
	if err := os.WriteFile(spec, []byte("export {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := WriteE2EEntry(entry, []string{spec}); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(entry)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "import ") || !strings.Contains(string(raw), "a.spec.ts") {
		t.Fatalf("unexpected entry: %s", raw)
	}
}

func TestBuildE2EBundleAlias(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))

	dir := t.TempDir()
	spec := filepath.Join(dir, "smoke.spec.ts")
	if err := os.WriteFile(spec, []byte(`
import { test, expect, page, runtime } from '@choysum/e2e';
test('x', async () => { void page; void runtime; void expect; });
`), 0o644); err != nil {
		t.Fatal(err)
	}
	entry := filepath.Join(dir, "entry.js")
	if err := WriteE2EEntry(entry, []string{spec}); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "bundle.js")
	res, err := BuildE2EBundle(E2EBundleOptions{
		RepoRoot:   repoRoot,
		EntryPath:  entry,
		Outfile:    out,
		WorkingDir: repoRoot,
		RunDir:     dir,
	})
	if err != nil {
		t.Fatalf("BuildE2EBundle: %v", err)
	}
	if res == nil || res.JS == "" {
		t.Fatal("empty bundle")
	}
	if !strings.Contains(res.JS, "__choysum_e2e_host__") && !strings.Contains(res.JS, "getHost") {
		t.Fatalf("bundle missing e2e facade markers")
	}
}
