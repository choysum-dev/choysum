// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package models

import (
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/rs/xid"
	"gorm.io/gorm"
)

func migrateI18nDualStoreTables(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := meta.EnsureDualStoreTables(db); err != nil {
		t.Fatalf("EnsureDualStoreTables: %v", err)
	}
}

func TestEnsureI18nMetaCreatesModelAndServices(t *testing.T) {
	rs := newTestScope(t)
	migrateI18nDualStoreTables(t, rs.Session().DB)

	moduleID := sql.NullString{String: xid.New().String(), Valid: true}
	if err := EnsureI18nMeta(rs, "auth", moduleID); err != nil {
		t.Fatalf("EnsureI18nMeta: %v", err)
	}
	if err := EnsureI18nMeta(rs, "auth", moduleID); err != nil {
		t.Fatalf("EnsureI18nMeta idempotent: %v", err)
	}

	decls, err := meta.ListDeclarations(rs.Session().DB, meta.DeclarationQuery{
		Application: "auth", Name: "I18n", Path: "go://i18n/auth", PreloadTree: true,
	})
	if err != nil || len(decls) != 1 {
		t.Fatalf("list declaration: n=%d err=%v", len(decls), err)
	}
	if !decls[0].Abstract || !decls[0].Readonly {
		t.Fatalf("expected abstract readonly declaration: %+v", decls[0])
	}

	var model meta.Model
	if err := rs.Session().Where("name = ? AND application = ?", "I18n", "auth").Take(&model).Error; err != nil {
		t.Fatalf("lookup effective Model: %v", err)
	}
	if !model.Abstract || !model.Readonly {
		t.Fatalf("expected abstract readonly effective I18n model: %+v", model)
	}

	var services []meta.Service
	if err := rs.Session().Where("model_id = ?", model.Id.String).Find(&services).Error; err != nil {
		t.Fatal(err)
	}
	if len(services) != 3 {
		t.Fatalf("expected 3 services, got %d", len(services))
	}
	byName := map[string]struct{}{}
	for _, svc := range services {
		byName[svc.Name] = struct{}{}
	}
	for _, name := range []string{"GetTranslations", "SearchTerms", "UpdateTerm"} {
		if _, ok := byName[name]; !ok {
			t.Fatalf("missing service %s in %#v", name, byName)
		}
	}
}

func TestEnsureI18nMetaSeedsTerminologyEditorAllows(t *testing.T) {
	rs := newTestScope(t)
	db := rs.Session().DB
	migrateI18nDualStoreTables(t, db)
	if err := db.Exec(`CREATE TABLE auth_role (
		id TEXT PRIMARY KEY,
		code TEXT,
		deleted_at DATETIME
	)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE auth_role_method_access (
		id TEXT PRIMARY KEY,
		role_id TEXT,
		meta_application_id TEXT,
		meta_model_id TEXT,
		meta_service_id TEXT,
		mode TEXT,
		source TEXT,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	)`).Error; err != nil {
		t.Fatal(err)
	}
	roleID := xid.New().String()
	if err := db.Exec(`INSERT INTO auth_role (id, code) VALUES (?, ?)`, roleID, "terminology.editor").Error; err != nil {
		t.Fatal(err)
	}

	seedTranslationTermEffective(t, db, "web")

	if err := EnsureI18nMeta(rs, "web", sql.NullString{}); err != nil {
		t.Fatalf("EnsureI18nMeta: %v", err)
	}

	var count int64
	if err := db.Table("auth_role_method_access").Where("role_id = ?", roleID).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 3 {
		t.Fatalf("expected 3 RoleMethodAccess rows (Search/Read/Update), got %d", count)
	}

	var getTranslationsAllows int64
	if err := db.Raw(`
		SELECT COUNT(*) FROM auth_role_method_access rma
		JOIN meta_service s ON s.id = rma.meta_service_id
		WHERE rma.role_id = ? AND s.name = ? AND rma.deleted_at IS NULL
	`, roleID, "GetTranslations").Scan(&getTranslationsAllows).Error; err != nil {
		t.Fatal(err)
	}
	if getTranslationsAllows != 0 {
		t.Fatalf("GetTranslations must not be bound to terminology.editor, got %d", getTranslationsAllows)
	}

	if err := EnsureI18nMeta(rs, "web", sql.NullString{}); err != nil {
		t.Fatalf("EnsureI18nMeta idempotent seed: %v", err)
	}
}

func seedTranslationTermEffective(t *testing.T, db *gorm.DB, application string) {
	t.Helper()
	if err := meta.EnsureAbstractModel(db, meta.AbstractModelSpec{
		Name:         translationTermModelName,
		Path:         "go://translation_term/" + application,
		Application:  application,
		ServiceNames: []string{"Search", "Read", "Update", "GetTranslations", "Create"},
	}); err != nil {
		t.Fatalf("EnsureAbstractModel TranslationTerm: %v", err)
	}
	if err := meta.FlushEffective(db, []meta.LogicalKey{{Application: application, Name: translationTermModelName}}); err != nil {
		t.Fatalf("FlushEffective TranslationTerm: %v", err)
	}
}

func TestEnsureI18nMetaEarlyReturns(t *testing.T) {
	if err := EnsureI18nMeta(nil, "", sql.NullString{}); err != nil {
		t.Fatalf("empty application: %v", err)
	}
	if err := EnsureI18nMeta(nil, coreApplication, sql.NullString{}); err != nil {
		t.Fatalf("core application: %v", err)
	}
	if err := EnsureI18nMeta(nil, "auth", sql.NullString{}); err != nil {
		t.Fatalf("nil scope: %v", err)
	}

	rs := newTestScope(t)
	if err := EnsureI18nMeta(rs, "auth", sql.NullString{}); err != nil {
		t.Fatalf("missing tables: %v", err)
	}
}

func TestEnsureI18nMetaPrefersCanonicalPath(t *testing.T) {
	rs := newTestScope(t)
	db := rs.Session().DB
	migrateI18nDualStoreTables(t, db)

	ext := &meta.Model{
		BaseModel: meta.BaseModel{
			Id: sql.NullString{String: xid.New().String(), Valid: true},
		},
		Name:        "I18n",
		Path:        "modules/ext/i18n_override.ts",
		Application: "auth",
		ClassName:   "I18n",
		Abstract:    true,
		Readonly:    true,
	}
	if err := meta.PersistModelTreeAsRaw(db, ext); err != nil {
		t.Fatalf("seed extension declaration: %v", err)
	}
	// Bump timestamps so Order(created_at DESC) would prefer the extension if name-only.
	if err := db.Table("meta_raw_model").Where("id = ?", ext.Id.String).Updates(map[string]any{
		"created_at": db.NowFunc().Add(time.Hour),
		"updated_at": db.NowFunc().Add(time.Hour),
	}).Error; err != nil {
		t.Fatalf("bump extension timestamps: %v", err)
	}

	moduleID := sql.NullString{String: xid.New().String(), Valid: true}
	if err := EnsureI18nMeta(rs, "auth", moduleID); err != nil {
		t.Fatalf("EnsureI18nMeta: %v", err)
	}

	canonicals, err := meta.ListDeclarations(db, meta.DeclarationQuery{
		Application: "auth", Path: "go://i18n/auth", PreloadTree: true,
	})
	if err != nil || len(canonicals) != 1 {
		t.Fatalf("canonical list: n=%d err=%v", len(canonicals), err)
	}
	if len(canonicals[0].Services) != 3 {
		t.Fatalf("want 3 services on canonical path, got %d", len(canonicals[0].Services))
	}
	exts, err := meta.ListDeclarations(db, meta.DeclarationQuery{
		Application: "auth", Path: ext.Path, PreloadTree: true,
	})
	if err != nil || len(exts) != 1 {
		t.Fatalf("ext list: n=%d err=%v", len(exts), err)
	}
	if len(exts[0].Services) != 0 {
		t.Fatalf("extension must not receive built-in services, got %d", len(exts[0].Services))
	}

	var effective meta.Model
	if err := db.Preload("Services").Where("name = ? AND application = ?", "I18n", "auth").Take(&effective).Error; err != nil {
		t.Fatalf("load effective: %v", err)
	}
	if effective.Path != "go://i18n/auth" {
		t.Fatalf("effective Path = %q, want go://i18n/auth", effective.Path)
	}
	if effective.Id.String != canonicals[0].Id.String {
		t.Fatalf("effective id = %q, want canonical id %q (not extension %q)", effective.Id.String, canonicals[0].Id.String, ext.Id.String)
	}
	if len(effective.Services) != 3 {
		t.Fatalf("want 3 effective services, got %d", len(effective.Services))
	}
}

func TestEnsureI18nMetaTipCreatedAtWins(t *testing.T) {
	rs := newTestScope(t)
	db := rs.Session().DB
	migrateI18nDualStoreTables(t, db)

	ext := &meta.Model{
		BaseModel: meta.BaseModel{
			Id: sql.NullString{String: xid.New().String(), Valid: true},
		},
		Name:        "I18n",
		Path:        "modules/ext/created_at_tip.ts",
		Application: "auth",
		ClassName:   "I18n",
		Abstract:    true,
		Readonly:    true,
	}
	if err := meta.PersistModelTreeAsRaw(db, ext); err != nil {
		t.Fatalf("seed extension: %v", err)
	}
	past := db.NowFunc().Add(-time.Hour)
	future := db.NowFunc().Add(2 * time.Hour)
	if err := db.Table("meta_raw_model").Where("id = ?", ext.Id.String).UpdateColumns(map[string]any{
		"created_at": future,
		"updated_at": past,
	}).Error; err != nil {
		t.Fatalf("bump timestamps: %v", err)
	}

	if err := EnsureI18nMeta(rs, "auth", sql.NullString{}); err != nil {
		t.Fatalf("EnsureI18nMeta: %v", err)
	}
	canonicals, err := meta.ListDeclarations(db, meta.DeclarationQuery{
		Application: "auth", Path: "go://i18n/auth",
	})
	if err != nil || len(canonicals) != 1 {
		t.Fatalf("canonical: n=%d err=%v", len(canonicals), err)
	}
	if !canonicals[0].CreatedAt.After(future) {
		t.Fatalf("canonical tip CreatedAt=%v, want after sibling CreatedAt=%v", canonicals[0].CreatedAt, future)
	}
}

func TestEnsureI18nMetaUpdatesModuleID(t *testing.T) {
	rs := newTestScope(t)
	db := rs.Session().DB
	migrateI18nDualStoreTables(t, db)
	if err := EnsureI18nMeta(rs, "auth", sql.NullString{}); err != nil {
		t.Fatal(err)
	}
	moduleID := sql.NullString{String: xid.New().String(), Valid: true}
	if err := EnsureI18nMeta(rs, "auth", moduleID); err != nil {
		t.Fatalf("update module id: %v", err)
	}
	decls, err := meta.ListDeclarations(db, meta.DeclarationQuery{Application: "auth", Name: "I18n"})
	if err != nil || len(decls) != 1 {
		t.Fatalf("list: n=%d err=%v", len(decls), err)
	}
	if decls[0].ModuleId.String != moduleID.String {
		t.Fatalf("ModuleId = %q, want %q", decls[0].ModuleId.String, moduleID.String)
	}
}

func TestEnsureI18nMetaErrorPaths(t *testing.T) {
	rs := newTestScope(t)
	db := rs.Session().DB
	migrateI18nDualStoreTables(t, db)

	if err := db.Exec("PRAGMA query_only = ON").Error; err != nil {
		t.Fatal(err)
	}
	if err := EnsureI18nMeta(rs, "auth", sql.NullString{}); err == nil || !strings.Contains(err.Error(), "create meta_raw_model") {
		t.Fatalf("expected create meta_raw_model error, got %v", err)
	}
	_ = db.Exec("PRAGMA query_only = OFF")

	if err := EnsureI18nMeta(rs, "crm", sql.NullString{}); err != nil {
		t.Fatal(err)
	}
	decls, err := meta.ListDeclarations(db, meta.DeclarationQuery{Application: "crm", Name: "I18n"})
	if err != nil || len(decls) != 1 {
		t.Fatalf("crm decl: n=%d err=%v", len(decls), err)
	}
	if err := db.Exec("DELETE FROM meta_raw_service WHERE model_id = ?", decls[0].Id.String).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("PRAGMA query_only = ON").Error; err != nil {
		t.Fatal(err)
	}
	if err := EnsureI18nMeta(rs, "crm", sql.NullString{}); err == nil || !strings.Contains(err.Error(), "create declaration service") {
		t.Fatalf("expected create declaration service error, got %v", err)
	}
	_ = db.Exec("PRAGMA query_only = OFF")

	if err := EnsureI18nMeta(rs, "erp", sql.NullString{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("PRAGMA query_only = ON").Error; err != nil {
		t.Fatal(err)
	}
	if err := EnsureI18nMeta(rs, "erp", sql.NullString{}); err == nil || !strings.Contains(err.Error(), "promote declaration tip") {
		t.Fatalf("expected promote tip error from EnsureI18nMeta, got %v", err)
	}
	_ = db.Exec("PRAGMA query_only = OFF")
}

func TestEnsureI18nMetaLookupErrors(t *testing.T) {
	rs := newTestScope(t)
	db := rs.Session().DB
	migrateI18nDualStoreTables(t, db)
	if err := db.Exec("ALTER TABLE meta_raw_model RENAME COLUMN path TO path_broken").Error; err != nil {
		t.Fatal(err)
	}
	if err := EnsureI18nMeta(rs, "auth", sql.NullString{}); err == nil || !strings.Contains(err.Error(), "lookup declaration") {
		t.Fatalf("expected lookup declaration error, got %v", err)
	}
}

func TestEnsureTerminologyEditorAllowsBranches(t *testing.T) {
	rs := newTestScope(t)
	db := rs.Session().DB
	migrateI18nDualStoreTables(t, db)
	if err := db.Exec(`CREATE TABLE auth_role (
		id TEXT PRIMARY KEY,
		code TEXT,
		deleted_at DATETIME
	)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE auth_role_method_access (
		id TEXT PRIMARY KEY,
		role_id TEXT,
		meta_application_id TEXT,
		meta_model_id TEXT,
		meta_service_id TEXT,
		mode TEXT,
		source TEXT,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	)`).Error; err != nil {
		t.Fatal(err)
	}

	// No role + no TranslationTerm → skip quietly.
	if err := ensureTerminologyEditorAllows(db, "web"); err != nil {
		t.Fatalf("missing role: %v", err)
	}

	roleID := xid.New().String()
	if err := db.Exec(`INSERT INTO auth_role (id, code) VALUES (?, ?)`, roleID, terminologyEditorRoleCode).Error; err != nil {
		t.Fatal(err)
	}
	// Role present but TranslationTerm missing → skip quietly.
	if err := ensureTerminologyEditorAllows(db, "web"); err != nil {
		t.Fatalf("missing TranslationTerm: %v", err)
	}

	if err := db.Exec("ALTER TABLE auth_role RENAME COLUMN code TO code_broken").Error; err != nil {
		t.Fatal(err)
	}
	if err := ensureTerminologyEditorAllows(db, "web"); err == nil || !strings.Contains(err.Error(), "lookup Terminology Editor role") {
		t.Fatalf("expected role lookup error, got %v", err)
	}
}

func TestEnsureI18nMetaSaveModuleAndServiceLookupErrors(t *testing.T) {
	rs := newTestScope(t)
	db := rs.Session().DB
	migrateI18nDualStoreTables(t, db)
	if err := EnsureI18nMeta(rs, "auth", sql.NullString{}); err != nil {
		t.Fatal(err)
	}

	if err := db.Exec("PRAGMA query_only = ON").Error; err != nil {
		t.Fatal(err)
	}
	moduleID := sql.NullString{String: xid.New().String(), Valid: true}
	if err := EnsureI18nMeta(rs, "auth", moduleID); err == nil || !strings.Contains(err.Error(), "update declaration module") {
		t.Fatalf("expected update declaration module error, got %v", err)
	}
	_ = db.Exec("PRAGMA query_only = OFF")

	if err := db.Exec("ALTER TABLE meta_model RENAME COLUMN name TO name_broken").Error; err != nil {
		t.Fatal(err)
	}
	if err := EnsureI18nMeta(rs, "auth", sql.NullString{}); err == nil || !strings.Contains(err.Error(), "flush I18n effective") {
		t.Fatalf("expected flush effective error, got %v", err)
	}
}

func TestEnsureTerminologyEditorAccessLookupAndSeedErrors(t *testing.T) {
	rs := newTestScope(t)
	db := rs.Session().DB
	migrateI18nDualStoreTables(t, db)
	if err := db.Exec(`CREATE TABLE auth_role (
		id TEXT PRIMARY KEY,
		code TEXT,
		deleted_at DATETIME
	)`).Error; err != nil {
		t.Fatal(err)
	}
	roleID := xid.New().String()
	if err := db.Exec(`INSERT INTO auth_role (id, code) VALUES (?, ?)`, roleID, terminologyEditorRoleCode).Error; err != nil {
		t.Fatal(err)
	}
	seedTranslationTermEffective(t, db, "auth")
	if err := db.Exec(`CREATE TABLE auth_role_method_access (
		id TEXT PRIMARY KEY,
		role_id TEXT
	)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := ensureTerminologyEditorAllows(db, "auth"); err == nil || !strings.Contains(err.Error(), "lookup RoleMethodAccess") {
		t.Fatalf("expected RoleMethodAccess lookup error, got %v", err)
	}

	if err := db.Exec("DROP TABLE auth_role_method_access").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE auth_role_method_access (
		id TEXT PRIMARY KEY,
		role_id TEXT,
		meta_application_id TEXT,
		meta_model_id TEXT,
		meta_service_id TEXT,
		mode TEXT,
		source TEXT,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("PRAGMA query_only = ON").Error; err != nil {
		t.Fatal(err)
	}
	if err := ensureTerminologyEditorAllows(db, "auth"); err == nil || !strings.Contains(err.Error(), "seed RoleMethodAccess") {
		t.Fatalf("expected seed RoleMethodAccess error, got %v", err)
	}
	_ = db.Exec("PRAGMA query_only = OFF")
}

func TestLoadEffectiveTranslationTermServiceIDsErrors(t *testing.T) {
	rs := newTestScope(t)
	db := rs.Session().DB
	migrateI18nDualStoreTables(t, db)
	if _, err := loadEffectiveTranslationTermServiceIDs(db, "crm"); err == nil || !errors.Is(err, gorm.ErrRecordNotFound) && !strings.Contains(err.Error(), "record not found") {
		// gorm Take returns ErrRecordNotFound
		if err == nil {
			t.Fatal("expected missing TranslationTerm model error")
		}
	}

	seedTranslationTermEffective(t, db, "erp")
	if err := db.Exec("ALTER TABLE meta_service RENAME COLUMN name TO name_broken").Error; err != nil {
		t.Fatal(err)
	}
	if _, err := loadEffectiveTranslationTermServiceIDs(db, "erp"); err == nil || !strings.Contains(err.Error(), "lookup effective Service") {
		t.Fatalf("expected effective service lookup error, got %v", err)
	}
}
