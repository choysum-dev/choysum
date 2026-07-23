// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package schema

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
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

	if isTranslatedTrigramField(nil) {
		t.Fatal("nil field")
	}
	noHints := &meta.IrField{Name: "Name"}
	noHintSpec := &meta.IrFieldResolvedSpec{
		FieldName:  "Name",
		Structural: meta.IrFieldStructuralSpec{Translate: &trueVal},
	}
	if err := noHints.SetResolvedSpec(noHintSpec); err != nil {
		t.Fatalf("SetResolvedSpec no hints: %v", err)
	}
	if isTranslatedTrigramField(noHints) {
		t.Fatal("translate without trigram hints must be false")
	}
}

func TestEnsureTranslatedTrigramIndexGuards(t *testing.T) {
	if err := ensureTranslatedTrigramIndex(nil, "base_language", "Name"); err != nil {
		t.Fatalf("nil db: %v", err)
	}
	db, err := gorm.Open(sqlite.Open("file:trigram_guards?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := ensureTranslatedTrigramIndex(db, "", "Name"); err != nil {
		t.Fatalf("empty table: %v", err)
	}
	if err := ensureTranslatedTrigramIndex(db, "base_language", ""); err != nil {
		t.Fatalf("empty column: %v", err)
	}
	if hasTrigram(nil) {
		t.Fatal("nil db must not report pg_trgm")
	}
}

func TestApplyTableTranslatedTrigramIndexesSkipsNonPostgres(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:trigram_apply?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.Exec(`CREATE TABLE base_language (id text primary key, name text)`).Error; err != nil {
		t.Fatalf("create table: %v", err)
	}
	trueVal := true
	trigram := "trigram"
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
	runtime := &schemaTestScope{
		ctx:     context.Background(),
		cfg:     &config.Config{Db: &config.DbConfig{Dialect: "sqlite"}, Server: config.NewDefaultServerConfig(), Log: config.NewDefaultLogConfig()},
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		session: &scope.Session{DB: db},
	}
	m := &modelMigrator{runtimeScope: runtime}
	if err := m.applyTableTranslatedTrigramIndexes("base_language", &meta.IrModel{Fields: []*meta.IrField{field}}); err != nil {
		t.Fatalf("sqlite apply must no-op: %v", err)
	}
	if db.Migrator().HasIndex("base_language", "idx_base_language_name_trgm") {
		t.Fatal("did not expect trigram index on sqlite")
	}
}

type dialectorWithName struct {
	gorm.Dialector
	name string
}

func (d dialectorWithName) Name() string { return d.name }

func openPostgresNamedSQLite(t *testing.T, dsn string) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db.Dialector = dialectorWithName{Dialector: db.Dialector, name: "postgres"}
	return db
}

func TestHasTrigramPostgresProbePaths(t *testing.T) {
	db := openPostgresNamedSQLite(t, "file:trigram_pg_probe?mode=memory&cache=shared")
	if hasTrigram(db) {
		t.Fatal("postgres-named sqlite without pg_extension must be false")
	}

	if err := db.Exec(`CREATE TABLE pg_extension (extname text)`).Error; err != nil {
		t.Fatalf("create pg_extension: %v", err)
	}
	if hasTrigram(db) {
		t.Fatal("empty pg_extension must be false")
	}
	if err := db.Exec(`INSERT INTO pg_extension (extname) VALUES ('pg_trgm')`).Error; err != nil {
		t.Fatalf("insert pg_trgm: %v", err)
	}
	if !hasTrigram(db) {
		t.Fatal("expected hasTrigram true after pg_trgm row")
	}

	dbPG := openPostgresNamedSQLite(t, "file:trigram_pg_alias?mode=memory&cache=shared")
	dbPG.Dialector = dialectorWithName{Dialector: dbPG.Dialector, name: "postgresql"}
	if err := dbPG.Exec(`CREATE TABLE pg_extension (extname text)`).Error; err != nil {
		t.Fatalf("create pg_extension postgresql: %v", err)
	}
	if err := dbPG.Exec(`INSERT INTO pg_extension (extname) VALUES ('pg_trgm')`).Error; err != nil {
		t.Fatalf("insert pg_trgm postgresql: %v", err)
	}
	if !hasTrigram(dbPG) {
		t.Fatal("postgresql alias must also detect pg_trgm")
	}

	dbNilDialector := openPostgresNamedSQLite(t, "file:trigram_nil_dialector?mode=memory&cache=shared")
	dbNilDialector.Dialector = nil
	if hasTrigram(dbNilDialector) {
		t.Fatal("nil dialector must be false")
	}
}

func TestEnsureTranslatedTrigramIndexWithExtension(t *testing.T) {
	db := openPostgresNamedSQLite(t, "file:trigram_pg_ensure?mode=memory&cache=shared")
	if err := db.Exec(`CREATE TABLE base_language (id text primary key, name text)`).Error; err != nil {
		t.Fatalf("create table: %v", err)
	}
	if err := db.Exec(`CREATE TABLE pg_extension (extname text)`).Error; err != nil {
		t.Fatalf("create pg_extension: %v", err)
	}
	if err := db.Exec(`INSERT INTO pg_extension (extname) VALUES ('pg_trgm')`).Error; err != nil {
		t.Fatalf("insert pg_trgm: %v", err)
	}

	if err := db.Exec(`CREATE INDEX idx_base_language_name_trgm ON base_language(name)`).Error; err != nil {
		t.Fatalf("create placeholder index: %v", err)
	}
	if err := ensureTranslatedTrigramIndex(db, "base_language", "Name"); err != nil {
		t.Fatalf("ensure with existing index: %v", err)
	}

	db2 := openPostgresNamedSQLite(t, "file:trigram_pg_ensure_err?mode=memory&cache=shared")
	if err := db2.Exec(`CREATE TABLE base_language (id text primary key, name text)`).Error; err != nil {
		t.Fatalf("create table 2: %v", err)
	}
	if err := db2.Exec(`CREATE TABLE pg_extension (extname text)`).Error; err != nil {
		t.Fatalf("create pg_extension 2: %v", err)
	}
	if err := db2.Exec(`INSERT INTO pg_extension (extname) VALUES ('pg_trgm')`).Error; err != nil {
		t.Fatalf("insert pg_trgm 2: %v", err)
	}
	err := ensureTranslatedTrigramIndex(db2, "base_language", "Name")
	if err == nil {
		t.Fatal("expected gin DDL to fail on sqlite")
	}
	if !strings.Contains(err.Error(), "ensure translated trigram index") {
		t.Fatalf("expected wrapped ensure error, got %v", err)
	}
}

func TestApplyTableTranslatedTrigramIndexesPostgresPath(t *testing.T) {
	db := openPostgresNamedSQLite(t, "file:trigram_pg_apply?mode=memory&cache=shared")
	if err := db.Exec(`CREATE TABLE base_language (id text primary key, name text)`).Error; err != nil {
		t.Fatalf("create table: %v", err)
	}
	if err := db.Exec(`CREATE TABLE pg_extension (extname text)`).Error; err != nil {
		t.Fatalf("create pg_extension: %v", err)
	}
	if err := db.Exec(`INSERT INTO pg_extension (extname) VALUES ('pg_trgm')`).Error; err != nil {
		t.Fatalf("insert pg_trgm: %v", err)
	}
	if err := db.Exec(`CREATE INDEX idx_base_language_name_trgm ON base_language(name)`).Error; err != nil {
		t.Fatalf("placeholder index: %v", err)
	}

	trueVal := true
	trigram := "trigram"
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
	skipField := &meta.IrField{Name: "Code"}
	skipSpec := &meta.IrFieldResolvedSpec{
		FieldName:  "Code",
		Structural: meta.IrFieldStructuralSpec{Translate: &trueVal},
	}
	if err := skipField.SetResolvedSpec(skipSpec); err != nil {
		t.Fatalf("SetResolvedSpec skip: %v", err)
	}

	runtime := &schemaTestScope{
		ctx:     context.Background(),
		cfg:     &config.Config{Db: &config.DbConfig{Dialect: "postgres"}, Server: config.NewDefaultServerConfig(), Log: config.NewDefaultLogConfig()},
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		session: &scope.Session{DB: db},
	}
	m := &modelMigrator{runtimeScope: runtime}
	if err := m.applyTableTranslatedTrigramIndexes("base_language", &meta.IrModel{Fields: []*meta.IrField{skipField, field}}); err != nil {
		t.Fatalf("postgres apply with existing index: %v", err)
	}

	dbErr := openPostgresNamedSQLite(t, "file:trigram_pg_apply_err?mode=memory&cache=shared")
	if err := dbErr.Exec(`CREATE TABLE base_language (id text primary key, name text)`).Error; err != nil {
		t.Fatalf("create table err db: %v", err)
	}
	if err := dbErr.Exec(`CREATE TABLE pg_extension (extname text)`).Error; err != nil {
		t.Fatalf("create pg_extension err db: %v", err)
	}
	if err := dbErr.Exec(`INSERT INTO pg_extension (extname) VALUES ('pg_trgm')`).Error; err != nil {
		t.Fatalf("insert pg_trgm err db: %v", err)
	}
	runtime.session = &scope.Session{DB: dbErr}
	if err := m.applyTableTranslatedTrigramIndexes("base_language", &meta.IrModel{Fields: []*meta.IrField{field}}); err == nil {
		t.Fatal("expected apply to surface ensure DDL error")
	}
}
