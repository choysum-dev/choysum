// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package i18nservice

import "testing"

func TestParseInt32Branches(t *testing.T) {
	cases := []struct {
		in       any
		fallback int
		want     int
	}{
		{nil, 7, 7},
		{int(3), 0, 3},
		{int32(4), 0, 4},
		{int64(5), 0, 5},
		{float64(6.9), 0, 6},
		{"", 9, 9},
		{"<nil>", 9, 9},
		{"12", 0, 12},
		{"x", 8, 8},
		{true, 3, 3},
		{false, 0, 0},
	}
	for _, tc := range cases {
		if got := parseInt32(tc.in, tc.fallback); got != tc.want {
			t.Fatalf("parseInt32(%v,%d)=%d want %d", tc.in, tc.fallback, got, tc.want)
		}
	}
}
