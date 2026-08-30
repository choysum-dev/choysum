// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package record

import (
	"testing"

	importpkg "github.com/choysum-dev/choysum/pkg/import"
)

func TestDefaultImportFields(t *testing.T) {
	partner, err := DefaultImportFields("partner.Partner")
	if err != nil {
		t.Fatalf("partner: %v", err)
	}
	if len(partner) == 0 {
		t.Fatal("partner defaults empty")
	}
	country, err := DefaultImportFields("base.Country")
	if err != nil {
		t.Fatalf("country: %v", err)
	}
	if len(country) == 0 {
		t.Fatal("country defaults empty")
	}
	_, err = DefaultImportFields("base.Currency")
	if err == nil {
		t.Fatal("expected unsupported model error")
	}
	if impErr, ok := importpkg.AsError(err); !ok || impErr.Code != importpkg.CodeModelNotFound {
		t.Fatalf("error = %v, want CodeModelNotFound", err)
	}
}
