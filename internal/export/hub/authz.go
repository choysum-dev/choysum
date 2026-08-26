// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package hub

import (
	"context"
	"fmt"
	"strings"

	"github.com/choysum-dev/choysum/internal/export/proto/exportpb"
	"github.com/choysum-dev/choysum/internal/export/registry"
	"github.com/choysum-dev/choysum/internal/export/runner"
	importcaller "github.com/choysum-dev/choysum/internal/import/caller"
	"github.com/choysum-dev/choysum/pkg/auth"
	"github.com/choysum-dev/choysum/pkg/auth/grpcclient"
	exportpkg "github.com/choysum-dev/choysum/pkg/export"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsexecutor"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const maxInlineCSVBytes = 16 * 1024 * 1024

func checkModelExportAccess(ctx context.Context, runtimeScope scope.Scope, targetModel, companyID string) error {
	targetModel = strings.TrimSpace(targetModel)
	if targetModel == "" {
		return status.Error(codes.InvalidArgument, "model is required")
	}

	companyID = strings.TrimSpace(companyID)
	required, lookupErr := modelCompanyFieldRequired(ctx, runtimeScope, targetModel)
	if lookupErr != nil {
		return status.Errorf(codes.InvalidArgument, "model lookup failed: %v", lookupErr)
	}
	if required && companyID == "" {
		return status.Error(codes.InvalidArgument, "company_id is required for company-scoped models")
	}

	serviceName := fmt.Sprintf("%s.Search", targetModel)
	allowed, err := grpcclient.CheckMethodAccess(ctx, companyID, serviceName)
	if err != nil {
		return status.Errorf(codes.PermissionDenied, "export access check failed: %v", err)
	}
	if !allowed {
		return status.Error(codes.PermissionDenied, "export access denied")
	}
	return nil
}

func modelCompanyFieldRequired(ctx context.Context, runtimeScope scope.Scope, targetModel string) (bool, error) {
	if runtimeScope == nil {
		return false, nil
	}
	session, ok := scope.SessionForScope(ctx, runtimeScope)
	if !ok || session == nil || session.DB == nil {
		return false, nil
	}
	app, name, err := splitModelFullName(targetModel)
	if err != nil {
		return false, err
	}
	model := &meta.Model{}
	if err := session.DB.Where("application = ? AND name = ?", app, name).First(model).Error; err != nil {
		return false, err
	}
	return model.CompanyField != nil && strings.TrimSpace(*model.CompanyField) != "", nil
}

func splitModelFullName(full string) (application, name string, err error) {
	full = strings.TrimSpace(full)
	dot := strings.Index(full, ".")
	if dot <= 0 || dot == len(full)-1 {
		return "", "", fmt.Errorf("invalid model")
	}
	return strings.TrimSpace(full[:dot]), strings.TrimSpace(full[dot+1:]), nil
}

func ensureIdentity(ctx context.Context) error {
	identity := auth.IdentityFromContext(ctx)
	if identity == nil || !identity.IsValid() || strings.TrimSpace(identity.GetUserID()) == "" {
		return status.Error(codes.Unauthenticated, "unauthenticated")
	}
	return nil
}

func activeCompanyID(ctx context.Context) string {
	identity := auth.IdentityFromContext(ctx)
	if identity == nil {
		return ""
	}
	metadata := identity.GetMetadata()
	if metadata == nil {
		return ""
	}
	if value, ok := metadata["activeCompanyId"]; ok {
		if text, ok := value.(string); ok {
			return strings.TrimSpace(text)
		}
	}
	if value, ok := metadata["companyId"]; ok {
		if text, ok := value.(string); ok {
			return strings.TrimSpace(text)
		}
	}
	return ""
}

func runExport(
	ctx context.Context,
	deps Deps,
	req *exportpb.ExportRunRequest,
	preview bool,
) (*exportpb.ExportRunResponse, error) {
	if err := ensureIdentity(ctx); err != nil {
		return nil, err
	}
	if deps.RuntimeScope == nil {
		return nil, status.Error(codes.Unavailable, "runtime scope unavailable")
	}
	if deps.Run == nil && deps.JSExecutor == nil {
		return nil, status.Error(codes.Unavailable, "js executor unavailable")
	}

	spec, err := toRecordSpec(req, preview)
	if err != nil {
		return nil, err
	}
	companyID := strings.TrimSpace(req.GetCompanyId())
	if companyID == "" {
		companyID = activeCompanyID(ctx)
		spec.Options.CompanyID = companyID
	}
	if err := checkModelExportAccess(ctx, deps.RuntimeScope, spec.Model, companyID); err != nil {
		return nil, err
	}
	if err := validateExportSpec(spec); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid export spec: %v", err)
	}

	runCtx := ctx
	if deps.JSExecutor != nil {
		runCtx = importcaller.ContextWithCaller(runCtx, importcaller.ExecutorCaller{Engine: jsExecutorAdapter{inner: deps.JSExecutor}})
	}

	runFn := deps.Run
	if runFn == nil {
		report, result, err := runExportWithResultFn(runCtx, deps.RuntimeScope, spec)
		if err != nil && len(report.Messages) == 0 {
			return nil, status.Errorf(codes.Internal, "export run failed: %v", err)
		}
		resp := reportResponse(report)
		attachInlineCSV(resp, result.CSVBytes)
		return resp, nil
	}
	report, err := runFn(runCtx, deps.RuntimeScope, spec)
	if err != nil && len(report.Messages) == 0 {
		return nil, status.Errorf(codes.Internal, "export run failed: %v", err)
	}
	return reportResponse(report), nil
}

type jsExecutorAdapter struct {
	inner jsexecutor.JsExecutor
}

func (a jsExecutorAdapter) Load(_ []*jsengine.JsScript) error { return nil }

func (a jsExecutorAdapter) Execute(ctx context.Context, req *jsengine.JsRequest) (*jsengine.JsResponse, error) {
	return a.inner.Execute(ctx, req)
}

func (a jsExecutorAdapter) Close() error { return nil }

func runExportWithResult(ctx context.Context, runtimeScope scope.Scope, spec exportpkg.Spec) (importpkg.Report, registry.Result, error) {
	return runner.RunWithResult(ctx, runtimeScope, spec)
}

// runExportWithResultFn is swappable in tests to cover the JSExecutor-only run path.
var runExportWithResultFn = runExportWithResult

// validateExportSpec is swappable in tests to cover defensive validation failures.
var validateExportSpec = exportpkg.ValidateSpec

func attachInlineCSV(resp *exportpb.ExportRunResponse, csvBytes []byte) {
	if resp == nil || len(csvBytes) == 0 {
		return
	}
	if len(csvBytes) <= maxInlineCSVBytes {
		resp.CsvData = append([]byte(nil), csvBytes...)
	}
}
