// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package authoptions

import (
	"reflect"
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
	merged, err := NormalizeAndMergeAuthConfig(cfg, t.TempDir())
	if err != nil {
		t.Fatalf("NormalizeAndMergeAuthConfig() error = %v", err)
	}
	if !reflect.DeepEqual(merged.JobTokenAllowedSANs, []string{"task.choysum.internal"}) {
		t.Fatalf("JobTokenAllowedSANs = %#v, want default SAN", merged.JobTokenAllowedSANs)
	}
}
