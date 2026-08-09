// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package backend

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/internal/module/lifecycle"
	modmeta "github.com/choysum-dev/choysum/internal/module/meta"
	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/rs/xid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	gormlogger "gorm.io/gorm/logger"
)

// gormWhereVarsContain reports whether any WHERE clause expression var equals want.
// Statement.Vars is still empty in Before("gorm:query"); values live on clause.Expr.
func gormWhereVarsContain(tx *gorm.DB, want string) bool {
	if tx == nil || tx.Statement == nil {
		return false
	}
	where, ok := tx.Statement.Clauses["WHERE"]
	if !ok {
		return false
	}
	w, ok := where.Expression.(clause.Where)
	if !ok {
		return false
	}
	for _, expr := range w.Exprs {
		e, ok := expr.(clause.Expr)
		if !ok {
			continue
		}
		for _, v := range e.Vars {
			if v == want {
				return true
			}
		}
	}
	return false
}

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

type nilWithContextScope struct {
	*identityTestScope
}

func (s *nilWithContextScope) WithContext(context.Context) scope.Scope { return nil }

func openIdentityTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := filepath.Join(t.TempDir(), "unit-identity.sqlite")
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	return db
}

func newIdentityTestScope(t *testing.T, db *gorm.DB) *identityTestScope {
	t.Helper()
	return &identityTestScope{
		ctx:     context.Background(),
		session: &scope.Session{DB: db},
		cfg:     &config.Config{Db: &config.DbConfig{Dialect: "sqlite"}},
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

func migrateIdentityTables(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.AutoMigrate(&meta.Module{}, &modmeta.ModelData{}, &meta.Model{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
}

func seedAuthInstalled(t *testing.T, db *gorm.DB) {
	t.Helper()
	mod := &meta.Module{Name: "auth", Status: meta.Installed, Version: "1.0.0"}
	if err := db.Create(mod).Error; err != nil {
		t.Fatalf("create auth module: %v", err)
	}
}

func seedUserAdminMapping(t *testing.T, db *gorm.DB, resID string) {
	t.Helper()
	row := &modmeta.ModelData{
		Module:      "auth",
		Name:        "user_admin",
		Application: "auth",
		ModelName:   "User",
		ModelId:     xid.New().String(),
		ResID:       resID,
	}
	if err := db.Create(row).Error; err != nil {
		t.Fatalf("create user_admin mapping: %v", err)
	}
}

func seedAuthUserModel(t *testing.T, db *gorm.DB, table string) {
	t.Helper()
	m := &meta.Model{
		BaseModel:   meta.BaseModel{Id: sql.NullString{String: xid.New().String(), Valid: true}},
		Name:        "User",
		Application: "auth",
		ModelTable:  table,
		Path:        "@/auth/service/models/user.ts",
	}
	if err := db.Create(m).Error; err != nil {
		t.Fatalf("create auth.User model: %v", err)
	}
}

func createUserTable(t *testing.T, db *gorm.DB, table string) {
	t.Helper()
	stmt := `CREATE TABLE IF NOT EXISTS "` + table + `" (id TEXT PRIMARY KEY, company_id TEXT)`
	if err := db.Exec(stmt).Error; err != nil {
		t.Fatalf("create user table: %v", err)
	}
}

func TestResolveUnitTestDefaultIdentityNilScope(t *testing.T) {
	_, ok, err := resolveUnitTestDefaultIdentity(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatal("expected ok=false for nil scope")
	}
}

func TestResolveUnitTestDefaultIdentityNilContextUsesBackground(t *testing.T) {
	db := openIdentityTestDB(t)
	runtimeScope := newIdentityTestScope(t, db)
	// Empty DB (no meta_module) → ok=false, but nil ctx must not panic.
	_, ok, err := resolveUnitTestDefaultIdentity(nil, runtimeScope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatal("expected ok=false without meta_module")
	}
}

func TestResolveUnitTestDefaultIdentityNilWithContextFallsBack(t *testing.T) {
	db := openIdentityTestDB(t)
	base := newIdentityTestScope(t, db)
	_, ok, err := resolveUnitTestDefaultIdentity(context.Background(), &nilWithContextScope{identityTestScope: base})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatal("expected ok=false without tables")
	}
}

func TestResolveUnitTestDefaultIdentityNilSessionOrDB(t *testing.T) {
	t.Run("nil_session", func(t *testing.T) {
		s := &identityTestScope{ctx: context.Background(), session: nil}
		_, ok, err := resolveUnitTestDefaultIdentity(context.Background(), s)
		if err != nil || ok {
			t.Fatalf("ok=%v err=%v, want ok=false nil err", ok, err)
		}
	})
	t.Run("nil_db", func(t *testing.T) {
		s := &identityTestScope{ctx: context.Background(), session: &scope.Session{DB: nil}}
		_, ok, err := resolveUnitTestDefaultIdentity(context.Background(), s)
		if err != nil || ok {
			t.Fatalf("ok=%v err=%v, want ok=false nil err", ok, err)
		}
	})
}

func TestResolveUnitTestDefaultIdentityNoMetaModuleTable(t *testing.T) {
	db := openIdentityTestDB(t)
	_, ok, err := resolveUnitTestDefaultIdentity(context.Background(), newIdentityTestScope(t, db))
	if err != nil || ok {
		t.Fatalf("ok=%v err=%v, want ok=false", ok, err)
	}
}

func TestResolveUnitTestDefaultIdentityAuthNotFound(t *testing.T) {
	db := openIdentityTestDB(t)
	migrateIdentityTables(t, db)
	_, ok, err := resolveUnitTestDefaultIdentity(context.Background(), newIdentityTestScope(t, db))
	if err != nil || ok {
		t.Fatalf("ok=%v err=%v, want ok=false when auth missing", ok, err)
	}
}

func TestResolveUnitTestDefaultIdentityAuthNotInstalled(t *testing.T) {
	db := openIdentityTestDB(t)
	migrateIdentityTables(t, db)
	if err := db.Create(&meta.Module{Name: "auth", Status: meta.ToInstall, Version: "1.0.0"}).Error; err != nil {
		t.Fatalf("create auth: %v", err)
	}
	_, ok, err := resolveUnitTestDefaultIdentity(context.Background(), newIdentityTestScope(t, db))
	if err != nil || ok {
		t.Fatalf("ok=%v err=%v, want ok=false when auth not installed", ok, err)
	}
}

func TestResolveUnitTestDefaultIdentityModelDataMissing(t *testing.T) {
	db := openIdentityTestDB(t)
	if err := db.AutoMigrate(&meta.Module{}); err != nil {
		t.Fatalf("migrate module only: %v", err)
	}
	seedAuthInstalled(t, db)
	_, ok, err := resolveUnitTestDefaultIdentity(context.Background(), newIdentityTestScope(t, db))
	if err != nil || ok {
		t.Fatalf("ok=%v err=%v, want ok=false without model_data table", ok, err)
	}
}

func TestResolveUnitTestDefaultIdentityUserAdminMissing(t *testing.T) {
	db := openIdentityTestDB(t)
	migrateIdentityTables(t, db)
	seedAuthInstalled(t, db)
	_, ok, err := resolveUnitTestDefaultIdentity(context.Background(), newIdentityTestScope(t, db))
	if err != nil || ok {
		t.Fatalf("ok=%v err=%v, want ok=false when user_admin missing", ok, err)
	}
}

func TestResolveUnitTestDefaultIdentityEmptyResID(t *testing.T) {
	db := openIdentityTestDB(t)
	migrateIdentityTables(t, db)
	seedAuthInstalled(t, db)
	seedUserAdminMapping(t, db, "   ")
	_, ok, err := resolveUnitTestDefaultIdentity(context.Background(), newIdentityTestScope(t, db))
	if err != nil || ok {
		t.Fatalf("ok=%v err=%v, want ok=false for empty res_id", ok, err)
	}
}

func TestResolveUnitTestDefaultIdentityLookupEffectiveMissing(t *testing.T) {
	db := openIdentityTestDB(t)
	migrateIdentityTables(t, db)
	seedAuthInstalled(t, db)
	seedUserAdminMapping(t, db, xid.New().String())
	_, ok, err := resolveUnitTestDefaultIdentity(context.Background(), newIdentityTestScope(t, db))
	// Missing auth.User projection is a seed gap (ok=false), not an operational abort.
	if err != nil || ok {
		t.Fatalf("ok=%v err=%v, want ok=false nil err when auth.User missing", ok, err)
	}
}

func TestResolveUnitTestDefaultIdentityEmptyModelTable(t *testing.T) {
	db := openIdentityTestDB(t)
	migrateIdentityTables(t, db)
	seedAuthInstalled(t, db)
	seedUserAdminMapping(t, db, xid.New().String())
	seedAuthUserModel(t, db, "   ")
	_, ok, err := resolveUnitTestDefaultIdentity(context.Background(), newIdentityTestScope(t, db))
	if err != nil || ok {
		t.Fatalf("ok=%v err=%v, want ok=false for empty ModelTable", ok, err)
	}
}

func TestResolveUnitTestDefaultIdentityUserRowNotFound(t *testing.T) {
	db := openIdentityTestDB(t)
	migrateIdentityTables(t, db)
	seedAuthInstalled(t, db)
	userID := xid.New().String()
	seedUserAdminMapping(t, db, userID)
	table := "auth_user_identity"
	seedAuthUserModel(t, db, table)
	createUserTable(t, db, table)

	_, ok, err := resolveUnitTestDefaultIdentity(context.Background(), newIdentityTestScope(t, db))
	if err != nil || ok {
		t.Fatalf("ok=%v err=%v, want ok=false when user row missing", ok, err)
	}
}

func TestResolveUnitTestDefaultIdentityCompanyMainFallback(t *testing.T) {
	db := openIdentityTestDB(t)
	migrateIdentityTables(t, db)
	seedAuthInstalled(t, db)
	userID := xid.New().String()
	companyID := xid.New().String()
	seedUserAdminMapping(t, db, userID)
	table := "auth_user_identity"
	seedAuthUserModel(t, db, table)
	createUserTable(t, db, table)
	if err := db.Exec(`INSERT INTO "`+table+`" (id, company_id) VALUES (?, ?)`, userID, "").Error; err != nil {
		t.Fatalf("insert user: %v", err)
	}
	if err := db.Create(&modmeta.ModelData{
		Module: "base", Name: "company_main", Application: "base",
		ModelName: "Company", ModelId: xid.New().String(), ResID: companyID,
	}).Error; err != nil {
		t.Fatalf("create company_main: %v", err)
	}

	got, ok, err := resolveUnitTestDefaultIdentity(context.Background(), newIdentityTestScope(t, db))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected ok=true with company_main fallback")
	}
	if got.UserID != userID || got.CompanyID != companyID {
		t.Fatalf("got %#v, want user=%s company=%s", got, userID, companyID)
	}
}

func TestResolveUnitTestDefaultIdentityCompanyMissing(t *testing.T) {
	db := openIdentityTestDB(t)
	migrateIdentityTables(t, db)
	seedAuthInstalled(t, db)
	userID := xid.New().String()
	seedUserAdminMapping(t, db, userID)
	table := "auth_user_identity"
	seedAuthUserModel(t, db, table)
	createUserTable(t, db, table)
	if err := db.Exec(`INSERT INTO "`+table+`" (id, company_id) VALUES (?, ?)`, userID, "").Error; err != nil {
		t.Fatalf("insert user: %v", err)
	}

	_, ok, err := resolveUnitTestDefaultIdentity(context.Background(), newIdentityTestScope(t, db))
	if err != nil || ok {
		t.Fatalf("ok=%v err=%v, want ok=false when company_main missing", ok, err)
	}
}

func TestResolveUnitTestDefaultIdentityCompanyEmptyResID(t *testing.T) {
	db := openIdentityTestDB(t)
	migrateIdentityTables(t, db)
	seedAuthInstalled(t, db)
	userID := xid.New().String()
	seedUserAdminMapping(t, db, userID)
	table := "auth_user_identity"
	seedAuthUserModel(t, db, table)
	createUserTable(t, db, table)
	if err := db.Exec(`INSERT INTO "`+table+`" (id, company_id) VALUES (?, ?)`, userID, "").Error; err != nil {
		t.Fatalf("insert user: %v", err)
	}
	if err := db.Create(&modmeta.ModelData{
		Module: "base", Name: "company_main", Application: "base",
		ModelName: "Company", ModelId: xid.New().String(), ResID: "  ",
	}).Error; err != nil {
		t.Fatalf("create company_main: %v", err)
	}

	_, ok, err := resolveUnitTestDefaultIdentity(context.Background(), newIdentityTestScope(t, db))
	if err != nil || ok {
		t.Fatalf("ok=%v err=%v, want ok=false for empty company res_id", ok, err)
	}
}

func TestResolveUnitTestDefaultIdentitySuccessWithCompanyOnUser(t *testing.T) {
	db := openIdentityTestDB(t)
	migrateIdentityTables(t, db)
	seedAuthInstalled(t, db)
	userID := xid.New().String()
	companyID := xid.New().String()
	seedUserAdminMapping(t, db, userID)
	table := "auth_user_identity"
	seedAuthUserModel(t, db, table)
	createUserTable(t, db, table)
	if err := db.Exec(`INSERT INTO "`+table+`" (id, company_id) VALUES (?, ?)`, userID, companyID).Error; err != nil {
		t.Fatalf("insert user: %v", err)
	}

	got, ok, err := resolveUnitTestDefaultIdentity(context.Background(), newIdentityTestScope(t, db))
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	if got.UserID != userID || got.CompanyID != companyID {
		t.Fatalf("got %#v", got)
	}
}

func TestResolveUnitTestDefaultIdentityOperationalDBErrors(t *testing.T) {
	t.Run("auth_module_query", func(t *testing.T) {
		db := openIdentityTestDB(t)
		// HasTable passes, but Select(id,status) fails on incomplete schema.
		if err := db.Exec(`CREATE TABLE meta_module (id TEXT)`).Error; err != nil {
			t.Fatalf("create broken meta_module: %v", err)
		}
		_, ok, err := resolveUnitTestDefaultIdentity(context.Background(), newIdentityTestScope(t, db))
		if err == nil || !strings.Contains(err.Error(), "load auth module for unit identity") {
			t.Fatalf("error=%v, want auth module load wrap", err)
		}
		if ok {
			t.Fatal("expected ok=false")
		}
	})

	t.Run("user_admin_mapping_query", func(t *testing.T) {
		db := openIdentityTestDB(t)
		migrateIdentityTables(t, db)
		seedAuthInstalled(t, db)
		if err := db.Exec(`ALTER TABLE meta_model_data RENAME TO meta_model_data_hidden`).Error; err != nil {
			t.Fatalf("rename: %v", err)
		}
		if err := db.Exec(`CREATE TABLE meta_model_data (broken INTEGER)`).Error; err != nil {
			t.Fatalf("create broken table: %v", err)
		}
		_, ok, err := resolveUnitTestDefaultIdentity(context.Background(), newIdentityTestScope(t, db))
		if err == nil || !strings.Contains(err.Error(), "load auth.user_admin mapping for unit identity") {
			t.Fatalf("error=%v, want user_admin mapping wrap", err)
		}
		if ok {
			t.Fatal("expected ok=false")
		}
	})

	t.Run("user_model_lookup_query", func(t *testing.T) {
		db := openIdentityTestDB(t)
		migrateIdentityTables(t, db)
		seedAuthInstalled(t, db)
		userID := xid.New().String()
		seedUserAdminMapping(t, db, userID)
		// Break meta_model Find so LookupEffectiveModel returns a non-NotFound error.
		if err := db.Exec(`ALTER TABLE meta_model RENAME TO meta_model_hidden`).Error; err != nil {
			t.Fatalf("rename meta_model: %v", err)
		}
		if err := db.Exec(`CREATE TABLE meta_model (broken INTEGER)`).Error; err != nil {
			t.Fatalf("create broken meta_model: %v", err)
		}
		_, ok, err := resolveUnitTestDefaultIdentity(context.Background(), newIdentityTestScope(t, db))
		if err == nil || !strings.Contains(err.Error(), "lookup auth.User for unit identity") {
			t.Fatalf("error=%v, want auth.User lookup wrap", err)
		}
		if ok {
			t.Fatal("expected ok=false")
		}
	})

	t.Run("user_row_query", func(t *testing.T) {
		db := openIdentityTestDB(t)
		migrateIdentityTables(t, db)
		seedAuthInstalled(t, db)
		userID := xid.New().String()
		seedUserAdminMapping(t, db, userID)
		table := "auth_user_identity"
		seedAuthUserModel(t, db, table)
		if err := db.Model(&meta.Model{}).Where("application = ? AND name = ?", "auth", "User").
			Update("model_table", "missing_user_table_xyz").Error; err != nil {
			t.Fatalf("update model_table: %v", err)
		}
		_, ok, err := resolveUnitTestDefaultIdentity(context.Background(), newIdentityTestScope(t, db))
		if err == nil || !strings.Contains(err.Error(), "load auth.user_admin row for unit identity") {
			t.Fatalf("error=%v, want user row load wrap", err)
		}
		if ok {
			t.Fatal("expected ok=false")
		}
	})

	t.Run("company_main_mapping_query", func(t *testing.T) {
		db := openIdentityTestDB(t)
		migrateIdentityTables(t, db)
		seedAuthInstalled(t, db)
		userID := xid.New().String()
		seedUserAdminMapping(t, db, userID)
		table := "auth_user_identity"
		seedAuthUserModel(t, db, table)
		createUserTable(t, db, table)
		if err := db.Exec(`INSERT INTO "`+table+`" (id, company_id) VALUES (?, ?)`, userID, "").Error; err != nil {
			t.Fatalf("insert user: %v", err)
		}
		if err := db.Create(&modmeta.ModelData{
			Module: "base", Name: "company_main", Application: "base",
			ModelName: "Company", ModelId: xid.New().String(), ResID: xid.New().String(),
		}).Error; err != nil {
			t.Fatalf("create company_main: %v", err)
		}
		// Fail only the company_main mapping Select (keyed by WHERE clause vars, not ordinal).
		if err := db.Callback().Query().Before("gorm:query").Register("identity_fail_company_main", func(tx *gorm.DB) {
			if tx.Statement == nil || tx.Statement.Schema == nil {
				return
			}
			if tx.Statement.Schema.Table != (&modmeta.ModelData{}).TableName() {
				return
			}
			if !gormWhereVarsContain(tx, "company_main") {
				return
			}
			_ = tx.AddError(errors.New("forced company_main query failure"))
		}); err != nil {
			t.Fatalf("register callback: %v", err)
		}
		t.Cleanup(func() {
			_ = db.Callback().Query().Remove("identity_fail_company_main")
		})
		_, ok, err := resolveUnitTestDefaultIdentity(context.Background(), newIdentityTestScope(t, db))
		if err == nil || !strings.Contains(err.Error(), "load base.company_main mapping for unit identity") {
			t.Fatalf("error=%v, want company_main mapping wrap", err)
		}
		if ok {
			t.Fatal("expected ok=false")
		}
	})
}

func TestUnitTestJsRequestContextShape(t *testing.T) {
	ctx := unitTestJsRequestContext(unitTestDefaultIdentity{UserID: "u1", CompanyID: "c1"})
	identity, _ := ctx["identity"].(map[string]interface{})
	if identity["userId"] != "u1" {
		t.Fatalf("userId=%v", identity["userId"])
	}
	biz, _ := ctx["ctx"].(map[string]interface{})
	if biz["activeCompanyId"] != "c1" {
		t.Fatalf("activeCompanyId=%v", biz["activeCompanyId"])
	}
	ids, _ := biz["enabledCompanyIds"].([]string)
	if len(ids) != 1 || ids[0] != "c1" {
		t.Fatalf("enabledCompanyIds=%v", biz["enabledCompanyIds"])
	}
	req, _ := ctx["req"].(map[string]interface{})
	if req["depth"] != 0 {
		t.Fatalf("depth=%v", req["depth"])
	}
}

func TestShouldSkipWebShellForUnitApp(t *testing.T) {
	cases := []struct {
		app  string
		want bool
	}{
		{"auth", false},
		{" AUTH ", false},
		{"web", true},
		{"base", true},
		{"", true},
	}
	for _, tc := range cases {
		if got := shouldSkipWebShellForUnitApp(tc.app); got != tc.want {
			t.Fatalf("shouldSkipWebShellForUnitApp(%q)=%v, want %v", tc.app, got, tc.want)
		}
	}
}

type recordingInstaller struct {
	calls []lifecycle.InstallRequest
	err   error
}

func (r *recordingInstaller) Install(ctx context.Context, req lifecycle.InstallRequest) error {
	r.calls = append(r.calls, req)
	return r.err
}

func TestEnsureMetaInstalledForUnitApp(t *testing.T) {
	t.Run("skip_base", func(t *testing.T) {
		inst := &recordingInstaller{}
		if err := ensureMetaInstalledForUnitApp(context.Background(), inst, "base"); err != nil {
			t.Fatalf("err=%v", err)
		}
		if len(inst.calls) != 0 {
			t.Fatalf("calls=%v", inst.calls)
		}
	})
	t.Run("install_web", func(t *testing.T) {
		inst := &recordingInstaller{}
		if err := ensureMetaInstalledForUnitApp(context.Background(), inst, "web"); err != nil {
			t.Fatalf("err=%v", err)
		}
		if len(inst.calls) != 1 || inst.calls[0].Name != "meta" || !inst.calls[0].SkipWebShell {
			t.Fatalf("calls=%+v", inst.calls)
		}
	})
	t.Run("install_error", func(t *testing.T) {
		inst := &recordingInstaller{err: errors.New("install meta failed")}
		err := ensureMetaInstalledForUnitApp(context.Background(), inst, "auth")
		if err == nil || !strings.Contains(err.Error(), "install meta failed") {
			t.Fatalf("err=%v", err)
		}
	})
}

func TestInstallUnitAppModules(t *testing.T) {
	t.Run("base_skips_meta", func(t *testing.T) {
		inst := &recordingInstaller{}
		if err := installUnitAppModules(context.Background(), inst, "base"); err != nil {
			t.Fatalf("err=%v", err)
		}
		if len(inst.calls) != 1 || inst.calls[0].Name != "base" || !inst.calls[0].SkipWebShell {
			t.Fatalf("calls=%+v", inst.calls)
		}
	})
	t.Run("auth_installs_meta", func(t *testing.T) {
		inst := &recordingInstaller{}
		if err := installUnitAppModules(context.Background(), inst, "auth"); err != nil {
			t.Fatalf("err=%v", err)
		}
		if len(inst.calls) != 2 || inst.calls[0].Name != "auth" || inst.calls[0].SkipWebShell || inst.calls[1].Name != "meta" {
			t.Fatalf("calls=%+v", inst.calls)
		}
	})
	t.Run("app_install_error", func(t *testing.T) {
		inst := &recordingInstaller{err: errors.New("app install failed")}
		err := installUnitAppModules(context.Background(), inst, "web")
		if err == nil || !strings.Contains(err.Error(), "app install failed") {
			t.Fatalf("err=%v", err)
		}
	})
}

func TestLoadUnitAppTestContext(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		prev := unitTestIdentityContextFn
		t.Cleanup(func() { unitTestIdentityContextFn = prev })
		unitTestIdentityContextFn = func(ctx context.Context, testScope scope.Scope) (map[string]interface{}, error) {
			return map[string]interface{}{"ok": true}, nil
		}
		jsCtx, err := loadUnitAppTestContext(context.Background(), nil)
		if err != nil || jsCtx["ok"] != true {
			t.Fatalf("jsCtx=%v err=%v", jsCtx, err)
		}
	})
	t.Run("error", func(t *testing.T) {
		prev := unitTestIdentityContextFn
		t.Cleanup(func() { unitTestIdentityContextFn = prev })
		unitTestIdentityContextFn = func(ctx context.Context, testScope scope.Scope) (map[string]interface{}, error) {
			return nil, errors.New("identity boom")
		}
		_, err := loadUnitAppTestContext(context.Background(), nil)
		if err == nil || err.Error() != "identity boom" {
			t.Fatalf("err=%v", err)
		}
	})
}

func TestJsContextWithUnitTestIdentity(t *testing.T) {
	t.Run("nil_scope", func(t *testing.T) {
		jsCtx, err := jsContextWithUnitTestIdentity(context.Background(), nil)
		if err != nil {
			t.Fatalf("err=%v", err)
		}
		if len(jsCtx) != 0 {
			t.Fatalf("jsCtx=%v", jsCtx)
		}
	})
	t.Run("identity_error", func(t *testing.T) {
		db := openIdentityTestDB(t)
		if err := db.Exec(`CREATE TABLE meta_module (id TEXT)`).Error; err != nil {
			t.Fatalf("create broken meta_module: %v", err)
		}
		_, err := jsContextWithUnitTestIdentity(context.Background(), newIdentityTestScope(t, db))
		if err == nil || !strings.Contains(err.Error(), "resolve unit test default identity") {
			t.Fatalf("err=%v", err)
		}
	})
	t.Run("ok_inject", func(t *testing.T) {
		db := openIdentityTestDB(t)
		migrateIdentityTables(t, db)
		seedAuthInstalled(t, db)
		userID := xid.New().String()
		companyID := xid.New().String()
		seedUserAdminMapping(t, db, userID)
		table := "auth_user_identity"
		seedAuthUserModel(t, db, table)
		createUserTable(t, db, table)
		if err := db.Exec(`INSERT INTO "`+table+`" (id, company_id) VALUES (?, ?)`, userID, companyID).Error; err != nil {
			t.Fatalf("insert user: %v", err)
		}
		jsCtx, err := jsContextWithUnitTestIdentity(context.Background(), newIdentityTestScope(t, db))
		if err != nil {
			t.Fatalf("err=%v", err)
		}
		identity, _ := jsCtx["identity"].(map[string]interface{})
		if identity["userId"] != userID {
			t.Fatalf("userId=%v", identity["userId"])
		}
	})
}

func TestShouldInstallMetaForUnitApp(t *testing.T) {
	cases := []struct {
		app  string
		want bool
	}{
		{"auth", true},
		{" AUTH ", true},
		{"web", true},
		{" Web ", true},
		{"base", false},
		{"", false},
	}
	for _, tc := range cases {
		if got := shouldInstallMetaForUnitApp(tc.app); got != tc.want {
			t.Fatalf("shouldInstallMetaForUnitApp(%q)=%v, want %v", tc.app, got, tc.want)
		}
	}
}
