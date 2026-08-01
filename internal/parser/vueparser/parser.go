// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package vueparser

import (
	"fmt"
	"path/filepath"

	"github.com/choysum-dev/choysum/internal/parser"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
)

type vueParser struct {
	runtimeScope scope.Scope
	module       *meta.Module
}

func (p *vueParser) Parse(pathAlias map[string]string, path string, content string) (*parser.ParserResult, error) {
	// entryPointsMap := make(map[string]string)
	// json.Unmarshal(p.module.EntryPoints, &entryPointsMap)
	// indextsfile := filepath.Join(p.module.Path, entryPointsMap["web"])

	// if path == indextsfile {
	// 	parser := &indexParser{
	// 		TsParser: &parser.TsParser{
	// 			Path:      path,
	// 			Content:   content,
	// 			Context:   p.runtimeScope.Context(),
	// 			PathAlias: pathAlias,
	// 		},
	// 		runtimeScope: p.runtimeScope,
	// 	}
	// 	return parser.parse()
	// }

	// parser := &vueFileParser{
	// 	TsParser: &parser.TsParser{
	// 		Path:      path,
	// 		Content:   content,
	// 		Context:   p.runtimeScope.Context(),
	// 		PathAlias: pathAlias,
	// 	},
	// 	runtimeScope: p.runtimeScope,
	// }
	// return parser.parse()
	ext := filepath.Ext(path)

	baseParser := &parser.TsParser{
		Path:      path,
		Content:   content,
		Context:   p.runtimeScope.Context(),
		PathAlias: pathAlias,
	}

	switch ext {
	case ".ts":
		parser := &tsParser{
			TsParser:     baseParser,
			runtimeScope: p.runtimeScope,
		}
		return parser.parse()
	case ".vue":
		parser := &vueFileParser{
			TsParser:     baseParser,
			runtimeScope: p.runtimeScope,
		}
		return parser.parse()
	default:
		return nil, fmt.Errorf("unsupported file type: %s", ext)
	}
}

func NewVueParser(runtimeScope scope.Scope, module *meta.Module) parser.Parser {
	return &vueParser{
		runtimeScope: runtimeScope,
		module:       module,
	}
}
