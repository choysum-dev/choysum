// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package schema

import (
	"testing"

	"github.com/choysum-dev/choysum/pkg/scope"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newSavedFilterRenameScope(t *testing.T, dsn string) (*schemaTestScope, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	return &schemaTestScope{session: &scope.Session{DB: db}}, db
}

func TestEnsureSavedFilterCreateUidRename(t *testing.T) {
	runtimeScope, db := newSavedFilterRenameScope(t, "file:sf_create_uid_rename?mode=memory&cache=shared")
	if err := db.Exec(`CREATE TABLE web_saved_filter (
		id text primary key,
		create_uid text,
		name text
	)`).Error; err != nil {
		t.Fatalf("create table: %v", err)
	}
	if err := db.Exec(`INSERT INTO web_saved_filter (id, create_uid, name) VALUES ('1', 'u1', 'fav')`).Error; err != nil {
		t.Fatalf("insert: %v", err)
	}

	if err := ensureSavedFilterCreateUidRename(runtimeScope); err != nil {
		t.Fatalf("rename: %v", err)
	}

	migrator := db.Migrator()
	if migrator.HasColumn(savedFilterTable, "create_uid") {
		t.Fatal("expected create_uid to be renamed away")
	}
	if !migrator.HasColumn(savedFilterTable, "created_uid") {
		t.Fatal("expected created_uid column")
	}
	var got string
	if err := db.Raw(`SELECT created_uid FROM web_saved_filter WHERE id = '1'`).Scan(&got).Error; err != nil {
		t.Fatalf("select: %v", err)
	}
	if got != "u1" {
		t.Fatalf("created_uid = %q, want u1", got)
	}
}

func TestEnsureSavedFilterCreateUidRenameBothColumns(t *testing.T) {
	runtimeScope, db := newSavedFilterRenameScope(t, "file:sf_create_uid_both?mode=memory&cache=shared")
	if err := db.Exec(`CREATE TABLE web_saved_filter (
		id text primary key,
		create_uid text,
		created_uid text,
		name text
	)`).Error; err != nil {
		t.Fatalf("create table: %v", err)
	}
	if err := db.Exec(`INSERT INTO web_saved_filter (id, create_uid, created_uid, name) VALUES
		('1', 'old', NULL, 'a'),
		('2', 'x', 'kept', 'b')`).Error; err != nil {
		t.Fatalf("insert: %v", err)
	}

	if err := ensureSavedFilterCreateUidRename(runtimeScope); err != nil {
		t.Fatalf("merge: %v", err)
	}
	migrator := db.Migrator()
	if migrator.HasColumn(savedFilterTable, "create_uid") {
		t.Fatal("expected create_uid dropped")
	}
	var a, b string
	if err := db.Raw(`SELECT created_uid FROM web_saved_filter WHERE id = '1'`).Scan(&a).Error; err != nil {
		t.Fatalf("select a: %v", err)
	}
	if err := db.Raw(`SELECT created_uid FROM web_saved_filter WHERE id = '2'`).Scan(&b).Error; err != nil {
		t.Fatalf("select b: %v", err)
	}
	if a != "old" {
		t.Fatalf("row1 created_uid = %q, want old", a)
	}
	if b != "kept" {
		t.Fatalf("row2 created_uid = %q, want kept", b)
	}
}

func TestEnsureSavedFilterCreateUidRenameNoop(t *testing.T) {
	if err := ensureSavedFilterCreateUidRename(nil); err != nil {
		t.Fatalf("nil scope: %v", err)
	}
	runtimeScope, db := newSavedFilterRenameScope(t, "file:sf_create_uid_noop?mode=memory&cache=shared")
	if err := ensureSavedFilterCreateUidRename(runtimeScope); err != nil {
		t.Fatalf("noop missing table: %v", err)
	}
	if err := db.Exec(`CREATE TABLE web_saved_filter (id text primary key, created_uid text)`).Error; err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := ensureSavedFilterCreateUidRename(runtimeScope); err != nil {
		t.Fatalf("noop new-only: %v", err)
	}
}
