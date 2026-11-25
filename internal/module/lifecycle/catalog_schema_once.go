// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"fmt"
	"sync"

	modmeta "github.com/choysum-dev/choysum/internal/module/meta"
	"gorm.io/gorm"
)

// Bridge Plan/Apply rebuilds a ModuleManager per request so company/session binding
// stays correct. Write paths (ensureMetaTables / migrateBaseModule) share this
// process memo so the same sql.DB pool does not remigrate CatalogEntities.
// Plan uses ReadOnlyResolver and skips warm-path AutoMigrate entirely.
//
// Memoization is keyed by the underlying *sql.DB pointer so parallel tests with
// distinct pools do not share state, while a long-lived server pool migrates once.

type catalogSchemaState struct {
	once sync.Once
	err  error
}

var (
	catalogSchemaMu    sync.Mutex
	catalogSchemaByKey = map[string]*catalogSchemaState{}

	// migrateCatalogEntities runs CatalogEntities AutoMigrate; tests may wrap it.
	migrateCatalogEntities = func(db *gorm.DB, entities ...any) error {
		return db.AutoMigrate(entities...)
	}
)

func catalogSchemaKey(db *gorm.DB) string {
	if db == nil {
		return "nil"
	}
	sqlDB, err := db.DB()
	if err != nil || sqlDB == nil {
		return fmt.Sprintf("gorm:%p", db)
	}
	return fmt.Sprintf("sql:%p", sqlDB)
}

func catalogSchemaStateFor(db *gorm.DB) *catalogSchemaState {
	key := catalogSchemaKey(db)
	catalogSchemaMu.Lock()
	defer catalogSchemaMu.Unlock()
	st, ok := catalogSchemaByKey[key]
	if !ok {
		st = &catalogSchemaState{}
		catalogSchemaByKey[key] = st
	}
	return st
}

// ensureCatalogSchemaMigrated AutoMigrates CatalogEntities at most once per
// underlying sql.DB in this process, then always re-checks the effective
// app/name unique index (cheap vs full AutoMigrate; catches live duplicates).
func ensureCatalogSchemaMigrated(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("ensure catalog schema: db is nil")
	}
	st := catalogSchemaStateFor(db)
	st.once.Do(func() {
		if err := migrateCatalogEntities(db, modmeta.CatalogEntities()...); err != nil {
			st.err = fmt.Errorf("auto migrate catalog entities: %w", err)
		}
	})
	if st.err != nil {
		return st.err
	}
	if err := modmeta.EnsureEffectiveAppNameUniqueIndex(db); err != nil {
		return fmt.Errorf("ensure effective app/name unique index: %w", err)
	}
	return nil
}

// rememberCatalogSchemaMigrated records that CatalogEntities were already
// applied (cold bootstrap) so a later ensureCatalogSchemaMigrated is a no-op.
func rememberCatalogSchemaMigrated(db *gorm.DB, err error) {
	if db == nil {
		return
	}
	st := catalogSchemaStateFor(db)
	st.once.Do(func() {
		st.err = err
	})
}

// resetCatalogSchemaMigrateForTest clears process memoization (tests only).
func resetCatalogSchemaMigrateForTest() {
	catalogSchemaMu.Lock()
	defer catalogSchemaMu.Unlock()
	catalogSchemaByKey = map[string]*catalogSchemaState{}
}
