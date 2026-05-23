// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package service

import (
	"github.com/choysum-dev/choysum/pkg/scope"
)

type runtimeOptions struct {
	distPath  string
	dbDialect string
}

func newRuntimeOptions(paths scope.PathsRuntimeOptions, hasPaths bool, dbOpts scope.DatabaseRuntimeOptions, hasDB bool) runtimeOptions {
	opts := runtimeOptions{}
	if hasPaths {
		opts.distPath = paths.DistPath
	}
	if hasDB {
		opts.dbDialect = dbOpts.Dialect
	}
	return opts
}

func runtimeOptionsFromScope(runtimeScope scope.Scope) runtimeOptions {
	if runtimeScope == nil {
		return newRuntimeOptions(scope.PathsRuntimeOptions{}, false, scope.DatabaseRuntimeOptions{}, false)
	}
	paths, hasPaths := scope.PathsRuntimeOptionsFromScope(runtimeScope)
	dbOpts, hasDB := scope.DatabaseRuntimeOptionsFromScope(runtimeScope)
	return newRuntimeOptions(paths, hasPaths, dbOpts, hasDB)
}
