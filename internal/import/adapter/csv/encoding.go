// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package csv

import (
	"unicode/utf8"

	importpkg "github.com/choysum-dev/choysum/pkg/import"
)

// StripUTF8BOM removes a leading UTF-8 BOM (EF BB BF) when present.
func StripUTF8BOM(data []byte) []byte {
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		return data[3:]
	}
	return data
}

// PrependUTF8BOM adds a UTF-8 BOM prefix for consumers that expect Excel-friendly CSV.
func PrependUTF8BOM(data []byte) []byte {
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		return data
	}
	out := make([]byte, 0, len(data)+3)
	out = append(out, 0xEF, 0xBB, 0xBF)
	return append(out, data...)
}

// ValidateUTF8 returns an error when data is not valid UTF-8.
func ValidateUTF8(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	if !utf8.Valid(data) {
		return importpkg.Errorf(importpkg.CodeInvalidEncoding, "CSV must be UTF-8 encoded")
	}
	return nil
}
