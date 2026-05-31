// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package envconfig

import "strings"

// RuntimeEnvironmentConfig owns root-level frontend/backend runtime environment maps.
type RuntimeEnvironmentConfig struct {
	Frontend map[string]any `mapstructure:"frontendEnv"`
	Backend  map[string]any `mapstructure:"backendEnv"`
}

func NewDefaultFrontendEnv(webBaseURL string, production bool) map[string]any {
	mode := "development"
	if production {
		mode = "production"
	}

	baseURL := strings.TrimSuffix(strings.TrimSpace(webBaseURL), "/") + "/"
	if strings.TrimSpace(baseURL) == "" {
		baseURL = "/"
	}

	return map[string]any{
		"BASE_URL":                    baseURL,
		"MODE":                        mode,
		"PROD":                        production,
		"DEV":                         !production,
		"SSR":                         false,
		"CHOYSUM_APP_NAME":            "Choysum",
		"CHOYSUM_APP_VERSION":         "v0.1.0",
		"CHOYSUM_MAINTENANCE_MODE":    false,
		"CHOYSUM_CSRF_ENABLED":        true,
		"CHOYSUM_ENABLE_REGISTRATION": true,
	}
}

func NewDefaultBackendEnv() map[string]any {
	return map[string]any{
		"CHOYSUM_SOFT_DELETE_ENABLED": true,
	}
}
