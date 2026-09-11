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
// so multiline dynamic imports are rejected.
var (
	illegalPlaywrightImportRE = regexp.MustCompile(
		`(?s)(?:\bfrom\s+|import\s*\(|require\s*\()\s*['"]@playwright/test['"]|(?:^|[^\w$])import\s+['"]@playwright/test['"]`,
	)
	illegalNodeBuiltinRE = regexp.MustCompile(
		`(?s)(?:\bfrom\s+|import\s*\(|require\s*\()\s*['"]node:(?:fs|path|crypto)['"]|(?:^|[^\w$])import\s+['"]node:(?:fs|path|crypto)['"]`,
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
	for _, loc := range idxs {
		lineNo := 1
		for i := 0; i < loc[0] && i < len(code); i++ {
			if code[i] == '\n' {
				lineNo++
			}
		}
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
				ch := s[i]
				if ch == '\\' && i+1 < len(s) {
					b.WriteByte(' ')
					b.WriteByte(' ')
					i += 2
					continue
				}
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
