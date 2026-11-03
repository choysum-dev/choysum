// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func openRetireTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "retire.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	return db
}

func mustExecRetire(t *testing.T, db *gorm.DB, sql string, args ...any) {
	t.Helper()
	if err := db.Exec(sql, args...).Error; err != nil {
		t.Fatalf("exec %q: %v", sql, err)
	}
}

func TestRetireMetaSettingTable_NilDB(t *testing.T) {
	if err := RetireMetaSettingTable(nil); err != nil {
		t.Fatalf("RetireMetaSettingTable(nil): %v", err)
	}
}

func TestRetireMetaSettingTable_AbsentIsNoop(t *testing.T) {
	db := openRetireTestDB(t)
	if err := RetireMetaSettingTable(db); err != nil {
		t.Fatalf("RetireMetaSettingTable: %v", err)
	}
	if db.Migrator().HasTable(retiredMetaSettingTable) {
		t.Fatal("expected meta_setting to remain absent")
	}
}

func TestRetireMetaSettingTable_DropsWithoutAuthWhenNoLegacyMarker(t *testing.T) {
	db := openRetireTestDB(t)
	mustExecRetire(t, db, `CREATE TABLE meta_setting (
		id TEXT PRIMARY KEY,
		key TEXT NOT NULL UNIQUE,
		value TEXT,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	)`)

	if err := RetireMetaSettingTable(db); err != nil {
		t.Fatalf("RetireMetaSettingTable: %v", err)
	}
	if db.Migrator().HasTable(retiredMetaSettingTable) {
		t.Fatal("expected meta_setting to be dropped when nothing to preserve")
	}
}

func TestRetireMetaSettingTable_KeepsLegacyUntilAuthTableExists(t *testing.T) {
	db := openRetireTestDB(t)
	mustExecRetire(t, db, `CREATE TABLE meta_setting (
		id TEXT PRIMARY KEY,
		key TEXT NOT NULL UNIQUE,
		value TEXT,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	)`)
	mustExecRetire(t, db, `INSERT INTO meta_setting (id, key, value, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		"legacy", legacySystemInitDoneKey, "true", time.Now(), time.Now())

	if err := RetireMetaSettingTable(db); err != nil {
		t.Fatalf("RetireMetaSettingTable: %v", err)
	}
	if !db.Migrator().HasTable(retiredMetaSettingTable) {
		t.Fatal("expected meta_setting kept until auth_app_setting exists")
	}

	mustExecRetire(t, db, `CREATE TABLE auth_app_setting (
		id TEXT PRIMARY KEY,
		key TEXT NOT NULL UNIQUE,
		value TEXT,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	)`)
	if err := RetireMetaSettingTable(db); err != nil {
		t.Fatalf("RetireMetaSettingTable after auth table: %v", err)
	}
	if db.Migrator().HasTable(retiredMetaSettingTable) {
		t.Fatal("expected meta_setting dropped after backfill")
	}
	var row struct{ Value string }
	if err := db.Table(authAppSettingTable).Select("value").Where(map[string]any{"key": authSetupCompletedAtKey}).Take(&row).Error; err != nil {
		t.Fatalf("expected backfilled setup.completed_at: %v", err)
	}
	if strings.TrimSpace(row.Value) == "" {
		t.Fatal("expected non-empty backfilled setup.completed_at")
	}
}

func TestRetireMetaSettingTable_BackfillsSetupCompletedAt(t *testing.T) {
	db := openRetireTestDB(t)
	mustExecRetire(t, db, `CREATE TABLE meta_setting (
		id TEXT PRIMARY KEY,
		key TEXT NOT NULL UNIQUE,
		value TEXT,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	)`)
	mustExecRetire(t, db, `CREATE TABLE auth_app_setting (
		id TEXT PRIMARY KEY,
		key TEXT NOT NULL UNIQUE,
		value TEXT,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	)`)
	mustExecRetire(t, db, `INSERT INTO meta_setting (id, key, value, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		"legacy", legacySystemInitDoneKey, "true", time.Now(), time.Now())

	if err := RetireMetaSettingTable(db); err != nil {
		t.Fatalf("RetireMetaSettingTable: %v", err)
	}
	if db.Migrator().HasTable(retiredMetaSettingTable) {
		t.Fatal("expected meta_setting to be dropped")
	}

	var row struct{ Value string }
	if err := db.Table(authAppSettingTable).Select("value").Where(map[string]any{"key": authSetupCompletedAtKey}).Take(&row).Error; err != nil {
		t.Fatalf("expected backfilled setup.completed_at: %v", err)
	}
	if strings.TrimSpace(row.Value) == "" {
		t.Fatal("expected non-empty backfilled setup.completed_at")
	}
}

func TestRetireMetaSettingTable_BlankSetupDeleteError(t *testing.T) {
	db := openRetireTestDB(t)
	mustExecRetire(t, db, `CREATE TABLE meta_setting (
		id TEXT PRIMARY KEY,
		key TEXT NOT NULL UNIQUE,
		value TEXT
	)`)
	mustExecRetire(t, db, `CREATE TABLE auth_app_setting (
		id TEXT PRIMARY KEY,
		key TEXT NOT NULL UNIQUE,
		value TEXT
	)`)
	mustExecRetire(t, db, `INSERT INTO auth_app_setting (id, key, value) VALUES (?, ?, ?)`,
		"blank", authSetupCompletedAtKey, " ")
	mustExecRetire(t, db, `INSERT INTO meta_setting (id, key, value) VALUES (?, ?, ?)`,
		"legacy", legacySystemInitDoneKey, "true")
	forced := errors.New("forced blank-row delete failure")
	registerFailDeletesOnTable(t, db, authAppSettingTable, forced)

	err := RetireMetaSettingTable(db)
	if err == nil || !strings.Contains(err.Error(), "backfill setup.completed_at from legacy meta_setting") {
		t.Fatalf("RetireMetaSettingTable error = %v, want wrapped backfill failure", err)
	}
	if !errors.Is(err, forced) {
		t.Fatalf("RetireMetaSettingTable error = %v, want forced cause", err)
	}
}

func TestRetireMetaSettingTable_KeepsBlankWhenLegacyAbsent(t *testing.T) {
	db := openRetireTestDB(t)
	mustExecRetire(t, db, `CREATE TABLE meta_setting (
		id TEXT PRIMARY KEY,
		key TEXT NOT NULL UNIQUE,
		value TEXT,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	)`)
	mustExecRetire(t, db, `CREATE TABLE auth_app_setting (
		id TEXT PRIMARY KEY,
		key TEXT NOT NULL UNIQUE,
		value TEXT,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	)`)
	blank := "   "
	mustExecRetire(t, db, `INSERT INTO auth_app_setting (id, key, value, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		"blank", authSetupCompletedAtKey, blank, time.Now(), time.Now())

	if err := RetireMetaSettingTable(db); err != nil {
		t.Fatalf("RetireMetaSettingTable: %v", err)
	}
	if db.Migrator().HasTable(retiredMetaSettingTable) {
		t.Fatal("expected meta_setting dropped")
	}
	var row struct{ Value string }
	if err := db.Table(authAppSettingTable).Select("value").Where(map[string]any{"key": authSetupCompletedAtKey}).Take(&row).Error; err != nil {
		t.Fatalf("expected blank setup.completed_at preserved: %v", err)
	}
	if row.Value != blank {
		t.Fatalf("setup.completed_at = %q, want preserved blank %q", row.Value, blank)
	}
}

func TestRetireMetaSettingTable_CompareDeleteSkipsConcurrentMarker(t *testing.T) {
	db := openRetireTestDB(t)
	mustExecRetire(t, db, `CREATE TABLE meta_setting (
		id TEXT PRIMARY KEY,
		key TEXT NOT NULL UNIQUE,
		value TEXT,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	)`)
	mustExecRetire(t, db, `CREATE TABLE auth_app_setting (
		id TEXT PRIMARY KEY,
		key TEXT NOT NULL UNIQUE,
		value TEXT,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	)`)
	mustExecRetire(t, db, `INSERT INTO auth_app_setting (id, key, value, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		"blank", authSetupCompletedAtKey, "   ", time.Now(), time.Now())
	mustExecRetire(t, db, `INSERT INTO meta_setting (id, key, value, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		"legacy", legacySystemInitDoneKey, "true", time.Now(), time.Now())

	peer := "2020-01-02T03:04:05Z"
	const name = "inject_peer_before_blank_delete"
	if err := db.Callback().Delete().Before("gorm:delete").Register(name, func(tx *gorm.DB) {
		if tx.Statement == nil || tx.Statement.Table != authAppSettingTable {
			return
		}
		_ = tx.Session(&gorm.Session{NewDB: true}).Exec(
			`UPDATE auth_app_setting SET value = ? WHERE key = ?`,
			peer, authSetupCompletedAtKey,
		).Error
	}); err != nil {
		t.Fatalf("register delete callback: %v", err)
	}
	t.Cleanup(func() { _ = db.Callback().Delete().Remove(name) })

	if err := RetireMetaSettingTable(db); err != nil {
		t.Fatalf("RetireMetaSettingTable: %v", err)
	}
	var row struct{ Value string }
	if err := db.Table(authAppSettingTable).Select("value").Where(map[string]any{"key": authSetupCompletedAtKey}).Take(&row).Error; err != nil {
		t.Fatalf("read setup.completed_at: %v", err)
	}
	if row.Value != peer {
		t.Fatalf("setup.completed_at = %q, want concurrent peer marker %q", row.Value, peer)
	}
}

func TestRetireMetaSettingTable_ReclaimsBlankSetupCompletedAt(t *testing.T) {
	db := openRetireTestDB(t)
	mustExecRetire(t, db, `CREATE TABLE meta_setting (
		id TEXT PRIMARY KEY,
		key TEXT NOT NULL UNIQUE,
		value TEXT,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	)`)
	mustExecRetire(t, db, `CREATE TABLE auth_app_setting (
		id TEXT PRIMARY KEY,
		key TEXT NOT NULL UNIQUE,
		value TEXT,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	)`)
	mustExecRetire(t, db, `INSERT INTO auth_app_setting (id, key, value, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		"blank", authSetupCompletedAtKey, "   ", time.Now(), time.Now())
	mustExecRetire(t, db, `INSERT INTO meta_setting (id, key, value, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		"legacy", legacySystemInitDoneKey, "true", time.Now(), time.Now())

	if err := RetireMetaSettingTable(db); err != nil {
		t.Fatalf("RetireMetaSettingTable: %v", err)
	}
	var row struct{ Value string }
	if err := db.Table(authAppSettingTable).Select("value").Where(map[string]any{"key": authSetupCompletedAtKey}).Take(&row).Error; err != nil {
		t.Fatalf("expected reclaimed setup.completed_at: %v", err)
	}
	if strings.TrimSpace(row.Value) == "" {
		t.Fatal("expected non-empty setup.completed_at after reclaiming blank row")
	}
}

func TestRetireMetaSettingTable_DoNothingKeepsConcurrentSetupMarker(t *testing.T) {
	db := openRetireTestDB(t)
	mustExecRetire(t, db, `CREATE TABLE meta_setting (
		id TEXT PRIMARY KEY,
		key TEXT NOT NULL UNIQUE,
		value TEXT,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	)`)
	mustExecRetire(t, db, `CREATE TABLE auth_app_setting (
		id TEXT PRIMARY KEY,
		key TEXT NOT NULL UNIQUE,
		value TEXT,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	)`)
	mustExecRetire(t, db, `INSERT INTO meta_setting (id, key, value, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		"legacy", legacySystemInitDoneKey, "true", time.Now(), time.Now())

	peer := "2020-01-02T03:04:05Z"
	const name = "inject_peer_setup_marker"
	if err := db.Callback().Create().Before("gorm:create").Register(name, func(tx *gorm.DB) {
		if tx.Statement == nil || tx.Statement.Table != authAppSettingTable {
			return
		}
		_ = tx.Session(&gorm.Session{NewDB: true}).Exec(
			`INSERT OR IGNORE INTO auth_app_setting (id, key, value, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
			"peer", authSetupCompletedAtKey, peer, time.Now(), time.Now(),
		).Error
	}); err != nil {
		t.Fatalf("register create callback: %v", err)
	}
	t.Cleanup(func() { _ = db.Callback().Create().Remove(name) })

	if err := RetireMetaSettingTable(db); err != nil {
		t.Fatalf("RetireMetaSettingTable: %v", err)
	}
	var row struct{ Value string }
	if err := db.Table(authAppSettingTable).Select("value").Where(map[string]any{"key": authSetupCompletedAtKey}).Take(&row).Error; err != nil {
		t.Fatalf("read setup.completed_at: %v", err)
	}
	if row.Value != peer {
		t.Fatalf("setup.completed_at = %q, want peer marker %q", row.Value, peer)
	}
}

func TestRetireMetaSettingTable_PreservesExistingSetupCompletedAt(t *testing.T) {
	db := openRetireTestDB(t)
	mustExecRetire(t, db, `CREATE TABLE meta_setting (
		id TEXT PRIMARY KEY,
		key TEXT NOT NULL UNIQUE,
		value TEXT,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	)`)
	mustExecRetire(t, db, `CREATE TABLE auth_app_setting (
		id TEXT PRIMARY KEY,
		key TEXT NOT NULL UNIQUE,
		value TEXT,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	)`)
	existing := "2020-01-02T03:04:05Z"
	mustExecRetire(t, db, `INSERT INTO auth_app_setting (id, key, value, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		"seed", authSetupCompletedAtKey, existing, time.Now(), time.Now())
	mustExecRetire(t, db, `INSERT INTO meta_setting (id, key, value, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		"legacy", legacySystemInitDoneKey, "true", time.Now(), time.Now())

	if err := RetireMetaSettingTable(db); err != nil {
		t.Fatalf("RetireMetaSettingTable: %v", err)
	}

	var row struct{ Value string }
	if err := db.Table(authAppSettingTable).Select("value").Where(map[string]any{"key": authSetupCompletedAtKey}).Take(&row).Error; err != nil {
		t.Fatalf("read setup.completed_at: %v", err)
	}
	if row.Value != existing {
		t.Fatalf("setup.completed_at = %q, want preserved %q", row.Value, existing)
	}
}

func TestRetireMetaSettingTable_IgnoresNonTruthyLegacy(t *testing.T) {
	db := openRetireTestDB(t)
	mustExecRetire(t, db, `CREATE TABLE meta_setting (
		id TEXT PRIMARY KEY,
		key TEXT NOT NULL UNIQUE,
		value TEXT,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	)`)
	mustExecRetire(t, db, `CREATE TABLE auth_app_setting (
		id TEXT PRIMARY KEY,
		key TEXT NOT NULL UNIQUE,
		value TEXT,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	)`)
	mustExecRetire(t, db, `INSERT INTO meta_setting (id, key, value, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		"legacy", legacySystemInitDoneKey, "false", time.Now(), time.Now())

	if err := RetireMetaSettingTable(db); err != nil {
		t.Fatalf("RetireMetaSettingTable: %v", err)
	}
	if db.Migrator().HasTable(retiredMetaSettingTable) {
		t.Fatal("expected meta_setting to be dropped")
	}
	var count int64
	if err := db.Table(authAppSettingTable).Where(map[string]any{"key": authSetupCompletedAtKey}).Count(&count).Error; err != nil {
		t.Fatalf("count setup.completed_at: %v", err)
	}
	if count != 0 {
		t.Fatalf("setup.completed_at rows=%d, want 0", count)
	}
}

func TestRetireMetaSettingTable_NoLegacyMarkerIsNoopBackfill(t *testing.T) {
	db := openRetireTestDB(t)
	mustExecRetire(t, db, `CREATE TABLE meta_setting (
		id TEXT PRIMARY KEY,
		key TEXT NOT NULL UNIQUE,
		value TEXT,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	)`)
	mustExecRetire(t, db, `CREATE TABLE auth_app_setting (
		id TEXT PRIMARY KEY,
		key TEXT NOT NULL UNIQUE,
		value TEXT,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	)`)

	if err := RetireMetaSettingTable(db); err != nil {
		t.Fatalf("RetireMetaSettingTable: %v", err)
	}
	if db.Migrator().HasTable(retiredMetaSettingTable) {
		t.Fatal("expected meta_setting to be dropped")
	}
	var count int64
	if err := db.Table(authAppSettingTable).Where(map[string]any{"key": authSetupCompletedAtKey}).Count(&count).Error; err != nil {
		t.Fatalf("count setup.completed_at: %v", err)
	}
	if count != 0 {
		t.Fatalf("setup.completed_at rows=%d, want 0", count)
	}
}

func TestRetireMetaSettingTable_ExistingSetupQueryError(t *testing.T) {
	db := openRetireTestDB(t)
	mustExecRetire(t, db, `CREATE TABLE meta_setting (
		id TEXT PRIMARY KEY,
		key TEXT NOT NULL UNIQUE,
		value TEXT
	)`)
	mustExecRetire(t, db, `CREATE TABLE auth_app_setting (
		id TEXT PRIMARY KEY,
		key TEXT NOT NULL UNIQUE
	)`)
	forced := errors.New("forced auth_app_setting query failure")
	registerFailQueriesOnTable(t, db, authAppSettingTable, forced)

	err := RetireMetaSettingTable(db)
	if err == nil || !strings.Contains(err.Error(), "backfill setup.completed_at from legacy meta_setting") {
		t.Fatalf("RetireMetaSettingTable error = %v, want wrapped backfill failure", err)
	}
	if !errors.Is(err, forced) {
		t.Fatalf("RetireMetaSettingTable error = %v, want forced cause", err)
	}
	if !db.Migrator().HasTable(retiredMetaSettingTable) {
		t.Fatal("expected meta_setting to remain when backfill fails")
	}
}

func TestRetireMetaSettingTable_LegacyQueryError(t *testing.T) {
	db := openRetireTestDB(t)
	mustExecRetire(t, db, `CREATE TABLE meta_setting (
		id TEXT PRIMARY KEY,
		key TEXT NOT NULL UNIQUE
	)`)
	mustExecRetire(t, db, `CREATE TABLE auth_app_setting (
		id TEXT PRIMARY KEY,
		key TEXT NOT NULL UNIQUE,
		value TEXT
	)`)
	forced := errors.New("forced meta_setting query failure")
	registerFailQueriesOnTable(t, db, retiredMetaSettingTable, forced)

	err := RetireMetaSettingTable(db)
	if err == nil || !strings.Contains(err.Error(), "backfill setup.completed_at from legacy meta_setting") {
		t.Fatalf("RetireMetaSettingTable error = %v, want wrapped backfill failure", err)
	}
	if !errors.Is(err, forced) {
		t.Fatalf("RetireMetaSettingTable error = %v, want forced cause", err)
	}
}

func TestRetireMetaSettingTable_LegacyQueryErrorWithoutAuthTable(t *testing.T) {
	db := openRetireTestDB(t)
	mustExecRetire(t, db, `CREATE TABLE meta_setting (
		id TEXT PRIMARY KEY,
		key TEXT NOT NULL UNIQUE
	)`)
	forced := errors.New("forced meta_setting query failure")
	registerFailQueriesOnTable(t, db, retiredMetaSettingTable, forced)

	err := RetireMetaSettingTable(db)
	if err == nil || !strings.Contains(err.Error(), "backfill setup.completed_at from legacy meta_setting") {
		t.Fatalf("RetireMetaSettingTable error = %v, want wrapped backfill failure", err)
	}
	if !errors.Is(err, forced) {
		t.Fatalf("RetireMetaSettingTable error = %v, want forced cause", err)
	}
	if !db.Migrator().HasTable(retiredMetaSettingTable) {
		t.Fatal("expected meta_setting kept when legacy probe fails")
	}
}

func TestRetireMetaSettingTable_BackfillCreateError(t *testing.T) {
	db := openRetireTestDB(t)
	mustExecRetire(t, db, `CREATE TABLE meta_setting (
		id TEXT PRIMARY KEY,
		key TEXT NOT NULL UNIQUE,
		value TEXT
	)`)
	mustExecRetire(t, db, `CREATE TABLE auth_app_setting (
		id TEXT PRIMARY KEY,
		key TEXT NOT NULL UNIQUE,
		value TEXT,
		created_at DATETIME,
		updated_at DATETIME
	)`)
	mustExecRetire(t, db, `INSERT INTO meta_setting (id, key, value) VALUES (?, ?, ?)`,
		"legacy", legacySystemInitDoneKey, "true")
	forced := errors.New("forced auth_app_setting create failure")
	registerFailCreatesOnTable(t, db, authAppSettingTable, forced)

	err := RetireMetaSettingTable(db)
	if err == nil || !strings.Contains(err.Error(), "backfill setup.completed_at from legacy meta_setting") {
		t.Fatalf("RetireMetaSettingTable error = %v, want wrapped backfill failure", err)
	}
	if !errors.Is(err, forced) {
		t.Fatalf("RetireMetaSettingTable error = %v, want forced cause", err)
	}
	if !db.Migrator().HasTable(retiredMetaSettingTable) {
		t.Fatal("expected meta_setting kept when backfill create fails")
	}
}

func TestRetireMetaSettingTable_DropTableError(t *testing.T) {
	db := openRetireTestDB(t)
	mustExecRetire(t, db, `CREATE TABLE meta_setting (
		id TEXT PRIMARY KEY,
		key TEXT NOT NULL UNIQUE,
		value TEXT
	)`)
	forced := errors.New("forced drop failure")
	registerFailDropMetaSetting(t, db, forced)

	err := RetireMetaSettingTable(db)
	if err == nil || !strings.Contains(err.Error(), "drop retired meta_setting") {
		t.Fatalf("RetireMetaSettingTable error = %v, want wrapped drop failure", err)
	}
	if !errors.Is(err, forced) {
		t.Fatalf("RetireMetaSettingTable error = %v, want forced cause", err)
	}
}

func registerFailQueriesOnTable(t *testing.T, db *gorm.DB, table string, fail error) {
	t.Helper()
	name := "fail_query_" + table
	if err := db.Callback().Query().Before("gorm:query").Register(name, func(tx *gorm.DB) {
		if tx.Statement != nil && tx.Statement.Table == table {
			tx.AddError(fail)
		}
	}); err != nil {
		t.Fatalf("register query callback: %v", err)
	}
	t.Cleanup(func() { _ = db.Callback().Query().Remove(name) })
}

func registerFailCreatesOnTable(t *testing.T, db *gorm.DB, table string, fail error) {
	t.Helper()
	name := "fail_create_" + table
	if err := db.Callback().Create().Before("gorm:create").Register(name, func(tx *gorm.DB) {
		if tx.Statement != nil && tx.Statement.Table == table {
			tx.AddError(fail)
		}
	}); err != nil {
		t.Fatalf("register create callback: %v", err)
	}
	t.Cleanup(func() { _ = db.Callback().Create().Remove(name) })
}

func registerFailDeletesOnTable(t *testing.T, db *gorm.DB, table string, fail error) {
	t.Helper()
	name := "fail_delete_" + table
	if err := db.Callback().Delete().Before("gorm:delete").Register(name, func(tx *gorm.DB) {
		if tx.Statement != nil && tx.Statement.Table == table {
			tx.AddError(fail)
		}
	}); err != nil {
		t.Fatalf("register delete callback: %v", err)
	}
	t.Cleanup(func() { _ = db.Callback().Delete().Remove(name) })
}

func registerFailDropMetaSetting(t *testing.T, db *gorm.DB, fail error) {
	t.Helper()
	const name = "fail_drop_meta_setting"
	if err := db.Callback().Raw().Before("gorm:raw").Register(name, func(tx *gorm.DB) {
		sql := ""
		if tx.Statement != nil {
			sql = strings.ToUpper(tx.Statement.SQL.String())
		}
		if strings.Contains(sql, "DROP") && strings.Contains(sql, "META_SETTING") {
			tx.AddError(fail)
		}
	}); err != nil {
		t.Fatalf("register drop callback: %v", err)
	}
	t.Cleanup(func() { _ = db.Callback().Raw().Remove(name) })
}
