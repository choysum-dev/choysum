// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package parser

import (
	"fmt"
	"strconv"
	"strings"
)

// ParseJSStringLiteral unquotes a JS/TS string or no-substitution template literal.
func ParseJSStringLiteral(text string) (string, error) {
	text = strings.TrimSpace(text)
	if len(text) < 2 {
		return "", fmt.Errorf("not a string literal")
	}
	if text[0] == '`' && text[len(text)-1] == '`' {
		return unquoteJSTemplateLiteral(text)
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

// unquoteJSTemplateLiteral unescapes a no-substitution `…` literal.
func unquoteJSTemplateLiteral(text string) (string, error) {
	inner := text[1 : len(text)-1]
	var b strings.Builder
	b.Grow(len(inner) + 2)
	b.WriteByte('"')
	for i := 0; i < len(inner); i++ {
		c := inner[i]
		if c == '\\' && i+1 < len(inner) {
			next := inner[i+1]
			switch next {
			case '`':
				b.WriteByte('`')
			case '\n', '\r':
				// Line continuation: omit the escape and the newline.
			default:
				b.WriteByte('\\')
				b.WriteByte(next)
			}
			i++
			continue
		}
		switch c {
		case '"':
			b.WriteString(`\"`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		default:
			b.WriteByte(c)
		}
	}
	b.WriteByte('"')
	return strconv.Unquote(b.String())
}
