// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package config

import (
	serverconfig "github.com/choysum-dev/choysum/internal/server/serverconfig"
	"github.com/spf13/viper"
)

type ServerConfig = serverconfig.ServerConfig
type GrpcClientConfig = serverconfig.GrpcClientConfig

func NewDefaultGrpcClientConfig() *GrpcClientConfig {
	return serverconfig.NewDefaultGrpcClientConfig()
}

func NewDefaultServerConfig() *ServerConfig {
	return serverconfig.NewDefaultServerConfig()
}

func applyServerViperDefaults(v *viper.Viper) error {
	return serverconfig.ApplyViperDefaults(v)
}
