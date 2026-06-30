// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package runtime

import (
	"strings"

	testsemantics "github.com/choysum-dev/choysum/internal/testing/semantics"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/scope"
	xfmt "golang.org/x/exp/errors/fmt"
)

type Options struct {
	DefaultChoysumPath    string
	ModulesPath           string
	TmpPath               string
	ModuleCatalogIndexURL string
}

func NewOptions(pathOpts scope.PathsRuntimeOptions, hasPathOpts bool) Options {
	if !hasPathOpts {
		return Options{}
	}
	return Options{
		DefaultChoysumPath:    pathOpts.DefaultChoysumPath,
		ModulesPath:           pathOpts.ModulesPath,
		TmpPath:               pathOpts.TmpPath,
		ModuleCatalogIndexURL: strings.TrimSpace(pathOpts.ModuleCatalogIndexURL),
	}
}

func OptionsFromScope(runtimeScope scope.Scope) Options {
	if runtimeScope == nil {
		return Options{}
	}
	pathOpts, hasPathOpts := scope.PathsRuntimeOptionsFromScope(runtimeScope)
	return NewOptions(pathOpts, hasPathOpts)
}

func RequireOptions(optionsGetter func() Options) (Options, error) {
	if optionsGetter == nil {
		return Options{}, xfmt.Errorf("cli runtime options getter is not initialized")
	}
	runtimeOptions := optionsGetter()
	if err := runtimeOptions.Validate(); err != nil {
		return Options{}, err
	}
	return runtimeOptions, nil
}

func RequireOptionsForCommand(commandName string, optionsGetter func() Options) (Options, error) {
	runtimeOptions, err := RequireOptions(optionsGetter)
	if err != nil {
		return Options{}, xfmt.Errorf("%s: %w", testsemantics.InvalidRuntimeOptionsMessage(commandName), err)
	}
	return runtimeOptions, nil
}

func ResolveModuleCatalogIndexURL(runtimeOptions Options) (string, error) {
	indexURL := strings.TrimSpace(runtimeOptions.ModuleCatalogIndexURL)
	if indexURL == "" {
		indexURL = config.DefaultModuleCatalogIndexURL
	}
	if err := config.ValidateModuleCatalogIndexURL(indexURL); err != nil {
		return "", err
	}
	return indexURL, nil
}

func (o Options) Validate() error {
	if strings.TrimSpace(o.DefaultChoysumPath) == "" {
		return xfmt.Errorf("cli runtime options: defaultChoysumPath is required")
	}
	if strings.TrimSpace(o.ModulesPath) == "" {
		return xfmt.Errorf("cli runtime options: modulesPath is required")
	}
	if strings.TrimSpace(o.TmpPath) == "" {
		return xfmt.Errorf("cli runtime options: tmpPath is required")
	}
	_, err := ResolveModuleCatalogIndexURL(o)
	return err
}
