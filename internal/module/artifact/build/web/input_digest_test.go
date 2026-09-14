// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package webmodulebuilder

import (
	"os"
	"path/filepath"
	"testing"
)

func TestComputeWebInputDigestStableAndSensitive(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	modPath := filepath.Join(root, "web")
	entry := filepath.Join(modPath, "web", "index.ts")
	if err := os.MkdirAll(filepath.Dir(entry), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(entry, []byte("export default {}\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	in := WebInputDigestInputs{
		ModulesPath: root,
		SourceMap:   false,
		Minify:      true,
		TreeShaking: true,
		WebEntryPoints: []webEntryRef{{
			ModuleName: "web",
			Version:    "1.0.0",
			EntryPath:  entry,
			ModulePath: modPath,
		}},
	}
	a, err := ComputeWebInputDigest(in)
	if err != nil || a == "" {
		t.Fatalf("ComputeWebInputDigest() = %q, %v", a, err)
	}
	b, err := ComputeWebInputDigest(in)
	if err != nil || b != a {
		t.Fatalf("digest not stable: %q vs %q (%v)", a, b, err)
	}
	in.SourceMap = true
	c, err := ComputeWebInputDigest(in)
	if err != nil || c == a {
		t.Fatalf("sourcemap change should alter digest: %q vs %q (%v)", a, c, err)
	}
	in.SourceMap = false
	if err := os.WriteFile(entry, []byte("export default { changed: true }\n"), 0o644); err != nil {
		t.Fatalf("rewrite entry: %v", err)
	}
	d, err := ComputeWebInputDigest(in)
	if err != nil || d == a {
		t.Fatalf("source edit should alter digest: %q vs %q (%v)", a, d, err)
	}
	apiWeb := filepath.Join(root, "api", "web", "crm", "index.ts")
	if err := os.MkdirAll(filepath.Dir(apiWeb), 0o755); err != nil {
		t.Fatalf("mkdir api web: %v", err)
	}
	if err := os.WriteFile(apiWeb, []byte("export const client = 1\n"), 0o644); err != nil {
		t.Fatalf("write api web: %v", err)
	}
	e, err := ComputeWebInputDigest(in)
	if err != nil || e == d {
		t.Fatalf("api/web change should alter digest: %q vs %q (%v)", d, e, err)
	}
	in.ForceRebuild = true
	forced, err := ComputeWebInputDigest(in)
	if err != nil || forced == "" || forced != e {
		t.Fatalf("ForceRebuild must still return a stampable digest, got %q (%v)", forced, err)
	}
}

func TestShouldSkipGlobalWebBuild(t *testing.T) {
	t.Parallel()
	dist := t.TempDir()
	if skip, err := ShouldSkipGlobalWebBuild(dist, "abc"); err != nil || skip {
		t.Fatalf("missing index should not skip: skip=%v err=%v", skip, err)
	}
	if err := os.WriteFile(filepath.Join(dist, "index.html"), []byte("<html></html>"), 0o644); err != nil {
		t.Fatalf("write index: %v", err)
	}
	if skip, err := ShouldSkipGlobalWebBuild(dist, "abc"); err != nil || skip {
		t.Fatalf("missing stamp should not skip: skip=%v err=%v", skip, err)
	}
	if err := WriteStoredWebInputDigest(dist, "abc"); err != nil {
		t.Fatalf("WriteStoredWebInputDigest: %v", err)
	}
	skip, err := ShouldSkipGlobalWebBuild(dist, "abc")
	if err != nil || !skip {
		t.Fatalf("matching digest should skip: skip=%v err=%v", skip, err)
	}
	skip, err = ShouldSkipGlobalWebBuild(dist, "other")
	if err != nil || skip {
		t.Fatalf("mismatch should not skip: skip=%v err=%v", skip, err)
	}
}
