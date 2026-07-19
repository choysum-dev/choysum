// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package parser

import "testing"

func TestParseJSStringLiteralQuotesAndTemplates(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{in: `"hello\nworld"`, want: "hello\nworld"},
		{in: `'hello'`, want: "hello"},
		{in: "`line\\nbreak`", want: "line\nbreak"},
		{in: "`has\\`tick`", want: "has`tick"},
		{in: "`plain`", want: "plain"},
	}
	for _, tt := range tests {
		got, err := ParseJSStringLiteral(tt.in)
		if err != nil {
			t.Fatalf("ParseJSStringLiteral(%q) error = %v", tt.in, err)
		}
		if got != tt.want {
			t.Fatalf("ParseJSStringLiteral(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
