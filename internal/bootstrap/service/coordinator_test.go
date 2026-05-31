// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package service

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	bootstrappb "github.com/choysum-dev/choysum/internal/bootstrap/proto/bootstrappb"
	modulestaging "github.com/choysum-dev/choysum/internal/module/artifact/staging"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/scope"
	statepkg "github.com/choysum-dev/choysum/pkg/state"
	"golang.org/x/crypto/bcrypt"
)

func TestCoordinatorStartInitializationSuccess(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()
	released := false
	callOrder := make([]string, 0, 5)
	installOpID := ""

	c := &coordinator{
		store: newInMemoryStatusStore(func() time.Time { return now }),
		now:   func() time.Time { return now },
		newOperationID: func() string {
			return "op-1"
		},
		runAsync: func(fn func()) {
			fn()
		},
		nextPollAfterMs: 500,
		acquireInitLease: func(ctx context.Context) (*leaseHandle, error) {
			_ = ctx
			return &leaseHandle{ownerID: "lease-owner", stopCh: make(chan struct{}), renewErr: make(chan error, 1)}, nil
		},
		releaseInitLease: func(handle *leaseHandle) {
			released = true
			close(handle.stopCh)
		},
		checkWorkspaceFreshness: func(ctx context.Context) error {
			_ = ctx
			callOrder = append(callOrder, "freshness")
			return nil
		},
		installMinimalModules: func(ctx context.Context) error {
			if got, ok := modulestaging.OpIDFromContext(ctx); ok {
				installOpID = got
			}
			callOrder = append(callOrder, "install")
			return nil
		},
		updateAdminAndMarker: func(ctx context.Context, input initializeInput) error {
			_ = ctx
			callOrder = append(callOrder, "update")
			if input.AdminUsername != "admin" {
				t.Fatalf("admin username = %q, want admin", input.AdminUsername)
			}
			return nil
		},
		validateRuntimeReady: func(ctx context.Context) error {
			_ = ctx
			callOrder = append(callOrder, "validate")
			return nil
		},
		switchMode: func(ctx context.Context) error {
			_ = ctx
			callOrder = append(callOrder, "switch")
			return nil
		},
	}

	_, reused, err := c.startInitialization(context.Background(), initializeInput{
		AdminUsername:  "admin",
		Password:       "secret",
		IdempotencyKey: "idem-1",
	})
	if err != nil {
		t.Fatalf("startInitialization() error = %v", err)
	}
	if reused {
		t.Fatal("expected first initialize call not to be reused")
	}

	op, ok := c.getInitializationStatus("op-1")
	if !ok {
		t.Fatal("expected operation op-1")
	}
	if op.State != bootstrappb.InitializationState_INITIALIZATION_STATE_SUCCEEDED {
		t.Fatalf("operation state = %v, want SUCCEEDED", op.State)
	}
	if op.Stage != bootstrappb.InitializationStage_INITIALIZATION_STAGE_DONE {
		t.Fatalf("operation stage = %v, want DONE", op.Stage)
	}
	if !op.ReadyForLogin || op.RedirectURL != bootstrapDefaultRedirectTo {
		t.Fatalf("ready/redirect = %v/%q, want true/%q", op.ReadyForLogin, op.RedirectURL, bootstrapDefaultRedirectTo)
	}
	if !released {
		t.Fatal("expected lease release to be called")
	}

	wantOrder := []string{"freshness", "install", "validate", "update", "switch"}
	if len(callOrder) != len(wantOrder) {
		t.Fatalf("call order length = %d, want %d (%v)", len(callOrder), len(wantOrder), callOrder)
	}
	for i, want := range wantOrder {
		if callOrder[i] != want {
			t.Fatalf("call order[%d] = %q, want %q (full order: %v)", i, callOrder[i], want, callOrder)
		}
	}
	if installOpID != "op-1" {
		t.Fatalf("install opid = %q, want %q", installOpID, "op-1")
	}
}

func TestCoordinatorStartInitializationIdempotencyReuse(t *testing.T) {
	c := &coordinator{
		store: newInMemoryStatusStore(time.Now),
		now:   time.Now,
		newOperationID: func() string {
			return "op-1"
		},
		runAsync: func(fn func()) {
			_ = fn
			// keep op pending to verify idempotent reuse path
		},
		nextPollAfterMs: 500,
	}

	op1, reused1, err := c.startInitialization(context.Background(), initializeInput{
		AdminUsername:  "admin",
		Password:       "secret",
		IdempotencyKey: "idem-1",
	})
	if err != nil {
		t.Fatalf("first startInitialization() error = %v", err)
	}
	if reused1 {
		t.Fatal("expected first operation not reused")
	}

	c.newOperationID = func() string { return "op-2" }
	op2, reused2, err := c.startInitialization(context.Background(), initializeInput{
		AdminUsername:  "admin",
		Password:       "secret",
		IdempotencyKey: "idem-1",
	})
	if err != nil {
		t.Fatalf("second startInitialization() error = %v", err)
	}
	if !reused2 {
		t.Fatal("expected second operation to be reused")
	}
	if op1.OperationID != op2.OperationID {
		t.Fatalf("reused operation id = %q, want %q", op2.OperationID, op1.OperationID)
	}
}

func TestCoordinatorStartInitializationConflict(t *testing.T) {
	c := &coordinator{
		store: newInMemoryStatusStore(time.Now),
		now:   time.Now,
		newOperationID: func() string {
			return "op-1"
		},
		runAsync: func(fn func()) {
			_ = fn
			// keep first operation active
		},
		nextPollAfterMs: 500,
	}

	_, _, err := c.startInitialization(context.Background(), initializeInput{
		AdminUsername:  "admin",
		Password:       "secret",
		IdempotencyKey: "idem-1",
	})
	if err != nil {
		t.Fatalf("first startInitialization() error = %v", err)
	}

	c.newOperationID = func() string { return "op-2" }
	_, _, err = c.startInitialization(context.Background(), initializeInput{
		AdminUsername:  "admin",
		Password:       "secret",
		IdempotencyKey: "idem-2",
	})
	if err == nil {
		t.Fatal("expected conflict error for concurrent initialization")
	}
	if got := bootstrapErrorCode(err); got != bootstrapErrCodeConflict {
		t.Fatalf("bootstrapErrorCode(conflict) = %q, want %q", got, bootstrapErrCodeConflict)
	}
}

func TestCoordinatorStartInitializationRejectsInvalidInput(t *testing.T) {
	c := &coordinator{
		store: newInMemoryStatusStore(time.Now),
		now:   time.Now,
		newOperationID: func() string {
			return "op-1"
		},
		runAsync:        func(fn func()) { _ = fn },
		nextPollAfterMs: 500,
	}

	_, _, err := c.startInitialization(context.Background(), initializeInput{})
	if err == nil {
		t.Fatal("expected input validation error")
	}
	if got := bootstrapErrorCode(err); got != bootstrapErrCodeInputInvalid {
		t.Fatalf("bootstrapErrorCode(input) = %q, want %q", got, bootstrapErrCodeInputInvalid)
	}
}

func TestCoordinatorStartInitializationRejectsNonFreshBeforeInstallAndAdmin(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()
	released := false
	installCalls := 0
	adminCalls := 0
	validateCalls := 0
	switchCalls := 0
	var logBuf bytes.Buffer

	c := &coordinator{
		runtimeScope: &freshnessTestScope{
			ctx:    context.Background(),
			config: &config.Config{},
			logger: slog.New(slog.NewTextHandler(io.MultiWriter(&logBuf, io.Discard), nil)),
		},
		store: newInMemoryStatusStore(func() time.Time { return now }),
		now:   func() time.Time { return now },
		newOperationID: func() string {
			return "op-nonfresh"
		},
		runAsync: func(fn func()) {
			fn()
		},
		nextPollAfterMs: 500,
		acquireInitLease: func(ctx context.Context) (*leaseHandle, error) {
			_ = ctx
			return &leaseHandle{ownerID: "lease-owner", stopCh: make(chan struct{}), renewErr: make(chan error, 1)}, nil
		},
		releaseInitLease: func(handle *leaseHandle) {
			released = true
			close(handle.stopCh)
		},
		checkWorkspaceFreshness: func(ctx context.Context) error {
			_ = ctx
			return newBootstrapError(bootstrapErrCodeWorkspaceNotFresh, "workspace is not fresh", nil)
		},
		installMinimalModules: func(ctx context.Context) error {
			_ = ctx
			installCalls++
			return nil
		},
		updateAdminAndMarker: func(ctx context.Context, input initializeInput) error {
			_ = ctx
			_ = input
			adminCalls++
			return nil
		},
		validateRuntimeReady: func(ctx context.Context) error {
			_ = ctx
			validateCalls++
			return nil
		},
		switchMode: func(ctx context.Context) error {
			_ = ctx
			switchCalls++
			return nil
		},
	}

	_, reused, err := c.startInitialization(context.Background(), initializeInput{
		AdminUsername:  "admin",
		Password:       "secret",
		IdempotencyKey: "idem-nonfresh",
	})
	if err != nil {
		t.Fatalf("startInitialization() error = %v", err)
	}
	if reused {
		t.Fatal("expected first initialize call not to be reused")
	}

	op, ok := c.getInitializationStatus("op-nonfresh")
	if !ok {
		t.Fatal("expected operation op-nonfresh")
	}
	if op.State != bootstrappb.InitializationState_INITIALIZATION_STATE_FAILED {
		t.Fatalf("operation state = %v, want FAILED", op.State)
	}
	if op.Stage != bootstrappb.InitializationStage_INITIALIZATION_STAGE_CHECK_WORKSPACE_FRESHNESS {
		t.Fatalf("operation stage = %v, want CHECK_WORKSPACE_FRESHNESS", op.Stage)
	}
	if op.ErrorCode != bootstrapErrCodeWorkspaceNotFresh {
		t.Fatalf("error code = %q, want %q", op.ErrorCode, bootstrapErrCodeWorkspaceNotFresh)
	}
	if installCalls != 0 {
		t.Fatalf("install calls = %d, want 0", installCalls)
	}
	if adminCalls != 0 {
		t.Fatalf("admin calls = %d, want 0", adminCalls)
	}
	if validateCalls != 0 {
		t.Fatalf("validate calls = %d, want 0", validateCalls)
	}
	if switchCalls != 0 {
		t.Fatalf("switch calls = %d, want 0", switchCalls)
	}
	if !released {
		t.Fatal("expected lease release to be called")
	}
	if !strings.Contains(logBuf.String(), "bootstrap initialization rejected") {
		t.Fatalf("expected non-fresh rejection log, got logs: %s", logBuf.String())
	}
}

type coordinatorTestLocker struct {
	acquireErr error
	acquired   int
	released   int
}

func (l *coordinatorTestLocker) Acquire(context.Context, string, string, time.Duration) error {
	l.acquired++
	return l.acquireErr
}

func (l *coordinatorTestLocker) Renew(context.Context, string, string, time.Duration) error {
	return nil
}

func (l *coordinatorTestLocker) Release(context.Context, string, string) error {
	l.released++
	return nil
}

func TestCoordinatorDefaultAcquireInitLeaseUsesLockerFactory(t *testing.T) {
	base, _ := newFreshnessTestCoordinator(t)
	locker := &coordinatorTestLocker{}
	c := newCoordinator(base.runtimeScope)
	c.lockerFactory = func(scope.Scope) statepkg.Locker {
		return locker
	}

	handle, err := c.defaultAcquireInitLease(context.Background())
	if err != nil {
		t.Fatalf("defaultAcquireInitLease() error = %v", err)
	}
	if handle == nil {
		t.Fatal("expected non-nil lease handle")
	}
	if locker.acquired != 1 {
		t.Fatalf("locker Acquire calls = %d, want 1", locker.acquired)
	}
	c.defaultReleaseInitLease(handle)
	if locker.released != 1 {
		t.Fatalf("locker Release calls = %d, want 1", locker.released)
	}
}

func TestCoordinatorDefaultAcquireInitLeaseMapsLeaseBusy(t *testing.T) {
	base, _ := newFreshnessTestCoordinator(t)
	locker := &coordinatorTestLocker{acquireErr: statepkg.ErrLeaseBusy}
	c := newCoordinator(base.runtimeScope)
	c.lockerFactory = func(scope.Scope) statepkg.Locker {
		return locker
	}

	handle, err := c.defaultAcquireInitLease(context.Background())
	if err == nil {
		t.Fatal("expected busy error")
	}
	if handle != nil {
		t.Fatalf("expected nil handle on busy error, got %#v", handle)
	}
	if got := bootstrapErrorCode(err); got != bootstrapErrCodeConflict {
		t.Fatalf("bootstrapErrorCode(busy) = %q, want %q", got, bootstrapErrCodeConflict)
	}
}

func TestCoordinatorStartInitializationSkipsAdminWhenRuntimeNotReady(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()
	released := false
	installCalls := 0
	validateCalls := 0
	adminCalls := 0
	switchCalls := 0

	c := &coordinator{
		store: newInMemoryStatusStore(func() time.Time { return now }),
		now:   func() time.Time { return now },
		newOperationID: func() string {
			return "op-runtime-not-ready"
		},
		runAsync: func(fn func()) {
			fn()
		},
		nextPollAfterMs: 500,
		acquireInitLease: func(ctx context.Context) (*leaseHandle, error) {
			_ = ctx
			return &leaseHandle{ownerID: "lease-owner", stopCh: make(chan struct{}), renewErr: make(chan error, 1)}, nil
		},
		releaseInitLease: func(handle *leaseHandle) {
			released = true
			close(handle.stopCh)
		},
		checkWorkspaceFreshness: func(ctx context.Context) error {
			_ = ctx
			return nil
		},
		installMinimalModules: func(ctx context.Context) error {
			_ = ctx
			installCalls++
			return nil
		},
		validateRuntimeReady: func(ctx context.Context) error {
			_ = ctx
			validateCalls++
			return newBootstrapError(bootstrapErrCodeRuntimeNotReady, "required system files are not ready", nil)
		},
		updateAdminAndMarker: func(ctx context.Context, input initializeInput) error {
			_ = ctx
			_ = input
			adminCalls++
			return nil
		},
		switchMode: func(ctx context.Context) error {
			_ = ctx
			switchCalls++
			return nil
		},
	}

	_, reused, err := c.startInitialization(context.Background(), initializeInput{
		AdminUsername:  "admin",
		Password:       "secret",
		IdempotencyKey: "idem-runtime-not-ready",
	})
	if err != nil {
		t.Fatalf("startInitialization() error = %v", err)
	}
	if reused {
		t.Fatal("expected first initialize call not to be reused")
	}

	op, ok := c.getInitializationStatus("op-runtime-not-ready")
	if !ok {
		t.Fatal("expected operation op-runtime-not-ready")
	}
	if op.State != bootstrappb.InitializationState_INITIALIZATION_STATE_FAILED {
		t.Fatalf("operation state = %v, want FAILED", op.State)
	}
	if op.Stage != bootstrappb.InitializationStage_INITIALIZATION_STAGE_VALIDATE_RUNTIME_READY {
		t.Fatalf("operation stage = %v, want VALIDATE_RUNTIME_READY", op.Stage)
	}
	if op.ErrorCode != bootstrapErrCodeRuntimeNotReady {
		t.Fatalf("error code = %q, want %q", op.ErrorCode, bootstrapErrCodeRuntimeNotReady)
	}
	if installCalls != 1 {
		t.Fatalf("install calls = %d, want 1", installCalls)
	}
	if validateCalls != 1 {
		t.Fatalf("validate calls = %d, want 1", validateCalls)
	}
	if adminCalls != 0 {
		t.Fatalf("admin calls = %d, want 0", adminCalls)
	}
	if switchCalls != 0 {
		t.Fatalf("switch calls = %d, want 0", switchCalls)
	}
	if !released {
		t.Fatal("expected lease release to be called")
	}
}

func TestNormalizeWirePasswordRejectsRawPassword(t *testing.T) {
	_, err := normalizeWirePassword("plain-password")
	if err == nil {
		t.Fatal("expected error for non-prefixed password")
	}
	if bootstrapErrorCode(err) != bootstrapErrCodeInputInvalid {
		t.Fatalf("bootstrapErrorCode(err) = %q, want %q", bootstrapErrorCode(err), bootstrapErrCodeInputInvalid)
	}
	if !strings.Contains(err.Error(), "password must be client-hashed") {
		t.Fatalf("error = %q, want client-hashed requirement", err.Error())
	}
}

func TestHashAdminPasswordForBootstrapAcceptsClientHash(t *testing.T) {
	wirePassword := "$CH$ABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCD"
	got, err := hashAdminPasswordForBootstrap(wirePassword)
	if err != nil {
		t.Fatalf("hashAdminPasswordForBootstrap() error = %v", err)
	}
	if got == "" {
		t.Fatal("expected non-empty bcrypt hash")
	}

	wantClientHashHex := strings.ToLower(strings.TrimPrefix(wirePassword, "$CH$"))
	if cmpErr := bcrypt.CompareHashAndPassword([]byte(got), []byte(wantClientHashHex)); cmpErr != nil {
		t.Fatalf("bcrypt.CompareHashAndPassword() error = %v", cmpErr)
	}
}

func TestNormalizeWirePasswordValidations(t *testing.T) {
	t.Run("rejects invalid hash length", func(t *testing.T) {
		_, err := normalizeWirePassword("$CH$abcd")
		if err == nil {
			t.Fatal("expected invalid length error")
		}
		if bootstrapErrorCode(err) != bootstrapErrCodeInputInvalid || !strings.Contains(err.Error(), "invalid client hash length") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("rejects invalid hash encoding", func(t *testing.T) {
		_, err := normalizeWirePassword("$CH$GGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGG")
		if err == nil {
			t.Fatal("expected invalid encoding error")
		}
		if bootstrapErrorCode(err) != bootstrapErrCodeInputInvalid || !strings.Contains(err.Error(), "invalid client hash encoding") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("normalizes uppercase hex", func(t *testing.T) {
		got, err := normalizeWirePassword("$CH$ABCDEFABCDEFABCDEFABCDEFABCDEFABCDEFABCDEFABCDEFABCDEFABCDEFABCD")
		if err != nil {
			t.Fatalf("normalizeWirePassword() error = %v", err)
		}
		if got != strings.ToLower("ABCDEFABCDEFABCDEFABCDEFABCDEFABCDEFABCDEFABCDEFABCDEFABCDEFABCD") {
			t.Fatalf("normalized hash = %q", got)
		}
	})
}
