// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package typecheck

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestNewTypecheckFS_EsmShDiskRewrite(t *testing.T) {
	dir := t.TempDir()
	esmFile := filepath.Join(dir, "pkg", "types", "esm.sh_vue@3.5.0_dist_vue.d.ts")
	mustMkdir(t, filepath.Dir(esmFile))
	mustWrite(t, esmFile, `declare module 'https://esm.sh/vue@3.5.35/dist/vue.d.mts' { export const x: number; }`+"\n")

	fs := newTypecheckFS(nil)
	got, ok := fs.ReadFile(filepath.ToSlash(esmFile))
	if !ok {
		t.Fatal("expected disk read")
	}
	if strings.Contains(got, "https://esm.sh/") {
		t.Fatalf("expected esm.sh rewrite on disk read, got %q", got)
	}
	if !strings.Contains(got, "declare module 'vue'") {
		t.Fatalf("got %q", got)
	}

	if content, ok := fs.ReadFile(filepath.ToSlash(filepath.Join(dir, "missing.ts"))); ok || content != "" {
		t.Fatalf("missing file should miss, ok=%v content=%q", ok, content)
	}
}

func TestRewriteEsmShDeclareModules_ShortMatchPassthrough(t *testing.T) {
	in := "declare module 'https://esm.sh/' { export {} }"
	got := rewriteEsmShDeclareModules(in)
	if got != in {
		t.Fatalf("short match should passthrough, got %q", got)
	}
}

func TestOverlayReadFile(t *testing.T) {
	dir := t.TempDir()
	diskFile := filepath.Join(dir, "on_disk.ts")
	mustWrite(t, diskFile, "export const fromDisk = 1;\n")

	overlayPath := filepath.ToSlash(filepath.Join(dir, "virtual.ts"))
	fs := newTypecheckFS(map[string]string{
		overlayPath: "export const fromOverlay = 2;\n",
	})

	got, ok := fs.ReadFile(overlayPath)
	if !ok || got != "export const fromOverlay = 2;\n" {
		t.Fatalf("overlay read = %q ok=%v", got, ok)
	}
	if !fs.FileExists(overlayPath) {
		t.Fatal("expected overlay FileExists")
	}

	diskSlash := filepath.ToSlash(diskFile)
	gotDisk, ok := fs.ReadFile(diskSlash)
	if !ok || gotDisk != "export const fromDisk = 1;\n" {
		t.Fatalf("disk read = %q ok=%v", gotDisk, ok)
	}
}
