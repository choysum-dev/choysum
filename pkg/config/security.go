// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package config

import serverconfig "github.com/choysum-dev/choysum/internal/server/serverconfig"

type SecurityConfig = serverconfig.SecurityConfig
type CSPConfig = serverconfig.CSPConfig
type CSPDirectives = serverconfig.CSPDirectives
type HSTSConfig = serverconfig.HSTSConfig
type CSRFConfig = serverconfig.CSRFConfig

func NewDefaultSecurityConfig() *SecurityConfig {
	return serverconfig.NewDefaultSecurityConfig()
}

func NewDefaultCSPConfig() *CSPConfig {
	return serverconfig.NewDefaultCSPConfig()
}

func NewDefaultHSTSConfig() *HSTSConfig {
	return serverconfig.NewDefaultHSTSConfig()
}

func NewDefaultCSRFConfig() *CSRFConfig {
	return serverconfig.NewDefaultCSRFConfig()
}
