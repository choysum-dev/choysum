// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package config

import (
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestNewDefaultAuthConfig(t *testing.T) {
	cfg := NewDefaultAuthConfig()
	if !cfg.Enabled || cfg.Type != "jwt" {
		t.Fatalf("unexpected auth top-level defaults: %#v", cfg)
	}
	if cfg.JWT == nil || cfg.HttpAuth == nil {
		t.Fatalf("expected nested auth defaults, got %#v", cfg)
	}
	if cfg.JWT.AccessTokenExpiry != 15*time.Minute || cfg.JWT.RefreshTokenExpiry != 7*24*time.Hour {
		t.Fatalf("unexpected jwt expiry defaults: %#v", cfg.JWT)
	}
	if filepath.Base(cfg.JWT.PrivateKeyFile) != "private.pem" || filepath.Base(cfg.JWT.PublicKeyFile) != "public.pem" {
		t.Fatalf("unexpected jwt key file defaults: private=%q public=%q", cfg.JWT.PrivateKeyFile, cfg.JWT.PublicKeyFile)
	}
	if filepath.Base(filepath.Dir(cfg.JWT.PrivateKeyFile)) != "jwtkeys" || filepath.Base(filepath.Dir(cfg.JWT.PublicKeyFile)) != "jwtkeys" {
		t.Fatalf("expected jwt keys to default under jwtkeys dir, got private=%q public=%q", cfg.JWT.PrivateKeyFile, cfg.JWT.PublicKeyFile)
	}
	if !cfg.JWT.AutoGenerateKeys || cfg.JWT.RevokeStore != "database" || cfg.JWT.IdentityCache == nil || !cfg.JWT.IdentityCache.Enabled || cfg.JWT.IdentityCache.Backend != "memory" || cfg.JWT.IdentityCache.Size != 10000 || cfg.JWT.IdentityCache.TTL != 5*time.Minute {
		t.Fatalf("unexpected jwt defaults: %#v", cfg.JWT)
	}
	if !cfg.HttpAuth.Enabled || cfg.HttpAuth.ResponseFormat != "json" || cfg.HttpAuth.CookieName != "auth_token" || cfg.HttpAuth.QueryParamName != "token" {
		t.Fatalf("unexpected http auth defaults: %#v", cfg.HttpAuth)
	}
	if !reflect.DeepEqual(cfg.HttpAuth.TokenExtractors, []string{"header", "cookie", "query"}) {
		t.Fatalf("unexpected token extractors: %#v", cfg.HttpAuth.TokenExtractors)
	}
	hasExcludedPath := func(paths []string, target string) bool {
		for _, p := range paths {
			if p == target {
				return true
			}
		}
		return false
	}
	if !hasExcludedPath(cfg.HttpAuth.ExcludedPaths, "/bootstrap") || !hasExcludedPath(cfg.HttpAuth.ExcludedPaths, "/bootstrap/") {
		t.Fatalf("expected bootstrap paths in auth.httpAuth.excludedPaths, got %#v", cfg.HttpAuth.ExcludedPaths)
	}
	if !cfg.GrpcAuthentication || !cfg.GrpcMethodAccess || !cfg.GrpcRecordRule || !cfg.GrpcCompanyFilter || !cfg.GrpcFieldRule {
		t.Fatalf("expected grpc auth switches enabled, got %#v", cfg)
	}
	if !reflect.DeepEqual(cfg.JobTokenAllowedSANs, []string{"task.choysum.internal"}) {
		t.Fatalf("unexpected JobTokenAllowedSANs: %#v", cfg.JobTokenAllowedSANs)
	}

	register := cfg.GrpcEntryPolicy["auth.User/Register"]
	if register == nil || !register.SkipAuthentication || !register.SkipMethodAccess || !register.SkipCompanyFilter || !register.SkipFieldRule {
		t.Fatalf("unexpected register policy: %#v", register)
	}
	login := cfg.GrpcEntryPolicy["auth.User/Login"]
	if login == nil || !login.SkipAuthentication || !login.SkipMethodAccess || !login.SkipCompanyFilter || !login.SkipFieldRule {
		t.Fatalf("unexpected login policy: %#v", login)
	}
	refresh := cfg.GrpcEntryPolicy["auth.User/RefreshTokens"]
	if refresh == nil || !refresh.SkipAuthentication || !refresh.SkipMethodAccess || !refresh.SkipCompanyFilter || !refresh.SkipFieldRule {
		t.Fatalf("unexpected refresh policy: %#v", refresh)
	}
	checkAccess := cfg.GrpcEntryPolicy["auth.User/CheckMethodAccess"]
	if checkAccess == nil || !checkAccess.SkipMethodAccess || !checkAccess.SkipCompanyFilter || !checkAccess.SkipFieldRule {
		t.Fatalf("unexpected CheckMethodAccess policy: %#v", checkAccess)
	}
	getRecordRule := cfg.GrpcEntryPolicy["auth.User/GetRecordRuleCondition"]
	if getRecordRule == nil || !getRecordRule.SkipMethodAccess || !getRecordRule.SkipCompanyFilter || !getRecordRule.SkipFieldRule {
		t.Fatalf("unexpected GetRecordRuleCondition policy: %#v", getRecordRule)
	}
	getActiveLanguages := cfg.GrpcEntryPolicy["base.Language/GetActiveLanguages"]
	if getActiveLanguages == nil || !getActiveLanguages.SkipAuthentication || !getActiveLanguages.SkipMethodAccess || !getActiveLanguages.SkipCompanyFilter || !getActiveLanguages.SkipFieldRule {
		t.Fatalf("unexpected GetActiveLanguages policy: %#v", getActiveLanguages)
	}
	if len(getActiveLanguages.RecordRuleAllow) == 0 || getActiveLanguages.RecordRuleAllow[0].Model != "base.Language" {
		t.Fatalf("unexpected GetActiveLanguages record rule allow: %#v", getActiveLanguages.RecordRuleAllow)
	}
	bootstrapInitialize := cfg.GrpcEntryPolicy["bootstrap.Workspace/Initialize"]
	if bootstrapInitialize == nil || !bootstrapInitialize.SkipAuthentication {
		t.Fatalf("unexpected bootstrap initialize policy: %#v", bootstrapInitialize)
	}
	bootstrapGetStatus := cfg.GrpcEntryPolicy["bootstrap.Workspace/GetInitializationStatus"]
	if bootstrapGetStatus == nil || !bootstrapGetStatus.SkipAuthentication {
		t.Fatalf("unexpected bootstrap get status policy: %#v", bootstrapGetStatus)
	}
	if len(register.RecordRuleAllow) == 0 || len(login.RecordRuleAllow) == 0 || len(refresh.RecordRuleAllow) == 0 {
		t.Fatal("expected default record rule allowlists to be populated")
	}
}
