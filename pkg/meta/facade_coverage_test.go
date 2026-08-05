// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func openFacadeDB(t *testing.T, name string) *gorm.DB {
	t.Helper()
	dsn := filepath.Join(t.TempDir(), name+".sqlite")
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err := EnsureDualStoreTables(db); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestRawFacade_CountAndDrop(t *testing.T) {
	if _, err := CountRawModelsByID(nil, "x", false); err == nil || !strings.Contains(err.Error(), "db is nil") {
		t.Fatalf("nil db: %v", err)
	}
	if err := DropRawModelTable(nil); err == nil || !strings.Contains(err.Error(), "db is nil") {
		t.Fatalf("nil drop: %v", err)
	}

	db := openFacadeDB(t, "raw-facade")
	m := &Model{
		BaseModel:   BaseModel{Id: sql.NullString{String: "raw-1", Valid: true}},
		Name:        "X",
		Path:        "/x.ts",
		Application: "app",
	}
	if err := PersistModelTreeAsRaw(db, m); err != nil {
		t.Fatal(err)
	}
	n, err := CountRawModelsByID(db, "raw-1", false)
	if err != nil || n != 1 {
		t.Fatalf("count=%d err=%v", n, err)
	}
	n, err = CountRawModelsByID(db, "  raw-1  ", true)
	if err != nil || n != 1 {
		t.Fatalf("unscoped count=%d err=%v", n, err)
	}
	if err := DropRawModelTable(db); err != nil {
		t.Fatal(err)
	}
	if db.Migrator().HasTable(RawModelTable) {
		t.Fatal("expected table dropped")
	}
}

func TestEnsureAbstractModel_UpsertErrorPath(t *testing.T) {
	db := openFacadeDB(t, "ensure-upsert-err")
	if err := DropRawModelTable(db); err != nil {
		t.Fatal(err)
	}
	if err := EnsureAbstractModel(db, AbstractModelSpec{
		Name: "I18n", Path: "go://i18n", Application: "auth",
	}); err == nil {
		t.Fatal("expected upsert error after drop")
	}
}

func TestEnsureAbstractModel_PreferErrorPath(t *testing.T) {
	db := openFacadeDB(t, "prefer-fail-ensure")
	// PreferDeclarationTip finishes with UpdateColumns; UpsertDeclaration may update too.
	// Seed via Upsert first, then enable the failing update callback and call Ensure again
	// so Upsert takes the existing-row path and Prefer's tip bump hits the callback.
	if err := EnsureAbstractModel(db, AbstractModelSpec{
		Name: "I18n", Path: "go://i18n/prefer-fail", Application: "auth",
	}); err != nil {
		t.Fatal(err)
	}
	if err := db.Callback().Update().Before("gorm:update").Register("test_fail_prefer_update", func(tx *gorm.DB) {
		_ = tx.AddError(fmt.Errorf("forced prefer update failure"))
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = db.Callback().Update().Remove("test_fail_prefer_update")
	})
	if err := EnsureAbstractModel(db, AbstractModelSpec{
		Name: "I18n", Path: "go://i18n/prefer-fail", Application: "auth",
	}); err == nil {
		t.Fatal("expected PreferDeclarationTip update error")
	}
}

func TestReplaceModuleDeclarations_GuardsAndErrors(t *testing.T) {
	keys, err := ReplaceModuleDeclarations(nil, "  ", nil)
	if err != nil || keys != nil {
		t.Fatalf("empty moduleID: %v %#v", err, keys)
	}
	if _, err := ReplaceModuleDeclarations(nil, "mod", nil); err == nil || !strings.Contains(err.Error(), "db is nil") {
		t.Fatalf("nil db: %v", err)
	}

	db := openFacadeDB(t, "replace-guards")
	keys, err = ReplaceModuleDeclarations(db, "mod-1", []*Model{
		nil,
		{Name: "NoPath", Path: "  ", Application: "a"},
		{Name: "A", Path: "/a.ts", Application: "a"},
		{Name: "A2", Path: "/a.ts", Application: "a"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) == 0 {
		t.Fatal("expected keys")
	}

	if err := DropRawModelTable(db); err != nil {
		t.Fatal(err)
	}
	if _, err := ReplaceModuleDeclarations(db, "mod-1", nil); err == nil {
		t.Fatal("expected list previous raw error")
	}

	db2 := openFacadeDB(t, "replace-eff-err")
	if err := db2.Migrator().DropTable(&Model{}); err != nil {
		t.Fatal(err)
	}
	if _, err := ReplaceModuleDeclarations(db2, "mod-2", nil); err == nil {
		t.Fatal("expected list previous effective error")
	}

	db3 := openFacadeDB(t, "replace-del-err")
	if err := PersistModelTreeAsRaw(db3, &Model{
		BaseModel:   BaseModel{Id: sql.NullString{String: "p1", Valid: true}},
		Name:        "P",
		Path:        "/p.ts",
		Application: "a",
		ModuleId:    sql.NullString{String: "mod-3", Valid: true},
	}); err != nil {
		t.Fatal(err)
	}
	if err := db3.Migrator().DropTable(RawServiceTable); err != nil {
		t.Fatal(err)
	}
	if _, err := ReplaceModuleDeclarations(db3, "mod-3", []*Model{
		{Name: "N", Path: "/n.ts", Application: "a"},
	}); err == nil {
		t.Fatal("expected delete previous raw error")
	}
}

func TestReplaceModuleDeclarations_PrevEffectiveKeys(t *testing.T) {
	db := openFacadeDB(t, "replace-eff-keys")
	modID := "mod-eff"
	eff := Model{
		BaseModel:   BaseModel{Id: sql.NullString{String: "eff-1", Valid: true}},
		Name:        "OldEff",
		Path:        "/old.ts",
		Application: "partner",
		ModuleId:    sql.NullString{String: modID, Valid: true},
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(&eff).Error; err != nil {
		t.Fatal(err)
	}
	keys, err := ReplaceModuleDeclarations(db, modID, []*Model{
		{Name: "New", Path: "/new.ts", Application: "partner"},
	})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, k := range keys {
		if k.Name == "OldEff" && k.Application == "partner" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected previous effective key, got %#v", keys)
	}
}

func TestReplaceModuleDeclarations_NilPrevDeclRow(t *testing.T) {
	db := openFacadeDB(t, "replace-nil-row")
	// Force ListDeclarations to return a nil element via soft path: not possible.
	// Cover appendKey invalid key (empty application/name) from prev effective zero Model.
	modID := "mod-zero"
	eff := Model{ModuleId: sql.NullString{String: modID, Valid: true}}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(&eff).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := ReplaceModuleDeclarations(db, modID, nil); err != nil {
		t.Fatal(err)
	}
}
