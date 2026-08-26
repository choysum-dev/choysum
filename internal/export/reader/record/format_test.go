// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package record

import (
	"testing"
)

func TestFormatCell_scalarAndM2O(t *testing.T) {
	tests := []struct {
		name   string
		record map[string]any
		field  string
		want   string
	}{
		{name: "string", record: map[string]any{"Name": "Alpha"}, field: "Name", want: "Alpha"},
		{name: "bool", record: map[string]any{"IsActive": true}, field: "IsActive", want: "true"},
		{name: "int", record: map[string]any{"Count": 7}, field: "Count", want: "7"},
		{name: "int64", record: map[string]any{"Count": int64(9)}, field: "Count", want: "9"},
		{name: "float", record: map[string]any{"Rate": 1.5}, field: "Rate", want: "1.5"},
		{name: "float whole", record: map[string]any{"Rate": float64(2)}, field: "Rate", want: "2"},
		{name: "nil", record: map[string]any{"Name": nil}, field: "Name", want: ""},
		{name: "slash flat", record: map[string]any{"DefaultCurrencyId/Code": "CNY"}, field: "DefaultCurrencyId/Code", want: "CNY"},
		{name: "slash nested", record: map[string]any{"DefaultCurrencyId": map[string]any{"Code": "USD"}}, field: "DefaultCurrencyId/Code", want: "USD"},
		{name: "slash missing", record: map[string]any{}, field: "DefaultCurrencyId/Code", want: ""},
		{name: "default type", record: map[string]any{"X": struct{ V string }{V: "z"}}, field: "X", want: "{z}"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := formatCell(tc.record, tc.field); got != tc.want {
				t.Fatalf("formatCell() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestRecordRows_empty(t *testing.T) {
	if got := recordRows(nil, []string{"Name"}); got != nil {
		t.Fatalf("recordRows(nil) = %#v", got)
	}
}

func TestParseSearchRecords_errors(t *testing.T) {
	if rows, err := parseSearchRecords(nil); err != nil || rows != nil {
		t.Fatalf("nil result = %#v, err = %v", rows, err)
	}
	if _, err := parseSearchRecords("bad"); err == nil {
		t.Fatal("expected type error")
	}
	if _, err := parseSearchRecords([]any{"bad"}); err == nil {
		t.Fatal("expected record type error")
	}
}
