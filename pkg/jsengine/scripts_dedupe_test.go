// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package jsengine

import "testing"

func TestDedupeInitScripts_ByContentHash_StableOrder(t *testing.T) {
	s1 := &JsScript{Content: "console.log(1)", FileName: "a.js"}
	s2 := &JsScript{Content: "console.log(1)", FileName: "b.js"} // same content
	s3 := &JsScript{Content: "console.log(2)", FileName: "c.js"}

	out, err := DedupeInitScripts([]*JsScript{s1, s2, s3})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("expected 2 scripts, got %d", len(out))
	}
	if out[0] != s1 {
		t.Fatalf("expected first script preserved")
	}
	if out[1] != s3 {
		t.Fatalf("expected third script preserved")
	}
}

func TestDedupeInitScripts_SameFileNameDifferentContent_Errors(t *testing.T) {
	s1 := &JsScript{Content: "console.log(1)", FileName: "same.js"}
	s2 := &JsScript{Content: "console.log(2)", FileName: "same.js"}

	_, err := DedupeInitScripts([]*JsScript{s1, s2})
	if err == nil {
		t.Fatalf("expected error")
	}
}
