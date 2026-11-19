// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package store

import (
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"testing"

	i18nmodels "github.com/choysum-dev/choysum/internal/i18n/models"
	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/scope"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type registryCoverageScope struct {
	ctx     context.Context
	logger  *slog.Logger
	session *scope.Session
}

func (s *registryCoverageScope) Run(fn func(scope.Scope) error) error { return fn(s) }
func (s *registryCoverageScope) Transactor() scope.Transactor {
	return scopetest.NewPassthroughTransactor(s)
}
func (s *registryCoverageScope) Session() *scope.Session { return s.session }
func (s *registryCoverageScope) WithContext(ctx context.Context) scope.Scope {
	if ctx == nil {
		ctx = s.ctx
	}
	return &registryCoverageScope{ctx: ctx, logger: s.logger, session: s.session}
}
func (s *registryCoverageScope) Context() context.Context {
	if s.ctx != nil {
		return s.ctx
	}
	return context.Background()
}
func (s *registryCoverageScope) Logger() *slog.Logger { return s.logger }

func TestRegistryApplicationForModuleMissWithoutWarm(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "registry-no-warm.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	rs := &registryCoverageScope{
		ctx:     context.Background(),
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		session: &scope.Session{DB: db},
	}
	reg := NewRegistry(rs)

	if app, ok := reg.ApplicationForModule("auth"); ok || app != "" {
		t.Fatalf("ApplicationForModule() = %q ok=%v, want miss without Remember/Warm", app, ok)
	}
}

func TestRegistryWarmRuntimeCache(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "warm-runtime.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	rs := &registryCoverageScope{
		ctx:     context.Background(),
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		session: &scope.Session{DB: db},
	}
	if err := db.Exec(`CREATE TABLE meta_module (
		id text primary key, name text, application_str text, status text, deleted_at datetime
	)`).Error; err != nil {
		t.Fatalf("create meta_module: %v", err)
	}
	if err := db.Exec(`CREATE TABLE base_language (
		id text primary key, code text, deleted_at datetime
	)`).Error; err != nil {
		t.Fatalf("create base_language: %v", err)
	}
	if err := i18nmodels.MigrateTranslationTermTable(rs, "auth"); err != nil {
		t.Fatalf("ensure term table: %v", err)
	}
	if err := db.Exec(
		`INSERT INTO meta_module (id, name, application_str, status, deleted_at) VALUES
			('1','auth','auth','installed', NULL),
			('2','gone','gone','installed', '2020-01-01')`,
	).Error; err != nil {
		t.Fatalf("seed module: %v", err)
	}
	if err := db.Exec(
		`INSERT INTO base_language (id, code, deleted_at) VALUES
			('1','zh_CN', NULL),
			('2','de_DE', '2020-01-01')`,
	).Error; err != nil {
		t.Fatalf("seed language: %v", err)
	}
	if err := rs.Session().Table("auth_translation_term").Create(&i18nmodels.TranslationTerm{
		Application: "auth", Module: "auth", Lang: "zh_CN",
		Scope: "a@b", Src: "Hello", Value: "你好",
		Kind: i18nmodels.KindLiteral, Source: i18nmodels.SourcePackaged,
	}).Error; err != nil {
		t.Fatalf("seed term: %v", err)
	}
	// Term-only language must be discovered without a matching base_language row.
	if err := rs.Session().Table("auth_translation_term").Create(&i18nmodels.TranslationTerm{
		Application: "auth", Module: "auth", Lang: "fr_FR",
		Scope: "a@b", Src: "Hello", Value: "Bonjour",
		Kind: i18nmodels.KindLiteral, Source: i18nmodels.SourcePackaged,
	}).Error; err != nil {
		t.Fatalf("seed fr term: %v", err)
	}

	reg := NewRegistry(rs)
	reg.RememberModuleApplication("stale", "stale")
	if err := reg.WarmRuntimeCache(context.Background()); err != nil {
		t.Fatalf("WarmRuntimeCache: %v", err)
	}
	// Drop meta_module so ApplicationForModule cannot fall back to DB.
	if err := db.Migrator().DropTable("meta_module"); err != nil {
		t.Fatalf("drop meta_module: %v", err)
	}
	app, ok := reg.ApplicationForModule("auth")
	if !ok || app != "auth" {
		t.Fatalf("ApplicationForModule(auth) = %q ok=%v", app, ok)
	}
	if _, ok := reg.ApplicationForModule("gone"); ok {
		t.Fatal("soft-deleted installed module must not be prefetched")
	}
	if _, ok := reg.ApplicationForModule("stale"); ok {
		t.Fatal("warm must rebuild moduleToApp and drop stale Remember entries")
	}
	val, ok := reg.Lookup("auth", "zh_CN", "a@b", "Hello", "")
	if !ok || val != "你好" {
		t.Fatalf("Lookup after WarmRuntimeCache = %q ok=%v", val, ok)
	}
	fr, ok := reg.Lookup("auth", "fr_FR", "a@b", "Hello", "")
	if !ok || fr != "Bonjour" {
		t.Fatalf("Lookup fr_FR (term-only lang) = %q ok=%v", fr, ok)
	}
	if _, ok := reg.Lookup("unprefilled", "zh_CN", "a@b", "Hello", ""); ok {
		t.Fatal("Lookup for module never prefetched must miss without DB")
	}
	reg.replaceModuleApplications(nil)
	if _, ok := reg.ApplicationForModule("auth"); ok {
		t.Fatal("replaceModuleApplications(nil) must clear the map")
	}
}

func TestWarmRuntimeCacheNilAndErrorPaths(t *testing.T) {
	if err := (*Registry)(nil).WarmRuntimeCache(context.Background()); err != nil {
		t.Fatalf("nil registry: %v", err)
	}
	if err := NewRegistry(nil).WarmRuntimeCache(context.Background()); err != nil {
		t.Fatalf("nil scope: %v", err)
	}
	emptySess := &registryCoverageScope{
		ctx:     context.Background(),
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		session: &scope.Session{},
	}
	if err := NewRegistry(emptySess).WarmRuntimeCache(context.Background()); err != nil {
		t.Fatalf("nil session.DB: %v", err)
	}

	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "warm-errors.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	rs := &registryCoverageScope{
		ctx:     context.Background(),
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		session: &scope.Session{DB: db},
	}

	// meta_module exists but missing application_str → installed-module Find fails.
	if err := db.Exec(`CREATE TABLE meta_module (id text primary key, name text, status text)`).Error; err != nil {
		t.Fatalf("create meta_module: %v", err)
	}
	if err := NewRegistry(rs).WarmRuntimeCache(context.Background()); err == nil {
		t.Fatal("expected list installed modules error")
	}
	if err := db.Migrator().DropTable("meta_module"); err != nil {
		t.Fatalf("drop meta_module: %v", err)
	}

	// base_language without code → language Pluck fails.
	if err := db.Exec(`CREATE TABLE base_language (id text primary key)`).Error; err != nil {
		t.Fatalf("create base_language: %v", err)
	}
	if err := NewRegistry(rs).WarmRuntimeCache(context.Background()); err == nil {
		t.Fatal("expected list language codes error")
	}
	if err := db.Migrator().DropTable("base_language"); err != nil {
		t.Fatalf("drop base_language: %v", err)
	}

	// Term-table lang Pluck fails → firstErr with empty langs returns at len==0.
	if err := db.Exec(`CREATE TABLE meta_module (
		id text primary key, name text, application_str text, status text
	)`).Error; err != nil {
		t.Fatalf("recreate meta_module: %v", err)
	}
	if err := db.Exec(
		`INSERT INTO meta_module (id, name, application_str, status) VALUES ('1','auth','auth','installed')`,
	).Error; err != nil {
		t.Fatalf("seed module: %v", err)
	}
	if err := db.Exec(`CREATE TABLE auth_translation_term (id text primary key)`).Error; err != nil {
		t.Fatalf("create broken term table: %v", err)
	}
	if err := NewRegistry(rs).WarmRuntimeCache(context.Background()); err == nil {
		t.Fatal("expected list term langs / empty langs firstErr")
	}
	if err := db.Migrator().DropTable("auth_translation_term"); err != nil {
		t.Fatalf("drop term table: %v", err)
	}

	// WarmLanguage failure surfaces as firstErr after langs are collected.
	if err := db.Exec(`CREATE TABLE base_language (id text primary key, code text)`).Error; err != nil {
		t.Fatalf("recreate base_language: %v", err)
	}
	if err := db.Exec(`INSERT INTO base_language (id, code) VALUES ('1','zh_CN')`).Error; err != nil {
		t.Fatalf("seed language: %v", err)
	}
	if err := i18nmodels.MigrateTranslationTermTable(rs, "auth"); err != nil {
		t.Fatalf("ensure term table: %v", err)
	}
	reg := NewRegistry(rs)
	ts := reg.StoreFor("auth")
	prev := warmAfterLoadHook
	t.Cleanup(func() { warmAfterLoadHook = prev })
	warmAfterLoadHook = func(lang string) {
		ts.mu.Lock()
		ts.warmEpoch[lang]++
		ts.mu.Unlock()
	}
	if err := reg.WarmRuntimeCache(context.Background()); err == nil {
		t.Fatal("expected WarmLanguage failure from WarmRuntimeCache")
	}
}

func TestRegistryExistingStore(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "existing-store.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	rs := &registryCoverageScope{
		ctx:     context.Background(),
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		session: &scope.Session{DB: db},
	}
	reg := NewRegistry(rs)

	if _, ok := reg.ExistingStore(""); ok {
		t.Fatal("ExistingStore('') should be false")
	}
	if _, ok := reg.ExistingStore("auth"); ok {
		t.Fatal("ExistingStore before StoreFor should be false")
	}
	created := reg.StoreFor("auth")
	got, ok := reg.ExistingStore("auth")
	if !ok || got != created {
		t.Fatalf("ExistingStore(auth) ok=%v store=%p want %p", ok, got, created)
	}
	got, ok = reg.ExistingStore("  auth  ")
	if !ok || got != created {
		t.Fatalf("ExistingStore trimmed ok=%v", ok)
	}
}
