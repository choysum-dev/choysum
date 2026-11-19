// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package defaultscope

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/config"
)

func TestSQLitePoolLimits(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		maxOpen  int
		maxIdle  int
		wantOpen int
		wantIdle int
	}{
		{name: "defaults", maxOpen: 0, maxIdle: 0, wantOpen: 4, wantIdle: 2},
		{name: "negative defaults", maxOpen: -1, maxIdle: -3, wantOpen: 4, wantIdle: 2},
		{name: "floor open and idle", maxOpen: 1, maxIdle: 1, wantOpen: 2, wantIdle: 2},
		{name: "cap open", maxOpen: 32, maxIdle: 16, wantOpen: 8, wantIdle: 8},
		{name: "idle clamped to open", maxOpen: 3, maxIdle: 10, wantOpen: 3, wantIdle: 3},
		{name: "respect mid values", maxOpen: 6, maxIdle: 4, wantOpen: 6, wantIdle: 4},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			open, idle := sqlitePoolLimits(tc.maxOpen, tc.maxIdle)
			if open != tc.wantOpen || idle != tc.wantIdle {
				t.Fatalf("sqlitePoolLimits(%d, %d) = (%d, %d), want (%d, %d)",
					tc.maxOpen, tc.maxIdle, open, idle, tc.wantOpen, tc.wantIdle)
			}
		})
	}
}

func TestNewDbAppliesSQLitePoolConfig(t *testing.T) {
	cfg := &config.Config{
		Db: &config.DbConfig{
			Dialect:      "sqlite",
			DSN:          filepath.Join(t.TempDir(), "pool.db"),
			MaxOpenConns: 9, // above sqlite cap → 8
			MaxIdleConns: 7,
		},
	}
	runtimeScope := NewDefaultScope(context.Background(), scopetest.FactoryInputFromConfig(cfg), testLogger()).(*defaultScope)
	sqlDB, err := runtimeScope.db.DB()
	if err != nil {
		t.Fatalf("db.DB: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	if sqlDB.Stats().MaxOpenConnections != 8 {
		t.Fatalf("MaxOpenConnections = %d, want 8 (capped)", sqlDB.Stats().MaxOpenConnections)
	}
}

func TestNewDbUsesSQLiteDefaultsWhenPoolUnset(t *testing.T) {
	cfg := &config.Config{
		Db: &config.DbConfig{
			Dialect: "sqlite",
			DSN:     filepath.Join(t.TempDir(), "pool-defaults.db"),
			// MaxOpen/Idle left at 0 → documented sqlite defaults via sqlitePoolLimits.
		},
	}
	runtimeScope := NewDefaultScope(context.Background(), scopetest.FactoryInputFromConfig(cfg), testLogger()).(*defaultScope)
	sqlDB, err := runtimeScope.db.DB()
	if err != nil {
		t.Fatalf("db.DB: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	if sqlDB.Stats().MaxOpenConnections != 4 {
		t.Fatalf("MaxOpenConnections = %d, want documented sqlite default 4", sqlDB.Stats().MaxOpenConnections)
	}
}
