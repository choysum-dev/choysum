// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lease

import (
	"context"
	"errors"
	"strings"
	"time"

	leasemodel "github.com/choysum-dev/choysum/internal/state/lease/model"

	"github.com/choysum-dev/choysum/pkg/scope"
	statepkg "github.com/choysum-dev/choysum/pkg/state"
	"golang.org/x/exp/errors/fmt"
	"gorm.io/gorm"
)

var (
	ErrLeaseBusy       = statepkg.ErrLeaseBusy
	ErrLeaseNotOwner   = statepkg.ErrLeaseNotOwner
	ErrLeaseNotHeld    = statepkg.ErrLeaseNotHeld
	ErrInvalidLeaseTTL = statepkg.ErrInvalidLeaseTTL
)

type Locker struct {
	runtimeScope scope.Scope
}

var _ statepkg.Locker = (*Locker)(nil)

func New(runtimeScope scope.Scope) *Locker {
	return &Locker{runtimeScope: runtimeScope}
}

func (l *Locker) runRequiresNew(ctx context.Context, fn func(scope.Scope) error) error {
	txRoot := l.runtimeScope.WithContext(ctx)
	return txRoot.Transactor().RequiresNew(ctx, func(txScope scope.Scope, _ scope.Transaction) error {
		return fn(txScope)
	})
}

func (l *Locker) sqliteSessionDB(ctx context.Context) (*gorm.DB, bool) {
	if l.runtimeScope == nil {
		return nil, false
	}
	if runtimeOptionsFromScope(l.runtimeScope).dbDialect != "sqlite" {
		return nil, false
	}
	sess, ok := scope.SessionForScope(ctx, l.runtimeScope)
	if !ok {
		return nil, false
	}
	if sess == nil || sess.DB == nil {
		return nil, false
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return sess.WithContext(ctx).Unscoped(), true
}

// Acquire tries to acquire the lease for resource.
//
// Semantics:
// - If row exists and owned by someone else and not expired: ErrLeaseBusy.
// - If row exists and owned by ownerId: refresh (idempotent).
// - If row exists and is expired: take over.
// - If no row exists: create it.
func (l *Locker) Acquire(ctx context.Context, resource, ownerId string, ttl time.Duration) error {
	if ttl <= 0 {
		return ErrInvalidLeaseTTL
	}
	if resource == "" || ownerId == "" {
		return fmt.Errorf("resource/ownerId must be non-empty")
	}

	now := time.Now()
	expiresAt := now.Add(ttl)

	// If we're already running inside an active SQLite transaction, reuse the
	// current session to avoid lock contention with a nested transaction.
	if db, ok := l.sqliteSessionDB(ctx); ok {
		return acquireLease(db, resource, ownerId, now, expiresAt)
	}

	// SQLite may intermittently return `database is locked` / `database schema is locked`
	// during intense DDL/write bursts. Retry briefly to avoid flaky module installs.
	//
	// Note: each statement may already block up to the SQLite busy timeout; keep the
	// retry count small to avoid multi-minute hangs.
	for attempt := 0; attempt < 6; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		err := l.runRequiresNew(ctx, func(runtimeScope scope.Scope) error {
			db := runtimeScope.Session().WithContext(ctx).Unscoped()
			return acquireLease(db, resource, ownerId, now, expiresAt)
		})
		if err == nil {
			return nil
		}
		if errors.Is(err, ErrLeaseBusy) {
			return err
		}
		if !isSqliteLocked(err) {
			return err
		}
		// Backoff
		time.Sleep(time.Duration(50*(attempt+1)) * time.Millisecond)
	}
	return fmt.Errorf("acquire lease: timed out due to database locks")
}

func acquireLease(db *gorm.DB, resource, ownerId string, now, expiresAt time.Time) error {
	newDB := func() *gorm.DB {
		// Keep the current transaction/connection when present. Using NewDB may
		// detach to a different connection and cause SQLite self-lock waits.
		return db.Session(&gorm.Session{}).Unscoped()
	}

	// Fast path: update existing expired/owned lease (also restores from soft delete).
	res := newDB().Model(&leasemodel.IrLockLease{}).
		Where("resource = ? AND (expires_at < ? OR owner_id = ?)", resource, now, ownerId).
		Updates(map[string]any{"owner_id": ownerId, "expires_at": expiresAt, "deleted_at": nil})
	if res.Error != nil {
		return fmt.Errorf("acquire lease update: %w", res.Error)
	}
	if res.RowsAffected == 1 {
		return nil
	}

	// Try insert.
	row := &leasemodel.IrLockLease{Resource: resource, OwnerId: ownerId, ExpiresAt: expiresAt}
	if err := newDB().Create(row).Error; err != nil {
		if !isUniqueViolation(err) {
			return fmt.Errorf("acquire lease insert: %w", err)
		}

		// Someone else inserted concurrently; re-check.
		var existing leasemodel.IrLockLease
		if err := newDB().Where("resource = ?", resource).Take(&existing).Error; err != nil {
			return fmt.Errorf("acquire lease reload: %w", err)
		}

		// Idempotent: treat as acquired, refresh.
		if existing.OwnerId == ownerId {
			if err := newDB().Model(&leasemodel.IrLockLease{}).
				Where("resource = ? AND owner_id = ?", resource, ownerId).
				Updates(map[string]any{"expires_at": expiresAt, "deleted_at": nil}).Error; err != nil {
				return fmt.Errorf("acquire lease refresh: %w", err)
			}
			return nil
		}

		// Try take over if expired.
		if existing.ExpiresAt.Before(now) {
			res2 := newDB().Model(&leasemodel.IrLockLease{}).
				Where("resource = ? AND expires_at < ?", resource, now).
				Updates(map[string]any{"owner_id": ownerId, "expires_at": expiresAt, "deleted_at": nil})
			if res2.Error != nil {
				return fmt.Errorf("acquire lease takeover: %w", res2.Error)
			}
			if res2.RowsAffected == 1 {
				return nil
			}
		}

		return ErrLeaseBusy
	}

	return nil
}

func (l *Locker) Renew(ctx context.Context, resource, ownerId string, ttl time.Duration) error {
	if ttl <= 0 {
		return ErrInvalidLeaseTTL
	}
	if resource == "" || ownerId == "" {
		return fmt.Errorf("resource/ownerId must be non-empty")
	}

	expiresAt := time.Now().Add(ttl)

	if db, ok := l.sqliteSessionDB(ctx); ok {
		for attempt := 0; attempt < 6; attempt++ {
			if err := ctx.Err(); err != nil {
				return err
			}
			err := renewLeaseOnDB(db, resource, ownerId, expiresAt)
			if err == nil {
				return nil
			}
			if errors.Is(err, ErrLeaseNotOwner) {
				return err
			}
			if !isSqliteLocked(err) {
				return err
			}
			time.Sleep(time.Duration(50*(attempt+1)) * time.Millisecond)
		}
		return fmt.Errorf("renew lease: timed out due to database locks")
	}

	// Same rationale as Acquire/Release: SQLite in write-heavy windows may surface
	// transient lock errors even with busy_timeout configured.
	for attempt := 0; attempt < 6; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		err := l.runRequiresNew(ctx, func(runtimeScope scope.Scope) error {
			db := runtimeScope.Session().WithContext(ctx).Unscoped()
			return renewLeaseOnDB(db, resource, ownerId, expiresAt)
		})
		if err == nil {
			return nil
		}
		if errors.Is(err, ErrLeaseNotOwner) {
			return err
		}
		if !isSqliteLocked(err) {
			return err
		}
		time.Sleep(time.Duration(50*(attempt+1)) * time.Millisecond)
	}
	return fmt.Errorf("renew lease: timed out due to database locks")
}

func renewLeaseOnDB(db *gorm.DB, resource, ownerId string, expiresAt time.Time) error {
	res := db.Session(&gorm.Session{}).Unscoped().Model(&leasemodel.IrLockLease{}).
		Where("resource = ? AND owner_id = ?", resource, ownerId).
		Updates(map[string]any{"expires_at": expiresAt, "deleted_at": nil})
	if res.Error != nil {
		return fmt.Errorf("renew lease: %w", res.Error)
	}
	if res.RowsAffected != 1 {
		return ErrLeaseNotOwner
	}
	return nil
}

func (l *Locker) Release(ctx context.Context, resource, ownerId string) error {
	if resource == "" || ownerId == "" {
		return fmt.Errorf("resource/ownerId must be non-empty")
	}

	if db, ok := l.sqliteSessionDB(ctx); ok {
		newDB := func() *gorm.DB {
			return db.Session(&gorm.Session{}).Unscoped()
		}
		res := newDB().Where("resource = ? AND owner_id = ?", resource, ownerId).Delete(&leasemodel.IrLockLease{})
		if res.Error != nil {
			return fmt.Errorf("release lease: %w", res.Error)
		}
		if res.RowsAffected == 1 {
			return nil
		}
		var existing leasemodel.IrLockLease
		err := newDB().Where("resource = ?", resource).Take(&existing).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrLeaseNotHeld
			}
			return fmt.Errorf("release lease reload: %w", err)
		}
		return ErrLeaseNotOwner
	}

	// Same rationale as Acquire: bound retries to avoid compounding SQLite busy_timeout.
	for attempt := 0; attempt < 6; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		err := l.runRequiresNew(ctx, func(runtimeScope scope.Scope) error {
			newDB := func() *gorm.DB {
				return runtimeScope.Session().WithContext(ctx).Session(&gorm.Session{}).Unscoped()
			}

			// Hard delete to avoid unique-index conflicts with soft deletes.
			res := newDB().Where("resource = ? AND owner_id = ?", resource, ownerId).Delete(&leasemodel.IrLockLease{})
			if res.Error != nil {
				return fmt.Errorf("release lease: %w", res.Error)
			}
			if res.RowsAffected == 1 {
				return nil
			}

			// Distinguish "doesn't exist" vs "not owner".
			var existing leasemodel.IrLockLease
			err := newDB().Where("resource = ?", resource).Take(&existing).Error
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return ErrLeaseNotHeld
				}
				return fmt.Errorf("release lease reload: %w", err)
			}
			return ErrLeaseNotOwner
		})
		if err == nil {
			return nil
		}
		if errors.Is(err, ErrLeaseNotHeld) || errors.Is(err, ErrLeaseNotOwner) {
			return err
		}
		if !isSqliteLocked(err) {
			return err
		}
		time.Sleep(time.Duration(50*(attempt+1)) * time.Millisecond)
	}
	return fmt.Errorf("release lease: timed out due to database locks")
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	// Cross-DB heuristic (sqlite/mysql/postgres) without driver-specific imports.
	if strings.Contains(msg, "duplicate key") && strings.Contains(msg, "unique") {
		return true
	}
	if strings.Contains(msg, "duplicate entry") {
		return true
	}
	if strings.Contains(msg, "unique constraint") || strings.Contains(msg, "unique failed") {
		return true
	}
	if strings.Contains(msg, "duplicate") && strings.Contains(msg, "key") {
		return true
	}
	return false
}

func isSqliteLocked(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "database is locked") || strings.Contains(msg, "database schema is locked")
}
