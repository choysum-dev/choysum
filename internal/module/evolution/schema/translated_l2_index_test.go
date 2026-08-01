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

func TestTranslatedL2IndexName(t *testing.T) {
	got := translatedL2IndexName("base_language", "name", "en_US")
	if got != "idx_base_language_name_en_us" {
		t.Fatalf("unexpected index name: %s", got)
	}
	gotZh := translatedL2IndexName("base_language", "name", "zh_CN")
	if gotZh != "idx_base_language_name_zh_cn" {
		t.Fatalf("unexpected zh index name: %s", gotZh)
	}
}

func TestTranslatedL2LangsWhitelist(t *testing.T) {
	if len(translatedL2Langs) != 2 {
		t.Fatalf("D17 whitelist must be exactly en_US+zh_CN, got %#v", translatedL2Langs)
	}
	if translatedL2Langs[0] != "en_US" || translatedL2Langs[1] != "zh_CN" {
		t.Fatalf("unexpected whitelist order/content: %#v", translatedL2Langs)
	}
}

func TestCreateTranslatedL2IndexSQL(t *testing.T) {
	sqliteSQL, ok := createTranslatedL2IndexSQL("sqlite", "base_language", "name", "idx_base_language_name_en_us", "en_US")
	if !ok {
		t.Fatal("expected sqlite SQL")
	}
	if !strings.Contains(sqliteSQL, `json_extract("name", '$.en_US')`) {
		t.Fatalf("unexpected sqlite SQL: %s", sqliteSQL)
	}

	mysqlSQL, ok := createTranslatedL2IndexSQL("mysql", "base_language", "name", "idx_base_language_name_zh_cn", "zh_CN")
	if !ok {
		t.Fatal("expected mysql SQL")
	}
	if !strings.Contains(mysqlSQL, "JSON_UNQUOTE(JSON_EXTRACT(`name`, '$.zh_CN'))") {
		t.Fatalf("unexpected mysql SQL: %s", mysqlSQL)
	}

	mariadbSQL, ok := createTranslatedL2IndexSQL("mariadb", "base_language", "name", "idx_base_language_name_en_us", "en_US")
	if !ok {
		t.Fatal("expected mariadb SQL")
	}
	if !strings.Contains(mariadbSQL, "JSON_UNQUOTE") {
		t.Fatalf("unexpected mariadb SQL: %s", mariadbSQL)
	}

	if _, ok := createTranslatedL2IndexSQL("postgres", "base_language", "name", "idx_x", "en_US"); ok {
		t.Fatal("postgres must not emit L2 DDL")
	}
	if _, ok := createTranslatedL2IndexSQL("sqlserver", "base_language", "name", "idx_x", "en_US"); ok {
		t.Fatal("sqlserver L2 is deferred (no expression index without computed columns)")
	}
	if _, ok := createTranslatedL2IndexSQL("sqlite", "", "name", "idx_x", "en_US"); ok {
		t.Fatal("empty table must not emit SQL")
	}
	if _, ok := createTranslatedL2IndexSQL("unknown", "base_language", "name", "idx_x", "en_US"); ok {
		t.Fatal("unknown dialect must not emit SQL")
	}
}

func TestEnsureTranslatedL2IndexesGuards(t *testing.T) {
	if err := ensureTranslatedL2Indexes(nil, "sqlite", "base_language", "Name"); err != nil {
		t.Fatalf("nil db: %v", err)
	}
	db, err := gorm.Open(sqlite.Open("file:l2_guards?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := ensureTranslatedL2Indexes(db, "unknown", "base_language", "Name"); err != nil {
		t.Fatalf("unknown dialect: %v", err)
	}
}

func TestEnsureTranslatedL2IndexesOnSQLite(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:l2_ensure?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.Exec(`CREATE TABLE base_language (id text primary key, name text)`).Error; err != nil {
		t.Fatalf("create table: %v", err)
	}
	if err := ensureTranslatedL2Indexes(db, "sqlite", "base_language", "Name"); err != nil {
		t.Fatalf("ensure L2: %v", err)
	}
	if !db.Migrator().HasIndex("base_language", "idx_base_language_name_en_us") {
		t.Fatal("expected en_US L2 index")
	}
	if !db.Migrator().HasIndex("base_language", "idx_base_language_name_zh_cn") {
		t.Fatal("expected zh_CN L2 index")
	}
	// Idempotent.
	if err := ensureTranslatedL2Indexes(db, "sqlite", "base_language", "Name"); err != nil {
		t.Fatalf("re-ensure L2: %v", err)
	}
}

func TestEnsureTranslatedL2IndexesSkipsPostgres(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:l2_skip_pg?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.Exec(`CREATE TABLE base_language (id text primary key, name text)`).Error; err != nil {
		t.Fatalf("create table: %v", err)
	}
	if err := ensureTranslatedL2Indexes(db, "postgres", "base_language", "Name"); err != nil {
		t.Fatalf("postgres skip must be non-fatal: %v", err)
	}
	if db.Migrator().HasIndex("base_language", "idx_base_language_name_en_us") {
		t.Fatal("did not expect L2 index when dialect=postgres")
	}
}

func TestApplyTableTranslatedL2IndexesRequiresTrigramOptIn(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:l2_optin?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.Exec(`CREATE TABLE base_language (id text primary key, name text)`).Error; err != nil {
		t.Fatalf("create table: %v", err)
	}

	trueVal := true
	trigram := "trigram"
	withOptIn := &meta.Field{Name: "Name"}
	spec := &meta.FieldResolvedSpec{
		FieldName: "Name",
		Structural: meta.FieldStructuralSpec{
			Translate: &trueVal,
			StorageHints: &meta.FieldStructuralStorageHints{
				Index: &trigram,
			},
		},
	}
	if err := withOptIn.SetResolvedSpec(spec); err != nil {
		t.Fatalf("SetResolvedSpec: %v", err)
	}

	withoutOptIn := &meta.Field{Name: "Name"}
	specNoIndex := &meta.FieldResolvedSpec{
		FieldName: "Name",
		Structural: meta.FieldStructuralSpec{
			Translate: &trueVal,
		},
	}
	if err := withoutOptIn.SetResolvedSpec(specNoIndex); err != nil {
		t.Fatalf("SetResolvedSpec no index: %v", err)
	}

	runtime := &schemaTestScope{
		ctx:     context.Background(),
		cfg:     &config.Config{Db: &config.DbConfig{Dialect: "sqlite"}, Server: config.NewDefaultServerConfig(), Log: config.NewDefaultLogConfig()},
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		session: &scope.Session{DB: db},
	}
	m := &modelMigrator{runtimeScope: runtime}
	model := &meta.Model{Fields: []*meta.Field{withoutOptIn}}
	if err := m.applyTableTranslatedL2Indexes("base_language", model); err != nil {
		t.Fatalf("apply without opt-in: %v", err)
	}
	if db.Migrator().HasIndex("base_language", "idx_base_language_name_en_us") {
		t.Fatal("translate without index:trigram must not create L2")
	}

	model.Fields = []*meta.Field{withOptIn}
	if err := m.applyTableTranslatedL2Indexes("base_language", model); err != nil {
		t.Fatalf("apply with opt-in: %v", err)
	}
	if !db.Migrator().HasIndex("base_language", "idx_base_language_name_en_us") {
		t.Fatal("expected L2 after trigram opt-in")
	}
}

func TestEnsureTranslatedL2IndexesEmptyNamesAndUnsupportedDialect(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:l2_empty?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := ensureTranslatedL2Indexes(db, "sqlite", "", "Name"); err != nil {
		t.Fatalf("empty table: %v", err)
	}
	if err := ensureTranslatedL2Indexes(db, "sqlite", "base_language", "  "); err != nil {
		t.Fatalf("empty column: %v", err)
	}
	if err := ensureTranslatedL2Indexes(db, "sqlserver", "base_language", "Name"); err != nil {
		t.Fatalf("sqlserver skip: %v", err)
	}
	if err := ensureTranslatedL2Indexes(db, "postgresql", "base_language", "Name"); err != nil {
		t.Fatalf("postgresql skip: %v", err)
	}
}

func TestEnsureTranslatedL2IndexesWrapsExecError(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:l2_mysql_err?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.Exec(`CREATE TABLE base_language (id text primary key, name text)`).Error; err != nil {
		t.Fatalf("create table: %v", err)
	}
	err = ensureTranslatedL2Indexes(db, "mysql", "base_language", "Name")
	if err == nil {
		t.Fatal("expected mysql DDL to fail on sqlite")
	}
	if !strings.Contains(err.Error(), "ensure translated L2 index") {
		t.Fatalf("expected wrapped L2 error, got %v", err)
	}
}

func TestApplyTableTranslatedL2IndexesPropagatesEnsureError(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:l2_apply_err?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db.Dialector = dialectorWithName{Dialector: db.Dialector, name: "mysql"}
	if err := db.Exec(`CREATE TABLE base_language (id text primary key, name text)`).Error; err != nil {
		t.Fatalf("create table: %v", err)
	}

	trueVal := true
	trigram := "trigram"
	field := &meta.Field{Name: "Name"}
	spec := &meta.FieldResolvedSpec{
		FieldName: "Name",
		Structural: meta.FieldStructuralSpec{
			Translate: &trueVal,
			StorageHints: &meta.FieldStructuralStorageHints{
				Index: &trigram,
			},
		},
	}
	if err := field.SetResolvedSpec(spec); err != nil {
		t.Fatalf("SetResolvedSpec: %v", err)
	}

	runtime := &schemaTestScope{
		ctx:     context.Background(),
		cfg:     &config.Config{Db: &config.DbConfig{Dialect: "mysql"}, Server: config.NewDefaultServerConfig(), Log: config.NewDefaultLogConfig()},
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		session: &scope.Session{DB: db},
	}
	m := &modelMigrator{runtimeScope: runtime}
	if err := m.applyTableTranslatedL2Indexes("base_language", &meta.Model{Fields: []*meta.Field{field}}); err == nil {
		t.Fatal("expected apply to surface mysql ensure error")
	}
}

func TestMigrateTableSchemaWrapsTranslatedL2IndexError(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:l2_migrate_wrap?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db.Dialector = dialectorWithName{Dialector: db.Dialector, name: "mysql"}

	trueVal := true
	trigram := "trigram"
	field := &meta.Field{Name: "Name"}
	spec := &meta.FieldResolvedSpec{
		FieldName: "Name",
		Structural: meta.FieldStructuralSpec{
			Name:       "Name",
			FieldType:  "varchar",
			Translate:  &trueVal,
			ColumnType: "jsonobject",
			StorageHints: &meta.FieldStructuralStorageHints{
				Index: &trigram,
			},
		},
		Migration: meta.FieldMigrationDecision{
			StorageKind:        "physical",
			ShouldCreateColumn: true,
			ResolvedColumnType: "jsonobject",
			ReasonCode:         "TRANSLATE_LANG_MAP",
		},
	}
	if err := field.SetResolvedSpec(spec); err != nil {
		t.Fatalf("SetResolvedSpec: %v", err)
	}
	model := &meta.Model{
		Name:       "Language",
		Path:       "base/language.ts",
		ModelTable: "base_language_l2_wrap",
		Fields:     []*meta.Field{field},
	}
	runtime := &schemaTestScope{
		ctx:     context.Background(),
		cfg:     &config.Config{Db: &config.DbConfig{Dialect: "mysql"}, Server: config.NewDefaultServerConfig(), Log: config.NewDefaultLogConfig()},
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		session: &scope.Session{DB: db},
	}
	m := &modelMigrator{runtimeScope: runtime}
	err = m.migrateTableSchema([]*meta.Model{model})
	if err == nil || !strings.Contains(err.Error(), "translated L2 indexes") {
		t.Fatalf("expected wrapped L2 migrate error, got %v", err)
	}
}
