// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package schema

import (
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/pkg/meta"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestTranslatedTrigramIndexName(t *testing.T) {
	got := translatedTrigramIndexName("base_language", "name")
	if got != "idx_base_language_name_trgm" {
		t.Fatalf("unexpected index name: %s", got)
	}
}

func TestCreateTranslatedTrigramIndexSQL(t *testing.T) {
	sql := createTranslatedTrigramIndexSQL("base_language", "name", "idx_base_language_name_trgm")
	if !strings.Contains(sql, "jsonb_path_query_array") {
		t.Fatalf("expected jsonb_path_query_array in DDL, got %s", sql)
	}
	if !strings.Contains(sql, "gin_trgm_ops") {
		t.Fatalf("expected gin_trgm_ops in DDL, got %s", sql)
	}
	if !strings.Contains(sql, `USING gin`) {
		t.Fatalf("expected USING gin in DDL, got %s", sql)
	}
}

func TestHasTrigramOnSQLiteIsFalse(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:trigram_probe?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if hasTrigram(db) {
		t.Fatal("sqlite must not report pg_trgm")
	}
}

func TestEnsureTranslatedTrigramIndexSkipsWithoutExtension(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:trigram_ensure?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.Exec(`CREATE TABLE base_language (id text primary key, name text)`).Error; err != nil {
		t.Fatalf("create table: %v", err)
	}
	if err := ensureTranslatedTrigramIndex(db, "base_language", "Name"); err != nil {
		t.Fatalf("ensure without pg_trgm must be non-fatal: %v", err)
	}
	if db.Migrator().HasIndex("base_language", "idx_base_language_name_trgm") {
		t.Fatal("did not expect trigram index on sqlite")
	}
}

func TestIsTranslatedTrigramField(t *testing.T) {
	trueVal := true
	trigram := "trigram"
	btree := "idx_name"

	field := &meta.IrField{Name: "Name"}
	spec := &meta.IrFieldResolvedSpec{
		FieldName: "Name",
		Structural: meta.IrFieldStructuralSpec{
			Translate: &trueVal,
			StorageHints: &meta.IrFieldStructuralStorageHints{
				Index: &trigram,
			},
		},
	}
	if err := field.SetResolvedSpec(spec); err != nil {
		t.Fatalf("SetResolvedSpec: %v", err)
	}
	if !isTranslatedTrigramField(field) {
		t.Fatal("expected trigram translate field")
	}

	spec.Structural.StorageHints.Index = &btree
	if err := field.SetResolvedSpec(spec); err != nil {
		t.Fatalf("SetResolvedSpec btree: %v", err)
	}
	if isTranslatedTrigramField(field) {
		t.Fatal("named btree index must not count as trigram")
	}
}
