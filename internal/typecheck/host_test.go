// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package typecheck

import (
	"path/filepath"
	"testing"
)

func TestOverlayReadFile(t *testing.T) {
	t.Parallel()
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
