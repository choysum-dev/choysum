// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

import (
	"fmt"

	"gorm.io/gorm"
)

const (
	legacyModuleMigrationTable = "meta_module_migration_history"
	legacyModuleIndexTable     = "meta_module_index"
	legacyModuleOriginTable    = "meta_module_origin"
	legacyManagementLogTable   = "meta_module_management_log"
)

type opsSchemaMigrator interface {
	HasTable(dst any) bool
	RenameTable(src any, dst any) error
	DropTable(dst ...any) error
}

// EnsureOpsSchemaCutover applies one-shot table renames/drops for the W1 ops
// schema cut before AutoMigrate. Migration history is renamed so retries do not
// re-run scripts. Index/origin snapshot tables are dropped (resync fills
// meta_module_source_snapshot). Management log is dropped (history moves to
// meta_module_operation in later work).
func EnsureOpsSchemaCutover(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("EnsureOpsSchemaCutover: db is nil")
	}
	if err := db.Exec("SELECT 1").Error; err != nil {
		return fmt.Errorf("EnsureOpsSchemaCutover: db unavailable: %w", err)
	}
	return ensureOpsSchemaCutover(db.Migrator(), func() error {
		return db.AutoMigrate(&ModuleMigration{})
	})
}

func ensureOpsSchemaCutover(migrator opsSchemaMigrator, autoMigrateMigration func() error) error {
	if migrator == nil {
		return fmt.Errorf("EnsureOpsSchemaCutover: migrator is nil")
	}

	newMigration := (ModuleMigration{}).TableName()
	if migrator.HasTable(legacyModuleMigrationTable) && !migrator.HasTable(newMigration) {
		if err := migrator.RenameTable(legacyModuleMigrationTable, newMigration); err != nil {
			return fmt.Errorf("rename %s -> %s: %w", legacyModuleMigrationTable, newMigration, err)
		}
	}
	// Align columns (e.g. soft-delete) after rename from a leaner legacy shape.
	if migrator.HasTable(newMigration) {
		if autoMigrateMigration == nil {
			return fmt.Errorf("auto migrate %s: autoMigrate is nil", newMigration)
		}
		if err := autoMigrateMigration(); err != nil {
			return fmt.Errorf("auto migrate %s: %w", newMigration, err)
		}
	}

	newSnapshot := (ModuleSourceSnapshot{}).TableName()
	for _, legacy := range []string{legacyModuleIndexTable, legacyModuleOriginTable} {
		if migrator.HasTable(legacy) && !migrator.HasTable(newSnapshot) {
			if err := migrator.DropTable(legacy); err != nil {
				return fmt.Errorf("drop legacy %s: %w", legacy, err)
			}
		} else if migrator.HasTable(legacy) && migrator.HasTable(newSnapshot) && legacy != newSnapshot {
			if err := migrator.DropTable(legacy); err != nil {
				return fmt.Errorf("drop leftover %s: %w", legacy, err)
			}
		}
	}

	if migrator.HasTable(legacyManagementLogTable) {
		if err := migrator.DropTable(legacyManagementLogTable); err != nil {
			return fmt.Errorf("drop %s: %w", legacyManagementLogTable, err)
		}
	}
	return nil
}
