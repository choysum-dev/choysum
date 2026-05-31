// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package config

import envconfig "github.com/choysum-dev/choysum/internal/module/artifact/config/env"

type RuntimeEnvironmentConfig = envconfig.RuntimeEnvironmentConfig

func NewDefaultFrontendEnv(c *Config) map[string]any {
	if c == nil {
		return envconfig.NewDefaultFrontendEnv("", false)
	}

	webBaseURL := ""
	production := false

	if c.Server != nil {
		webBaseURL = c.Server.WebBaseURL
	}
	if c.Compile != nil {
		production = c.Compile.Production
	}

	return envconfig.NewDefaultFrontendEnv(webBaseURL, production)
}

func NewDefaultBackendEnv() map[string]any {
	return envconfig.NewDefaultBackendEnv()
}
