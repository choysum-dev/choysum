// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package hooks

import (
	"strings"

	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/scope"
)

type runtimeOptions struct {
	addonsPath        string
	distPath          string
	compileBundleMode string
	authInternalKey   string
}

func newRuntimeOptions(pathOpts scope.PathsRuntimeOptions, hasPathOpts bool, compileOpts scope.CompileRuntimeOptions, hasCompileOpts bool, authOpts scope.AuthRuntimeOptions, hasAuthOpts bool) runtimeOptions {
	compileDefaults := config.NewDefaultCompileConfig()

	opts := runtimeOptions{
		compileBundleMode: compileDefaults.BundleMode,
	}
	if hasPathOpts {
		opts.addonsPath = pathOpts.AddonsPath
		opts.distPath = pathOpts.DistPath
	}
	if hasCompileOpts && strings.TrimSpace(compileOpts.BundleMode) != "" {
		opts.compileBundleMode = compileOpts.BundleMode
	}
	if hasAuthOpts {
		opts.authInternalKey = authOpts.InternalKey
	}
	return opts
}

func runtimeOptionsFromScope(runtimeScope scope.Scope) runtimeOptions {
	if runtimeScope == nil {
		return newRuntimeOptions(scope.PathsRuntimeOptions{}, false, scope.CompileRuntimeOptions{}, false, scope.AuthRuntimeOptions{}, false)
	}
	pathOpts, hasPathOpts := scope.PathsRuntimeOptionsFromScope(runtimeScope)
	compileOpts, hasCompileOpts := scope.CompileRuntimeOptionsFromScope(runtimeScope)
	authOpts, hasAuthOpts := scope.AuthRuntimeOptionsFromScope(runtimeScope)
	return newRuntimeOptions(pathOpts, hasPathOpts, compileOpts, hasCompileOpts, authOpts, hasAuthOpts)
}
