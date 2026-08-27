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

func TestApplyExportTemplatePurgeOK(t *testing.T) {
	if err := applyExportTemplatePurge(nil, nil); err != nil {
		t.Fatalf("nil args: %v", err)
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
