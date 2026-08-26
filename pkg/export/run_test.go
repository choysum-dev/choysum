// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package exportpkg_test

import (
	"context"
	"errors"
	"testing"

	exportpkg "github.com/choysum-dev/choysum/pkg/export"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
	"github.com/choysum-dev/choysum/pkg/scope"
)

func TestRun_ValidateSpecError(t *testing.T) {
	t.Cleanup(func() { exportpkg.SetRun(nil) })
	exportpkg.SetRun(func(context.Context, scope.Scope, exportpkg.Spec) (importpkg.Report, error) {
		t.Fatal("runFn should not be called when validation fails")
		return importpkg.Report{}, nil
	})

	_, err := exportpkg.Run(context.Background(), nil, exportpkg.Spec{})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestRun_UnregisteredRunner(t *testing.T) {
	t.Cleanup(func() { exportpkg.SetRun(nil) })

	spec := exportpkg.Spec{
		Profile: exportpkg.ProfileRecord,
		Caller:  exportpkg.CallerUser,
		Model:   "base.Country",
		Format:  "csv",
	}

	_, err := exportpkg.Run(context.Background(), nil, spec)
	if err == nil {
		t.Fatal("expected runner_not_registered error")
	}
	var expErr *exportpkg.Error
	if !errors.As(err, &expErr) || expErr.Code != exportpkg.CodeRunnerNotRegistered {
		t.Fatalf("Run() error = %v, want CodeRunnerNotRegistered", err)
	}
}

func TestRun_AsyncRejected(t *testing.T) {
	exportpkg.SetRun(func(context.Context, scope.Scope, exportpkg.Spec) (importpkg.Report, error) {
		return importpkg.Report{}, nil
	})
	t.Cleanup(func() { exportpkg.SetRun(nil) })

	spec := exportpkg.Spec{
		Profile: exportpkg.ProfileRecord,
		Caller:  exportpkg.CallerUser,
		Model:   "base.Country",
		Format:  "csv",
		Async:   true,
	}

	_, err := exportpkg.Run(context.Background(), nil, spec)
	if !errors.Is(err, exportpkg.ErrAsyncNotSupported) {
		t.Fatalf("Run(async) error = %v, want ErrAsyncNotSupported", err)
	}
}

func TestSetRun(t *testing.T) {
	t.Cleanup(func() { exportpkg.SetRun(nil) })

	called := false
	exportpkg.SetRun(func(context.Context, scope.Scope, exportpkg.Spec) (importpkg.Report, error) {
		called = true
		return importpkg.Report{Profile: importpkg.ProfileRecord}, nil
	})

	spec := exportpkg.Spec{
		Profile: exportpkg.ProfileRecord,
		Caller:  exportpkg.CallerUser,
		Model:   "base.Country",
		Format:  "csv",
	}
	report, err := exportpkg.Run(context.Background(), nil, spec)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !called || report.Profile != importpkg.ProfileRecord {
		t.Fatalf("SetRun hook not invoked: called=%v report=%+v", called, report)
	}

	exportpkg.SetRun(nil)
	_, err = exportpkg.Run(context.Background(), nil, spec)
	if err == nil {
		t.Fatal("expected unregistered error after SetRun(nil)")
	}
}
