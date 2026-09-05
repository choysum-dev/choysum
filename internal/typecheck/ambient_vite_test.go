// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package typecheck

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestViteClientOverlay_Embedded(t *testing.T) {
	rel, content := ViteClientOverlay()
	if rel != "vite/client.d.ts" {
		t.Fatalf("rel = %q", rel)
	}
	if !strings.Contains(content, "interface ImportMeta") {
		t.Fatalf("missing ImportMeta in embed: %q", content[:min(80, len(content))])
	}
	if !strings.Contains(content, `declare module "vite/client"`) {
		t.Fatal("missing vite/client module declaration")
	}
}

func TestSubpathStubOverlay(t *testing.T) {
	rel, content := SubpathStubOverlay()
	if rel != "subpath-stubs.d.ts" {
		t.Fatalf("rel = %q", rel)
	}
	if !strings.Contains(content, `declare module "dayjs/locale/*"`) {
		t.Fatal("missing dayjs stub")
	}
	if strings.Contains(content, "interface ImportMetaEnv") || strings.Contains(content, `declare module "*.css"`) {
		t.Fatal("subpath stubs must not redeclare vite/client globals or CSS modules")
	}
}

func TestAmbientRoot_Absolute(t *testing.T) {
	dir := t.TempDir()
	got := AmbientRoot(dir)
	if !filepath.IsAbs(got) {
		t.Fatalf("AmbientRoot must be absolute, got %q", got)
	}
	want := filepath.Join(dir, ambientDirName)
	if got != want {
		t.Fatalf("AmbientRoot = %q, want %q", got, want)
	}

	gotRel := AmbientRoot("modules")
	if !filepath.IsAbs(gotRel) {
		t.Fatalf("AmbientRoot must be absolute for relative input, got %q", gotRel)
	}
	absModules, err := filepath.Abs("modules")
	if err != nil {
		t.Fatal(err)
	}
	wantRel := filepath.Join(absModules, ambientDirName)
	if gotRel != wantRel {
		t.Fatalf("AmbientRoot(relative) = %q, want %q", gotRel, wantRel)
	}
}

func TestBuiltInAmbientOverlays_NoDiskVite(t *testing.T) {
	dir := t.TempDir()
	modules := filepath.Join(dir, "modules")
	mustMkdir(t, modules)
	// Intentionally no node_modules/vite.
	overlays := BuiltInAmbientOverlays(modules)
	if len(overlays) != 2 {
		t.Fatalf("overlays = %#v", overlays)
	}
	files := AmbientRootFiles(modules)
	if len(files) != 2 {
		t.Fatalf("files = %v", files)
	}
	for _, f := range files {
		if !strings.Contains(f, ambientDirName) {
			t.Fatalf("ambient path %q missing %s", f, ambientDirName)
		}
		if _, ok := overlays[f]; !ok {
			t.Fatalf("root %q not in overlays %#v", f, overlays)
		}
	}
}

func TestMergeOverlays(t *testing.T) {
	if mergeOverlays(nil, map[string]string{}) != nil {
		t.Fatal("empty merge must be nil")
	}
	got := mergeOverlays(
		map[string]string{"/a.ts": "1", "  ": "x"},
		map[string]string{"/a.ts": "2", "/b.ts": "3"},
	)
	if len(got) != 2 {
		t.Fatalf("expected 2 elements, got %d: %#v", len(got), got)
	}
	a := normalizePathKey("/a.ts")
	b := normalizePathKey("/b.ts")
	if got[a] != "2" || got[b] != "3" {
		t.Fatalf("got %#v", got)
	}
}
