// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package extract

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/choysum-dev/choysum/internal/parser"
)

// Temporary Vue <template> extract (D12e). Keep all regex logic in this file so S1+
// can replace it with a full Vue template AST walker.

var (
	// Matches a whole {{ ... }} interpolation so multiple _t() calls inside one
	// block (e.g. ternaries) can each be extracted via collectTCalls.
	reTemplateMustache = regexp.MustCompile(`\{\{\s*([\s\S]*?)\s*\}\}`)

	// Attribute bindings capture the whole value so multiple _t() calls inside
	// one binding (e.g. ternaries) can each be extracted.
	reTemplateAttrDouble = regexp.MustCompile(`(?::|v-bind:)\w[\w-]*\s*=\s*"([^"]*)"`)
	reTemplateAttrSingle = regexp.MustCompile(`(?::|v-bind:)\w[\w-]*\s*=\s*'([^']*)'`)

	// _t msgid argument inside an already-captured expression/attribute value.
	reTemplateTCall = regexp.MustCompile("(_t)\\s*\\(\\s*(['\"\\x60].*?['\"\\x60]|[^\\s),]+)\\s*(?:,|\\))")
)

// CollectTemplateRegex extracts literal `_t` calls from Vue template HTML text.
// Standalone extraction uses the component path; CollectVue may supply a bound default scope.
func CollectTemplateRegex(opts CollectOptions, templateHTML string) ([]TermOccurrence, []ExtractIssue) {
	return collectTemplateRegex(opts, templateHTML, "")
}

func collectTemplateRegex(opts CollectOptions, templateHTML string, boundScope string) ([]TermOccurrence, []ExtractIssue) {
	scopePath := ScopePathFromRelPath(opts.RelPath)
	scope := strings.TrimSpace(boundScope)
	if scope == "" {
		scope = scopePath
	}

	var terms []TermOccurrence
	var issues []ExtractIssue

	collect := func(msgidExpr string, offset int) {
		line, col := offsetToLineCol(templateHTML, offset)
		src, err := parser.ParseJSStringLiteral(strings.TrimSpace(msgidExpr))
		if err != nil {
			// Bare identifier / expression → non-literal.
			if looksLikeNonLiteral(msgidExpr) {
				issues = append(issues, ExtractIssue{
					Severity:   IssueSeverityWarn,
					Code:       IssueNonLiteralMsgid,
					Message:    fmt.Sprintf("non-literal msgid in template; skipped: %s", strings.TrimSpace(msgidExpr)),
					SourcePath: opts.RelPath,
					Line:       line,
					Col:        col,
				})
			}
			return
		}
		terms = append(terms, TermOccurrence{
			Module:     opts.ModuleName,
			Scope:      scope,
			Src:        src,
			Kind:       KindLiteral,
			SourcePath: opts.RelPath,
			Line:       line,
			Col:        col,
		})
	}

	collectTCalls := func(fragment string, baseOffset int) {
		for _, m := range reTemplateTCall.FindAllStringSubmatchIndex(fragment, -1) {
			if len(m) < 6 {
				continue
			}
			msgid := fragment[m[4]:m[5]]
			collect(msgid, baseOffset+m[0])
		}
	}

	for _, m := range reTemplateMustache.FindAllStringSubmatchIndex(templateHTML, -1) {
		// groups: 0 full, 1 inner expression
		if len(m) < 4 {
			continue
		}
		expr := templateHTML[m[2]:m[3]]
		collectTCalls(expr, m[2])
	}

	for _, re := range []*regexp.Regexp{reTemplateAttrDouble, reTemplateAttrSingle} {
		for _, m := range re.FindAllStringSubmatchIndex(templateHTML, -1) {
			// groups: 0 full, 1 attr value
			if len(m) < 4 {
				continue
			}
			value := templateHTML[m[2]:m[3]]
			collectTCalls(value, m[2])
		}
	}

	return terms, issues
}

func looksLikeNonLiteral(expr string) bool {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return false
	}
	if (expr[0] == '\'' && expr[len(expr)-1] == '\'') ||
		(expr[0] == '"' && expr[len(expr)-1] == '"') ||
		(expr[0] == '`' && expr[len(expr)-1] == '`') {
		return false
	}
	return true
}

func offsetToLineCol(source string, offset int) (int, int) {
	if offset < 0 {
		return 1, 1
	}
	if offset > len(source) {
		offset = len(source)
	}
	line, col := 1, 1
	for i := 0; i < offset; i++ {
		if source[i] == '\n' {
			line++
			col = 1
			continue
		}
		col++
	}
	return line, col
}
