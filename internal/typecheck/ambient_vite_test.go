// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package typecheck

import (
	"os"
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
	relModules := filepath.Join(dir, "modules")
	mustMkdir(t, relModules)
	// Use a relative modules path from cwd.
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	got := AmbientRoot("modules")
	if !filepath.IsAbs(got) {
		t.Fatalf("AmbientRoot must be absolute, got %q", got)
	}
	if !strings.HasSuffix(filepath.ToSlash(got), "/modules/"+ambientDirName) {
		t.Fatalf("AmbientRoot = %q", got)
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
			// keys may differ only by slash normalization
			found := false
			for k := range overlays {
				if normalizePathKey(k) == f {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("root %q not in overlays %#v", f, overlays)
			}
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
	if got["/a.ts"] != "2" || got["/b.ts"] != "3" {
		t.Fatalf("got %#v", got)
	}
}
