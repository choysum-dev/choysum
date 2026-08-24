// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package dataloader

import (
	"encoding/json"
	"testing"

	"github.com/choysum-dev/choysum/pkg/meta"
)

func TestManifestDataFiles(t *testing.T) {
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
