// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package generator

import (
	"strings"

	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/scope"
)

type runtimeOptions struct {
	addonsPath         string
	distPath           string
	defaultChoysumPath string
	compileBundleMode  string
}

func newRuntimeOptions(pathOpts scope.PathsRuntimeOptions, hasPathOpts bool, compileOpts scope.CompileRuntimeOptions, hasCompileOpts bool) runtimeOptions {
	compileDefaults := config.NewDefaultCompileConfig()

	opts := runtimeOptions{compileBundleMode: compileDefaults.BundleMode}
	if hasPathOpts {
		opts.addonsPath = pathOpts.AddonsPath
		opts.distPath = pathOpts.DistPath
		opts.defaultChoysumPath = pathOpts.DefaultChoysumPath
	}
	if hasCompileOpts && strings.TrimSpace(compileOpts.BundleMode) != "" {
		opts.compileBundleMode = compileOpts.BundleMode
	}

	return opts
}

func runtimeOptionsFromScope(runtimeScope scope.Scope) runtimeOptions {
	if runtimeScope == nil {
		return newRuntimeOptions(scope.PathsRuntimeOptions{}, false, scope.CompileRuntimeOptions{}, false)
	}
	pathOpts, hasPathOpts := scope.PathsRuntimeOptionsFromScope(runtimeScope)
	compileOpts, hasCompileOpts := scope.CompileRuntimeOptionsFromScope(runtimeScope)
	return newRuntimeOptions(pathOpts, hasPathOpts, compileOpts, hasCompileOpts)
}

func (o runtimeOptions) isBundleMode() bool {
	return strings.EqualFold(strings.TrimSpace(o.compileBundleMode), "bundle")
}
