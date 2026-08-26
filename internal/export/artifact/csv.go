// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package artifact

import (
	"context"
	"strings"

	importartifact "github.com/choysum-dev/choysum/internal/import/artifact"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
	"github.com/choysum-dev/choysum/pkg/scope"
)

const exportCSVMimeType = "text/csv; charset=utf-8"

// AttachReportCSV stores export CSV bytes and sets report.ArtifactRef when company id is present.
func AttachReportCSV(ctx context.Context, runtimeScope scope.Scope, companyID string, csvBytes []byte, report *importpkg.Report) error {
	if report == nil || len(csvBytes) == 0 {
		return nil
	}
	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return nil
	}
	ref, err := storeContentArtifact(ctx, runtimeScope, companyID, csvBytes, exportCSVMimeType)
	if err != nil {
		return err
	}
	report.ArtifactRef = ref
	return nil
}

var storeContentArtifact = importartifact.StoreContentArtifact
