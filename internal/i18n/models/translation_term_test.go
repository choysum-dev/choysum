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

func TestEnsureTranslationTermTableSkipsCoreAndEmpty(t *testing.T) {
	rs := newTestScope(t)

	if err := EnsureTranslationTermTable(rs, "core"); err != nil {
		t.Fatalf("EnsureTranslationTermTable(core): %v", err)
	}
	if rs.Session().Migrator().HasTable("core_translation_term") {
		t.Fatal("expected no core_translation_term table")
	}

	if err := EnsureTranslationTermTable(rs, ""); err != nil {
		t.Fatalf("EnsureTranslationTermTable(\"\"): %v", err)
	}
	if err := EnsureTranslationTermTable(nil, "auth"); err != nil {
		t.Fatalf("EnsureTranslationTermTable(nil scope): %v", err)
	}
	if err := EnsureTranslationTermTable(&testScope{}, "auth"); err != nil {
		t.Fatalf("EnsureTranslationTermTable(nil session): %v", err)
	}
}

func TestEnsureTranslationTermTableCreatesAuthTable(t *testing.T) {
	rs := newTestScope(t)

	if err := EnsureTranslationTermTable(rs, "auth"); err != nil {
		t.Fatalf("EnsureTranslationTermTable(auth): %v", err)
	}
	if !rs.Session().Migrator().HasTable("auth_translation_term") {
		t.Fatal("expected auth_translation_term table")
	}
	if rs.Session().Migrator().HasTable("core_translation_term") {
		t.Fatal("ensure auth must not create core_translation_term")
	}

	if err := EnsureTranslationTermTable(rs, "auth"); err != nil {
		t.Fatalf("EnsureTranslationTermTable(auth) second call: %v", err)
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
}

func TestEnsureTranslationTermTableMultiApp(t *testing.T) {
	rs := newTestScope(t)

	for _, app := range []string{"auth", "web"} {
		if err := EnsureTranslationTermTable(rs, app); err != nil {
			t.Fatalf("EnsureTranslationTermTable(%s): %v", app, err)
		}
	}
	if !rs.Session().Migrator().HasTable("auth_translation_term") || !rs.Session().Migrator().HasTable("web_translation_term") {
		t.Fatal("expected both auth_translation_term and web_translation_term")
	}
}

func TestTranslationTermUniqueKeyAndApplicationColumn(t *testing.T) {
	rs := newTestScope(t)
	if err := EnsureTranslationTermTable(rs, "auth"); err != nil {
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

func TestEnsureTranslationTermTableDoesNotRegisterIrModel(t *testing.T) {
	rs := newTestScope(t)
	if err := rs.Session().AutoMigrate(&meta.IrModel{}); err != nil {
		t.Fatalf("migrate IrModel: %v", err)
	}
	if err := EnsureTranslationTermTable(rs, "auth"); err != nil {
		t.Fatalf("ensure: %v", err)
	}

	var count int64
	if err := rs.Session().Model(&meta.IrModel{}).Where("name = ?", "TranslationTerm").Count(&count).Error; err != nil {
		t.Fatalf("count IrModel: %v", err)
	}
	if count != 0 {
		t.Fatalf("MVP must not register TranslationTerm IrModel, got count=%d", count)
	}
}
