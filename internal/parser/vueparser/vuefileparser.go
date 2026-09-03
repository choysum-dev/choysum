// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package vueparser

import (
	"path/filepath"
	"strings"

	"github.com/antchfx/htmlquery"
	tscore "github.com/buke/typescript-go-internal/v7/pkg/core"
	"github.com/choysum-dev/choysum/internal/parser"
	"github.com/choysum-dev/choysum/internal/parser/vueparser/vuesfchtmlparser"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	xfmt "golang.org/x/exp/errors/fmt"
	"golang.org/x/net/html"
)

type vueFileParser struct {
	*parser.TsParser
	runtimeScope scope.Scope
}

func (p *vueFileParser) parseScriptBlock(path string, scriptContent string, parserResult *parser.ParserResult) error {
	ctx, err := parseTSGoCtxWithKind(p.PathAlias, path, scriptContent, tscore.ScriptKindTS, true)
	if err != nil {
		return xfmt.Errorf("failed to parse script block with tsgo: %w", err)
	}
	mergeImports(parserResult.Imports, ctx.Imports)
	mergeExports(parserResult.Exports, ctx.Exports)
	parserResult.DynamicImports = mergeDynamicImports(parserResult.DynamicImports, ctx.DynamicImports)

	uiDecls, uiIssues := collectUiResourceDecls(path, scriptContent)
	parserResult.UiResourceDecls = append(parserResult.UiResourceDecls, uiDecls...)
	parserResult.UiResourceDeclIssues = append(parserResult.UiResourceDeclIssues, uiIssues...)

	return nil
}

func (p *vueFileParser) parseScriptNodes(parserResult *parser.ParserResult) error {
	scriptNode := parserResult.RawScriptNode
	scriptSetupNode := parserResult.RawScriptSetupNode
	path := parserResult.Path
	parserResult.Imports = make(map[string]*parser.Import)
	parserResult.Exports = make(map[string]*parser.Export)

	component := &meta.Component{
		Name: strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)),
		Path: p.Path,
	}

	if scriptSetupNode != nil {
		scriptContent := htmlquery.InnerText(scriptSetupNode)
		err := p.parseScriptBlock(path, scriptContent, parserResult)
		if err != nil {
			return err
		}

		for _, attr := range scriptSetupNode.Attr {
			if attr.Key == "_name" {
				component.Name = attr.Val
			}
		}
	}

	if scriptNode != nil {
		scriptContent := htmlquery.InnerText(scriptNode)
		err := p.parseScriptBlock(path, scriptContent, parserResult)
		if err != nil {
			return err
		}

		for _, attr := range scriptNode.Attr {
			if attr.Key == "_name" {
				component.Name = attr.Val
			}
		}
	}

	if scriptNode != nil {
		scriptContent := htmlquery.InnerText(scriptNode)
		componentResult, err := parseVueComponentWithTSGo(p.PathAlias, path, scriptContent)
		if err != nil {
			return xfmt.Errorf("failed to parse vue component with tsgo: %w", err)
		}
		if componentResult.extendsProperty != nil {
			parserResult.VueExtendsProperty = componentResult.extendsProperty
		}
		if len(componentResult.componentPropertys) > 0 {
			parserResult.VueComponentsPropertys = componentResult.componentPropertys
		}
		if componentResult.rawExtends != "" {
			component.RawExtends = componentResult.rawExtends
			component.Extends = componentResult.rawExtends
		}
	}

	if _, ok := parserResult.Exports["default"]; !ok {
		parserResult.Exports["default"] = &parser.Export{
			ReferenceIdent: "default",
			ModuleSpecPath: p.Path,
		}
	}

	parserResult.VueComponent = component

	return nil
}

func (p *vueFileParser) getScriptNode(scriptNodes []*html.Node) (scriptNode *html.Node, scriptSetupNode *html.Node) {
	for _, node := range scriptNodes {
		hasSetup := false
		for _, attr := range node.Attr {
			if attr.Key == "setup" {
				hasSetup = true
				break
			}
		}
		if hasSetup {
			scriptSetupNode = node
		} else {
			scriptNode = node
		}
	}
	return
}

func (p *vueFileParser) parse() (*parser.ParserResult, error) {
	scriptNodes, templateNode, styleNodes, err := vuesfchtmlparser.ParseVueSfcToHtmlNode(strings.NewReader(p.Content))
	if err != nil {
		return nil, xfmt.Errorf("failed to parse vue sfc to html node: %w", err)
	}

	scriptNode, scriptSetupNode := p.getScriptNode(scriptNodes)

	parserResult := &parser.ParserResult{
		Path:               p.Path,
		RawContent:         p.Content,
		RawScriptNode:      scriptNode,
		RawScriptSetupNode: scriptSetupNode,
		RawTemplateNode:    templateNode,
		RawStyleNodes:      styleNodes,
	}

	err = p.parseScriptNodes(parserResult)
	if err != nil {
		return nil, err
	}

	return parserResult, nil
}
