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
	oldSvc := &Service{
		BaseModel: BaseModel{Id: sql.NullString{String: "old-svc", Valid: true}},
		Name:      "Read",
		ModelId:   shell.Id,
	}
	newSvc := &Service{
		BaseModel: BaseModel{Id: sql.NullString{String: "new-svc", Valid: true}},
		Name:      "Read",
		ModelId:   eff.Id,
	}
	if err := db.Create(oldSvc).Error; err != nil {
		t.Fatalf("create old service: %v", err)
	}
	if err := db.Create(newSvc).Error; err != nil {
		t.Fatalf("create new service: %v", err)
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
		`INSERT INTO auth_role_method_access (id, meta_model_id, meta_service_id) VALUES (?, ?, ?)`,
		"ma1", "shell-id", "old-svc",
	).Error; err != nil {
		t.Fatalf("seed method access: %v", err)
	}

	if err := remapACLToEffectiveModelIDs(db); err != nil {
		t.Fatalf("remapACLToEffectiveModelIDs: %v", err)
	}

	var rrModelID, frModelID, frFieldID, maModelID, maServiceID string
	if err := db.Raw(`SELECT meta_model_id FROM auth_role_record_rule WHERE id = ?`, "rr1").Scan(&rrModelID).Error; err != nil {
		t.Fatalf("read rr: %v", err)
	}
	if err := db.Raw(`SELECT meta_model_id, meta_field_id FROM auth_role_field_rule WHERE id = ?`, "fr1").Row().Scan(&frModelID, &frFieldID); err != nil {
		t.Fatalf("read fr: %v", err)
	}
	if err := db.Raw(`SELECT meta_model_id, meta_service_id FROM auth_role_method_access WHERE id = ?`, "ma1").Row().Scan(&maModelID, &maServiceID); err != nil {
		t.Fatalf("read ma: %v", err)
	}
	if rrModelID != "eff-id" || maModelID != "eff-id" || frModelID != "eff-id" {
		t.Fatalf("expected meta_model_id remapped to eff-id, got rr=%q fr=%q ma=%q", rrModelID, frModelID, maModelID)
	}
	if frFieldID != "new-field" {
		t.Fatalf("expected meta_field_id remapped to new-field, got %q", frFieldID)
	}
	if maServiceID != "new-svc" {
		t.Fatalf("expected meta_service_id remapped to new-svc, got %q", maServiceID)
	}

	// Idempotent: values stay on effective ids.
	if err := remapACLToEffectiveModelIDs(db); err != nil {
		t.Fatalf("second RemapACL: %v", err)
	}
	var rr2, fr2Model, fr2Field, ma2Model, ma2Svc string
	if err := db.Raw(`SELECT meta_model_id FROM auth_role_record_rule WHERE id = ?`, "rr1").Scan(&rr2).Error; err != nil {
		t.Fatalf("re-read rr: %v", err)
	}
	if err := db.Raw(`SELECT meta_model_id, meta_field_id FROM auth_role_field_rule WHERE id = ?`, "fr1").Row().Scan(&fr2Model, &fr2Field); err != nil {
		t.Fatalf("re-read fr: %v", err)
	}
	if err := db.Raw(`SELECT meta_model_id, meta_service_id FROM auth_role_method_access WHERE id = ?`, "ma1").Row().Scan(&ma2Model, &ma2Svc); err != nil {
		t.Fatalf("re-read ma: %v", err)
	}
	if rr2 != "eff-id" || fr2Model != "eff-id" || fr2Field != "new-field" || ma2Model != "eff-id" || ma2Svc != "new-svc" {
		t.Fatalf("second run changed values: rr=%q frModel=%q frField=%q maModel=%q maSvc=%q", rr2, fr2Model, fr2Field, ma2Model, ma2Svc)
	}

	if err := remapACLToEffectiveModelIDs(nil); err == nil {
		t.Fatal("expected nil db error")
	}
}

func TestRemapACLToEffectiveModelIDs_FieldOnlyWhenModelAlreadyEffective(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:acl-remap-field-only?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := EnsureDualStoreTables(db); err != nil {
		t.Fatalf("ensure dual store: %v", err)
	}
	if err := db.Exec(`CREATE TABLE auth_role_field_rule (
		id TEXT PRIMARY KEY,
		meta_model_id TEXT,
		meta_field_id TEXT,
		meta_application_id TEXT
	)`).Error; err != nil {
		t.Fatalf("create field rule table: %v", err)
	}

	ts := time.Now().UTC()
	eff := &Model{
		BaseModel:   BaseModel{Id: sql.NullString{String: "eff-only", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:        "User",
		Path:        "/eff",
		Application: "auth",
	}
	if err := db.Create(eff).Error; err != nil {
		t.Fatalf("create eff: %v", err)
	}
	staleField := &Field{
		BaseModel: BaseModel{Id: sql.NullString{String: "stale-field", Valid: true}},
		Name:      "Login",
		ModelId:   sql.NullString{String: "gone-model", Valid: true},
	}
	liveField := &Field{
		BaseModel: BaseModel{Id: sql.NullString{String: "live-field", Valid: true}},
		Name:      "Login",
		ModelId:   eff.Id,
	}
	if err := db.Create(staleField).Error; err != nil {
		t.Fatalf("create stale field: %v", err)
	}
	if err := db.Create(liveField).Error; err != nil {
		t.Fatalf("create live field: %v", err)
	}
	// meta_model_id already effective; only field id is stale (no old→new model map).
	if err := db.Exec(
		`INSERT INTO auth_role_field_rule (id, meta_model_id, meta_field_id) VALUES (?, ?, ?)`,
		"fr-stale", "eff-only", "stale-field",
	).Error; err != nil {
		t.Fatalf("seed field rule: %v", err)
	}

	if err := remapACLToEffectiveModelIDs(db); err != nil {
		t.Fatalf("RemapACL: %v", err)
	}
	var modelID, fieldID string
	if err := db.Raw(`SELECT meta_model_id, meta_field_id FROM auth_role_field_rule WHERE id = ?`, "fr-stale").
		Row().Scan(&modelID, &fieldID); err != nil {
		t.Fatalf("read field rule: %v", err)
	}
	if modelID != "eff-only" || fieldID != "live-field" {
		t.Fatalf("expected field-only remap, got model=%q field=%q", modelID, fieldID)
	}
}
