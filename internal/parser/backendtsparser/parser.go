// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package backendtsparser

import (
	"github.com/choysum-dev/choysum/internal/parser"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
)

type backendtsParser struct {
	runtimeScope scope.Scope
	module       *meta.IrModule
}

func (p *backendtsParser) Parse(pathAlias map[string]string, path string, content string) (*parser.ParserResult, error) {
	// use new parser to avoid concurrent access
	modelParser := &tsFileParser{
		runtimeScope: p.runtimeScope,
		TsParser: &parser.TsParser{
			Path:      path,
			Content:   content,
			Context:   p.runtimeScope.Context(),
			PathAlias: pathAlias,
		},
	}
	return modelParser.parse()
}

func NewTsParser(runtimeScope scope.Scope, module *meta.IrModule) parser.Parser {
	return &backendtsParser{
		runtimeScope: runtimeScope,
		module:       module,
	}
}
