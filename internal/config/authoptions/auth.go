// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package authoptions

import (
	"crypto/rand"
	"encoding/hex"
	"path/filepath"
	"strings"
	"sync"

	"github.com/spf13/viper"
	xfmt "golang.org/x/exp/errors/fmt"
)

// AuthConfig captures authentication and authorization runtime settings.
type AuthConfig struct {
	Enabled             bool                          `mapstructure:"enabled"`
	Type                string                        `mapstructure:"type"`
	JWT                 *JWTConfig                    `mapstructure:"jwt"`
	HttpAuth            *HttpAuthConfig               `mapstructure:"httpAuth"`
	GrpcAuthentication  bool                          `mapstructure:"grpcAuthentication"`
	GrpcMethodAccess    bool                          `mapstructure:"grpcMethodAccess"`
	GrpcRecordRule      bool                          `mapstructure:"grpcRecordRule"`
	GrpcCompanyFilter   bool                          `mapstructure:"grpcCompanyFilter"`
	GrpcFieldRule       bool                          `mapstructure:"grpcFieldRule"`
	GrpcEntryPolicy     map[string]*EntryMethodConfig `mapstructure:"grpcEntryPolicy"`
	InternalKey         string                        `mapstructure:"internalKey"`
	JobTokenAllowedSANs []string                      `mapstructure:"jobTokenAllowedSANs"`
	AuthzDecisionLog    string                        `mapstructure:"authzDecisionLog"`
	AuthzDecisionAudit  bool                          `mapstructure:"authzDecisionAudit"`
}

var (
	processFallbackInternalKeyOnce sync.Once
	processFallbackInternalKey     string
	processFallbackInternalKeyErr  error
)

func fallbackInternalKeyForProcess() (string, error) {
	processFallbackInternalKeyOnce.Do(func() {
		buf := make([]byte, 32)
		if _, err := rand.Read(buf); err != nil {
			processFallbackInternalKeyErr = err
			return
		}
		processFallbackInternalKey = "dev-auto-" + hex.EncodeToString(buf)
	})
	if processFallbackInternalKeyErr != nil {
		return "", processFallbackInternalKeyErr
	}
	return processFallbackInternalKey, nil
}

func isProductionEnvironment(environment string) bool {
	return strings.EqualFold(strings.TrimSpace(environment), "production")
}

// HttpAuthConfig configures HTTP request authentication behavior.
type HttpAuthConfig struct {
	Enabled         bool     `mapstructure:"enabled"`
	ExcludedPaths   []string `mapstructure:"excludedPaths"`
	ExcludedRegex   []string `mapstructure:"excludedRegex"`
	TokenExtractors []string `mapstructure:"tokenExtractors"`
	ResponseFormat  string   `mapstructure:"responseFormat"`
	CookieName      string   `mapstructure:"cookieName"`
	QueryParamName  string   `mapstructure:"queryParamName"`
}

// NewDefaultAuthConfigWithChoysumRoot builds the default auth config for a Choysum root directory.
func NewDefaultAuthConfigWithChoysumRoot(choysumRoot string) (*AuthConfig, error) {
	trimmedRoot := strings.TrimSpace(choysumRoot)
	if trimmedRoot == "" {
		return nil, xfmt.Errorf("choysumRoot is required")
	}
	if absChoysumRoot, err := filepath.Abs(trimmedRoot); err == nil {
		trimmedRoot = absChoysumRoot
	}
	trimmedRoot = filepath.Clean(trimmedRoot)
	if trimmedRoot == "." || trimmedRoot == string(filepath.Separator) {
		return nil, xfmt.Errorf("choysumRoot must be a non-root directory")
	}
	jwtKeysDir := filepath.Join(trimmedRoot, "jwtkeys")
	httpAuth := defaultHTTPAuthConfig()

	return &AuthConfig{
		Enabled: true,
		Type:    "jwt",
		JWT:     defaultJWTConfig(jwtKeysDir),
		HttpAuth: &HttpAuthConfig{
			Enabled:         httpAuth.Enabled,
			ExcludedPaths:   append([]string(nil), httpAuth.ExcludedPaths...),
			ExcludedRegex:   append([]string(nil), httpAuth.ExcludedRegex...),
			TokenExtractors: append([]string(nil), httpAuth.TokenExtractors...),
			ResponseFormat:  httpAuth.ResponseFormat,
			CookieName:      httpAuth.CookieName,
			QueryParamName:  httpAuth.QueryParamName,
		},
		GrpcAuthentication: true,
		GrpcMethodAccess:   true,
		GrpcRecordRule:     true,
		GrpcCompanyFilter:  true,
		GrpcFieldRule:      true,
		AuthzDecisionLog:   "",
		AuthzDecisionAudit: false,
		InternalKey:        "",
		JobTokenAllowedSANs: []string{
			"task.choysum.internal",
		},
		GrpcEntryPolicy: defaultGrpcEntryPolicy(),
	}, nil
}

// ApplyViperDefaults registers the default auth settings on the provided Viper instance.
func ApplyViperDefaults(v *viper.Viper) error {
	if v == nil {
		return xfmt.Errorf("viper instance is required")
	}

	v.SetDefault("auth.enabled", true)
	v.SetDefault("auth.type", "jwt")
	applyJWTViperDefaults(v)
	v.SetDefault("auth.grpcAuthentication", true)
	v.SetDefault("auth.grpcMethodAccess", true)
	v.SetDefault("auth.grpcRecordRule", true)
	v.SetDefault("auth.grpcCompanyFilter", true)
	v.SetDefault("auth.grpcFieldRule", true)
	httpAuth := defaultHTTPAuthConfig()
	v.SetDefault("auth.httpAuth.enabled", httpAuth.Enabled)
	v.SetDefault("auth.httpAuth.excludedPaths", httpAuth.ExcludedPaths)
	v.SetDefault("auth.httpAuth.excludedRegex", httpAuth.ExcludedRegex)
	v.SetDefault("auth.httpAuth.tokenExtractors", httpAuth.TokenExtractors)
	v.SetDefault("auth.httpAuth.responseFormat", httpAuth.ResponseFormat)
	v.SetDefault("auth.httpAuth.cookieName", httpAuth.CookieName)
	v.SetDefault("auth.httpAuth.queryParamName", httpAuth.QueryParamName)
	v.SetDefault("auth.authzDecisionLog", "")
	v.SetDefault("auth.authzDecisionAudit", false)
	return nil
}

// RejectLegacyJWTIdentityCacheKeys rejects deprecated JWT identity cache keys in config files.
func RejectLegacyJWTIdentityCacheKeys(v *viper.Viper) error {
	return rejectLegacyJWTIdentityCacheKeys(v)
}

// NormalizeAndMergeAuthConfig merges user auth settings with defaults derived from the Choysum path.
func NormalizeAndMergeAuthConfig(cfg *AuthConfig, defaultChoysumPath string, serverEnvironment string) (*AuthConfig, error) {
	defaults, err := NewDefaultAuthConfigWithChoysumRoot(defaultChoysumPath)
	if err != nil {
		return nil, xfmt.Errorf("new default auth config: %w", err)
	}

	if cfg == nil {
		cfg = defaults
	}

	if strings.TrimSpace(cfg.Type) == "" {
		cfg.Type = defaults.Type
	}
	if cfg.JWT == nil && defaults.JWT != nil {
		jwtCopy := *defaults.JWT
		cfg.JWT = &jwtCopy
	} else if cfg.JWT != nil && cfg.JWT.IdentityCache == nil && defaults.JWT != nil && defaults.JWT.IdentityCache != nil {
		identityCacheCopy := *defaults.JWT.IdentityCache
		cfg.JWT.IdentityCache = &identityCacheCopy
	}
	if cfg.HttpAuth == nil && defaults.HttpAuth != nil {
		httpAuthCopy := *defaults.HttpAuth
		cfg.HttpAuth = &httpAuthCopy
	}
	if cfg.GrpcEntryPolicy == nil {
		cfg.GrpcEntryPolicy = defaults.GrpcEntryPolicy
	} else {
		for k, v := range defaults.GrpcEntryPolicy {
			if _, ok := cfg.GrpcEntryPolicy[k]; !ok {
				cfg.GrpcEntryPolicy[k] = v
			}
		}
	}
	if len(cfg.JobTokenAllowedSANs) == 0 {
		cfg.JobTokenAllowedSANs = defaults.JobTokenAllowedSANs
	}

	cfg.InternalKey = strings.TrimSpace(cfg.InternalKey)
	if cfg.InternalKey == "" {
		if isProductionEnvironment(serverEnvironment) {
			return nil, xfmt.Errorf("auth.internalKey must be explicitly configured when server.environment is production")
		}
		fallback, err := fallbackInternalKeyForProcess()
		if err != nil {
			return nil, xfmt.Errorf("generate development auth.internalKey fallback: %w", err)
		}
		cfg.InternalKey = fallback
	}

	return cfg, nil
}

func defaultHTTPAuthConfig() *HttpAuthConfig {
	return &HttpAuthConfig{
		Enabled: true,
		ExcludedPaths: []string{
			"/health",
			"/healthz",
			"/readyz",
			"/metrics",
			"/debug/",
			"/openapi/",
			"/swagger",
			"/",
			"/web/",
			"/bootstrap",
			"/bootstrap/",
			"/index.js",
			"/index.html",
			"/favicon.ico",
			"/assets/",
		},
		ExcludedRegex:   []string{},
		TokenExtractors: []string{"header", "cookie", "query"},
		ResponseFormat:  "json",
		CookieName:      "auth_token",
		QueryParamName:  "token",
	}
}
