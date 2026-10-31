// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package systemidentity

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"log/slog"
	"path/filepath"
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
