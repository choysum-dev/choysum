// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package hub

import (
	"strings"

	"github.com/choysum-dev/choysum/internal/export/proto/exportpb"
	exportpkg "github.com/choysum-dev/choysum/pkg/export"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func toRecordSpec(req *exportpb.ExportRunRequest, preview bool) (exportpkg.Spec, error) {
	if req == nil {
		return exportpkg.Spec{}, status.Error(codes.InvalidArgument, "request is required")
	}

	model := strings.TrimSpace(req.GetModel())
	if model == "" {
		return exportpkg.Spec{}, status.Error(codes.InvalidArgument, "model is required")
	}

	mode, err := modeFromProto(req.GetMode())
	if err != nil {
		return exportpkg.Spec{}, err
	}

	spec := exportpkg.Spec{
		Profile: exportpkg.ProfileRecord,
		Caller:  exportpkg.CallerUser,
		Mode:    mode,
		Format:  "csv",
		Model:   model,
		Fields:  append([]string(nil), req.GetFields()...),
		Domain:  strings.TrimSpace(req.GetDomain()),
		Ids:     append([]string(nil), req.GetIds()...),
		Options: exportpkg.Options{
			CompanyID: strings.TrimSpace(req.GetCompanyId()),
		},
	}
	if preview && mode != exportpkg.ModeTemplate {
		spec.Limit = previewRowLimit
	}
	return spec, nil
}

func modeFromProto(mode exportpb.ExportMode) (exportpkg.Mode, error) {
	switch mode {
	case exportpb.ExportMode_EXPORT_MODE_UNSPECIFIED, exportpb.ExportMode_EXPORT_MODE_DATA:
		return exportpkg.ModeData, nil
	case exportpb.ExportMode_EXPORT_MODE_TEMPLATE:
		return exportpkg.ModeTemplate, nil
	default:
		return "", status.Error(codes.InvalidArgument, "unsupported export mode")
	}
}
