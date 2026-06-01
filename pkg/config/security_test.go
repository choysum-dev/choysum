// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package config

import "testing"

func TestNewDefaultSecurityConfig(t *testing.T) {
	cfg := NewDefaultSecurityConfig()
	if cfg.CSP == nil || cfg.HSTS == nil || cfg.CSRF == nil {
		t.Fatalf("expected nested security configs, got %#v", cfg)
	}
}

func TestNewDefaultCSPConfig(t *testing.T) {
	cfg := NewDefaultCSPConfig()
	if !cfg.Enabled || cfg.ReportOnly {
		t.Fatalf("unexpected CSP flags: %#v", cfg)
	}
	if len(cfg.ExcludedPaths) == 0 {
		t.Fatal("expected CSP excluded paths")
	}
	if len(cfg.Development.ScriptSrc) == 0 || len(cfg.Production.ScriptSrc) == 0 {
		t.Fatalf("expected CSP directives to be initialized: %#v", cfg)
	}
	if len(cfg.Production.BaseURI) == 0 {
		t.Fatalf("expected production BaseURI defaults: %#v", cfg.Production)
	}
}

func TestNewDefaultHSTSConfig(t *testing.T) {
	cfg := NewDefaultHSTSConfig()
	if !cfg.Enabled || cfg.MaxAge != 31536000 || !cfg.IncludeSubdomains || !cfg.Preload {
		t.Fatalf("unexpected HSTS defaults: %#v", cfg)
	}
}

func TestNewDefaultCSRFConfig(t *testing.T) {
	cfg := NewDefaultCSRFConfig()
	if !cfg.Enabled || cfg.CookieName != "XSRF-TOKEN" || cfg.HeaderName != "X-XSRF-TOKEN" || cfg.CookiePath != "/" || cfg.SameSite != "strict" {
		t.Fatalf("unexpected CSRF defaults: %#v", cfg)
	}
	if len(cfg.ExcludedPaths) == 0 {
		t.Fatal("expected CSRF excluded paths")
	}
}
