// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package jwtauth

import (
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/scope"
)

type runtimeOptions struct {
	authJWT *config.JWTConfig
}

func newRuntimeOptions(authOpts scope.AuthRuntimeOptions, hasAuthOpts bool) runtimeOptions {
	opts := runtimeOptions{}
	if !hasAuthOpts {
		return opts
	}
	opts.authJWT = authOpts.JWT
	return opts
}

func runtimeOptionsFromScope(runtimeScope scope.Scope) runtimeOptions {
	if runtimeScope == nil {
		return newRuntimeOptions(scope.AuthRuntimeOptions{}, false)
	}
	authOpts, hasAuthOpts := scope.AuthRuntimeOptionsFromScope(runtimeScope)
	return newRuntimeOptions(authOpts, hasAuthOpts)
}
