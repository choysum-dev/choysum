// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package esmresolver

import (
	"regexp"
	"strings"
)

// Matches a file that is solely one ambient `declare module '…' { … }` wrapper
// (optionally preceded/followed by comments/whitespace). esm.sh emits this
// shape for CommonJS packages such as fast-deep-equal; mapping that file via
// compilerOptions.paths yields TS2306 ("File is not a module") until the body
// is promoted to a real top-level module.
//
// Go's regexp engine does not support backreferences, so single- and
// double-quoted module names are matched with two alternatives.
var ambientModuleFilePattern = regexp.MustCompile(`(?s)\A\s*(?:(?://[^\n]*\n)|(?:/\*[\s\S]*?\*/)\s*)*declare\s+module\s+(?:'([^']+)'|"([^"]+)")\s*\{`)

var ambientModuleExportPattern = regexp.MustCompile(`(?m)^\s*export\b`)

// Optional leading "async" is consumed but not kept: ambient decls cannot be async.
var topLevelDeclStartPattern = regexp.MustCompile(`^([ \t]*)(?:async\s+)?((?:function\*?|class|enum|namespace|const|let|var)\b)`)

// promoteAmbientModuleForPathsTarget rewrites a single ambient declare-module
// wrapper that already exports something into a top-level .d.ts module suitable
// for compilerOptions.paths targets.
//
// Augmentation-only declare modules (interfaces/types, no export) and files with
// multiple declare modules are left unchanged so vue/pinia/moment bridging keeps
// working.
func promoteAmbientModuleForPathsTarget(content string) string {
	if strings.TrimSpace(content) == "" {
		return content
	}

	loc := ambientModuleFilePattern.FindStringSubmatchIndex(content)
	if loc == nil {
		return content
	}
	bodyOpen := loc[1] // index just after '{'
	if bodyOpen <= 0 || bodyOpen > len(content) {
		return content
	}

	bodyClose := findMatchingBrace(content, bodyOpen-1)
	if bodyClose < 0 {
		return content
	}
	trailing := strings.TrimSpace(content[bodyClose+1:])
	if trailing != "" && !isOnlyComments(trailing) {
		return content
	}

	body := content[bodyOpen:bodyClose]
	if !ambientModuleExportPattern.MatchString(body) {
		return content
	}

	// Reject nested ambient modules — promotion is only safe for a single wrapper.
	if strings.Contains(body, "declare module") {
		return content
	}

	promoted := ensureDeclareOnTopLevelDecls(body)
	promoted = strings.TrimRight(promoted, " \t\r\n") + "\n"
	return promoted
}

func findMatchingBrace(content string, openIdx int) int {
	if openIdx < 0 || openIdx >= len(content) || content[openIdx] != '{' {
		return -1
	}
	depth := 0
	inLineComment := false
	inBlockComment := false
	inSingle := false
	inDouble := false
	inTemplate := false
	escape := false

	for i := openIdx; i < len(content); i++ {
		ch := content[i]
		if inLineComment {
			if ch == '\n' {
				inLineComment = false
			}
			continue
		}
		if inBlockComment {
			if ch == '*' && i+1 < len(content) && content[i+1] == '/' {
				inBlockComment = false
				i++
			}
			continue
		}
		if inSingle {
			if escape {
				escape = false
				continue
			}
			if ch == '\\' {
				escape = true
				continue
			}
			if ch == '\'' {
				inSingle = false
			}
			continue
		}
		if inDouble {
			if escape {
				escape = false
				continue
			}
			if ch == '\\' {
				escape = true
				continue
			}
			if ch == '"' {
				inDouble = false
			}
			continue
		}
		if inTemplate {
			if escape {
				escape = false
				continue
			}
			if ch == '\\' {
				escape = true
				continue
			}
			if ch == '`' {
				inTemplate = false
			}
			continue
		}

		if ch == '/' && i+1 < len(content) {
			next := content[i+1]
			if next == '/' {
				inLineComment = true
				i++
				continue
			}
			if next == '*' {
				inBlockComment = true
				i++
				continue
			}
		}
		switch ch {
		case '\'':
			inSingle = true
		case '"':
			inDouble = true
		case '`':
			inTemplate = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func isOnlyComments(s string) bool {
	rest := strings.TrimSpace(s)
	for rest != "" {
		if strings.HasPrefix(rest, "//") {
			if idx := strings.IndexByte(rest, '\n'); idx >= 0 {
				rest = strings.TrimSpace(rest[idx+1:])
				continue
			}
			return true
		}
		if strings.HasPrefix(rest, "/*") {
			end := strings.Index(rest, "*/")
			if end < 0 {
				return false
			}
			rest = strings.TrimSpace(rest[end+2:])
			continue
		}
		return false
	}
	return true
}

func ensureDeclareOnTopLevelDecls(body string) string {
	lines := strings.Split(body, "\n")
	minIndent := -1
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") || strings.HasPrefix(trimmed, "*") {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " \t"))
		if minIndent == -1 || indent < minIndent {
			minIndent = indent
		}
	}

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") || strings.HasPrefix(trimmed, "*") {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " \t"))
		if minIndent >= 0 && indent != minIndent {
			continue
		}
		if strings.HasPrefix(trimmed, "declare ") || strings.HasPrefix(trimmed, "export ") {
			continue
		}
		loc := topLevelDeclStartPattern.FindStringSubmatchIndex(line)
		if loc == nil {
			continue
		}
		indentStr := line[loc[2]:loc[3]]
		keyword := line[loc[4]:loc[5]]
		rest := line[loc[5]:]
		lines[i] = indentStr + "declare " + keyword + rest
	}
	return strings.Join(lines, "\n")
}
