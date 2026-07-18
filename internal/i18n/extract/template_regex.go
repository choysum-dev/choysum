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
	// Matches {{ _t('...') }} with optional whitespace.
	reTemplateMustache = regexp.MustCompile(`\{\{\s*(_t)\s*\(\s*(['"].*?['"]|[^\s),]+)\s*(?:,|\))`)

	// Attribute bindings: :title="_t('...')" or :title='_t("...")' (Go regexp has no backrefs).
	reTemplateAttrDouble = regexp.MustCompile(`(?::|v-bind:)\w[\w-]*\s*=\s*"\s*(_t)\s*\(\s*(.*?)\s*(?:,|\))\s*"`)
	reTemplateAttrSingle = regexp.MustCompile(`(?::|v-bind:)\w[\w-]*\s*=\s*'\s*(_t)\s*\(\s*(.*?)\s*(?:,|\))\s*'`)
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

	for _, m := range reTemplateMustache.FindAllStringSubmatchIndex(templateHTML, -1) {
		// groups: 0 full, 1 _t, 2 msgid expr
		if len(m) < 6 {
			continue
		}
		msgid := templateHTML[m[4]:m[5]]
		collect(msgid, m[0])
	}

	for _, re := range []*regexp.Regexp{reTemplateAttrDouble, reTemplateAttrSingle} {
		for _, m := range re.FindAllStringSubmatchIndex(templateHTML, -1) {
			// groups: 0 full, 1 _t, 2 msgid expr
			if len(m) < 6 {
				continue
			}
			msgid := templateHTML[m[4]:m[5]]
			collect(msgid, m[0])
		}
	}

	return terms, issues
}

func looksLikeNonLiteral(expr string) bool {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return false
	}
	if (expr[0] == '\'' && expr[len(expr)-1] == '\'') || (expr[0] == '"' && expr[len(expr)-1] == '"') {
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
