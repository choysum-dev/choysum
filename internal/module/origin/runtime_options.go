// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package origin

import (
	"strings"

	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/scope"
)

type runtimeOptions struct {
	modulesPath                          string
	configPath                           string
	defaultChoysumPath                   string
	moduleCatalogIndexURL                string
	moduleInstallRegistryFallbackEnabled bool
}

func newRuntimeOptions(pathOpts scope.PathsRuntimeOptions, hasPathOpts bool) runtimeOptions {
	if !hasPathOpts {
		return runtimeOptions{moduleInstallRegistryFallbackEnabled: false}
	}
	moduleCatalogIndexURL := strings.TrimSpace(pathOpts.ModuleCatalogIndexURL)
	if moduleCatalogIndexURL == "" {
		moduleCatalogIndexURL = config.DefaultModuleCatalogIndexURL
	}

	return runtimeOptions{
		modulesPath:                          pathOpts.ModulesPath,
		configPath:                           pathOpts.ConfigPath,
		defaultChoysumPath:                   pathOpts.DefaultChoysumPath,
		moduleCatalogIndexURL:                moduleCatalogIndexURL,
		moduleInstallRegistryFallbackEnabled: pathOpts.ModuleInstallRegistryFallbackEnabled,
	}
}

func runtimeOptionsFromScope(runtimeScope scope.Scope) runtimeOptions {
	if runtimeScope == nil {
		return newRuntimeOptions(scope.PathsRuntimeOptions{}, false)
	}
	pathOpts, hasPathOpts := scope.PathsRuntimeOptionsFromScope(runtimeScope)
	return newRuntimeOptions(pathOpts, hasPathOpts)
}
