// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package envconfig

import "testing"

func TestNewDefaultFrontendEnv(t *testing.T) {
	t.Run("development with trimmed empty base URL falls back to root", func(t *testing.T) {
		env := NewDefaultFrontendEnv("   ", false)
		if env["BASE_URL"] != "/" {
			t.Fatalf("BASE_URL = %#v, want /", env["BASE_URL"])
		}
		if env["MODE"] != "development" || env["PROD"] != false || env["DEV"] != true {
			t.Fatalf("unexpected development flags: %#v", env)
		}
		if env["SSR"] != false {
			t.Fatalf("SSR = %#v, want false", env["SSR"])
		}
	})

	t.Run("production trims trailing slash and keeps static app keys", func(t *testing.T) {
		env := NewDefaultFrontendEnv("/portal/", true)
		if env["BASE_URL"] != "/portal/" {
			t.Fatalf("BASE_URL = %#v, want /portal/", env["BASE_URL"])
		}
		if env["MODE"] != "production" || env["PROD"] != true || env["DEV"] != false {
			t.Fatalf("unexpected production flags: %#v", env)
		}
		if env["CHOYSUM_APP_NAME"] != "Choysum" || env["CHOYSUM_ENABLE_REGISTRATION"] != true || env["CHOYSUM_CSRF_ENABLED"] != true {
			t.Fatalf("unexpected static frontend env keys: %#v", env)
		}
	})
}

func TestNewDefaultBackendEnv(t *testing.T) {
	env := NewDefaultBackendEnv()
	if env["CHOYSUM_SOFT_DELETE_ENABLED"] != true {
		t.Fatalf("CHOYSUM_SOFT_DELETE_ENABLED = %#v, want true", env["CHOYSUM_SOFT_DELETE_ENABLED"])
	}
}
