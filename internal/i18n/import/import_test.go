// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package i18nimport_test

import (
	"context"
	"io"
	"log/slog"
	"os"
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

func TestUpsertPackagedTermsMigratesMissingTable(t *testing.T) {
	rs := newTestScope(t)
	if rs.Session().Migrator().HasTable("auth_translation_term") {
		t.Fatal("expected missing table before upsert")
	}
	poText := []byte(`
msgctxt "web/a@new"
msgid "Hello"
msgstr "你好"
`)
	reg := store.NewRegistry(rs)
	stats, err := i18nimport.UpsertPackagedTerms(rs, reg, "auth", "auth", "zh_CN", poText)
	if err != nil {
		t.Fatalf("UpsertPackagedTerms: %v", err)
	}
	if stats.Upserted != 1 {
		t.Fatalf("stats=%+v", stats)
	}
	if !rs.Session().Migrator().HasTable("auth_translation_term") {
		t.Fatal("expected UpsertPackagedTerms to migrate missing table")
	}
	var row i18nmodels.TranslationTerm
	if err := rs.Session().Table("auth_translation_term").Where("src = ?", "Hello").Take(&row).Error; err != nil {
		t.Fatal(err)
	}
	if row.Value != "你好" {
		t.Fatalf("row=%+v", row)
	}
}

func TestUpsertPackagedTermsMigrateMissingTableError(t *testing.T) {
	rs := newTestScope(t)
	sqlDB, err := rs.Session().DB.DB()
	if err != nil {
		t.Fatalf("db.DB: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	poText := []byte(`
msgctxt "web/a@new"
msgid "Hello"
msgstr "你好"
`)
	_, err = i18nimport.UpsertPackagedTerms(rs, nil, "auth", "auth", "zh_CN", poText)
	if err == nil {
		t.Fatal("expected migrate error for missing table on closed DB")
	}
}

func TestUpsertPackagedTermsMsgctxtOverrideAndObsolete(t *testing.T) {
	rs := newTestScope(t)
	if err := i18nmodels.MigrateTranslationTermTable(rs, "auth"); err != nil {
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
	stats, err := i18nimport.UpsertPackagedTerms(rs, reg, "auth", "auth", "zh_CN", poText)
	if err != nil {
		t.Fatalf("UpsertPackagedTerms: %v", err)
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
	if err := i18nmodels.MigrateTranslationTermTable(rs, "auth"); err != nil {
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

func TestUpsertPackagedTermsMultilineMsgstr(t *testing.T) {
	rs := newTestScope(t)
	if err := i18nmodels.MigrateTranslationTermTable(rs, "auth"); err != nil {
		t.Fatal(err)
	}
	poText := []byte(`
msgctxt "web/a@body"
msgid ""
"Hello "
"world"
msgstr ""
"你好"
"世界"
`)
	reg := store.NewRegistry(rs)
	stats, err := i18nimport.UpsertPackagedTerms(rs, reg, "auth", "auth", "zh_CN", poText)
	if err != nil {
		t.Fatal(err)
	}
	if stats.Upserted != 1 {
		t.Fatalf("stats=%+v", stats)
	}
	var row i18nmodels.TranslationTerm
	if err := rs.Session().Table("auth_translation_term").Where("src = ?", "Hello world").Take(&row).Error; err != nil {
		t.Fatal(err)
	}
	if row.Value != "你好世界" {
		t.Fatalf("multiline msgstr = %q", row.Value)
	}
}

func TestUpsertPackagedTermsKindFromExtractedComment(t *testing.T) {
	rs := newTestScope(t)
	if err := i18nmodels.MigrateTranslationTermTable(rs, "auth"); err != nil {
		t.Fatal(err)
	}
	poText := []byte(`
msgid ""
msgstr "Language: zh_CN\n"

#. kind: custom_title
#: web/menu/menus.ts
msgctxt "web/menu/menus.ts@base.menu.company"
msgid "Company Management"
msgstr "公司管理"

#. kind: custom_option
msgctxt "service/models/bank_account.ts@Type.checking"
msgid "Checking"
msgstr "支票"

msgctxt "web/a@literal"
msgid "Hello"
msgstr "你好"
`)
	reg := store.NewRegistry(rs)
	stats, err := i18nimport.UpsertPackagedTerms(rs, reg, "auth", "auth", "zh_CN", poText)
	if err != nil {
		t.Fatalf("UpsertPackagedTerms: %v", err)
	}
	if stats.Upserted != 3 {
		t.Fatalf("stats=%+v", stats)
	}

	var title i18nmodels.TranslationTerm
	if err := rs.Session().Table("auth_translation_term").
		Where("src = ? AND kind = ?", "Company Management", "custom_title").
		Take(&title).Error; err != nil {
		t.Fatal(err)
	}
	if title.Value != "公司管理" {
		t.Fatalf("custom kind row: %+v", title)
	}

	var sel i18nmodels.TranslationTerm
	if err := rs.Session().Table("auth_translation_term").
		Where("src = ? AND kind = ?", "Checking", "custom_option").
		Take(&sel).Error; err != nil {
		t.Fatal(err)
	}

	var lit i18nmodels.TranslationTerm
	if err := rs.Session().Table("auth_translation_term").
		Where("src = ? AND kind = ?", "Hello", i18nmodels.KindLiteral).
		Take(&lit).Error; err != nil {
		t.Fatal(err)
	}

	// Same scope/src different kind can coexist.
	if err := rs.Session().Table("auth_translation_term").Create(&i18nmodels.TranslationTerm{
		Application: "auth", Module: "auth", Lang: "zh_CN",
		Scope: "web/a@literal", Src: "Hello", Value: "字段你好",
		Kind: "custom_label", Source: i18nmodels.SourcePackaged,
	}).Error; err != nil {
		t.Fatal(err)
	}
	_ = reg.StoreFor("auth").WarmLanguage("zh_CN")
	titleVal, ok := reg.Lookup("auth", "zh_CN", "web/menu/menus.ts@base.menu.company", "Company Management", "custom_title")
	if !ok || titleVal != "公司管理" {
		t.Fatalf("custom cache = %q ok=%v", titleVal, ok)
	}
	fieldVal, ok := reg.Lookup("auth", "zh_CN", "web/a@literal", "Hello", "custom_label")
	if !ok || fieldVal != "字段你好" {
		t.Fatalf("field cache = %q ok=%v", fieldVal, ok)
	}
	litVal, ok := reg.Lookup("auth", "zh_CN", "web/a@literal", "Hello", "")
	if !ok || litVal != "你好" {
		t.Fatalf("literal cache = %q ok=%v", litVal, ok)
	}
}

func TestUpsertPackagedTermsPurgesOnlyRetiredS7Kinds(t *testing.T) {
	rs := newTestScope(t)
	if err := i18nmodels.MigrateTranslationTermTable(rs, "auth"); err != nil {
		t.Fatal(err)
	}
	table := rs.Session().Table("auth_translation_term")
	retiredKinds := []string{"field_label", "selection_label", "menu", "route", "action"}
	rows := []i18nmodels.TranslationTerm{
		{
			Application: "auth", Module: "auth", Lang: "zh_CN",
			Scope: "web/view", Src: "Literal", Value: "字面量",
			Kind: i18nmodels.KindLiteral, Source: i18nmodels.SourcePackaged,
		},
		{
			Application: "auth", Module: "auth", Lang: "zh_CN",
			Scope: "future/metadata", Src: "Future", Value: "未来",
			Kind: "future_metadata", Source: i18nmodels.SourceOverride,
		},
		{
			Application: "auth", Module: "other", Lang: "zh_CN",
			Scope: "other/menu", Src: "Other", Value: "其他",
			Kind: "menu", Source: i18nmodels.SourcePackaged,
		},
	}
	for i, kind := range retiredKinds {
		source := i18nmodels.SourcePackaged
		if i%2 != 0 {
			source = i18nmodels.SourceOverride
		}
		rows = append(rows, i18nmodels.TranslationTerm{
			Application: "auth", Module: "auth", Lang: "zh_CN",
			Scope: "retired/" + kind, Src: "Retired " + kind, Value: "旧值",
			Kind: kind, Source: source,
		})
	}
	rows = append(rows, i18nmodels.TranslationTerm{
		Application: "auth", Module: "auth", Lang: "fr_FR",
		Scope: "retired/menu/fr", Src: "Retired menu", Value: "Ancien",
		Kind: "menu", Source: i18nmodels.SourceOverride,
	})
	if err := table.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}

	reg := store.NewRegistry(rs)
	reg.RememberModuleApplication("auth", "auth")
	if err := reg.StoreFor("auth").WarmLanguage("zh_CN"); err != nil {
		t.Fatal(err)
	}
	if err := reg.StoreFor("auth").WarmLanguage("fr_FR"); err != nil {
		t.Fatal(err)
	}

	poText := []byte(`
msgid ""
msgstr "Language: zh_CN\n"

msgctxt "web/new"
msgid "New"
msgstr "新的"

#. kind: menu
msgctxt "retired/reintroduced"
msgid "Reintroduced"
msgstr "不应保留"
`)
	stats, err := i18nimport.UpsertPackagedTerms(rs, reg, "auth", "auth", "zh_CN", poText)
	if err != nil {
		t.Fatalf("UpsertPackagedTerms: %v", err)
	}
	if stats.PurgedRetired != 7 {
		t.Fatalf("PurgedRetired = %d, want 7", stats.PurgedRetired)
	}

	var retiredCount int64
	if err := rs.Session().Table("auth_translation_term").
		Where("module = ? AND kind IN ?", "auth", retiredKinds).
		Count(&retiredCount).Error; err != nil {
		t.Fatal(err)
	}
	if retiredCount != 0 {
		t.Fatalf("retired S7 row count = %d, want 0", retiredCount)
	}
	for _, tc := range []struct {
		module string
		kind   string
		src    string
	}{
		{module: "auth", kind: i18nmodels.KindLiteral, src: "Literal"},
		{module: "auth", kind: "future_metadata", src: "Future"},
		{module: "other", kind: "menu", src: "Other"},
	} {
		var count int64
		if err := rs.Session().Table("auth_translation_term").
			Where("module = ? AND kind = ? AND src = ?", tc.module, tc.kind, tc.src).
			Count(&count).Error; err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Fatalf("preserved row (%s, %s, %s) count = %d, want 1", tc.module, tc.kind, tc.src, count)
		}
	}
	if value, ok := reg.Lookup("auth", "zh_CN", "future/metadata", "Future", "future_metadata"); !ok || value != "未来" {
		t.Fatalf("future nonliteral cache = %q ok=%v", value, ok)
	}
	if _, ok := reg.Lookup("auth", "fr_FR", "retired/menu/fr", "Retired menu", "menu"); ok {
		t.Fatal("retired override remained in warmed fr_FR cache")
	}
	if _, ok := reg.Lookup("auth", "zh_CN", "retired/reintroduced", "Reintroduced", "menu"); ok {
		t.Fatal("retired kind from imported PO remained in cache")
	}
}

func TestUpsertPackagedTermsUsesCallerTransaction(t *testing.T) {
	rs := newTestScope(t)
	if err := i18nmodels.MigrateTranslationTermTable(rs, "auth"); err != nil {
		t.Fatal(err)
	}
	table := rs.Session().Table("auth_translation_term")
	if err := table.Create(&[]i18nmodels.TranslationTerm{
		{
			Application: "auth", Module: "auth", Lang: "zh_CN",
			Scope: "retired/menu", Src: "Retired", Value: "旧值",
			Kind: "menu", Source: i18nmodels.SourceOverride,
		},
		{
			Application: "auth", Module: "auth", Lang: "zh_CN",
			Scope: "literal/missing", Src: "Keep", Value: "保留",
			Kind: i18nmodels.KindLiteral, Source: i18nmodels.SourcePackaged,
		},
	}).Error; err != nil {
		t.Fatal(err)
	}

	tx := rs.Session().Begin()
	if tx.Error != nil {
		t.Fatal(tx.Error)
	}
	txScope := &testScope{
		ctx:     rs.ctx,
		logger:  rs.logger,
		session: &scope.Session{DB: tx},
	}
	stats, err := i18nimport.UpsertPackagedTerms(txScope, nil, "auth", "auth", "zh_CN", []byte(`
msgctxt "literal/new"
msgid "New"
msgstr "新的"
`))
	if err != nil {
		_ = tx.Rollback()
		t.Fatalf("UpsertPackagedTerms: %v", err)
	}
	if stats.PurgedRetired != 1 {
		_ = tx.Rollback()
		t.Fatalf("PurgedRetired = %d, want 1", stats.PurgedRetired)
	}
	var inTxRetired int64
	if err := tx.Table("auth_translation_term").Where("kind = ?", "menu").Count(&inTxRetired).Error; err != nil {
		_ = tx.Rollback()
		t.Fatal(err)
	}
	if inTxRetired != 0 {
		_ = tx.Rollback()
		t.Fatalf("retired rows inside transaction = %d, want 0", inTxRetired)
	}
	if err := tx.Rollback().Error; err != nil {
		t.Fatal(err)
	}

	var retiredAfterRollback, newAfterRollback, literalAfterRollback int64
	if err := rs.Session().Table("auth_translation_term").
		Where("kind = ?", "menu").Count(&retiredAfterRollback).Error; err != nil {
		t.Fatal(err)
	}
	if err := rs.Session().Table("auth_translation_term").
		Where("scope = ?", "literal/new").Count(&newAfterRollback).Error; err != nil {
		t.Fatal(err)
	}
	if err := rs.Session().Table("auth_translation_term").
		Where("scope = ?", "literal/missing").Count(&literalAfterRollback).Error; err != nil {
		t.Fatal(err)
	}
	if retiredAfterRollback != 1 || newAfterRollback != 0 || literalAfterRollback != 1 {
		t.Fatalf(
			"rollback counts retired=%d new=%d literal=%d, want 1/0/1",
			retiredAfterRollback, newAfterRollback, literalAfterRollback,
		)
	}
}

func TestUpsertPackagedTermsSkipsCoreAndImportModuleI18nDir(t *testing.T) {
	rs := newTestScope(t)
	if err := i18nmodels.MigrateTranslationTermTable(rs, "auth"); err != nil {
		t.Fatal(err)
	}
	reg := store.NewRegistry(rs)

	stats, err := i18nimport.UpsertPackagedTerms(rs, reg, "core", "auth", "zh_CN", []byte(`msgid "x"`))
	if err != nil || stats == nil || stats.Upserted != 0 {
		t.Fatalf("core skip stats=%#v err=%v", stats, err)
	}

	root := t.TempDir()
	i18nDir := filepath.Join(root, "i18n")
	if err := os.MkdirAll(i18nDir, 0o755); err != nil {
		t.Fatal(err)
	}
	poBody := []byte(`msgid ""
msgstr ""

msgctxt "a@b"
msgid "Hello"
msgstr "你好"
`)
	if err := os.WriteFile(filepath.Join(i18nDir, "zh_CN.po"), poBody, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(i18nDir, "en.po"), poBody, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(i18nDir, "readme.txt"), []byte("skip"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := i18nimport.ImportModuleI18nDir(rs, reg, "auth", "auth", root); err != nil {
		t.Fatalf("ImportModuleI18nDir: %v", err)
	}
	var count int64
	if err := rs.Session().Table("auth_translation_term").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count < 2 {
		t.Fatalf("expected terms from both po files, count=%d", count)
	}
	if err := i18nimport.ImportModuleI18nDir(rs, reg, "auth", "auth", filepath.Join(root, "missing")); err != nil {
		t.Fatalf("missing i18n dir should no-op: %v", err)
	}
}

func TestUpsertPackagedTermsUpdatesExistingPackaged(t *testing.T) {
	rs := newTestScope(t)
	if err := i18nmodels.MigrateTranslationTermTable(rs, "auth"); err != nil {
		t.Fatal(err)
	}
	if err := rs.Session().Table("auth_translation_term").Create(&i18nmodels.TranslationTerm{
		Application: "auth", Module: "auth", Lang: "zh_CN",
		Scope: "a@t", Src: "Hello", Value: "旧",
		Kind: i18nmodels.KindLiteral, Source: i18nmodels.SourcePackaged,
	}).Error; err != nil {
		t.Fatal(err)
	}
	reg := store.NewRegistry(rs)
	stats, err := i18nimport.UpsertPackagedTerms(rs, reg, "auth", "auth", "zh_CN", []byte(`
#, fuzzy
msgctxt "a@t"
msgid "Hello"
msgstr "新值"
`))
	if err != nil {
		t.Fatal(err)
	}
	if stats.Upserted != 1 {
		t.Fatalf("stats=%+v", stats)
	}
	var row i18nmodels.TranslationTerm
	if err := rs.Session().Table("auth_translation_term").Where("src = ?", "Hello").Take(&row).Error; err != nil {
		t.Fatal(err)
	}
	if row.Value != "新值" || row.Source != i18nmodels.SourcePackaged {
		t.Fatalf("row=%+v", row)
	}
	val, ok := reg.Lookup("auth", "zh_CN", "a@t", "Hello", "")
	if !ok || val != "新值" {
		t.Fatalf("cache = %q ok=%v", val, ok)
	}
}

func TestDeleteModuleTermsNoops(t *testing.T) {
	rs := newTestScope(t)
	reg := store.NewRegistry(rs)
	if err := i18nimport.DeleteModuleTerms(rs, reg, "core", "auth"); err != nil {
		t.Fatal(err)
	}
	if err := i18nimport.DeleteModuleTerms(rs, reg, "", "auth"); err != nil {
		t.Fatal(err)
	}
	if err := i18nimport.DeleteModuleTerms(rs, reg, "auth", ""); err != nil {
		t.Fatal(err)
	}
	if err := i18nimport.DeleteModuleTerms(nil, reg, "auth", "auth"); err == nil {
		t.Fatal("expected missing session error")
	}
	// No table yet — should no-op.
	if err := i18nimport.DeleteModuleTerms(rs, reg, "auth", "auth"); err != nil {
		t.Fatal(err)
	}
}

func TestUpsertPackagedTermsMissingSession(t *testing.T) {
	reg := store.NewRegistry(nil)
	if _, err := i18nimport.UpsertPackagedTerms(nil, reg, "auth", "auth", "zh_CN", []byte(`msgctxt "a" msgid "x" msgstr "y"`)); err == nil {
		t.Fatal("expected missing session")
	}
}
