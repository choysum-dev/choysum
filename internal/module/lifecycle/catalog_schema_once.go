// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"fmt"
	"sync"

	modmeta "github.com/choysum-dev/choysum/internal/module/meta"
	"github.com/choysum-dev/choysum/pkg/meta"
	"gorm.io/gorm"
)

// Bridge Plan/Apply rebuilds a ModuleManager per request so company/session binding
// stays correct. Write paths (ensureMetaTables / migrateBaseModule) share this
// process memo so the same sql.DB pool does not remigrate CatalogEntities.
// Plan uses ReadOnlyResolver and skips warm-path AutoMigrate entirely.
//
// Memoization is keyed by the underlying *sql.DB pointer so parallel tests with
// distinct pools do not share state, while a long-lived server pool migrates once.
// Only successful migrates are memoized; failures retry. Before trusting a hit,
// HasTable(meta_module_operation) is checked so a recycled *sql.DB pointer cannot
// skip migrate on a fresh empty pool. Per-pool mu serializes check+migrate so
// concurrent cold callers do not race AutoMigrate.

type catalogSchemaState struct {
	mu   sync.Mutex
	done bool
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
	// Empty / unopened *gorm.DB has no Config; db.DB() would panic.
	if db.Config == nil {
		return fmt.Sprintf("gorm:%p", db)
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
// underlying sql.DB in this process (success only), then re-checks the effective
// app/name unique index when meta_model exists.
func ensureCatalogSchemaMigrated(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("ensure catalog schema: db is nil")
	}
	st := catalogSchemaStateFor(db)
	st.mu.Lock()
	defer st.mu.Unlock()

	if st.done {
		// Pointer-keyed memo can outlive a closed pool; refuse a hit when the
		// sentinel ops table is missing on this DB.
		if db.Migrator().HasTable(&modmeta.ModuleOperation{}) {
			return ensureEffectiveAppNameIndexIfModelExists(db)
		}
		st.done = false
	}

	if err := migrateCatalogEntities(db, modmeta.CatalogEntities()...); err != nil {
		return fmt.Errorf("auto migrate catalog entities: %w", err)
	}
	if err := ensureEffectiveAppNameIndexIfModelExists(db); err != nil {
		return err
	}

	st.done = true
	return nil
}

func ensureEffectiveAppNameIndexIfModelExists(db *gorm.DB) error {
	if !db.Migrator().HasTable(&meta.Model{}) {
		return nil
	}
	if err := modmeta.EnsureEffectiveAppNameUniqueIndex(db); err != nil {
		return fmt.Errorf("ensure effective app/name unique index: %w", err)
	}
	return nil
}

// rememberCatalogSchemaMigrated records a successful CatalogEntities migrate so
// a later ensureCatalogSchemaMigrated is a no-op. Failures are not recorded.
func rememberCatalogSchemaMigrated(db *gorm.DB, err error) {
	if db == nil || err != nil {
		return
	}
	st := catalogSchemaStateFor(db)
	st.mu.Lock()
	st.done = true
	st.mu.Unlock()
}

// resetCatalogSchemaMigrateForTest clears process memoization (tests only).
func resetCatalogSchemaMigrateForTest() {
	catalogSchemaMu.Lock()
	defer catalogSchemaMu.Unlock()
	catalogSchemaByKey = map[string]*catalogSchemaState{}
}
