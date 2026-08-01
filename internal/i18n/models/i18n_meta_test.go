// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package models

import (
	"database/sql"
	"testing"

	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/rs/xid"
)

func TestEnsureI18nMetaCreatesModelAndServices(t *testing.T) {
	rs := newTestScope(t)
	if err := rs.Session().AutoMigrate(&meta.Model{}, &meta.Service{}); err != nil {
		t.Fatalf("automigrate: %v", err)
	}

	moduleID := sql.NullString{String: xid.New().String(), Valid: true}
	if err := EnsureI18nMeta(rs, "auth", moduleID); err != nil {
		t.Fatalf("EnsureI18nMeta: %v", err)
	}
	if err := EnsureI18nMeta(rs, "auth", moduleID); err != nil {
		t.Fatalf("EnsureI18nMeta idempotent: %v", err)
	}

	var model meta.Model
	if err := rs.Session().Where("name = ? AND application = ?", "I18n", "auth").Take(&model).Error; err != nil {
		t.Fatalf("lookup Model: %v", err)
	}
	if !model.Abstract || !model.Readonly {
		t.Fatalf("expected abstract readonly I18n model: %+v", model)
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
	if err := db.AutoMigrate(&meta.Model{}, &meta.Service{}); err != nil {
		t.Fatal(err)
	}
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
}
