// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package parser

import (
	"fmt"
	"strconv"
	"strings"
)

// ParseJSStringLiteral unquotes a JS/TS single- or double-quoted string literal.
func ParseJSStringLiteral(text string) (string, error) {
	text = strings.TrimSpace(text)
	if len(text) < 2 {
		return "", fmt.Errorf("not a string literal")
	}
	if (text[0] != '\'' || text[len(text)-1] != '\'') && (text[0] != '"' || text[len(text)-1] != '"') {
		return "", fmt.Errorf("not a string literal")
	}
	value, err := strconv.Unquote(text)
	if err == nil {
		return value, nil
	}
	if text[0] == '\'' {
		value, err = strconv.Unquote("\"" + strings.ReplaceAll(text[1:len(text)-1], "\"", "\\\"") + "\"")
		if err == nil {
			return value, nil
		}
	}
	return "", err
}
