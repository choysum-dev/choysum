// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package langcode

// MaxLen matches base.Language.Code (varchar 16).
const MaxLen = 16

// Valid reports whether lang is a safe terminology language code
// (alphanumeric, underscore, hyphen; length ≤ MaxLen).
// Rejects path separators and traversal sequences used in *.po filenames.
func Valid(lang string) bool {
	if lang == "" || len(lang) > MaxLen {
		return false
	}
	for _, r := range lang {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_', r == '-':
		default:
			return false
		}
	}
	return true
}
