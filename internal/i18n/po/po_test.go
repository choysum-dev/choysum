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

func TestWriteObsoletePrefixesTranslatorComments(t *testing.T) {
	var buf bytes.Buffer
	if err := Write(&buf, []Entry{{
		TranslatorComments: []string{"retired"},
		Msgctxt:            "old",
		Msgid:              "Gone",
		Msgstr:             "没了",
		Obsolete:           true,
	}}); err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	if !strings.Contains(got, "#~ # retired\n") {
		t.Fatalf("expected obsolete translator comment prefix, got %q", got)
	}
	if strings.Contains(got, "\n# retired\n") {
		t.Fatalf("translator comment must not be active when obsolete, got %q", got)
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

func TestEntryKeyKindHeaderAndSort(t *testing.T) {
	e := Entry{Msgctxt: "a", Msgid: "Hello", ExtractedComments: []string{"kind: custom"}}
	if e.Key() != "a\x00Hello\x00custom" {
		t.Fatalf("Key = %q", e.Key())
	}
	if (Entry{Msgid: "", Msgctxt: ""}).Kind() != "literal" {
		t.Fatal("default kind")
	}
	if (Entry{ExtractedComments: []string{"kind:"}}).Kind() != "literal" {
		t.Fatal("empty kind after prefix")
	}
	if !IsHeader(Entry{Msgid: ""}) || IsHeader(Entry{Msgid: "x"}) || IsHeader(Entry{Msgid: "", Obsolete: true}) {
		t.Fatal("IsHeader mismatch")
	}

	entries := []Entry{
		{Msgctxt: "b", Msgid: "2", Obsolete: true},
		{Msgctxt: "a", Msgid: "2"},
		{Msgctxt: "a", Msgid: "1", ExtractedComments: []string{"kind: z"}},
		{Msgctxt: "a", Msgid: "1", ExtractedComments: []string{"kind: a"}},
	}
	SortEntries(entries)
	if entries[0].Msgid != "1" || entries[0].Kind() != "a" {
		t.Fatalf("sort first = %+v", entries[0])
	}
	if entries[1].Kind() != "z" {
		t.Fatalf("sort second kind = %q", entries[1].Kind())
	}
	if entries[2].Msgctxt != "a" || entries[2].Msgid != "2" {
		t.Fatalf("sort third = %+v", entries[2])
	}
	if !entries[3].Obsolete {
		t.Fatal("obsolete should sort last")
	}
}

func TestParseCommentBranchesAndUnquoteFallback(t *testing.T) {
	src := `# translator note
#. extracted
#: a.ts:1 b.ts:2
#, fuzzy, python-format
msgctxt "scope"
msgid "Hello"
msgstr "你好"
`
	entries, err := Parse(strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("entries=%d", len(entries))
	}
	e := entries[0]
	if len(e.TranslatorComments) != 1 || e.TranslatorComments[0] != "translator note" {
		t.Fatalf("translator = %#v", e.TranslatorComments)
	}
	if len(e.ExtractedComments) != 1 || e.ExtractedComments[0] != "extracted" {
		t.Fatalf("extracted = %#v", e.ExtractedComments)
	}
	if len(e.References) != 2 {
		t.Fatalf("refs = %#v", e.References)
	}
	if len(e.Flags) != 2 || e.Flags[0] != "fuzzy" {
		t.Fatalf("flags = %#v", e.Flags)
	}

	if got := unquotePo(`"broken`); got != `"broken` {
		t.Fatalf("unquotePo broken = %q", got)
	}
	if got := unquotePo(`"ok"`); got != "ok" {
		t.Fatalf("unquotePo ok = %q", got)
	}
}

func TestParseMultilinePluralAndWriteFlags(t *testing.T) {
	src := `msgid ""
"Hello "
"World"
msgstr ""
"你好"
"世界"

msgid "One"
msgstr[0] "singular"

msgctxt "a"
msgid "Alpha"
msgstr "甲"
msgctxt "b"
msgid "Beta"
msgstr "乙"
`
	entries, err := Parse(strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	foundHello := false
	for _, e := range entries {
		if e.Msgid == "Hello World" && e.Msgstr == "你好世界" {
			foundHello = true
		}
	}
	if !foundHello {
		t.Fatalf("multiline missing: %#v", entries)
	}

	if _, err := Parse(strings.NewReader("msgid \"x\"\nfoo bar\n")); err == nil {
		t.Fatal("expected unsupported line error")
	}
	if got := unquotePo(`"\q"`); got != `\q` {
		t.Fatalf("unquotePo fallback = %q", got)
	}

	var buf bytes.Buffer
	if err := Write(&buf, []Entry{
		{
			TranslatorComments: []string{"note"},
			ExtractedComments:  []string{"kind: literal"},
			References:         []string{"a.ts:1"},
			Flags:              []string{"fuzzy"},
			Msgctxt:            "scope",
			Msgid:              "Hello",
			Msgstr:             "你好",
		},
		{
			ExtractedComments: []string{"old"},
			Msgid:             "Gone",
			Msgstr:            "没了",
			Obsolete:          true,
		},
	}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "#, fuzzy") || !strings.Contains(out, "#~ #. old") {
		t.Fatalf("write flags/comments: %q", out)
	}

	entries2 := []Entry{
		{Msgctxt: "b", Msgid: "1"},
		{Msgctxt: "a", Msgid: "1"},
	}
	SortEntries(entries2)
	if entries2[0].Msgctxt != "a" {
		t.Fatalf("msgctxt sort = %#v", entries2)
	}
}
