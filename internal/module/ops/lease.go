// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package ops

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/choysum-dev/choysum/pkg/oerrors"
	"github.com/choysum-dev/choysum/pkg/scope"
	statepkg "github.com/choysum-dev/choysum/pkg/state"
	"github.com/rs/xid"
)

const moduleManagementLeaseResource = "module_management"

// Overridable in tests so renew/cancel paths can run without a 60s wait.
var moduleManagementLeaseTTL = 60 * time.Second

// withModuleManagementLease serializes mutating ops against lifecycle install/upgrade/uninstall
// using the same lease resource ModuleManager holds.
// fn receives a cancelable context that is cancelled if the lease is lost on Renew.
func (o *Ops) withModuleManagementLease(ctx context.Context, fn func(context.Context) error) error {
	if o == nil {
		return errors.New("ops: nil host")
	}
	if fn == nil {
		return nil
	}
	if o.LockerFactory == nil || o.Scope == nil {
		if ctx == nil {
			ctx = context.Background()
		}
		return fn(ctx)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	locker := o.LockerFactory(o.Scope)
	if locker == nil {
		return fn(ctx)
	}
	ownerID := xid.New().String()
	ttl := moduleManagementLeaseTTL
	if ttl <= 0 {
		ttl = 60 * time.Second
	}
	if err := locker.Acquire(ctx, moduleManagementLeaseResource, ownerID, ttl); err != nil {
		if errors.Is(err, statepkg.ErrLeaseBusy) {
			retryAfterMs := int64(ttl / time.Millisecond)
			return oerrors.New("meta.lock", "LEASE_CONFLICT", "lease is busy").
				WithMetadata("retry_after_ms", fmt.Sprintf("%d", retryAfterMs))
		}
		return err
	}

	opCtx, cancelOp := context.WithCancel(ctx)
	defer cancelOp()

	heartbeatCtx, cancelHeartbeat := context.WithCancel(ctx)
	defer cancelHeartbeat()
	done := make(chan struct{})
	go func() {
		defer close(done)
		interval := ttl / 2
		if interval <= 0 {
			interval = time.Second
		}
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-heartbeatCtx.Done():
				return
			case <-ticker.C:
				if err := locker.Renew(heartbeatCtx, moduleManagementLeaseResource, ownerID, ttl); err != nil {
					if lg := scopeLogger(o.Scope); lg != nil {
						lg.Warn("ops module_management lease renew failed", "error", err)
					}
					if errors.Is(err, statepkg.ErrLeaseNotOwner) || errors.Is(err, statepkg.ErrLeaseNotHeld) {
						cancelOp()
						return
					}
				}
			}
		}
	}()
	defer func() {
		cancelHeartbeat()
		<-done
		releaseOpsLease(o.Scope, locker, ctx, moduleManagementLeaseResource, ownerID)
	}()
	return fn(opCtx)
}

func releaseOpsLease(runtimeScope scope.Scope, locker statepkg.Locker, operationCtx context.Context, resource, ownerID string) {
	if locker == nil {
		return
	}
	tryRelease := func(c context.Context) error {
		if c == nil {
			c = context.Background()
		}
		return locker.Release(c, resource, ownerID)
	}
	if err := tryRelease(operationCtx); err == nil {
		return
	} else if errors.Is(err, statepkg.ErrLeaseNotOwner) || errors.Is(err, statepkg.ErrLeaseNotHeld) {
		return
	}
	if err := tryRelease(context.Background()); err != nil &&
		!errors.Is(err, statepkg.ErrLeaseNotOwner) &&
		!errors.Is(err, statepkg.ErrLeaseNotHeld) {
		if lg := scopeLogger(runtimeScope); lg != nil {
			lg.Warn("ops lease release failed", "resource", resource, "error", err)
		}
	}
}

func scopeLogger(runtimeScope scope.Scope) *slog.Logger {
	if runtimeScope == nil {
		return nil
	}
	return runtimeScope.Logger()
}
