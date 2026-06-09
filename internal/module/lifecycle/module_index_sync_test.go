// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"context"
	"encoding/json"
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
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	statepkg "github.com/choysum-dev/choysum/pkg/state"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
func (s *moduleIndexSyncTestScope) Session() *scope.Session              { return s.session }
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

func newModuleIndexSyncScope(modulesPath string, db *gorm.DB) *moduleIndexSyncTestScope {
	scopeSession := &scope.Session{}
	if db != nil {
		scopeSession.DB = db
	}
	return &moduleIndexSyncTestScope{
		ctx:     context.Background(),
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		cfg:     &config.Config{ModulesPath: modulesPath},
		session: scopeSession,
	}
}

func writePackageJSON(t *testing.T, modulesPath, moduleName, content string) {
	t.Helper()
	moduleDir := filepath.Join(modulesPath, moduleName)
	if err := os.MkdirAll(moduleDir, 0o755); err != nil {
		t.Fatalf("mkdir module dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(moduleDir, "package.json"), []byte(content), 0o644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}
}

func TestModuleManagerResolveGeneratedAPIRootUsesDefaultChoysumPath(t *testing.T) {
	defaultChoysumPath := t.TempDir()
	manager := &ModuleManager{runtimeOptions: runtimeOptions{
		modulesPath:        filepath.Join(t.TempDir(), "modules"),
		defaultChoysumPath: defaultChoysumPath,
		compileBundleMode:  string(config.BundleModeApplication),
	}}

	root, err := manager.resolveGeneratedAPIRoot()
	if err != nil {
		t.Fatalf("resolveGeneratedAPIRoot() error = %v", err)
	}
	if root != filepath.Join(defaultChoysumPath, "generated") {
		t.Fatalf("resolveGeneratedAPIRoot() = %q, want %q", root, filepath.Join(defaultChoysumPath, "generated"))
	}
}

func TestModuleManagerRefreshModuleIndexForLocalModules(t *testing.T) {
	modulesPath := t.TempDir()
	writePackageJSON(t, modulesPath, "partner", `{"name":"@acme/choysum-partner","version":"0.3.0","choysum":{"moduleName":"partner","application":"partner"}}`)

	db := newModuleIndexSyncDB(t)
	runtimeScope := newModuleIndexSyncScope(modulesPath, db)
	manager := NewModuleManager(runtimeScope, nil)
	manager.bootstrapOnce.Do(func() {})

	err := manager.refreshModuleIndexForLocalModules(context.Background(), []string{"partner", "missing", "partner", " "})
	if err != nil {
		t.Fatalf("refreshModuleIndexForLocalModules() error = %v", err)
	}

	var partner metadata.IrModuleIndex
	if err := db.Where("module_name = ? AND origin_type = ? AND origin_ref = ?", "partner", "local", "local").Take(&partner).Error; err != nil {
		t.Fatalf("load partner module index row: %v", err)
	}
	if !partner.Available {
		t.Fatal("expected partner index row to be available")
	}
	if !partner.LocalPath.Valid || partner.LocalPath.String != filepath.Join(modulesPath, "partner") {
		t.Fatalf("unexpected partner local path: %#v", partner.LocalPath)
	}
	if partner.LastErrorMessage.Valid && strings.TrimSpace(partner.LastErrorMessage.String) != "" {
		t.Fatalf("expected partner last error to be empty, got %#v", partner.LastErrorMessage)
	}

	var missing metadata.IrModuleIndex
	if err := db.Where("module_name = ? AND origin_type = ? AND origin_ref = ?", "missing", "local", "local").Take(&missing).Error; err != nil {
		t.Fatalf("load missing module index row: %v", err)
	}
	if missing.Available {
		t.Fatal("expected missing index row to be unavailable")
	}
	if !missing.LocalPath.Valid || missing.LocalPath.String != filepath.Join(modulesPath, "missing") {
		t.Fatalf("unexpected missing local path: %#v", missing.LocalPath)
	}
	if !missing.LastErrorMessage.Valid || missing.LastErrorMessage.String != "package.json not found" {
		t.Fatalf("unexpected missing last error message: %#v", missing.LastErrorMessage)
	}
}

func TestModuleManagerBuildBackendAppToDirWritesModuleBasedEntryImports(t *testing.T) {
	modulesPath := t.TempDir()
	db := newModuleIndexSyncDB(t)
	if err := db.AutoMigrate(meta.Entities()...); err != nil {
		t.Fatalf("auto migrate meta entities: %v", err)
	}

	absEntry := filepath.Join(t.TempDir(), "service", "entry.ts")
	if err := db.Create(&meta.IrModule{Name: "crm_partner", ApplicationStr: "crm", Status: meta.Installed, ServiceEntryPoint: "service/main.ts"}).Error; err != nil {
		t.Fatalf("seed relative module: %v", err)
	}
	if err := db.Create(&meta.IrModule{Name: "crm_company", ApplicationStr: "crm", Status: meta.Installed, ServiceEntryPoint: absEntry}).Error; err != nil {
		t.Fatalf("seed absolute module: %v", err)
	}

	runtimeScope := newModuleIndexSyncScope(modulesPath, db)
	manager := NewModuleManager(runtimeScope, nil)
	manager.bootstrapOnce.Do(func() {})
	distAppDir := t.TempDir()

	_ = manager.buildBackendAppToDir(context.Background(), "crm", distAppDir)

	entryFilePath := filepath.Join(distAppDir, "__choysum_app_entry.ts")
	entryRaw, err := os.ReadFile(entryFilePath)
	if err != nil {
		t.Fatalf("read app entry file %q: %v", entryFilePath, err)
	}
	entryText := string(entryRaw)

	if !strings.Contains(entryText, filepath.Join(modulesPath, "crm_partner", "service/main.ts")) {
		t.Fatalf("expected app entry to include module-relative service import, got %q", entryText)
	}
	if !strings.Contains(entryText, absEntry) {
		t.Fatalf("expected app entry to preserve absolute service import, got %q", entryText)
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

func TestSyncLocalModuleIndex_ModulesPathRequired(t *testing.T) {
	runtimeScope := newModuleIndexSyncScope("", nil)
	locker := &moduleIndexSyncTestLocker{}

	_, err := SyncLocalModuleIndex(context.Background(), runtimeScope, func(scope.Scope) statepkg.Locker {
		return locker
	})
	if err == nil || !strings.Contains(err.Error(), "modules_path is required") {
		t.Fatalf("expected modules_path required error, got %v", err)
	}
	if locker.released != 1 {
		t.Fatalf("locker Release calls = %d, want 1", locker.released)
	}
}

func TestSyncLocalModuleIndex_SyncsRowsAndReconcilesMissingModules(t *testing.T) {
	modulesPath := t.TempDir()
	writePackageJSON(t, modulesPath, "partner", `{"name":"@acme/choysum-partner","version":"0.1.0","choysum":{"moduleName":"partner","application":"partner"}}`)
	writePackageJSON(t, modulesPath, "broken", `{`)
	if err := os.MkdirAll(filepath.Join(modulesPath, "node_modules"), 0o755); err != nil {
		t.Fatalf("mkdir node_modules: %v", err)
	}
	if err := os.WriteFile(filepath.Join(modulesPath, "README.md"), []byte("ignored"), 0o644); err != nil {
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

	runtimeScope := newModuleIndexSyncScope(modulesPath, db)
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
	if !stale.LastErrorMessage.Valid || stale.LastErrorMessage.String != "package.json not found" {
		t.Fatalf("unexpected stale error message = %#v", stale.LastErrorMessage)
	}
	if stale.LastBatchSyncAt != nil {
		t.Fatalf("expected last_batch_sync_at to stay nil when batch has errors, got %v", stale.LastBatchSyncAt)
	}
}

func TestSyncLocalModuleIndex_AllSuccessUpdatesBatchSyncAt(t *testing.T) {
	modulesPath := t.TempDir()
	writePackageJSON(t, modulesPath, "partner", `{"name":"@acme/choysum-partner","version":"0.2.0","choysum":{"moduleName":"partner","application":"partner"}}`)

	db := newModuleIndexSyncDB(t)
	runtimeScope := newModuleIndexSyncScope(modulesPath, db)

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

func TestModuleIndexSyncHelperFunctions(t *testing.T) {
	t.Parallel()

	t.Run("shouldSkipModuleDir", func(t *testing.T) {
		cases := []struct {
			name string
			want bool
		}{
			{name: "", want: true},
			{name: ".cache", want: true},
			{name: "tmp", want: true},
			{name: "node_modules", want: true},
			{name: "dist", want: true},
			{name: "auth", want: false},
		}
		for _, tc := range cases {
			if got := shouldSkipModuleDir(tc.name); got != tc.want {
				t.Fatalf("shouldSkipModuleDir(%q) = %v, want %v", tc.name, got, tc.want)
			}
		}
	})

	t.Run("fmtSyncRevision", func(t *testing.T) {
		if got := fmtSyncRevision(nil); got != "" {
			t.Fatalf("fmtSyncRevision(nil) = %q, want empty", got)
		}

		tmpFile := filepath.Join(t.TempDir(), "package.json")
		if err := os.WriteFile(tmpFile, []byte(`{"name":"pkg"}`), 0o644); err != nil {
			t.Fatalf("write temp file: %v", err)
		}
		info, err := os.Stat(tmpFile)
		if err != nil {
			t.Fatalf("stat temp file: %v", err)
		}
		if got := fmtSyncRevision(info); strings.TrimSpace(got) == "" {
			t.Fatal("fmtSyncRevision(non-nil) should return non-empty revision")
		}
	})
}

func TestReadPackageJSONValidation(t *testing.T) {
	t.Parallel()

	t.Run("missing file", func(t *testing.T) {
		if _, _, err := readPackageJSON(filepath.Join(t.TempDir(), "missing.json")); err == nil {
			t.Fatal("expected readPackageJSON missing file error")
		}
	})

	t.Run("missing version", func(t *testing.T) {
		pkgPath := filepath.Join(t.TempDir(), "package.json")
		raw := `{"name":"@acme/choysum-auth","choysum":{"moduleName":"auth","application":"auth"}}`
		if err := os.WriteFile(pkgPath, []byte(raw), 0o644); err != nil {
			t.Fatalf("write package.json: %v", err)
		}

		data, version, err := readPackageJSON(pkgPath)
		if err == nil || (!strings.Contains(err.Error(), "missing version") && !strings.Contains(err.Error(), "empty package version")) {
			t.Fatalf("expected missing version error, got version=%q err=%v", version, err)
		}
		if string(data) != raw {
			t.Fatalf("readPackageJSON should return original payload, got %q", string(data))
		}
	})

	t.Run("valid payload", func(t *testing.T) {
		pkgPath := filepath.Join(t.TempDir(), "package.json")
		raw := `{"name":"@acme/choysum-auth","version":"1.0.0","choysum":{"moduleName":"auth","application":"auth"}}`
		if err := os.WriteFile(pkgPath, []byte(raw), 0o644); err != nil {
			t.Fatalf("write package.json: %v", err)
		}

		_, version, err := readPackageJSON(pkgPath)
		if err != nil {
			t.Fatalf("readPackageJSON(valid) error = %v", err)
		}
		if version != "v1.0.0" {
			t.Fatalf("readPackageJSON(valid) version = %q, want %q", version, "v1.0.0")
		}
	})
}

func TestSanitizeModuleIndexErrorBranches(t *testing.T) {
	t.Parallel()

	modulesPath := filepath.Join(t.TempDir(), "modules")
	runtimeScope := newModuleIndexSyncScope(modulesPath, nil)

	if got := SanitizeModuleIndexError(runtimeScope, nil); got != "package.json parsing failed" {
		t.Fatalf("SanitizeModuleIndexError(nil) = %q, want %q", got, "package.json parsing failed")
	}

	grpcErr := status.Error(codes.InvalidArgument, filepath.Join(modulesPath, "auth", "package.json")+": invalid field")
	if got := SanitizeModuleIndexError(runtimeScope, grpcErr); !strings.Contains(got, "<modulesPath>") {
		t.Fatalf("expected redacted modules path in grpc error, got %q", got)
	}

	pathErr := &os.PathError{Op: "open", Path: filepath.Join(modulesPath, "auth", "package.json"), Err: errors.New("permission denied")}
	if got := SanitizeModuleIndexError(runtimeScope, pathErr); got != "open package.json" {
		t.Fatalf("SanitizeModuleIndexError(path error) = %q, want %q", got, "open package.json")
	}

	if got := redactModuleIndexError(runtimeScope, ""); got != "package.json parsing failed" {
		t.Fatalf("redactModuleIndexError(empty) = %q, want default", got)
	}
}

func TestSyncLocalModuleIndex_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("read dir failure", func(t *testing.T) {
		runtimeScope := newModuleIndexSyncScope(filepath.Join(t.TempDir(), "missing"), nil)
		_, err := SyncLocalModuleIndex(context.Background(), runtimeScope, func(scope.Scope) statepkg.Locker {
			return &moduleIndexSyncTestLocker{}
		})
		if err == nil {
			t.Fatal("expected read dir error")
		}
	})

	t.Run("nil context falls back to background", func(t *testing.T) {
		modulesPath := t.TempDir()
		writePackageJSON(t, modulesPath, "partner", `{"name":"@acme/choysum-partner","version":"0.1.0","choysum":{"moduleName":"partner","application":"partner"}}`)
		db := newModuleIndexSyncDB(t)
		runtimeScope := newModuleIndexSyncScope(modulesPath, db)

		stats, err := SyncLocalModuleIndex(nil, runtimeScope, func(scope.Scope) statepkg.Locker {
			return &moduleIndexSyncTestLocker{}
		})
		if err != nil {
			t.Fatalf("SyncLocalModuleIndex(nil ctx) error = %v", err)
		}
		if stats.Total != 1 || stats.Success != 1 || stats.Failed != 0 {
			t.Fatalf("unexpected stats for nil ctx sync: %+v", stats)
		}
	})

	t.Run("context canceled during scan", func(t *testing.T) {
		modulesPath := t.TempDir()
		if err := os.MkdirAll(filepath.Join(modulesPath, "partner"), 0o755); err != nil {
			t.Fatalf("mkdir module dir: %v", err)
		}
		runtimeScope := newModuleIndexSyncScope(modulesPath, newModuleIndexSyncDB(t))
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := SyncLocalModuleIndex(ctx, runtimeScope, func(scope.Scope) statepkg.Locker {
			return &moduleIndexSyncTestLocker{}
		})
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context canceled error, got %v", err)
		}
	})
}

func TestIsTableMissingInSessionBranches(t *testing.T) {
	t.Parallel()

	if isTableMissingInSession(nil, "meta_ir_setting") {
		t.Fatal("nil session should not be treated as missing table")
	}

	s := &scope.Session{}
	if isTableMissingInSession(s, "meta_ir_setting") {
		t.Fatal("session without DB should not be treated as missing table")
	}

	db := newModuleIndexSyncDB(t)
	s = &scope.Session{DB: db}
	if isTableMissingInSession(s, "") {
		t.Fatal("empty table name should return false")
	}
	if isTableMissingInSession(s, "meta_ir_setting") {
		t.Fatal("existing table should not be treated as missing")
	}
	if !isTableMissingInSession(s, "meta_ir_not_exists") {
		t.Fatal("unknown table should be treated as missing")
	}
}

func TestReadPackageJSONAndHelpers(t *testing.T) {
	packageJSONPath := filepath.Join(t.TempDir(), "package.json")
	if err := os.WriteFile(packageJSONPath, []byte(`{"name":"@acme/choysum-meta","version":"0.1.0","choysum":{"moduleName":"meta","application":"meta"}}`), 0o644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}

	raw, version, err := readPackageJSON(packageJSONPath)
	if err != nil {
		t.Fatalf("readPackageJSON() error = %v", err)
	}
	if strings.TrimSpace(string(raw)) == "" || version != "v0.1.0" {
		t.Fatalf("unexpected package.json parse result raw=%q version=%q", string(raw), version)
	}

	if shouldSkipModuleDir("tmp") != true || shouldSkipModuleDir("partner") != false {
		t.Fatal("shouldSkipModuleDir() returned unexpected value")
	}
	if got := fmtSyncRevision(nil); got != "" {
		t.Fatalf("fmtSyncRevision(nil) = %q, want empty", got)
	}
}

func TestSanitizeModuleIndexError_PathAndDefault(t *testing.T) {
	runtimeScope := newModuleIndexSyncScope("/tmp/choysum/modules", nil)
	pathErr := &os.PathError{Op: "open", Path: "/tmp/choysum/modules/meta/package.json", Err: os.ErrNotExist}
	if got := SanitizeModuleIndexError(runtimeScope, pathErr); got != "open package.json" {
		t.Fatalf("sanitized path error = %q, want %q", got, "open package.json")
	}
	if got := SanitizeModuleIndexError(runtimeScope, nil); got != "package.json parsing failed" {
		t.Fatalf("SanitizeModuleIndexError(nil) = %q, want default", got)
	}

	got := SanitizeModuleIndexError(runtimeScope, errors.New(" read /tmp/choysum/modules/meta/package.json failed "))
	if strings.Contains(got, "/tmp/choysum/modules") {
		t.Fatalf("expected modules path to be redacted, got %q", got)
	}
}

type moduleManagerInstallOriginCoordinator struct {
	module              *meta.IrModule
	resolveInstallCalls int
}

func (c *moduleManagerInstallOriginCoordinator) Peek(context.Context, string) (*meta.IrModule, error) {
	if c == nil || c.module == nil {
		return nil, nil
	}
	cloned := *c.module
	return &cloned, nil
}

func (c *moduleManagerInstallOriginCoordinator) ResolveInstallModule(context.Context, string) (*meta.IrModule, error) {
	c.resolveInstallCalls++
	if c.module == nil {
		return nil, nil
	}
	cloned := *c.module
	return &cloned, nil
}

func (c *moduleManagerInstallOriginCoordinator) Fetch(context.Context, string) (*meta.IrModule, error) {
	if c.module == nil {
		return nil, nil
	}
	cloned := *c.module
	return &cloned, nil
}

func (c *moduleManagerInstallOriginCoordinator) Purge(context.Context, string) error {
	return nil
}

type moduleManagerNoopScriptExecutor struct {
	scripts []*jsengine.JsScript
}

func (e *moduleManagerNoopScriptExecutor) Execute(context.Context, *jsengine.JsRequest) (*jsengine.JsResponse, error) {
	return &jsengine.JsResponse{}, nil
}

func (e *moduleManagerNoopScriptExecutor) GetJsScripts() []*jsengine.JsScript {
	return e.scripts
}

func (e *moduleManagerNoopScriptExecutor) SetJsScripts(scripts []*jsengine.JsScript) {
	e.scripts = scripts
}

func (e *moduleManagerNoopScriptExecutor) Reload(scripts ...*jsengine.JsScript) error {
	e.scripts = scripts
	return nil
}

func TestModuleManagerInstallRunsAppStageCallbacks(t *testing.T) {
	modulesPath := t.TempDir()
	distPath := filepath.Join(t.TempDir(), "dist")
	tmpPath := filepath.Join(t.TempDir(), "tmp")
	defaultChoysumPath := filepath.Join(t.TempDir(), ".choysum")

	db := newModuleIndexSyncDB(t)
	if err := db.AutoMigrate(meta.Entities()...); err != nil {
		t.Fatalf("auto migrate meta entities: %v", err)
	}

	runtimeScope := newModuleIndexSyncScope(modulesPath, db)
	runtimeScope.cfg.DistPath = distPath
	runtimeScope.cfg.TmpPath = tmpPath
	runtimeScope.cfg.DefaultChoysumPath = defaultChoysumPath
	runtimeScope.cfg.Compile = &config.CompileConfig{BundleMode: string(config.BundleModeApplication)}

	locker := &moduleIndexSyncTestLocker{}
	coordinator := &moduleManagerInstallOriginCoordinator{module: &meta.IrModule{
		Name:           "auth",
		ApplicationStr: "crm",
		Version:        "v1.2.0",
	}}
	manager := NewModuleManager(
		runtimeScope,
		&moduleManagerNoopScriptExecutor{},
		WithLockerFactory(func(scope.Scope) statepkg.Locker { return locker }),
		WithOriginCoordinatorFactory(func(scope.Scope) OriginCoordinator { return coordinator }),
	)
	manager.bootstrapOnce.Do(func() {})

	dependsRaw, err := json.Marshal([]string{"missing_dep"})
	if err != nil {
		t.Fatalf("marshal depends: %v", err)
	}
	if err := db.Create(&meta.IrModule{
		Name:       "auth",
		Status:     meta.Installed,
		Version:    "v1.0.0",
		DependsStr: dependsRaw,
		Path:       filepath.Join(modulesPath, "auth"),
	}).Error; err != nil {
		t.Fatalf("seed installed module: %v", err)
	}

	if err := manager.Install(context.Background(), "auth"); err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	if locker.acquired != 1 || locker.released != 1 {
		t.Fatalf("locker calls acquired=%d released=%d, want 1/1", locker.acquired, locker.released)
	}
	if coordinator.resolveInstallCalls == 0 {
		t.Fatal("expected ResolveInstallModule to be called")
	}

	for _, dir := range []string{
		filepath.Join(distPath, "apps", "crm"),
		filepath.Join(defaultChoysumPath, "generated", "proto", "crm"),
		filepath.Join(defaultChoysumPath, "generated", "web", "crm"),
		filepath.Join(defaultChoysumPath, "generated", "service", "crm"),
	} {
		if info, statErr := os.Stat(dir); statErr != nil || !info.IsDir() {
			t.Fatalf("expected staged app output dir %q, stat err=%v info=%#v", dir, statErr, info)
		}
	}
}

func TestModuleManagerUninstallRunsAppStageCallbacks(t *testing.T) {
	modulesPath := t.TempDir()
	distPath := filepath.Join(t.TempDir(), "dist")
	tmpPath := filepath.Join(t.TempDir(), "tmp")
	defaultChoysumPath := filepath.Join(t.TempDir(), ".choysum")

	db := newModuleIndexSyncDB(t)
	if err := db.AutoMigrate(meta.Entities()...); err != nil {
		t.Fatalf("auto migrate meta entities: %v", err)
	}

	runtimeScope := newModuleIndexSyncScope(modulesPath, db)
	runtimeScope.cfg.DistPath = distPath
	runtimeScope.cfg.TmpPath = tmpPath
	runtimeScope.cfg.DefaultChoysumPath = defaultChoysumPath
	runtimeScope.cfg.Compile = &config.CompileConfig{BundleMode: string(config.BundleModeApplication)}

	locker := &moduleIndexSyncTestLocker{}
	manager := NewModuleManager(
		runtimeScope,
		&moduleManagerNoopScriptExecutor{},
		WithLockerFactory(func(scope.Scope) statepkg.Locker { return locker }),
	)
	manager.bootstrapOnce.Do(func() {})

	if err := db.Create(&meta.IrModule{
		Name:           "auth",
		Status:         meta.Installed,
		Version:        "v1.0.0",
		ApplicationStr: "crm",
		Path:           filepath.Join(modulesPath, "auth"),
	}).Error; err != nil {
		t.Fatalf("seed installed module: %v", err)
	}

	if err := manager.Uninstall(context.Background(), "auth"); err != nil {
		t.Fatalf("Uninstall() error = %v", err)
	}
	if locker.acquired != 1 || locker.released != 1 {
		t.Fatalf("locker calls acquired=%d released=%d, want 1/1", locker.acquired, locker.released)
	}

	var mod meta.IrModule
	if err := db.Unscoped().Where("name = ?", "auth").Take(&mod).Error; err != nil {
		t.Fatalf("load uninstalled module row: %v", err)
	}
	if mod.Status != meta.Uninstalled {
		t.Fatalf("module status after uninstall = %q, want %q", mod.Status, meta.Uninstalled)
	}

	for _, dir := range []string{
		filepath.Join(distPath, "apps", "crm"),
		filepath.Join(defaultChoysumPath, "generated", "proto", "crm"),
		filepath.Join(defaultChoysumPath, "generated", "web", "crm"),
		filepath.Join(defaultChoysumPath, "generated", "service", "crm"),
	} {
		if info, statErr := os.Stat(dir); statErr != nil || !info.IsDir() {
			t.Fatalf("expected staged app output dir %q, stat err=%v info=%#v", dir, statErr, info)
		}
	}
}
