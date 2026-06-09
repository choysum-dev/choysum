// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package backend

import (
	"github.com/choysum-dev/choysum/pkg/scope"
)

type runtimeOptions struct {
	hasConfig             bool
	modulesPath           string
	distPath              string
	tmpPath               string
	compileBundleMode     string
	serverJsEngineFactory string
	authEnabled           bool
}

func newRuntimeOptions(pathOpts scope.PathsRuntimeOptions, hasPathOpts bool, compileOpts scope.CompileRuntimeOptions, hasCompileOpts bool, serverOpts scope.ServerRuntimeOptions, hasServerOpts bool, authOpts scope.AuthRuntimeOptions, hasAuthOpts bool) runtimeOptions {
	opts := runtimeOptions{}
	if !hasPathOpts {
		return opts
	}
	opts.hasConfig = true
	opts.modulesPath = pathOpts.ModulesPath
	opts.distPath = pathOpts.DistPath
	opts.tmpPath = pathOpts.TmpPath
	if hasCompileOpts {
		opts.compileBundleMode = compileOpts.BundleMode
	}
	if hasServerOpts {
		opts.serverJsEngineFactory = serverOpts.JsEngineFactory
	}
	opts.authEnabled = hasAuthOpts && authOpts.Enabled
	return opts
}

func runtimeOptionsFromScope(runtimeScope scope.Scope) runtimeOptions {
	if runtimeScope == nil {
		return newRuntimeOptions(scope.PathsRuntimeOptions{}, false, scope.CompileRuntimeOptions{}, false, scope.ServerRuntimeOptions{}, false, scope.AuthRuntimeOptions{}, false)
	}
	pathOpts, hasPathOpts := scope.PathsRuntimeOptionsFromScope(runtimeScope)
	compileOpts, hasCompileOpts := scope.CompileRuntimeOptionsFromScope(runtimeScope)
	serverOpts, hasServerOpts := scope.ServerRuntimeOptionsFromScope(runtimeScope)
	authOpts, hasAuthOpts := scope.AuthRuntimeOptionsFromScope(runtimeScope)
	return newRuntimeOptions(pathOpts, hasPathOpts, compileOpts, hasCompileOpts, serverOpts, hasServerOpts, authOpts, hasAuthOpts)
}
