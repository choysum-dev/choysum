// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package state

import (
	"context"
	"errors"
	"time"

	"github.com/choysum-dev/choysum/pkg/scope"
)

var (
	ErrLeaseBusy       = errors.New("lease is busy")
	ErrLeaseNotOwner   = errors.New("lease not owned")
	ErrLeaseNotHeld    = errors.New("lease not held")
	ErrInvalidLeaseTTL = errors.New("invalid lease ttl")
)

// Locker freezes the minimal mutex-style lease semantics used by the default
// CE runtime. It does not imply fencing-token guarantees.
type Locker interface {
	Acquire(ctx context.Context, resource string, ownerID string, ttl time.Duration) error
	Renew(ctx context.Context, resource string, ownerID string, ttl time.Duration) error
	Release(ctx context.Context, resource string, ownerID string) error
}

// LockerFactory constructs a scope-bound Locker implementation.
type LockerFactory = func(scope.Scope) Locker
