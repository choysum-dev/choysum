// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package httpauth

import (
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/scope"
)

type runtimeOptions struct {
	enabled  bool
	httpAuth *config.HttpAuthConfig
}

func runtimeOptionsFromScope(runtimeScope scope.Scope) runtimeOptions {
	if runtimeScope == nil {
		return runtimeOptions{}
	}
	authOpts, ok := scope.AuthRuntimeOptionsFromScope(runtimeScope)
	if !ok || !authOpts.Enabled || authOpts.HttpAuth == nil || !authOpts.HttpAuth.Enabled {
		return runtimeOptions{}
	}
	return runtimeOptions{
		enabled:  true,
		httpAuth: authOpts.HttpAuth,
	}
}
