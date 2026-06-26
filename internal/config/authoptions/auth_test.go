// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package authoptions

import (
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func TestApplyViperDefaultsSetsAuthKeys(t *testing.T) {
	v := viper.New()
	if err := ApplyViperDefaults(v); err != nil {
		t.Fatalf("ApplyViperDefaults() error = %v", err)
	}

	if !v.GetBool("auth.enabled") || v.GetString("auth.type") != "jwt" {
		t.Fatalf("unexpected auth defaults: enabled=%v type=%q", v.GetBool("auth.enabled"), v.GetString("auth.type"))
	}
	if !v.GetBool("auth.httpAuth.enabled") || v.GetString("auth.httpAuth.cookieName") != "auth_token" {
		t.Fatalf("unexpected http auth defaults: enabled=%v cookie=%q", v.GetBool("auth.httpAuth.enabled"), v.GetString("auth.httpAuth.cookieName"))
	}
	if v.GetString("auth.authzDecisionLog") != "" || v.GetBool("auth.authzDecisionAudit") {
		t.Fatalf("unexpected authz decision defaults: log=%q audit=%v", v.GetString("auth.authzDecisionLog"), v.GetBool("auth.authzDecisionAudit"))
	}
}

func TestNormalizeAndMergeAuthConfigBackfillsEmptyAllowedSANs(t *testing.T) {
	cfg := &AuthConfig{JobTokenAllowedSANs: []string{}}
	merged, err := NormalizeAndMergeAuthConfig(cfg, t.TempDir(), "development")
	if err != nil {
		t.Fatalf("NormalizeAndMergeAuthConfig() error = %v", err)
	}
	if !reflect.DeepEqual(merged.JobTokenAllowedSANs, []string{"task.choysum.internal"}) {
		t.Fatalf("JobTokenAllowedSANs = %#v, want default SAN", merged.JobTokenAllowedSANs)
	}
}

func TestNormalizeAndMergeAuthConfigAutoGeneratesInternalKeyOutsideProduction(t *testing.T) {
	first, err := NormalizeAndMergeAuthConfig(&AuthConfig{}, t.TempDir(), "development")
	if err != nil {
		t.Fatalf("NormalizeAndMergeAuthConfig() first call error = %v", err)
	}
	second, err := NormalizeAndMergeAuthConfig(&AuthConfig{}, t.TempDir(), "default")
	if err != nil {
		t.Fatalf("NormalizeAndMergeAuthConfig() second call error = %v", err)
	}
	if strings.TrimSpace(first.InternalKey) == "" {
		t.Fatal("expected non-empty generated auth.internalKey outside production")
	}
	if first.InternalKey != second.InternalKey {
		t.Fatalf("expected process fallback auth.internalKey to stay stable, first=%q second=%q", first.InternalKey, second.InternalKey)
	}
}

func TestNormalizeAndMergeAuthConfigRequiresExplicitInternalKeyInProduction(t *testing.T) {
	_, err := NormalizeAndMergeAuthConfig(&AuthConfig{}, t.TempDir(), "production")
	if err == nil {
		t.Fatal("expected production auth.internalKey validation error")
	}
	if !strings.Contains(err.Error(), "auth.internalKey must be explicitly configured") {
		t.Fatalf("unexpected error: %v", err)
	}

	merged, err := NormalizeAndMergeAuthConfig(&AuthConfig{InternalKey: "  prod-secret  "}, t.TempDir(), "production")
	if err != nil {
		t.Fatalf("expected explicit production auth.internalKey to pass, got %v", err)
	}
	if merged.InternalKey != "prod-secret" {
		t.Fatalf("internalKey = %q, want trimmed explicit key", merged.InternalKey)
	}
}
