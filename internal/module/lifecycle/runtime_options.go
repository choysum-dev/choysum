// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"strings"

	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/scope"
	xfmt "golang.org/x/exp/errors/fmt"
)

type runtimeOptions struct {
	modulesPath        string
	distPath           string
	tmpPath            string
	defaultChoysumPath string
	compileBundleMode  string
}

func newRuntimeOptions(pathOpts scope.PathsRuntimeOptions, hasPathOpts bool, compileOpts scope.CompileRuntimeOptions, hasCompileOpts bool) runtimeOptions {
	compileDefaults := config.NewDefaultCompileConfig()

	opts := runtimeOptions{
		compileBundleMode: compileDefaults.BundleMode,
	}
	if hasPathOpts {
		opts.modulesPath = pathOpts.ModulesPath
		opts.distPath = pathOpts.DistPath
		opts.tmpPath = pathOpts.TmpPath
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

func hasRuntimeOptions(opts runtimeOptions) bool {
	return strings.TrimSpace(opts.compileBundleMode) != ""
}

func (m *ModuleManager) resolvedRuntimeOptions() runtimeOptions {
	if m != nil && hasRuntimeOptions(m.runtimeOptions) {
		return m.runtimeOptions
	}
	if m != nil && m.runtimeScope != nil {
		return runtimeOptionsFromScope(m.runtimeScope)
	}
	if m != nil {
		return m.runtimeOptions
	}
	return newRuntimeOptions(scope.PathsRuntimeOptions{}, false, scope.CompileRuntimeOptions{}, false)
}

func (o runtimeOptions) Validate() error {
	if strings.TrimSpace(o.modulesPath) == "" {
		return xfmt.Errorf("module runtime options: modulesPath is required")
	}
	if strings.TrimSpace(o.distPath) == "" {
		return xfmt.Errorf("module runtime options: distPath is required")
	}
	if strings.TrimSpace(o.tmpPath) == "" {
		return xfmt.Errorf("module runtime options: tmpPath is required")
	}
	if strings.TrimSpace(o.defaultChoysumPath) == "" {
		return xfmt.Errorf("module runtime options: defaultChoysumPath is required")
	}
	if strings.TrimSpace(o.compileBundleMode) == "" {
		return xfmt.Errorf("module runtime options: compileBundleMode is required")
	}
	return nil
}
