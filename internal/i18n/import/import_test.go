// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package i18nimport_test

import (
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"testing"

	i18nimport "github.com/choysum-dev/choysum/internal/i18n/import"
	i18nmodels "github.com/choysum-dev/choysum/internal/i18n/models"
	"github.com/choysum-dev/choysum/internal/i18n/store"
	"github.com/choysum-dev/choysum/internal/testing/scopetest"
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
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "import.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	return &testScope{
		ctx:     context.Background(),
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		session: &scope.Session{DB: db},
	}
}

func TestImportModulePoMsgctxtOverrideAndObsolete(t *testing.T) {
	rs := newTestScope(t)
	if err := i18nmodels.EnsureTranslationTermTable(rs, "auth"); err != nil {
		t.Fatal(err)
	}
	table := rs.Session().Table("auth_translation_term")
	if err := table.Create(&i18nmodels.TranslationTerm{
		Application: "auth",
		Module:      "auth",
		Lang:        "zh_CN",
		Scope:       "web/a@title",
		Src:         "Hello",
		Value:       "编辑器改的",
		Kind:        i18nmodels.KindLiteral,
		Source:      i18nmodels.SourceOverride,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := table.Create(&i18nmodels.TranslationTerm{
		Application: "auth",
		Module:      "auth",
		Lang:        "zh_CN",
		Scope:       "web/a@keep",
		Src:         "Keep",
		Value:       "保留旧",
		Kind:        i18nmodels.KindLiteral,
		Source:      i18nmodels.SourcePackaged,
	}).Error; err != nil {
		t.Fatal(err)
	}

	poText := []byte(`
msgid ""
msgstr "Language: zh_CN\n"

msgctxt "web/a@title"
msgid "Hello"
msgstr "你好"

msgid "NoContext"
msgstr "无上下文"

msgctxt "web/a@new"
msgid "New"
msgstr "新的"

#~ msgctxt "web/a@keep"
#~ msgid "Keep"
#~ msgstr "应忽略"
`)

	reg := store.NewRegistry(rs)
	stats, err := i18nimport.ImportModulePo(rs, reg, "auth", "auth", "zh_CN", poText)
	if err != nil {
		t.Fatalf("ImportModulePo: %v", err)
	}
	if stats.Upserted != 1 || stats.SkippedOverride != 1 || stats.RejectedNoCtxt != 1 || stats.SkippedObsolete != 1 {
		t.Fatalf("unexpected stats: %+v", stats)
	}

	var override i18nmodels.TranslationTerm
	if err := rs.Session().Table("auth_translation_term").Where("src = ?", "Hello").Take(&override).Error; err != nil {
		t.Fatal(err)
	}
	if override.Value != "编辑器改的" || override.Source != i18nmodels.SourceOverride {
		t.Fatalf("override overwritten: %+v", override)
	}

	var keep i18nmodels.TranslationTerm
	if err := rs.Session().Table("auth_translation_term").Where("src = ?", "Keep").Take(&keep).Error; err != nil {
		t.Fatal(err)
	}
	if keep.Value != "保留旧" {
		t.Fatalf("obsolete must not delete/overwrite Keep: %+v", keep)
	}

	var neu i18nmodels.TranslationTerm
	if err := rs.Session().Table("auth_translation_term").Where("src = ?", "New").Take(&neu).Error; err != nil {
		t.Fatal(err)
	}
	if neu.Scope != "web/a@new" || neu.Value != "新的" || neu.Source != i18nmodels.SourcePackaged {
		t.Fatalf("new term: %+v", neu)
	}

	val, ok := reg.Lookup("auth", "zh_CN", "web/a@new", "New", "")
	if !ok || val != "新的" {
		t.Fatalf("cache Lookup after import = %q ok=%v", val, ok)
	}
}

func TestDeleteModuleTerms(t *testing.T) {
	rs := newTestScope(t)
	if err := i18nmodels.EnsureTranslationTermTable(rs, "auth"); err != nil {
		t.Fatal(err)
	}
	table := rs.Session().Table("auth_translation_term")
	if err := table.Create(&i18nmodels.TranslationTerm{
		Application: "auth", Module: "auth", Lang: "zh_CN",
		Scope: "a", Src: "X", Value: "Y", Kind: i18nmodels.KindLiteral, Source: i18nmodels.SourcePackaged,
	}).Error; err != nil {
		t.Fatal(err)
	}
	reg := store.NewRegistry(rs)
	reg.RememberModuleApplication("auth", "auth")
	_ = reg.StoreFor("auth").WarmLanguage("zh_CN")

	if err := i18nimport.DeleteModuleTerms(rs, reg, "auth", "auth"); err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := table.Where("module = ?", "auth").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("expected 0 rows, got %d", count)
	}
	if _, ok := reg.Lookup("auth", "zh_CN", "a", "X", ""); ok {
		t.Fatal("expected cache miss after delete")
	}
}

func TestImportModuleI18nDirSkipMissing(t *testing.T) {
	rs := newTestScope(t)
	reg := store.NewRegistry(rs)
	if err := i18nimport.ImportModuleI18nDir(rs, reg, "auth", "auth", filepath.Join(t.TempDir(), "missing")); err != nil {
		t.Fatalf("missing i18n dir should skip: %v", err)
	}
}
