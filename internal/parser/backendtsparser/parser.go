// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package backendtsparser

import (
	"log/slog"

	"github.com/choysum-dev/choysum/internal/parser"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
)

type backendtsParser struct {
	runtimeScope scope.Scope
	module       *meta.Module
	semantic     *semanticTypeResolver
}

func (p *backendtsParser) Parse(pathAlias map[string]string, path string, content string) (*parser.ParserResult, error) {
	// use new parser to avoid concurrent access
	ownerModule := ""
	if p.module != nil {
		ownerModule = p.module.Name
	}
	modelParser := &tsFileParser{
		runtimeScope: p.runtimeScope,
		ownerModule:  ownerModule,
		semantic:     p.semantic,
		TsParser: &parser.TsParser{
			Path:      path,
			Content:   content,
			Context:   p.runtimeScope.Context(),
			PathAlias: pathAlias,
		},
	}
	return modelParser.parse()
}

func NewTsParser(runtimeScope scope.Scope, module *meta.Module) parser.Parser {
	var logger *slog.Logger
	if runtimeScope != nil {
		logger = runtimeScope.Logger()
	}
	return &backendtsParser{
		runtimeScope: runtimeScope,
		module:       module,
		semantic:     newSemanticTypeResolver(logger),
	}
}
