// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package runner

import (
	"github.com/choysum-dev/choysum/pkg/scope"
)

type runtimeOptions struct {
	hasConfig   bool
	modulesPath string
	npmPath     string
	tmpPath     string
}

func newRuntimeOptions(pathOpts scope.PathsRuntimeOptions, hasPathOpts bool) runtimeOptions {
	opts := runtimeOptions{}
	if !hasPathOpts {
		return opts
	}
	opts.hasConfig = true
	opts.modulesPath = pathOpts.ModulesPath
	opts.npmPath = pathOpts.NpmPath
	opts.tmpPath = pathOpts.TmpPath
	return opts
}

func runtimeOptionsFromScope(runtimeScope scope.Scope) runtimeOptions {
	if runtimeScope == nil {
		return newRuntimeOptions(scope.PathsRuntimeOptions{}, false)
	}
	pathOpts, hasPathOpts := scope.PathsRuntimeOptionsFromScope(runtimeScope)
	return newRuntimeOptions(pathOpts, hasPathOpts)
}
