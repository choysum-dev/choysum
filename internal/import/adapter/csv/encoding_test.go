// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package csv_test

import (
	"bytes"
	"testing"

	"github.com/choysum-dev/choysum/internal/import/adapter/csv"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
)

func TestStripUTF8BOM(t *testing.T) {
	raw := append([]byte{0xEF, 0xBB, 0xBF}, []byte("Name,Code\nA,C1\n")...)
	got := string(csv.StripUTF8BOM(raw))
	if got == string(raw) {
		t.Fatal("expected BOM to be stripped")
	}
	if !bytes.HasPrefix([]byte(got), []byte("Name,")) {
		t.Fatalf("unexpected content after BOM strip: %q", got)
	}
}

func TestValidateUTF8_RejectsLatin1(t *testing.T) {
	err := csv.ValidateUTF8([]byte{0xE9})
	if err == nil {
		t.Fatal("expected invalid encoding error")
	}
	impErr, ok := importpkg.AsError(err)
	if !ok || impErr.Code != importpkg.CodeInvalidEncoding {
		t.Fatalf("error = %v, want CodeInvalidEncoding", err)
	}
}

func TestValidateUTF8_EmptyOK(t *testing.T) {
	if err := csv.ValidateUTF8(nil); err != nil {
		t.Fatalf("empty: %v", err)
	}
	if err := csv.ValidateUTF8([]byte("ok")); err != nil {
		t.Fatalf("ascii: %v", err)
	}
}

func TestStripUTF8BOM_NoBOM(t *testing.T) {
	raw := []byte("Name")
	if string(csv.StripUTF8BOM(raw)) != "Name" {
		t.Fatal("expected unchanged")
	}
}

func TestPrependUTF8BOM(t *testing.T) {
	raw := []byte("Name,Code\nA,C1\n")
	got := csv.PrependUTF8BOM(raw)
	if !bytes.HasPrefix(got, []byte{0xEF, 0xBB, 0xBF}) {
		t.Fatal("expected BOM prefix")
	}
	if string(csv.StripUTF8BOM(got)) != string(raw) {
		t.Fatalf("round trip failed: %q", got)
	}
	if len(csv.PrependUTF8BOM(got)) != len(got) {
		t.Fatal("PrependUTF8BOM should be idempotent")
	}
}
