// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package dataloader

import (
	"encoding/json"
	"testing"

	"github.com/choysum-dev/choysum/pkg/meta"
)

func TestManifestDataFiles(t *testing.T) {
	if files, err := ManifestDataFiles(nil); err != nil || files != nil {
		t.Fatalf("ManifestDataFiles(nil) = %v, %v", files, err)
	}

	dataJSON, err := json.Marshal([]string{"data/bootstrap.json"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	mod := &meta.Module{DataStr: dataJSON}
	files, err := ManifestDataFiles(mod)
	if err != nil {
		t.Fatalf("ManifestDataFiles: %v", err)
	}
	if len(files) != 1 || files[0] != "data/bootstrap.json" {
		t.Fatalf("files = %v", files)
	}
}

func TestManifestDemoFiles(t *testing.T) {
	if files, err := ManifestDemoFiles(nil); err != nil || files != nil {
		t.Fatalf("ManifestDemoFiles(nil) = %v, %v", files, err)
	}

	demoJSON, err := json.Marshal([]string{"data/demo.json"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	files, err := ManifestDemoFiles(&meta.Module{DemoStr: demoJSON})
	if err != nil {
		t.Fatalf("ManifestDemoFiles: %v", err)
	}
	if len(files) != 1 || files[0] != "data/demo.json" {
		t.Fatalf("files = %v", files)
	}
}

func TestManifestDataFiles_invalidJSON(t *testing.T) {
	_, err := ManifestDataFiles(&meta.Module{DataStr: []byte("{")})
	if err == nil {
		t.Fatal("expected decode error")
	}
}
