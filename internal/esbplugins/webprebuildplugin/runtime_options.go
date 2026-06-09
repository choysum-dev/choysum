// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package webprebuildplugin

import "github.com/choysum-dev/choysum/pkg/scope"

type runtimeOptions struct {
	modulesPath string
}

func newRuntimeOptions(paths scope.PathsRuntimeOptions, ok bool) runtimeOptions {
	if !ok {
		return runtimeOptions{}
	}
	return runtimeOptions{modulesPath: paths.ModulesPath}
}

func runtimeOptionsFromScope(runtimeScope scope.Scope) runtimeOptions {
	if runtimeScope == nil {
		return newRuntimeOptions(scope.PathsRuntimeOptions{}, false)
	}
	paths, ok := scope.PathsRuntimeOptionsFromScope(runtimeScope)
	return newRuntimeOptions(paths, ok)
}
