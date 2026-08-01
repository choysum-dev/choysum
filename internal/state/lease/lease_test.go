// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lease

import (
	"context"
	"errors"
	leasemodel "github.com/choysum-dev/choysum/internal/state/lease/model"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/choysum-dev/choysum/internal/defaultscope"
	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/scope"
	"gorm.io/gorm"
)

type stubScope struct {
	ctx     context.Context
	cfg     *config.Config
	session *scope.Session
	runErr  error
	runFn   func(func(scope.Scope) error) error
	newFn   func(context.Context) scope.Scope
	uowFn   func(context.Context, scope.TransactionOptions, scope.TxFunc) error
	props   *[]scope.Propagation
}

type stubTransaction struct {
	ctx     context.Context
	session *scope.Session
}

type stubTransactor struct {
	runtimeScope *stubScope
}

func (e *stubScope) Run(fn func(scope.Scope) error) error {
	if e.runFn != nil {
		return e.runFn(fn)
	}
	return e.runErr
}

func (e *stubScope) Transactor() scope.Transactor { return &stubTransactor{runtimeScope: e} }

func (e *stubScope) Session() *scope.Session { return e.session }

func (e *stubScope) WithContext(ctx context.Context) scope.Scope {
	if e.newFn != nil {
		return e.newFn(ctx)
	}
	return &stubScope{ctx: ctx, cfg: e.cfg, session: e.session, runErr: e.runErr, runFn: e.runFn, newFn: e.newFn, uowFn: e.uowFn, props: e.props}
}

func (e *stubScope) Context() context.Context {
	if e.ctx != nil {
		return e.ctx
	}
	return context.Background()
}

func (e *stubScope) Logger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func (e *stubScope) Config() *config.Config { return e.cfg }

func (e *stubScope) FactoryInput() scope.FactoryInput {
	if e == nil {
		return nil
	}
	return scopetest.FactoryInputFromConfig(e.cfg)
}

func (u *stubTransactor) Do(ctx context.Context, opts scope.TransactionOptions, fn scope.TxFunc) error {
	if opts.Propagation == "" {
		opts.Propagation = scope.PropagationRequired
	}
	if u.runtimeScope.props != nil {
		*u.runtimeScope.props = append(*u.runtimeScope.props, opts.Propagation)
	}
	if u.runtimeScope.uowFn != nil {
		return u.runtimeScope.uowFn(ctx, opts, fn)
	}
	txCtx := ctx
	if txCtx == nil {
		txCtx = u.runtimeScope.Context()
	}
	txScope := u.runtimeScope.WithContext(txCtx)
	tx := &stubTransaction{session: txScope.Session()}
	tx.ctx = scope.ContextWithTransaction(txCtx, tx)
	invoke := func() error {
		return fn(txScope, tx)
	}
	if u.runtimeScope.runFn != nil {
		return u.runtimeScope.runFn(func(scope.Scope) error { return invoke() })
	}
	if u.runtimeScope.runErr != nil {
		return u.runtimeScope.runErr
	}
	return invoke()
}

func (u *stubTransactor) Required(ctx context.Context, fn scope.TxFunc) error {
	return u.Do(ctx, scope.TransactionOptions{Propagation: scope.PropagationRequired}, fn)
}

func (u *stubTransactor) RequiresNew(ctx context.Context, fn scope.TxFunc) error {
	return u.Do(ctx, scope.TransactionOptions{Propagation: scope.PropagationRequiresNew}, fn)
}

func (u *stubTransactor) Nested(ctx context.Context, fn scope.TxFunc) error {
	return u.Do(ctx, scope.TransactionOptions{Propagation: scope.PropagationNested}, fn)
}

func (tx *stubTransaction) Context() context.Context {
	if tx == nil {
		return nil
	}
	return tx.ctx
}

func (tx *stubTransaction) Session() *scope.Session {
	if tx == nil {
		return nil
	}
	return tx.session
}

func (tx *stubTransaction) Savepoint(string) error           { return nil }
func (tx *stubTransaction) RollbackToSavepoint(string) error { return nil }
func (tx *stubTransaction) ReleaseSavepoint(string) error    { return nil }

func newSQLiteLocker(t *testing.T) (*Locker, scope.Scope) {
	t.Helper()
	cfg := &config.Config{
		Db: &config.DbConfig{
			Dialect:         "sqlite",
			DSN:             filepath.Join(t.TempDir(), "lease.db"),
			MaxIdleConns:    1,
			MaxOpenConns:    1,
			ConnMaxLifetime: 60,
		},
		Log: config.NewDefaultLogConfig(),
	}

	runtimeScope := defaultscope.NewDefaultScope(
		context.Background(),
		scopetest.FactoryInputFromConfig(cfg),
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)

	if err := runtimeScope.Session().AutoMigrate(&leasemodel.LockLease{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}

	if sqlDB, err := runtimeScope.Session().DB.DB(); err == nil {
		t.Cleanup(func() { _ = sqlDB.Close() })
	}

	return New(runtimeScope), runtimeScope
}

func fetchLease(t *testing.T, runtimeScope scope.Scope, resource string) (*leasemodel.LockLease, bool) {
	t.Helper()

	var record leasemodel.LockLease
	err := runtimeScope.Session().Unscoped().Where("resource = ?", resource).Take(&record).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false
	}
	if err != nil {
		t.Fatalf("load lease %q: %v", resource, err)
	}
	return &record, true
}

func registerLeaseInsertConflict(t *testing.T, db *gorm.DB, resource, ownerId string, expiresAt time.Time) {
	t.Helper()

	callbackName := "lease_test_conflict_" + strings.NewReplacer("/", "_", "-", "_").Replace(resource)
	inserted := false
	if err := db.Callback().Create().Before("gorm:create").Register(callbackName, func(tx *gorm.DB) {
		if inserted {
			return
		}
		leaseRow, ok := tx.Statement.Dest.(*leasemodel.LockLease)
		if !ok || leaseRow.Resource != resource {
			return
		}
		inserted = true
		now := time.Now()
		if err := tx.Session(&gorm.Session{NewDB: true}).Exec(
			"INSERT INTO meta_lock_lease (id, created_at, updated_at, resource, owner_id, expires_at) VALUES (?, ?, ?, ?, ?, ?)",
			callbackName,
			now,
			now,
			resource,
			ownerId,
			expiresAt,
		).Error; err != nil {
			t.Fatalf("inject conflicting lease row for %s: %v", resource, err)
		}
	}); err != nil {
		t.Fatalf("register create callback for %s: %v", resource, err)
	}
	t.Cleanup(func() {
		if err := db.Callback().Create().Remove(callbackName); err != nil {
			t.Fatalf("remove create callback for %s: %v", resource, err)
		}
	})
}

func TestSQLiteSessionDBAndErrorHelpers(t *testing.T) {
	locker := &Locker{}
	if db, ok := locker.sqliteSessionDB(context.Background()); ok || db != nil {
		t.Fatalf("expected nil sqlite session db for nil env, got db=%v ok=%v", db, ok)
	}

	locker.runtimeScope = &stubScope{}
	if db, ok := locker.sqliteSessionDB(context.Background()); ok || db != nil {
		t.Fatalf("expected nil sqlite session db for nil config, got db=%v ok=%v", db, ok)
	}

	locker.runtimeScope = &stubScope{cfg: &config.Config{Db: &config.DbConfig{Dialect: "postgres"}}}
	if db, ok := locker.sqliteSessionDB(context.Background()); ok || db != nil {
		t.Fatalf("expected non-sqlite env to be rejected, got db=%v ok=%v", db, ok)
	}

	locker.runtimeScope = &stubScope{cfg: &config.Config{Db: &config.DbConfig{Dialect: "sqlite"}}}
	if db, ok := locker.sqliteSessionDB(context.Background()); ok || db != nil {
		t.Fatalf("expected sqlite env without session to be rejected, got db=%v ok=%v", db, ok)
	}

	realLocker, realRuntimeScope := newSQLiteLocker(t)
	if db, ok := realLocker.sqliteSessionDB(context.TODO()); !ok || db == nil {
		t.Fatalf("expected sqlite session db, got db=%v ok=%v", db, ok)
	}
	if !isUniqueViolation(errors.New("UNIQUE constraint failed: meta_lock_lease.resource")) {
		t.Fatal("expected sqlite unique violation to be detected")
	}
	if isUniqueViolation(nil) || isUniqueViolation(errors.New("other error")) {
		t.Fatal("unexpected unique violation detection result")
	}
	if !isSqliteLocked(errors.New("database schema is locked: main")) {
		t.Fatal("expected sqlite locked error to be detected")
	}
	if isSqliteLocked(nil) || isSqliteLocked(errors.New("plain error")) {
		t.Fatal("unexpected sqlite lock detection result")
	}

	if record, ok := fetchLease(t, realRuntimeScope, "missing"); ok || record != nil {
		t.Fatalf("expected no row for missing lease, got %#v", record)
	}
	_ = realRuntimeScope
}

func TestAcquireCreateBusyTakeoverAndRestoreDeletedLease(t *testing.T) {
	locker, runtimeScope := newSQLiteLocker(t)

	if err := locker.Acquire(context.Background(), "resource-a", "owner-a", time.Minute); err != nil {
		t.Fatalf("Acquire create: %v", err)
	}
	record, ok := fetchLease(t, runtimeScope, "resource-a")
	if !ok || record.OwnerId != "owner-a" {
		t.Fatalf("unexpected created lease: ok=%v record=%#v", ok, record)
	}
	firstExpiry := record.ExpiresAt

	if err := locker.Acquire(context.Background(), "resource-a", "owner-a", 2*time.Minute); err != nil {
		t.Fatalf("Acquire idempotent refresh: %v", err)
	}
	record, _ = fetchLease(t, runtimeScope, "resource-a")
	if !record.ExpiresAt.After(firstExpiry) {
		t.Fatalf("expected refreshed expiry after %v, got %v", firstExpiry, record.ExpiresAt)
	}

	if err := locker.Acquire(context.Background(), "resource-a", "owner-b", time.Minute); !errors.Is(err, ErrLeaseBusy) {
		t.Fatalf("Acquire busy error = %v, want %v", err, ErrLeaseBusy)
	}

	expired := &leasemodel.LockLease{Resource: "resource-expired", OwnerId: "owner-old", ExpiresAt: time.Now().Add(-time.Minute)}
	if err := runtimeScope.Session().Create(expired).Error; err != nil {
		t.Fatalf("create expired lease: %v", err)
	}
	if err := runtimeScope.Session().Delete(expired).Error; err != nil {
		t.Fatalf("soft delete expired lease: %v", err)
	}
	if err := locker.Acquire(context.Background(), "resource-expired", "owner-new", time.Minute); err != nil {
		t.Fatalf("Acquire takeover expired deleted row: %v", err)
	}
	record, ok = fetchLease(t, runtimeScope, "resource-expired")
	if !ok || record.OwnerId != "owner-new" || record.DeletedAt.Valid {
		t.Fatalf("unexpected restored expired lease: ok=%v record=%#v", ok, record)
	}
}

func TestAcquireWithinRunUsesSQLiteSessionFastPath(t *testing.T) {
	baseLocker, runtimeScope := newSQLiteLocker(t)

	err := runtimeScope.Run(func(executionScope scope.Scope) error {
		locker := New(executionScope)
		if db, ok := locker.sqliteSessionDB(context.Background()); !ok || db == nil {
			return errors.New("expected sqlite session db inside transaction")
		}
		if err := locker.Acquire(context.Background(), "resource-run", "owner-run", time.Minute); err != nil {
			return err
		}

		var count int64
		if err := executionScope.Session().Model(&leasemodel.LockLease{}).Where("resource = ?", "resource-run").Count(&count).Error; err != nil {
			return err
		}
		if count != 1 {
			return errors.New("expected lease row to be visible inside transaction")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Run with Acquire: %v", err)
	}

	record, ok := fetchLease(t, runtimeScope, "resource-run")
	if !ok || record.OwnerId != "owner-run" {
		t.Fatalf("unexpected persisted run lease: ok=%v record=%#v", ok, record)
	}
	_ = baseLocker
}

func TestBaseScopeLockerPrefersContextSessionOnSQLiteFastPath(t *testing.T) {
	_, baseScope := newSQLiteLocker(t)
	_, runtimeScope := newSQLiteLocker(t)

	locker := New(&stubScope{
		cfg:     &config.Config{Db: &config.DbConfig{Dialect: "sqlite"}},
		session: baseScope.Session(),
	})

	runtimeTx := &stubTransaction{session: runtimeScope.Session()}
	runtimeCtx := scope.ContextWithTransaction(context.Background(), runtimeTx)
	if db, ok := locker.sqliteSessionDB(runtimeCtx); !ok || db == nil {
		t.Fatalf("expected sqlite session db from runtime context, got db=%v ok=%v", db, ok)
	}

	if err := locker.Acquire(runtimeCtx, "resource-ctx", "owner-a", time.Minute); err != nil {
		t.Fatalf("Acquire with runtime ctx: %v", err)
	}
	if record, ok := fetchLease(t, runtimeScope, "resource-ctx"); !ok || record.OwnerId != "owner-a" {
		t.Fatalf("unexpected runtime lease after acquire: ok=%v record=%#v", ok, record)
	}
	if record, ok := fetchLease(t, baseScope, "resource-ctx"); ok || record != nil {
		t.Fatalf("expected base scope to remain untouched after acquire, got ok=%v record=%#v", ok, record)
	}

	if err := locker.Renew(runtimeCtx, "resource-ctx", "owner-a", 2*time.Minute); err != nil {
		t.Fatalf("Renew with runtime ctx: %v", err)
	}
	record, ok := fetchLease(t, runtimeScope, "resource-ctx")
	if !ok || record.OwnerId != "owner-a" {
		t.Fatalf("unexpected runtime lease after renew: ok=%v record=%#v", ok, record)
	}

	if err := locker.Release(runtimeCtx, "resource-ctx", "owner-a"); err != nil {
		t.Fatalf("Release with runtime ctx: %v", err)
	}
	if released, ok := fetchLease(t, runtimeScope, "resource-ctx"); ok || released != nil {
		t.Fatalf("expected runtime lease to be released, got ok=%v record=%#v", ok, released)
	}
	if record, ok := fetchLease(t, baseScope, "resource-ctx"); ok || record != nil {
		t.Fatalf("expected base scope to remain untouched after release, got ok=%v record=%#v", ok, record)
	}
}

func TestRenewAndReleasePaths(t *testing.T) {
	locker, runtimeScope := newSQLiteLocker(t)
	if err := locker.Acquire(context.Background(), "resource-r", "owner-a", time.Minute); err != nil {
		t.Fatalf("Acquire before renew: %v", err)
	}
	before, _ := fetchLease(t, runtimeScope, "resource-r")

	if err := locker.Renew(context.Background(), "resource-r", "owner-a", 3*time.Minute); err != nil {
		t.Fatalf("Renew owner: %v", err)
	}
	after, _ := fetchLease(t, runtimeScope, "resource-r")
	if !after.ExpiresAt.After(before.ExpiresAt) {
		t.Fatalf("expected renewed expiry after %v, got %v", before.ExpiresAt, after.ExpiresAt)
	}

	if err := locker.Renew(context.Background(), "resource-r", "owner-b", time.Minute); !errors.Is(err, ErrLeaseNotOwner) {
		t.Fatalf("Renew non-owner error = %v, want %v", err, ErrLeaseNotOwner)
	}
	if err := locker.Release(context.Background(), "resource-r", "owner-b"); !errors.Is(err, ErrLeaseNotOwner) {
		t.Fatalf("Release non-owner error = %v, want %v", err, ErrLeaseNotOwner)
	}

	if err := locker.Release(context.Background(), "resource-r", "owner-a"); err != nil {
		t.Fatalf("Release owner: %v", err)
	}
	if released, ok := fetchLease(t, runtimeScope, "resource-r"); ok || released != nil {
		t.Fatalf("expected released row to be removed, got ok=%v record=%#v", ok, released)
	}

	if err := locker.Release(context.Background(), "resource-r", "owner-a"); !errors.Is(err, ErrLeaseNotHeld) {
		t.Fatalf("Release missing row error = %v, want %v", err, ErrLeaseNotHeld)
	}
	if err := locker.Release(context.Background(), "missing", "owner-a"); !errors.Is(err, ErrLeaseNotHeld) {
		t.Fatalf("Release absent lease error = %v, want %v", err, ErrLeaseNotHeld)
	}
}

func TestValidationAndContextCancellation(t *testing.T) {
	locker := &Locker{runtimeScope: &stubScope{cfg: &config.Config{Db: &config.DbConfig{Dialect: "postgres"}}}}

	if err := locker.Acquire(context.Background(), "resource", "owner", 0); !errors.Is(err, ErrInvalidLeaseTTL) {
		t.Fatalf("Acquire invalid ttl error = %v, want %v", err, ErrInvalidLeaseTTL)
	}
	if err := locker.Renew(context.Background(), "resource", "owner", 0); !errors.Is(err, ErrInvalidLeaseTTL) {
		t.Fatalf("Renew invalid ttl error = %v, want %v", err, ErrInvalidLeaseTTL)
	}
	if err := locker.Acquire(context.Background(), "", "owner", time.Minute); err == nil {
		t.Fatal("expected Acquire to reject empty resource")
	}
	if err := locker.Renew(context.Background(), "resource", "", time.Minute); err == nil {
		t.Fatal("expected Renew to reject empty owner")
	}
	if err := locker.Release(context.Background(), "", "owner"); err == nil {
		t.Fatal("expected Release to reject empty resource")
	}

	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if err := locker.Acquire(canceled, "resource", "owner", time.Minute); !errors.Is(err, context.Canceled) {
		t.Fatalf("Acquire canceled ctx error = %v, want %v", err, context.Canceled)
	}
	if err := locker.Renew(canceled, "resource", "owner", time.Minute); !errors.Is(err, context.Canceled) {
		t.Fatalf("Renew canceled ctx error = %v, want %v", err, context.Canceled)
	}
	if err := locker.Release(canceled, "resource", "owner"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Release canceled ctx error = %v, want %v", err, context.Canceled)
	}
}

func TestAcquireRenewReleaseThroughRequiresNewPath(t *testing.T) {
	_, baseScope := newSQLiteLocker(t)
	propagations := make([]scope.Propagation, 0, 8)
	runScope := &stubScope{
		cfg:     &config.Config{Db: &config.DbConfig{Dialect: "postgres"}},
		session: baseScope.Session(),
		props:   &propagations,
	}
	runScope.runFn = func(fn func(scope.Scope) error) error {
		return fn(runScope)
	}
	locker := New(runScope)

	if err := locker.Acquire(context.Background(), "resource-run", "owner-a", time.Minute); err != nil {
		t.Fatalf("Acquire through Run: %v", err)
	}
	record, ok := fetchLease(t, baseScope, "resource-run")
	if !ok || record.OwnerId != "owner-a" {
		t.Fatalf("unexpected created lease via Run: ok=%v record=%#v", ok, record)
	}
	firstExpiry := record.ExpiresAt

	if err := locker.Renew(context.Background(), "resource-run", "owner-a", 2*time.Minute); err != nil {
		t.Fatalf("Renew through Run: %v", err)
	}
	record, _ = fetchLease(t, baseScope, "resource-run")
	if !record.ExpiresAt.After(firstExpiry) {
		t.Fatalf("expected renewed expiry after %v, got %v", firstExpiry, record.ExpiresAt)
	}

	if err := locker.Acquire(context.Background(), "resource-run", "owner-b", time.Minute); !errors.Is(err, ErrLeaseBusy) {
		t.Fatalf("Acquire busy through Run = %v, want %v", err, ErrLeaseBusy)
	}
	if err := locker.Renew(context.Background(), "resource-run", "owner-b", time.Minute); !errors.Is(err, ErrLeaseNotOwner) {
		t.Fatalf("Renew non-owner through Run = %v, want %v", err, ErrLeaseNotOwner)
	}
	if err := locker.Release(context.Background(), "resource-run", "owner-b"); !errors.Is(err, ErrLeaseNotOwner) {
		t.Fatalf("Release non-owner through Run = %v, want %v", err, ErrLeaseNotOwner)
	}
	if err := locker.Release(context.Background(), "resource-run", "owner-a"); err != nil {
		t.Fatalf("Release owner through Run: %v", err)
	}
	if released, ok := fetchLease(t, baseScope, "resource-run"); ok || released != nil {
		t.Fatalf("expected released row removed via Run, got ok=%v record=%#v", ok, released)
	}
	if err := locker.Release(context.Background(), "resource-run", "owner-a"); !errors.Is(err, ErrLeaseNotHeld) {
		t.Fatalf("Release not held through Run = %v, want %v", err, ErrLeaseNotHeld)
	}
	if len(propagations) == 0 {
		t.Fatal("expected non-sqlite path to record propagation choices")
	}
	for _, propagation := range propagations {
		if propagation != scope.PropagationRequiresNew {
			t.Fatalf("unexpected propagation = %q, want requires_new", propagation)
		}
	}
}

func TestAcquireRetriesTransientLocksOnRequiresNewPath(t *testing.T) {
	_, baseScope := newSQLiteLocker(t)
	attempts := 0
	propagations := make([]scope.Propagation, 0, 4)
	runScope := &stubScope{
		cfg:     &config.Config{Db: &config.DbConfig{Dialect: "postgres"}},
		session: baseScope.Session(),
		props:   &propagations,
	}
	runScope.runFn = func(fn func(scope.Scope) error) error {
		attempts++
		if attempts == 1 {
			return errors.New("database is locked")
		}
		return fn(runScope)
	}
	locker := New(runScope)

	if err := locker.Acquire(context.Background(), "resource-retry", "owner-a", time.Minute); err != nil {
		t.Fatalf("Acquire with retry: %v", err)
	}
	if attempts < 2 {
		t.Fatalf("expected retry attempts, got %d", attempts)
	}
	for _, propagation := range propagations {
		if propagation != scope.PropagationRequiresNew {
			t.Fatalf("unexpected propagation = %q, want requires_new", propagation)
		}
	}
	if record, ok := fetchLease(t, baseScope, "resource-retry"); !ok || record.OwnerId != "owner-a" {
		t.Fatalf("unexpected retried lease: ok=%v record=%#v", ok, record)
	}
}

func TestAcquireLeaseHandlesCreateConflictBranches(t *testing.T) {
	_, runtimeScope := newSQLiteLocker(t)
	db := runtimeScope.Session().WithContext(context.Background()).Unscoped()

	t.Run("same owner refreshes after unique conflict", func(t *testing.T) {
		resource := "resource-conflict-same"
		originalExpiry := time.Now().Add(15 * time.Second)
		targetExpiry := time.Now().Add(2 * time.Minute)
		registerLeaseInsertConflict(t, db, resource, "owner-a", originalExpiry)

		if err := acquireLease(db, resource, "owner-a", time.Now(), targetExpiry); err != nil {
			t.Fatalf("acquireLease same-owner conflict: %v", err)
		}

		record, ok := fetchLease(t, runtimeScope, resource)
		if !ok || record.OwnerId != "owner-a" {
			t.Fatalf("unexpected same-owner conflict lease: ok=%v record=%#v", ok, record)
		}
		if !record.ExpiresAt.After(originalExpiry) {
			t.Fatalf("expected refreshed expiry after %v, got %v", originalExpiry, record.ExpiresAt)
		}
	})

	t.Run("expired rival is taken over after unique conflict", func(t *testing.T) {
		resource := "resource-conflict-expired"
		now := time.Now()
		registerLeaseInsertConflict(t, db, resource, "owner-old", now.Add(-time.Minute))

		if err := acquireLease(db, resource, "owner-new", now, now.Add(3*time.Minute)); err != nil {
			t.Fatalf("acquireLease expired-owner conflict: %v", err)
		}

		record, ok := fetchLease(t, runtimeScope, resource)
		if !ok || record.OwnerId != "owner-new" {
			t.Fatalf("unexpected expired-owner conflict lease: ok=%v record=%#v", ok, record)
		}
		if !record.ExpiresAt.After(now) {
			t.Fatalf("expected new expiry after now=%v, got %v", now, record.ExpiresAt)
		}
	})

	t.Run("active rival stays busy after unique conflict", func(t *testing.T) {
		resource := "resource-conflict-busy"
		now := time.Now()
		registerLeaseInsertConflict(t, db, resource, "owner-b", now.Add(2*time.Minute))

		if err := acquireLease(db, resource, "owner-a", now, now.Add(3*time.Minute)); !errors.Is(err, ErrLeaseBusy) {
			t.Fatalf("acquireLease active-owner conflict = %v, want %v", err, ErrLeaseBusy)
		}

		record, ok := fetchLease(t, runtimeScope, resource)
		if !ok || record.OwnerId != "owner-b" {
			t.Fatalf("unexpected busy conflict lease: ok=%v record=%#v", ok, record)
		}
	})
}

func TestAcquireRenewReleaseTimeoutAfterRepeatedLocks(t *testing.T) {
	_, baseScope := newSQLiteLocker(t)

	newLockedRunScope := func() (*stubScope, *int) {
		attempts := 0
		runScope := &stubScope{
			cfg:     &config.Config{Db: &config.DbConfig{Dialect: "postgres"}},
			session: baseScope.Session(),
		}
		runScope.runFn = func(fn func(scope.Scope) error) error {
			attempts++
			return errors.New("database is locked")
		}
		return runScope, &attempts
	}

	t.Run("acquire times out", func(t *testing.T) {
		runScope, attempts := newLockedRunScope()
		locker := New(runScope)

		err := locker.Acquire(context.Background(), "resource-timeout-acquire", "owner-a", time.Minute)
		if err == nil || err.Error() != "acquire lease: timed out due to database locks" {
			t.Fatalf("Acquire timeout error = %v", err)
		}
		if *attempts != 6 {
			t.Fatalf("expected 6 Acquire attempts, got %d", *attempts)
		}
	})

	t.Run("renew times out", func(t *testing.T) {
		runScope, attempts := newLockedRunScope()
		locker := New(runScope)

		err := locker.Renew(context.Background(), "resource-timeout-renew", "owner-a", time.Minute)
		if err == nil || err.Error() != "renew lease: timed out due to database locks" {
			t.Fatalf("Renew timeout error = %v", err)
		}
		if *attempts != 6 {
			t.Fatalf("expected 6 Renew attempts, got %d", *attempts)
		}
	})

	t.Run("release times out", func(t *testing.T) {
		runScope, attempts := newLockedRunScope()
		locker := New(runScope)

		err := locker.Release(context.Background(), "resource-timeout-release", "owner-a")
		if err == nil || err.Error() != "release lease: timed out due to database locks" {
			t.Fatalf("Release timeout error = %v", err)
		}
		if *attempts != 6 {
			t.Fatalf("expected 6 Release attempts, got %d", *attempts)
		}
	})
}
