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
)

func TestEnsureTerminologyEditorAllowsEarlyReturns(t *testing.T) {
	if err := EnsureTerminologyEditorAllows(nil, "auth"); err != nil {
		t.Fatalf("nil scope: %v", err)
	}
	if err := EnsureTerminologyEditorAllows(newTestScope(t), "core"); err != nil {
		t.Fatalf("core: %v", err)
	}
	if err := EnsureTerminologyEditorAllows(newTestScope(t), ""); err != nil {
		t.Fatalf("empty app: %v", err)
	}
	if err := EnsureTerminologyEditorAllows(newTestScope(t), "  "); err != nil {
		t.Fatalf("whitespace app: %v", err)
	}
	// No effective catalog / role tables: should no-op rather than error.
	if err := EnsureTerminologyEditorAllows(newTestScope(t), "auth"); err != nil {
		t.Fatalf("auth without catalog: %v", err)
	}
}

func TestEnsureTerminologyEditorAllowsSeedsSearchBrowseUpdateCount(t *testing.T) {
	rs := newTestScope(t)
	db := rs.Session().DB
	if err := meta.EnsureDualStoreTables(db); err != nil {
		t.Fatalf("EnsureDualStoreTables: %v", err)
	}
	for _, ddl := range []string{
		`CREATE TABLE auth_role (
			id TEXT PRIMARY KEY,
			code TEXT,
			deleted_at DATETIME
		)`,
		`CREATE TABLE auth_role_method_access (
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
		)`,
	} {
		if err := db.Exec(ddl).Error; err != nil {
			t.Fatalf("create acl table: %v", err)
		}
	}

	roleID := xid.New().String()
	if err := db.Exec(`INSERT INTO auth_role (id, code) VALUES (?, ?)`, roleID, terminologyEditorRoleCode).Error; err != nil {
		t.Fatalf("insert role: %v", err)
	}

	ts := time.Now().UTC()
	model := &meta.Model{
		BaseModel:   meta.BaseModel{Id: sql.NullString{String: xid.New().String(), Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:        translationTermModelName,
		Path:        "auth/translation_term.ts",
		Application: "auth",
	}
	if err := db.Create(model).Error; err != nil {
		t.Fatalf("create model: %v", err)
	}
	serviceIDs := map[string]string{}
	for _, methodName := range terminologyEditorServiceMethods {
		svc := &meta.Service{
			BaseModel: meta.BaseModel{Id: sql.NullString{String: xid.New().String(), Valid: true}, CreatedAt: ts, UpdatedAt: ts},
			Name:      methodName,
			ModelId:   model.Id,
		}
		if err := db.Create(svc).Error; err != nil {
			t.Fatalf("create service %s: %v", methodName, err)
		}
		serviceIDs[methodName] = svc.Id.String
	}

	if err := EnsureTerminologyEditorAllows(rs, "auth"); err != nil {
		t.Fatalf("EnsureTerminologyEditorAllows: %v", err)
	}

	var count int64
	if err := db.Table(authRoleMethodAccessTable).
		Where("role_id = ? AND deleted_at IS NULL", roleID).
		Count(&count).Error; err != nil {
		t.Fatalf("count acl: %v", err)
	}
	if count != int64(len(terminologyEditorServiceMethods)) {
		t.Fatalf("acl rows = %d, want %d", count, len(terminologyEditorServiceMethods))
	}
	for _, methodName := range terminologyEditorServiceMethods {
		var n int64
		if err := db.Table(authRoleMethodAccessTable).
			Where("role_id = ? AND meta_service_id = ? AND mode = ? AND deleted_at IS NULL", roleID, serviceIDs[methodName], "allow").
			Count(&n).Error; err != nil {
			t.Fatalf("count %s: %v", methodName, err)
		}
		if n != 1 {
			t.Fatalf("%s acl rows = %d", methodName, n)
		}
	}

	// Idempotent: existing rows must not be duplicated.
	if err := EnsureTerminologyEditorAllows(rs, "auth"); err != nil {
		t.Fatalf("second EnsureTerminologyEditorAllows: %v", err)
	}
	if err := db.Table(authRoleMethodAccessTable).
		Where("role_id = ? AND deleted_at IS NULL", roleID).
		Count(&count).Error; err != nil {
		t.Fatalf("recount acl: %v", err)
	}
	if count != int64(len(terminologyEditorServiceMethods)) {
		t.Fatalf("after idempotent seed acl rows = %d", count)
	}
}

func TestEnsureTerminologyEditorAllowsNoopWithoutRoleOrModel(t *testing.T) {
	rs := newTestScope(t)
	db := rs.Session().DB
	if err := meta.EnsureDualStoreTables(db); err != nil {
		t.Fatalf("EnsureDualStoreTables: %v", err)
	}

	// Catalog present but auth ACL tables missing → ensureTerminologyEditorAllows no-op.
	if err := EnsureTerminologyEditorAllows(rs, "auth"); err != nil {
		t.Fatalf("without auth tables: %v", err)
	}

	for _, ddl := range []string{
		`CREATE TABLE auth_role (
			id TEXT PRIMARY KEY,
			code TEXT,
			deleted_at DATETIME
		)`,
		`CREATE TABLE auth_role_method_access (
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
		)`,
	} {
		if err := db.Exec(ddl).Error; err != nil {
			t.Fatalf("create acl table: %v", err)
		}
	}

	// No terminology.editor role → no-op.
	if err := EnsureTerminologyEditorAllows(rs, "auth"); err != nil {
		t.Fatalf("without role: %v", err)
	}

	roleID := xid.New().String()
	if err := db.Exec(`INSERT INTO auth_role (id, code) VALUES (?, ?)`, roleID, terminologyEditorRoleCode).Error; err != nil {
		t.Fatalf("insert role: %v", err)
	}
	// Role present but TranslationTerm model missing → ErrRecordNotFound swallowed.
	if err := EnsureTerminologyEditorAllows(rs, "auth"); err != nil {
		t.Fatalf("without model: %v", err)
	}
}

func TestEnsureTerminologyEditorAllowsServiceLookupError(t *testing.T) {
	rs := newTestScope(t)
	db := rs.Session().DB
	if err := meta.EnsureDualStoreTables(db); err != nil {
		t.Fatalf("EnsureDualStoreTables: %v", err)
	}
	for _, ddl := range []string{
		`CREATE TABLE auth_role (
			id TEXT PRIMARY KEY,
			code TEXT,
			deleted_at DATETIME
		)`,
		`CREATE TABLE auth_role_method_access (
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
		)`,
	} {
		if err := db.Exec(ddl).Error; err != nil {
			t.Fatalf("create acl table: %v", err)
		}
	}
	roleID := xid.New().String()
	if err := db.Exec(`INSERT INTO auth_role (id, code) VALUES (?, ?)`, roleID, terminologyEditorRoleCode).Error; err != nil {
		t.Fatalf("insert role: %v", err)
	}
	ts := time.Now().UTC()
	model := &meta.Model{
		BaseModel:   meta.BaseModel{Id: sql.NullString{String: xid.New().String(), Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:        translationTermModelName,
		Path:        "auth/translation_term.ts",
		Application: "auth",
	}
	if err := db.Create(model).Error; err != nil {
		t.Fatalf("create model: %v", err)
	}
	if err := db.Exec(`ALTER TABLE meta_service RENAME COLUMN name TO name_x`).Error; err != nil {
		t.Fatalf("rename meta_service.name: %v", err)
	}
	err := EnsureTerminologyEditorAllows(rs, "auth")
	if err == nil || !strings.Contains(err.Error(), "lookup effective Service") {
		t.Fatalf("expected service lookup error, got %v", err)
	}
}

func TestSeedRoleMethodAllowCountError(t *testing.T) {
	rs := newTestScope(t)
	db := rs.Session().DB
	if err := db.Exec(`CREATE TABLE auth_role_method_access (id TEXT PRIMARY KEY)`).Error; err != nil {
		t.Fatalf("create broken acl table: %v", err)
	}
	err := seedRoleMethodAllow(db, "role-1", "svc-1", "Search")
	if err == nil || !strings.Contains(err.Error(), "lookup RoleMethodAccess") {
		t.Fatalf("expected count error, got %v", err)
	}
}

func TestEnsureTerminologyEditorAllowsRoleLookupError(t *testing.T) {
	rs := newTestScope(t)
	db := rs.Session().DB
	if err := meta.EnsureDualStoreTables(db); err != nil {
		t.Fatalf("EnsureDualStoreTables: %v", err)
	}
	// auth_role exists but lacks columns referenced by the query.
	if err := db.Exec(`CREATE TABLE auth_role (name TEXT)`).Error; err != nil {
		t.Fatalf("create broken auth_role: %v", err)
	}
	if err := db.Exec(`CREATE TABLE auth_role_method_access (
		id TEXT PRIMARY KEY,
		role_id TEXT,
		meta_service_id TEXT,
		deleted_at DATETIME
	)`).Error; err != nil {
		t.Fatalf("create acl table: %v", err)
	}
	err := EnsureTerminologyEditorAllows(rs, "auth")
	if err == nil || !strings.Contains(err.Error(), "lookup Terminology Editor role") {
		t.Fatalf("expected role lookup error, got %v", err)
	}
}

func TestSeedRoleMethodAllowEmptyServiceID(t *testing.T) {
	rs := newTestScope(t)
	if err := seedRoleMethodAllow(rs.Session().DB, "role", "  ", "Search"); err != nil {
		t.Fatalf("empty service id: %v", err)
	}
}

func TestEnsureTerminologyEditorAllowsSeedCreateError(t *testing.T) {
	rs := newTestScope(t)
	db := rs.Session().DB
	if err := meta.EnsureDualStoreTables(db); err != nil {
		t.Fatalf("EnsureDualStoreTables: %v", err)
	}
	for _, ddl := range []string{
		`CREATE TABLE auth_role (
			id TEXT PRIMARY KEY,
			code TEXT,
			deleted_at DATETIME
		)`,
		`CREATE TABLE auth_role_method_access (
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
		)`,
	} {
		if err := db.Exec(ddl).Error; err != nil {
			t.Fatalf("create acl table: %v", err)
		}
	}
	if err := db.Exec(`CREATE TRIGGER no_acl_insert BEFORE INSERT ON auth_role_method_access
		BEGIN SELECT RAISE(ABORT, 'blocked'); END`).Error; err != nil {
		t.Fatalf("create trigger: %v", err)
	}

	roleID := xid.New().String()
	if err := db.Exec(`INSERT INTO auth_role (id, code) VALUES (?, ?)`, roleID, terminologyEditorRoleCode).Error; err != nil {
		t.Fatalf("insert role: %v", err)
	}
	ts := time.Now().UTC()
	model := &meta.Model{
		BaseModel:   meta.BaseModel{Id: sql.NullString{String: xid.New().String(), Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:        translationTermModelName,
		Path:        "auth/translation_term.ts",
		Application: "auth",
	}
	if err := db.Create(model).Error; err != nil {
		t.Fatalf("create model: %v", err)
	}
	for _, methodName := range terminologyEditorServiceMethods {
		svc := &meta.Service{
			BaseModel: meta.BaseModel{Id: sql.NullString{String: xid.New().String(), Valid: true}, CreatedAt: ts, UpdatedAt: ts},
			Name:      methodName,
			ModelId:   model.Id,
		}
		if err := db.Create(svc).Error; err != nil {
			t.Fatalf("create service %s: %v", methodName, err)
		}
	}

	err := EnsureTerminologyEditorAllows(rs, "auth")
	if err == nil || !strings.Contains(err.Error(), "seed RoleMethodAccess") {
		t.Fatalf("expected seed error, got %v", err)
	}
}
