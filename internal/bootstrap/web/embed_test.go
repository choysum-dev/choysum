// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package web

import (
	"io/fs"
	"testing"
)

func TestLoadDistFS_DefaultEmbed(t *testing.T) {
	distFS, source, root, err := LoadDistFS("")
	if err != nil {
		t.Fatalf("LoadDistFS() error = %v", err)
	}
	if source != "embed" {
		t.Fatalf("LoadDistFS() source = %q, want embed", source)
	}
	if root == "" {
		t.Fatal("LoadDistFS() root should not be empty")
	}
	if _, err := fs.ReadFile(distFS, "index.html"); err != nil {
		t.Fatalf("ReadFile(index.html) error = %v", err)
	}
}

func TestLoadDistFS_Disk(t *testing.T) {
	distFS, source, root, err := LoadDistFS("disk")
	if err != nil {
		t.Fatalf("LoadDistFS(disk) error = %v", err)
	}
	if source != "disk" {
		t.Fatalf("LoadDistFS(disk) source = %q, want disk", source)
	}
	if root == "" {
		t.Fatal("LoadDistFS(disk) root should not be empty")
	}
	if _, err := fs.ReadFile(distFS, "index.html"); err != nil {
		t.Fatalf("ReadFile(index.html) error = %v", err)
	}
}
