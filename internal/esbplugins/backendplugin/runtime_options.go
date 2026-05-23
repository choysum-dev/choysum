// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package backendplugin

import (
	"strings"

	"github.com/choysum-dev/choysum/pkg/scope"
	xfmt "golang.org/x/exp/errors/fmt"
)

type runtimeOptions struct {
	addonsPath string
}

func newRuntimeOptions(paths scope.PathsRuntimeOptions, ok bool) runtimeOptions {
	if !ok {
		return runtimeOptions{}
	}
	return runtimeOptions{addonsPath: paths.AddonsPath}
}

func runtimeOptionsFromScope(runtimeScope scope.Scope) runtimeOptions {
	if runtimeScope == nil {
		return newRuntimeOptions(scope.PathsRuntimeOptions{}, false)
	}
	paths, ok := scope.PathsRuntimeOptionsFromScope(runtimeScope)
	return newRuntimeOptions(paths, ok)
}

func hasRuntimeOptions(opts runtimeOptions) bool {
	return strings.TrimSpace(opts.addonsPath) != ""
}

func (p *BackendPlugin) resolvedRuntimeOptions() runtimeOptions {
	if p != nil && hasRuntimeOptions(p.runtimeOptions) {
		return p.runtimeOptions
	}
	if p != nil && p.Env != nil {
		return runtimeOptionsFromScope(p.Env)
	}
	if p != nil {
		return p.runtimeOptions
	}
	return newRuntimeOptions(scope.PathsRuntimeOptions{}, false)
}

func (o runtimeOptions) Validate() error {
	if strings.TrimSpace(o.addonsPath) == "" {
		return xfmt.Errorf("backend plugin runtime options: addonsPath is required")
	}
	return nil
}
