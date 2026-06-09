// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package webplugin

import (
	"strings"

	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/scope"
	xfmt "golang.org/x/exp/errors/fmt"
)

type runtimeOptions struct {
	modulesPath       string
	distPath          string
	webBaseURL        string
	serverEnvironment string
	serverEnabledTLS  bool
}

func newRuntimeOptions(pathOpts scope.PathsRuntimeOptions, hasPathOpts bool, serverOpts scope.ServerRuntimeOptions, hasServerOpts bool) runtimeOptions {
	serverDefaults := config.NewDefaultServerConfig()

	opts := runtimeOptions{
		webBaseURL:        serverDefaults.WebBaseURL,
		serverEnvironment: serverDefaults.Environment,
		serverEnabledTLS:  serverDefaults.EnabledTLS,
	}

	if hasPathOpts {
		opts.modulesPath = pathOpts.ModulesPath
		opts.distPath = pathOpts.DistPath
	}
	if hasServerOpts {
		opts.webBaseURL = serverOpts.WebBaseURL
		opts.serverEnvironment = serverOpts.Environment
		opts.serverEnabledTLS = serverOpts.EnabledTLS
	}

	return opts
}

func runtimeOptionsFromScope(runtimeScope scope.Scope) runtimeOptions {
	if runtimeScope == nil {
		return newRuntimeOptions(scope.PathsRuntimeOptions{}, false, scope.ServerRuntimeOptions{}, false)
	}
	pathOpts, hasPathOpts := scope.PathsRuntimeOptionsFromScope(runtimeScope)
	serverOpts, hasServerOpts := scope.ServerRuntimeOptionsFromScope(runtimeScope)
	return newRuntimeOptions(pathOpts, hasPathOpts, serverOpts, hasServerOpts)
}

func hasRuntimeOptions(opts runtimeOptions) bool {
	return strings.TrimSpace(opts.webBaseURL) != ""
}

func (p *WebPlugin) resolvedRuntimeOptions() runtimeOptions {
	if p != nil && hasRuntimeOptions(p.runtimeOptions) {
		return p.runtimeOptions
	}
	if p != nil && p.Env != nil {
		return runtimeOptionsFromScope(p.Env)
	}
	if p != nil {
		return p.runtimeOptions
	}
	return newRuntimeOptions(scope.PathsRuntimeOptions{}, false, scope.ServerRuntimeOptions{}, false)
}

func (o runtimeOptions) Validate() error {
	if strings.TrimSpace(o.modulesPath) == "" {
		return xfmt.Errorf("web plugin runtime options: modulesPath is required")
	}
	if strings.TrimSpace(o.distPath) == "" {
		return xfmt.Errorf("web plugin runtime options: distPath is required")
	}
	if strings.TrimSpace(o.webBaseURL) == "" {
		return xfmt.Errorf("web plugin runtime options: webBaseURL is required")
	}
	if strings.TrimSpace(o.serverEnvironment) == "" {
		return xfmt.Errorf("web plugin runtime options: serverEnvironment is required")
	}
	return nil
}
