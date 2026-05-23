// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package serverconfig

type SecurityConfig struct {
	CSP  *CSPConfig  `mapstructure:"csp"`
	HSTS *HSTSConfig `mapstructure:"hsts"`
	CSRF *CSRFConfig `mapstructure:"csrf"`
}

type CSPConfig struct {
	Enabled       bool     `mapstructure:"enabled"`
	ReportOnly    bool     `mapstructure:"report_only"`
	ReportURI     string   `mapstructure:"report_uri"`
	ExcludedPaths []string `mapstructure:"excluded_paths"`

	Development CSPDirectives `mapstructure:"development"`
	Production  CSPDirectives `mapstructure:"production"`
}

type CSPDirectives struct {
	DefaultSrc     []string `mapstructure:"default-src"`
	ScriptSrc      []string `mapstructure:"script-src"`
	StyleSrc       []string `mapstructure:"style-src"`
	ImgSrc         []string `mapstructure:"img-src"`
	ConnectSrc     []string `mapstructure:"connect-src"`
	FontSrc        []string `mapstructure:"font-src"`
	ObjectSrc      []string `mapstructure:"object-src"`
	MediaSrc       []string `mapstructure:"media-src"`
	FrameSrc       []string `mapstructure:"frame-src"`
	WorkerSrc      []string `mapstructure:"worker-src"`
	FrameAncestors []string `mapstructure:"frame-ancestors"`
	FormAction     []string `mapstructure:"form-action"`
	BaseURI        []string `mapstructure:"base-uri"`
	ChildSrc       []string `mapstructure:"child-src"`
	ManifestSrc    []string `mapstructure:"manifest-src"`
}

type HSTSConfig struct {
	Enabled           bool `mapstructure:"enabled"`
	MaxAge            int  `mapstructure:"max_age"`
	IncludeSubdomains bool `mapstructure:"include_subdomains"`
	Preload           bool `mapstructure:"preload"`
}

type CSRFConfig struct {
	Enabled       bool     `mapstructure:"enabled"`
	CookieName    string   `mapstructure:"cookieName"`
	HeaderName    string   `mapstructure:"headerName"`
	CookiePath    string   `mapstructure:"cookiePath"`
	CookieDomain  string   `mapstructure:"cookieDomain"`
	Secure        bool     `mapstructure:"secure"`
	SameSite      string   `mapstructure:"sameSite"`
	MaxAge        int      `mapstructure:"maxAge"`
	ExcludedPaths []string `mapstructure:"excludedPaths"`
}

func NewDefaultSecurityConfig() *SecurityConfig {
	return &SecurityConfig{
		CSP:  NewDefaultCSPConfig(),
		HSTS: NewDefaultHSTSConfig(),
		CSRF: NewDefaultCSRFConfig(),
	}
}

func NewDefaultCSPConfig() *CSPConfig {
	return &CSPConfig{
		Enabled:    true,
		ReportOnly: false,
		ExcludedPaths: []string{
			"/health",
			"/metrics",
			"/debug/",
			"/openapi/",
			"/swagger",
			"/assets/",
		},
		Development: CSPDirectives{
			DefaultSrc:     []string{"'self'"},
			ScriptSrc:      []string{"'self'", "'unsafe-eval'", "'unsafe-inline'"},
			StyleSrc:       []string{"'self'", "'unsafe-inline'"},
			ImgSrc:         []string{"'self'", "data:"},
			ConnectSrc:     []string{"'self'", "ws:", "wss:"},
			FontSrc:        []string{"'self'"},
			ObjectSrc:      []string{"'none'"},
			MediaSrc:       []string{"'self'"},
			FrameSrc:       []string{"'none'"},
			WorkerSrc:      []string{"'self'", "blob:"},
			FrameAncestors: []string{"'none'"},
			FormAction:     []string{"'self'"},
			ChildSrc:       []string{"'none'"},
		},
		Production: CSPDirectives{
			DefaultSrc:     []string{"'self'"},
			ScriptSrc:      []string{"'self'"},
			StyleSrc:       []string{"'self'"},
			ImgSrc:         []string{"'self'", "data:"},
			ConnectSrc:     []string{"'self'"},
			FontSrc:        []string{"'self'"},
			ObjectSrc:      []string{"'none'"},
			MediaSrc:       []string{"'self'"},
			FrameSrc:       []string{"'none'"},
			WorkerSrc:      []string{"'self'"},
			FrameAncestors: []string{"'none'"},
			FormAction:     []string{"'self'"},
			BaseURI:        []string{"'self'"},
			ChildSrc:       []string{"'none'"},
		},
	}
}

func NewDefaultHSTSConfig() *HSTSConfig {
	return &HSTSConfig{
		Enabled:           true,
		MaxAge:            31536000,
		IncludeSubdomains: true,
		Preload:           true,
	}
}

func NewDefaultCSRFConfig() *CSRFConfig {
	return &CSRFConfig{
		Enabled:    true,
		CookieName: "XSRF-TOKEN",
		HeaderName: "X-XSRF-TOKEN",
		Secure:     false,
		CookiePath: "/",
		SameSite:   "strict",
		MaxAge:     0,
		ExcludedPaths: []string{
			"/health",
			"/metrics",
			"/debug/",
			"/openapi/",
			"/swagger",
			"/assets/",
			"/api/auth/login",
			"/api/auth/register",
			"/api/auth/refresh",
		},
	}
}
