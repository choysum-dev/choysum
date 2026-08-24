// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package csv_test

import (
	"testing"

	"github.com/choysum-dev/choysum/internal/import/adapter/csv"
	recordplan "github.com/choysum-dev/choysum/internal/import/plan/record"
)

func TestBuildRecordPlan_ColumnMapping(t *testing.T) {
	raw := []byte("Name,Code,Currency\nAlpha,AL1,CNY\n")
	plan, err := csv.BuildRecordPlan("base.Country", raw, map[string]string{
		"Currency": "DefaultCurrencyId/Code",
	})
	if err != nil {
		t.Fatalf("BuildRecordPlan: %v", err)
	}
	if len(plan.Units) != 1 {
		t.Fatalf("unit count = %d, want 1", len(plan.Units))
	}
	unit, ok := plan.Units[0].(recordplan.Unit)
	if !ok {
		t.Fatalf("unit type = %T", plan.Units[0])
	}
	if unit.Values["DefaultCurrencyId/Code"] != "CNY" {
		t.Fatalf("mapped value = %q", unit.Values["DefaultCurrencyId/Code"])
	}
	if unit.Values["Name"] != "Alpha" || unit.Values["Code"] != "AL1" {
		t.Fatalf("unexpected values: %#v", unit.Values)
	}
}

func TestReadTable_BOM(t *testing.T) {
	raw := append([]byte{0xEF, 0xBB, 0xBF}, []byte("Name\nX\n")...)
	table, err := csv.ReadTable(raw)
	if err != nil {
		t.Fatalf("ReadTable: %v", err)
	}
	if len(table.Headers) != 1 || table.Headers[0] != "Name" {
		t.Fatalf("headers = %#v", table.Headers)
	}
	if len(table.Rows) != 1 || table.Rows[0][0] != "X" {
		t.Fatalf("rows = %#v", table.Rows)
	}
}
