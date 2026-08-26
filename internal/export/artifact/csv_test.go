// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package artifact_test

import (
	"context"
	"testing"

	exportartifact "github.com/choysum-dev/choysum/internal/export/artifact"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
)

func TestAttachReportCSV_skipsWithoutCompany(t *testing.T) {
	report := importpkg.Report{}
	if err := exportartifact.AttachReportCSV(context.Background(), nil, "", []byte("a,b\n1,2\n"), &report); err != nil {
		t.Fatalf("AttachReportCSV: %v", err)
	}
	if report.ArtifactRef != "" {
		t.Fatalf("ArtifactRef = %q, want empty", report.ArtifactRef)
	}
}

func TestAttachReportCSV_skipsEmptyBytes(t *testing.T) {
	report := importpkg.Report{}
	if err := exportartifact.AttachReportCSV(context.Background(), nil, "co-1", nil, &report); err != nil {
		t.Fatalf("AttachReportCSV: %v", err)
	}
}
