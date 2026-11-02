// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package service

import (
	"context"
	"strings"
	"testing"
	"time"

	modmeta "github.com/choysum-dev/choysum/internal/module/meta"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	statepkg "github.com/choysum-dev/choysum/pkg/state"
)

func TestRejectAlreadyInstalledAuthModuleSkipsWhenModuleTableMissing(t *testing.T) {
	c, _ := newFreshnessTestCoordinator(t)

	if err := c.rejectAlreadyInstalledAuthModule(context.Background()); err != nil {
		t.Fatalf("rejectAlreadyInstalledAuthModule() error = %v, want nil when module table is missing", err)
	}
}

func TestDefaultAcquireInitLeaseAutoMigratesMissingLockLeaseTable(t *testing.T) {
	c, db := newFreshnessTestCoordinator(t)
	locker := &coordinatorTestLocker{}
	c.lockerFactory = func(scope.Scope) statepkg.Locker {
		return locker
	}

	if db.Migrator().HasTable("meta_lock_lease") {
		t.Fatal("expected missing lock lease table before acquire")
	}

	handle, err := c.defaultAcquireInitLease(context.Background())
	if err != nil {
		t.Fatalf("defaultAcquireInitLease() error = %v", err)
	}
	if handle == nil {
		t.Fatal("expected non-nil lease handle")
	}
	if !db.Migrator().HasTable("meta_lock_lease") {
		t.Fatal("expected lock lease table to be auto-migrated")
	}
	c.defaultReleaseInitLease(handle)
}

func TestDefaultAcquireInitLeaseReturnsGateErrorWhenAutoMigrateFails(t *testing.T) {
	c, db := newFreshnessTestCoordinator(t)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("sql DB: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close sql DB: %v", err)
	}

	handle, err := c.defaultAcquireInitLease(context.Background())
	if err == nil {
		t.Fatal("expected gate error when auto-migrate fails")
	}
	if handle != nil {
		t.Fatalf("expected nil handle, got %#v", handle)
	}
	if got := bootstrapErrorCode(err); got != bootstrapErrCodeGateError {
		t.Fatalf("bootstrapErrorCode(err) = %q, want %q", got, bootstrapErrCodeGateError)
	}
	if !strings.Contains(err.Error(), "failed to initialize the setup lock") {
		t.Fatalf("error = %q, want setup lock initialization failure", err.Error())
	}
}

func TestDefaultUpdateAdminAndMarkerDBLookupFailures(t *testing.T) {
	wirePassword := "$CH$" + strings.Repeat("ab", 32)
	fixedNow := time.Unix(1_700_000_000, 0).UTC()

	t.Run("model_data lookup failure", func(t *testing.T) {
		c, db := newFreshnessTestCoordinator(t)
		c.now = func() time.Time { return fixedNow }
		mustExec(t, db, "CREATE TABLE meta_model_data (id TEXT, module TEXT, name TEXT, model TEXT, res_id TEXT, no_update INTEGER)")

		err := c.defaultUpdateAdminAndMarker(context.Background(), initializeInput{
			AdminUsername: "admin",
			Password:      wirePassword,
		})
		if err == nil {
			t.Fatal("expected admin update failure")
		}
		if got := bootstrapErrorCode(err); got != bootstrapErrCodeAdminUpdateFailed {
			t.Fatalf("bootstrapErrorCode(err) = %q, want %q", got, bootstrapErrCodeAdminUpdateFailed)
		}
		if !strings.Contains(err.Error(), "failed to save administrator setup") {
			t.Fatalf("error = %q, want generic admin save failure", err.Error())
		}
	})

	t.Run("model_data name not found", func(t *testing.T) {
		c, db := newFreshnessTestCoordinator(t)
		c.now = func() time.Time { return fixedNow }
		if err := db.AutoMigrate(&modmeta.ModelData{}); err != nil {
			t.Fatalf("auto migrate model_data: %v", err)
		}

		err := c.defaultUpdateAdminAndMarker(context.Background(), initializeInput{
			AdminUsername: "admin",
			Password:      wirePassword,
		})
		if err == nil {
			t.Fatal("expected admin update failure")
		}
		if got := bootstrapErrorCode(err); got != bootstrapErrCodeAdminUpdateFailed {
			t.Fatalf("bootstrapErrorCode(err) = %q, want %q", got, bootstrapErrCodeAdminUpdateFailed)
		}
		if !strings.Contains(err.Error(), "administrator account reference was not found") {
			t.Fatalf("error = %q, want admin name-not-found message", err.Error())
		}
	})

	t.Run("model lookup failure", func(t *testing.T) {
		c, db := newFreshnessTestCoordinator(t)
		c.now = func() time.Time { return fixedNow }
		mustExec(t, db, "CREATE TABLE meta_model_data (id TEXT, module TEXT, name TEXT, model TEXT, res_id TEXT, no_update INTEGER, deleted_at DATETIME)")
		mustExec(t, db, "INSERT INTO meta_model_data(id, module, name, model, res_id, no_update, deleted_at) VALUES(?, ?, ?, ?, ?, ?, NULL)",
			"data-1", "auth", "user_admin", "auth.User", "user-1", 0)
		mustExec(t, db, "CREATE TABLE meta_model (id TEXT, application TEXT, name TEXT, model_table TEXT)")

		err := c.defaultUpdateAdminAndMarker(context.Background(), initializeInput{
			AdminUsername: "admin",
			Password:      wirePassword,
		})
		if err == nil {
			t.Fatal("expected admin update failure")
		}
		if got := bootstrapErrorCode(err); got != bootstrapErrCodeAdminUpdateFailed {
			t.Fatalf("bootstrapErrorCode(err) = %q, want %q", got, bootstrapErrCodeAdminUpdateFailed)
		}
		if !strings.Contains(err.Error(), "failed to save administrator setup") {
			t.Fatalf("error = %q, want generic admin save failure", err.Error())
		}
	})

	t.Run("effective model not found", func(t *testing.T) {
		c, db := newFreshnessTestCoordinator(t)
		c.now = func() time.Time { return fixedNow }
		if err := db.AutoMigrate(&modmeta.ModelData{}, &meta.Model{}); err != nil {
			t.Fatalf("auto migrate: %v", err)
		}
		if err := db.Create(&modmeta.ModelData{
			Module: "auth", Name: "user_admin", Application: "auth", ModelName: "User", ModelId: "missing", ResID: "user-1",
		}).Error; err != nil {
			t.Fatalf("seed model_data: %v", err)
		}

		err := c.defaultUpdateAdminAndMarker(context.Background(), initializeInput{
			AdminUsername: "admin",
			Password:      wirePassword,
		})
		if err == nil {
			t.Fatal("expected admin update failure")
		}
		if got := bootstrapErrorCode(err); got != bootstrapErrCodeAdminUpdateFailed {
			t.Fatalf("bootstrapErrorCode(err) = %q, want %q", got, bootstrapErrCodeAdminUpdateFailed)
		}
		if !strings.Contains(err.Error(), "administrator account schema is not available") {
			t.Fatalf("error = %q, want schema unavailable message", err.Error())
		}
	})

	t.Run("effective model table missing", func(t *testing.T) {
		c, db := newFreshnessTestCoordinator(t)
		c.now = func() time.Time { return fixedNow }
		if err := db.AutoMigrate(&modmeta.ModelData{}, &meta.Model{}); err != nil {
			t.Fatalf("auto migrate: %v", err)
		}
		authUserModel := &meta.Model{
			Application: "auth",
			Name:        "User",
			ModelTable:  "  ",
			Path:        "/tmp",
		}
		if err := db.Create(authUserModel).Error; err != nil {
			t.Fatalf("seed auth.User: %v", err)
		}
		if err := db.Create(&modmeta.ModelData{
			Module: "auth", Name: "user_admin", Application: "auth", ModelName: "User", ModelId: authUserModel.Id.String, ResID: "user-1",
		}).Error; err != nil {
			t.Fatalf("seed model_data: %v", err)
		}

		err := c.defaultUpdateAdminAndMarker(context.Background(), initializeInput{
			AdminUsername: "admin",
			Password:      wirePassword,
		})
		if err == nil {
			t.Fatal("expected admin update failure")
		}
		if got := bootstrapErrorCode(err); got != bootstrapErrCodeAdminUpdateFailed {
			t.Fatalf("bootstrapErrorCode(err) = %q, want %q", got, bootstrapErrCodeAdminUpdateFailed)
		}
		if !strings.Contains(err.Error(), "administrator account schema is not available") {
			t.Fatalf("error = %q, want schema unavailable message", err.Error())
		}
	})
}

func TestDefaultUpdateAdminAndMarkerSuccessWritesSetupCompletedAt(t *testing.T) {
	c, db := newFreshnessTestCoordinator(t)
	fixedNow := time.Unix(1_700_000_000, 0).UTC()
	c.now = func() time.Time { return fixedNow }
	wirePassword := "$CH$" + strings.Repeat("cd", 32)

	if err := db.AutoMigrate(&modmeta.ModelData{}, &meta.Model{}); err != nil {
		t.Fatalf("auto migrate admin tables: %v", err)
	}
	mustExec(t, db, `CREATE TABLE auth_user (
		id TEXT PRIMARY KEY,
		username TEXT,
		password_hash TEXT,
		updated_at DATETIME
	)`)
	mustExec(t, db, `CREATE TABLE auth_app_setting (
		id TEXT PRIMARY KEY,
		key TEXT UNIQUE,
		value TEXT,
		created_at DATETIME,
		updated_at DATETIME
	)`)
	resID := "admin-user-id"
	mustExec(t, db, "INSERT INTO auth_user(id, username, password_hash, updated_at) VALUES(?, ?, ?, ?)",
		resID, "old-admin", "old-hash", fixedNow)
	authUserModel := &meta.Model{
		Application: "auth",
		Name:        "User",
		ModelTable:  "auth_user",
		Path:        "/tmp",
	}
	if err := db.Create(authUserModel).Error; err != nil {
		t.Fatalf("seed auth.User model: %v", err)
	}
	if err := db.Create(&modmeta.ModelData{
		Module: "auth", Name: "user_admin", Application: "auth", ModelName: "User", ModelId: authUserModel.Id.String, ResID: resID,
	}).Error; err != nil {
		t.Fatalf("seed model_data: %v", err)
	}

	if err := c.defaultUpdateAdminAndMarker(context.Background(), initializeInput{
		AdminUsername: "new-admin",
		Password:      wirePassword,
	}); err != nil {
		t.Fatalf("defaultUpdateAdminAndMarker() error = %v", err)
	}

	var row struct {
		Username     string
		PasswordHash string
	}
	if err := db.Table("auth_user").Where("id = ?", resID).Take(&row).Error; err != nil {
		t.Fatalf("query updated admin user: %v", err)
	}
	if row.Username != "new-admin" || row.PasswordHash == "" || row.PasswordHash == "old-hash" {
		t.Fatalf("unexpected updated admin row: %#v", row)
	}

	var setupRow authAppSettingRow
	if err := db.Table(authAppSettingTable).Where("key = ?", authSetupCompletedAtKey).Take(&setupRow).Error; err != nil {
		t.Fatalf("query %s: %v", authSetupCompletedAtKey, err)
	}
	if setupRow.Value != fixedNow.Format(time.RFC3339) {
		t.Fatalf("%s = %q, want %q", authSetupCompletedAtKey, setupRow.Value, fixedNow.Format(time.RFC3339))
	}

	var initCount int64
	if err := db.Table("meta_setting").Where("key LIKE ?", "system.init.%").Count(&initCount).Error; err == nil && initCount != 0 {
		t.Fatalf("expected no system.init.* meta_setting rows, got %d", initCount)
	}
}

func TestUpsertAuthAppSettingBranches(t *testing.T) {
	t.Run("nil session", func(t *testing.T) {
		if err := upsertAuthAppSetting(nil, "k", "v", time.Now().UTC()); err == nil || !strings.Contains(err.Error(), "database session is not available") {
			t.Fatalf("upsertAuthAppSetting(nil) error = %v, want unavailable session error", err)
		}
	})

	c, db := newFreshnessTestCoordinator(t)
	mustExec(t, db, `CREATE TABLE auth_app_setting (
		id TEXT PRIMARY KEY,
		key TEXT UNIQUE,
		value TEXT,
		created_at DATETIME,
		updated_at DATETIME
	)`)
	session := c.runtimeScope.Session()
	now := time.Unix(1_700_000_000, 0).UTC()

	if err := upsertAuthAppSetting(session, authSetupCompletedAtKey, "first", now); err != nil {
		t.Fatalf("upsertAuthAppSetting(create) error = %v", err)
	}
	if err := upsertAuthAppSetting(session, authSetupCompletedAtKey, "second", now.Add(time.Second)); err != nil {
		t.Fatalf("upsertAuthAppSetting(update) error = %v", err)
	}
	var row authAppSettingRow
	if err := db.Where("key = ?", authSetupCompletedAtKey).Take(&row).Error; err != nil {
		t.Fatalf("query setup row: %v", err)
	}
	if row.Value != "second" {
		t.Fatalf("value = %q, want second", row.Value)
	}

	t.Run("query failure", func(t *testing.T) {
		localDB, err := db.DB()
		if err != nil {
			t.Fatalf("sql DB: %v", err)
		}
		if err := localDB.Close(); err != nil {
			t.Fatalf("close sql DB: %v", err)
		}
		if err := upsertAuthAppSetting(session, authSetupCompletedAtKey, "x", now); err == nil {
			t.Fatal("expected query failure from closed database connection")
		}
	})
}
