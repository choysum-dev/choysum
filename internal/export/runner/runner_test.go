// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package runner_test

import (
	"context"
	"errors"
	"testing"

	stubreader "github.com/choysum-dev/choysum/internal/export/reader/stub"
	"github.com/choysum-dev/choysum/internal/export/registry"
	_ "github.com/choysum-dev/choysum/internal/export/runner"
	exportpkg "github.com/choysum-dev/choysum/pkg/export"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
)

func TestRun_StubReader_Ok(t *testing.T) {
	spec := exportpkg.Spec{
		Profile: exportpkg.ProfileRecord,
		Caller:  exportpkg.CallerUser,
		Model:   "base.Country",
		Format:  "csv",
		Options: exportpkg.Options{StubUnitCount: 3},
	}
	report, err := exportpkg.Run(context.Background(), nil, spec)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if report.Profile != importpkg.ProfileRecord {
		t.Fatalf("profile = %q", report.Profile)
	}
	if report.Stats.Total != 3 || report.Stats.Ok != 3 || report.Stats.Error != 0 {
		t.Fatalf("stats = %+v", report.Stats)
	}
	if len(report.Messages) != 0 {
		t.Fatalf("messages = %+v", report.Messages)
	}
}

func TestRun_StubReader_MessageOnError(t *testing.T) {
	spec := exportpkg.Spec{
		Profile: exportpkg.ProfileRecord,
		Caller:  exportpkg.CallerUser,
		Model:   "base.Country",
		Format:  "csv",
		Options: exportpkg.Options{StubUnitCount: 2, StubFailUnitIndex: 2},
	}
	report, err := exportpkg.Run(context.Background(), nil, spec)
	if err == nil {
		t.Fatal("expected stub failure")
	}
	var expErr *exportpkg.Error
	if !errors.As(err, &expErr) {
		t.Fatalf("error = %v, want *exportpkg.Error", err)
	}
	if len(report.Messages) == 0 {
		t.Fatal("expected report messages on error")
	}
	if report.Messages[0].Type != importpkg.MessageError || report.Messages[0].Row != 2 {
		t.Fatalf("message = %+v", report.Messages[0])
	}
	if report.Stats.Error < 1 {
		t.Fatalf("stats = %+v", report.Stats)
	}
}

func TestRun_ReaderNotRegistered(t *testing.T) {
	registry.ResetForTest()
	t.Cleanup(func() {
		registry.Register(exportpkg.ProfileRecord, stubreader.Reader{})
		registry.Register(exportpkg.ProfileTerminology, stubreader.Reader{})
	})

	_, err := exportpkg.Run(context.Background(), nil, exportpkg.Spec{
		Profile: exportpkg.ProfileRecord,
		Caller:  exportpkg.CallerUser,
		Model:   "base.Country",
		Format:  "csv",
	})
	if !errors.Is(err, exportpkg.ErrReaderNotRegistered) {
		t.Fatalf("error = %v, want ErrReaderNotRegistered", err)
	}
}
