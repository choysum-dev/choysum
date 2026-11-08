// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package ops

import (
	"context"
	"errors"
	"strings"
	"testing"

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
	if err := o.SetWithDemo(context.Background(), true); err != nil {
		t.Fatalf("SetWithDemo: %v", err)
	}
	_ = db.Callback().Query().Remove("ops_hide_first_with_demo_take")
	if !o.ResolveWithDemo(context.Background()) {
		t.Fatal("expected true after create-race fallback update")
	}
}
