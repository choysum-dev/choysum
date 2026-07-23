// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	i18nmodels "github.com/choysum-dev/choysum/internal/i18n/models"
	moduleresult "github.com/choysum-dev/choysum/internal/module/artifact/result"
	internaltask "github.com/choysum-dev/choysum/internal/task"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/rs/xid"
	"gorm.io/gorm"
)

type commitStubBuilder struct{}

func (commitStubBuilder) Build() (*moduleresult.BuildResult, error) {
	return &moduleresult.BuildResult{}, nil
}

type commitStubSplitBuilder struct {
	persistCalls int
	persistErr   error
}

func (b *commitStubSplitBuilder) Build() (*moduleresult.BuildResult, error) {
	return &moduleresult.BuildResult{}, nil
}

func (b *commitStubSplitBuilder) BuildWithoutPersist() (*moduleresult.BuildResult, error) {
	return &moduleresult.BuildResult{}, nil
}

func (b *commitStubSplitBuilder) Persist(result *moduleresult.BuildResult) error {
	b.persistCalls++
	return b.persistErr
}

func TestCommitInstallSoftDeleteRestoreAndSave(t *testing.T) {
	runtimeScope := newLifecycleCommitTestScope(t)
	modulePath := t.TempDir()
	i18nDir := filepath.Join(modulePath, "i18n")
	if err := os.MkdirAll(i18nDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(i18nDir, "zh_CN.po"), []byte(`
msgid ""
msgstr ""

msgctxt "a@t"
msgid "Hello"
msgstr "你好"
`), 0o644); err != nil {
		t.Fatal(err)
	}

	mod := &meta.IrModule{
		Name:           "demo",
		Version:        "1.0.0",
		Status:         meta.ToInstall,
		Path:           modulePath,
		ApplicationStr: "auth",
	}
	mod.Id = sql.NullString{String: xid.New().String(), Valid: true}
	mod.DeletedAt = gorm.DeletedAt{Time: time.Now().UTC(), Valid: true}
	if err := runtimeScope.Session().Unscoped().Create(mod).Error; err != nil {
		t.Fatalf("create soft-deleted module: %v", err)
	}
	// Installer typically holds a fresh module descriptor; DeletedAt lives only in DB.
	mod.DeletedAt = gorm.DeletedAt{}

	installer := &moduleInstaller{
		module:        mod,
		runtimeScope:  runtimeScope,
		moduleManager: &ModuleManager{runtimeScope: runtimeScope, jsExecutor: &moduleManagerNoopScriptExecutor{}},
		ctx:           newOpContext(),
		builder:       nil,
	}
	if err := installer.commitInstall(nil, false); err != nil {
		t.Fatalf("commitInstall: %v", err)
	}

	var got meta.IrModule
	if err := runtimeScope.Session().Where("name = ?", "demo").Take(&got).Error; err != nil {
		t.Fatalf("load module: %v", err)
	}
	if got.DeletedAt.Valid {
		t.Fatal("expected soft-delete restored")
	}
	if got.Status != meta.Installed {
		t.Fatalf("status = %q, want installed", got.Status)
	}

	var term i18nmodels.TranslationTerm
	if err := runtimeScope.Session().Table("auth_translation_term").Where("src = ?", "Hello").Take(&term).Error; err != nil {
		t.Fatalf("expected imported term: %v", err)
	}
}

func TestFinalizeInstallNoopHooks(t *testing.T) {
	runtimeScope := newLifecycleCommitTestScope(t)
	installer := &moduleInstaller{
		module:        &meta.IrModule{Name: "demo", Path: t.TempDir()},
		runtimeScope:  runtimeScope,
		moduleManager: &ModuleManager{runtimeScope: runtimeScope, jsExecutor: &moduleManagerNoopScriptExecutor{}},
		ctx:           newOpContext(),
	}
	if err := installer.finalizeInstall(nil); err != nil {
		t.Fatalf("finalizeInstall: %v", err)
	}
}

func TestCommitInstallPersistLaterBranches(t *testing.T) {
	runtimeScope := newLifecycleCommitTestScope(t)
	mod := &meta.IrModule{
		Name:   "demo",
		Path:   t.TempDir(),
		Status: meta.ToInstall,
	}
	mod.Id = sql.NullString{String: xid.New().String(), Valid: true}

	split := &commitStubSplitBuilder{}
	installer := &moduleInstaller{
		module:        mod,
		runtimeScope:  runtimeScope,
		moduleManager: &ModuleManager{runtimeScope: runtimeScope, jsExecutor: &moduleManagerNoopScriptExecutor{}},
		ctx:           newOpContext(),
		builder:       split,
	}
	if err := installer.commitInstall(&moduleresult.BuildResult{}, true); err != nil {
		t.Fatalf("persistLater success: %v", err)
	}
	if split.persistCalls != 1 {
		t.Fatalf("persistCalls = %d", split.persistCalls)
	}

	installer.builder = commitStubBuilder{}
	if err := installer.commitInstall(&moduleresult.BuildResult{}, true); err == nil || !strings.Contains(err.Error(), "does not support Persist") {
		t.Fatalf("expected Persist unsupported error, got %v", err)
	}
}

func TestCommitInstallMetaAndDocumentSchedules(t *testing.T) {
	db := newModuleIndexSyncDB(t)
	if err := db.AutoMigrate(&internaltask.Schedule{}, &meta.IrModule{}); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if err := db.Create(&internaltask.Schedule{
		Id: "sch_legacy", Active: true, Name: "meta.module_index.daily_sync",
		TargetApp: "meta", FullMethod: "meta.IrModuleIndex/Sync",
		SchedulerUserId: "admin", TriggeredByUserId: "admin",
		CronExpr: "0 0 * * *", Timezone: "UTC", CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	modulesPath := t.TempDir()
	runtimeScope := newModuleIndexSyncScope(modulesPath, db)

	metaMod := &meta.IrModule{Name: "meta", Path: filepath.Join(modulesPath, "meta"), Status: meta.ToInstall}
	metaMod.Id = sql.NullString{String: xid.New().String(), Valid: true}
	installer := &moduleInstaller{
		module:        metaMod,
		runtimeScope:  runtimeScope,
		moduleManager: &ModuleManager{runtimeScope: runtimeScope, jsExecutor: &moduleManagerNoopScriptExecutor{}},
		ctx:           newOpContext(),
	}
	if err := installer.commitInstall(nil, false); err != nil {
		t.Fatalf("meta commitInstall: %v", err)
	}
	if err := internaltask.WhereScheduleNameEq(db, "meta.module_index.daily_sync").Take(&internaltask.Schedule{}).Error; err == nil {
		t.Fatal("expected legacy schedule deleted")
	}

	docMod := &meta.IrModule{Name: "document", Path: filepath.Join(modulesPath, "document"), Status: meta.ToInstall}
	docMod.Id = sql.NullString{String: xid.New().String(), Valid: true}
	installer.module = docMod
	if err := installer.commitInstall(nil, false); err != nil {
		t.Fatalf("document commitInstall: %v", err)
	}
	var gc internaltask.Schedule
	if err := internaltask.WhereScheduleNameEq(db, "document.attachment.gc").Take(&gc).Error; err != nil {
		t.Fatalf("expected GC schedule: %v", err)
	}
	if !gc.Active || gc.CronExpr != "*/5 * * * *" {
		t.Fatalf("gc schedule = %+v", gc)
	}
	if internaltask.DecodeTranslatedScheduleName(gc.Name) != "document.attachment.gc" {
		t.Fatalf("expected translated schedule name, got %q", gc.Name)
	}

	// Update existing GC schedule path.
	if err := installer.commitInstall(nil, false); err != nil {
		t.Fatalf("document commitInstall update: %v", err)
	}
}

func TestEnsureDocumentAttachmentGCScheduleDirect(t *testing.T) {
	db := newModuleIndexSyncDB(t)
	if err := db.AutoMigrate(&internaltask.Schedule{}); err != nil {
		t.Fatal(err)
	}
	runtimeScope := newModuleIndexSyncScope(t.TempDir(), db)
	if err := ensureDocumentAttachmentGCSchedule(runtimeScope); err != nil {
		t.Fatal(err)
	}
	if err := ensureDocumentAttachmentGCSchedule(runtimeScope); err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := internaltask.WhereScheduleNameEq(db.Model(&internaltask.Schedule{}), "document.attachment.gc").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("count = %d", count)
	}
	_ = mustJSON(map[string]any{"ok": true})
}

func TestRestoreModuleIfSoftDeletedStandalone(t *testing.T) {
	runtimeScope := newLifecycleCommitTestScope(t)
	mod := &meta.IrModule{Name: "solo", Status: meta.Uninstalled, Path: t.TempDir()}
	mod.Id = sql.NullString{String: xid.New().String(), Valid: true}
	mod.DeletedAt = gorm.DeletedAt{Time: time.Now().UTC(), Valid: true}
	if err := runtimeScope.Session().Unscoped().Create(mod).Error; err != nil {
		t.Fatal(err)
	}
	installer := &moduleInstaller{module: mod, runtimeScope: runtimeScope}
	if err := installer.restoreModuleIfSoftDeleted(); err != nil {
		t.Fatal(err)
	}
	var got meta.IrModule
	if err := runtimeScope.Session().Where("name = ?", "solo").Take(&got).Error; err != nil {
		t.Fatal(err)
	}
	if got.DeletedAt.Valid {
		t.Fatal("expected restored")
	}
}
