// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

import (
	"errors"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func openCutoverDB(t *testing.T, name string) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+name+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB(): %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	return db
}

type stubOpsMigrator struct {
	tables      map[string]bool
	renameErr   error
	dropErr     error
	dropped     []string
	renamedFrom string
	renamedTo   string
}

func (m *stubOpsMigrator) HasTable(dst any) bool {
	switch v := dst.(type) {
	case string:
		return m.tables[v]
	default:
		return false
	}
}

func (m *stubOpsMigrator) RenameTable(src any, dst any) error {
	m.renamedFrom, _ = src.(string)
	m.renamedTo, _ = dst.(string)
	if m.renameErr != nil {
		return m.renameErr
	}
	delete(m.tables, m.renamedFrom)
	m.tables[m.renamedTo] = true
	return nil
}

func (m *stubOpsMigrator) DropTable(dst ...any) error {
	for _, item := range dst {
		name, _ := item.(string)
		m.dropped = append(m.dropped, name)
		if m.dropErr != nil {
			return m.dropErr
		}
		delete(m.tables, name)
	}
	return nil
}

func TestEnsureOpsSchemaCutover_NilDB(t *testing.T) {
	t.Parallel()
	if err := EnsureOpsSchemaCutover(nil); err == nil || !strings.Contains(err.Error(), "db is nil") {
		t.Fatalf("EnsureOpsSchemaCutover(nil) = %v, want db is nil", err)
	}
}

func TestEnsureOpsSchemaCutover_SkipWhenNoLegacy(t *testing.T) {
	t.Parallel()
	db := openCutoverDB(t, "cutover_skip_"+t.Name())
	if err := EnsureOpsSchemaCutover(db); err != nil {
		t.Fatalf("EnsureOpsSchemaCutover(fresh) = %v, want nil", err)
	}
	if db.Migrator().HasTable((ModuleMigration{}).TableName()) {
		t.Fatal("fresh DB short-circuit must not create migration table")
	}
}

func TestEnsureOpsSchemaCutover_ClosedDB(t *testing.T) {
	t.Parallel()
	db := openCutoverDB(t, "cutover_closed_"+t.Name())
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB(): %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	if err := EnsureOpsSchemaCutover(db); err == nil || !strings.Contains(err.Error(), "db unavailable") {
		t.Fatalf("EnsureOpsSchemaCutover(closed) = %v, want db unavailable", err)
	}
}

func TestEnsureOpsSchemaCutover_RenamesMigrationHistory(t *testing.T) {
	t.Parallel()
	db := openCutoverDB(t, "cutover_migration_"+t.Name())

	if err := db.Exec(`CREATE TABLE meta_module_migration_history (
		id TEXT PRIMARY KEY,
		module_name TEXT,
		version TEXT,
		phase TEXT,
		script TEXT,
		checksum TEXT,
		status TEXT
	)`).Error; err != nil {
		t.Fatalf("create legacy table: %v", err)
	}
	if err := db.Exec(`INSERT INTO meta_module_migration_history
		(id, module_name, version, phase, script, checksum, status)
		VALUES ('1', 'meta', '0.1.0', 'pre', 'init', 'abc', 'success')`).Error; err != nil {
		t.Fatalf("seed legacy row: %v", err)
	}

	if err := EnsureOpsSchemaCutover(db); err != nil {
		t.Fatalf("EnsureOpsSchemaCutover: %v", err)
	}

	migrator := db.Migrator()
	if migrator.HasTable(legacyModuleMigrationTable) {
		t.Fatalf("legacy table still present")
	}
	if !migrator.HasTable((ModuleMigration{}).TableName()) {
		t.Fatalf("new migration table missing")
	}

	var status string
	if err := db.Table((ModuleMigration{}).TableName()).
		Select("status").
		Where("module_name = ? AND script = ?", "meta", "init").
		Scan(&status).Error; err != nil {
		t.Fatalf("query renamed row: %v", err)
	}
	if status != "success" {
		t.Fatalf("status = %q, want success", status)
	}

	if err := EnsureOpsSchemaCutover(db); err != nil {
		t.Fatalf("second EnsureOpsSchemaCutover: %v", err)
	}
}

func TestEnsureOpsSchemaCutover_DropsLegacyIndexAndLog(t *testing.T) {
	t.Parallel()
	db := openCutoverDB(t, "cutover_drop_"+t.Name())

	for _, table := range []string{legacyModuleIndexTable, legacyModuleOriginTable, legacyManagementLogTable} {
		if err := db.Exec("CREATE TABLE " + table + " (id TEXT PRIMARY KEY)").Error; err != nil {
			t.Fatalf("create %s: %v", table, err)
		}
	}

	if err := EnsureOpsSchemaCutover(db); err != nil {
		t.Fatalf("EnsureOpsSchemaCutover: %v", err)
	}

	migrator := db.Migrator()
	for _, table := range []string{legacyModuleIndexTable, legacyModuleOriginTable, legacyManagementLogTable} {
		if migrator.HasTable(table) {
			t.Fatalf("expected %s dropped", table)
		}
	}
	if !migrator.HasTable((ModuleSourceSnapshot{}).TableName()) {
		t.Fatal("expected source snapshot table created before legacy drop")
	}
}

func TestEnsureOpsSchemaCutover_DropsLeftoverWhenSnapshotExists(t *testing.T) {
	t.Parallel()
	db := openCutoverDB(t, "cutover_leftover_"+t.Name())
	if err := db.AutoMigrate(&ModuleSourceSnapshot{}); err != nil {
		t.Fatalf("auto migrate snapshot: %v", err)
	}
	if err := db.Exec("CREATE TABLE " + legacyModuleIndexTable + " (id TEXT PRIMARY KEY)").Error; err != nil {
		t.Fatalf("create legacy index: %v", err)
	}

	if err := EnsureOpsSchemaCutover(db); err != nil {
		t.Fatalf("EnsureOpsSchemaCutover: %v", err)
	}
	if db.Migrator().HasTable(legacyModuleIndexTable) {
		t.Fatal("expected leftover legacy index dropped")
	}
}

func TestEnsureOpsSchemaCutover_DropsLeftoverMigrationHistory(t *testing.T) {
	t.Parallel()
	db := openCutoverDB(t, "cutover_both_migration_"+t.Name())
	if err := db.AutoMigrate(&ModuleMigration{}); err != nil {
		t.Fatalf("auto migrate migration: %v", err)
	}
	if err := db.Exec("CREATE TABLE " + legacyModuleMigrationTable + " (id TEXT PRIMARY KEY)").Error; err != nil {
		t.Fatalf("create legacy migration history: %v", err)
	}

	if err := EnsureOpsSchemaCutover(db); err != nil {
		t.Fatalf("EnsureOpsSchemaCutover: %v", err)
	}
	if db.Migrator().HasTable(legacyModuleMigrationTable) {
		t.Fatal("expected leftover legacy migration history dropped")
	}
	if !db.Migrator().HasTable((ModuleMigration{}).TableName()) {
		t.Fatal("expected new migration table retained")
	}
}

func TestEnsureOpsSchemaCutover_StubErrorPaths(t *testing.T) {
	t.Parallel()

	t.Run("nil migrator", func(t *testing.T) {
		err := ensureOpsSchemaCutover(nil, func() error { return nil }, func() error { return nil })
		if err == nil || !strings.Contains(err.Error(), "migrator is nil") {
			t.Fatalf("error = %v, want migrator is nil", err)
		}
	})

	t.Run("rename error", func(t *testing.T) {
		m := &stubOpsMigrator{
			tables:    map[string]bool{legacyModuleMigrationTable: true},
			renameErr: errors.New("rename blocked"),
		}
		err := ensureOpsSchemaCutover(m, func() error { return nil }, func() error { return nil })
		if err == nil || !strings.Contains(err.Error(), "rename blocked") {
			t.Fatalf("error = %v, want rename blocked", err)
		}
	})

	t.Run("auto migrate nil and error", func(t *testing.T) {
		m := &stubOpsMigrator{tables: map[string]bool{legacyModuleMigrationTable: true}}
		err := ensureOpsSchemaCutover(m, nil, func() error { return nil })
		if err == nil || !strings.Contains(err.Error(), "autoMigrate is nil") {
			t.Fatalf("error = %v, want autoMigrate is nil", err)
		}
		m = &stubOpsMigrator{tables: map[string]bool{legacyModuleMigrationTable: true}}
		err = ensureOpsSchemaCutover(m, func() error { return errors.New("migrate blocked") }, func() error { return nil })
		if err == nil || !strings.Contains(err.Error(), "migrate blocked") {
			t.Fatalf("error = %v, want migrate blocked", err)
		}
	})

	t.Run("drop legacy errors", func(t *testing.T) {
		m := &stubOpsMigrator{
			tables:  map[string]bool{legacyModuleIndexTable: true},
			dropErr: errors.New("drop blocked"),
		}
		err := ensureOpsSchemaCutover(m, func() error { return nil }, func() error {
			m.tables[(ModuleSourceSnapshot{}).TableName()] = true
			return nil
		})
		if err == nil || !strings.Contains(err.Error(), "drop legacy") {
			t.Fatalf("error = %v, want drop legacy", err)
		}

		m = &stubOpsMigrator{
			tables: map[string]bool{
				legacyModuleOriginTable:              true,
				(ModuleSourceSnapshot{}).TableName(): true,
			},
			dropErr: errors.New("leftover blocked"),
		}
		err = ensureOpsSchemaCutover(m, func() error { return nil }, func() error { return nil })
		if err == nil || !strings.Contains(err.Error(), "drop legacy") {
			t.Fatalf("error = %v, want drop legacy", err)
		}
	})

	t.Run("nil migrator needsCutover", func(t *testing.T) {
		if opsSchemaNeedsCutover(nil) {
			t.Fatal("opsSchemaNeedsCutover(nil) = true, want false")
		}
	})

	t.Run("drop management log error", func(t *testing.T) {
		m := &stubOpsMigrator{
			tables:  map[string]bool{legacyManagementLogTable: true},
			dropErr: errors.New("log drop blocked"),
		}
		err := ensureOpsSchemaCutover(m, func() error { return nil }, func() error { return nil })
		if err == nil || !strings.Contains(err.Error(), "log drop blocked") {
			t.Fatalf("error = %v, want log drop blocked", err)
		}
	})

	t.Run("auto migrate snapshot nil and error", func(t *testing.T) {
		m := &stubOpsMigrator{tables: map[string]bool{legacyModuleIndexTable: true}}
		err := ensureOpsSchemaCutover(m, func() error { return nil }, nil)
		if err == nil || !strings.Contains(err.Error(), "autoMigrate is nil") {
			t.Fatalf("error = %v, want snapshot autoMigrate is nil", err)
		}
		m = &stubOpsMigrator{tables: map[string]bool{legacyModuleIndexTable: true}}
		err = ensureOpsSchemaCutover(m, func() error { return nil }, func() error {
			return errors.New("snapshot migrate blocked")
		})
		if err == nil || !strings.Contains(err.Error(), "snapshot migrate blocked") {
			t.Fatalf("error = %v, want snapshot migrate blocked", err)
		}
	})

	t.Run("drop leftover migration history error", func(t *testing.T) {
		m := &stubOpsMigrator{
			tables: map[string]bool{
				legacyModuleMigrationTable:      true,
				(ModuleMigration{}).TableName(): true,
			},
			dropErr: errors.New("history drop blocked"),
		}
		err := ensureOpsSchemaCutover(m, func() error { return nil }, func() error { return nil })
		if err == nil || !strings.Contains(err.Error(), "history drop blocked") {
			t.Fatalf("error = %v, want history drop blocked", err)
		}
	})
}
