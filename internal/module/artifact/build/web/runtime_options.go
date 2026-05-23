// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package webmodulebuilder

import (
	"strings"

	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/scope"
	xfmt "golang.org/x/exp/errors/fmt"
)

type runtimeOptions struct {
	addonsPath         string
	distPath           string
	defaultChoysumPath string
	webBaseURL         string
	frontendEnv        map[string]any
	compileSourceMap   bool
	compileMinify      bool
	compileTreeShaking bool
}

func newRuntimeOptions(pathOpts scope.PathsRuntimeOptions, hasPathOpts bool, serverOpts scope.ServerRuntimeOptions, hasServerOpts bool, envOpts scope.RuntimeEnvironmentOptions, hasEnvOpts bool, compileOpts scope.CompileRuntimeOptions, hasCompileOpts bool) runtimeOptions {
	compileDefaults := config.NewDefaultCompileConfig()
	serverDefaults := config.NewDefaultServerConfig()

	opts := runtimeOptions{
		webBaseURL:         serverDefaults.WebBaseURL,
		frontendEnv:        map[string]any{},
		compileSourceMap:   compileDefaults.SourceMap,
		compileMinify:      compileDefaults.Minify,
		compileTreeShaking: compileDefaults.TreeShaking,
	}

	if hasPathOpts {
		opts.addonsPath = pathOpts.AddonsPath
		opts.distPath = pathOpts.DistPath
		opts.defaultChoysumPath = pathOpts.DefaultChoysumPath
	}
	if hasServerOpts {
		opts.webBaseURL = serverOpts.WebBaseURL
	}
	if hasEnvOpts && envOpts.FrontendEnv != nil {
		opts.frontendEnv = envOpts.FrontendEnv
	}
	if hasCompileOpts {
		opts.compileSourceMap = compileOpts.SourceMap
		opts.compileMinify = compileOpts.Minify
		opts.compileTreeShaking = compileOpts.TreeShaking
	}

	return opts
}

func runtimeOptionsFromScope(runtimeScope scope.Scope) runtimeOptions {
	if runtimeScope == nil {
		return newRuntimeOptions(scope.PathsRuntimeOptions{}, false, scope.ServerRuntimeOptions{}, false, scope.RuntimeEnvironmentOptions{}, false, scope.CompileRuntimeOptions{}, false)
	}
	pathOpts, hasPathOpts := scope.PathsRuntimeOptionsFromScope(runtimeScope)
	serverOpts, hasServerOpts := scope.ServerRuntimeOptionsFromScope(runtimeScope)
	envOpts, hasEnvOpts := scope.RuntimeEnvironmentOptionsFromScope(runtimeScope)
	compileOpts, hasCompileOpts := scope.CompileRuntimeOptionsFromScope(runtimeScope)
	return newRuntimeOptions(pathOpts, hasPathOpts, serverOpts, hasServerOpts, envOpts, hasEnvOpts, compileOpts, hasCompileOpts)
}

func hasRuntimeOptions(opts runtimeOptions) bool {
	return strings.TrimSpace(opts.distPath) != ""
}

func (b *WebModuleBuilder) resolvedRuntimeOptions() runtimeOptions {
	if b != nil && hasRuntimeOptions(b.runtimeOptions) {
		return b.runtimeOptions
	}
	if b != nil && b.runtimeScope != nil {
		return runtimeOptionsFromScope(b.runtimeScope)
	}
	if b != nil {
		return b.runtimeOptions
	}
	return newRuntimeOptions(scope.PathsRuntimeOptions{}, false, scope.ServerRuntimeOptions{}, false, scope.RuntimeEnvironmentOptions{}, false, scope.CompileRuntimeOptions{}, false)
}

func (o runtimeOptions) Validate() error {
	if strings.TrimSpace(o.addonsPath) == "" {
		return xfmt.Errorf("web builder runtime options: addonsPath is required")
	}
	if strings.TrimSpace(o.distPath) == "" {
		return xfmt.Errorf("web builder runtime options: distPath is required")
	}
	if strings.TrimSpace(o.defaultChoysumPath) == "" {
		return xfmt.Errorf("web builder runtime options: defaultChoysumPath is required")
	}
	if strings.TrimSpace(o.webBaseURL) == "" {
		return xfmt.Errorf("web builder runtime options: webBaseURL is required")
	}
	return nil
}
