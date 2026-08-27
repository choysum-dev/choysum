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

func ensureWebExportTemplateTable(t *testing.T, db *gorm.DB) {
	t.Helper()
	stmt := `
CREATE TABLE IF NOT EXISTS web_export_template (
  id TEXT PRIMARY KEY,
  application TEXT NOT NULL,
  model_name TEXT NOT NULL,
  name TEXT NOT NULL
)`
	if err := db.Exec(stmt).Error; err != nil {
		t.Fatalf("create web_export_template: %v", err)
	}
}

func TestModuleUninstallerPurgesExportTemplateWhenLastMetaModelGone(t *testing.T) {
	runtimeScope := newLifecycleCommitTestScope(t)
	db := runtimeScope.Session().DB
	if err := db.AutoMigrate(modmeta.CatalogEntities()...); err != nil {
		t.Fatalf("AutoMigrate CatalogEntities: %v", err)
	}
	ensureWebExportTemplateTable(t, db)

	mod := &meta.Module{Name: "demo_et", Status: meta.Installed, Version: "1.0.0"}
	mod.Id = sql.NullString{String: xid.New().String(), Valid: true}
	if err := db.Create(mod).Error; err != nil {
		t.Fatalf("create module: %v", err)
	}

	rawID := xid.New().String()
	if _, err := modmeta.ReplaceModuleDeclarations(db, mod.Id.String, []*meta.Model{{
		BaseModel:   meta.BaseModel{Id: sql.NullString{String: rawID, Valid: true}},
		Name:        "Item",
		Path:        "@/demo_et/service/models/item.ts",
		Application: "demo",
		ModelTable:  "demo_item",
		ModuleId:    mod.Id,
	}}); err != nil {
		t.Fatalf("create raw model: %v", err)
	}
	if err := modmeta.FlushEffective(db, []modmeta.LogicalKey{{Application: "demo", Name: "Item"}}); err != nil {
		t.Fatalf("flush effective: %v", err)
	}

	tplID := xid.New().String()
	if err := db.Exec(
		`INSERT INTO web_export_template(id, application, model_name, name) VALUES (?, ?, ?, ?)`,
		tplID, "demo", "Item", "Basic",
	).Error; err != nil {
		t.Fatalf("insert export template: %v", err)
	}
	otherID := xid.New().String()
	if err := db.Exec(
		`INSERT INTO web_export_template(id, application, model_name, name) VALUES (?, ?, ?, ?)`,
		otherID, "demo", "Other", "Keep",
	).Error; err != nil {
		t.Fatalf("insert other export template: %v", err)
	}

	uninstaller := &moduleUninstaller{
		runtimeScope:  runtimeScope,
		module:        mod,
		moduleManager: &ModuleManager{runtimeScope: runtimeScope},
		ctx:           newOpContext(),
	}
	if err := uninstaller.cleanModels(); err != nil {
		t.Fatalf("cleanModels: %v", err)
	}

	var remaining int64
	if err := db.Raw(`SELECT COUNT(1) FROM web_export_template WHERE id = ?`, tplID).Scan(&remaining).Error; err != nil {
		t.Fatalf("count purged row: %v", err)
	}
	if remaining != 0 {
		t.Fatalf("expected export template for demo.Item to be purged")
	}
	if err := db.Raw(`SELECT COUNT(1) FROM web_export_template WHERE id = ?`, otherID).Scan(&remaining).Error; err != nil {
		t.Fatalf("count kept row: %v", err)
	}
	if remaining != 1 {
		t.Fatalf("expected export template for demo.Other to remain")
	}
}

func TestPurgeExportTemplatesKeepsRowsWhenMetaModelSurvives(t *testing.T) {
	runtimeScope := newLifecycleCommitTestScope(t)
	db := runtimeScope.Session().DB
	if err := db.AutoMigrate(modmeta.CatalogEntities()...); err != nil {
		t.Fatalf("AutoMigrate CatalogEntities: %v", err)
	}
	ensureWebExportTemplateTable(t, db)

	mod := &meta.Module{Name: "demo_et_keep", Status: meta.Installed, Version: "1.0.0"}
	mod.Id = sql.NullString{String: xid.New().String(), Valid: true}
	if err := db.Create(mod).Error; err != nil {
		t.Fatalf("create module: %v", err)
	}
	rawID := xid.New().String()
	if _, err := modmeta.ReplaceModuleDeclarations(db, mod.Id.String, []*meta.Model{{
		BaseModel:   meta.BaseModel{Id: sql.NullString{String: rawID, Valid: true}},
		Name:        "Item",
		Path:        "@/demo_et_keep/service/models/item.ts",
		Application: "demo",
		ModelTable:  "demo_item",
		ModuleId:    mod.Id,
	}}); err != nil {
		t.Fatalf("create raw model: %v", err)
	}
	if err := modmeta.FlushEffective(db, []modmeta.LogicalKey{{Application: "demo", Name: "Item"}}); err != nil {
		t.Fatalf("flush effective: %v", err)
	}
	tplID := xid.New().String()
	if err := db.Exec(
		`INSERT INTO web_export_template(id, application, model_name, name) VALUES (?, ?, ?, ?)`,
		tplID, "demo", "Item", "KeepMe",
	).Error; err != nil {
		t.Fatalf("insert export template: %v", err)
	}

	if err := purgeExportTemplatesForGoneModels(db, []modmeta.LogicalKey{{Application: "demo", Name: "Item"}}); err != nil {
		t.Fatalf("purgeExportTemplatesForGoneModels: %v", err)
	}
	var remaining int64
	if err := db.Raw(`SELECT COUNT(1) FROM web_export_template WHERE id = ?`, tplID).Scan(&remaining).Error; err != nil {
		t.Fatalf("count kept row: %v", err)
	}
	if remaining != 1 {
		t.Fatalf("expected export template to remain while meta model survives")
	}
}

func TestApplyExportTemplatePurgeOK(t *testing.T) {
	if err := applyExportTemplatePurge(nil, nil); err != nil {
		t.Fatalf("nil args: %v", err)
	}
}

func TestApplyExportTemplatePurgeSuccess(t *testing.T) {
	runtimeScope := newLifecycleCommitTestScope(t)
	db := runtimeScope.Session().DB
	ensureWebExportTemplateTable(t, db)
	if err := db.Exec(`INSERT INTO web_export_template(id, application, model_name, name) VALUES ('1','demo','Item','x')`).Error; err != nil {
		t.Fatalf("insert: %v", err)
	}
	if err := applyExportTemplatePurge(db, []modmeta.LogicalKey{{Application: "demo", Name: "Item"}}); err != nil {
		t.Fatalf("applyExportTemplatePurge: %v", err)
	}
	var remaining int64
	if err := db.Raw(`SELECT COUNT(1) FROM web_export_template WHERE id = '1'`).Scan(&remaining).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if remaining != 0 {
		t.Fatalf("expected export template purge to delete row")
	}
}

func TestWebExportTemplateTableExists(t *testing.T) {
	if ok, err := webExportTemplateTableExists(nil); err != nil || ok {
		t.Fatalf("nil db: ok=%v err=%v", ok, err)
	}
	runtimeScope := newLifecycleCommitTestScope(t)
	db := runtimeScope.Session().DB
	if ok, err := webExportTemplateTableExists(db); err != nil || ok {
		t.Fatalf("missing table: ok=%v err=%v", ok, err)
	}
	ensureWebExportTemplateTable(t, db)
	if ok, err := webExportTemplateTableExists(db); err != nil || !ok {
		t.Fatalf("present table: ok=%v err=%v", ok, err)
	}
}

func TestWebExportTemplateTableExistsProbeError(t *testing.T) {
	runtimeScope := newLifecycleCommitTestScope(t)
	db := runtimeScope.Session().DB
	ensureWebExportTemplateTable(t, db)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB(): %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close sql DB: %v", err)
	}

	ok, probeErr := webExportTemplateTableExists(db)
	if ok || probeErr == nil || !strings.Contains(probeErr.Error(), "error checking web_export_template existence") {
		t.Fatalf("ok=%v err=%v, want probe wrap", ok, probeErr)
	}
	if purgeErr := purgeExportTemplatesForGoneModels(db, []modmeta.LogicalKey{{Application: "demo", Name: "Item"}}); purgeErr == nil ||
		!strings.Contains(purgeErr.Error(), "error checking web_export_template existence") {
		t.Fatalf("purge error=%v, want probe failure propagated", purgeErr)
	}
}

func TestPurgeExportTemplatesForGoneModelsGuards(t *testing.T) {
	if err := purgeExportTemplatesForGoneModels(nil, []modmeta.LogicalKey{{Application: "a", Name: "B"}}); err != nil {
		t.Fatalf("nil db: %v", err)
	}
	runtimeScope := newLifecycleCommitTestScope(t)
	db := runtimeScope.Session().DB
	if err := purgeExportTemplatesForGoneModels(db, nil); err != nil {
		t.Fatalf("empty keys: %v", err)
	}
	ensureWebExportTemplateTable(t, db)
	if err := purgeExportTemplatesForGoneModels(db, []modmeta.LogicalKey{
		{},
		{Application: " ", Name: "Item"},
		{Application: "demo", Name: " "},
		{Application: "demo", Name: "Item"},
		{Application: "demo", Name: "Item"},
	}); err != nil {
		t.Fatalf("invalid/dup keys: %v", err)
	}
}

func TestPurgeExportTemplatesDeleteError(t *testing.T) {
	runtimeScope := newLifecycleCommitTestScope(t)
	db := runtimeScope.Session().DB
	if err := db.AutoMigrate(modmeta.CatalogEntities()...); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	ensureWebExportTemplateTable(t, db)
	if err := db.Exec(`INSERT INTO web_export_template(id, application, model_name, name) VALUES ('1','demo','Item','x')`).Error; err != nil {
		t.Fatalf("insert: %v", err)
	}
	if err := db.Exec(`CREATE TRIGGER deny_et_delete BEFORE DELETE ON web_export_template BEGIN SELECT RAISE(ABORT, 'deny delete'); END`).Error; err != nil {
		t.Fatalf("create trigger: %v", err)
	}
	err := purgeExportTemplatesForGoneModels(db, []modmeta.LogicalKey{{Application: "demo", Name: "Item"}})
	if err == nil || !strings.Contains(err.Error(), "error deleting web export templates") {
		t.Fatalf("error=%v, want delete wrap", err)
	}
}

func TestApplyExportTemplatePurgePropagatesError(t *testing.T) {
	runtimeScope := newLifecycleCommitTestScope(t)
	db := runtimeScope.Session().DB
	ensureWebExportTemplateTable(t, db)
	if err := db.Migrator().DropTable(&meta.Model{}); err != nil {
		t.Fatalf("drop meta_model: %v", err)
	}
	err := applyExportTemplatePurge(db, []modmeta.LogicalKey{{Application: "demo", Name: "Item"}})
	if err == nil || !strings.Contains(err.Error(), "error counting surviving meta models for export template purge") {
		t.Fatalf("error=%v", err)
	}
}

func TestModuleUninstallerCleanModelsNoOpWhenExportTemplateTableMissing(t *testing.T) {
	runtimeScope := newLifecycleCommitTestScope(t)
	db := runtimeScope.Session().DB
	if err := db.AutoMigrate(modmeta.CatalogEntities()...); err != nil {
		t.Fatalf("AutoMigrate CatalogEntities: %v", err)
	}

	mod := &meta.Module{Name: "demo_et_missing", Status: meta.Installed, Version: "1.0.0"}
	mod.Id = sql.NullString{String: xid.New().String(), Valid: true}
	if err := db.Create(mod).Error; err != nil {
		t.Fatalf("create module: %v", err)
	}
	if _, err := modmeta.ReplaceModuleDeclarations(db, mod.Id.String, []*meta.Model{{
		BaseModel:   meta.BaseModel{Id: sql.NullString{String: xid.New().String(), Valid: true}},
		Name:        "Item",
		Path:        "@/demo_et_missing/service/models/item.ts",
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
		t.Fatalf("cleanModels() with missing web_export_template should no-op, got %v", err)
	}
}

func TestPurgeExportTemplatesCountError(t *testing.T) {
	runtimeScope := newLifecycleCommitTestScope(t)
	db := runtimeScope.Session().DB
	ensureWebExportTemplateTable(t, db)
	if err := db.Migrator().DropTable(&meta.Model{}); err != nil {
		t.Fatalf("drop meta_model: %v", err)
	}
	err := purgeExportTemplatesForGoneModels(db, []modmeta.LogicalKey{{Application: "demo", Name: "Item"}})
	if err == nil || !strings.Contains(err.Error(), "error counting surviving meta models for export template purge") {
		t.Fatalf("error=%v", err)
	}
}
