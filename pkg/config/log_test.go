// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package config

import "testing"

func TestNewDefaultLogConfig(t *testing.T) {
	cfg := NewDefaultLogConfig()
	if cfg.Format != "" || cfg.Level != "info" {
		t.Fatalf("unexpected log defaults: %#v", cfg)
	}
	if cfg.Access {
		t.Fatalf("Access default = true, want false")
	}
	if cfg.AccessSlowMs != 1000 {
		t.Fatalf("AccessSlowMs = %d, want 1000", cfg.AccessSlowMs)
	}
}
