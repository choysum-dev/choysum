// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package extract

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/choysum-dev/choysum/internal/parser"
)

// Vue template metadata extract (S7): field label= / search-view-title=.

var (
	// O*Field / *Field opening tags (self-closing or with children).
	reMetaFieldTag = regexp.MustCompile(`(?is)<((?:O[\w]+)|(?:[\w-]*Field))\b([^>]*?)(/?)>`)

	reAttrPropDouble   = regexp.MustCompile(`(?i)\bprop\s*=\s*"([^"]*)"`)
	reAttrPropSingle   = regexp.MustCompile(`(?i)\bprop\s*=\s*'([^']*)'`)
	reAttrLabelDouble  = regexp.MustCompile(`(?i)\blabel\s*=\s*"([^"]*)"`)
	reAttrLabelSingle  = regexp.MustCompile(`(?i)\blabel\s*=\s*'([^']*)'`)
	reAttrLabelBindDbl = regexp.MustCompile(`(?i)(?::|v-bind:)label\s*=\s*"\s*(['"].*?['"])\s*"`)
	reAttrLabelBindSgl = regexp.MustCompile(`(?i)(?::|v-bind:)label\s*=\s*'\s*(["'].*?["'])\s*'`)

	reSearchViewTitleDbl = regexp.MustCompile(`(?i)\bsearch-view-title\s*=\s*"([^"]*)"`)
	reSearchViewTitleSgl = regexp.MustCompile(`(?i)\bsearch-view-title\s*=\s*'([^']*)'`)
)

// CollectMetadataVueTemplate extracts field_label terms from Vue template HTML.
func CollectMetadataVueTemplate(opts CollectOptions, templateHTML string) ([]TermOccurrence, []ExtractIssue) {
	scopePath := ScopePathFromRelPath(opts.RelPath)
	var terms []TermOccurrence
	var issues []ExtractIssue

	for _, m := range reMetaFieldTag.FindAllStringSubmatchIndex(templateHTML, -1) {
		if len(m) < 6 {
			continue
		}
		attrs := templateHTML[m[4]:m[5]]
		offset := m[0]
		line, col := offsetToLineCol(templateHTML, offset)

		prop := firstSubmatch(attrs, reAttrPropDouble, reAttrPropSingle)
		labelSrc, labelOK, labelWarn := metaLabelFromAttrs(attrs)
		if labelWarn != "" {
			issues = append(issues, ExtractIssue{
				Severity:   IssueSeverityWarn,
				Code:       IssueNonLiteralMsgid,
				Message:    labelWarn,
				SourcePath: opts.RelPath,
				Line:       line,
				Col:        col,
			})
		}
		if !labelOK || labelSrc == "" {
			continue
		}
		location := prop
		if location == "" {
			location = "label"
		}
		terms = append(terms, TermOccurrence{
			Module:     opts.ModuleName,
			Scope:      FormatScope(scopePath, location),
			Src:        labelSrc,
			Kind:       KindFieldLabel,
			SourcePath: opts.RelPath,
			Line:       line,
			Col:        col,
		})
	}

	for _, re := range []*regexp.Regexp{reSearchViewTitleDbl, reSearchViewTitleSgl} {
		for _, m := range re.FindAllStringSubmatchIndex(templateHTML, -1) {
			if len(m) < 4 {
				continue
			}
			raw := templateHTML[m[2]:m[3]]
			line, col := offsetToLineCol(templateHTML, m[0])
			terms = append(terms, TermOccurrence{
				Module:     opts.ModuleName,
				Scope:      FormatScope(scopePath, "searchViewTitle"),
				Src:        raw,
				Kind:       KindFieldLabel,
				SourcePath: opts.RelPath,
				Line:       line,
				Col:        col,
			})
		}
	}

	return terms, issues
}

func firstSubmatch(attrs string, res ...*regexp.Regexp) string {
	for _, re := range res {
		if m := re.FindStringSubmatch(attrs); len(m) >= 2 {
			return m[1]
		}
	}
	return ""
}

func metaLabelFromAttrs(attrs string) (src string, ok bool, warn string) {
	// Prefer :label / v-bind:label before plain label= (plain regex also matches ":label").
	for _, re := range []*regexp.Regexp{reAttrLabelBindDbl, reAttrLabelBindSgl} {
		if m := re.FindStringSubmatch(attrs); len(m) >= 2 {
			raw := strings.TrimSpace(m[1])
			lit, err := parser.ParseJSStringLiteral(raw)
			if err != nil {
				if len(raw) >= 2 && raw[0] == '\'' && raw[len(raw)-1] == '\'' {
					return raw[1 : len(raw)-1], true, ""
				}
				return "", false, fmt.Sprintf("non-literal :label; skipped: %s", raw)
			}
			return lit, true, ""
		}
	}
	if m := reAttrLabelDouble.FindStringSubmatchIndex(attrs); len(m) >= 4 {
		if m[0] == 0 || attrs[m[0]-1] != ':' {
			return attrs[m[2]:m[3]], true, ""
		}
	}
	if m := reAttrLabelSingle.FindStringSubmatchIndex(attrs); len(m) >= 4 {
		if m[0] == 0 || attrs[m[0]-1] != ':' {
			return attrs[m[2]:m[3]], true, ""
		}
	}
	return "", false, ""
}
