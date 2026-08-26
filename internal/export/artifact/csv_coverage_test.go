// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package artifact

import (
	"context"
	"errors"
	"testing"

	importpkg "github.com/choysum-dev/choysum/pkg/import"
	"github.com/choysum-dev/choysum/pkg/scope"
)

func TestAttachReportCSV_success(t *testing.T) {
	orig := storeContentArtifact
	t.Cleanup(func() { storeContentArtifact = orig })
	storeContentArtifact = func(context.Context, scope.Scope, string, []byte, string) (string, error) {
		return "content-1", nil
	}

	report := importpkg.Report{}
	if err := AttachReportCSV(context.Background(), nil, "co-1", []byte("a,b\n1,2\n"), &report); err != nil {
		t.Fatalf("AttachReportCSV: %v", err)
	}
	if report.ArtifactRef != "content-1" {
		t.Fatalf("ArtifactRef = %q", report.ArtifactRef)
	}
}

func TestAttachReportCSV_storeError(t *testing.T) {
	orig := storeContentArtifact
	t.Cleanup(func() { storeContentArtifact = orig })
	storeContentArtifact = func(context.Context, scope.Scope, string, []byte, string) (string, error) {
		return "", errors.New("store failed")
	}

	report := importpkg.Report{}
	err := AttachReportCSV(context.Background(), nil, "co-1", []byte("x"), &report)
	if err == nil {
		t.Fatal("expected store error")
	}
}
