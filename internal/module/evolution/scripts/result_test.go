// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package scripts

import "testing"

func TestParseResult(t *testing.T) {
	result, err := ParseResult(map[string]any{"ok": true, "message": "done", "code": "OK"})
	if err != nil || !result.Ok || result.Message != "done" {
		t.Fatalf("unexpected ParseResult output: %#v err=%v", result, err)
	}
	if _, err := ParseResult(make(chan int)); err == nil {
		t.Fatal("expected ParseResult to fail for unsupported raw value")
	}
}
