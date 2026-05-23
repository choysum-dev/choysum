// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package scripts

import (
	"context"
	metadata "github.com/choysum-dev/choysum/internal/module/metadata"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/scope"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type testScope struct {
	db *gorm.DB
}

func (e *testScope) Run(fn func(runtimeScope scope.Scope) error) error { return fn(e) }
func (e *testScope) Transactor() scope.Transactor                      { return scopetest.NewPassthroughTransactor(e) }
func (e *testScope) Session() *scope.Session                           { return &scope.Session{DB: e.db} }
func (e *testScope) WithContext(ctx context.Context) scope.Scope       { return e }
func (e *testScope) Context() context.Context                          { return context.Background() }
func (e *testScope) Logger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
func (e *testScope) Config() *config.Config { return &config.Config{} }

func (e *testScope) FactoryInput() scope.FactoryInput {
	return scopetest.FactoryInputFromConfig(e.Config())
}

func newTestScope(t *testing.T) scope.Scope {
	t.Helper()
	dsn := "file:migration_history_" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&metadata.IrModuleMigrationHistory{}); err != nil {
		t.Fatalf("migrate test schema: %v", err)
	}
	return &testScope{db: db}
}

func TestHistoryStore_PrepareMarksRunningAndSkipsOnSuccess(t *testing.T) {
	runtimeScope := newTestScope(t)
	store := NewHistoryStore(runtimeScope)
	ctx := context.Background()
	first := Script{
		ModuleName: "meta",
		Version:    "0.1.0",
		Phase:      PhasePre,
		Name:       "init",
		Checksum:   "abc",
	}

	entry, skip, err := store.Prepare(ctx, first)
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if skip {
		t.Fatalf("expected skip=false on first run")
	}
	if entry == nil || entry.Status != "running" {
		t.Fatalf("expected running entry, got %#v", entry)
	}

	if err := store.MarkSuccess(ctx, entry); err != nil {
		t.Fatalf("mark success: %v", err)
	}

	entry2, skip2, err := store.Prepare(ctx, first)
	if err != nil {
		t.Fatalf("prepare after success: %v", err)
	}
	if !skip2 {
		t.Fatalf("expected skip=true on same checksum")
	}
	if entry2 == nil || entry2.Status != "success" {
		t.Fatalf("expected success entry, got %#v", entry2)
	}
}

func TestHistoryStore_PrepareResetsOnChecksumChange(t *testing.T) {
	runtimeScope := newTestScope(t)
	store := NewHistoryStore(runtimeScope)
	ctx := context.Background()
	first := Script{
		ModuleName: "meta",
		Version:    "0.1.0",
		Phase:      PhasePre,
		Name:       "init",
		Checksum:   "abc",
	}

	entry, skip, err := store.Prepare(ctx, first)
	if err != nil || skip {
		t.Fatalf("prepare: %v skip=%v", err, skip)
	}
	if err := store.MarkSuccess(ctx, entry); err != nil {
		t.Fatalf("mark success: %v", err)
	}

	changed := first
	changed.Checksum = "def"
	entry2, skip2, err := store.Prepare(ctx, changed)
	if err != nil {
		t.Fatalf("prepare after checksum change: %v", err)
	}
	if skip2 {
		t.Fatalf("expected skip=false on checksum change")
	}
	if entry2 == nil || entry2.Status != "running" {
		t.Fatalf("expected running entry, got %#v", entry2)
	}
	if entry2.Checksum != "def" {
		t.Fatalf("expected checksum updated to def, got %q", entry2.Checksum)
	}
}

func TestHistoryStore_MarkFailedPersistsError(t *testing.T) {
	runtimeScope := newTestScope(t)
	store := NewHistoryStore(runtimeScope)
	ctx := context.Background()
	script := Script{
		ModuleName: "meta",
		Version:    "0.1.0",
		Phase:      PhasePre,
		Name:       "init",
		Checksum:   "abc",
	}

	entry, skip, err := store.Prepare(ctx, script)
	if err != nil || skip {
		t.Fatalf("prepare: %v skip=%v", err, skip)
	}
	if err := store.MarkFailed(ctx, entry, "boom"); err != nil {
		t.Fatalf("mark failed: %v", err)
	}

	var stored metadata.IrModuleMigrationHistory
	if err := runtimeScope.Session().WithContext(ctx).Where("module_name = ? AND version = ? AND script = ?", "meta", "0.1.0", "init").Take(&stored).Error; err != nil {
		t.Fatalf("reload stored history: %v", err)
	}
	if stored.Status != "failed" || stored.Error != "boom" {
		t.Fatalf("unexpected stored history: %#v", stored)
	}

	if err := (*HistoryStore)(nil).MarkFailed(ctx, entry, "ignored"); err != nil {
		t.Fatalf("nil store MarkFailed() error = %v", err)
	}
	if err := store.MarkFailed(ctx, nil, "ignored"); err != nil {
		t.Fatalf("nil entry MarkFailed() error = %v", err)
	}
}

func TestHistoryStore_PrefersContextScope(t *testing.T) {
	runtimeScope := newTestScope(t)
	store := NewHistoryStore(runtimeScope)
	runtimeDB, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "migration_history_runtime.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open runtime sqlite: %v", err)
	}
	if err := runtimeDB.AutoMigrate(&metadata.IrModuleMigrationHistory{}); err != nil {
		t.Fatalf("migrate runtime schema: %v", err)
	}
	runtimeSession := &scope.Session{DB: runtimeDB}
	ctx := scope.ContextWithScope(context.Background(), &testScope{db: runtimeDB})
	script := Script{
		ModuleName: "meta",
		Version:    "0.1.0",
		Phase:      PhasePre,
		Name:       "init",
		Checksum:   "abc",
	}

	entry, skip, err := store.Prepare(ctx, script)
	if err != nil || skip {
		t.Fatalf("prepare: %v skip=%v", err, skip)
	}
	if err := store.MarkSuccess(ctx, entry); err != nil {
		t.Fatalf("mark success: %v", err)
	}

	var baseCount int64
	if err := runtimeScope.Session().WithContext(context.Background()).Model(&metadata.IrModuleMigrationHistory{}).Count(&baseCount).Error; err != nil {
		t.Fatalf("count base history rows: %v", err)
	}
	if baseCount != 0 {
		t.Fatalf("base history row count = %d, want 0", baseCount)
	}

	var runtimeRow metadata.IrModuleMigrationHistory
	if err := runtimeSession.WithContext(context.Background()).Where("module_name = ? AND version = ? AND script = ?", "meta", "0.1.0", "init").Take(&runtimeRow).Error; err != nil {
		t.Fatalf("load runtime history row: %v", err)
	}
	if runtimeRow.Status != "success" || runtimeRow.Checksum != "abc" {
		t.Fatalf("unexpected runtime history row: %#v", runtimeRow)
	}
}
