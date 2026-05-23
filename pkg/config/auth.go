// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package config

import (
	authoptions "github.com/choysum-dev/choysum/internal/config/authoptions"
	"github.com/spf13/viper"
	xfmt "golang.org/x/exp/errors/fmt"
)

type JWTConfig = authoptions.JWTConfig
type JWTIdentityCacheConfig = authoptions.JWTIdentityCacheConfig
type AuthConfig = authoptions.AuthConfig
type EntryMethodConfig = authoptions.EntryMethodConfig
type EntryRecordRuleAllow = authoptions.EntryRecordRuleAllow
type HttpAuthConfig = authoptions.HttpAuthConfig

func NewDefaultAuthConfig() *AuthConfig {
	choysumRoot, err := ResolveDefaultChoysumPaths()
	if err != nil {
		panic(err)
	}
	cfg, err := NewDefaultAuthConfigWithChoysumRoot(choysumRoot)
	if err != nil {
		panic(err)
	}
	return cfg
}

func NewDefaultAuthConfigWithChoysumRoot(choysumRoot string) (*AuthConfig, error) {
	return authoptions.NewDefaultAuthConfigWithChoysumRoot(choysumRoot)
}

func applyAuthViperDefaults(v *viper.Viper) error {
	return authoptions.ApplyViperDefaults(v)
}

func rejectLegacyJWTIdentityCacheKeys(v *viper.Viper) error {
	return authoptions.RejectLegacyJWTIdentityCacheKeys(v)
}

func (c *Config) normalizeAndMergeAuthConfig() error {
	if c == nil {
		return xfmt.Errorf("config is required")
	}
	merged, err := authoptions.NormalizeAndMergeAuthConfig(c.Auth, c.DefaultChoysumPath)
	if err != nil {
		return err
	}
	c.Auth = merged
	return nil
}
