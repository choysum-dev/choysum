// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cspmiddleware

import (
	"strings"

	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/scope"
)

type runtimeOptions struct {
	excludedPaths []string
	reportOnly    bool
	reportURI     string
	cspConfig     *config.CSPConfig
	hstsConfig    *config.HSTSConfig
	environment   string
	useHTTPS      bool
}

func runtimeOptionsFromScope(runtimeScope scope.Scope) runtimeOptions {
	opts := runtimeOptions{
		cspConfig:  config.NewDefaultCSPConfig(),
		hstsConfig: config.NewDefaultHSTSConfig(),
	}
	if runtimeScope == nil {
		return opts
	}

	serverOpts, ok := scope.ServerRuntimeOptionsFromScope(runtimeScope)
	if !ok {
		return opts
	}

	cspConfig := serverOpts.CSP
	hstsConfig := serverOpts.HSTS
	if serverOpts.SecurityMissing {
		runtimeScope.Logger().Warn("security config missing; using defaults")
	}
	if cspConfig == nil {
		runtimeScope.Logger().Warn("csp config missing; using defaults")
		cspConfig = config.NewDefaultCSPConfig()
	}
	if hstsConfig == nil {
		runtimeScope.Logger().Warn("hsts config missing; using defaults")
		hstsConfig = config.NewDefaultHSTSConfig()
	}

	opts.excludedPaths = cspConfig.ExcludedPaths
	opts.reportOnly = cspConfig.ReportOnly
	opts.reportURI = cspConfig.ReportURI
	opts.cspConfig = cspConfig
	opts.hstsConfig = hstsConfig
	opts.environment = strings.TrimSpace(serverOpts.Environment)
	opts.useHTTPS = serverOpts.EnabledTLS
	return opts
}
