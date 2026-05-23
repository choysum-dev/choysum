// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package registry

import (
	"github.com/choysum-dev/choysum/pkg/scope"
)

type runtimeOptions struct {
	serverRegister string
}

func newRuntimeOptions(serverOpts scope.ServerRuntimeOptions, hasServerOpts bool) runtimeOptions {
	opts := runtimeOptions{}
	if hasServerOpts {
		opts.serverRegister = serverOpts.Register
	}
	return opts
}

func runtimeOptionsFromScope(runtimeScope scope.Scope) runtimeOptions {
	if runtimeScope == nil {
		return newRuntimeOptions(scope.ServerRuntimeOptions{}, false)
	}
	serverOpts, hasServerOpts := scope.ServerRuntimeOptionsFromScope(runtimeScope)
	return newRuntimeOptions(serverOpts, hasServerOpts)
}
