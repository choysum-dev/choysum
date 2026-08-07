// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

import (
	"database/sql"
	"strings"
	"testing"

	pkgmeta "github.com/choysum-dev/choysum/pkg/meta"
)

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
