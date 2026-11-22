// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package backendtsparser

import "testing"

func TestEncodeModelOrderByOption(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		raw  any
		want string
	}{
		{name: "nil", raw: nil, want: ""},
		{name: "string shorthand", raw: "Name", want: ""},
		{name: "object asc default", raw: map[string]any{"field": "Name"}, want: `[{"field":"Name","order":"asc"}]`},
		{name: "object desc case", raw: map[string]any{"field": "Seq", "order": "DESC"}, want: `[{"field":"Seq","order":"desc"}]`},
		{name: "array mixed", raw: []any{
			map[string]any{"field": "Seq", "order": "desc"},
			map[string]any{"field": "Name", "order": "asc"},
		}, want: `[{"field":"Seq","order":"desc"},{"field":"Name","order":"asc"}]`},
		{name: "skips non-object items", raw: []any{"Name", map[string]any{"field": "Ok", "order": "asc"}}, want: `[{"field":"Ok","order":"asc"}]`},
		{name: "skips empty field", raw: map[string]any{"field": "  ", "order": "asc"}, want: ""},
		{name: "skips non-identifier field", raw: map[string]any{"field": "Owner'sName", "order": "asc"}, want: ""},
		{name: "skips invalid order", raw: map[string]any{"field": "Name", "order": "DESCENDING"}, want: ""},
		{name: "skips non-string field", raw: map[string]any{"field": 12, "order": "asc"}, want: ""},
		{name: "skips non-string order treated empty then asc", raw: map[string]any{"field": "Name", "order": 1}, want: `[{"field":"Name","order":"asc"}]`},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := encodeModelOrderByOption(tc.raw)
			if got != tc.want {
				t.Fatalf("encodeModelOrderByOption() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestFmtString(t *testing.T) {
	t.Parallel()
	if got := fmtString(nil); got != "" {
		t.Fatalf("fmtString(nil) = %q", got)
	}
	if got := fmtString(42); got != "" {
		t.Fatalf("fmtString(42) = %q", got)
	}
	if got := fmtString("Name"); got != "Name" {
		t.Fatalf("fmtString(Name) = %q", got)
	}
}
