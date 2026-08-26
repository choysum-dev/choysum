// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package terms

import (
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/internal/i18n/po"
)

func TestBuildPOEntriesKindAndReferences(t *testing.T) {
	entries := BuildPOEntries("zh_CN", []Item{{
		Scope:    "a@b",
		Src:      "Title",
		Value:    "标题",
		Kind:     "menu",
		Module:   "auth",
		Source:   "po",
		Comments: "Login.vue:10 Login.vue:11",
		Status:   "translated",
	}})
	var body *po.Entry
	for i := range entries {
		if entries[i].Msgid == "Title" {
			body = &entries[i]
			break
		}
	}
	if body == nil {
		t.Fatal("entry missing")
	}
	if !strings.Contains(strings.Join(body.ExtractedComments, " "), "kind: menu") {
		t.Fatalf("extracted = %#v", body.ExtractedComments)
	}
	if len(body.References) != 2 || body.References[0] != "Login.vue:10" {
		t.Fatalf("refs = %#v", body.References)
	}
	if !strings.Contains(strings.Join(body.TranslatorComments, " "), "source: po") {
		t.Fatalf("translator = %#v", body.TranslatorComments)
	}
}

func TestBuildPOEntriesOmitsLiteralKindAndMarksFuzzy(t *testing.T) {
	entries := BuildPOEntries("zh_CN", []Item{{
		Scope:  "a@b",
		Src:    "Hello",
		Value:  "你好",
		Kind:   "literal",
		Status: "fuzzy",
	}})
	var hello *po.Entry
	for i := range entries {
		if entries[i].Msgid == "Hello" {
			hello = &entries[i]
			break
		}
	}
	if hello == nil {
		t.Fatal("entry missing")
	}
	for _, c := range hello.ExtractedComments {
		if strings.HasPrefix(strings.ToLower(c), "kind:") {
			t.Fatalf("literal kind should be omitted: %#v", hello.ExtractedComments)
		}
	}
	if len(hello.Flags) != 1 || hello.Flags[0] != "fuzzy" {
		t.Fatalf("flags = %#v", hello.Flags)
	}
}

func TestMoveHeaderFirst(t *testing.T) {
	entries := []po.Entry{
		{Msgid: "a", Msgstr: "A"},
		{Msgid: "", Msgstr: "header"},
		{Msgid: "b", Msgstr: "B"},
	}
	got := moveHeaderFirst(entries)
	if !po.IsHeader(got[0]) || got[1].Msgid != "a" {
		t.Fatalf("order = %#v", got)
	}
}
