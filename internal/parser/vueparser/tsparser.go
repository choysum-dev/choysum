// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package vueparser

import (
	"path/filepath"

	"github.com/choysum-dev/choysum/internal/parser"
	"github.com/choysum-dev/choysum/pkg/scope"
	xfmt "golang.org/x/exp/errors/fmt"
)

type tsParser struct {
	*parser.TsParser
	runtimeScope scope.Scope
}

func (p *tsParser) parse() (*parser.ParserResult, error) {
	modules_path := runtimeOptionsFromScope(p.runtimeScope).modulesPath
	// Keep historical skip behavior for known compatibility paths.
	// modules/core/service/orm/metadata/field.ts
	// modules/core/service/runtime/onchange/types.ts
	if p.Path == filepath.Join(modules_path, "core", "client", "store.ts") ||
		p.Path == filepath.Join(modules_path, "core", "service", "orm", "metadata", "field.ts") ||
		p.Path == filepath.Join(modules_path, "core", "service", "runtime", "onchange", "types.ts") {
		return &parser.ParserResult{
			Path:       p.Path,
			RawContent: p.Content,
		}, nil
	}

	ctx, err := parseTSGoCtx(p.PathAlias, p.Path, p.Content)
	if err != nil {
		return nil, xfmt.Errorf("tsParser failed to parse %s with tsgo: %w", p.Path, err)
	}

	uiDecls, uiIssues := collectUiResourceDecls(p.Path, p.Content)

	return &parser.ParserResult{
		Path:                 p.Path,
		RawContent:           p.Content,
		Imports:              ctx.Imports,
		Exports:              ctx.Exports,
		DynamicImports:       ctx.DynamicImports,
		UiResourceDecls:      uiDecls,
		UiResourceDeclIssues: uiIssues,
	}, nil

}
