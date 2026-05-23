// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package serverconfig

import (
	"fmt"

	"github.com/spf13/viper"
)

type ServerConfig struct {
	BindAddress        string            `mapstructure:"bindAddress"`
	Port               int               `mapstructure:"port"`
	EnableGzip         bool              `mapstructure:"enableGzip"`
	EnabledTLS         bool              `mapstructure:"enabledTLS"`
	TLSCertFile        string            `mapstructure:"tlsCertFile"`
	TLSKeyFile         string            `mapstructure:"tlsKeyFile"`
	TLSCaFile          string            `mapstructure:"tlsCaFile"`
	TLSServerName      string            `mapstructure:"tlsServerName"`
	EnableGrpcWebProxy bool              `mapstructure:"enableGrpcWebProxy"`
	HotReload          bool              `mapstructure:"hotReload"`
	Register           string            `mapstructure:"register"`
	Environment        string            `mapstructure:"environment"`
	RuntimeEngine      string            `mapstructure:"runtimeEngine"`
	Security           *SecurityConfig   `mapstructure:"security"`
	GrpcClient         *GrpcClientConfig `mapstructure:"grpcClient"`
	WebBaseURL         string            `mapstructure:"webBaseUrl"`
	RootRedirectURL    string            `mapstructure:"rootRedirectUrl"`
	JsEngineFactory    string            `mapstructure:"jsEngineFactory"`
	JsExecutorFactory  string            `mapstructure:"jsExecutorFactory"`
}

type GrpcClientConfig struct {
	MaxCachedConns int `mapstructure:"maxCachedConns"`
}

func NewDefaultGrpcClientConfig() *GrpcClientConfig {
	return &GrpcClientConfig{MaxCachedConns: 128}
}

func NewDefaultServerConfig() *ServerConfig {
	return &ServerConfig{
		BindAddress:        "0.0.0.0",
		Port:               9527,
		EnableGzip:         true,
		EnabledTLS:         false,
		TLSCertFile:        "",
		TLSKeyFile:         "",
		TLSCaFile:          "",
		TLSServerName:      "",
		EnableGrpcWebProxy: true,
		HotReload:          false,
		Register:           "local",
		Environment:        "default",
		RuntimeEngine:      "default",
		Security:           NewDefaultSecurityConfig(),
		GrpcClient:         NewDefaultGrpcClientConfig(),
		WebBaseURL:         "/web",
		RootRedirectURL:    "",
		JsEngineFactory:    "quickjs",
		JsExecutorFactory:  "default",
	}
}

func ApplyViperDefaults(v *viper.Viper) error {
	if v == nil {
		return fmt.Errorf("viper instance is required")
	}
	v.SetDefault("server.jsExecutorFactory", "default")
	return nil
}
