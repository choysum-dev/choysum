// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package provision

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	modmeta "github.com/choysum-dev/choysum/internal/module/meta"
	"github.com/choysum-dev/choysum/internal/state/lease"
	"github.com/choysum-dev/choysum/pkg/scope"
	statepkg "github.com/choysum-dev/choysum/pkg/state"
	"github.com/rs/xid"
	xfmt "golang.org/x/exp/errors/fmt"
)

const (
	// LeaseResource is shared with bootstrap Initialize so host provision and
	// the legacy Workspace button cannot install concurrently.
	LeaseResource = "system:init"
)

// leaseTTL / leaseRenewInterval / leaseReleaseWait are overridable in tests so
// renew failure, retry-bound, and release-wait paths stay fast.
var (
	leaseTTL           = 60 * time.Second
	leaseRenewInterval = 20 * time.Second
	leaseReleaseWait   = 5 * time.Second
)

// ErrLeaseBusy means another provision/init holds the workspace lease.
var ErrLeaseBusy = lease.ErrLeaseBusy

type leaseHandle struct {
	ownerID  string
	stopCh   chan struct{}
	done     chan struct{} // closed when renewProvisionLease exits
	renewErr chan error
	cancel   context.CancelFunc
	release  sync.Once
}

func defaultLockerFactory(runtimeScope scope.Scope) statepkg.Locker {
	return lease.New(runtimeScope)
}

func acquireProvisionLease(ctx context.Context, runtimeScope scope.Scope, lockerFactory statepkg.LockerFactory, cancel context.CancelFunc) (*leaseHandle, error) {
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
		done:     make(chan struct{}),
		renewErr: make(chan error, 1),
		cancel:   cancel,
	}
	probeDone := make(chan struct{})
	go renewProvisionLease(ctx, runtimeScope, locker, handle, probeDone)
	<-probeDone
	if err := checkProvisionLeaseRenew(handle); err != nil {
		releaseProvisionLease(runtimeScope, lockerFactory, handle)
		return nil, xfmt.Errorf("provision lease lost: %w", err)
	}
	return handle, nil
}

func renewProvisionLease(ctx context.Context, runtimeScope scope.Scope, locker statepkg.Locker, handle *leaseHandle, probeDone chan struct{}) {
	defer close(handle.done)
	var probeOnce sync.Once
	signalProbe := func() {
		if probeDone == nil {
			return
		}
		probeOnce.Do(func() { close(probeDone) })
	}
	defer signalProbe()

	ticker := time.NewTicker(leaseRenewInterval)
	defer ticker.Stop()
	lastOK := time.Now()
	renewOnce := func() (stop bool) {
		err := locker.Renew(ctx, LeaseResource, handle.ownerID, leaseTTL)
		if err == nil {
			lastOK = time.Now()
			return false
		}
		notOwner := errors.Is(err, lease.ErrLeaseNotOwner)
		if !notOwner && ctx.Err() == nil && isRecoverableSqliteLeaseErr(runtimeScope, err) {
			return false
		}
		// Retry transient dialect errors while the last successful renew
		// still leaves time before another owner can take the lease.
		if !notOwner && ctx.Err() == nil && time.Since(lastOK) < leaseTTL-leaseRenewInterval {
			return false
		}
		notifyLeaseRenewErr(handle, err)
		return true
	}
	// Probe once immediately so a lost lease is visible before install starts.
	if renewOnce() {
		signalProbe()
		return
	}
	signalProbe()
	for {
		select {
		case <-handle.stopCh:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			if renewOnce() {
				return
			}
		}
	}
}

func notifyLeaseRenewErr(handle *leaseHandle, err error) {
	if handle == nil {
		return
	}
	select {
	case handle.renewErr <- err:
	default:
	}
	if handle.cancel != nil {
		handle.cancel()
	}
}

func releaseProvisionLease(runtimeScope scope.Scope, lockerFactory statepkg.LockerFactory, handle *leaseHandle) {
	if handle == nil {
		return
	}
	handle.release.Do(func() {
		close(handle.stopCh)
		if handle.done != nil {
			select {
			case <-handle.done:
			case <-time.After(leaseReleaseWait):
			}
		}
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
	})
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
