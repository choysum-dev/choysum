// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"database/sql"
	"strings"
	"testing"

	modmeta "github.com/choysum-dev/choysum/internal/module/meta"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/rs/xid"
	"gorm.io/gorm"
)

func ensureWebSavedFilterTable(t *testing.T, db *gorm.DB) {
	t.Helper()
	stmt := `
CREATE TABLE IF NOT EXISTS web_saved_filter (
  id TEXT PRIMARY KEY,
  application TEXT NOT NULL,
  model_name TEXT NOT NULL,
  name TEXT NOT NULL
)`
	if err := db.Exec(stmt).Error; err != nil {
		t.Fatalf("create web_saved_filter: %v", err)
	}
}

func TestModuleUninstallerPurgesSavedFilterWhenLastMetaModelGone(t *testing.T) {
	runtimeScope := newLifecycleCommitTestScope(t)
	db := runtimeScope.Session().DB
	if err := db.AutoMigrate(modmeta.CatalogEntities()...); err != nil {
		t.Fatalf("AutoMigrate CatalogEntities: %v", err)
	}
	ensureWebSavedFilterTable(t, db)

	mod := &meta.Module{Name: "demo_sf", Status: meta.Installed, Version: "1.0.0"}
	mod.Id = sql.NullString{String: xid.New().String(), Valid: true}
	if err := db.Create(mod).Error; err != nil {
		t.Fatalf("create module: %v", err)
	}

	rawID := xid.New().String()
	if _, err := modmeta.ReplaceModuleDeclarations(db, mod.Id.String, []*meta.Model{{
		BaseModel:   meta.BaseModel{Id: sql.NullString{String: rawID, Valid: true}},
		Name:        "Item",
		Path:        "@/demo_sf/service/models/item.ts",
		Application: "demo",
		ModelTable:  "demo_item",
		ModuleId:    mod.Id,
	}}); err != nil {
		t.Fatalf("create raw model: %v", err)
	}
	if err := modmeta.FlushEffective(db, []modmeta.LogicalKey{{Application: "demo", Name: "Item"}}); err != nil {
		t.Fatalf("flush effective: %v", err)
	}

	favID := xid.New().String()
	if err := db.Exec(
		`INSERT INTO web_saved_filter(id, application, model_name, name) VALUES (?, ?, ?, ?)`,
		favID, "demo", "Item", "Active",
	).Error; err != nil {
		t.Fatalf("insert saved filter: %v", err)
	}
	otherID := xid.New().String()
	if err := db.Exec(
		`INSERT INTO web_saved_filter(id, application, model_name, name) VALUES (?, ?, ?, ?)`,
		otherID, "demo", "Other", "Keep",
	).Error; err != nil {
		t.Fatalf("insert other saved filter: %v", err)
	}

	uninstaller := &moduleUninstaller{
		runtimeScope:  runtimeScope,
		module:        mod,
		moduleManager: &ModuleManager{runtimeScope: runtimeScope},
		ctx:           newOpContext(),
	}
	if err := uninstaller.cleanModels(); err != nil {
		t.Fatalf("cleanModels() error = %v", err)
	}

	var remaining int64
	if err := db.Raw(`SELECT COUNT(*) FROM web_saved_filter WHERE id = ?`, favID).Scan(&remaining).Error; err != nil {
		t.Fatalf("count purged filter: %v", err)
	}
	if remaining != 0 {
		t.Fatalf("saved filter for gone model remaining = %d, want 0", remaining)
	}
	var otherRemaining int64
	if err := db.Raw(`SELECT COUNT(*) FROM web_saved_filter WHERE id = ?`, otherID).Scan(&otherRemaining).Error; err != nil {
		t.Fatalf("count other filter: %v", err)
	}
	if otherRemaining != 1 {
		t.Fatalf("unrelated saved filter remaining = %d, want 1", otherRemaining)
	}
}

func TestModuleUninstallerKeepsSavedFilterWhenIMDSurvivorRemains(t *testing.T) {
	runtimeScope := newLifecycleCommitTestScope(t)
	db := runtimeScope.Session().DB
	if err := db.AutoMigrate(modmeta.CatalogEntities()...); err != nil {
		t.Fatalf("AutoMigrate CatalogEntities: %v", err)
	}
	ensureWebSavedFilterTable(t, db)

	baseMod := &meta.Module{Name: "partner", Status: meta.Installed, Version: "1.0.0"}
	baseMod.Id = sql.NullString{String: xid.New().String(), Valid: true}
	if err := db.Create(baseMod).Error; err != nil {
		t.Fatalf("create base module: %v", err)
	}
	extMod := &meta.Module{Name: "partner_commercial", Status: meta.Installed, Version: "1.0.0"}
	extMod.Id = sql.NullString{String: xid.New().String(), Valid: true}
	if err := db.Create(extMod).Error; err != nil {
		t.Fatalf("create ext module: %v", err)
	}

	basePath := "@/partner/service/models/partner.ts"
	if _, err := modmeta.ReplaceModuleDeclarations(db, baseMod.Id.String, []*meta.Model{{
		BaseModel:   meta.BaseModel{Id: sql.NullString{String: xid.New().String(), Valid: true}},
		Name:        "Partner",
		Path:        basePath,
		Application: "partner",
		ModelTable:  "partner_partner",
		ModuleId:    baseMod.Id,
	}}); err != nil {
		t.Fatalf("create base raw model: %v", err)
	}
	if _, err := modmeta.ReplaceModuleDeclarations(db, extMod.Id.String, []*meta.Model{{
		BaseModel:   meta.BaseModel{Id: sql.NullString{String: xid.New().String(), Valid: true}},
		Name:        "Partner",
		Path:        "@/partner_commercial/service/models/partner.ts",
		Application: "partner",
		ModelTable:  "partner_partner",
		ModuleId:    extMod.Id,
		Extends:     basePath,
	}}); err != nil {
		t.Fatalf("create ext raw model: %v", err)
	}
	if err := modmeta.FlushEffective(db, []modmeta.LogicalKey{{Application: "partner", Name: "Partner"}}); err != nil {
		t.Fatalf("flush effective: %v", err)
	}

	favID := xid.New().String()
	if err := db.Exec(
		`INSERT INTO web_saved_filter(id, application, model_name, name) VALUES (?, ?, ?, ?)`,
		favID, "partner", "Partner", "Customers",
	).Error; err != nil {
		t.Fatalf("insert saved filter: %v", err)
	}

	uninstaller := &moduleUninstaller{
		runtimeScope:  runtimeScope,
		module:        extMod,
		moduleManager: &ModuleManager{runtimeScope: runtimeScope},
		ctx:           newOpContext(),
	}
	if err := uninstaller.cleanModels(); err != nil {
		t.Fatalf("cleanModels() error = %v", err)
	}

	var remaining int64
	if err := db.Raw(`SELECT COUNT(*) FROM web_saved_filter WHERE id = ?`, favID).Scan(&remaining).Error; err != nil {
		t.Fatalf("count saved filter: %v", err)
	}
	if remaining != 1 {
		t.Fatalf("IMD survivor should keep saved filter, remaining = %d, want 1", remaining)
	}
}

func TestModuleUninstallerSavedFilterMissingTableNoOp(t *testing.T) {
	runtimeScope := newLifecycleCommitTestScope(t)
	db := runtimeScope.Session().DB
	if err := db.AutoMigrate(modmeta.CatalogEntities()...); err != nil {
		t.Fatalf("AutoMigrate CatalogEntities: %v", err)
	}

	mod := &meta.Module{Name: "demo_sf_missing", Status: meta.Installed, Version: "1.0.0"}
	mod.Id = sql.NullString{String: xid.New().String(), Valid: true}
	if err := db.Create(mod).Error; err != nil {
		t.Fatalf("create module: %v", err)
	}
	if _, err := modmeta.ReplaceModuleDeclarations(db, mod.Id.String, []*meta.Model{{
		BaseModel:   meta.BaseModel{Id: sql.NullString{String: xid.New().String(), Valid: true}},
		Name:        "Item",
		Path:        "@/demo_sf_missing/service/models/item.ts",
		Application: "demo",
		ModelTable:  "demo_item",
		ModuleId:    mod.Id,
	}}); err != nil {
		t.Fatalf("create raw model: %v", err)
	}

	uninstaller := &moduleUninstaller{
		runtimeScope:  runtimeScope,
		module:        mod,
		moduleManager: &ModuleManager{runtimeScope: runtimeScope},
		ctx:           newOpContext(),
	}
	if err := uninstaller.cleanModels(); err != nil {
		t.Fatalf("cleanModels() with missing web_saved_filter should no-op, got %v", err)
	}
}

func TestWebSavedFilterTableExists(t *testing.T) {
	if ok, err := webSavedFilterTableExists(nil); err != nil || ok {
		t.Fatalf("nil db: ok=%v err=%v", ok, err)
	}
	runtimeScope := newLifecycleCommitTestScope(t)
	db := runtimeScope.Session().DB
	if ok, err := webSavedFilterTableExists(db); err != nil || ok {
		t.Fatalf("missing table: ok=%v err=%v", ok, err)
	}
	ensureWebSavedFilterTable(t, db)
	if ok, err := webSavedFilterTableExists(db); err != nil || !ok {
		t.Fatalf("present table: ok=%v err=%v", ok, err)
	}
}

func TestPurgeSavedFiltersForGoneModelsGuards(t *testing.T) {
	if err := purgeSavedFiltersForGoneModels(nil, []modmeta.LogicalKey{{Application: "a", Name: "B"}}); err != nil {
		t.Fatalf("nil db: %v", err)
	}
	runtimeScope := newLifecycleCommitTestScope(t)
	db := runtimeScope.Session().DB
	if err := purgeSavedFiltersForGoneModels(db, nil); err != nil {
		t.Fatalf("empty keys: %v", err)
	}
	ensureWebSavedFilterTable(t, db)
	if err := purgeSavedFiltersForGoneModels(db, []modmeta.LogicalKey{
		{},
		{Application: " ", Name: "Item"},
		{Application: "demo", Name: " "},
		{Application: "demo", Name: "Item"},
		{Application: "demo", Name: "Item"}, // duplicate
	}); err != nil {
		t.Fatalf("invalid/dup keys: %v", err)
	}
}

func TestPurgeSavedFiltersCountError(t *testing.T) {
	runtimeScope := newLifecycleCommitTestScope(t)
	db := runtimeScope.Session().DB
	ensureWebSavedFilterTable(t, db)
	// Keep web_saved_filter visible to HasTable, but break meta_model Count.
	if err := db.Migrator().DropTable(&meta.Model{}); err != nil {
		t.Fatalf("drop meta_model: %v", err)
	}
	err := purgeSavedFiltersForGoneModels(db, []modmeta.LogicalKey{{Application: "demo", Name: "Item"}})
	if err == nil || !strings.Contains(err.Error(), "error counting surviving meta models for saved filter purge") {
		t.Fatalf("error=%v", err)
	}
}

func TestPurgeSavedFiltersDeleteError(t *testing.T) {
	runtimeScope := newLifecycleCommitTestScope(t)
	db := runtimeScope.Session().DB
	if err := db.AutoMigrate(modmeta.CatalogEntities()...); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	ensureWebSavedFilterTable(t, db)
	if err := db.Exec(`INSERT INTO web_saved_filter(id, application, model_name, name) VALUES ('1','demo','Item','x')`).Error; err != nil {
		t.Fatalf("insert: %v", err)
	}
	if err := db.Exec(`CREATE TRIGGER deny_sf_delete BEFORE DELETE ON web_saved_filter BEGIN SELECT RAISE(ABORT, 'deny delete'); END`).Error; err != nil {
		t.Fatalf("create trigger: %v", err)
	}
	err := purgeSavedFiltersForGoneModels(db, []modmeta.LogicalKey{{Application: "demo", Name: "Item"}})
	if err == nil || !strings.Contains(err.Error(), "error deleting web saved filters") {
		t.Fatalf("error=%v, want delete wrap", err)
	}
}

func TestApplySavedFilterPurgePropagatesError(t *testing.T) {
	runtimeScope := newLifecycleCommitTestScope(t)
	db := runtimeScope.Session().DB
	ensureWebSavedFilterTable(t, db)
	if err := db.Migrator().DropTable(&meta.Model{}); err != nil {
		t.Fatalf("drop meta_model: %v", err)
	}
	err := applySavedFilterPurge(db, []modmeta.LogicalKey{{Application: "demo", Name: "Item"}})
	if err == nil || !strings.Contains(err.Error(), "error counting surviving meta models for saved filter purge") {
		t.Fatalf("error=%v", err)
	}
}

func TestApplySavedFilterPurgeOK(t *testing.T) {
	if err := applySavedFilterPurge(nil, nil); err != nil {
		t.Fatalf("nil args: %v", err)
	}
}
