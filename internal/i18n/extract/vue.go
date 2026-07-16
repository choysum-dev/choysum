// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package extract

import (
	"strings"

	"github.com/antchfx/htmlquery"
	"github.com/choysum-dev/choysum/internal/parser/vueparser/vuesfchtmlparser"
	"golang.org/x/net/html"
)

// CollectVue extracts terms from a Vue SFC: script blocks via AST, template via regex (D12e).
func CollectVue(opts CollectOptions, content string) ([]TermOccurrence, []ExtractIssue) {
	scriptContents, templateHTML, err := splitVueSFC(content)
	if err != nil {
		return nil, []ExtractIssue{{
			Severity:   IssueSeverityWarn,
			Code:       "vue_parse_error",
			Message:    err.Error(),
			SourcePath: opts.RelPath,
			Line:       1,
			Col:        1,
		}}
	}

	var terms []TermOccurrence
	var issues []ExtractIssue

	for _, scriptContent := range scriptContents {
		if strings.TrimSpace(scriptContent) == "" {
			continue
		}
		t, i := CollectScript(opts, scriptContent)
		terms = append(terms, t...)
		issues = append(issues, i...)
	}

	if templateHTML != "" {
		t, i := CollectTemplateRegex(opts, templateHTML)
		terms = append(terms, t...)
		issues = append(issues, i...)
	}

	return terms, issues
}

func splitVueSFC(content string) (scriptContents []string, templateHTML string, err error) {
	scriptNodes, templateNode, _, err := vuesfchtmlparser.ParseVueSfcToHtmlNode(strings.NewReader(content))
	if err != nil {
		return nil, "", err
	}
	for _, scriptNode := range scriptNodes {
		if scriptNode == nil {
			continue
		}
		scriptContents = append(scriptContents, htmlquery.InnerText(scriptNode))
	}
	if templateNode != nil {
		templateHTML = templateInnerHTML(templateNode)
	}
	return scriptContents, templateHTML, nil
}

func templateInnerHTML(templateNode *html.Node) string {
	if templateNode == nil {
		return ""
	}
	rendered, err := vuesfchtmlparser.RenderVueSfcFromHtmlNode(templateNode)
	if err != nil {
		return htmlquery.InnerText(templateNode)
	}
	return rendered
}
