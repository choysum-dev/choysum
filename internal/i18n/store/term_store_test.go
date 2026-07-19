// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package store_test

import (
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"testing"

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
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "store.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	return &testScope{
		ctx:     context.Background(),
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		session: &scope.Session{DB: db},
	}
}

func TestTermStoreWarmLookupInvalidate(t *testing.T) {
	rs := newTestScope(t)
	if err := i18nmodels.EnsureTranslationTermTable(rs, "auth"); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	table := rs.Session().Table("auth_translation_term")
	if err := table.Create(&i18nmodels.TranslationTerm{
		Application: "auth",
		Module:      "auth",
		Lang:        "zh_CN",
		Scope:       "web/pages/Login@title",
		Src:         "Sign in",
		Value:       "登录",
		Kind:        i18nmodels.KindLiteral,
		Source:      i18nmodels.SourcePackaged,
	}).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	ts := store.NewTermStore(rs, "auth")
	if _, ok := ts.Lookup("auth", "zh_CN", "web/pages/Login@title", "Sign in", ""); ok {
		t.Fatal("expected miss before warm")
	}
	if err := ts.WarmLanguage("zh_CN"); err != nil {
		t.Fatalf("warm: %v", err)
	}
	val, ok := ts.Lookup("auth", "zh_CN", "web/pages/Login@title", "Sign in", "")
	if !ok || val != "登录" {
		t.Fatalf("Lookup = %q ok=%v", val, ok)
	}
	hash := ts.TermHash("zh_CN")
	if hash == "" {
		t.Fatal("expected non-empty termHash after warm")
	}

	ts.InvalidateModule("auth")
	if _, ok := ts.Lookup("auth", "zh_CN", "web/pages/Login@title", "Sign in", ""); ok {
		t.Fatal("expected miss after InvalidateModule")
	}
	if ts.TermHash("zh_CN") == hash {
		t.Fatal("expected termHash bump after InvalidateModule")
	}

	ts.EvictLanguage("zh_CN")
	if ts.TermHash("zh_CN") != "" {
		t.Fatal("expected empty hash after EvictLanguage")
	}
}

func TestRegistryLookupViaModuleMapping(t *testing.T) {
	rs := newTestScope(t)
	if err := i18nmodels.EnsureTranslationTermTable(rs, "auth"); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	if err := rs.Session().Table("auth_translation_term").Create(&i18nmodels.TranslationTerm{
		Application: "auth",
		Module:      "auth",
		Lang:        "zh_CN",
		Scope:       "a@b",
		Src:         "Hello",
		Value:       "你好",
		Kind:        i18nmodels.KindLiteral,
		Source:      i18nmodels.SourcePackaged,
	}).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	reg := store.NewRegistry(rs)
	reg.RememberModuleApplication("auth", "auth")
	if err := reg.StoreFor("auth").WarmLanguage("zh_CN"); err != nil {
		t.Fatalf("warm: %v", err)
	}
	val, ok := reg.Lookup("auth", "zh_CN", "a@b", "Hello", "")
	if !ok || val != "你好" {
		t.Fatalf("Registry.Lookup = %q ok=%v", val, ok)
	}
}

func TestRegistryLookupFrameworkModuleInHostStore(t *testing.T) {
	rs := newTestScope(t)
	if err := i18nmodels.EnsureTranslationTermTable(rs, "auth"); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	if err := rs.Session().Table("auth_translation_term").Create(&i18nmodels.TranslationTerm{
		Application: "auth",
		Module:      "core",
		Lang:        "zh_CN",
		Scope:       "service/a@m",
		Src:         "Denied",
		Value:       "拒绝",
		Kind:        i18nmodels.KindLiteral,
		Source:      i18nmodels.SourcePackaged,
	}).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	reg := store.NewRegistry(rs)
	reg.RememberModuleApplication("auth", "auth")
	reg.RememberModuleApplication("core", "core") // IrModule sentinel; must not block host lookup
	if err := reg.StoreFor("auth").WarmLanguage("zh_CN"); err != nil {
		t.Fatalf("warm: %v", err)
	}
	val, ok := reg.Lookup("core", "zh_CN", "service/a@m", "Denied", "")
	if !ok || val != "拒绝" {
		t.Fatalf("framework Lookup = %q ok=%v", val, ok)
	}
}

func TestTermStoreExplicitKindAndTermsByModulesLiteralOnly(t *testing.T) {
	rs := newTestScope(t)
	if err := i18nmodels.EnsureTranslationTermTable(rs, "auth"); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	table := rs.Session().Table("auth_translation_term")
	rows := []i18nmodels.TranslationTerm{
		{
			Application: "auth", Module: "auth", Lang: "zh_CN",
			Scope: "web/menu/menus.ts@base.menu.company", Src: "Company Management", Value: "公司管理",
			Kind: "custom_title", Source: i18nmodels.SourcePackaged,
		},
		{
			Application: "auth", Module: "auth", Lang: "zh_CN",
			Scope: "web/pages/Login@title", Src: "Sign in", Value: "登录",
			Kind: i18nmodels.KindLiteral, Source: i18nmodels.SourcePackaged,
		},
		{
			Application: "auth", Module: "auth", Lang: "zh_CN",
			Scope: "web/pages/Login@title", Src: "Sign in", Value: "登录字段",
			Kind: "custom_label", Source: i18nmodels.SourcePackaged,
		},
	}
	for i := range rows {
		if err := table.Create(&rows[i]).Error; err != nil {
			t.Fatalf("seed %d: %v", i, err)
		}
	}

	ts := store.NewTermStore(rs, "auth")
	if err := ts.WarmLanguage("zh_CN"); err != nil {
		t.Fatalf("warm: %v", err)
	}

	titleVal, ok := ts.Lookup("auth", "zh_CN", "web/menu/menus.ts@base.menu.company", "Company Management", "custom_title")
	if !ok || titleVal != "公司管理" {
		t.Fatalf("custom kind Lookup = %q ok=%v", titleVal, ok)
	}
	fieldVal, ok := ts.Lookup("auth", "zh_CN", "web/pages/Login@title", "Sign in", "custom_label")
	if !ok || fieldVal != "登录字段" {
		t.Fatalf("custom label Lookup = %q ok=%v", fieldVal, ok)
	}
	litVal, ok := ts.Lookup("auth", "zh_CN", "web/pages/Login@title", "Sign in", "")
	if !ok || litVal != "登录" {
		t.Fatalf("literal Lookup = %q ok=%v", litVal, ok)
	}

	exported := ts.TermsByModules("zh_CN", []string{"auth"})
	if got := exported["auth"]["web/pages/Login@title"]["Sign in"]; got != "登录" {
		t.Fatalf("TermsByModules literal = %q", got)
	}
	if _, hasCustom := exported["auth"]["web/menu/menus.ts@base.menu.company"]; hasCustom {
		t.Fatal("TermsByModules must omit explicit non-literal kinds")
	}
}


func TestTermStoreSearchTermsAndUpsertOverride(t *testing.T) {
	rs := newTestScope(t)
	if err := i18nmodels.EnsureTranslationTermTable(rs, "auth"); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	table := rs.Session().Table("auth_translation_term")
	for _, row := range []i18nmodels.TranslationTerm{
		{Application: "auth", Module: "auth", Lang: "zh_CN", Scope: "a@one", Src: "Alpha", Value: "甲", Kind: i18nmodels.KindLiteral, Source: i18nmodels.SourcePackaged},
		{Application: "auth", Module: "auth", Lang: "zh_CN", Scope: "a@two", Src: "Beta", Value: "乙", Kind: i18nmodels.KindLiteral, Source: i18nmodels.SourcePackaged},
		{Application: "auth", Module: "web", Lang: "zh_CN", Scope: "w@x", Src: "Gamma", Value: "丙", Kind: i18nmodels.KindLiteral, Source: i18nmodels.SourcePackaged},
	} {
		if err := table.Create(&row).Error; err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	ts := store.NewTermStore(rs, "auth")
	items, total, err := ts.SearchTerms("zh_CN", nil, "Alpha", 10, 0)
	if err != nil || total != 1 || len(items) != 1 || items[0].Src != "Alpha" {
		t.Fatalf("q filter items=%#v total=%d err=%v", items, total, err)
	}
	items, total, err = ts.SearchTerms("zh_CN", []string{"web"}, "", 10, 0)
	if err != nil || total != 1 || items[0].Module != "web" {
		t.Fatalf("module filter items=%#v total=%d err=%v", items, total, err)
	}
	items, total, err = ts.SearchTerms("zh_CN", nil, "", 1, 1)
	if err != nil || total != 3 || len(items) != 1 {
		t.Fatalf("pagination items=%#v total=%d err=%v", items, total, err)
	}
	if _, _, err := ts.SearchTerms("", nil, "", 10, 0); err != nil {
		t.Fatalf("empty lang should no-op: %v", err)
	}

	created, err := ts.UpsertOverride("auth", "zh_CN", "a@new", "Fresh", "", "新")
	if err != nil {
		t.Fatalf("UpsertOverride create: %v", err)
	}
	if created.Value != "新" || created.Source != i18nmodels.SourceOverride {
		t.Fatalf("created = %#v", created)
	}
	val, ok := ts.Lookup("auth", "zh_CN", "a@new", "Fresh", "")
	if !ok || val != "新" {
		t.Fatalf("Lookup after create = %q ok=%v", val, ok)
	}
	if _, err := ts.UpsertOverride("", "zh_CN", "a", "s", "", "v"); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestEmptyTermHashAndApplication(t *testing.T) {
	rs := newTestScope(t)
	ts := store.NewTermStore(rs, "auth")
	if ts.Application() != "auth" {
		t.Fatalf("Application = %q", ts.Application())
	}
	hash := store.EmptyTermHash()
	if hash == "" || hash != store.EmptyTermHash() {
		t.Fatalf("EmptyTermHash unstable: %q", hash)
	}
	store.ResetSharedRegistryForTests()
	reg := store.RegistryFor(rs)
	if reg == nil {
		t.Fatal("RegistryFor nil")
	}
	if _, ok := reg.ApplicationForModule(""); ok {
		t.Fatal("empty module should miss")
	}
}
