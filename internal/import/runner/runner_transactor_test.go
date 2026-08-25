// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package runner_test

import (
	"context"
	"strings"
	"testing"

	stubadapter "github.com/choysum-dev/choysum/internal/import/adapter/stub"
	"github.com/choysum-dev/choysum/internal/import/runner"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
)

func TestRun_StopKeep_PartialCommit(t *testing.T) {
	runtimeScope := testRuntimeScope(t)
	spec := testRecordSpec(importpkg.Options{
		StubUnitCount:     3,
		StubFailUnitIndex: 2,
	})
	spec.Policy = importpkg.PolicyStopKeep

	report, err := runner.Run(context.Background(), runtimeScope, spec)
	if err == nil {
		t.Fatal("expected error")
	}
	if report.Stats.Ok != 1 {
		t.Fatalf("stats ok = %d, want 1", report.Stats.Ok)
	}
	if report.Stats.Skip != 1 {
		t.Fatalf("stats skip = %d, want 1", report.Stats.Skip)
	}
	if count := countStubRows(t, runtimeScope); count != 1 {
		t.Fatalf("row count = %d, want 1 after stop_keep", count)
	}
}

func TestRun_BestEffort_SkipsUnit(t *testing.T) {
	runtimeScope := testRuntimeScope(t)
	spec := testRecordSpec(importpkg.Options{
		StubUnitCount:     3,
		StubFailUnitIndex: 2,
	})
	spec.Policy = importpkg.PolicyBestEffort

	report, err := runner.Run(context.Background(), runtimeScope, spec)
	if err != nil {
		t.Fatalf("best_effort should not return top-level error: %v", err)
	}
	if report.Stats.Ok != 2 {
		t.Fatalf("stats ok = %d, want 2", report.Stats.Ok)
	}
	if report.Stats.Error != 1 {
		t.Fatalf("stats error = %d, want 1", report.Stats.Error)
	}
	if count := countStubRows(t, runtimeScope); count != 2 {
		t.Fatalf("row count = %d, want 2 after best_effort", count)
	}
}

func TestRun_StubFormatRegistered(t *testing.T) {
	runtimeScope := testRuntimeScope(t)
	spec := testRecordSpec(importpkg.Options{StubUnitCount: 1})

	report, err := runner.Run(context.Background(), runtimeScope, spec)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if report.Stats.Total != 1 || report.Stats.Ok != 1 {
		t.Fatalf("unexpected stats: %+v", report.Stats)
	}
	if spec.Source.Format != stubadapter.Format {
		t.Fatalf("unexpected format %q", spec.Source.Format)
	}
}

func TestRun_DryRun_StopKeep_CommitsNothing(t *testing.T) {
	runtimeScope := testRuntimeScope(t)
	spec := testRecordSpec(importpkg.Options{StubUnitCount: 2})
	spec.Policy = importpkg.PolicyStopKeep
	spec.DryRun = true

	report, err := runner.Run(context.Background(), runtimeScope, spec)
	if err != nil {
		t.Fatalf("Run dry-run stop_keep: %v", err)
	}
	if report.Stats.Ok != 2 {
		t.Fatalf("stats ok = %d, want 2", report.Stats.Ok)
	}
	if count := countStubRows(t, runtimeScope); count != 0 {
		t.Fatalf("row count = %d, want 0 after dry-run stop_keep", count)
	}
}

func TestRun_BestEffort_FailsOnFirstUnit(t *testing.T) {
	runtimeScope := testRuntimeScope(t)
	spec := testRecordSpec(importpkg.Options{
		StubUnitCount:     3,
		StubFailUnitIndex: 1,
	})
	spec.Policy = importpkg.PolicyBestEffort

	report, err := runner.Run(context.Background(), runtimeScope, spec)
	if err != nil {
		t.Fatalf("best_effort should not return top-level error: %v", err)
	}
	if report.Stats.Ok != 2 {
		t.Fatalf("stats ok = %d, want 2", report.Stats.Ok)
	}
	if report.Stats.Error != 1 {
		t.Fatalf("stats error = %d, want 1", report.Stats.Error)
	}
	if count := countStubRows(t, runtimeScope); count != 2 {
		t.Fatalf("row count = %d, want 2 after best_effort", count)
	}
}

func TestRun_StopKeep_FailsOnFirstUnit(t *testing.T) {
	runtimeScope := testRuntimeScope(t)
	spec := testRecordSpec(importpkg.Options{
		StubUnitCount:     3,
		StubFailUnitIndex: 1,
	})
	spec.Policy = importpkg.PolicyStopKeep

	report, err := runner.Run(context.Background(), runtimeScope, spec)
	if err == nil {
		t.Fatal("expected error")
	}
	if report.Stats.Ok != 0 {
		t.Fatalf("stats ok = %d, want 0", report.Stats.Ok)
	}
	if report.Stats.Skip != 2 {
		t.Fatalf("stats skip = %d, want 2", report.Stats.Skip)
	}
	if count := countStubRows(t, runtimeScope); count != 0 {
		t.Fatalf("row count = %d, want 0 after stop_keep first-unit failure", count)
	}
}

func TestRun_BestEffort_AttachesErrorArtifact(t *testing.T) {
	runtimeScope := newArtifactRunnerScope(t)
	spec := testRecordSpec(importpkg.Options{
		StubUnitCount:     2,
		StubFailUnitIndex: 1,
	})
	spec.Policy = importpkg.PolicyBestEffort
	spec.Options.CompanyID = "cmp-artifact-runner"

	report, err := runner.Run(context.Background(), runtimeScope, spec)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if report.ArtifactRef == "" {
		t.Fatal("expected artifact_ref on report with messages")
	}
}

func TestRun_DryRun_BestEffort_SkipsErrorArtifact(t *testing.T) {
	runtimeScope := newArtifactRunnerScope(t)
	spec := testRecordSpec(importpkg.Options{
		StubUnitCount:     2,
		StubFailUnitIndex: 1,
	})
	spec.Policy = importpkg.PolicyBestEffort
	spec.DryRun = true
	spec.Options.CompanyID = "cmp-artifact-runner"

	report, err := runner.Run(context.Background(), runtimeScope, spec)
	if err != nil {
		t.Fatalf("Run dry-run best_effort: %v", err)
	}
	if report.ArtifactRef != "" {
		t.Fatalf("artifact_ref = %q, want empty during dry-run", report.ArtifactRef)
	}
}

func TestRun_BestEffort_ReturnsArtifactError(t *testing.T) {
	runtimeScope := testRuntimeScope(t)
	spec := testRecordSpec(importpkg.Options{
		StubUnitCount:     2,
		StubFailUnitIndex: 1,
	})
	spec.Policy = importpkg.PolicyBestEffort
	spec.Options.CompanyID = "cmp-artifact-runner"

	report, err := runner.Run(context.Background(), runtimeScope, spec)
	if err == nil || !strings.Contains(err.Error(), "create stored content") {
		t.Fatalf("Run() error = %v, want artifact persistence error", err)
	}
	if report.Stats.Error != 1 {
		t.Fatalf("stats error = %d, want 1", report.Stats.Error)
	}
}

func TestRun_DryRun_BestEffort_CommitsNothing(t *testing.T) {
	runtimeScope := testRuntimeScope(t)
	spec := testRecordSpec(importpkg.Options{StubUnitCount: 2})
	spec.Policy = importpkg.PolicyBestEffort
	spec.DryRun = true

	report, err := runner.Run(context.Background(), runtimeScope, spec)
	if err != nil {
		t.Fatalf("Run dry-run best_effort: %v", err)
	}
	if report.Stats.Ok != 2 {
		t.Fatalf("stats ok = %d, want 2", report.Stats.Ok)
	}
	if count := countStubRows(t, runtimeScope); count != 0 {
		t.Fatalf("row count = %d, want 0 after dry-run best_effort", count)
	}
}
