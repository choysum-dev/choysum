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
	csvsink "github.com/choysum-dev/choysum/internal/export/sink/csv"
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

type fakeSink struct {
	err error
}

func (f fakeSink) Write(context.Context, scope.Scope, plan.Plan, *registry.Result) error {
	return f.err
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

func TestRun_readErrorWithWarningOnly(t *testing.T) {
	registry.ResetForTest()
	registry.Register(exportpkg.ProfileRecord, fakeReader{
		result: registry.Result{
			UnitCount: 2,
			Messages: []registry.Message{
				{Type: "warning", Row: 1, Code: "retired", Text: "purged row"},
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
	if len(report.Messages) < 2 {
		t.Fatalf("messages = %+v", report.Messages)
	}
	if report.Messages[len(report.Messages)-1].Type != importpkg.MessageError {
		t.Fatalf("last message = %+v, want error from readErr", report.Messages[len(report.Messages)-1])
	}
	if report.Stats.Error != 1 || report.Stats.Warning != 1 {
		t.Fatalf("stats = %+v", report.Stats)
	}
}

func TestBuildStats_syntheticErrorWithOutcomes(t *testing.T) {
	stats := buildStats(registry.Result{
		Outcomes: registry.Outcomes{Total: 2, Ok: 2, Warning: 1},
	}, []importpkg.Message{
		{Type: importpkg.MessageWarning, Row: 1, Text: "warn"},
	}, true)
	if stats.Error != 1 || stats.Ok != 1 || stats.Warning != 1 {
		t.Fatalf("stats = %+v", stats)
	}
}

func TestBuildStats_syntheticErrorConvertsSkip(t *testing.T) {
	stats := buildStats(registry.Result{
		Outcomes: registry.Outcomes{Total: 1, Ok: 0, Skip: 1},
	}, []importpkg.Message{
		{Type: importpkg.MessageSkip, Row: 1, Text: "skipped"},
	}, true)
	if stats.Error != 1 || stats.Skip != 0 || stats.Ok+stats.Error+stats.Skip != stats.Total {
		t.Fatalf("stats = %+v", stats)
	}
}

func TestBuildStats_derivesTotalFromMessages(t *testing.T) {
	stats := buildStats(registry.Result{}, []importpkg.Message{
		{Type: importpkg.MessageError, Row: 1},
	}, false)
	if stats.Total != 1 || stats.Error != 1 {
		t.Fatalf("stats = %+v", stats)
	}
}

func TestOutcomeRank_prefersErrorOverSkip(t *testing.T) {
	if outcomeRank(importpkg.MessageSkip) >= outcomeRank(importpkg.MessageError) {
		t.Fatal("error should outrank skip")
	}
	if outcomeRank(importpkg.MessageWarning) != 1 {
		t.Fatalf("warning rank = %d, want 1", outcomeRank(importpkg.MessageWarning))
	}
	if outcomeRank(importpkg.MessageType("bogus")) != 3 {
		t.Fatalf("default rank = %d, want 3", outcomeRank(importpkg.MessageType("bogus")))
	}
}

func TestHasErrorClassMessage(t *testing.T) {
	if hasErrorClassMessage([]importpkg.Message{{Type: importpkg.MessageWarning}}) {
		t.Fatal("warning-only should not count as error class")
	}
	if !hasErrorClassMessage([]importpkg.Message{{Type: importpkg.MessageError}}) {
		t.Fatal("error should count as error class")
	}
}

func TestStatsFromMessages_dedupesRows(t *testing.T) {
	stats := statsFromMessages(3, []importpkg.Message{
		{Type: importpkg.MessageError, Row: 1},
		{Type: importpkg.MessageError, Row: 1},
		{Type: importpkg.MessageSkip, Row: 2},
	})
	if stats.Error != 1 || stats.Skip != 1 || stats.Ok != 1 {
		t.Fatalf("stats = %+v", stats)
	}

	stats = statsFromMessages(2, []importpkg.Message{
		{Type: importpkg.MessageSkip, Row: 1},
		{Type: importpkg.MessageError, Row: 1},
	})
	if stats.Error != 1 || stats.Skip != 0 || stats.Ok != 1 {
		t.Fatalf("error-over-skip stats = %+v", stats)
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

	stats = statsFromMessages(1, []importpkg.Message{
		{Type: importpkg.MessageError, Row: 1},
		{Type: importpkg.MessageSkip, Row: 2},
	})
	if stats.Ok != 0 {
		t.Fatalf("clamped ok = %d, want 0", stats.Ok)
	}

	stats = statsFromMessages(2, []importpkg.Message{{Row: 1, Text: "empty type"}})
	if stats.Error != 1 || stats.Ok != 1 {
		t.Fatalf("empty type stats = %+v", stats)
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

func TestRun_readErrorUsesReaderOutcomes(t *testing.T) {
	registry.ResetForTest()
	registry.Register(exportpkg.ProfileRecord, fakeReader{
		result: registry.Result{
			UnitCount: 1,
			Outcomes: registry.Outcomes{
				Total: 1,
				Ok:    0,
				Error: 1,
			},
			Messages: []registry.Message{
				{Type: "error", Row: 1, Text: "one"},
				{Type: "error", Row: 1, Text: "duplicate diagnostic"},
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
	if report.Stats.Ok != 0 || report.Stats.Error != 1 {
		t.Fatalf("stats = %+v", report.Stats)
	}
}

func TestToImportMessages_empty(t *testing.T) {
	if got := toImportMessages(nil); got != nil {
		t.Fatalf("toImportMessages(nil) = %+v, want nil", got)
	}
}

func TestRun_sinkLookupError(t *testing.T) {
	registry.ResetForTest()
	registry.ResetSinksForTest()
	registry.Register(exportpkg.ProfileRecord, fakeReader{
		result: registry.Result{
			Headers: []string{"Name"},
			Rows:    [][]string{{"A"}},
		},
	})
	restoreStubReaders(t)

	_, err := Run(context.Background(), nil, exportpkg.Spec{
		Profile: exportpkg.ProfileRecord,
		Caller:  exportpkg.CallerUser,
		Model:   "base.Country",
		Format:  "csv",
	})
	if err == nil {
		t.Fatal("expected sink lookup error")
	}
}

func TestRun_noHeadersSkipsSink(t *testing.T) {
	registry.ResetForTest()
	registry.Register(exportpkg.ProfileRecord, fakeReader{
		result: registry.Result{
			Outcomes: registry.Outcomes{Total: 0, Ok: 0},
		},
	})
	registry.RegisterSink("csv", fakeSink{err: errors.New("should not run")})
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
	if report.Stats.Total != 0 {
		t.Fatalf("stats = %+v", report.Stats)
	}
}

func TestRun_sinkWriteError(t *testing.T) {
	registry.ResetForTest()
	registry.Register(exportpkg.ProfileRecord, fakeReader{
		result: registry.Result{
			Headers: []string{"Name"},
			Rows:    [][]string{{"A"}},
		},
	})
	registry.RegisterSink("csv", fakeSink{err: errors.New("write failed")})
	restoreStubReaders(t)

	_, err := Run(context.Background(), nil, exportpkg.Spec{
		Profile: exportpkg.ProfileRecord,
		Caller:  exportpkg.CallerUser,
		Model:   "base.Country",
		Format:  "csv",
	})
	if err == nil {
		t.Fatal("expected sink write error")
	}
}

func TestRun_sinkSuccess(t *testing.T) {
	registry.ResetForTest()
	registry.Register(exportpkg.ProfileRecord, fakeReader{
		result: registry.Result{
			Headers: []string{"Name", "Code"},
			Rows:    [][]string{{"Alpha", "A1"}},
			Outcomes: registry.Outcomes{
				Total: 1,
				Ok:    1,
			},
		},
	})
	registry.RegisterSink("csv", csvsink.Writer{})
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
	if report.Stats.Total != 1 {
		t.Fatalf("stats = %+v", report.Stats)
	}
}

func TestRun_attachReportCSV(t *testing.T) {
	orig := attachReportCSV
	t.Cleanup(func() { attachReportCSV = orig })
	attachReportCSV = func(_ context.Context, _ scope.Scope, companyID string, csvBytes []byte, report *importpkg.Report) error {
		if companyID != "co-1" || len(csvBytes) == 0 {
			t.Fatalf("companyID=%q csvBytes=%d", companyID, len(csvBytes))
		}
		report.ArtifactRef = "ref-1"
		return nil
	}

	registry.ResetForTest()
	registry.Register(exportpkg.ProfileRecord, fakeReader{
		result: registry.Result{
			Headers: []string{"Name"},
			Rows:    [][]string{{"A"}},
			Outcomes: registry.Outcomes{
				Total: 1,
				Ok:    1,
			},
		},
	})
	registry.RegisterSink("csv", csvsink.Writer{})
	restoreStubReaders(t)

	report, err := Run(context.Background(), nil, exportpkg.Spec{
		Profile: exportpkg.ProfileRecord,
		Caller:  exportpkg.CallerUser,
		Model:   "base.Country",
		Format:  "csv",
		Options: exportpkg.Options{CompanyID: "co-1"},
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if report.ArtifactRef != "ref-1" {
		t.Fatalf("ArtifactRef = %q", report.ArtifactRef)
	}
}

func TestRun_attachReportCSV_storeError(t *testing.T) {
	orig := attachReportCSV
	t.Cleanup(func() { attachReportCSV = orig })
	attachReportCSV = func(context.Context, scope.Scope, string, []byte, *importpkg.Report) error {
		return errors.New("store failed")
	}

	registry.ResetForTest()
	registry.Register(exportpkg.ProfileRecord, fakeReader{
		result: registry.Result{
			Headers: []string{"Name"},
			Rows:    [][]string{{"A"}},
			Outcomes: registry.Outcomes{
				Total: 1,
				Ok:    1,
			},
		},
	})
	registry.RegisterSink("csv", csvsink.Writer{})
	restoreStubReaders(t)

	_, err := Run(context.Background(), nil, exportpkg.Spec{
		Profile: exportpkg.ProfileRecord,
		Caller:  exportpkg.CallerUser,
		Model:   "base.Country",
		Format:  "csv",
		Options: exportpkg.Options{CompanyID: "co-1"},
	})
	if err == nil {
		t.Fatal("expected artifact store error")
	}
}
