// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package generator

import "testing"

func TestParseModelOrderByJSON(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		raw  string
		want []ModelOrderBySpec
	}{
		{name: "empty", raw: "", want: nil},
		{name: "whitespace", raw: "   ", want: nil},
		{name: "invalid json", raw: "{", want: nil},
		{name: "empty array", raw: "[]", want: nil},
		{name: "happy path", raw: `[{"field":"Name","order":"asc"},{"field":"Seq","order":"DESC"}]`, want: []ModelOrderBySpec{
			{Field: "Name", Order: "asc"},
			{Field: "Seq", Order: "desc"},
		}},
		{name: "default empty order to asc", raw: `[{"field":"Name","order":""}]`, want: []ModelOrderBySpec{
			{Field: "Name", Order: "asc"},
		}},
		{name: "skip empty field", raw: `[{"field":"","order":"asc"},{"field":"Ok","order":"desc"}]`, want: []ModelOrderBySpec{
			{Field: "Ok", Order: "desc"},
		}},
		{name: "skip non-identifier", raw: `[{"field":"Owner'sName","order":"asc"}]`, want: nil},
		{name: "skip invalid order", raw: `[{"field":"Name","order":"DESCENDING"}]`, want: nil},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := parseModelOrderByJSON(tc.raw)
			if len(got) != len(tc.want) {
				t.Fatalf("len=%d want %d (%#v)", len(got), len(tc.want), got)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("[%d]=%#v want %#v", i, got[i], tc.want[i])
				}
			}
		})
	}
}
