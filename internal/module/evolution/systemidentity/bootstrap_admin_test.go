// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package systemidentity

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"

	modmeta "github.com/choysum-dev/choysum/internal/module/meta"
	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/rs/xid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

type identityTestScope struct {
	ctx     context.Context
	session *scope.Session
	cfg     *config.Config
	logger  *slog.Logger
}

func (s *identityTestScope) Run(fn func(scope.Scope) error) error { return fn(s) }
func (s *identityTestScope) Transactor() scope.Transactor {
	return scopetest.NewPassthroughTransactor(s)
}
func (s *identityTestScope) Session() *scope.Session { return s.session }
func (s *identityTestScope) WithContext(ctx context.Context) scope.Scope {
	clone := *s
	clone.ctx = ctx
	return &clone
}
func (s *identityTestScope) Context() context.Context {
	if s != nil && s.ctx != nil {
		return s.ctx
	}
	return context.Background()
}
func (s *identityTestScope) Logger() *slog.Logger {
	if s != nil && s.logger != nil {
		return s.logger
	}
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
func (s *identityTestScope) Config() *config.Config {
	if s != nil && s.cfg != nil {
		return s.cfg
	}
	return &config.Config{}
}
func (s *identityTestScope) FactoryInput() scope.FactoryInput {
	return scopetest.FactoryInputFromConfig(s.Config())
}

func openDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "sysid.sqlite")), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	return db
}

func newScope(t *testing.T, db *gorm.DB) *identityTestScope {
	t.Helper()
	return &identityTestScope{
		ctx:     context.Background(),
		session: &scope.Session{DB: db},
		cfg:     &config.Config{Db: &config.DbConfig{Dialect: "sqlite"}},
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

func seedBootstrapAdmin(t *testing.T, db *gorm.DB, userID, companyID string) {
	t.Helper()
	if err := db.AutoMigrate(&meta.Module{}, &modmeta.ModelData{}, &meta.Model{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	if err := db.Create(&meta.Module{Name: "auth", Status: meta.Installed, Version: "1.0.0"}).Error; err != nil {
		t.Fatalf("auth module: %v", err)
	}
	if err := db.Create(&modmeta.ModelData{
		Module: "auth", Name: "user_admin", Application: "auth",
		ModelName: "User", ModelId: xid.New().String(), ResID: userID,
	}).Error; err != nil {
		t.Fatalf("user_admin mapping: %v", err)
	}
	table := "auth_user_sysid"
	if err := db.Create(&meta.Model{
		BaseModel:   meta.BaseModel{Id: sql.NullString{String: xid.New().String(), Valid: true}},
		Name:        "User",
		Application: "auth",
		ModelTable:  table,
		Path:        "@/auth/service/models/user.ts",
	}).Error; err != nil {
		t.Fatalf("auth.User model: %v", err)
	}
	if err := db.Exec(`CREATE TABLE "` + table + `" (id TEXT PRIMARY KEY, company_id TEXT)`).Error; err != nil {
		t.Fatalf("user table: %v", err)
	}
	if err := db.Exec(`INSERT INTO "`+table+`" (id, company_id) VALUES (?, ?)`, userID, companyID).Error; err != nil {
		t.Fatalf("insert user: %v", err)
	}
}

func TestResolveBootstrapAdminNilScope(t *testing.T) {
	t.Parallel()
	_, ok, err := ResolveBootstrapAdmin(context.Background(), nil)
	if err != nil || ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
}

func TestResolveBootstrapAdminMissingAuth(t *testing.T) {
	t.Parallel()
	db := openDB(t)
	_, ok, err := ResolveBootstrapAdmin(context.Background(), newScope(t, db))
	if err != nil || ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
}

func TestResolveBootstrapAdminSuccess(t *testing.T) {
	t.Parallel()
	db := openDB(t)
	userID := xid.New().String()
	companyID := xid.New().String()
	seedBootstrapAdmin(t, db, userID, companyID)

	got, ok, err := ResolveBootstrapAdmin(context.Background(), newScope(t, db))
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	if got.UserID != userID || got.CompanyID != companyID {
		t.Fatalf("got %#v, want user=%s company=%s", got, userID, companyID)
	}
}

func TestResolvePrincipalUserIDPrefersSession(t *testing.T) {
	t.Parallel()
	userID, companyID, err := ResolvePrincipalUserID(context.Background(), nil, "session-u")
	if err != nil {
		t.Fatal(err)
	}
	if userID != "session-u" || companyID != "" {
		t.Fatalf("user=%q company=%q", userID, companyID)
	}
}

func TestResolvePrincipalUserIDSoftWhenAuthAbsent(t *testing.T) {
	t.Parallel()
	userID, companyID, err := ResolvePrincipalUserID(context.Background(), nil, "")
	if err != nil {
		t.Fatalf("auth-free scope should soft-empty, got err=%v", err)
	}
	if userID != "" || companyID != "" {
		t.Fatalf("user=%q company=%q, want empty", userID, companyID)
	}
}

func TestResolvePrincipalUserIDUnavailableWhenAuthInstalledWithoutSeed(t *testing.T) {
	t.Parallel()
	db := openDB(t)
	if err := db.AutoMigrate(&meta.Module{}); err != nil {
		t.Fatalf("migrate module: %v", err)
	}
	if err := db.Create(&meta.Module{Name: "auth", Status: meta.Installed, Version: "1.0.0"}).Error; err != nil {
		t.Fatalf("seed auth module: %v", err)
	}
	_, _, err := ResolvePrincipalUserID(context.Background(), newScope(t, db), "")
	if !errors.Is(err, ErrBootstrapAdminUnavailable) {
		t.Fatalf("err=%v, want ErrBootstrapAdminUnavailable", err)
	}
}

func TestResolvePrincipalUserIDFromBootstrap(t *testing.T) {
	t.Parallel()
	db := openDB(t)
	userID := xid.New().String()
	companyID := xid.New().String()
	seedBootstrapAdmin(t, db, userID, companyID)

	gotUser, gotCompany, err := ResolvePrincipalUserID(context.Background(), newScope(t, db), "")
	if err != nil {
		t.Fatal(err)
	}
	if gotUser != userID || gotCompany != companyID {
		t.Fatalf("user=%q company=%q", gotUser, gotCompany)
	}
}

func TestCompanyContextPayload(t *testing.T) {
	t.Parallel()
	if len(CompanyContextPayload("")) != 0 {
		t.Fatal("expected empty map")
	}
	got := CompanyContextPayload("c1")
	if got["activeCompanyId"] != "c1" {
		t.Fatalf("%#v", got)
	}
	ids, _ := got["enabledCompanyIds"].([]string)
	if len(ids) != 1 || ids[0] != "c1" {
		t.Fatalf("%#v", got)
	}
}

type nilTransactorIdentityScope struct {
	*identityTestScope
}

func (s *nilTransactorIdentityScope) Transactor() scope.Transactor { return nil }

func (s *nilTransactorIdentityScope) WithContext(ctx context.Context) scope.Scope {
	out := *s
	if s.identityTestScope != nil {
		inner := *s.identityTestScope
		inner.ctx = ctx
		out.identityTestScope = &inner
	}
	return &out
}

type nilSessionIdentityScope struct {
	*identityTestScope
}

func (s *nilSessionIdentityScope) Session() *scope.Session { return nil }

func (s *nilSessionIdentityScope) WithContext(ctx context.Context) scope.Scope {
	out := *s
	if s.identityTestScope != nil {
		inner := *s.identityTestScope
		inner.ctx = ctx
		out.identityTestScope = &inner
	}
	return &out
}

type nilSessionInCallbackTransactor struct {
	root scope.Scope
}

func (t nilSessionInCallbackTransactor) Do(ctx context.Context, opts scope.TransactionOptions, fn scope.TxFunc) error {
	return t.Required(ctx, fn)
}

func (t nilSessionInCallbackTransactor) Required(ctx context.Context, fn scope.TxFunc) error {
	return fn(&nilSessionCallbackScope{Scope: t.root}, nil)
}

func (t nilSessionInCallbackTransactor) RequiresNew(ctx context.Context, fn scope.TxFunc) error {
	return t.Required(ctx, fn)
}

func (t nilSessionInCallbackTransactor) Nested(ctx context.Context, fn scope.TxFunc) error {
	return t.Required(ctx, fn)
}

type nilSessionCallbackScope struct {
	scope.Scope
}

func (s *nilSessionCallbackScope) Session() *scope.Session { return nil }

type nilDBSessionIdentityScope struct {
	*identityTestScope
}

func (s *nilDBSessionIdentityScope) Session() *scope.Session { return &scope.Session{} }

type nilWithContextIdentityScope struct {
	*identityTestScope
}

func (s *nilWithContextIdentityScope) WithContext(context.Context) scope.Scope { return nil }

type errorRequiredTransactor struct {
	err error
}

func (t errorRequiredTransactor) Do(ctx context.Context, _ scope.TransactionOptions, fn scope.TxFunc) error {
	return t.Required(ctx, fn)
}

func (t errorRequiredTransactor) Required(ctx context.Context, fn scope.TxFunc) error {
	if t.err != nil {
		return t.err
	}
	return fmt.Errorf("unexpected Required call")
}

func (t errorRequiredTransactor) RequiresNew(ctx context.Context, fn scope.TxFunc) error {
	return t.Required(ctx, fn)
}

func (t errorRequiredTransactor) Nested(ctx context.Context, fn scope.TxFunc) error {
	return t.Required(ctx, fn)
}

type errorTransactorIdentityScope struct {
	*identityTestScope
	txErr error
}

func (s *errorTransactorIdentityScope) Transactor() scope.Transactor {
	return errorRequiredTransactor{err: s.txErr}
}

func (s *errorTransactorIdentityScope) WithContext(ctx context.Context) scope.Scope {
	out := *s
	if s.identityTestScope != nil {
		inner := *s.identityTestScope
		inner.ctx = ctx
		out.identityTestScope = &inner
	}
	return &out
}

func TestResolveBootstrapAdminScopeEdgeCases(t *testing.T) {
	t.Parallel()
	db := openDB(t)
	base := newScope(t, db)

	t.Run("nil context uses background", func(t *testing.T) {
		_, ok, err := ResolveBootstrapAdmin(nil, base)
		if err != nil || ok {
			t.Fatalf("ok=%v err=%v", ok, err)
		}
	})

	t.Run("nil transactor", func(t *testing.T) {
		s := &nilTransactorIdentityScope{identityTestScope: base}
		_, ok, err := ResolveBootstrapAdmin(context.Background(), s)
		if err != nil || ok {
			t.Fatalf("ok=%v err=%v", ok, err)
		}
	})

	t.Run("nil session", func(t *testing.T) {
		s := &nilSessionIdentityScope{identityTestScope: base}
		_, ok, err := ResolveBootstrapAdmin(context.Background(), s)
		if err != nil || ok {
			t.Fatalf("ok=%v err=%v", ok, err)
		}
	})

	t.Run("session without db", func(t *testing.T) {
		s := &nilDBSessionIdentityScope{identityTestScope: base}
		_, ok, err := ResolveBootstrapAdmin(context.Background(), s)
		if err != nil || ok {
			t.Fatalf("ok=%v err=%v", ok, err)
		}
	})

	t.Run("WithContext nil falls back to root scope", func(t *testing.T) {
		s := &nilWithContextIdentityScope{identityTestScope: base}
		_, ok, err := ResolveBootstrapAdmin(context.Background(), s)
		if err != nil || ok {
			t.Fatalf("ok=%v err=%v", ok, err)
		}
	})

	t.Run("nil session inside transactor callback", func(t *testing.T) {
		inner := newScope(t, db)
		s := &nilSessionInTransactorScope{identityTestScope: inner, inner: inner}
		_, ok, err := ResolveBootstrapAdmin(context.Background(), s)
		if err != nil || ok {
			t.Fatalf("ok=%v err=%v", ok, err)
		}
	})
}

type nilSessionInTransactorScope struct {
	*identityTestScope
	inner *identityTestScope
}

func (s *nilSessionInTransactorScope) Transactor() scope.Transactor {
	return nilSessionInCallbackTransactor{root: s.inner}
}

func (s *nilSessionInTransactorScope) WithContext(ctx context.Context) scope.Scope {
	out := *s
	if s.identityTestScope != nil {
		inner := *s.identityTestScope
		inner.ctx = ctx
		out.identityTestScope = &inner
	}
	return &out
}

func TestResolveBootstrapAdminPartialSeedStates(t *testing.T) {
	t.Parallel()
	userID := xid.New().String()
	companyID := xid.New().String()

	t.Run("module table without auth row", func(t *testing.T) {
		db := openDB(t)
		if err := db.AutoMigrate(&meta.Module{}); err != nil {
			t.Fatal(err)
		}
		_, ok, err := ResolveBootstrapAdmin(context.Background(), newScope(t, db))
		if err != nil || ok {
			t.Fatalf("ok=%v err=%v", ok, err)
		}
	})

	t.Run("auth installed without user_admin mapping", func(t *testing.T) {
		db := openDB(t)
		if err := db.AutoMigrate(&meta.Module{}, &modmeta.ModelData{}); err != nil {
			t.Fatal(err)
		}
		if err := db.Create(&meta.Module{Name: "auth", Status: meta.Installed, Version: "1.0.0"}).Error; err != nil {
			t.Fatal(err)
		}
		_, ok, err := ResolveBootstrapAdmin(context.Background(), newScope(t, db))
		if err != nil || ok {
			t.Fatalf("ok=%v err=%v", ok, err)
		}
		_, _, perr := ResolvePrincipalUserID(context.Background(), newScope(t, db), "")
		if !errors.Is(perr, ErrBootstrapAdminUnavailable) {
			t.Fatalf("principal err=%v, want ErrBootstrapAdminUnavailable", perr)
		}
	})

	t.Run("auth not installed", func(t *testing.T) {
		db := openDB(t)
		if err := db.AutoMigrate(&meta.Module{}); err != nil {
			t.Fatal(err)
		}
		if err := db.Create(&meta.Module{Name: "auth", Status: meta.Uninstalled, Version: "1.0.0"}).Error; err != nil {
			t.Fatal(err)
		}
		_, ok, err := ResolveBootstrapAdmin(context.Background(), newScope(t, db))
		if err != nil || ok {
			t.Fatalf("ok=%v err=%v", ok, err)
		}
	})

	t.Run("empty user_admin res_id", func(t *testing.T) {
		db := openDB(t)
		if err := db.AutoMigrate(&meta.Module{}, &modmeta.ModelData{}); err != nil {
			t.Fatal(err)
		}
		if err := db.Create(&meta.Module{Name: "auth", Status: meta.Installed, Version: "1.0.0"}).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Create(&modmeta.ModelData{
			Module: "auth", Name: "user_admin", Application: "auth",
			ModelName: "User", ModelId: xid.New().String(), ResID: "  ",
		}).Error; err != nil {
			t.Fatal(err)
		}
		_, ok, err := ResolveBootstrapAdmin(context.Background(), newScope(t, db))
		if err != nil || ok {
			t.Fatalf("ok=%v err=%v", ok, err)
		}
	})

	t.Run("missing auth.User model", func(t *testing.T) {
		db := openDB(t)
		if err := db.AutoMigrate(&meta.Module{}, &modmeta.ModelData{}, &meta.Model{}); err != nil {
			t.Fatal(err)
		}
		if err := db.Create(&meta.Module{Name: "auth", Status: meta.Installed, Version: "1.0.0"}).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Create(&modmeta.ModelData{
			Module: "auth", Name: "user_admin", Application: "auth",
			ModelName: "User", ModelId: xid.New().String(), ResID: userID,
		}).Error; err != nil {
			t.Fatal(err)
		}
		_, ok, err := ResolveBootstrapAdmin(context.Background(), newScope(t, db))
		if err != nil || ok {
			t.Fatalf("ok=%v err=%v", ok, err)
		}
	})

	t.Run("empty model table", func(t *testing.T) {
		db := openDB(t)
		if err := db.AutoMigrate(&meta.Module{}, &modmeta.ModelData{}, &meta.Model{}); err != nil {
			t.Fatal(err)
		}
		if err := db.Create(&meta.Module{Name: "auth", Status: meta.Installed, Version: "1.0.0"}).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Create(&modmeta.ModelData{
			Module: "auth", Name: "user_admin", Application: "auth",
			ModelName: "User", ModelId: xid.New().String(), ResID: userID,
		}).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Create(&meta.Model{
			BaseModel:   meta.BaseModel{Id: sql.NullString{String: xid.New().String(), Valid: true}},
			Name:        "User",
			Application: "auth",
			ModelTable:  "  ",
		}).Error; err != nil {
			t.Fatal(err)
		}
		_, ok, err := ResolveBootstrapAdmin(context.Background(), newScope(t, db))
		if err != nil || ok {
			t.Fatalf("ok=%v err=%v", ok, err)
		}
	})

	t.Run("missing user row", func(t *testing.T) {
		db := openDB(t)
		if err := db.AutoMigrate(&meta.Module{}, &modmeta.ModelData{}, &meta.Model{}); err != nil {
			t.Fatal(err)
		}
		if err := db.Create(&meta.Module{Name: "auth", Status: meta.Installed, Version: "1.0.0"}).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Create(&modmeta.ModelData{
			Module: "auth", Name: "user_admin", Application: "auth",
			ModelName: "User", ModelId: xid.New().String(), ResID: userID,
		}).Error; err != nil {
			t.Fatal(err)
		}
		table := "auth_user_missing_row"
		if err := db.Create(&meta.Model{
			BaseModel:   meta.BaseModel{Id: sql.NullString{String: xid.New().String(), Valid: true}},
			Name:        "User",
			Application: "auth",
			ModelTable:  table,
		}).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Exec(`CREATE TABLE "` + table + `" (id TEXT PRIMARY KEY, company_id TEXT)`).Error; err != nil {
			t.Fatal(err)
		}
		_, ok, err := ResolveBootstrapAdmin(context.Background(), newScope(t, db))
		if err != nil || ok {
			t.Fatalf("ok=%v err=%v", ok, err)
		}
	})

	t.Run("company_main fallback when user company empty", func(t *testing.T) {
		db := openDB(t)
		if err := db.AutoMigrate(&meta.Module{}, &modmeta.ModelData{}, &meta.Model{}); err != nil {
			t.Fatal(err)
		}
		if err := db.Create(&meta.Module{Name: "auth", Status: meta.Installed, Version: "1.0.0"}).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Create(&modmeta.ModelData{
			Module: "auth", Name: "user_admin", Application: "auth",
			ModelName: "User", ModelId: xid.New().String(), ResID: userID,
		}).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Create(&modmeta.ModelData{
			Module: "base", Name: "company_main", Application: "base",
			ModelName: "Company", ModelId: xid.New().String(), ResID: companyID,
		}).Error; err != nil {
			t.Fatal(err)
		}
		table := "auth_user_fallback_co"
		if err := db.Create(&meta.Model{
			BaseModel:   meta.BaseModel{Id: sql.NullString{String: xid.New().String(), Valid: true}},
			Name:        "User",
			Application: "auth",
			ModelTable:  table,
		}).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Exec(`CREATE TABLE "` + table + `" (id TEXT PRIMARY KEY, company_id TEXT)`).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Exec(`INSERT INTO "`+table+`" (id, company_id) VALUES (?, ?)`, userID, "").Error; err != nil {
			t.Fatal(err)
		}
		got, ok, err := ResolveBootstrapAdmin(context.Background(), newScope(t, db))
		if err != nil || !ok {
			t.Fatalf("ok=%v err=%v", ok, err)
		}
		if got.UserID != userID || got.CompanyID != companyID {
			t.Fatalf("got %#v want user=%s company=%s", got, userID, companyID)
		}
	})

	t.Run("empty company after fallback", func(t *testing.T) {
		db := openDB(t)
		if err := db.AutoMigrate(&meta.Module{}, &modmeta.ModelData{}, &meta.Model{}); err != nil {
			t.Fatal(err)
		}
		if err := db.Create(&meta.Module{Name: "auth", Status: meta.Installed, Version: "1.0.0"}).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Create(&modmeta.ModelData{
			Module: "auth", Name: "user_admin", Application: "auth",
			ModelName: "User", ModelId: xid.New().String(), ResID: userID,
		}).Error; err != nil {
			t.Fatal(err)
		}
		table := "auth_user_no_co"
		if err := db.Create(&meta.Model{
			BaseModel:   meta.BaseModel{Id: sql.NullString{String: xid.New().String(), Valid: true}},
			Name:        "User",
			Application: "auth",
			ModelTable:  table,
		}).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Exec(`CREATE TABLE "` + table + `" (id TEXT PRIMARY KEY, company_id TEXT)`).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Exec(`INSERT INTO "`+table+`" (id, company_id) VALUES (?, ?)`, userID, "").Error; err != nil {
			t.Fatal(err)
		}
		_, ok, err := ResolveBootstrapAdmin(context.Background(), newScope(t, db))
		if err != nil || ok {
			t.Fatalf("ok=%v err=%v", ok, err)
		}
	})
}

func registerFailQueriesOnTable(t *testing.T, db *gorm.DB, table string, fail error) {
	t.Helper()
	name := "fail_query_" + table
	if err := db.Callback().Query().Before("gorm:query").Register(name, func(tx *gorm.DB) {
		if tx.Statement != nil && tx.Statement.Table == table {
			tx.AddError(fail)
		}
	}); err != nil {
		t.Fatalf("register callback: %v", err)
	}
	t.Cleanup(func() { _ = db.Callback().Query().Remove(name) })
}

func TestResolveBootstrapAdminOperationalErrors(t *testing.T) {
	t.Parallel()
	userID := xid.New().String()
	forced := errors.New("forced query failure")

	t.Run("auth module take fails", func(t *testing.T) {
		db := openDB(t)
		if err := db.AutoMigrate(&meta.Module{}); err != nil {
			t.Fatal(err)
		}
		if err := db.Create(&meta.Module{Name: "auth", Status: meta.Installed, Version: "1.0.0"}).Error; err != nil {
			t.Fatal(err)
		}
		registerFailQueriesOnTable(t, db, (&meta.Module{}).TableName(), forced)
		_, ok, err := ResolveBootstrapAdmin(context.Background(), newScope(t, db))
		if err == nil || ok || !strings.Contains(err.Error(), "load auth module") {
			t.Fatalf("ok=%v err=%v", ok, err)
		}
	})

	t.Run("user_admin mapping take fails", func(t *testing.T) {
		db := openDB(t)
		if err := db.AutoMigrate(&meta.Module{}, &modmeta.ModelData{}); err != nil {
			t.Fatal(err)
		}
		if err := db.Create(&meta.Module{Name: "auth", Status: meta.Installed, Version: "1.0.0"}).Error; err != nil {
			t.Fatal(err)
		}
		registerFailQueriesOnTable(t, db, (&modmeta.ModelData{}).TableName(), forced)
		_, ok, err := ResolveBootstrapAdmin(context.Background(), newScope(t, db))
		if err == nil || ok || !strings.Contains(err.Error(), "user_admin mapping") {
			t.Fatalf("ok=%v err=%v", ok, err)
		}
	})

	t.Run("auth.User lookup fails", func(t *testing.T) {
		db := openDB(t)
		if err := db.AutoMigrate(&meta.Module{}, &modmeta.ModelData{}); err != nil {
			t.Fatal(err)
		}
		if err := db.Create(&meta.Module{Name: "auth", Status: meta.Installed, Version: "1.0.0"}).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Create(&modmeta.ModelData{
			Module: "auth", Name: "user_admin", Application: "auth",
			ModelName: "User", ModelId: xid.New().String(), ResID: userID,
		}).Error; err != nil {
			t.Fatal(err)
		}
		registerFailQueriesOnTable(t, db, (&meta.Model{}).TableName(), forced)
		_, ok, err := ResolveBootstrapAdmin(context.Background(), newScope(t, db))
		if err == nil || ok || !strings.Contains(err.Error(), "lookup auth.User") {
			t.Fatalf("ok=%v err=%v", ok, err)
		}
	})

	t.Run("user row take fails", func(t *testing.T) {
		db := openDB(t)
		if err := db.AutoMigrate(&meta.Module{}, &modmeta.ModelData{}, &meta.Model{}); err != nil {
			t.Fatal(err)
		}
		if err := db.Create(&meta.Module{Name: "auth", Status: meta.Installed, Version: "1.0.0"}).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Create(&modmeta.ModelData{
			Module: "auth", Name: "user_admin", Application: "auth",
			ModelName: "User", ModelId: xid.New().String(), ResID: userID,
		}).Error; err != nil {
			t.Fatal(err)
		}
		table := "auth_user_row_err"
		if err := db.Create(&meta.Model{
			BaseModel:   meta.BaseModel{Id: sql.NullString{String: xid.New().String(), Valid: true}},
			Name: "User", Application: "auth", ModelTable: table,
		}).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Exec(`CREATE TABLE "` + table + `" (id TEXT PRIMARY KEY, company_id TEXT)`).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Exec(`INSERT INTO "`+table+`" (id, company_id) VALUES (?, ?)`, userID, "co").Error; err != nil {
			t.Fatal(err)
		}
		registerFailQueriesOnTable(t, db, table, forced)
		_, ok, err := ResolveBootstrapAdmin(context.Background(), newScope(t, db))
		if err == nil || ok || !strings.Contains(err.Error(), "user_admin row") {
			t.Fatalf("ok=%v err=%v", ok, err)
		}
	})

	t.Run("company_main mapping take fails", func(t *testing.T) {
		db := openDB(t)
		if err := db.AutoMigrate(&meta.Module{}, &modmeta.ModelData{}, &meta.Model{}); err != nil {
			t.Fatal(err)
		}
		if err := db.Create(&meta.Module{Name: "auth", Status: meta.Installed, Version: "1.0.0"}).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Create(&modmeta.ModelData{
			Module: "auth", Name: "user_admin", Application: "auth",
			ModelName: "User", ModelId: xid.New().String(), ResID: userID,
		}).Error; err != nil {
			t.Fatal(err)
		}
		table := "auth_user_co_err"
		if err := db.Create(&meta.Model{
			BaseModel:   meta.BaseModel{Id: sql.NullString{String: xid.New().String(), Valid: true}},
			Name: "User", Application: "auth", ModelTable: table,
		}).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Exec(`CREATE TABLE "` + table + `" (id TEXT PRIMARY KEY, company_id TEXT)`).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Exec(`INSERT INTO "`+table+`" (id, company_id) VALUES (?, ?)`, userID, "").Error; err != nil {
			t.Fatal(err)
		}
		name := "fail_company_main_query"
		var modelDataQueries int
		if err := db.Callback().Query().Before("gorm:query").Register(name, func(tx *gorm.DB) {
			if tx.Statement != nil && tx.Statement.Table == (&modmeta.ModelData{}).TableName() {
				modelDataQueries++
				if modelDataQueries >= 2 {
					tx.AddError(forced)
				}
			}
		}); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = db.Callback().Query().Remove(name) })
		_, ok, err := ResolveBootstrapAdmin(context.Background(), newScope(t, db))
		if err == nil || ok || !strings.Contains(err.Error(), "company_main mapping") {
			t.Fatalf("ok=%v err=%v", ok, err)
		}
	})

	t.Run("company_main empty res_id soft no inject", func(t *testing.T) {
		db := openDB(t)
		if err := db.AutoMigrate(&meta.Module{}, &modmeta.ModelData{}, &meta.Model{}); err != nil {
			t.Fatal(err)
		}
		if err := db.Create(&meta.Module{Name: "auth", Status: meta.Installed, Version: "1.0.0"}).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Create(&modmeta.ModelData{
			Module: "auth", Name: "user_admin", Application: "auth",
			ModelName: "User", ModelId: xid.New().String(), ResID: userID,
		}).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Create(&modmeta.ModelData{
			Module: "base", Name: "company_main", Application: "base",
			ModelName: "Company", ModelId: xid.New().String(), ResID: "  ",
		}).Error; err != nil {
			t.Fatal(err)
		}
		table := "auth_user_empty_co_main"
		if err := db.Create(&meta.Model{
			BaseModel:   meta.BaseModel{Id: sql.NullString{String: xid.New().String(), Valid: true}},
			Name: "User", Application: "auth", ModelTable: table,
		}).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Exec(`CREATE TABLE "` + table + `" (id TEXT PRIMARY KEY, company_id TEXT)`).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Exec(`INSERT INTO "`+table+`" (id, company_id) VALUES (?, ?)`, userID, "").Error; err != nil {
			t.Fatal(err)
		}
		_, ok, err := ResolveBootstrapAdmin(context.Background(), newScope(t, db))
		if err != nil || ok {
			t.Fatalf("ok=%v err=%v", ok, err)
		}
	})
}

func TestResolvePrincipalUserIDReturnsResolveError(t *testing.T) {
	t.Parallel()
	db := openDB(t)
	base := newScope(t, db)
	s := &errorTransactorIdentityScope{identityTestScope: base, txErr: errors.New("tx failed")}
	_, _, err := ResolvePrincipalUserID(context.Background(), s, "")
	if err == nil || err.Error() != "tx failed" {
		t.Fatalf("err=%v", err)
	}
}
