// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	metadata "github.com/choysum-dev/choysum/internal/module/metadata"
	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/scope"
	statepkg "github.com/choysum-dev/choysum/pkg/state"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type moduleIndexSyncTestScope struct {
	ctx     context.Context
	logger  *slog.Logger
	cfg     *config.Config
	session *scope.Session
}

func (s *moduleIndexSyncTestScope) Run(fn func(scope.Scope) error) error { return fn(s) }
func (s *moduleIndexSyncTestScope) Session() *scope.Session               { return s.session }
func (s *moduleIndexSyncTestScope) Transactor() scope.Transactor {
	return scopetest.NewPassthroughTransactor(s)
}
func (s *moduleIndexSyncTestScope) WithContext(ctx context.Context) scope.Scope {
	clone := *s
	clone.ctx = ctx
	return &clone
}
func (s *moduleIndexSyncTestScope) Context() context.Context { return s.ctx }
func (s *moduleIndexSyncTestScope) Logger() *slog.Logger     { return s.logger }
func (s *moduleIndexSyncTestScope) FactoryInput() scope.FactoryInput {
	return scopetest.FactoryInputFromConfig(s.cfg)
}

type moduleIndexSyncTestLocker struct {
	acquireErr error
	renewErr   error
	releaseErr error
	acquired   int
	renewed    int
	released   int
	lastTTL    time.Duration
}

func (l *moduleIndexSyncTestLocker) Acquire(context.Context, string, string, time.Duration) error {
	l.acquired++
	if l.lastTTL == 0 {
		l.lastTTL = 60 * time.Second
	}
	return l.acquireErr
}

func (l *moduleIndexSyncTestLocker) Renew(context.Context, string, string, time.Duration) error {
	l.renewed++
	return l.renewErr
}

func (l *moduleIndexSyncTestLocker) Release(context.Context, string, string) error {
	l.released++
	return l.releaseErr
}

func newModuleIndexSyncDB(t *testing.T) *gorm.DB {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "module-index-sync.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&metadata.IrModuleIndex{}, &metadata.IrSetting{}); err != nil {
		t.Fatalf("auto migrate meta tables: %v", err)
	}
	return db
}

func newModuleIndexSyncScope(addonsPath string, db *gorm.DB) *moduleIndexSyncTestScope {
	scopeSession := &scope.Session{}
	if db != nil {
		scopeSession.DB = db
	}
	return &moduleIndexSyncTestScope{
		ctx:     context.Background(),
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		cfg:     &config.Config{AddonsPath: addonsPath},
		session: scopeSession,
	}
}

func writeManifest(t *testing.T, addonsPath, moduleName, content string) {
	t.Helper()
	moduleDir := filepath.Join(addonsPath, moduleName)
	if err := os.MkdirAll(moduleDir, 0o755); err != nil {
		t.Fatalf("mkdir module dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(moduleDir, "manifest.json"), []byte(content), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
}

func TestSyncLocalModuleIndex_NilLockerFactory(t *testing.T) {
	stats, err := SyncLocalModuleIndex(context.Background(), nil, nil)
	if err == nil || !strings.Contains(err.Error(), "locker factory is nil") {
		t.Fatalf("expected nil locker factory error, got stats=%+v err=%v", stats, err)
	}
}

func TestSyncLocalModuleIndex_LeaseBusyMappedToDomainError(t *testing.T) {
	runtimeScope := newModuleIndexSyncScope(t.TempDir(), nil)
	locker := &moduleIndexSyncTestLocker{acquireErr: statepkg.ErrLeaseBusy}

	stats, err := SyncLocalModuleIndex(context.Background(), runtimeScope, func(scope.Scope) statepkg.Locker {
		return locker
	})
	if err == nil {
		t.Fatal("expected LEASE_CONFLICT error")
	}
	if !strings.Contains(err.Error(), "LEASE_CONFLICT") {
		t.Fatalf("expected LEASE_CONFLICT, got %v", err)
	}
	if stats.Total != 0 || stats.Success != 0 || stats.Failed != 0 {
		t.Fatalf("unexpected stats = %+v", stats)
	}
	if locker.acquired != 1 {
		t.Fatalf("locker Acquire calls = %d, want 1", locker.acquired)
	}
	if locker.released != 0 {
		t.Fatalf("locker Release calls = %d, want 0 on failed acquire", locker.released)
	}
}

func TestSyncLocalModuleIndex_AddonsPathRequired(t *testing.T) {
	runtimeScope := newModuleIndexSyncScope("", nil)
	locker := &moduleIndexSyncTestLocker{}

	_, err := SyncLocalModuleIndex(context.Background(), runtimeScope, func(scope.Scope) statepkg.Locker {
		return locker
	})
	if err == nil || !strings.Contains(err.Error(), "addons_path is required") {
		t.Fatalf("expected addons_path required error, got %v", err)
	}
	if locker.released != 1 {
		t.Fatalf("locker Release calls = %d, want 1", locker.released)
	}
}

func TestSyncLocalModuleIndex_SyncsRowsAndReconcilesMissingModules(t *testing.T) {
	addonsPath := t.TempDir()
	writeManifest(t, addonsPath, "partner", `{"name":"partner","version":"0.1.0"}`)
	writeManifest(t, addonsPath, "broken", `{`)
	if err := os.MkdirAll(filepath.Join(addonsPath, "node_modules"), 0o755); err != nil {
		t.Fatalf("mkdir node_modules: %v", err)
	}
	if err := os.WriteFile(filepath.Join(addonsPath, "README.md"), []byte("ignored"), 0o644); err != nil {
		t.Fatalf("write ignored file: %v", err)
	}

	db := newModuleIndexSyncDB(t)
	if err := db.Create(&metadata.IrModuleIndex{
		ModuleName: "stale",
		OriginType: "local",
		OriginRef:  "local",
		Available:  true,
	}).Error; err != nil {
		t.Fatalf("seed stale index row: %v", err)
	}

	runtimeScope := newModuleIndexSyncScope(addonsPath, db)
	locker := &moduleIndexSyncTestLocker{}

	stats, err := SyncLocalModuleIndex(context.Background(), runtimeScope, func(scope.Scope) statepkg.Locker {
		return locker
	})
	if err != nil {
		t.Fatalf("SyncLocalModuleIndex() error = %v", err)
	}
	if stats.Total != 2 || stats.Success != 1 || stats.Failed != 1 {
		t.Fatalf("unexpected stats = %+v", stats)
	}
	if locker.released != 1 {
		t.Fatalf("locker Release calls = %d, want 1", locker.released)
	}

	var partner metadata.IrModuleIndex
	if err := db.Where("module_name = ? AND origin_type = ? AND origin_ref = ?", "partner", "local", "local").Take(&partner).Error; err != nil {
		t.Fatalf("load partner row: %v", err)
	}
	if !partner.Available {
		t.Fatal("expected partner to be available")
	}
	if !partner.Version.Valid || partner.Version.String != "v0.1.0" {
		t.Fatalf("unexpected partner version = %#v", partner.Version)
	}
	if partner.LastErrorMessage.Valid && strings.TrimSpace(partner.LastErrorMessage.String) != "" {
		t.Fatalf("expected empty partner error, got %#v", partner.LastErrorMessage)
	}

	var broken metadata.IrModuleIndex
	if err := db.Where("module_name = ? AND origin_type = ? AND origin_ref = ?", "broken", "local", "local").Take(&broken).Error; err != nil {
		t.Fatalf("load broken row: %v", err)
	}
	if broken.Available {
		t.Fatal("expected broken to be unavailable")
	}
	if !broken.LastErrorMessage.Valid || strings.TrimSpace(broken.LastErrorMessage.String) == "" {
		t.Fatalf("expected broken error message, got %#v", broken.LastErrorMessage)
	}

	var stale metadata.IrModuleIndex
	if err := db.Where("module_name = ? AND origin_type = ? AND origin_ref = ?", "stale", "local", "local").Take(&stale).Error; err != nil {
		t.Fatalf("load stale row: %v", err)
	}
	if stale.Available {
		t.Fatal("expected stale row to be marked unavailable")
	}
	if !stale.LastErrorMessage.Valid || stale.LastErrorMessage.String != "manifest.json not found" {
		t.Fatalf("unexpected stale error message = %#v", stale.LastErrorMessage)
	}
	if stale.LastBatchSyncAt != nil {
		t.Fatalf("expected last_batch_sync_at to stay nil when batch has errors, got %v", stale.LastBatchSyncAt)
	}
}

func TestSyncLocalModuleIndex_AllSuccessUpdatesBatchSyncAt(t *testing.T) {
	addonsPath := t.TempDir()
	writeManifest(t, addonsPath, "partner", `{"name":"partner","version":"0.2.0"}`)

	db := newModuleIndexSyncDB(t)
	runtimeScope := newModuleIndexSyncScope(addonsPath, db)

	stats, err := SyncLocalModuleIndex(context.Background(), runtimeScope, func(scope.Scope) statepkg.Locker {
		return &moduleIndexSyncTestLocker{}
	})
	if err != nil {
		t.Fatalf("SyncLocalModuleIndex() error = %v", err)
	}
	if stats.Total != 1 || stats.Success != 1 || stats.Failed != 0 {
		t.Fatalf("unexpected stats = %+v", stats)
	}

	var rows []metadata.IrModuleIndex
	if err := db.Where("origin_type = ? AND origin_ref = ?", "local", "local").Find(&rows).Error; err != nil {
		t.Fatalf("query local rows: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("local row count = %d, want 1", len(rows))
	}
	if rows[0].LastBatchSyncAt == nil {
		t.Fatal("expected last_batch_sync_at to be set when sync has no errors")
	}
}

func TestModuleIndexLockTTL_UsesSettingAndClampsRange(t *testing.T) {
	db := newModuleIndexSyncDB(t)
	runtimeScope := newModuleIndexSyncScope(t.TempDir(), db)

	if err := db.Create(&metadata.IrSetting{Key: "meta.module_index.sync_lock_ttl_ms", Value: "999999"}).Error; err != nil {
		t.Fatalf("seed setting: %v", err)
	}
	if got := moduleIndexLockTTL(context.Background(), runtimeScope); got != 120*time.Second {
		t.Fatalf("ttl = %v, want %v", got, 120*time.Second)
	}

	if err := db.Model(&metadata.IrSetting{}).Where("key = ?", "meta.module_index.sync_lock_ttl_ms").Update("value", "10").Error; err != nil {
		t.Fatalf("update setting low value: %v", err)
	}
	if got := moduleIndexLockTTL(context.Background(), runtimeScope); got != 1*time.Second {
		t.Fatalf("ttl = %v, want %v", got, 1*time.Second)
	}
}

func TestModuleIndexLockTTL_FallbackCases(t *testing.T) {
	t.Run("nil session", func(t *testing.T) {
		runtimeScope := newModuleIndexSyncScope(t.TempDir(), nil)
		runtimeScope.session = nil
		if got := moduleIndexLockTTL(context.Background(), runtimeScope); got != 60*time.Second {
			t.Fatalf("ttl = %v, want %v", got, 60*time.Second)
		}
	})

	t.Run("setting table missing", func(t *testing.T) {
		dbPath := filepath.Join(t.TempDir(), "ttl-missing-table.db")
		db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
		if err != nil {
			t.Fatalf("open sqlite db: %v", err)
		}
		if err := db.AutoMigrate(&metadata.IrModuleIndex{}); err != nil {
			t.Fatalf("auto migrate module index table: %v", err)
		}
		runtimeScope := newModuleIndexSyncScope(t.TempDir(), db)
		if got := moduleIndexLockTTL(context.Background(), runtimeScope); got != 60*time.Second {
			t.Fatalf("ttl = %v, want %v", got, 60*time.Second)
		}
	})

	t.Run("invalid setting value", func(t *testing.T) {
		db := newModuleIndexSyncDB(t)
		if err := db.Create(&metadata.IrSetting{Key: "meta.module_index.sync_lock_ttl_ms", Value: "invalid"}).Error; err != nil {
			t.Fatalf("seed invalid setting: %v", err)
		}
		runtimeScope := newModuleIndexSyncScope(t.TempDir(), db)
		if got := moduleIndexLockTTL(context.Background(), runtimeScope); got != 60*time.Second {
			t.Fatalf("ttl = %v, want %v", got, 60*time.Second)
		}
	})
}

func TestReadManifestAndHelpers(t *testing.T) {
	manifestPath := filepath.Join(t.TempDir(), "manifest.json")
	if err := os.WriteFile(manifestPath, []byte(`{"version":"0.1.0"}`), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	raw, version, err := readManifest(manifestPath)
	if err != nil {
		t.Fatalf("readManifest() error = %v", err)
	}
	if strings.TrimSpace(string(raw)) == "" || version != "v0.1.0" {
		t.Fatalf("unexpected manifest parse result raw=%q version=%q", string(raw), version)
	}

	if shouldSkipModuleDir("tmp") != true || shouldSkipModuleDir("partner") != false {
		t.Fatal("shouldSkipModuleDir() returned unexpected value")
	}
	if got := fmtSyncRevision(nil); got != "" {
		t.Fatalf("fmtSyncRevision(nil) = %q, want empty", got)
	}
}

func TestSanitizeModuleIndexError_PathAndDefault(t *testing.T) {
	runtimeScope := newModuleIndexSyncScope("/tmp/choysum/addons", nil)
	pathErr := &os.PathError{Op: "open", Path: "/tmp/choysum/addons/meta/manifest.json", Err: os.ErrNotExist}
	if got := SanitizeModuleIndexError(runtimeScope, pathErr); got != "open manifest.json" {
		t.Fatalf("sanitized path error = %q, want %q", got, "open manifest.json")
	}
	if got := SanitizeModuleIndexError(runtimeScope, nil); got != "manifest parsing failed" {
		t.Fatalf("SanitizeModuleIndexError(nil) = %q, want default", got)
	}

	got := SanitizeModuleIndexError(runtimeScope, errors.New(" read /tmp/choysum/addons/meta/manifest.json failed "))
	if strings.Contains(got, "/tmp/choysum/addons") {
		t.Fatalf("expected addons path to be redacted, got %q", got)
	}
}
