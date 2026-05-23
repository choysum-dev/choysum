// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package jobtoken

import "github.com/choysum-dev/choysum/pkg/scope"

type runtimeOptions struct {
	authConfigured         bool
	serverEnvironment      string
	authInternalKey        string
	authJobTokenAllowedSAN []string
}

func newRuntimeOptions(serverOpts scope.ServerRuntimeOptions, hasServerOpts bool, authOpts scope.AuthRuntimeOptions, hasAuthOpts bool) runtimeOptions {
	opts := runtimeOptions{}
	if hasServerOpts {
		opts.serverEnvironment = serverOpts.Environment
	}
	if hasAuthOpts {
		opts.authConfigured = true
		opts.authInternalKey = authOpts.InternalKey
		if len(authOpts.JobTokenAllowedSANs) > 0 {
			opts.authJobTokenAllowedSAN = append([]string{}, authOpts.JobTokenAllowedSANs...)
		}
	}
	return opts
}

func runtimeOptionsFromScope(runtimeScope scope.Scope) runtimeOptions {
	if runtimeScope == nil {
		return newRuntimeOptions(scope.ServerRuntimeOptions{}, false, scope.AuthRuntimeOptions{}, false)
	}
	serverOpts, hasServerOpts := scope.ServerRuntimeOptionsFromScope(runtimeScope)
	authOpts, hasAuthOpts := scope.AuthRuntimeOptionsFromScope(runtimeScope)
	return newRuntimeOptions(serverOpts, hasServerOpts, authOpts, hasAuthOpts)
}
