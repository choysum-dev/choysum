// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cmd

import (
	"strings"

	testsemantics "github.com/choysum-dev/choysum/internal/testing/semantics"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/scope"
	xfmt "golang.org/x/exp/errors/fmt"
)

type cliRuntimeOptions struct {
	defaultChoysumPath    string
	modulesPath           string
	tmpPath               string
	moduleCatalogIndexURL string
}

func newCliRuntimeOptions(pathOpts scope.PathsRuntimeOptions, hasPathOpts bool) cliRuntimeOptions {
	if !hasPathOpts {
		return cliRuntimeOptions{}
	}
	return cliRuntimeOptions{
		defaultChoysumPath:    pathOpts.DefaultChoysumPath,
		modulesPath:           pathOpts.ModulesPath,
		tmpPath:               pathOpts.TmpPath,
		moduleCatalogIndexURL: strings.TrimSpace(pathOpts.ModuleCatalogIndexURL),
	}
}

func newCliRuntimeOptionsFromScopeInputOptions(options *scopeInputConfigOptions) cliRuntimeOptions {
	if options == nil {
		return cliRuntimeOptions{}
	}
	return cliRuntimeOptions{
		defaultChoysumPath:    options.DefaultChoysumPath,
		modulesPath:           options.ModulesPath,
		tmpPath:               options.TmpPath,
		moduleCatalogIndexURL: strings.TrimSpace(options.ModuleCatalogIndexURL),
	}
}

func cliRuntimeOptionsFromScope(runtimeScope scope.Scope) cliRuntimeOptions {
	if runtimeScope == nil {
		return cliRuntimeOptions{}
	}
	pathOpts, hasPathOpts := scope.PathsRuntimeOptionsFromScope(runtimeScope)
	return newCliRuntimeOptions(pathOpts, hasPathOpts)
}

func requireCliRuntimeOptions(optionsGetter func() cliRuntimeOptions) (cliRuntimeOptions, error) {
	if optionsGetter == nil {
		return cliRuntimeOptions{}, xfmt.Errorf("cli runtime options getter is not initialized")
	}
	runtimeOptions := optionsGetter()
	if err := runtimeOptions.Validate(); err != nil {
		return cliRuntimeOptions{}, err
	}
	return runtimeOptions, nil
}

func requireCliRuntimeOptionsForCommand(commandName string, optionsGetter func() cliRuntimeOptions) (cliRuntimeOptions, error) {
	runtimeOptions, err := requireCliRuntimeOptions(optionsGetter)
	if err != nil {
		return cliRuntimeOptions{}, xfmt.Errorf("%s: %w", testsemantics.InvalidRuntimeOptionsMessage(commandName), err)
	}
	return runtimeOptions, nil
}

func (o cliRuntimeOptions) Validate() error {
	if strings.TrimSpace(o.defaultChoysumPath) == "" {
		return xfmt.Errorf("cli runtime options: defaultChoysumPath is required")
	}
	if strings.TrimSpace(o.modulesPath) == "" {
		return xfmt.Errorf("cli runtime options: modulesPath is required")
	}
	if strings.TrimSpace(o.tmpPath) == "" {
		return xfmt.Errorf("cli runtime options: tmpPath is required")
	}
	moduleCatalogIndexURL := strings.TrimSpace(o.moduleCatalogIndexURL)
	if moduleCatalogIndexURL == "" {
		moduleCatalogIndexURL = config.DefaultModuleCatalogIndexURL
	}
	if err := config.ValidateModuleCatalogIndexURL(moduleCatalogIndexURL); err != nil {
		return err
	}
	return nil
}
