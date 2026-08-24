// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package vueparser

import (
	tsast "github.com/buke/typescript-go-internal/pkg/ast"
	tscore "github.com/buke/typescript-go-internal/pkg/core"
	"github.com/choysum-dev/choysum/internal/parser"
	"github.com/choysum-dev/choysum/internal/parser/tsgoctx"
)

// tsParseCtx is a package-local alias of the shared TS parse context.
type tsParseCtx = tsgoctx.ParseCtx

func parseTSGoCtx(pathAlias map[string]string, path string, content string) (*tsParseCtx, error) {
	return tsgoctx.Parse(pathAlias, path, content)
}

func parseTSGoCtxWithKind(pathAlias map[string]string, path string, content string, forcedScriptKind tscore.ScriptKind, useForcedScriptKind bool) (*tsParseCtx, error) {
	return tsgoctx.ParseWithKind(pathAlias, path, content, forcedScriptKind, useForcedScriptKind)
}

func mergeImports(dst map[string]*parser.Import, src map[string]*parser.Import) {
	tsgoctx.MergeImports(dst, src)
}

func mergeExports(dst map[string]*parser.Export, src map[string]*parser.Export) {
	tsgoctx.MergeExports(dst, src)
}

func exportDeclarationName(stmt *tsast.Node) string {
	return tsgoctx.ExportDeclarationName(stmt)
}
