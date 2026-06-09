// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package discovery

import (
	"github.com/choysum-dev/choysum/pkg/scope"
)

type runtimeOptions struct {
	hasConfig   bool
	modulesPath string
}

func newRuntimeOptions(paths scope.PathsRuntimeOptions, ok bool) runtimeOptions {
	opts := runtimeOptions{}
	if !ok {
		return opts
	}
	opts.hasConfig = true
	opts.modulesPath = paths.ModulesPath
	return opts
}

func runtimeOptionsFromScope(runtimeScope scope.Scope) runtimeOptions {
	if runtimeScope == nil {
		return newRuntimeOptions(scope.PathsRuntimeOptions{}, false)
	}
	paths, ok := scope.PathsRuntimeOptionsFromScope(runtimeScope)
	return newRuntimeOptions(paths, ok)
}
