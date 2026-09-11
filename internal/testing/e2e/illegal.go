// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package e2e

import (
	"os"
	"regexp"
	"strings"

	xfmt "golang.org/x/exp/errors/fmt"
)

// Illegal marks that must not appear in QJS-only e2e specs.
// Patterns allow whitespace/newlines between import(/require( and the module string
// so multiline dynamic imports are rejected. Specifiers may use ', ", or `.
// Optional subpaths (node:fs/promises, @playwright/test/reporter) are also rejected.
var (
	illegalPlaywrightImportRE = regexp.MustCompile(
		`(?s)(?:\bfrom\s+|import\s*\(|require\s*\()\s*['"` + "`" + `]@playwright/test(?:/[^'"` + "`" + `\s]+)?['"` + "`" + `]|(?:^|[^\w$])import\s+['"` + "`" + `]@playwright/test(?:/[^'"` + "`" + `\s]+)?['"` + "`" + `]`,
	)
	illegalNodeBuiltinRE = regexp.MustCompile(
		`(?s)(?:\bfrom\s+|import\s*\(|require\s*\()\s*['"` + "`" + `]node:(?:fs|path|crypto)(?:/[^'"` + "`" + `\s]+)?['"` + "`" + `]|(?:^|[^\w$])import\s+['"` + "`" + `]node:(?:fs|path|crypto)(?:/[^'"` + "`" + `\s]+)?['"` + "`" + `]`,
	)
)

// IllegalE2EMark describes one forbidden import/require found in a spec file.
type IllegalE2EMark struct {
	Path string
	Kind string // "playwright" or "node-builtin"
	Line string
}

// ScanIllegalE2EMarks reports forbidden Playwright / Node builtin usages in specs.
// Comments and non-module string literals are blanked so docs/fixtures do not false-positive;
// module-specifier strings after from/import/require remain scannable.
func ScanIllegalE2EMarks(specFiles []string) ([]IllegalE2EMark, error) {
	var out []IllegalE2EMark
	for _, path := range specFiles {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, xfmt.Errorf("read %s: %w", path, err)
		}
		source := string(raw)
		code := blankJSCommentsAndNonModuleStrings(source)
		out = append(out, findIllegalMarks(path, source, code, illegalPlaywrightImportRE, "playwright")...)
		out = append(out, findIllegalMarks(path, source, code, illegalNodeBuiltinRE, "node-builtin")...)
	}
	return out, nil
}

func findIllegalMarks(path, source, code string, re *regexp.Regexp, kind string) []IllegalE2EMark {
	idxs := re.FindAllStringIndex(code, -1)
	if len(idxs) == 0 {
		return nil
	}
	lines := strings.Split(source, "\n")
	out := make([]IllegalE2EMark, 0, len(idxs))
	seen := map[int]bool{}
	lastIdx := 0
	lineNo := 1
	for _, loc := range idxs {
		for i := lastIdx; i < loc[0] && i < len(code); i++ {
			if code[i] == '\n' {
				lineNo++
			}
		}
		lastIdx = loc[0]
		if seen[lineNo] {
			continue
		}
		seen[lineNo] = true
		snippet := ""
		if lineNo >= 1 && lineNo <= len(lines) {
			snippet = strings.TrimSpace(lines[lineNo-1])
		}
		if snippet == "" && loc[0] < loc[1] && loc[1] <= len(code) {
			snippet = strings.TrimSpace(code[loc[0]:loc[1]])
		}
		out = append(out, IllegalE2EMark{Path: path, Kind: kind, Line: snippet})
	}
	return out
}

// CheckIllegalE2EMarks fails when any scanned spec contains illegal marks.
func CheckIllegalE2EMarks(specFiles []string) error {
	marks, err := ScanIllegalE2EMarks(specFiles)
	if err != nil {
		return err
	}
	if len(marks) == 0 {
		return nil
	}
	var b strings.Builder
	b.WriteString("e2e: illegal marks in specs (QJS-only; no @playwright/test or node:fs|path|crypto):")
	for _, m := range marks {
		b.WriteString("\n  ")
		b.WriteString(m.Path)
		b.WriteString(" [")
		b.WriteString(m.Kind)
		b.WriteString("]: ")
		b.WriteString(m.Line)
	}
	return xfmt.Errorf("%s", b.String())
}

// blankJSCommentsAndNonModuleStrings blanks // and /* */ comments and string/template
// literals that are not import/require/from module specifiers, preserving newlines.
func blankJSCommentsAndNonModuleStrings(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	i := 0
	for i < len(s) {
		if i+1 < len(s) && s[i] == '/' && s[i+1] == '/' {
			for i < len(s) && s[i] != '\n' {
				b.WriteByte(' ')
				i++
			}
			continue
		}
		if i+1 < len(s) && s[i] == '/' && s[i+1] == '*' {
			b.WriteByte(' ')
			b.WriteByte(' ')
			i += 2
			for i < len(s) {
				if i+1 < len(s) && s[i] == '*' && s[i+1] == '/' {
					b.WriteByte(' ')
					b.WriteByte(' ')
					i += 2
					break
				}
				if s[i] == '\n' {
					b.WriteByte('\n')
				} else {
					b.WriteByte(' ')
				}
				i++
			}
			continue
		}
		if s[i] == '/' && canStartJSRegexp(s, i) {
			// Blank top-level regexp literals so pattern text cannot false-positive illegal imports.
			end := skipJSRegexpLiteral(s, i)
			for i < end {
				if s[i] == '\n' {
					b.WriteByte('\n')
				} else {
					b.WriteByte(' ')
				}
				i++
			}
			continue
		}
		if s[i] == '\'' || s[i] == '"' || s[i] == '`' {
			quote := s[i]
			if isModuleSpecifierContext(s, i) {
				// Keep module specifier text so import/require regexes still match.
				b.WriteByte(s[i])
				i++
				for i < len(s) {
					ch := s[i]
					b.WriteByte(ch)
					i++
					if ch == '\\' && i < len(s) {
						b.WriteByte(s[i])
						i++
						continue
					}
					if ch == quote {
						break
					}
				}
				continue
			}
			b.WriteByte(' ')
			i++
			for i < len(s) {
				if s[i] == '\\' && i+1 < len(s) {
					if s[i+1] == '\n' {
						b.WriteByte(' ')
						b.WriteByte('\n')
					} else {
						b.WriteByte(' ')
						b.WriteByte(' ')
					}
					i += 2
					continue
				}
				if quote == '`' && s[i] == '$' && i+1 < len(s) && s[i+1] == '{' {
					// Keep ${...} executable so import()/require() inside interpolations stay visible.
					b.WriteByte('$')
					b.WriteByte('{')
					i += 2
					end := findTemplateInterpClose(s, i)
					bodyEnd := end
					if bodyEnd > i && s[bodyEnd-1] == '}' {
						bodyEnd--
					}
					b.WriteString(blankJSCommentsAndNonModuleStrings(s[i:bodyEnd]))
					if bodyEnd < end {
						b.WriteByte('}')
					}
					i = end
					continue
				}
				ch := s[i]
				if ch == quote {
					b.WriteByte(' ')
					i++
					break
				}
				if ch == '\n' {
					b.WriteByte('\n')
				} else {
					b.WriteByte(' ')
				}
				i++
			}
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

// findTemplateInterpClose returns the index after the matching '}' for a
// `${` started at start (start points at the first body byte). Braces inside
// comments, strings, nested templates, and regexp literals are ignored.
func findTemplateInterpClose(s string, start int) int {
	depth := 1
	i := start
	for i < len(s) && depth > 0 {
		c := s[i]
		if c == '/' && i+1 < len(s) {
			if s[i+1] == '/' {
				i += 2
				for i < len(s) && s[i] != '\n' {
					i++
				}
				continue
			}
			if s[i+1] == '*' {
				i += 2
				for i+1 < len(s) && !(s[i] == '*' && s[i+1] == '/') {
					i++
				}
				if i+1 < len(s) {
					i += 2
				} else {
					i = len(s)
				}
				continue
			}
			if canStartJSRegexp(s, i) {
				i = skipJSRegexpLiteral(s, i)
				continue
			}
		}
		if c == '\'' || c == '"' {
			i = skipJSQuoted(s, i)
			continue
		}
		if c == '`' {
			i = skipJSTemplateLiteral(s, i)
			continue
		}
		if c == '{' {
			depth++
		} else if c == '}' {
			depth--
		}
		i++
	}
	return i
}

func canStartJSRegexp(s string, slashIdx int) bool {
	j := slashIdx - 1
	for j >= 0 && (s[j] == ' ' || s[j] == '\t' || s[j] == '\n' || s[j] == '\r') {
		j--
	}
	if j < 0 {
		return true
	}
	c := s[j]
	if c >= '0' && c <= '9' {
		return false
	}
	if isJSIdentByte(c) {
		start := j
		for start > 0 && isJSIdentByte(s[start-1]) {
			start--
		}
		switch s[start : j+1] {
		case "return", "case", "throw", "typeof", "void", "delete", "await", "new", "of", "in", "instanceof", "else", "do", "yield":
			return true
		default:
			return false
		}
	}
	switch c {
	case ')', ']', '}':
		return false
	case '"', '\'', '`':
		return false
	case '+', '-':
		if j > 0 && s[j-1] == c {
			return false // ++ / --
		}
		return true
	default:
		return true
	}
}

func isJSIdentByte(c byte) bool {
	return c == '_' || c == '$' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

// skipJSRegexpLiteral advances past /pattern/flags starting at s[i] == '/'.
// Character-class braces (e.g. /[}]/) are not treated as interpolation closers.
func skipJSRegexpLiteral(s string, i int) int {
	if i >= len(s) || s[i] != '/' {
		return i
	}
	i++
	inClass := false
	for i < len(s) {
		c := s[i]
		if c == '\\' && i+1 < len(s) {
			i += 2
			continue
		}
		if c == '\n' {
			return i + 1 // include newline so blankers can preserve it
		}
		if c == '[' && !inClass {
			inClass = true
			i++
			continue
		}
		if c == ']' && inClass {
			inClass = false
			i++
			continue
		}
		if c == '/' && !inClass {
			i++
			for i < len(s) && isJSRegexpFlag(s[i]) {
				i++
			}
			return i
		}
		i++
	}
	return i
}

func isJSRegexpFlag(c byte) bool {
	switch c {
	case 'd', 'g', 'i', 'm', 's', 'u', 'v', 'y':
		return true
	default:
		return false
	}
}

func skipJSQuoted(s string, i int) int {
	q := s[i]
	i++
	for i < len(s) {
		if s[i] == '\\' && i+1 < len(s) {
			i += 2
			continue
		}
		if s[i] == q {
			return i + 1
		}
		i++
	}
	return i
}

func skipJSTemplateLiteral(s string, i int) int {
	i++ // opening `
	for i < len(s) {
		if s[i] == '\\' && i+1 < len(s) {
			i += 2
			continue
		}
		if s[i] == '`' {
			return i + 1
		}
		if s[i] == '$' && i+1 < len(s) && s[i+1] == '{' {
			i = findTemplateInterpClose(s, i+2)
			continue
		}
		i++
	}
	return i
}

// isModuleSpecifierContext reports whether a quote at idx is the module string of
// from/import/require (possibly across whitespace/newlines).
func isModuleSpecifierContext(s string, idx int) bool {
	j := idx - 1
	for j >= 0 && (s[j] == ' ' || s[j] == '\t' || s[j] == '\n' || s[j] == '\r') {
		j--
	}
	if j < 0 {
		return false
	}
	// from 'x' / import('x') / require('x') / import 'x'
	if s[j] == '(' {
		j--
		for j >= 0 && (s[j] == ' ' || s[j] == '\t' || s[j] == '\n' || s[j] == '\r') {
			j--
		}
		return hasIdentBefore(s, j, "import") || hasIdentBefore(s, j, "require")
	}
	return hasIdentBefore(s, j, "from") || hasIdentBefore(s, j, "import")
}

func hasIdentBefore(s string, end int, ident string) bool {
	n := len(ident)
	if end+1 < n {
		return false
	}
	start := end - n + 1
	if s[start:end+1] != ident {
		return false
	}
	if start > 0 {
		c := s[start-1]
		if c == '_' || c == '$' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			return false
		}
	}
	return true
}
