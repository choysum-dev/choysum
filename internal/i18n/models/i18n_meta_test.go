// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package models

import (
	"database/sql"
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

	var rawModel meta.RawModel
	if err := rs.Session().Where("name = ? AND application = ?", "I18n", "auth").Take(&rawModel).Error; err != nil {
		t.Fatalf("lookup raw Model: %v", err)
	}
	if !rawModel.Abstract || !rawModel.Readonly {
		t.Fatalf("expected abstract readonly raw I18n model: %+v", rawModel)
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

	if err := EnsureI18nMeta(rs, "web", sql.NullString{}); err != nil {
		t.Fatalf("EnsureI18nMeta: %v", err)
	}

	var count int64
	if err := db.Table("auth_role_method_access").Where("role_id = ?", roleID).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("expected 2 RoleMethodAccess rows, got %d", count)
	}

	// Second run exercises the count>0 continue branch.
	if err := EnsureI18nMeta(rs, "web", sql.NullString{}); err != nil {
		t.Fatalf("EnsureI18nMeta idempotent seed: %v", err)
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
	// No meta tables → early return.
	if err := EnsureI18nMeta(rs, "auth", sql.NullString{}); err != nil {
		t.Fatalf("missing tables: %v", err)
	}
}

func TestEnsureI18nMetaPrefersCanonicalPath(t *testing.T) {
	rs := newTestScope(t)
	db := rs.Session().DB
	migrateI18nDualStoreTables(t, db)

	// Newer same-name extension must not receive built-in I18n services.
	ext := meta.RawModel{
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
	if err := db.Create(&ext).Error; err != nil {
		t.Fatalf("seed extension raw: %v", err)
	}
	// Bump timestamps so Order(created_at DESC) would prefer the extension if name-only.
	if err := db.Model(&ext).Updates(map[string]any{
		"created_at": db.NowFunc().Add(time.Hour),
		"updated_at": db.NowFunc().Add(time.Hour),
	}).Error; err != nil {
		t.Fatalf("bump extension timestamps: %v", err)
	}

	moduleID := sql.NullString{String: xid.New().String(), Valid: true}
	if err := EnsureI18nMeta(rs, "auth", moduleID); err != nil {
		t.Fatalf("EnsureI18nMeta: %v", err)
	}

	var canonical meta.RawModel
	if err := db.Where("path = ? AND application = ?", "go://i18n/auth", "auth").Take(&canonical).Error; err != nil {
		t.Fatalf("canonical raw missing: %v", err)
	}
	var canonicalSvc int64
	if err := db.Model(&meta.RawService{}).Where("model_id = ?", canonical.Id.String).Count(&canonicalSvc).Error; err != nil {
		t.Fatal(err)
	}
	if canonicalSvc != 3 {
		t.Fatalf("want 3 services on canonical path, got %d", canonicalSvc)
	}
	var extSvc int64
	if err := db.Model(&meta.RawService{}).Where("model_id = ?", ext.Id.String).Count(&extSvc).Error; err != nil {
		t.Fatal(err)
	}
	if extSvc != 0 {
		t.Fatalf("extension must not receive built-in services, got %d", extSvc)
	}

	var effective meta.Model
	if err := db.Preload("Services").Where("name = ? AND application = ?", "I18n", "auth").Take(&effective).Error; err != nil {
		t.Fatalf("load effective: %v", err)
	}
	if effective.Path != "go://i18n/auth" {
		t.Fatalf("effective Path = %q, want go://i18n/auth", effective.Path)
	}
	if len(effective.Services) != 3 {
		t.Fatalf("want 3 effective services, got %d", len(effective.Services))
	}
}

func TestPromoteI18nCanonicalTip(t *testing.T) {
	if err := promoteI18nCanonicalTip(nil, "auth", nil); err == nil || !strings.Contains(err.Error(), "canonical raw is nil") {
		t.Fatalf("nil canonical: %v", err)
	}
	rs := newTestScope(t)
	db := rs.Session().DB
	migrateI18nDualStoreTables(t, db)
	if err := promoteI18nCanonicalTip(db, "auth", &meta.RawModel{}); err == nil || !strings.Contains(err.Error(), "canonical raw is nil") {
		t.Fatalf("invalid id: %v", err)
	}

	raw, err := ensureI18nRawModel(db, "auth", sql.NullString{})
	if err != nil {
		t.Fatalf("ensureI18nRawModel: %v", err)
	}
	if err := db.Migrator().DropTable(&meta.RawModel{}); err != nil {
		t.Fatal(err)
	}
	if err := promoteI18nCanonicalTip(db, "auth", raw); err == nil || !strings.Contains(err.Error(), "load I18n sibling tips") {
		t.Fatalf("expected load error, got %v", err)
	}

	rs2 := newTestScope(t)
	db2 := rs2.Session().DB
	migrateI18nDualStoreTables(t, db2)
	raw2, err := ensureI18nRawModel(db2, "crm", sql.NullString{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db2.Exec("PRAGMA query_only = ON").Error; err != nil {
		t.Fatal(err)
	}
	if err := promoteI18nCanonicalTip(db2, "crm", raw2); err == nil || !strings.Contains(err.Error(), "promote I18n canonical tip") {
		t.Fatalf("expected promote error, got %v", err)
	}
	_ = db2.Exec("PRAGMA query_only = OFF")
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
	var rawModel meta.RawModel
	if err := db.Where("name = ? AND application = ?", "I18n", "auth").Take(&rawModel).Error; err != nil {
		t.Fatal(err)
	}
	if rawModel.ModuleId.String != moduleID.String {
		t.Fatalf("raw ModuleId = %q, want %q", rawModel.ModuleId.String, moduleID.String)
	}
}

func TestEnsureI18nMetaErrorPaths(t *testing.T) {
	rs := newTestScope(t)
	db := rs.Session().DB
	migrateI18nDualStoreTables(t, db)

	// Force create raw Model failure via query_only.
	if err := db.Exec("PRAGMA query_only = ON").Error; err != nil {
		t.Fatal(err)
	}
	if err := EnsureI18nMeta(rs, "auth", sql.NullString{}); err == nil || !strings.Contains(err.Error(), "create I18n raw Model") {
		t.Fatalf("expected create I18n raw Model error, got %v", err)
	}
	_ = db.Exec("PRAGMA query_only = OFF")

	// Seed model, then force raw service create failure.
	if err := EnsureI18nMeta(rs, "crm", sql.NullString{}); err != nil {
		t.Fatal(err)
	}
	var rawModel meta.RawModel
	if err := db.Where("name = ? AND application = ?", "I18n", "crm").Take(&rawModel).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Where("model_id = ?", rawModel.Id.String).Delete(&meta.RawService{}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("PRAGMA query_only = ON").Error; err != nil {
		t.Fatal(err)
	}
	if err := EnsureI18nMeta(rs, "crm", sql.NullString{}); err == nil || !strings.Contains(err.Error(), "create raw Service") {
		t.Fatalf("expected create raw Service error, got %v", err)
	}
	_ = db.Exec("PRAGMA query_only = OFF")
}

func TestEnsureI18nMetaLookupErrors(t *testing.T) {
	rs := newTestScope(t)
	db := rs.Session().DB
	migrateI18nDualStoreTables(t, db)
	// Corrupt meta_raw_model so Take fails with a non-RecordNotFound error.
	if err := db.Exec("ALTER TABLE meta_raw_model RENAME COLUMN path TO path_broken").Error; err != nil {
		t.Fatal(err)
	}
	if err := EnsureI18nMeta(rs, "auth", sql.NullString{}); err == nil || !strings.Contains(err.Error(), "lookup I18n raw Model") {
		t.Fatalf("expected lookup I18n raw Model error, got %v", err)
	}
}

func TestEnsureI18nRawServicesLookupError(t *testing.T) {
	rs := newTestScope(t)
	db := rs.Session().DB
	migrateI18nDualStoreTables(t, db)
	raw, err := ensureI18nRawModel(db, "auth", sql.NullString{})
	if err != nil {
		t.Fatalf("ensureI18nRawModel: %v", err)
	}
	if err := db.Exec("ALTER TABLE meta_raw_service RENAME COLUMN name TO name_broken").Error; err != nil {
		t.Fatal(err)
	}
	if err := ensureI18nRawServices(db, raw); err == nil || !strings.Contains(err.Error(), "lookup raw Service") {
		t.Fatalf("expected lookup raw Service error, got %v", err)
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

	// No terminology.editor role → empty roleID early return.
	if err := ensureTerminologyEditorAllows(db, map[string]string{"SearchTerms": "svc-1"}); err != nil {
		t.Fatalf("missing role: %v", err)
	}

	roleID := xid.New().String()
	if err := db.Exec(`INSERT INTO auth_role (id, code) VALUES (?, ?)`, roleID, terminologyEditorRoleCode).Error; err != nil {
		t.Fatal(err)
	}
	// Empty serviceID skipped.
	if err := ensureTerminologyEditorAllows(db, map[string]string{"SearchTerms": "", "UpdateTerm": ""}); err != nil {
		t.Fatalf("empty service ids: %v", err)
	}

	// Role lookup SQL error.
	if err := db.Exec("ALTER TABLE auth_role RENAME COLUMN code TO code_broken").Error; err != nil {
		t.Fatal(err)
	}
	if err := ensureTerminologyEditorAllows(db, map[string]string{"SearchTerms": "svc-1"}); err == nil || !strings.Contains(err.Error(), "lookup Terminology Editor role") {
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

	// Force Save(moduleId) failure.
	if err := db.Exec("PRAGMA query_only = ON").Error; err != nil {
		t.Fatal(err)
	}
	moduleID := sql.NullString{String: xid.New().String(), Valid: true}
	if err := EnsureI18nMeta(rs, "auth", moduleID); err == nil || !strings.Contains(err.Error(), "update I18n raw Model module") {
		t.Fatalf("expected update raw module error, got %v", err)
	}
	_ = db.Exec("PRAGMA query_only = OFF")

	// Corrupt meta_model for lookup effective Model error after raw write path succeeds.
	if err := db.Exec("ALTER TABLE meta_model RENAME COLUMN name TO name_broken").Error; err != nil {
		t.Fatal(err)
	}
	if err := EnsureI18nMeta(rs, "auth", sql.NullString{}); err == nil || !strings.Contains(err.Error(), "recompute I18n effective") {
		t.Fatalf("expected recompute effective error, got %v", err)
	}
}

func TestEnsureTerminologyEditorAccessLookupAndSeedErrors(t *testing.T) {
	rs := newTestScope(t)
	db := rs.Session().DB
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
	if err := db.Exec(`CREATE TABLE auth_role_method_access (
		id TEXT PRIMARY KEY,
		role_id TEXT
	)`).Error; err != nil {
		t.Fatal(err)
	}
	// Count fails because meta_service_id / deleted_at columns are missing.
	if err := ensureTerminologyEditorAllows(db, map[string]string{"SearchTerms": "svc-1"}); err == nil || !strings.Contains(err.Error(), "lookup RoleMethodAccess") {
		t.Fatalf("expected RoleMethodAccess lookup error, got %v", err)
	}

	// Recreate proper access table then force seed insert failure with query_only.
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
	if err := ensureTerminologyEditorAllows(db, map[string]string{"SearchTerms": "svc-1"}); err == nil || !strings.Contains(err.Error(), "seed RoleMethodAccess") {
		t.Fatalf("expected seed RoleMethodAccess error, got %v", err)
	}
	_ = db.Exec("PRAGMA query_only = OFF")
}

func TestEnsureI18nRawServicesInvalidRaw(t *testing.T) {
	rs := newTestScope(t)
	migrateI18nDualStoreTables(t, rs.Session().DB)

	if err := ensureI18nRawServices(rs.Session().DB, nil); err == nil || !strings.Contains(err.Error(), "I18n raw Model is nil") {
		t.Fatalf("nil raw: %v", err)
	}
	if err := ensureI18nRawServices(rs.Session().DB, &meta.RawModel{}); err == nil || !strings.Contains(err.Error(), "I18n raw Model is nil") {
		t.Fatalf("invalid id raw: %v", err)
	}
}

func TestLoadEffectiveI18nServiceIDsErrors(t *testing.T) {
	rs := newTestScope(t)
	db := rs.Session().DB
	migrateI18nDualStoreTables(t, db)
	if err := EnsureI18nMeta(rs, "crm", sql.NullString{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("ALTER TABLE meta_model RENAME COLUMN name TO name_broken").Error; err != nil {
		t.Fatal(err)
	}
	if _, err := loadEffectiveI18nServiceIDs(db, "crm"); err == nil || !strings.Contains(err.Error(), "lookup I18n effective Model") {
		t.Fatalf("expected effective model lookup error, got %v", err)
	}

	rs2 := newTestScope(t)
	db2 := rs2.Session().DB
	migrateI18nDualStoreTables(t, db2)
	if err := EnsureI18nMeta(rs2, "erp", sql.NullString{}); err != nil {
		t.Fatal(err)
	}
	if err := db2.Exec("ALTER TABLE meta_service RENAME COLUMN name TO name_broken").Error; err != nil {
		t.Fatal(err)
	}
	if _, err := loadEffectiveI18nServiceIDs(db2, "erp"); err == nil || !strings.Contains(err.Error(), "lookup effective Service") {
		t.Fatalf("expected effective service lookup error, got %v", err)
	}
}
