// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package models

import (
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type testScope struct {
	ctx     context.Context
	logger  *slog.Logger
	session *scope.Session
}

func (s *testScope) Run(fn func(scope.Scope) error) error { return fn(s) }

func (s *testScope) Transactor() scope.Transactor {
	return scopetest.NewPassthroughTransactor(s)
}

func (s *testScope) Session() *scope.Session { return s.session }

func (s *testScope) WithContext(ctx context.Context) scope.Scope {
	if ctx == nil {
		ctx = s.ctx
	}
	return &testScope{ctx: ctx, logger: s.logger, session: s.session}
}

func (s *testScope) Context() context.Context {
	if s.ctx != nil {
		return s.ctx
	}
	return context.Background()
}

func (s *testScope) Logger() *slog.Logger { return s.logger }

func newTestScope(t *testing.T) *testScope {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "translation_term.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	return &testScope{
		ctx:     context.Background(),
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		session: &scope.Session{DB: db},
	}
}

func TestTranslationTermTableName(t *testing.T) {
	if got := TranslationTermTableName("auth"); got != "auth_translation_term" {
		t.Fatalf("TranslationTermTableName(auth) = %q", got)
	}
}

func TestValidApplicationIdentifier(t *testing.T) {
	if !ValidApplicationIdentifier("auth") || !ValidApplicationIdentifier("partner_bank") {
		t.Fatal("expected valid identifiers")
	}
	for _, bad := range []string{"", "auth-web", "auth;drop", `auth"`, "auth table"} {
		if ValidApplicationIdentifier(bad) {
			t.Fatalf("ValidApplicationIdentifier(%q) = true, want false", bad)
		}
	}
}

func TestMigrateTranslationTermTableRejectsUnsafeApplication(t *testing.T) {
	rs := newTestScope(t)
	err := MigrateTranslationTermTable(rs, `auth"; drop table users; --`)
	if err == nil || !strings.Contains(err.Error(), "invalid application name") {
		t.Fatalf("expected invalid application name error, got %v", err)
	}
	if rs.Session().Migrator().HasTable(`auth"; drop table users; --_translation_term`) {
		t.Fatal("did not expect unsafe table to be created")
	}
}

func TestMigrateTranslationTermTableSkipsCoreAndEmpty(t *testing.T) {
	rs := newTestScope(t)

	if err := MigrateTranslationTermTable(rs, "core"); err != nil {
		t.Fatalf("MigrateTranslationTermTable(core): %v", err)
	}
	if rs.Session().Migrator().HasTable("core_translation_term") {
		t.Fatal("expected no core_translation_term table")
	}

	if err := MigrateTranslationTermTable(rs, ""); err != nil {
		t.Fatalf("MigrateTranslationTermTable(\"\"): %v", err)
	}
	if err := MigrateTranslationTermTable(nil, "auth"); err != nil {
		t.Fatalf("MigrateTranslationTermTable(nil scope): %v", err)
	}
	if err := MigrateTranslationTermTable(&testScope{}, "auth"); err != nil {
		t.Fatalf("MigrateTranslationTermTable(nil session): %v", err)
	}
}

func TestMigrateTranslationTermTableCreatesAuthTable(t *testing.T) {
	rs := newTestScope(t)

	if err := MigrateTranslationTermTable(rs, "auth"); err != nil {
		t.Fatalf("MigrateTranslationTermTable(auth): %v", err)
	}
	if !rs.Session().Migrator().HasTable("auth_translation_term") {
		t.Fatal("expected auth_translation_term table")
	}
	if rs.Session().Migrator().HasTable("core_translation_term") {
		t.Fatal("ensure auth must not create core_translation_term")
	}

	if err := MigrateTranslationTermTable(rs, "auth"); err != nil {
		t.Fatalf("MigrateTranslationTermTable(auth) second call: %v", err)
	}
}

func TestNormalizeKind(t *testing.T) {
	if got := NormalizeKind(""); got != KindLiteral {
		t.Fatalf("empty = %q", got)
	}
	if got := NormalizeKind("  "); got != KindLiteral {
		t.Fatalf("whitespace = %q", got)
	}
	if got := NormalizeKind("custom_title"); got != "custom_title" {
		t.Fatalf("custom = %q", got)
	}
}

func TestCreateUniqueIndexSQLMySQLUsesSrcPrefix(t *testing.T) {
	got := createUniqueIndexSQL("mysql", "auth_translation_term", "uq_auth_translation_term_key")
	want := "CREATE UNIQUE INDEX `uq_auth_translation_term_key` ON `auth_translation_term` (module(64), lang, scope(255), src(255), kind)"
	if got != want {
		t.Fatalf("createUniqueIndexSQL(mysql) = %q, want %q", got, want)
	}
	sqlite := createUniqueIndexSQL("sqlite", "auth_translation_term", "uq_auth_translation_term_key")
	if strings.Contains(sqlite, "src(255)") {
		t.Fatalf("sqlite index must not use MySQL prefix length, got %q", sqlite)
	}
	pg := createUniqueIndexSQL("postgres", "auth_translation_term", "uq_auth_translation_term_key")
	wantPG := `CREATE UNIQUE INDEX "uq_auth_translation_term_key" ON "auth_translation_term" (module, lang, scope, src, kind)`
	if pg != wantPG {
		t.Fatalf("createUniqueIndexSQL(postgres) = %q, want %q", pg, wantPG)
	}
}

func TestMigrateTranslationTermTableAutoMigrateError(t *testing.T) {
	rs := newTestScope(t)
	sqlDB, err := rs.Session().DB.DB()
	if err != nil {
		t.Fatalf("db.DB: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	err = MigrateTranslationTermTable(rs, "auth")
	if err == nil || !strings.Contains(err.Error(), "migrate auth_translation_term") {
		t.Fatalf("expected AutoMigrate wrap, got %v", err)
	}
}

func TestMigrateTranslationTermTableUniqueIndexError(t *testing.T) {
	rs := newTestScope(t)
	sqlDB, err := rs.Session().DB.DB()
	if err != nil {
		t.Fatalf("db.DB: %v", err)
	}
	// PRAGMA query_only is per-connection; pin the pool so migrate reuses it.
	sqlDB.SetMaxOpenConns(1)
	if err := MigrateTranslationTermTable(rs, "auth"); err != nil {
		t.Fatalf("initial migrate: %v", err)
	}
	indexName := translationTermUniqueIndexName("auth_translation_term")
	if err := rs.Session().Exec("DROP INDEX IF EXISTS `" + indexName + "`").Error; err != nil {
		t.Fatalf("drop index: %v", err)
	}
	if err := rs.Session().Exec("PRAGMA query_only = ON").Error; err != nil {
		t.Fatalf("query_only: %v", err)
	}
	err = MigrateTranslationTermTable(rs, "auth")
	if err == nil || !strings.Contains(err.Error(), "unique index") {
		t.Fatalf("expected unique index wrap, got %v", err)
	}
}

func TestMigrateTranslationTermTableMultiApp(t *testing.T) {
	rs := newTestScope(t)

	for _, app := range []string{"auth", "web"} {
		if err := MigrateTranslationTermTable(rs, app); err != nil {
			t.Fatalf("MigrateTranslationTermTable(%s): %v", app, err)
		}
	}
	if !rs.Session().Migrator().HasTable("auth_translation_term") || !rs.Session().Migrator().HasTable("web_translation_term") {
		t.Fatal("expected both auth_translation_term and web_translation_term")
	}
}

func TestTranslationTermUniqueKeyAndApplicationColumn(t *testing.T) {
	rs := newTestScope(t)
	if err := MigrateTranslationTermTable(rs, "auth"); err != nil {
		t.Fatalf("ensure: %v", err)
	}

	table := rs.Session().Table("auth_translation_term")
	row := &TranslationTerm{
		Application: "auth",
		Module:      "auth",
		Lang:        "zh_CN",
		Scope:       "web/pages/Login@title",
		Src:         "Sign in",
		Value:       "登录",
		Kind:        KindLiteral,
		Source:      SourcePackaged,
	}
	if err := table.Create(row).Error; err != nil {
		t.Fatalf("create first row: %v", err)
	}
	if row.Application != "auth" {
		t.Fatalf("Application column = %q, want auth", row.Application)
	}

	dup := &TranslationTerm{
		Application: "auth-other",
		Module:      "auth",
		Lang:        "zh_CN",
		Scope:       "web/pages/Login@title",
		Src:         "Sign in",
		Value:       "登录2",
		Kind:        KindLiteral,
		Source:      SourceOverride,
	}
	if err := rs.Session().Table("auth_translation_term").Create(dup).Error; err == nil {
		t.Fatal("expected unique constraint violation for duplicate (Module,Lang,Scope,Src,Kind)")
	}

	otherScope := &TranslationTerm{
		Application: "auth",
		Module:      "auth",
		Lang:        "zh_CN",
		Scope:       "web/pages/Login@button",
		Src:         "Sign in",
		Value:       "登录",
		Kind:        KindLiteral,
		Source:      SourcePackaged,
	}
	if err := rs.Session().Table("auth_translation_term").Create(otherScope).Error; err != nil {
		t.Fatalf("create different Scope: %v", err)
	}
}

func TestMigrateTranslationTermTableDoesNotRegisterModel(t *testing.T) {
	rs := newTestScope(t)
	if err := rs.Session().AutoMigrate(&meta.Model{}); err != nil {
		t.Fatalf("migrate Model: %v", err)
	}
	if err := MigrateTranslationTermTable(rs, "auth"); err != nil {
		t.Fatalf("ensure: %v", err)
	}

	var count int64
	if err := rs.Session().Model(&meta.Model{}).Where("name = ?", "TranslationTerm").Count(&count).Error; err != nil {
		t.Fatalf("count Model: %v", err)
	}
	if count != 0 {
		t.Fatalf("MVP must not register TranslationTerm Model, got count=%d", count)
	}
}
