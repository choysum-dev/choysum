// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

import (
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	pkgmeta "github.com/choysum-dev/choysum/pkg/meta"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func openDeclarationTestDB(t *testing.T, name string) *gorm.DB {
	t.Helper()
	dsn := filepath.Join(t.TempDir(), name+".sqlite")
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err := ensureDualStoreTables(db); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestReplaceModuleDeclarations_GuardsAndErrors(t *testing.T) {
	keys, err := ReplaceModuleDeclarations(nil, "  ", nil)
	if err != nil || keys != nil {
		t.Fatalf("empty moduleID: %v %#v", err, keys)
	}
	if _, err := ReplaceModuleDeclarations(nil, "mod", nil); err == nil || !strings.Contains(err.Error(), "db is nil") {
		t.Fatalf("nil db: %v", err)
	}

	db := openDeclarationTestDB(t, "replace-guards")
	keys, err = ReplaceModuleDeclarations(db, "mod-1", []*pkgmeta.Model{
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

	if err := dropRawModelTable(db); err != nil {
		t.Fatal(err)
	}
	if _, err := ReplaceModuleDeclarations(db, "mod-1", nil); err == nil {
		t.Fatal("expected list previous raw error")
	}

	db2 := openDeclarationTestDB(t, "replace-eff-err")
	if err := db2.Migrator().DropTable(&pkgmeta.Model{}); err != nil {
		t.Fatal(err)
	}
	if _, err := ReplaceModuleDeclarations(db2, "mod-2", nil); err == nil {
		t.Fatal("expected list previous effective error")
	}

	db3 := openDeclarationTestDB(t, "replace-del-err")
	if err := persistModelTreeAsRaw(db3, &pkgmeta.Model{
		BaseModel:   pkgmeta.BaseModel{Id: sql.NullString{String: "p1", Valid: true}},
		Name:        "P",
		Path:        "/p.ts",
		Application: "a",
		ModuleId:    sql.NullString{String: "mod-3", Valid: true},
	}); err != nil {
		t.Fatal(err)
	}
	if err := db3.Migrator().DropTable(rawServiceTable); err != nil {
		t.Fatal(err)
	}
	if _, err := ReplaceModuleDeclarations(db3, "mod-3", []*pkgmeta.Model{
		{Name: "N", Path: "/n.ts", Application: "a"},
	}); err == nil {
		t.Fatal("expected delete previous raw error")
	}
}

func TestReplaceModuleDeclarations_PrevEffectiveKeys(t *testing.T) {
	db := openDeclarationTestDB(t, "replace-eff-keys")
	modID := "mod-eff"
	eff := pkgmeta.Model{
		BaseModel:   pkgmeta.BaseModel{Id: sql.NullString{String: "eff-1", Valid: true}},
		Name:        "OldEff",
		Path:        "/old.ts",
		Application: "partner",
		ModuleId:    sql.NullString{String: modID, Valid: true},
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(&eff).Error; err != nil {
		t.Fatal(err)
	}
	keys, err := ReplaceModuleDeclarations(db, modID, []*pkgmeta.Model{
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
	db := openDeclarationTestDB(t, "replace-nil-row")
	// Cover appendKey invalid key (empty application/name) from prev effective zero pkgmeta.Model.
	modID := "mod-zero"
	eff := pkgmeta.Model{ModuleId: sql.NullString{String: modID, Valid: true}}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(&eff).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := ReplaceModuleDeclarations(db, modID, nil); err != nil {
		t.Fatal(err)
	}
}

func TestListDeclarations_NilDBAndPathFilter(t *testing.T) {
	if _, err := ListDeclarations(nil, DeclarationQuery{}); err == nil || !strings.Contains(err.Error(), "db is nil") {
		t.Fatalf("nil db: %v", err)
	}

	db := openDeclarationTestDB(t, "list-path")
	modID := "mod-path"
	if err := persistModelTreeAsRaw(db, &pkgmeta.Model{
		Name: "A", Path: "/only.ts", Application: "crm",
		ModuleId: sql.NullString{String: modID, Valid: true},
	}); err != nil {
		t.Fatal(err)
	}
	got, err := ListDeclarations(db, DeclarationQuery{Path: "/only.ts"})
	if err != nil || len(got) != 1 || got[0].Path != "/only.ts" {
		t.Fatalf("path filter: n=%d err=%v got=%#v", len(got), err, got)
	}
	miss, err := ListDeclarations(db, DeclarationQuery{Path: "/missing.ts"})
	if err != nil || len(miss) != 0 {
		t.Fatalf("missing path: n=%d err=%v", len(miss), err)
	}
}

func TestRemoveModuleDeclarations_GuardsAndErrors(t *testing.T) {
	db := openDeclarationTestDB(t, "remove-guards")
	modID := "mod-rm"
	if err := persistModelTreeAsRaw(db, &pkgmeta.Model{
		Name: "Dup", Path: "/a.ts", Application: "crm",
		ModuleId: sql.NullString{String: modID, Valid: true},
	}); err != nil {
		t.Fatal(err)
	}
	if err := persistModelTreeAsRaw(db, &pkgmeta.Model{
		Name: "Dup", Path: "/b.ts", Application: "crm",
		ModuleId: sql.NullString{String: modID, Valid: true},
	}); err != nil {
		t.Fatal(err)
	}
	// Invalid key from effective shell with empty application/name is skipped.
	eff := pkgmeta.Model{ModuleId: sql.NullString{String: modID, Valid: true}}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(&eff).Error; err != nil {
		t.Fatal(err)
	}
	keys, err := RemoveModuleDeclarations(db, modID)
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 1 || keys[0].Application != "crm" || keys[0].Name != "Dup" {
		t.Fatalf("expected deduped key, got %#v", keys)
	}

	db2 := openDeclarationTestDB(t, "remove-list-err")
	if err := dropRawModelTable(db2); err != nil {
		t.Fatal(err)
	}
	if _, err := RemoveModuleDeclarations(db2, "mod-x"); err == nil || !strings.Contains(err.Error(), "list previous raw models") {
		t.Fatalf("expected list previous raw error, got %v", err)
	}

	db3 := openDeclarationTestDB(t, "remove-eff-err")
	if err := db3.Migrator().DropTable(&pkgmeta.Model{}); err != nil {
		t.Fatal(err)
	}
	if _, err := RemoveModuleDeclarations(db3, "mod-y"); err == nil || !strings.Contains(err.Error(), "list previous effective models") {
		t.Fatalf("expected list previous effective error, got %v", err)
	}

	db4 := openDeclarationTestDB(t, "remove-del-err")
	if err := persistModelTreeAsRaw(db4, &pkgmeta.Model{
		BaseModel:   pkgmeta.BaseModel{Id: sql.NullString{String: "p1", Valid: true}},
		Name:        "P",
		Path:        "/p.ts",
		Application: "a",
		ModuleId:    sql.NullString{String: "mod-z", Valid: true},
	}); err != nil {
		t.Fatal(err)
	}
	if err := db4.Migrator().DropTable(rawServiceTable); err != nil {
		t.Fatal(err)
	}
	if _, err := RemoveModuleDeclarations(db4, "mod-z"); err == nil || !strings.Contains(err.Error(), "delete raw models") {
		t.Fatalf("expected delete raw models error, got %v", err)
	}
}
