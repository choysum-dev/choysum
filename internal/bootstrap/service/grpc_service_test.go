// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package service

import (
	"context"
	"net/url"
	"testing"
	"time"

	bootstrappb "github.com/choysum-dev/choysum/internal/bootstrap/proto/bootstrappb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeWorkspaceCoordinator struct {
	startFn func(ctx context.Context, input initializeInput) (operationSnapshot, bool, error)
	getFn   func(operationID string) (operationSnapshot, bool)
}

func (f *fakeWorkspaceCoordinator) startInitialization(ctx context.Context, input initializeInput) (operationSnapshot, bool, error) {
	if f.startFn == nil {
		return operationSnapshot{}, false, nil
	}
	return f.startFn(ctx, input)
}

func (f *fakeWorkspaceCoordinator) getInitializationStatus(operationID string) (operationSnapshot, bool) {
	if f.getFn == nil {
		return operationSnapshot{}, false
	}
	return f.getFn(operationID)
}

func TestWorkspaceServerInitializeRejectsNilRequest(t *testing.T) {
	srv := &workspaceServer{coordinator: &fakeWorkspaceCoordinator{}}

	_, err := srv.Initialize(context.Background(), nil)
	if err == nil {
		t.Fatal("expected invalid argument error")
	}
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
}

func TestWorkspaceServerInitializeMapsCoordinatorError(t *testing.T) {
	srv := &workspaceServer{coordinator: &fakeWorkspaceCoordinator{
		startFn: func(ctx context.Context, input initializeInput) (operationSnapshot, bool, error) {
			_ = ctx
			_ = input
			return operationSnapshot{}, false, newBootstrapError(bootstrapErrCodeInputInvalid, "invalid input", nil)
		},
	}}

	_, err := srv.Initialize(context.Background(), &bootstrappb.Workspace_Initialize_Req{AdminUsername: " ", Password: ""})
	if err == nil {
		t.Fatal("expected mapped grpc error")
	}
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
}

func TestWorkspaceServerInitializeMapsWorkspaceNotFreshError(t *testing.T) {
	srv := &workspaceServer{coordinator: &fakeWorkspaceCoordinator{
		startFn: func(ctx context.Context, input initializeInput) (operationSnapshot, bool, error) {
			_ = ctx
			_ = input
			return operationSnapshot{}, false, newBootstrapError(bootstrapErrCodeWorkspaceNotFresh, "workspace is not fresh", nil)
		},
	}}

	_, err := srv.Initialize(context.Background(), &bootstrappb.Workspace_Initialize_Req{AdminUsername: "admin", Password: "secret"})
	if err == nil {
		t.Fatal("expected mapped grpc error")
	}
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.FailedPrecondition)
	}
}

func TestWorkspaceServerInitializeMapsBootstrapErrorCodes(t *testing.T) {
	tests := []struct {
		name     string
		errCode  string
		wantCode codes.Code
	}{
		{name: "runtime prepare", errCode: bootstrapErrCodeRuntimePrepare, wantCode: codes.FailedPrecondition},
		{name: "runtime not ready", errCode: bootstrapErrCodeRuntimeNotReady, wantCode: codes.FailedPrecondition},
		{name: "conflict", errCode: bootstrapErrCodeConflict, wantCode: codes.Aborted},
		{name: "switch failed", errCode: bootstrapErrCodeSwitchFailed, wantCode: codes.Unavailable},
		{name: "admin update failed", errCode: bootstrapErrCodeAdminUpdateFailed, wantCode: codes.Internal},
		{name: "gate error", errCode: bootstrapErrCodeGateError, wantCode: codes.Internal},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := &workspaceServer{coordinator: &fakeWorkspaceCoordinator{
				startFn: func(ctx context.Context, input initializeInput) (operationSnapshot, bool, error) {
					_ = ctx
					_ = input
					return operationSnapshot{}, false, newBootstrapError(tc.errCode, "boom", nil)
				},
			}}

			_, err := srv.Initialize(context.Background(), &bootstrappb.Workspace_Initialize_Req{AdminUsername: "admin", Password: "secret"})
			if err == nil {
				t.Fatal("expected mapped grpc error")
			}
			if status.Code(err) != tc.wantCode {
				t.Fatalf("status code = %v, want %v", status.Code(err), tc.wantCode)
			}
		})
	}
}

func TestWorkspaceServerGetInitializationStatusNotFound(t *testing.T) {
	srv := &workspaceServer{coordinator: &fakeWorkspaceCoordinator{
		getFn: func(operationID string) (operationSnapshot, bool) {
			_ = operationID
			return operationSnapshot{}, false
		},
	}}

	_, err := srv.GetInitializationStatus(context.Background(), &bootstrappb.Workspace_GetInitializationStatus_Req{OperationId: "missing"})
	if err == nil {
		t.Fatal("expected not found error")
	}
	if status.Code(err) != codes.NotFound {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.NotFound)
	}
}

func TestWorkspaceServerGetInitializationStatusSuccess(t *testing.T) {
	srv := &workspaceServer{coordinator: &fakeWorkspaceCoordinator{
		getFn: func(operationID string) (operationSnapshot, bool) {
			return operationSnapshot{
				OperationID:     operationID,
				State:           bootstrappb.InitializationState_INITIALIZATION_STATE_RUNNING,
				Stage:           bootstrappb.InitializationStage_INITIALIZATION_STAGE_UPDATE_ADMIN,
				ProgressPercent: 66,
				NextPollAfterMs: 1000,
			}, true
		},
	}}

	resp, err := srv.GetInitializationStatus(context.Background(), &bootstrappb.Workspace_GetInitializationStatus_Req{OperationId: "op-1"})
	if err != nil {
		t.Fatalf("GetInitializationStatus() error = %v", err)
	}
	if resp.GetOperationId() != "op-1" {
		t.Fatalf("operation_id = %q, want op-1", resp.GetOperationId())
	}
	if resp.GetState() != bootstrappb.InitializationState_INITIALIZATION_STATE_RUNNING {
		t.Fatalf("state = %v, want RUNNING", resp.GetState())
	}
}

func TestWorkspaceServerInitializeFlowReadyForLoginRedirect(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()
	released := false
	switchCalls := 0

	c := &coordinator{
		store: newInMemoryStatusStore(func() time.Time { return now }),
		now:   func() time.Time { return now },
		newOperationID: func() string {
			return "op-e2e-1"
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
		installMinimalModules: func(ctx context.Context) error {
			_ = ctx
			return nil
		},
		updateAdminAndMarker: func(ctx context.Context, input initializeInput) error {
			_ = ctx
			if input.AdminUsername != "admin" {
				t.Fatalf("admin username = %q, want admin", input.AdminUsername)
			}
			return nil
		},
		validateRuntimeReady: func(ctx context.Context) error {
			_ = ctx
			return nil
		},
		switchMode: func(ctx context.Context) error {
			_ = ctx
			switchCalls++
			return nil
		},
	}

	srv := &workspaceServer{coordinator: c}

	initResp, err := srv.Initialize(context.Background(), &bootstrappb.Workspace_Initialize_Req{
		AdminUsername:  "admin",
		Password:       "secret",
		IdempotencyKey: "idem-e2e-1",
	})
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	if !initResp.GetAccepted() {
		t.Fatal("accepted = false, want true")
	}
	if initResp.GetOperationId() == "" {
		t.Fatal("operation_id is empty")
	}

	statusResp, err := srv.GetInitializationStatus(context.Background(), &bootstrappb.Workspace_GetInitializationStatus_Req{
		OperationId: initResp.GetOperationId(),
	})
	if err != nil {
		t.Fatalf("GetInitializationStatus() error = %v", err)
	}
	if statusResp.GetOperationId() != initResp.GetOperationId() {
		t.Fatalf("operation_id = %q, want %q", statusResp.GetOperationId(), initResp.GetOperationId())
	}
	if statusResp.GetState() != bootstrappb.InitializationState_INITIALIZATION_STATE_SUCCEEDED {
		t.Fatalf("state = %v, want SUCCEEDED", statusResp.GetState())
	}
	if statusResp.GetStage() != bootstrappb.InitializationStage_INITIALIZATION_STAGE_DONE {
		t.Fatalf("stage = %v, want DONE", statusResp.GetStage())
	}
	if !statusResp.GetReadyForLogin() {
		t.Fatal("ready_for_login = false, want true")
	}
	if statusResp.GetRedirectUrl() != bootstrapDefaultRedirectTo {
		t.Fatalf("redirect_url = %q, want %q", statusResp.GetRedirectUrl(), bootstrapDefaultRedirectTo)
	}

	redirectURL, err := url.Parse(statusResp.GetRedirectUrl())
	if err != nil {
		t.Fatalf("parse redirect_url error = %v", err)
	}
	if redirectURL.IsAbs() {
		t.Fatalf("redirect_url should be relative path, got absolute %q", statusResp.GetRedirectUrl())
	}
	if redirectURL.Path != "/web/login" {
		t.Fatalf("redirect path = %q, want /web/login", redirectURL.Path)
	}
	if switchCalls != 1 {
		t.Fatalf("switch mode call count = %d, want 1", switchCalls)
	}
	if !released {
		t.Fatal("expected lease release to be called")
	}
}
