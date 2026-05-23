// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cmd

import (
	"strings"

	"github.com/choysum-dev/choysum/pkg/scope"
	xfmt "golang.org/x/exp/errors/fmt"
)

type cliRuntimeOptions struct {
	defaultChoysumPath string
	addonsPath         string
	npmPath            string
	tmpPath            string
}

func newCliRuntimeOptions(pathOpts scope.PathsRuntimeOptions, hasPathOpts bool) cliRuntimeOptions {
	if !hasPathOpts {
		return cliRuntimeOptions{}
	}
	return cliRuntimeOptions{
		defaultChoysumPath: pathOpts.DefaultChoysumPath,
		addonsPath:         pathOpts.AddonsPath,
		npmPath:            pathOpts.NpmPath,
		tmpPath:            pathOpts.TmpPath,
	}
}

func newCliRuntimeOptionsFromScopeInputOptions(options *scopeInputConfigOptions) cliRuntimeOptions {
	if options == nil {
		return cliRuntimeOptions{}
	}
	return cliRuntimeOptions{
		defaultChoysumPath: options.DefaultChoysumPath,
		addonsPath:         options.AddonsPath,
		npmPath:            options.NpmPath,
		tmpPath:            options.TmpPath,
	}
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

func (o cliRuntimeOptions) Validate() error {
	if strings.TrimSpace(o.defaultChoysumPath) == "" {
		return xfmt.Errorf("cli runtime options: defaultChoysumPath is required")
	}
	if strings.TrimSpace(o.addonsPath) == "" {
		return xfmt.Errorf("cli runtime options: addonsPath is required")
	}
	if strings.TrimSpace(o.npmPath) == "" {
		return xfmt.Errorf("cli runtime options: npmPath is required")
	}
	if strings.TrimSpace(o.tmpPath) == "" {
		return xfmt.Errorf("cli runtime options: tmpPath is required")
	}
	return nil
}
