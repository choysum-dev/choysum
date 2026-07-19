// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package po

import (
	"bytes"
	"strings"
	"testing"
)

func TestParseMsgstrWithTabSeparator(t *testing.T) {
	src := "msgctxt \"scope\"\nmsgid \"Hello\"\nmsgstr\t\"你好\"\n"
	entries, err := Parse(strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Msgstr != "你好" {
		t.Fatalf("entries=%#v", entries)
	}
}

func TestUnquotePoShortInput(t *testing.T) {
	for _, in := range []string{"", `"`, `'`} {
		if got := unquotePo(in); got != strings.TrimSpace(in) {
			t.Fatalf("unquotePo(%q) = %q, want trimmed input", in, got)
		}
	}
}

func TestParseFlushesBeforeCommentWithoutBlankLine(t *testing.T) {
	src := `msgctxt "a"
msgid "Hello"
msgstr "你好"
#. kind: literal
msgctxt "b"
msgid "Bye"
msgstr "再见"
`
	entries, err := Parse(strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("entries=%d %#v", len(entries), entries)
	}
	if entries[0].Msgid != "Hello" || len(entries[0].ExtractedComments) != 0 {
		t.Fatalf("first entry should not absorb following kind comment: %+v", entries[0])
	}
	if entries[1].Msgid != "Bye" || entries[1].Kind() != "literal" {
		t.Fatalf("second entry should own kind comment: %+v", entries[1])
	}
}

func TestParseAndWriteRoundTrip(t *testing.T) {
	src := `#: a.ts:1
msgctxt "scope"
msgid "Hello"
msgstr "你好"

#~ msgctxt "old"
#~ msgid "Gone"
#~ msgstr "没了"
`
	entries, err := Parse(strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("entries=%d %#v", len(entries), entries)
	}
	if entries[0].Msgctxt != "scope" || entries[0].Msgid != "Hello" || entries[0].Msgstr != "你好" {
		t.Fatalf("active entry: %+v", entries[0])
	}
	if !entries[1].Obsolete || entries[1].Msgid != "Gone" || entries[1].Msgstr != "没了" {
		t.Fatalf("obsolete entry: %+v", entries[1])
	}

	var buf bytes.Buffer
	if err := Write(&buf, entries); err != nil {
		t.Fatal(err)
	}
	again, err := Parse(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if len(again) != 2 || again[1].Msgstr != "没了" {
		t.Fatalf("round-trip: %#v", again)
	}
}
