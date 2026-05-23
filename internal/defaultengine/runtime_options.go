// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package defaultengine

import (
	"github.com/choysum-dev/choysum/pkg/scope"
)

type runtimeOptions struct {
	dbDialect string
}

func newRuntimeOptions(dbOpts scope.DatabaseRuntimeOptions, ok bool) runtimeOptions {
	opts := runtimeOptions{}
	if ok {
		opts.dbDialect = dbOpts.Dialect
	}
	return opts
}

func runtimeOptionsFromScope(runtimeScope scope.Scope) runtimeOptions {
	if runtimeScope == nil {
		return newRuntimeOptions(scope.DatabaseRuntimeOptions{}, false)
	}
	dbOpts, ok := scope.DatabaseRuntimeOptionsFromScope(runtimeScope)
	return newRuntimeOptions(dbOpts, ok)
}
