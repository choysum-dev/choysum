// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

import (
	"database/sql"
	"fmt"
	"strings"
	"testing"

	pkgmeta "github.com/choysum-dev/choysum/pkg/meta"
	"gorm.io/gorm"
)

// countRawModelsByID counts declaration rows with the given id (including soft-deleted when unscoped).
func countRawModelsByID(db *gorm.DB, id string, unscoped bool) (int64, error) {
	if db == nil {
		return 0, fmt.Errorf("db is nil")
	}
	id = strings.TrimSpace(id)
	q := db.Model(&rawModel{})
	if unscoped {
		q = q.Unscoped()
	}
	var n int64
	err := q.Where("id = ?", id).Count(&n).Error
	return n, err
}

// dropRawModelTable drops meta_raw_model (tests that force list/load failures).
func dropRawModelTable(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	return db.Migrator().DropTable(rawModelTable)
}

func TestRawHelpers_CountAndDrop(t *testing.T) {
	if _, err := countRawModelsByID(nil, "x", false); err == nil || !strings.Contains(err.Error(), "db is nil") {
		t.Fatalf("nil db: %v", err)
	}
	if err := dropRawModelTable(nil); err == nil || !strings.Contains(err.Error(), "db is nil") {
		t.Fatalf("nil drop: %v", err)
	}

	db := openDeclarationTestDB(t, "raw-helpers")
	m := &pkgmeta.Model{
		BaseModel:   pkgmeta.BaseModel{Id: sql.NullString{String: "raw-1", Valid: true}},
		Name:        "X",
		Path:        "/x.ts",
		Application: "app",
	}
	if err := persistModelTreeAsRaw(db, m); err != nil {
		t.Fatal(err)
	}
	n, err := countRawModelsByID(db, "raw-1", false)
	if err != nil || n != 1 {
		t.Fatalf("count=%d err=%v", n, err)
	}
	n, err = countRawModelsByID(db, "  raw-1  ", true)
	if err != nil || n != 1 {
		t.Fatalf("unscoped count=%d err=%v", n, err)
	}
	if err := dropRawModelTable(db); err != nil {
		t.Fatal(err)
	}
	if db.Migrator().HasTable(rawModelTable) {
		t.Fatal("expected table dropped")
	}
}

func TestRawHelpers_SoftDeleteScopedCount(t *testing.T) {
	db := openDeclarationTestDB(t, "raw-helpers-soft")
	m := &pkgmeta.Model{
		BaseModel:   pkgmeta.BaseModel{Id: sql.NullString{String: "raw-soft", Valid: true}},
		Name:        "Soft",
		Path:        "/soft.ts",
		Application: "app",
	}
	if err := persistModelTreeAsRaw(db, m); err != nil {
		t.Fatal(err)
	}
	if err := db.Where("id = ?", "raw-soft").Delete(&rawModel{}).Error; err != nil {
		t.Fatalf("soft delete: %v", err)
	}
	n, err := countRawModelsByID(db, "raw-soft", false)
	if err != nil || n != 0 {
		t.Fatalf("scoped count after soft delete = %d err=%v, want 0", n, err)
	}
	n, err = countRawModelsByID(db, "raw-soft", true)
	if err != nil || n != 1 {
		t.Fatalf("unscoped count after soft delete = %d err=%v, want 1", n, err)
	}
}
