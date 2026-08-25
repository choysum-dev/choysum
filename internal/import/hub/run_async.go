// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package hub

import (
	"context"
	"encoding/json"
	"strings"

	importproto "github.com/choysum-dev/choysum/internal/import/proto"
	"github.com/choysum-dev/choysum/internal/import/proto/importpb"
	"github.com/choysum-dev/choysum/pkg/auth"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const enqueueImportJobService = "task.ImportJob.EnqueueRecordImport"

var (
	jobRecordFromSpec = importpkg.JobRecordFromSpec
	jsonUnmarshal     = json.Unmarshal
)

type enqueueImportJobInput struct {
	TargetModel  string         `json:"targetModel"`
	SourceRef    string         `json:"sourceRef"`
	CompanyID    string         `json:"companyId,omitempty"`
	Policy       string         `json:"policy"`
	Profile      string         `json:"profile"`
	SpecSnapshot map[string]any `json:"specSnapshot"`
}

type enqueueImportJobResult struct {
	ImportJobID string `json:"importJobId"`
	TaskJobID   string `json:"taskJobId"`
}

func runImportAsync(
	ctx context.Context,
	deps Deps,
	req *importpb.ImportRunAsyncRequest,
) (*importpb.ImportRunAsyncResponse, error) {
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
	if err := checkModelImportAccess(ctx, deps.RuntimeScope, spec.Model, companyID); err != nil {
		return nil, err
	}
	if err := validateImportSpec(spec); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid import spec: %v", err)
	}

	var specSnapshot map[string]any
	jobRecord, err := jobRecordFromSpec(spec)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "marshal spec snapshot: %v", err)
	}
	if err := jsonUnmarshal(jobRecord.SpecSnapshotJSON, &specSnapshot); err != nil {
		return nil, status.Errorf(codes.Internal, "decode spec snapshot: %v", err)
	}

	userID := schedulerUserID(ctx)

	enqueueInput := enqueueImportJobInput{
		TargetModel:  spec.Model,
		SourceRef:    strings.TrimSpace(spec.Source.DocumentRef),
		CompanyID:    companyID,
		Policy:       string(importpkg.EffectivePolicy(spec)),
		Profile:      string(spec.Profile),
		SpecSnapshot: specSnapshot,
	}
	result, err := enqueueImportJob(ctx, deps, enqueueInput, userID)
	if err != nil {
		return nil, err
	}

	emptyReport := importpkg.Report{
		Profile: spec.Profile,
		Policy:  importpkg.EffectivePolicy(spec),
		DryRun:  false,
		Meta: &importpkg.ReportMeta{
			TargetModel: spec.Model,
			SourceRef:   spec.Source.DocumentRef,
		},
	}
	return &importpb.ImportRunAsyncResponse{
		ImportJobId: result.ImportJobID,
		TaskJobId:   result.TaskJobID,
		Report:      importproto.ReportToProto(emptyReport),
	}, nil
}

func enqueueImportJob(ctx context.Context, deps Deps, input enqueueImportJobInput, userID string) (enqueueImportJobResult, error) {
	resp, err := deps.JSExecutor.Execute(ctx, &jsengine.JsRequest{
		Id:      "import-run-async",
		Service: enqueueImportJobService,
		Args:    []any{input},
		Context: map[string]any{
			"userId": userID,
		},
	})
	if err != nil {
		return enqueueImportJobResult{}, status.Errorf(codes.Internal, "enqueue import job: %v", err)
	}
	if resp == nil || resp.Result == nil {
		return enqueueImportJobResult{}, status.Error(codes.Internal, "enqueue import job returned empty result")
	}
	raw, err := json.Marshal(resp.Result)
	if err != nil {
		return enqueueImportJobResult{}, status.Errorf(codes.Internal, "marshal enqueue result: %v", err)
	}
	var result enqueueImportJobResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return enqueueImportJobResult{}, status.Errorf(codes.Internal, "decode enqueue result: %v", err)
	}
	if strings.TrimSpace(result.ImportJobID) == "" || strings.TrimSpace(result.TaskJobID) == "" {
		return enqueueImportJobResult{}, status.Error(codes.Internal, "enqueue import job returned incomplete ids")
	}
	return result, nil
}

func toAsyncRecordSpec(req *importpb.ImportRunRequest) (importpkg.Spec, error) {
	if req == nil {
		return importpkg.Spec{}, status.Error(codes.InvalidArgument, "request is required")
	}
	targetModel := strings.TrimSpace(req.GetTargetModel())
	sourceRef := strings.TrimSpace(req.GetSourceRef())
	if targetModel == "" || sourceRef == "" {
		return importpkg.Spec{}, status.Error(codes.InvalidArgument, "target_model and source_ref are required")
	}
	policy, err := asyncPolicyFromProto(req.GetPolicy())
	if err != nil {
		return importpkg.Spec{}, err
	}
	columnMapping := req.GetColumnMapping()
	if columnMapping == nil {
		columnMapping = map[string]string{}
	}
	return importpkg.Spec{
		Profile: importpkg.ProfileRecord,
		Caller:  importpkg.CallerUser,
		Policy:  policy,
		Async:   true,
		Model:   targetModel,
		Source: importpkg.Source{
			Format:      "csv",
			DocumentRef: sourceRef,
		},
		Options: importpkg.Options{
			ColumnMapping: columnMapping,
			CompanyID:     strings.TrimSpace(req.GetCompanyId()),
		},
	}, nil
}

func asyncPolicyFromProto(policy importpb.ImportPolicy) (importpkg.Policy, error) {
	switch policy {
	case importpb.ImportPolicy_IMPORT_POLICY_UNSPECIFIED, importpb.ImportPolicy_IMPORT_POLICY_ATOMIC:
		return importpkg.PolicyAtomic, nil
	case importpb.ImportPolicy_IMPORT_POLICY_STOP_KEEP:
		return importpkg.PolicyStopKeep, nil
	case importpb.ImportPolicy_IMPORT_POLICY_BEST_EFFORT:
		return importpkg.PolicyBestEffort, nil
	default:
		return "", status.Error(codes.InvalidArgument, "unsupported import policy")
	}
}

func schedulerUserID(ctx context.Context) string {
	identity := auth.IdentityFromContext(ctx)
	if identity == nil {
		return ""
	}
	return strings.TrimSpace(identity.GetUserID())
}
