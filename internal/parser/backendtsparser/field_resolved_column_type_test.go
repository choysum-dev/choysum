// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package backendtsparser

import "testing"

func TestResolveColumnType_MonetaryMapsToDecimal(t *testing.T) {
	if got := resolveColumnType("monetary"); got != "decimal" {
		t.Fatalf("resolveColumnType(monetary)=%q, want decimal", got)
	}
	// Logical FieldType remains distinct; only physical column type collapses.
	if got := resolveColumnType("decimal"); got != "decimal" {
		t.Fatalf("resolveColumnType(decimal)=%q, want decimal", got)
	}
}
