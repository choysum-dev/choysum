// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package runner_test

import (
	"context"
	"testing"

	"github.com/choysum-dev/choysum/internal/import/runner"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
	"github.com/choysum-dev/choysum/pkg/scope"
)

// TestRun_Atomic_FallsBackWhenNestedUnsupported covers the lifecycle-style
// Transactor path that returns ErrNestedUnsupported from Nested.
func TestRun_Atomic_FallsBackWhenNestedUnsupported(t *testing.T) {
	base := testRuntimeScope(t)
	runtimeScope := &transactorOverrideScope{
		Scope: base,
		tx: fallbackToRequiredTransactor{
			base: base.Transactor(),
		},
	}
	spec := testRecordSpec(importpkg.Options{StubUnitCount: 1})

	report, err := runner.Run(context.Background(), runtimeScope, spec)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if report.Stats.Ok != 1 {
		t.Fatalf("stats ok = %d, want 1", report.Stats.Ok)
	}
	if count := countStubRows(t, base); count != 1 {
		t.Fatalf("row count = %d, want 1", count)
	}
}

type transactorOverrideScope struct {
	scope.Scope
	tx scope.Transactor
}

func (s *transactorOverrideScope) Transactor() scope.Transactor { return s.tx }

type fallbackToRequiredTransactor struct {
	base scope.Transactor
}

func (t fallbackToRequiredTransactor) Do(ctx context.Context, opts scope.TransactionOptions, fn scope.TxFunc) error {
	return t.base.Do(ctx, opts, fn)
}
func (t fallbackToRequiredTransactor) Required(ctx context.Context, fn scope.TxFunc) error {
	return t.base.Required(ctx, fn)
}
func (t fallbackToRequiredTransactor) RequiresNew(ctx context.Context, fn scope.TxFunc) error {
	return t.base.RequiresNew(ctx, fn)
}
func (t fallbackToRequiredTransactor) Nested(ctx context.Context, fn scope.TxFunc) error {
	return scope.ErrNestedUnsupported
}
