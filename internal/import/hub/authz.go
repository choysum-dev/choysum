// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package hub

import (
	"context"
	"fmt"
	"strings"

	importcaller "github.com/choysum-dev/choysum/internal/import/caller"
	importproto "github.com/choysum-dev/choysum/internal/import/proto"
	"github.com/choysum-dev/choysum/internal/import/proto/importpb"
	"github.com/choysum-dev/choysum/internal/import/runner"
	"github.com/choysum-dev/choysum/pkg/auth"
	"github.com/choysum-dev/choysum/pkg/auth/grpcclient"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsexecutor"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func checkModelImportAccess(ctx context.Context, runtimeScope scope.Scope, targetModel, companyID string) error {
	targetModel = strings.TrimSpace(targetModel)
	if targetModel == "" {
		return status.Error(codes.InvalidArgument, "target_model is required")
	}

	companyID = strings.TrimSpace(companyID)
	if companyFieldRequired(runtimeScope, targetModel) && companyID == "" {
		return status.Error(codes.InvalidArgument, "company_id is required for company-scoped models")
	}

	for _, method := range []string{"Create", "Write"} {
		serviceName := fmt.Sprintf("%s.%s", targetModel, method)
		allowed, err := grpcclient.CheckMethodAccess(ctx, companyID, serviceName)
		if err != nil {
			return status.Errorf(codes.PermissionDenied, "import access check failed: %v", err)
		}
		if !allowed {
			return status.Error(codes.PermissionDenied, "import access denied")
		}
	}
	return nil
}

func companyFieldRequired(runtimeScope scope.Scope, targetModel string) bool {
	if runtimeScope == nil {
		return false
	}
	session, ok := scope.SessionForScope(context.Background(), runtimeScope)
	if !ok || session == nil || session.DB == nil {
		return false
	}
	app, name, err := splitModelFullName(targetModel)
	if err != nil {
		return false
	}
	model := &meta.Model{}
	if err := session.DB.Where("application = ? AND name = ?", app, name).First(model).Error; err != nil {
		return false
	}
	return model.CompanyField != nil && strings.TrimSpace(*model.CompanyField) != ""
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

func reportResponse(report importpkg.Report) *importpb.ImportRunResponse {
	return &importpb.ImportRunResponse{Report: importproto.ReportToProto(report)}
}

func runImport(
	ctx context.Context,
	deps Deps,
	req *importpb.ImportRunRequest,
	dryRun bool,
) (*importpb.ImportRunResponse, error) {
	if err := ensureIdentity(ctx); err != nil {
		return nil, err
	}
	if deps.RuntimeScope == nil {
		return nil, status.Error(codes.Unavailable, "runtime scope unavailable")
	}
	if deps.Run == nil && deps.JSExecutor == nil {
		return nil, status.Error(codes.Unavailable, "js executor unavailable")
	}

	spec, err := toRecordSpec(req, dryRun)
	if err != nil {
		return nil, err
	}
	companyID := strings.TrimSpace(req.GetCompanyId())
	if companyID == "" {
		companyID = activeCompanyID(ctx)
		spec.Options.CompanyID = companyID
	}
	if err := checkModelImportAccess(ctx, deps.RuntimeScope, spec.Model, companyID); err != nil {
		return nil, err
	}
	if err := importpkg.ValidateSpec(spec); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid import spec: %v", err)
	}

	reader := deps.SourceReader
	if reader == nil {
		reader = DocumentSourceReader{RuntimeScope: deps.RuntimeScope}
	}
	runCtx := ContextWithSourceReader(ctx, reader)
	if deps.JSExecutor != nil {
		runCtx = importcaller.ContextWithCaller(runCtx, importcaller.ExecutorCaller{Engine: jsExecutorAdapter{inner: deps.JSExecutor}})
	}

	runFn := deps.Run
	if runFn == nil {
		runFn = runner.Run
	}
	report, err := runFn(runCtx, deps.RuntimeScope, spec)
	if err != nil && len(report.Messages) == 0 {
		return nil, status.Errorf(codes.Internal, "import run failed: %v", err)
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
