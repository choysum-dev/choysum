// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package langcode

import "testing"

func TestValid(t *testing.T) {
	for _, lang := range []string{"zh_CN", "en", "en-US", "pt_BR", "a1"} {
		if !Valid(lang) {
			t.Fatalf("Valid(%q) = false, want true", lang)
		}
	}
	for _, lang := range []string{
		"",
		"../zh_CN",
		"zh/CN",
		"zh\\CN",
		"zh CN",
		"zh.CN",
		stringsRepeat("x", MaxLen+1),
	} {
		if Valid(lang) {
			t.Fatalf("Valid(%q) = true, want false", lang)
		}
	}
}

func stringsRepeat(s string, n int) string {
	out := make([]byte, 0, len(s)*n)
	for i := 0; i < n; i++ {
		out = append(out, s...)
	}
	return string(out)
}
