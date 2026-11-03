// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package provision

import (
	"context"
	"errors"
	"strings"
	"time"

	modmeta "github.com/choysum-dev/choysum/internal/module/meta"
	"github.com/choysum-dev/choysum/internal/state/lease"
	"github.com/choysum-dev/choysum/pkg/scope"
	statepkg "github.com/choysum-dev/choysum/pkg/state"
	"github.com/rs/xid"
)

const (
	// LeaseResource is shared with bootstrap Initialize so host provision and
	// the legacy Workspace button cannot install concurrently.
	LeaseResource = "system:init"
	leaseTTL      = 60 * time.Second
	leaseRenew    = 20 * time.Second
)

// ErrLeaseBusy means another provision/init holds the workspace lease.
var ErrLeaseBusy = lease.ErrLeaseBusy

type leaseHandle struct {
	ownerID  string
	stopCh   chan struct{}
	renewErr chan error
}

func defaultLockerFactory(runtimeScope scope.Scope) statepkg.Locker {
	return lease.New(runtimeScope)
}

func acquireProvisionLease(ctx context.Context, runtimeScope scope.Scope, lockerFactory statepkg.LockerFactory) (*leaseHandle, error) {
	if runtimeScope == nil {
		return nil, errors.New("scope is not available")
	}
	session := runtimeScope.Session()
	if session == nil || session.DB == nil {
		return nil, errors.New("database session is not available")
	}
	if !session.Migrator().HasTable((&modmeta.LockLease{}).TableName()) {
		if err := session.AutoMigrate(&modmeta.LockLease{}); err != nil {
			return nil, err
		}
	}
	if lockerFactory == nil {
		lockerFactory = defaultLockerFactory
	}
	locker := lockerFactory(runtimeScope)
	ownerID := xid.New().String()
	if err := locker.Acquire(ctx, LeaseResource, ownerID, leaseTTL); err != nil {
		return nil, err
	}
	handle := &leaseHandle{
		ownerID:  ownerID,
		stopCh:   make(chan struct{}),
		renewErr: make(chan error, 1),
	}
	go renewProvisionLease(ctx, runtimeScope, locker, handle)
	return handle, nil
}

func renewProvisionLease(ctx context.Context, runtimeScope scope.Scope, locker statepkg.Locker, handle *leaseHandle) {
	ticker := time.NewTicker(leaseRenew)
	defer ticker.Stop()
	for {
		select {
		case <-handle.stopCh:
			return
		case <-ticker.C:
			if err := locker.Renew(ctx, LeaseResource, handle.ownerID, leaseTTL); err != nil {
				if errors.Is(err, lease.ErrLeaseNotOwner) || !isRecoverableSqliteLeaseErr(runtimeScope, err) {
					select {
					case handle.renewErr <- err:
					default:
					}
					return
				}
			}
		}
	}
}

func releaseProvisionLease(runtimeScope scope.Scope, lockerFactory statepkg.LockerFactory, handle *leaseHandle) {
	if handle == nil {
		return
	}
	close(handle.stopCh)
	if runtimeScope == nil {
		return
	}
	if lockerFactory == nil {
		lockerFactory = defaultLockerFactory
	}
	locker := lockerFactory(runtimeScope)
	releaseCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = locker.Release(releaseCtx, LeaseResource, handle.ownerID)
}

func checkProvisionLeaseRenew(handle *leaseHandle) error {
	if handle == nil {
		return nil
	}
	select {
	case err := <-handle.renewErr:
		return err
	default:
		return nil
	}
}

func isRecoverableSqliteLeaseErr(runtimeScope scope.Scope, err error) bool {
	if err == nil || runtimeScope == nil {
		return false
	}
	dbOpts, ok := scope.DatabaseRuntimeOptionsFromScope(runtimeScope)
	if !ok || !strings.EqualFold(strings.TrimSpace(dbOpts.Dialect), "sqlite") {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "database is locked") || strings.Contains(msg, "database schema is locked")
}
