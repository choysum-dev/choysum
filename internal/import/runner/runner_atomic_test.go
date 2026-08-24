// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package runner_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/choysum-dev/choysum/internal/defaultscope"
	stubadapter "github.com/choysum-dev/choysum/internal/import/adapter/stub"
	"github.com/choysum-dev/choysum/internal/import/runner"
	stubwriter "github.com/choysum-dev/choysum/internal/import/writer/stub"
	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/config"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
	"github.com/choysum-dev/choysum/pkg/scope"
)

func testRuntimeScope(t *testing.T) scope.Scope {
	t.Helper()
	cfg := &config.Config{
		Db: &config.DbConfig{
			Dialect: "sqlite",
			DSN:     filepath.Join(t.TempDir(), "import.db"),
		},
	}
	runtimeScope := defaultscope.NewDefaultScope(
		context.Background(),
		scopetest.FactoryInputFromConfig(cfg),
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
	if err := runtimeScope.Session().AutoMigrate(&stubwriter.Row{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	return runtimeScope
}

func testRecordSpec(opts importpkg.Options) importpkg.Spec {
	spec := importpkg.Spec{
		Profile: importpkg.ProfileRecord,
		Caller:  importpkg.CallerUser,
		Policy:  importpkg.PolicyAtomic,
		Model:   "base.Country",
		Source:  importpkg.Source{Format: stubadapter.Format},
		Options: opts,
	}
	return spec
}

func countStubRows(t *testing.T, runtimeScope scope.Scope) int64 {
	t.Helper()
	var count int64
	if err := runtimeScope.Session().Model(&stubwriter.Row{}).Count(&count).Error; err != nil {
		t.Fatalf("Count: %v", err)
	}
	return count
}

func TestRun_Atomic_RollsBackOnHardError(t *testing.T) {
	runtimeScope := testRuntimeScope(t)
	spec := testRecordSpec(importpkg.Options{
		StubUnitCount:     3,
		StubFailUnitIndex: 2,
	})

	report, err := runner.Run(context.Background(), runtimeScope, spec)
	if err == nil {
		t.Fatal("expected error")
	}
	if report.Stats.Error != 1 {
		t.Fatalf("report stats error = %d, want 1", report.Stats.Error)
	}
	if count := countStubRows(t, runtimeScope); count != 0 {
		t.Fatalf("row count = %d, want 0 after atomic rollback", count)
	}
}

func TestRun_DryRun_CommitsNothing(t *testing.T) {
	runtimeScope := testRuntimeScope(t)
	spec := testRecordSpec(importpkg.Options{StubUnitCount: 2})
	spec.DryRun = true

	report, err := runner.Run(context.Background(), runtimeScope, spec)
	if err != nil {
		t.Fatalf("Run dry-run: %v", err)
	}
	if !report.DryRun {
		t.Fatal("expected dry_run true in report")
	}
	if report.Stats.Ok != 2 {
		t.Fatalf("stats ok = %d, want 2", report.Stats.Ok)
	}
	if count := countStubRows(t, runtimeScope); count != 0 {
		t.Fatalf("row count = %d, want 0 after dry-run rollback", count)
	}
}

func TestRun_Atomic_DryRun_ReturnsErrorOnUnitFailure(t *testing.T) {
	runtimeScope := testRuntimeScope(t)
	spec := testRecordSpec(importpkg.Options{
		StubUnitCount:     2,
		StubFailUnitIndex: 1,
	})
	spec.DryRun = true

	_, err := runner.Run(context.Background(), runtimeScope, spec)
	if err == nil {
		t.Fatal("expected dry-run atomic error when unit fails")
	}
	if count := countStubRows(t, runtimeScope); count != 0 {
		t.Fatalf("row count = %d, want 0", count)
	}
}

func TestValidateSpec_CallerProfileMatrix(t *testing.T) {
	cases := []struct {
		profile importpkg.Profile
		caller  importpkg.Caller
		ok      bool
	}{
		{importpkg.ProfileInitdata, importpkg.CallerLifecycle, true},
		{importpkg.ProfileInitdata, importpkg.CallerE2E, true},
		{importpkg.ProfileInitdata, importpkg.CallerUser, false},
		{importpkg.ProfileTerminology, importpkg.CallerCLI, true},
		{importpkg.ProfileTerminology, importpkg.CallerUser, false},
		{importpkg.ProfileRecord, importpkg.CallerUser, true},
	}
	for _, tc := range cases {
		spec := importpkg.Spec{
			Profile: tc.profile,
			Caller:  tc.caller,
			Policy:  importpkg.PolicyAtomic,
			Module:  "base",
			Model:   "base.Country",
			Source:  importpkg.Source{Format: stubadapter.Format},
		}
		err := importpkg.ValidateSpec(spec)
		if tc.ok && err != nil {
			t.Fatalf("profile=%s caller=%s: unexpected error %v", tc.profile, tc.caller, err)
		}
		if !tc.ok && !errors.Is(err, importpkg.ErrCallerProfileDenied) {
			t.Fatalf("profile=%s caller=%s: error = %v, want ErrCallerProfileDenied", tc.profile, tc.caller, err)
		}
	}
}
