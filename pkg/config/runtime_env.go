// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package config

import envconfig "github.com/choysum-dev/choysum/internal/module/artifact/config/env"

type RuntimeEnvironmentConfig = envconfig.RuntimeEnvironmentConfig

func NewDefaultFrontendEnv(c *Config) map[string]any {
	if c == nil {
		return envconfig.NewDefaultFrontendEnv("", false, false)
	}

	webBaseURL := ""
	production := false
	enableClientHashing := false

	if c.Server != nil {
		webBaseURL = c.Server.WebBaseURL
	}
	if c.Compile != nil {
		production = c.Compile.Production
	}
	if c.Auth != nil {
		enableClientHashing = c.Auth.EnableClientHashing
	}

	return envconfig.NewDefaultFrontendEnv(webBaseURL, production, enableClientHashing)
}

func NewDefaultBackendEnv() map[string]any {
	return envconfig.NewDefaultBackendEnv()
}
