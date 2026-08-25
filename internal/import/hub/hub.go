// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package hub

import (
	"context"
	"strings"

	"github.com/choysum-dev/choysum/internal/import/adapter/csv"
	"github.com/choysum-dev/choysum/internal/import/proto/importpb"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
	"github.com/choysum-dev/choysum/pkg/jsexecutor"
	"github.com/choysum-dev/choysum/pkg/scope"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Deps wires ImportHub runtime dependencies.
type Deps struct {
	RuntimeScope scope.Scope
	JSExecutor   jsexecutor.JsExecutor
	SourceReader SourceReader
	Run          func(context.Context, scope.Scope, importpkg.Spec) (importpkg.Report, error)
}

// Hub implements import.ImportHub gRPC methods.
type Hub struct {
	importpb.UnimplementedImportHubServer
	deps Deps
}

// New builds an ImportHub with the given dependencies.
func New(deps Deps) *Hub {
	return &Hub{deps: deps}
}

// ParseHeaders reads CSV headers from a document source reference.
func (h *Hub) ParseHeaders(ctx context.Context, req *importpb.ParseHeadersRequest) (*importpb.ParseHeadersResponse, error) {
	if err := ensureIdentity(ctx); err != nil {
		return nil, err
	}
	sourceRef := strings.TrimSpace(req.GetSourceRef())
	if sourceRef == "" {
		return nil, status.Error(codes.InvalidArgument, "source_ref is required")
	}

	reader := h.deps.SourceReader
	if reader == nil {
		reader = DocumentSourceReader{RuntimeScope: h.deps.RuntimeScope}
	}
	raw, err := reader.Read(ctx, sourceRef)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "read source: %v", err)
	}

	table, err := csv.ReadTable(raw)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "parse CSV headers: %v", err)
	}

	return &importpb.ParseHeadersResponse{
		Headers:   append([]string(nil), table.Headers...),
		Delimiter: ",",
		HeaderRow: 1,
	}, nil
}

// Preview runs a dry-run import with atomic policy.
func (h *Hub) Preview(ctx context.Context, req *importpb.ImportRunRequest) (*importpb.ImportRunResponse, error) {
	return runImport(ctx, h.deps, req, true)
}

// Run executes a committed atomic import.
func (h *Hub) Run(ctx context.Context, req *importpb.ImportRunRequest) (*importpb.ImportRunResponse, error) {
	return runImport(ctx, h.deps, req, false)
}

// RunAsync enqueues a background record import via task.ImportJob + task.Job.
func (h *Hub) RunAsync(ctx context.Context, req *importpb.ImportRunAsyncRequest) (*importpb.ImportRunAsyncResponse, error) {
	return runImportAsync(ctx, h.deps, req)
}
