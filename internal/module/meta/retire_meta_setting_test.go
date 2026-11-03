// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

import (
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

func TestRetireMetaSettingTable_DropsWithoutAuthTable(t *testing.T) {
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
	if db.Migrator().HasTable(retiredMetaSettingTable) {
		t.Fatal("expected meta_setting to be dropped")
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
