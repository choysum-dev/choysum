// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package terms

import (
	"strings"

	"github.com/choysum-dev/choysum/internal/i18n/po"
)

// BuildPOEntries converts collected terms into sorted PO entries with a header block.
func BuildPOEntries(lang string, items []Item) []po.Entry {
	entries := make([]po.Entry, 0, len(items)+1)
	entries = append(entries, po.Entry{
		Msgid: "",
		Msgstr: "Content-Type: text/plain; charset=UTF-8\n" +
			"Content-Transfer-Encoding: 8bit\n" +
			"Language: " + lang + "\n" +
			"X-Generator: choysum-i18n-gateway\n",
	})
	for _, item := range items {
		e := po.Entry{
			Msgctxt: item.Scope,
			Msgid:   item.Src,
			Msgstr:  item.Value,
		}
		if item.Module != "" {
			e.ExtractedComments = append(e.ExtractedComments, "module: "+item.Module)
		}
		if item.Source != "" {
			e.TranslatorComments = append(e.TranslatorComments, "source: "+item.Source)
		}
		if strings.EqualFold(item.Status, "fuzzy") {
			e.Flags = append(e.Flags, "fuzzy")
		}
		entries = append(entries, e)
	}
	po.SortEntries(entries)
	return moveHeaderFirst(entries)
}

func moveHeaderFirst(entries []po.Entry) []po.Entry {
	var header []po.Entry
	var rest []po.Entry
	for _, e := range entries {
		if po.IsHeader(e) {
			header = append(header, e)
			continue
		}
		rest = append(rest, e)
	}
	return append(header, rest...)
}
