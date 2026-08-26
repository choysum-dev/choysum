// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package runner

import (
	"context"
	"errors"
	"testing"

	"github.com/choysum-dev/choysum/internal/export/plan"
	stubreader "github.com/choysum-dev/choysum/internal/export/reader/stub"
	"github.com/choysum-dev/choysum/internal/export/registry"
	exportpkg "github.com/choysum-dev/choysum/pkg/export"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
	"github.com/choysum-dev/choysum/pkg/scope"
)

func restoreStubReaders(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		registry.Register(exportpkg.ProfileRecord, stubreader.Reader{})
		registry.Register(exportpkg.ProfileTerminology, stubreader.Reader{})
	})
}

type fakeReader struct {
	result registry.Result
	err    error
}

func (f fakeReader) Read(context.Context, scope.Scope, plan.Plan) (registry.Result, error) {
	return f.result, f.err
}

func TestRun_planValidationError(t *testing.T) {
	_, err := Run(context.Background(), nil, exportpkg.Spec{})
	if !errors.Is(err, exportpkg.ErrProfileNotApproved) {
		t.Fatalf("Run() error = %v, want ErrProfileNotApproved", err)
	}
}

func TestRun_readerNotRegistered(t *testing.T) {
	registry.ResetForTest()
	restoreStubReaders(t)

	_, err := Run(context.Background(), nil, exportpkg.Spec{
		Profile: exportpkg.ProfileRecord,
		Caller:  exportpkg.CallerUser,
		Model:   "base.Country",
		Format:  "csv",
	})
	if !errors.Is(err, exportpkg.ErrReaderNotRegistered) {
		t.Fatalf("Run() error = %v, want ErrReaderNotRegistered", err)
	}
}

func TestRun_plainReadError(t *testing.T) {
	registry.ResetForTest()
	registry.Register(exportpkg.ProfileRecord, fakeReader{
		result: registry.Result{UnitCount: 2},
		err:    errors.New("read failed"),
	})
	restoreStubReaders(t)

	report, err := Run(context.Background(), nil, exportpkg.Spec{
		Profile: exportpkg.ProfileRecord,
		Caller:  exportpkg.CallerUser,
		Model:   "base.Country",
		Format:  "csv",
	})
	if err == nil {
		t.Fatal("expected read error")
	}
	if len(report.Messages) != 1 || report.Messages[0].Text != "read failed" {
		t.Fatalf("messages = %+v", report.Messages)
	}
	if report.Stats.Error != 1 || report.Stats.Ok != 1 {
		t.Fatalf("stats = %+v", report.Stats)
	}
}

func TestRun_readErrorWithSkipMessages(t *testing.T) {
	registry.ResetForTest()
	registry.Register(exportpkg.ProfileRecord, fakeReader{
		result: registry.Result{
			UnitCount: 3,
			Messages: []registry.Message{
				{Type: "skip", Row: 1, Code: "empty", Text: "skipped"},
				{Type: "error", Row: 2, Code: "constraint", Text: "bad"},
			},
		},
		err: errors.New("partial"),
	})
	restoreStubReaders(t)

	report, err := Run(context.Background(), nil, exportpkg.Spec{
		Profile: exportpkg.ProfileRecord,
		Caller:  exportpkg.CallerUser,
		Model:   "base.Country",
		Format:  "csv",
	})
	if err == nil {
		t.Fatal("expected read error")
	}
	if report.Stats.Error != 1 || report.Stats.Skip != 1 || report.Stats.Ok != 1 {
		t.Fatalf("stats = %+v", report.Stats)
	}
}

func TestRun_okWithMessages(t *testing.T) {
	registry.ResetForTest()
	registry.Register(exportpkg.ProfileRecord, fakeReader{
		result: registry.Result{
			UnitCount: 2,
			Messages: []registry.Message{
				{Type: "", Row: 1, Text: "default error type"},
			},
		},
	})
	restoreStubReaders(t)

	report, err := Run(context.Background(), nil, exportpkg.Spec{
		Profile: exportpkg.ProfileRecord,
		Caller:  exportpkg.CallerUser,
		Model:   "base.Country",
		Format:  "csv",
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if report.Stats.Ok != 1 || report.Stats.Error != 1 || len(report.Messages) != 1 || report.Messages[0].Type != importpkg.MessageError {
		t.Fatalf("report = %+v", report)
	}
}

func TestRun_okWithWarningMessages(t *testing.T) {
	registry.ResetForTest()
	registry.Register(exportpkg.ProfileRecord, fakeReader{
		result: registry.Result{
			UnitCount: 2,
			Messages: []registry.Message{
				{Type: "warning", Row: 1, Code: "retired", Text: "purged row"},
			},
		},
	})
	restoreStubReaders(t)

	report, err := Run(context.Background(), nil, exportpkg.Spec{
		Profile: exportpkg.ProfileRecord,
		Caller:  exportpkg.CallerUser,
		Model:   "base.Country",
		Format:  "csv",
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if report.Stats.Ok != 2 || report.Stats.Warning != 1 || report.Stats.Error != 0 {
		t.Fatalf("stats = %+v", report.Stats)
	}
}

func TestRun_readErrorWithWarningMessages(t *testing.T) {
	registry.ResetForTest()
	registry.Register(exportpkg.ProfileRecord, fakeReader{
		result: registry.Result{
			UnitCount: 2,
			Messages: []registry.Message{
				{Type: "warning", Row: 1, Code: "retired", Text: "purged row"},
				{Type: "error", Row: 2, Code: "constraint", Text: "bad"},
			},
		},
		err: errors.New("partial"),
	})
	restoreStubReaders(t)

	report, err := Run(context.Background(), nil, exportpkg.Spec{
		Profile: exportpkg.ProfileRecord,
		Caller:  exportpkg.CallerUser,
		Model:   "base.Country",
		Format:  "csv",
	})
	if err == nil {
		t.Fatal("expected read error")
	}
	if report.Stats.Error != 1 || report.Stats.Warning != 1 || report.Stats.Ok != 1 {
		t.Fatalf("stats = %+v", report.Stats)
	}
}

func TestStatsFromMessages(t *testing.T) {
	stats := statsFromMessages(5, []importpkg.Message{
		{Type: importpkg.MessageError},
		{Type: importpkg.MessageSkip},
		{Type: importpkg.MessageWarning},
	})
	if stats.Total != 5 || stats.Ok != 3 || stats.Error != 1 || stats.Skip != 1 || stats.Warning != 1 {
		t.Fatalf("stats = %+v", stats)
	}
}

func TestReportMeta_nilWhenEmpty(t *testing.T) {
	if got := reportMeta(plan.Plan{}); got != nil {
		t.Fatalf("reportMeta(empty) = %+v, want nil", got)
	}
}

func TestReportMeta_populated(t *testing.T) {
	meta := reportMeta(plan.Plan{Model: "base.Country"})
	if meta == nil || meta.TargetModel != "base.Country" {
		t.Fatalf("meta = %+v", meta)
	}
}

func TestRun_exportReadErrorWithoutMessages(t *testing.T) {
	registry.ResetForTest()
	registry.Register(exportpkg.ProfileRecord, fakeReader{
		result: registry.Result{UnitCount: 1},
		err:    &exportpkg.Error{Code: "constraint", Text: "bad row", Row: 1},
	})
	restoreStubReaders(t)

	report, err := Run(context.Background(), nil, exportpkg.Spec{
		Profile: exportpkg.ProfileRecord,
		Caller:  exportpkg.CallerUser,
		Model:   "base.Country",
		Format:  "csv",
	})
	if err == nil {
		t.Fatal("expected read error")
	}
	if len(report.Messages) != 1 || report.Messages[0].Code != "constraint" {
		t.Fatalf("messages = %+v", report.Messages)
	}
}

func TestRun_readErrorClampsNegativeOk(t *testing.T) {
	registry.ResetForTest()
	registry.Register(exportpkg.ProfileRecord, fakeReader{
		result: registry.Result{
			UnitCount: 1,
			Messages: []registry.Message{
				{Type: "error", Row: 1, Text: "one"},
				{Type: "error", Row: 2, Text: "two"},
			},
		},
		err: errors.New("fail"),
	})
	restoreStubReaders(t)

	report, err := Run(context.Background(), nil, exportpkg.Spec{
		Profile: exportpkg.ProfileRecord,
		Caller:  exportpkg.CallerUser,
		Model:   "base.Country",
		Format:  "csv",
	})
	if err == nil {
		t.Fatal("expected read error")
	}
	if report.Stats.Ok != 0 || report.Stats.Error != 2 {
		t.Fatalf("stats = %+v", report.Stats)
	}
}

func TestToImportMessages_empty(t *testing.T) {
	if got := toImportMessages(nil); got != nil {
		t.Fatalf("toImportMessages(nil) = %+v, want nil", got)
	}
}
