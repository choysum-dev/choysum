// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package provision

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type setupStateTestScope struct {
	ctx    context.Context
	db     *gorm.DB
	config *config.Config
	logger *slog.Logger
}

func (e *setupStateTestScope) Run(fn func(scope.Scope) error) error { return fn(e) }

func (e *setupStateTestScope) Transactor() scope.Transactor {
	return scopetest.NewPassthroughTransactor(e)
}

func (e *setupStateTestScope) Session() *scope.Session {
	if e == nil || e.db == nil {
		return nil
	}
	return &scope.Session{DB: e.db}
}

func (e *setupStateTestScope) WithContext(ctx context.Context) scope.Scope {
	return &setupStateTestScope{ctx: ctx, db: e.db, config: e.config, logger: e.logger}
}

func (e *setupStateTestScope) Context() context.Context {
	if e != nil && e.ctx != nil {
		return e.ctx
	}
	return context.Background()
}

func (e *setupStateTestScope) Logger() *slog.Logger {
	if e != nil && e.logger != nil {
		return e.logger
	}
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func (e *setupStateTestScope) Config() *config.Config {
	if e != nil && e.config != nil {
		return e.config
	}
	return &config.Config{}
}

func (e *setupStateTestScope) FactoryInput() scope.FactoryInput {
	return scopetest.FactoryInputFromConfig(e.Config())
}

func newSetupStateScope(t *testing.T) (*setupStateTestScope, *gorm.DB) {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "setup_state.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	return &setupStateTestScope{ctx: context.Background(), db: db}, db
}

func mustExec(t *testing.T, db *gorm.DB, sql string, values ...any) {
	t.Helper()
	if err := db.Exec(sql, values...).Error; err != nil {
		t.Fatalf("exec %q: %v", sql, err)
	}
}

func createMetaModuleTable(t *testing.T, db *gorm.DB) {
	t.Helper()
	mustExec(t, db, `CREATE TABLE meta_module (
		id TEXT PRIMARY KEY,
		name TEXT,
		status TEXT,
		deleted_at DATETIME,
		created_at DATETIME,
		updated_at DATETIME
	)`)
}

func insertAuthModule(t *testing.T, db *gorm.DB, status meta.Status) {
	t.Helper()
	mustExec(t, db, `INSERT INTO meta_module (id, name, status, deleted_at, created_at, updated_at) VALUES (?, ?, ?, NULL, ?, ?)`,
		"mod-auth", "auth", string(status), time.Now(), time.Now())
}

func createAuthAppSettingTable(t *testing.T, db *gorm.DB) {
	t.Helper()
	mustExec(t, db, `CREATE TABLE auth_app_setting (
		id TEXT PRIMARY KEY,
		key TEXT NOT NULL UNIQUE,
		value TEXT,
		created_at DATETIME,
		updated_at DATETIME
	)`)
}

func TestResolveNilScope(t *testing.T) {
	phase, err := Resolve(nil)
	if err == nil {
		t.Fatal("Resolve(nil) expected error")
	}
	if phase != PhaseEmpty {
		t.Fatalf("phase = %q, want %q", phase, PhaseEmpty)
	}
}

func TestResolveNilSession(t *testing.T) {
	runtimeScope := &setupStateTestScope{ctx: context.Background(), db: nil}
	phase, err := Resolve(runtimeScope)
	if err == nil {
		t.Fatal("expected session error")
	}
	if phase != PhaseEmpty {
		t.Fatalf("phase = %q, want %q", phase, PhaseEmpty)
	}
}

func TestResolveEmptyWhenMetaModuleMissing(t *testing.T) {
	runtimeScope, _ := newSetupStateScope(t)
	phase, err := Resolve(runtimeScope)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if phase != PhaseEmpty {
		t.Fatalf("phase = %q, want %q", phase, PhaseEmpty)
	}
}

func TestResolveEmptyWhenAuthNotInstalled(t *testing.T) {
	runtimeScope, db := newSetupStateScope(t)
	createMetaModuleTable(t, db)
	insertAuthModule(t, db, meta.ToInstall)

	phase, err := Resolve(runtimeScope)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if phase != PhaseEmpty {
		t.Fatalf("phase = %q, want %q", phase, PhaseEmpty)
	}
}

func TestResolveEmptyWhenAuthModuleRowMissing(t *testing.T) {
	runtimeScope, db := newSetupStateScope(t)
	createMetaModuleTable(t, db)
	mustExec(t, db, `INSERT INTO meta_module (id, name, status, deleted_at, created_at, updated_at) VALUES (?, ?, ?, NULL, ?, ?)`,
		"mod-web", "web", string(meta.Installed), time.Now(), time.Now())

	phase, err := Resolve(runtimeScope)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if phase != PhaseEmpty {
		t.Fatalf("phase = %q, want %q", phase, PhaseEmpty)
	}
}

func TestResolveEmptyWhenAuthAppSettingTableMissing(t *testing.T) {
	runtimeScope, db := newSetupStateScope(t)
	createMetaModuleTable(t, db)
	insertAuthModule(t, db, meta.Installed)

	phase, err := Resolve(runtimeScope)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if phase != PhaseEmpty {
		t.Fatalf("phase = %q, want %q (missing auth_app_setting must not be Ready)", phase, PhaseEmpty)
	}
}

func TestResolveEmptyWhenMetaModuleQueryBroken(t *testing.T) {
	runtimeScope, db := newSetupStateScope(t)
	mustExec(t, db, `CREATE TABLE meta_module (id TEXT PRIMARY KEY, name TEXT, status TEXT)`)
	mustExec(t, db, `INSERT INTO meta_module (id, name, status) VALUES (?, ?, ?)`,
		"mod-auth", "auth", string(meta.Installed))

	phase, err := Resolve(runtimeScope)
	if err == nil {
		t.Fatal("expected query error")
	}
	if phase != PhaseEmpty {
		t.Fatalf("phase = %q, want %q", phase, PhaseEmpty)
	}
}

func TestResolveEmptyWhenAuthAppSettingQueryBroken(t *testing.T) {
	runtimeScope, db := newSetupStateScope(t)
	createMetaModuleTable(t, db)
	insertAuthModule(t, db, meta.Installed)
	mustExec(t, db, `CREATE TABLE auth_app_setting (id TEXT PRIMARY KEY, key TEXT NOT NULL UNIQUE)`)

	phase, err := Resolve(runtimeScope)
	if err == nil {
		t.Fatal("expected auth_app_setting query error")
	}
	if phase != PhaseEmpty {
		t.Fatalf("phase = %q, want %q", phase, PhaseEmpty)
	}
}

func TestResolveSetupPendingWhenMarkerAbsent(t *testing.T) {
	runtimeScope, db := newSetupStateScope(t)
	createMetaModuleTable(t, db)
	insertAuthModule(t, db, meta.Installed)
	createAuthAppSettingTable(t, db)

	phase, err := Resolve(runtimeScope)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if phase != PhaseSetupPending {
		t.Fatalf("phase = %q, want %q", phase, PhaseSetupPending)
	}
}

func TestResolveSetupPendingWhenMarkerEmpty(t *testing.T) {
	runtimeScope, db := newSetupStateScope(t)
	createMetaModuleTable(t, db)
	insertAuthModule(t, db, meta.Installed)
	createAuthAppSettingTable(t, db)
	mustExec(t, db, `INSERT INTO auth_app_setting (id, key, value, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		"s1", authSetupCompletedAtKey, "   ", time.Now(), time.Now())

	phase, err := Resolve(runtimeScope)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if phase != PhaseSetupPending {
		t.Fatalf("phase = %q, want %q", phase, PhaseSetupPending)
	}
}

func TestResolveReadyWhenMarkerSet(t *testing.T) {
	runtimeScope, db := newSetupStateScope(t)
	createMetaModuleTable(t, db)
	insertAuthModule(t, db, meta.Installed)
	createAuthAppSettingTable(t, db)
	mustExec(t, db, `INSERT INTO auth_app_setting (id, key, value, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		"s1", authSetupCompletedAtKey, time.Now().UTC().Format(time.RFC3339), time.Now(), time.Now())

	phase, err := Resolve(runtimeScope)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if phase != PhaseReady {
		t.Fatalf("phase = %q, want %q", phase, PhaseReady)
	}
}

func TestIsMissingSQLTableError(t *testing.T) {
	cases := []struct {
		err  error
		want bool
	}{
		{nil, false},
		{errors.New("no such table: auth_app_setting"), true},
		{errors.New("Unknown table 'auth_app_setting'"), true},
		{errors.New("Table 'db.auth_app_setting' doesn't exist"), true},
		{errors.New("Error 1146: Table `db`.`auth_app_setting` doesn't exist"), true},
		{errors.New("Column 'value' in table 'auth_app_setting' doesn't exist"), false},
		{errors.New(`relation "auth_app_setting" does not exist`), true},
		{errors.New(`constraint "fk" of relation "auth_app_setting" does not exist`), false},
		{errors.New("column foo does not exist"), false},
		{errors.New("forced find failure"), false},
	}
	for _, tc := range cases {
		if got := isMissingSQLTableError(tc.err); got != tc.want {
			t.Fatalf("isMissingSQLTableError(%v) = %v, want %v", tc.err, got, tc.want)
		}
	}
}
