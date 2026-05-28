// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package config

import "testing"

func TestNewDefaultDbConfig(t *testing.T) {
	cfg := NewDefaultDbConfig()
	if cfg.Dialect != "sqlite" || cfg.MaxIdleConns != 2*maxProcs || cfg.MaxOpenConns != 4*maxProcs || cfg.ConnMaxLifetime != 3600 {
		t.Fatalf("unexpected db defaults: %#v", cfg)
	}
}

func TestDefaultSQLitePath(t *testing.T) {
	got := DefaultSQLitePath("/tmp/choysum-state")
	if got != "/tmp/choysum-state/choysum.sqlite" {
		t.Fatalf("DefaultSQLitePath() = %q", got)
	}
}

func TestDefaultSQLitePath_EmptyRoot(t *testing.T) {
	if got := DefaultSQLitePath("   "); got != "" {
		t.Fatalf("DefaultSQLitePath(whitespace) = %q, want empty", got)
	}
}

func TestDefaultSQLiteDSN(t *testing.T) {
	got := DefaultSQLiteDSN("/tmp/choysum-state")
	want := "file:/tmp/choysum-state/choysum.sqlite?mode=rwc&_fk=1&_busy_timeout=60000&_journal_mode=WAL"
	if got != want {
		t.Fatalf("DefaultSQLiteDSN() = %q, want %q", got, want)
	}
}
