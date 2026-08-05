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

func TestLookupEffectiveModel_PrefersEmptyModuleID(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:lookup-eff?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := EnsureDualStoreTables(db); err != nil {
		t.Fatalf("ensure dual store: %v", err)
	}
	ts := time.Now().UTC()
	shell := &Model{
		BaseModel:   BaseModel{Id: sql.NullString{String: "shell", Valid: true}, CreatedAt: ts, UpdatedAt: ts.Add(time.Hour)},
		Name:        "Partner",
		Path:        "/shell",
		Application: "partner",
		ModuleId:    sql.NullString{String: "mod-1", Valid: true},
	}
	eff := &Model{
		BaseModel:   BaseModel{Id: sql.NullString{String: "eff", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:        "Partner",
		Path:        "/eff",
		Application: "partner",
	}
	if err := db.Create(shell).Error; err != nil {
		t.Fatalf("create shell: %v", err)
	}
	if err := db.Create(eff).Error; err != nil {
		t.Fatalf("create eff: %v", err)
	}

	got, err := LookupEffectiveModel(db, "partner", "Partner")
	if err != nil {
		t.Fatalf("LookupEffectiveModel: %v", err)
	}
	if got.Id.String != "eff" {
		t.Fatalf("expected empty-module_id row, got %#v", got)
	}

	if _, err := LookupEffectiveModel(db, "partner", "Missing"); !IsEffectiveModelNotFound(err) {
		t.Fatalf("expected not found, got %v", err)
	}
	if _, err := LookupEffectiveModel(nil, "a", "b"); err == nil {
		t.Fatal("expected nil db error")
	}
	if _, err := LookupEffectiveModel(db, "", "x"); err == nil {
		t.Fatal("expected empty key error")
	}
}
