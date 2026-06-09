// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package registry

import (
	"github.com/choysum-dev/choysum/pkg/scope"
)

type runtimeOptions struct {
	modulesPath    string
	npmRegistryURL string
}

func newRuntimeOptions(pathOpts scope.PathsRuntimeOptions, hasPathOpts bool) runtimeOptions {
	if !hasPathOpts {
		return runtimeOptions{}
	}
	return runtimeOptions{modulesPath: pathOpts.ModulesPath, npmRegistryURL: pathOpts.NpmRegistryURL}
}

func runtimeOptionsFromScope(runtimeScope scope.Scope) runtimeOptions {
	if runtimeScope == nil {
		return newRuntimeOptions(scope.PathsRuntimeOptions{}, false)
	}
	pathOpts, hasPathOpts := scope.PathsRuntimeOptionsFromScope(runtimeScope)
	return newRuntimeOptions(pathOpts, hasPathOpts)
}
