// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package importpkg_test

import (
	"context"
	"errors"
	"testing"

	importpkg "github.com/choysum-dev/choysum/pkg/import"
	"github.com/choysum-dev/choysum/pkg/scope"
)

func TestRun_ValidateSpecError(t *testing.T) {
	t.Cleanup(func() { importpkg.SetRun(nil) })
	importpkg.SetRun(func(context.Context, scope.Scope, importpkg.Spec) (importpkg.Report, error) {
		t.Fatal("runFn should not be called when validation fails")
		return importpkg.Report{}, nil
	})

	_, err := importpkg.Run(context.Background(), nil, importpkg.Spec{})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestRun_UnregisteredRunner(t *testing.T) {
	t.Cleanup(func() { importpkg.SetRun(nil) })

	spec := importpkg.Spec{
		Profile: importpkg.ProfileRecord,
		Caller:  importpkg.CallerUser,
		Policy:  importpkg.PolicyAtomic,
		Model:   "base.Country",
		Source:  importpkg.Source{Format: "stub"},
	}

	_, err := importpkg.Run(context.Background(), nil, spec)
	if err == nil {
		t.Fatal("expected runner_not_registered error")
	}
	var impErr *importpkg.Error
	if !errors.As(err, &impErr) || impErr.Code != importpkg.CodeRunnerNotRegistered {
		t.Fatalf("Run() error = %v, want CodeRunnerNotRegistered", err)
	}
}

func TestRun_AsyncRejected(t *testing.T) {
	importpkg.SetRun(func(context.Context, scope.Scope, importpkg.Spec) (importpkg.Report, error) {
		return importpkg.Report{}, nil
	})
	t.Cleanup(func() { importpkg.SetRun(nil) })

	spec := importpkg.Spec{
		Profile: importpkg.ProfileRecord,
		Caller:  importpkg.CallerUser,
		Policy:  importpkg.PolicyAtomic,
		Model:   "base.Country",
		Source:  importpkg.Source{Format: "csv"},
		Async:   true,
	}

	_, err := importpkg.Run(context.Background(), nil, spec)
	if !errors.Is(err, importpkg.ErrAsyncNotSupported) {
		t.Fatalf("Run(async) error = %v, want ErrAsyncNotSupported", err)
	}
}

func TestSetRun(t *testing.T) {
	t.Cleanup(func() { importpkg.SetRun(nil) })

	called := false
	importpkg.SetRun(func(context.Context, scope.Scope, importpkg.Spec) (importpkg.Report, error) {
		called = true
		return importpkg.Report{Profile: importpkg.ProfileRecord}, nil
	})

	spec := importpkg.Spec{
		Profile: importpkg.ProfileRecord,
		Caller:  importpkg.CallerUser,
		Policy:  importpkg.PolicyAtomic,
		Model:   "base.Country",
		Source:  importpkg.Source{Format: "csv"},
	}
	report, err := importpkg.Run(context.Background(), nil, spec)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !called || report.Profile != importpkg.ProfileRecord {
		t.Fatalf("SetRun hook not invoked: called=%v report=%+v", called, report)
	}

	importpkg.SetRun(nil)
	_, err = importpkg.Run(context.Background(), nil, spec)
	if err == nil {
		t.Fatal("expected unregistered error after SetRun(nil)")
	}
}
