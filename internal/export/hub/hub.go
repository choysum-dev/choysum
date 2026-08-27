// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package hub

import (
	"context"
	"strings"

	exportproto "github.com/choysum-dev/choysum/internal/export/proto"
	"github.com/choysum-dev/choysum/internal/export/proto/exportpb"
	exportpkg "github.com/choysum-dev/choysum/pkg/export"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
	"github.com/choysum-dev/choysum/pkg/jsexecutor"
	"github.com/choysum-dev/choysum/pkg/scope"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const previewRowLimit = 5

// Deps wires ExportHub runtime dependencies.
type Deps struct {
	RuntimeScope scope.Scope
	JSExecutor   jsexecutor.JsExecutor
	Run          func(context.Context, scope.Scope, exportpkg.Spec) (importpkg.Report, error)
}

// Hub implements export.ExportHub gRPC methods.
type Hub struct {
	exportpb.UnimplementedExportHubServer
	deps Deps
}

// New builds an ExportHub with the given dependencies.
func New(deps Deps) *Hub {
	return &Hub{deps: deps}
}

// DescribeFields returns exportable field metadata for a record model.
func (h *Hub) DescribeFields(ctx context.Context, req *exportpb.DescribeFieldsRequest) (*exportpb.DescribeFieldsResponse, error) {
	if err := ensureIdentity(ctx); err != nil {
		return nil, err
	}
	if h.deps.RuntimeScope == nil {
		return nil, status.Error(codes.Unavailable, "runtime scope unavailable")
	}
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	modelName := strings.TrimSpace(req.GetModel())
	if modelName == "" {
		return nil, status.Error(codes.InvalidArgument, "model is required")
	}
	companyID := activeCompanyID(ctx)
	if err := checkModelExportAccess(ctx, h.deps.RuntimeScope, modelName, companyID); err != nil {
		return nil, err
	}
	return describeFields(ctx, h.deps.RuntimeScope, req)
}

// Preview runs a small sample export or header-only template preview.
func (h *Hub) Preview(ctx context.Context, req *exportpb.ExportRunRequest) (*exportpb.ExportRunResponse, error) {
	return runExport(ctx, h.deps, req, true)
}

// Run executes a synchronous record export.
func (h *Hub) Run(ctx context.Context, req *exportpb.ExportRunRequest) (*exportpb.ExportRunResponse, error) {
	return runExport(ctx, h.deps, req, false)
}

// RunAsync is not implemented until async export lands.
func (h *Hub) RunAsync(ctx context.Context, _ *exportpb.ExportRunAsyncRequest) (*exportpb.ExportRunAsyncResponse, error) {
	if err := ensureIdentity(ctx); err != nil {
		return nil, err
	}
	return nil, status.Error(codes.Unimplemented, "async export is not supported yet")
}

func reportResponse(report importpkg.Report) *exportpb.ExportRunResponse {
	return &exportpb.ExportRunResponse{Report: exportproto.ReportToProto(report)}
}
