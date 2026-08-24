// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package hub

import (
	"strings"

	"github.com/choysum-dev/choysum/internal/import/proto/importpb"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func toRecordSpec(req *importpb.ImportRunRequest, dryRun bool) (importpkg.Spec, error) {
	if req == nil {
		return importpkg.Spec{}, status.Error(codes.InvalidArgument, "request is required")
	}

	targetModel := strings.TrimSpace(req.GetTargetModel())
	sourceRef := strings.TrimSpace(req.GetSourceRef())
	if targetModel == "" || sourceRef == "" {
		return importpkg.Spec{}, status.Error(codes.InvalidArgument, "target_model and source_ref are required")
	}

	policy, err := policyFromProto(req.GetPolicy())
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
		DryRun:  dryRun,
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

func policyFromProto(policy importpb.ImportPolicy) (importpkg.Policy, error) {
	switch policy {
	case importpb.ImportPolicy_IMPORT_POLICY_UNSPECIFIED, importpb.ImportPolicy_IMPORT_POLICY_ATOMIC:
		return importpkg.PolicyAtomic, nil
	case importpb.ImportPolicy_IMPORT_POLICY_STOP_KEEP, importpb.ImportPolicy_IMPORT_POLICY_BEST_EFFORT:
		return "", status.Error(codes.InvalidArgument, "non-atomic policy is not supported on web import hub")
	default:
		return "", status.Error(codes.InvalidArgument, "unsupported import policy")
	}
}
