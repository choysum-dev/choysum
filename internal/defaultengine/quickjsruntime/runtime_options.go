// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package quickjsruntime

import (
	"strings"

	"github.com/choysum-dev/choysum/pkg/scope"
)

type runtimeOptions struct {
	addonsPath string
}

func newRuntimeOptions(pathOpts scope.PathsRuntimeOptions, hasPathOpts bool) runtimeOptions {
	opts := runtimeOptions{}
	if hasPathOpts {
		opts.addonsPath = pathOpts.AddonsPath
	}
	return opts
}

func runtimeOptionsFromScope(runtimeScope scope.Scope) runtimeOptions {
	if runtimeScope == nil {
		return newRuntimeOptions(scope.PathsRuntimeOptions{}, false)
	}
	pathOpts, hasPathOpts := scope.PathsRuntimeOptionsFromScope(runtimeScope)
	return newRuntimeOptions(pathOpts, hasPathOpts)
}

func (o runtimeOptions) hasAddonsPath() bool {
	return strings.TrimSpace(o.addonsPath) != ""
}
