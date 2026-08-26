// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package record_test

import (
	"testing"

	"github.com/choysum-dev/choysum/internal/export/reader/record"
)

func TestDefaultExportFields_Country(t *testing.T) {
	fields, err := record.DefaultExportFields("base.Country")
	if err != nil {
		t.Fatalf("DefaultExportFields: %v", err)
	}
	if len(fields) < 3 || fields[0] != "Name" {
		t.Fatalf("fields = %#v", fields)
	}
}

func TestDefaultExportFields_UnsupportedModel(t *testing.T) {
	_, err := record.DefaultExportFields("base.Partner")
	if err == nil {
		t.Fatal("expected unsupported model error")
	}
}
