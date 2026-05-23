// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package authoptions

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
	xfmt "golang.org/x/exp/errors/fmt"
)

// JWTConfig captures JWT issuance, key, and cache settings.
type JWTConfig struct {
	AccessTokenExpiry  time.Duration           `mapstructure:"accessTokenExpiry"`
	RefreshTokenExpiry time.Duration           `mapstructure:"refreshTokenExpiry"`
	PrivateKeyFile     string                  `mapstructure:"privateKeyFile"`
	PublicKeyFile      string                  `mapstructure:"publicKeyFile"`
	AutoGenerateKeys   bool                    `mapstructure:"autoGenerateKeys"`
	RevokeStore        string                  `mapstructure:"revokeStore"`
	IdentityCache      *JWTIdentityCacheConfig `mapstructure:"identityCache"`
}

// JWTIdentityCacheConfig configures the identity cache used during JWT validation.
type JWTIdentityCacheConfig struct {
	Enabled bool          `mapstructure:"enabled"`
	Backend string        `mapstructure:"backend"`
	Size    int           `mapstructure:"size"`
	TTL     time.Duration `mapstructure:"ttl"`
}

func defaultJWTConfig(jwtKeysDir string) *JWTConfig {
	return &JWTConfig{
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
		AutoGenerateKeys:   true,
		PrivateKeyFile:     filepath.Join(jwtKeysDir, "private.pem"),
		PublicKeyFile:      filepath.Join(jwtKeysDir, "public.pem"),
		RevokeStore:        "database",
		IdentityCache: &JWTIdentityCacheConfig{
			Enabled: true,
			Backend: "memory",
			Size:    10000,
			TTL:     5 * time.Minute,
		},
	}
}

func applyJWTViperDefaults(v *viper.Viper) {
	v.SetDefault("auth.jwt.accessTokenExpiry", (15 * time.Minute).String())
	v.SetDefault("auth.jwt.refreshTokenExpiry", (7 * 24 * time.Hour).String())
	v.SetDefault("auth.jwt.autoGenerateKeys", true)
	v.SetDefault("auth.jwt.revokeStore", "database")
	v.SetDefault("auth.jwt.identityCache.enabled", true)
	v.SetDefault("auth.jwt.identityCache.backend", "memory")
	v.SetDefault("auth.jwt.identityCache.size", 10000)
	v.SetDefault("auth.jwt.identityCache.ttl", (5 * time.Minute).String())
}

func rejectLegacyJWTIdentityCacheKeys(v *viper.Viper) error {
	invalid := make([]string, 0, 3)
	appendLegacyKey := func(legacyKey string, newKey string) {
		if v.InConfig(legacyKey) {
			invalid = append(invalid, legacyKey+" (use "+newKey+")")
		}
	}

	appendLegacyKey("auth.jwt.cacheEnabled", "auth.jwt.identityCache.enabled")
	appendLegacyKey("auth.jwt.cacheSize", "auth.jwt.identityCache.size")
	appendLegacyKey("auth.jwt.cacheTTL", "auth.jwt.identityCache.ttl")

	if len(invalid) == 0 {
		return nil
	}
	return xfmt.Errorf("legacy JWT identity cache config keys are no longer supported: %s", strings.Join(invalid, ", "))
}
