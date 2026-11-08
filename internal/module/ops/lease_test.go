// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package ops

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/choysum-dev/choysum/pkg/oerrors"
	"github.com/choysum-dev/choysum/pkg/scope"
	statepkg "github.com/choysum-dev/choysum/pkg/state"
)

type leaseTestLocker struct {
	mu           sync.Mutex
	acquireErr   error
	renewErr     error
	renewAfterN  int
	renewCalls   int
	releaseErr   error
	releaseCalls int
	acquired     int
}

func (l *leaseTestLocker) Acquire(context.Context, string, string, time.Duration) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.acquired++
	return l.acquireErr
}

func (l *leaseTestLocker) Renew(context.Context, string, string, time.Duration) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.renewCalls++
	if l.renewAfterN > 0 && l.renewCalls >= l.renewAfterN {
		return l.renewErr
	}
	return nil
}

func (l *leaseTestLocker) Release(context.Context, string, string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.releaseCalls++
	return l.releaseErr
}

type flakyReleaseLocker struct {
	onRelease func() error
}

func (l *flakyReleaseLocker) Acquire(context.Context, string, string, time.Duration) error {
	return nil
}
func (l *flakyReleaseLocker) Renew(context.Context, string, string, time.Duration) error {
	return nil
}
func (l *flakyReleaseLocker) Release(context.Context, string, string) error {
	return l.onRelease()
}

func TestWithModuleManagementLease_NilHostAndNilFn(t *testing.T) {
	t.Parallel()
	if err := (*Ops)(nil).withModuleManagementLease(context.Background(), func(context.Context) error {
		return nil
	}); err == nil || !strings.Contains(err.Error(), "nil host") {
		t.Fatalf("nil host: %v", err)
	}
	o := &Ops{}
	if err := o.withModuleManagementLease(context.Background(), nil); err != nil {
		t.Fatalf("nil fn: %v", err)
	}
}

func TestWithModuleManagementLease_NoLockerPassthrough(t *testing.T) {
	t.Parallel()
	db := newOpsTestDB(t)
	o := &Ops{Scope: newOpsTestScope(t, db)}
	called := false
	if err := o.withModuleManagementLease(nil, func(ctx context.Context) error {
		called = true
		if ctx == nil {
			t.Fatal("expected non-nil ctx")
		}
		return nil
	}); err != nil {
		t.Fatalf("passthrough: %v", err)
	}
	if !called {
		t.Fatal("fn not called")
	}
	o.LockerFactory = func(scope.Scope) statepkg.Locker { return nil }
	if err := o.withModuleManagementLease(context.Background(), func(context.Context) error { return errors.New("boom") }); err == nil || err.Error() != "boom" {
		t.Fatalf("nil locker fn err = %v", err)
	}
}

func TestWithModuleManagementLease_AcquireBusy(t *testing.T) {
	t.Parallel()
	db := newOpsTestDB(t)
	locker := &leaseTestLocker{acquireErr: statepkg.ErrLeaseBusy}
	o := &Ops{
		Scope: newOpsTestScope(t, db),
		LockerFactory: func(scope.Scope) statepkg.Locker {
			return locker
		},
	}
	err := o.withModuleManagementLease(context.Background(), func(context.Context) error {
		t.Fatal("fn should not run")
		return nil
	})
	if err == nil {
		t.Fatal("expected LEASE_CONFLICT")
	}
	info := oerrors.GetErrorInfo(err)
	if info == nil || info.Code != "LEASE_CONFLICT" {
		t.Fatalf("err = %v", err)
	}
}

func TestWithModuleManagementLease_AcquireOtherError(t *testing.T) {
	t.Parallel()
	db := newOpsTestDB(t)
	locker := &leaseTestLocker{acquireErr: errors.New("acquire boom")}
	o := &Ops{
		Scope: newOpsTestScope(t, db),
		LockerFactory: func(scope.Scope) statepkg.Locker {
			return locker
		},
	}
	if err := o.withModuleManagementLease(context.Background(), func(context.Context) error {
		return nil
	}); err == nil || !strings.Contains(err.Error(), "acquire boom") {
		t.Fatalf("err = %v", err)
	}
}

func TestWithModuleManagementLease_SuccessReleases(t *testing.T) {
	t.Parallel()
	db := newOpsTestDB(t)
	locker := &leaseTestLocker{}
	o := &Ops{
		Scope: newOpsTestScope(t, db),
		LockerFactory: func(scope.Scope) statepkg.Locker {
			return locker
		},
	}
	if err := o.withModuleManagementLease(context.Background(), func(ctx context.Context) error {
		if ctx.Err() != nil {
			t.Fatalf("ctx already cancelled: %v", ctx.Err())
		}
		return nil
	}); err != nil {
		t.Fatalf("lease: %v", err)
	}
	if locker.acquired != 1 || locker.releaseCalls != 1 {
		t.Fatalf("acquired=%d released=%d", locker.acquired, locker.releaseCalls)
	}
}

func TestWithModuleManagementLease_RenewNotOwnerCancelsFn(t *testing.T) {
	// Not parallel: mutates package-level moduleManagementLeaseTTL.
	db := newOpsTestDB(t)
	prevTTL := moduleManagementLeaseTTL
	moduleManagementLeaseTTL = 40 * time.Millisecond
	t.Cleanup(func() { moduleManagementLeaseTTL = prevTTL })

	locker := &leaseTestLocker{renewAfterN: 1, renewErr: statepkg.ErrLeaseNotOwner}
	o := &Ops{
		Scope: newOpsTestScope(t, db),
		LockerFactory: func(scope.Scope) statepkg.Locker {
			return locker
		},
	}
	sawCancel := false
	err := o.withModuleManagementLease(context.Background(), func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			sawCancel = true
			return ctx.Err()
		case <-time.After(2 * time.Second):
			return errors.New("timeout waiting for lease cancel")
		}
	})
	if !sawCancel {
		t.Fatal("expected fn context cancelled on renew loss")
	}
	if err == nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v", err)
	}
}

func TestWithModuleManagementLease_RenewNotHeldCancelsFn(t *testing.T) {
	// Not parallel: mutates package-level moduleManagementLeaseTTL.
	db := newOpsTestDB(t)
	prevTTL := moduleManagementLeaseTTL
	moduleManagementLeaseTTL = 40 * time.Millisecond
	t.Cleanup(func() { moduleManagementLeaseTTL = prevTTL })

	locker := &leaseTestLocker{renewAfterN: 1, renewErr: statepkg.ErrLeaseNotHeld}
	o := &Ops{
		Scope: newOpsTestScope(t, db),
		LockerFactory: func(scope.Scope) statepkg.Locker {
			return locker
		},
	}
	err := o.withModuleManagementLease(context.Background(), func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	})
	if err == nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v", err)
	}
}

func TestWithModuleManagementLease_TransientRenewContinues(t *testing.T) {
	// Not parallel: mutates package-level moduleManagementLeaseTTL.
	db := newOpsTestDB(t)
	prevTTL := moduleManagementLeaseTTL
	moduleManagementLeaseTTL = 40 * time.Millisecond
	t.Cleanup(func() { moduleManagementLeaseTTL = prevTTL })

	locker := &leaseTestLocker{renewAfterN: 1, renewErr: errors.New("transient")}
	o := &Ops{
		Scope: newOpsTestScope(t, db),
		LockerFactory: func(scope.Scope) statepkg.Locker {
			return locker
		},
	}
	done := make(chan struct{})
	err := o.withModuleManagementLease(context.Background(), func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return errors.New("should not cancel on transient renew")
		case <-time.After(80 * time.Millisecond):
			close(done)
			return nil
		}
	})
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	select {
	case <-done:
	default:
		t.Fatal("fn did not complete")
	}
}

func TestReleaseOpsLease_Paths(t *testing.T) {
	t.Parallel()
	releaseOpsLease(nil, nil, context.Background(), "r", "o")

	releaseOpsLease(nil, &leaseTestLocker{releaseErr: statepkg.ErrLeaseNotOwner}, context.Background(), "r", "o")

	db := newOpsTestDB(t)
	releaseOpsLease(newOpsTestScope(t, db), &leaseTestLocker{releaseErr: errors.New("persist fail")}, context.Background(), "r", "o")

	calls := 0
	flaky := &flakyReleaseLocker{onRelease: func() error {
		calls++
		if calls == 1 {
			return errors.New("transient release")
		}
		return nil
	}}
	releaseOpsLease(newOpsTestScope(t, db), flaky, context.Background(), "r", "o")
	if calls != 2 {
		t.Fatalf("release calls = %d", calls)
	}

	releaseOpsLease(newOpsTestScope(t, db), &leaseTestLocker{releaseErr: statepkg.ErrLeaseNotHeld}, nil, "r", "o")
}

func TestScopeLoggerNil(t *testing.T) {
	t.Parallel()
	if scopeLogger(nil) != nil {
		t.Fatal("expected nil logger")
	}
}
