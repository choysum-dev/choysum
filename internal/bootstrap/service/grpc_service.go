// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package service

import (
	"context"
	"strings"

	bootstrappb "github.com/choysum-dev/choysum/internal/bootstrap/proto/bootstrappb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type workspaceCoordinator interface {
	startInitialization(ctx context.Context, input initializeInput) (operationSnapshot, bool, error)
	getInitializationStatus(operationID string) (operationSnapshot, bool)
}

func (s *workspaceServer) Initialize(ctx context.Context, req *bootstrappb.Workspace_Initialize_Req) (*bootstrappb.Workspace_Initialize_Resp, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	op, _, err := s.coordinator.startInitialization(ctx, initializeInput{
		AdminUsername:        req.GetAdminUsername(),
		Password:             req.GetPassword(),
		ClientHashingEnabled: req.GetClientHashingEnabled(),
		IdempotencyKey:       strings.TrimSpace(req.GetIdempotencyKey()),
	})
	if err != nil {
		grpcErr := toGRPCError(err)
		if s.runtimeScope != nil && s.runtimeScope.Logger() != nil {
			s.runtimeScope.Logger().Warn(
				"Bootstrap Initialize rejected",
				"error_code", bootstrapErrorCode(err),
				"grpc_code", status.Code(grpcErr).String(),
				"error", err.Error(),
			)
		}
		return nil, grpcErr
	}

	return toInitializeResponse(op), nil
}

func (s *workspaceServer) GetInitializationStatus(ctx context.Context, req *bootstrappb.Workspace_GetInitializationStatus_Req) (*bootstrappb.Workspace_GetInitializationStatus_Resp, error) {
	_ = ctx
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	operationID := strings.TrimSpace(req.GetOperationId())
	if operationID == "" {
		return nil, status.Error(codes.InvalidArgument, "operation_id is required")
	}

	op, ok := s.coordinator.getInitializationStatus(operationID)
	if !ok {
		if s.runtimeScope != nil && s.runtimeScope.Logger() != nil {
			s.runtimeScope.Logger().Warn(
				"Bootstrap status query missing operation",
				"operation_id", operationID,
				"grpc_code", codes.NotFound.String(),
			)
		}
		return nil, status.Error(codes.NotFound, "initialization operation not found")
	}

	return toStatusResponse(op), nil
}
