// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package terms

// Item is one TranslationTerm row used for PO export.
type Item struct {
	Application string
	Module      string
	Scope       string
	Src         string
	Value       string
	Kind        string
	Source      string
	Comments    string
	Status      string
}

// SearchResult is one TranslationTerm Search page.
type SearchResult struct {
	Lang   string
	Items  []Item
	Total  int64
	Limit  int
	Offset int
}
