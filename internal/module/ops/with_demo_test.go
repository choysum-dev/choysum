// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package ops

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestResolveWithDemo_DefaultFalseThenSetTrue(t *testing.T) {
	t.Parallel()
	db := newOpsTestDB(t)
	o := &Ops{Scope: newOpsTestScope(t, db)}

	if o.ResolveWithDemo(context.Background()) {
		t.Fatal("expected default false")
	}
	if err := o.SetWithDemo(context.Background(), true); err != nil {
		t.Fatalf("SetWithDemo: %v", err)
	}
	if !o.ResolveWithDemo(context.Background()) {
		t.Fatal("expected true after SetWithDemo")
	}
	// Idempotent upsert.
	if err := o.SetWithDemo(context.Background(), true); err != nil {
		t.Fatalf("SetWithDemo again: %v", err)
	}
	if !o.ResolveWithDemo(context.Background()) {
		t.Fatal("expected still true")
	}
}

func TestEnsureMetaAppSettingTableNilDB(t *testing.T) {
	t.Parallel()
	if err := ensureMetaAppSettingTable(nil); err == nil || !strings.Contains(err.Error(), "nil db") {
		t.Fatalf("ensureMetaAppSettingTable(nil) = %v", err)
	}
}

func TestEnsureMetaAppSettingTable_CreateRaceTolerated(t *testing.T) {
	t.Parallel()
	dbPath := filepath.Join(t.TempDir(), "race.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	prev := metaAppSettingCreateTable
	t.Cleanup(func() { metaAppSettingCreateTable = prev })

	// Simulate lost create race: peer created the table, our CreateTable errors.
	metaAppSettingCreateTable = func(m gorm.Migrator) error {
		if err := m.CreateTable(&metaAppSetting{}); err != nil {
			return err
		}
		return errors.New("simulated create race")
	}
	if err := ensureMetaAppSettingTable(db); err != nil {
		t.Fatalf("ensure after simulated race: %v", err)
	}
	if !db.Migrator().HasTable(&metaAppSetting{}) {
		t.Fatal("expected table after race recovery")
	}

	// Create fails and table still missing → surface the create error.
	db2Path := filepath.Join(t.TempDir(), "race-fail.db")
	db2, err := gorm.Open(sqlite.Open(db2Path), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	metaAppSettingCreateTable = func(gorm.Migrator) error {
		return errors.New("disk full")
	}
	err = ensureMetaAppSettingTable(db2)
	if err == nil || !strings.Contains(err.Error(), "disk full") {
		t.Fatalf("ensure create fail = %v, want disk full", err)
	}
}

func TestEnsureMetaAppSettingTable_CreateOnceOnly(t *testing.T) {
	t.Parallel()
	dbPath := filepath.Join(t.TempDir(), "ensure.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if db.Migrator().HasTable(&metaAppSetting{}) {
		t.Fatal("expected no meta_app_setting before ensure")
	}
	if err := ensureMetaAppSettingTable(db); err != nil {
		t.Fatalf("ensure create: %v", err)
	}
	if !db.Migrator().HasTable(&metaAppSetting{}) {
		t.Fatal("expected meta_app_setting after ensure")
	}
	// Second call must be a no-op (no AutoMigrate / alter).
	if err := ensureMetaAppSettingTable(db); err != nil {
		t.Fatalf("ensure existing: %v", err)
	}
}

func TestSetWithDemo_CreatesTableWhenMissing(t *testing.T) {
	t.Parallel()
	dbPath := filepath.Join(t.TempDir(), "latch.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	o := &Ops{Scope: newOpsTestScope(t, db)}
	if db.Migrator().HasTable(&metaAppSetting{}) {
		t.Fatal("expected empty db without meta_app_setting")
	}
	if err := o.SetWithDemo(context.Background(), true); err != nil {
		t.Fatalf("SetWithDemo: %v", err)
	}
	if !db.Migrator().HasTable(&metaAppSetting{}) {
		t.Fatal("expected SetWithDemo to create meta_app_setting when missing")
	}
	if !o.ResolveWithDemo(context.Background()) {
		t.Fatal("expected latch true")
	}
}

func TestSetWithDemo_DoesNotAutoMigrateExistingTable(t *testing.T) {
	t.Parallel()
	db := newOpsTestDB(t)
	o := &Ops{Scope: newOpsTestScope(t, db)}
	// Poison AutoMigrate-style ALTER on this table; ensure path must not touch schema.
	if err := db.Callback().Raw().Before("gorm:raw").Register("ops_forbid_automigrate_ddl", func(tx *gorm.DB) {
		sql := ""
		if tx.Statement != nil {
			sql = strings.ToUpper(tx.Statement.SQL.String())
		}
		if strings.Contains(sql, "ALTER TABLE") && strings.Contains(sql, "META_APP_SETTING") {
			_ = tx.AddError(errors.New("unexpected AutoMigrate alter on meta_app_setting"))
		}
	}); err != nil {
		t.Fatalf("callback: %v", err)
	}
	t.Cleanup(func() { _ = db.Callback().Raw().Remove("ops_forbid_automigrate_ddl") })
	if err := o.SetWithDemo(context.Background(), true); err != nil {
		t.Fatalf("SetWithDemo on existing table: %v", err)
	}
	if !o.ResolveWithDemo(context.Background()) {
		t.Fatal("expected latch true")
	}
}

func TestSetWithDemo_RejectsFalse(t *testing.T) {
	t.Parallel()
	db := newOpsTestDB(t)
	o := &Ops{Scope: newOpsTestScope(t, db)}
	if err := o.SetWithDemo(context.Background(), false); err == nil {
		t.Fatal("expected error for false")
	}
}

func TestSetWithDemo_TakeUnexpectedError(t *testing.T) {
	t.Parallel()
	db := newOpsTestDB(t)
	o := &Ops{Scope: newOpsTestScope(t, db)}
	if err := db.AutoMigrate(&metaAppSetting{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	if err := db.Callback().Query().Before("gorm:query").Register("ops_fail_with_demo_take", func(tx *gorm.DB) {
		if tx.Statement != nil && tx.Statement.Table == "meta_app_setting" {
			_ = tx.AddError(errors.New("take boom"))
		}
	}); err != nil {
		t.Fatalf("callback: %v", err)
	}
	err := o.SetWithDemo(context.Background(), true)
	if err == nil || !strings.Contains(err.Error(), "take boom") {
		t.Fatalf("expected take boom, got %v", err)
	}
}

func TestIsDuplicateKeyError(t *testing.T) {
	t.Parallel()
	if isDuplicateKeyError(nil) {
		t.Fatal("nil")
	}
	if !isDuplicateKeyError(gorm.ErrDuplicatedKey) {
		t.Fatal("ErrDuplicatedKey")
	}
	if !isDuplicateKeyError(errors.New("UNIQUE constraint failed: meta_app_setting.key")) {
		t.Fatal("sqlite unique")
	}
	if !isDuplicateKeyError(errors.New("Duplicate entry 'x' for key 'PRIMARY'")) {
		t.Fatal("mysql duplicate")
	}
	if isDuplicateKeyError(errors.New("disk full")) {
		t.Fatal("non-duplicate")
	}
}

func TestSetWithDemo_NonDuplicateCreateDoesNotRecover(t *testing.T) {
	t.Parallel()
	db := newOpsTestDB(t)
	o := &Ops{Scope: newOpsTestScope(t, db)}
	if err := db.AutoMigrate(&metaAppSetting{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	if err := db.Create(&metaAppSetting{Key: withDemoSettingKey, Value: "false"}).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}
	takes := 0
	if err := db.Callback().Query().Before("gorm:query").Register("ops_hide_first_with_demo_take_nondup", func(tx *gorm.DB) {
		if tx.Statement == nil || tx.Statement.Table != "meta_app_setting" {
			return
		}
		takes++
		if takes == 1 {
			_ = tx.AddError(gorm.ErrRecordNotFound)
		}
	}); err != nil {
		t.Fatalf("query callback: %v", err)
	}
	t.Cleanup(func() { _ = db.Callback().Query().Remove("ops_hide_first_with_demo_take_nondup") })
	if err := db.Callback().Create().Before("gorm:create").Register("ops_fail_with_demo_create_nondup", func(tx *gorm.DB) {
		if tx.Statement != nil && tx.Statement.Table == "meta_app_setting" {
			_ = tx.AddError(errors.New("disk full"))
		}
	}); err != nil {
		t.Fatalf("create callback: %v", err)
	}
	t.Cleanup(func() { _ = db.Callback().Create().Remove("ops_fail_with_demo_create_nondup") })
	err := o.SetWithDemo(context.Background(), true)
	if err == nil || !strings.Contains(err.Error(), "disk full") {
		t.Fatalf("expected disk full, got %v", err)
	}
	var row metaAppSetting
	if err := db.Where(map[string]any{"key": withDemoSettingKey}).Take(&row).Error; err != nil {
		t.Fatalf("take: %v", err)
	}
	if row.Value != "false" {
		t.Fatalf("value=%q want false (no silent recover)", row.Value)
	}
}

func TestSetWithDemo_CreateConflictFallsBackToUpdate(t *testing.T) {
	t.Parallel()
	db := newOpsTestDB(t)
	o := &Ops{Scope: newOpsTestScope(t, db)}
	if err := db.AutoMigrate(&metaAppSetting{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	if err := db.Create(&metaAppSetting{Key: withDemoSettingKey, Value: "false"}).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}
	// First data Take pretends the peer row is invisible so Create runs and hits UNIQUE.
	takes := 0
	if err := db.Callback().Query().Before("gorm:query").Register("ops_hide_first_with_demo_take", func(tx *gorm.DB) {
		if tx.Statement == nil || tx.Statement.Table != "meta_app_setting" {
			return
		}
		takes++
		if takes == 1 {
			_ = tx.AddError(gorm.ErrRecordNotFound)
		}
	}); err != nil {
		t.Fatalf("query callback: %v", err)
	}
	t.Cleanup(func() { _ = db.Callback().Query().Remove("ops_hide_first_with_demo_take") })
	if err := o.SetWithDemo(context.Background(), true); err != nil {
		t.Fatalf("SetWithDemo: %v", err)
	}
	if !o.ResolveWithDemo(context.Background()) {
		t.Fatal("expected true after create-race fallback update")
	}
}
