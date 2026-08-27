// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package hub

import (
	"context"
	"encoding/json"
	"strings"

	exportproto "github.com/choysum-dev/choysum/internal/export/proto"
	"github.com/choysum-dev/choysum/internal/export/proto/exportpb"
	"github.com/choysum-dev/choysum/pkg/auth"
	exportpkg "github.com/choysum-dev/choysum/pkg/export"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const enqueueExportDataTransferJobService = "task.DataTransferJob.EnqueueRecordExport"

var (
	exportDataTransferJobRecordFromSpec = exportpkg.DataTransferJobRecordFromSpec
	exportJSONUnmarshal                 = json.Unmarshal
)

type enqueueExportDataTransferJobInput struct {
	TargetModel  string         `json:"targetModel"`
	SourceRef    string         `json:"sourceRef"`
	CompanyID    string         `json:"companyId,omitempty"`
	Profile      string         `json:"profile"`
	SpecSnapshot map[string]any `json:"specSnapshot"`
}

type enqueueExportDataTransferJobResult struct {
	DataTransferJobID string `json:"dataTransferJobId"`
	TaskJobID         string `json:"taskJobId"`
}

func runExportAsync(
	ctx context.Context,
	deps Deps,
	req *exportpb.ExportRunAsyncRequest,
) (*exportpb.ExportRunAsyncResponse, error) {
	if err := ensureIdentity(ctx); err != nil {
		return nil, err
	}
	if deps.RuntimeScope == nil {
		return nil, status.Error(codes.Unavailable, "runtime scope unavailable")
	}
	if deps.JSExecutor == nil {
		return nil, status.Error(codes.Unavailable, "js executor unavailable")
	}
	if req == nil || req.GetRun() == nil {
		return nil, status.Error(codes.InvalidArgument, "run request is required")
	}

	runReq := req.GetRun()
	spec, err := toAsyncRecordSpec(runReq)
	if err != nil {
		return nil, err
	}
	companyID := strings.TrimSpace(runReq.GetCompanyId())
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

	var specSnapshot map[string]any
	jobRecord, err := exportDataTransferJobRecordFromSpec(spec)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "marshal spec snapshot: %v", err)
	}
	if err := exportJSONUnmarshal(jobRecord.SpecSnapshotJSON, &specSnapshot); err != nil {
		return nil, status.Errorf(codes.Internal, "decode spec snapshot: %v", err)
	}

	userID := exportSchedulerUserID(ctx)

	enqueueInput := enqueueExportDataTransferJobInput{
		TargetModel:  spec.Model,
		SourceRef:    jobRecord.SourceRef,
		CompanyID:    companyID,
		Profile:      string(spec.Profile),
		SpecSnapshot: specSnapshot,
	}
	result, err := enqueueExportDataTransferJob(ctx, deps, enqueueInput, userID)
	if err != nil {
		return nil, err
	}

	emptyReport := importpkg.Report{
		Profile: importpkg.ProfileRecord,
		DryRun:  false,
		Meta: &importpkg.ReportMeta{
			TargetModel: spec.Model,
		},
	}
	return &exportpb.ExportRunAsyncResponse{
		DataTransferJobId: result.DataTransferJobID,
		TaskJobId:         result.TaskJobID,
		Report:            exportproto.ReportToProto(emptyReport),
	}, nil
}

func enqueueExportDataTransferJob(ctx context.Context, deps Deps, input enqueueExportDataTransferJobInput, userID string) (enqueueExportDataTransferJobResult, error) {
	resp, err := deps.JSExecutor.Execute(ctx, &jsengine.JsRequest{
		Id:      "export-run-async",
		Service: enqueueExportDataTransferJobService,
		Args:    []any{input},
		Context: map[string]any{
			"userId": userID,
		},
	})
	if err != nil {
		return enqueueExportDataTransferJobResult{}, status.Errorf(codes.Internal, "enqueue data transfer job: %v", err)
	}
	if resp == nil || resp.Result == nil {
		return enqueueExportDataTransferJobResult{}, status.Error(codes.Internal, "enqueue data transfer job returned empty result")
	}
	raw, err := json.Marshal(resp.Result)
	if err != nil {
		return enqueueExportDataTransferJobResult{}, status.Errorf(codes.Internal, "marshal enqueue result: %v", err)
	}
	var result enqueueExportDataTransferJobResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return enqueueExportDataTransferJobResult{}, status.Errorf(codes.Internal, "decode enqueue result: %v", err)
	}
	if strings.TrimSpace(result.DataTransferJobID) == "" || strings.TrimSpace(result.TaskJobID) == "" {
		return enqueueExportDataTransferJobResult{}, status.Error(codes.Internal, "enqueue data transfer job returned incomplete ids")
	}
	return result, nil
}

func toAsyncRecordSpec(req *exportpb.ExportRunRequest) (exportpkg.Spec, error) {
	profile := strings.TrimSpace(req.GetProfile())
	if profile == string(exportpkg.ProfileTerminology) {
		return exportpkg.Spec{}, status.Error(codes.InvalidArgument, "async export is only supported for record profile")
	}
	spec, err := toRecordSpec(req, false)
	if err != nil {
		return exportpkg.Spec{}, err
	}
	spec.Async = true
	return spec, nil
}

func exportSchedulerUserID(ctx context.Context) string {
	identity := auth.IdentityFromContext(ctx)
	if identity == nil {
		return ""
	}
	return strings.TrimSpace(identity.GetUserID())
}
