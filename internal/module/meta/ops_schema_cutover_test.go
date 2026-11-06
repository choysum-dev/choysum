// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

import (
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
	return db
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

	// Idempotent.
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
}
