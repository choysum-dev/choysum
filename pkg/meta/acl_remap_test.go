// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

import (
	"database/sql"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestRemapACLToEffectiveModelIDs(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:acl-remap?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := EnsureDualStoreTables(db); err != nil {
		t.Fatalf("ensure dual store: %v", err)
	}

	// Minimal auth ACL tables.
	for _, ddl := range []string{
		`CREATE TABLE auth_role_record_rule (
			id TEXT PRIMARY KEY,
			meta_model_id TEXT,
			meta_application_id TEXT
		)`,
		`CREATE TABLE auth_role_field_rule (
			id TEXT PRIMARY KEY,
			meta_model_id TEXT,
			meta_field_id TEXT,
			meta_application_id TEXT
		)`,
		`CREATE TABLE auth_role_method_access (
			id TEXT PRIMARY KEY,
			meta_model_id TEXT,
			meta_service_id TEXT,
			meta_application_id TEXT
		)`,
	} {
		if err := db.Exec(ddl).Error; err != nil {
			t.Fatalf("create acl table: %v", err)
		}
	}

	ts := time.Now().UTC()
	shell := &Model{
		BaseModel:   BaseModel{Id: sql.NullString{String: "shell-id", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:        "User",
		Path:        "/shell",
		Application: "auth",
		ModuleId:    sql.NullString{String: "mod", Valid: true},
	}
	eff := &Model{
		BaseModel:   BaseModel{Id: sql.NullString{String: "eff-id", Valid: true}, CreatedAt: ts, UpdatedAt: ts.Add(time.Hour)},
		Name:        "User",
		Path:        "/eff",
		Application: "auth",
	}
	if err := db.Create(shell).Error; err != nil {
		t.Fatalf("create shell: %v", err)
	}
	if err := db.Create(eff).Error; err != nil {
		t.Fatalf("create eff: %v", err)
	}
	oldField := &Field{
		BaseModel: BaseModel{Id: sql.NullString{String: "old-field", Valid: true}},
		Name:      "PasswordHash",
		ModelId:   shell.Id,
	}
	newField := &Field{
		BaseModel: BaseModel{Id: sql.NullString{String: "new-field", Valid: true}},
		Name:      "PasswordHash",
		ModelId:   eff.Id,
	}
	if err := db.Create(oldField).Error; err != nil {
		t.Fatalf("create old field: %v", err)
	}
	if err := db.Create(newField).Error; err != nil {
		t.Fatalf("create new field: %v", err)
	}

	if err := db.Exec(
		`INSERT INTO auth_role_record_rule (id, meta_model_id) VALUES (?, ?)`,
		"rr1", "shell-id",
	).Error; err != nil {
		t.Fatalf("seed record rule: %v", err)
	}
	if err := db.Exec(
		`INSERT INTO auth_role_field_rule (id, meta_model_id, meta_field_id) VALUES (?, ?, ?)`,
		"fr1", "shell-id", "old-field",
	).Error; err != nil {
		t.Fatalf("seed field rule: %v", err)
	}
	if err := db.Exec(
		`INSERT INTO auth_role_method_access (id, meta_model_id) VALUES (?, ?)`,
		"ma1", "shell-id",
	).Error; err != nil {
		t.Fatalf("seed method access: %v", err)
	}

	if err := RemapACLToEffectiveModelIDs(db); err != nil {
		t.Fatalf("RemapACLToEffectiveModelIDs: %v", err)
	}

	var rrModelID, frModelID, frFieldID, maModelID string
	if err := db.Raw(`SELECT meta_model_id FROM auth_role_record_rule WHERE id = ?`, "rr1").Scan(&rrModelID).Error; err != nil {
		t.Fatalf("read rr: %v", err)
	}
	if err := db.Raw(`SELECT meta_model_id, meta_field_id FROM auth_role_field_rule WHERE id = ?`, "fr1").Row().Scan(&frModelID, &frFieldID); err != nil {
		t.Fatalf("read fr: %v", err)
	}
	if err := db.Raw(`SELECT meta_model_id FROM auth_role_method_access WHERE id = ?`, "ma1").Scan(&maModelID).Error; err != nil {
		t.Fatalf("read ma: %v", err)
	}
	if rrModelID != "eff-id" || maModelID != "eff-id" || frModelID != "eff-id" {
		t.Fatalf("expected meta_model_id remapped to eff-id, got rr=%q fr=%q ma=%q", rrModelID, frModelID, maModelID)
	}
	if frFieldID != "new-field" {
		t.Fatalf("expected meta_field_id remapped to new-field, got %q", frFieldID)
	}

	// Idempotent.
	if err := RemapACLToEffectiveModelIDs(db); err != nil {
		t.Fatalf("second RemapACL: %v", err)
	}
	if err := RemapACLToEffectiveModelIDs(nil); err == nil {
		t.Fatal("expected nil db error")
	}
}
