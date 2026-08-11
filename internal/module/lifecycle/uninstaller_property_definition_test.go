// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"database/sql"
	"errors"
	"strings"
	"testing"

	modmeta "github.com/choysum-dev/choysum/internal/module/meta"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/rs/xid"
	"gorm.io/gorm"
)

func ensureDemoPropertyDefinitionTable(t *testing.T, db *gorm.DB) {
	t.Helper()
	stmt := `
CREATE TABLE IF NOT EXISTS demo_property_definition (
  id TEXT PRIMARY KEY,
  target_model TEXT,
  container_model TEXT,
  properties_field TEXT,
  container_id TEXT
)`
	if err := db.Exec(stmt).Error; err != nil {
		t.Fatalf("create demo_property_definition: %v", err)
	}
}

func TestModuleUninstallerPurgesPropertyDefinitionsWhenLastMetaModelGone(t *testing.T) {
	runtimeScope := newLifecycleCommitTestScope(t)
	db := runtimeScope.Session().DB
	if err := db.AutoMigrate(modmeta.CatalogEntities()...); err != nil {
		t.Fatalf("AutoMigrate CatalogEntities: %v", err)
	}
	ensureDemoPropertyDefinitionTable(t, db)

	mod := &meta.Module{Name: "demo_pd", Status: meta.Installed, Version: "1.0.0"}
	mod.Id = sql.NullString{String: xid.New().String(), Valid: true}
	if err := db.Create(mod).Error; err != nil {
		t.Fatalf("create module: %v", err)
	}

	rawID := xid.New().String()
	if _, err := modmeta.ReplaceModuleDeclarations(db, mod.Id.String, []*meta.Model{{
		BaseModel:   meta.BaseModel{Id: sql.NullString{String: rawID, Valid: true}},
		Name:        "Item",
		Path:        "@/demo_pd/service/models/item.ts",
		Application: "demo",
		ModelTable:  "demo_item",
		ModuleId:    mod.Id,
	}}); err != nil {
		t.Fatalf("create raw model: %v", err)
	}
	if err := modmeta.FlushEffective(db, []modmeta.LogicalKey{{Application: "demo", Name: "Item"}}); err != nil {
		t.Fatalf("flush effective: %v", err)
	}

	targetID := xid.New().String()
	if err := db.Exec(
		`INSERT INTO demo_property_definition(id, target_model, container_model, properties_field) VALUES (?, ?, ?, ?)`,
		targetID, "Item", nil, "Properties",
	).Error; err != nil {
		t.Fatalf("insert target definition: %v", err)
	}
	containerID := xid.New().String()
	if err := db.Exec(
		`INSERT INTO demo_property_definition(id, target_model, container_model, properties_field) VALUES (?, ?, ?, ?)`,
		containerID, "Child", "Item", "Properties",
	).Error; err != nil {
		t.Fatalf("insert container definition: %v", err)
	}
	keepID := xid.New().String()
	if err := db.Exec(
		`INSERT INTO demo_property_definition(id, target_model, container_model, properties_field) VALUES (?, ?, ?, ?)`,
		keepID, "Other", "Other", "Properties",
	).Error; err != nil {
		t.Fatalf("insert keep definition: %v", err)
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

	var targetRemaining, containerRemaining, keepRemaining int64
	if err := db.Raw(`SELECT COUNT(*) FROM demo_property_definition WHERE id = ?`, targetID).Scan(&targetRemaining).Error; err != nil {
		t.Fatalf("count target: %v", err)
	}
	if err := db.Raw(`SELECT COUNT(*) FROM demo_property_definition WHERE id = ?`, containerID).Scan(&containerRemaining).Error; err != nil {
		t.Fatalf("count container: %v", err)
	}
	if err := db.Raw(`SELECT COUNT(*) FROM demo_property_definition WHERE id = ?`, keepID).Scan(&keepRemaining).Error; err != nil {
		t.Fatalf("count keep: %v", err)
	}
	if targetRemaining != 0 || containerRemaining != 0 {
		t.Fatalf("gone-model defs remaining target=%d container=%d, want 0", targetRemaining, containerRemaining)
	}
	if keepRemaining != 1 {
		t.Fatalf("unrelated definition remaining = %d, want 1", keepRemaining)
	}
}

func TestModuleUninstallerKeepsPropertyDefinitionsWhenIMDSurvivorRemains(t *testing.T) {
	runtimeScope := newLifecycleCommitTestScope(t)
	db := runtimeScope.Session().DB
	if err := db.AutoMigrate(modmeta.CatalogEntities()...); err != nil {
		t.Fatalf("AutoMigrate CatalogEntities: %v", err)
	}
	ensureDemoPropertyDefinitionTable(t, db)

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

	// Use partner_property_definition for the partner application.
	if err := db.Exec(`
CREATE TABLE IF NOT EXISTS partner_property_definition (
  id TEXT PRIMARY KEY,
  target_model TEXT,
  container_model TEXT,
  properties_field TEXT,
  container_id TEXT
)`).Error; err != nil {
		t.Fatalf("create partner_property_definition: %v", err)
	}
	defID := xid.New().String()
	if err := db.Exec(
		`INSERT INTO partner_property_definition(id, target_model, container_model, properties_field) VALUES (?, ?, ?, ?)`,
		defID, "Partner", nil, "Properties",
	).Error; err != nil {
		t.Fatalf("insert definition: %v", err)
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
	if err := db.Raw(`SELECT COUNT(*) FROM partner_property_definition WHERE id = ?`, defID).Scan(&remaining).Error; err != nil {
		t.Fatalf("count definition: %v", err)
	}
	if remaining != 1 {
		t.Fatalf("IMD survivor should keep definition, remaining = %d, want 1", remaining)
	}
}

func TestModuleUninstallerPropertyDefinitionMissingTableNoOp(t *testing.T) {
	runtimeScope := newLifecycleCommitTestScope(t)
	db := runtimeScope.Session().DB
	if err := db.AutoMigrate(modmeta.CatalogEntities()...); err != nil {
		t.Fatalf("AutoMigrate CatalogEntities: %v", err)
	}

	mod := &meta.Module{Name: "demo_pd_missing", Status: meta.Installed, Version: "1.0.0"}
	mod.Id = sql.NullString{String: xid.New().String(), Valid: true}
	if err := db.Create(mod).Error; err != nil {
		t.Fatalf("create module: %v", err)
	}
	if _, err := modmeta.ReplaceModuleDeclarations(db, mod.Id.String, []*meta.Model{{
		BaseModel:   meta.BaseModel{Id: sql.NullString{String: xid.New().String(), Valid: true}},
		Name:        "Item",
		Path:        "@/demo_pd_missing/service/models/item.ts",
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
		t.Fatalf("cleanModels() with missing property definition table should no-op, got %v", err)
	}
}

func TestPropertyDefinitionTableName(t *testing.T) {
	if got := propertyDefinitionTableName("partner"); got != "partner_property_definition" {
		t.Fatalf("got %q", got)
	}
	if got := propertyDefinitionTableName("MyApp"); got != "my_app_property_definition" {
		t.Fatalf("got %q", got)
	}
}

func TestIsMissingSQLTableError(t *testing.T) {
	if isMissingSQLTableError(nil) {
		t.Fatal("nil must be false")
	}
	for _, msg := range []string{
		"no such table: demo_property_definition",
		"Table 'db.demo' doesn't exist",
		"relation \"demo\" does not exist",
		"Unknown table 'demo'",
	} {
		if !isMissingSQLTableError(errors.New(msg)) {
			t.Fatalf("expected missing-table for %q", msg)
		}
	}
	if isMissingSQLTableError(errors.New("database is locked")) {
		t.Fatal("unrelated errors must be false")
	}
}

func TestPropertyDefinitionTableExists(t *testing.T) {
	if ok, err := propertyDefinitionTableExists(nil, "demo_property_definition"); err != nil || ok {
		t.Fatalf("nil db: ok=%v err=%v", ok, err)
	}
	if ok, err := propertyDefinitionTableExists(nil, ""); err != nil || ok {
		t.Fatalf("empty table: ok=%v err=%v", ok, err)
	}
	runtimeScope := newLifecycleCommitTestScope(t)
	db := runtimeScope.Session().DB
	if ok, err := propertyDefinitionTableExists(db, "demo_property_definition"); err != nil || ok {
		t.Fatalf("missing table: ok=%v err=%v", ok, err)
	}
	ensureDemoPropertyDefinitionTable(t, db)
	if ok, err := propertyDefinitionTableExists(db, "demo_property_definition"); err != nil || !ok {
		t.Fatalf("present table: ok=%v err=%v", ok, err)
	}
}

func TestPropertyDefinitionTableExistsProbeError(t *testing.T) {
	runtimeScope := newLifecycleCommitTestScope(t)
	db := runtimeScope.Session().DB
	ensureDemoPropertyDefinitionTable(t, db)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB(): %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close sql DB: %v", err)
	}

	ok, probeErr := propertyDefinitionTableExists(db, "demo_property_definition")
	if ok || probeErr == nil || !strings.Contains(probeErr.Error(), "error checking demo_property_definition existence") {
		t.Fatalf("ok=%v err=%v, want probe wrap", ok, probeErr)
	}
	if purgeErr := purgePropertyDefinitionsForGoneModels(db, []modmeta.LogicalKey{{Application: "demo", Name: "Item"}}); purgeErr == nil ||
		!strings.Contains(purgeErr.Error(), "error checking demo_property_definition existence") {
		t.Fatalf("purge error=%v, want probe failure propagated", purgeErr)
	}
}

func TestPurgePropertyDefinitionsForGoneModelsGuards(t *testing.T) {
	if err := purgePropertyDefinitionsForGoneModels(nil, []modmeta.LogicalKey{{Application: "a", Name: "B"}}); err != nil {
		t.Fatalf("nil db: %v", err)
	}
	runtimeScope := newLifecycleCommitTestScope(t)
	db := runtimeScope.Session().DB
	if err := purgePropertyDefinitionsForGoneModels(db, nil); err != nil {
		t.Fatalf("empty keys: %v", err)
	}
	ensureDemoPropertyDefinitionTable(t, db)
	if err := purgePropertyDefinitionsForGoneModels(db, []modmeta.LogicalKey{
		{},
		{Application: " ", Name: "Item"},
		{Application: "demo", Name: " "},
		{Application: "demo", Name: "Item"},
		{Application: "demo", Name: "Item"}, // duplicate
	}); err != nil {
		t.Fatalf("invalid/dup keys: %v", err)
	}
}

func TestPurgePropertyDefinitionsCountError(t *testing.T) {
	runtimeScope := newLifecycleCommitTestScope(t)
	db := runtimeScope.Session().DB
	ensureDemoPropertyDefinitionTable(t, db)
	if err := db.Migrator().DropTable(&meta.Model{}); err != nil {
		t.Fatalf("drop meta_model: %v", err)
	}
	err := purgePropertyDefinitionsForGoneModels(db, []modmeta.LogicalKey{{Application: "demo", Name: "Item"}})
	if err == nil || !strings.Contains(err.Error(), "error counting surviving meta models for property definition purge") {
		t.Fatalf("error=%v", err)
	}
}

func TestPurgePropertyDefinitionsDeleteError(t *testing.T) {
	runtimeScope := newLifecycleCommitTestScope(t)
	db := runtimeScope.Session().DB
	if err := db.AutoMigrate(modmeta.CatalogEntities()...); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	ensureDemoPropertyDefinitionTable(t, db)
	if err := db.Exec(`INSERT INTO demo_property_definition(id, target_model, container_model, properties_field) VALUES ('1','Item',NULL,'Properties')`).Error; err != nil {
		t.Fatalf("insert: %v", err)
	}
	if err := db.Exec(`CREATE TRIGGER deny_pd_delete BEFORE DELETE ON demo_property_definition BEGIN SELECT RAISE(ABORT, 'deny delete'); END`).Error; err != nil {
		t.Fatalf("create trigger: %v", err)
	}
	err := purgePropertyDefinitionsForGoneModels(db, []modmeta.LogicalKey{{Application: "demo", Name: "Item"}})
	if err == nil || !strings.Contains(err.Error(), "error deleting property definitions") {
		t.Fatalf("error=%v, want delete wrap", err)
	}
}

func TestApplyPropertyDefinitionPurgePropagatesError(t *testing.T) {
	runtimeScope := newLifecycleCommitTestScope(t)
	db := runtimeScope.Session().DB
	ensureDemoPropertyDefinitionTable(t, db)
	if err := db.Migrator().DropTable(&meta.Model{}); err != nil {
		t.Fatalf("drop meta_model: %v", err)
	}
	err := applyPropertyDefinitionPurge(db, []modmeta.LogicalKey{{Application: "demo", Name: "Item"}})
	if err == nil || !strings.Contains(err.Error(), "error counting surviving meta models for property definition purge") {
		t.Fatalf("error=%v", err)
	}
}

func TestApplyPropertyDefinitionPurgeOK(t *testing.T) {
	if err := applyPropertyDefinitionPurge(nil, nil); err != nil {
		t.Fatalf("nil args: %v", err)
	}
}

func TestModuleUninstallerCleanModelsPropagatesPropertyDefinitionPurgeError(t *testing.T) {
	runtimeScope := newLifecycleCommitTestScope(t)
	db := runtimeScope.Session().DB
	if err := db.AutoMigrate(modmeta.CatalogEntities()...); err != nil {
		t.Fatalf("AutoMigrate CatalogEntities: %v", err)
	}
	ensureDemoPropertyDefinitionTable(t, db)

	mod := &meta.Module{Name: "demo_pd_purge_err", Status: meta.Installed, Version: "1.0.0"}
	mod.Id = sql.NullString{String: xid.New().String(), Valid: true}
	if err := db.Create(mod).Error; err != nil {
		t.Fatalf("create module: %v", err)
	}
	if _, err := modmeta.ReplaceModuleDeclarations(db, mod.Id.String, []*meta.Model{{
		BaseModel:   meta.BaseModel{Id: sql.NullString{String: xid.New().String(), Valid: true}},
		Name:        "Item",
		Path:        "@/demo_pd_purge_err/service/models/item.ts",
		Application: "demo",
		ModelTable:  "demo_item",
		ModuleId:    mod.Id,
	}}); err != nil {
		t.Fatalf("create raw model: %v", err)
	}
	if err := modmeta.FlushEffective(db, []modmeta.LogicalKey{{Application: "demo", Name: "Item"}}); err != nil {
		t.Fatalf("flush effective: %v", err)
	}
	if err := db.Exec(
		`INSERT INTO demo_property_definition(id, target_model, container_model, properties_field) VALUES (?, ?, ?, ?)`,
		xid.New().String(), "Item", nil, "Properties",
	).Error; err != nil {
		t.Fatalf("insert definition: %v", err)
	}
	if err := db.Exec(`CREATE TRIGGER deny_pd_clean_delete BEFORE DELETE ON demo_property_definition BEGIN SELECT RAISE(ABORT, 'deny delete'); END`).Error; err != nil {
		t.Fatalf("create trigger: %v", err)
	}

	uninstaller := &moduleUninstaller{
		runtimeScope:  runtimeScope,
		module:        mod,
		moduleManager: &ModuleManager{runtimeScope: runtimeScope},
		ctx:           newOpContext(),
	}
	err := uninstaller.cleanModels()
	if err == nil || !strings.Contains(err.Error(), "error deleting property definitions") {
		t.Fatalf("cleanModels() error=%v, want property definition purge failure", err)
	}
}
