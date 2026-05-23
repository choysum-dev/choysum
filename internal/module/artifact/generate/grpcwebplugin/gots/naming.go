// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package gots

import (
	"path/filepath"
	"strings"
	"unicode"
)

var tsReservedWords = map[string]struct{}{
	"break": {}, "case": {}, "catch": {}, "class": {}, "const": {}, "continue": {},
	"debugger": {}, "default": {}, "delete": {}, "do": {}, "else": {}, "enum": {},
	"export": {}, "extends": {}, "false": {}, "finally": {}, "for": {}, "function": {},
	"if": {}, "import": {}, "in": {}, "instanceof": {}, "new": {}, "null": {},
	"return": {}, "super": {}, "switch": {}, "this": {}, "throw": {}, "true": {},
	"try": {}, "typeof": {}, "var": {}, "void": {}, "while": {}, "with": {},
	"as": {}, "implements": {}, "interface": {}, "let": {}, "package": {}, "private": {},
	"protected": {}, "public": {}, "static": {}, "yield": {}, "any": {}, "boolean": {},
	"constructor": {}, "declare": {}, "get": {}, "module": {}, "require": {}, "number": {},
	"set": {}, "string": {}, "symbol": {}, "type": {}, "from": {}, "of": {},
}

func ProtoFileToTSFile(path string) string {
	if strings.HasSuffix(path, ".proto") {
		return strings.TrimSuffix(path, ".proto") + "_pb.ts"
	}
	return path + "_pb.ts"
}

func ProtoFileToFileConst(path string) string {
	name := strings.TrimSuffix(path, ".proto")
	name = filepath.ToSlash(name)
	name = strings.ReplaceAll(name, "/", "_")
	name = sanitizeIdentifier(name)
	if name == "" {
		name = "file"
	}
	return EscapeTSIdentifier("file_" + name)
}

func ProtoFieldToTSField(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "_"
	}

	if strings.Contains(name, "_") {
		parts := splitIdentifierParts(name)
		if len(parts) == 0 {
			return "_"
		}
		first := strings.ToLower(parts[0])
		for _, p := range parts[1:] {
			first += upperFirst(strings.ToLower(p))
		}
		return EscapeTSIdentifier(first)
	}

	return EscapeTSIdentifier(name)
}

func ProtoNameToExport(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "_"
	}
	if !strings.Contains(name, "_") {
		name = sanitizeIdentifier(name)
		if name == "" {
			return "_"
		}
		return EscapeTSIdentifier(upperFirst(name))
	}

	parts := splitIdentifierParts(name)
	if len(parts) == 0 {
		return "_"
	}
	var b strings.Builder
	for _, p := range parts {
		// Keep inner casing in each underscore-delimited segment. This preserves
		// identifiers like IrApplication_* instead of collapsing to Irapplication*.
		b.WriteString(upperFirst(p))
	}
	return EscapeTSIdentifier(b.String())
}

func NestedNamesToExport(parts []string) string {
	partsOut := make([]string, 0, len(parts))
	for _, p := range parts {
		n := ProtoNameToExport(p)
		if n == "_" {
			continue
		}
		partsOut = append(partsOut, n)
	}
	if len(partsOut) == 0 {
		return "_"
	}
	return EscapeTSIdentifier(strings.Join(partsOut, "_"))
}

func EscapeTSIdentifier(name string) string {
	name = sanitizeIdentifier(name)
	if name == "" {
		name = "_"
	}
	if !isIdentifierStart(rune(name[0])) {
		name = "_" + name
	}
	if _, ok := tsReservedWords[name]; ok {
		name += "_"
	}
	return name
}

func splitIdentifierParts(s string) []string {
	s = sanitizeIdentifier(s)
	if s == "" {
		return nil
	}
	raw := strings.Split(s, "_")
	parts := make([]string, 0, len(raw))
	for _, p := range raw {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		parts = append(parts, p)
	}
	return parts
}

func sanitizeIdentifier(s string) string {
	var b strings.Builder
	for _, r := range s {
		if isIdentifierContinue(r) {
			b.WriteRune(r)
			continue
		}
		b.WriteByte('_')
	}
	out := b.String()
	for strings.Contains(out, "__") {
		out = strings.ReplaceAll(out, "__", "_")
	}
	return strings.Trim(out, "_")
}

func upperFirst(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

func isIdentifierStart(r rune) bool {
	return r == '_' || r == '$' || unicode.IsLetter(r)
}

func isIdentifierContinue(r rune) bool {
	return isIdentifierStart(r) || unicode.IsDigit(r)
}
