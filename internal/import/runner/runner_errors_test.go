// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package runner_test

import (
	"context"
	"errors"
	"testing"

	"github.com/choysum-dev/choysum/internal/import/adapter"
	stubadapter "github.com/choysum-dev/choysum/internal/import/adapter/stub"
	"github.com/choysum-dev/choysum/internal/import/plan"
	"github.com/choysum-dev/choysum/internal/import/registry"
	"github.com/choysum-dev/choysum/internal/import/runner"
	stubwriter "github.com/choysum-dev/choysum/internal/import/writer/stub"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
)

type failingBuilder struct{}

func (failingBuilder) Build(context.Context, importpkg.Spec) (plan.Plan, error) {
	return plan.Plan{}, errors.New("build failed")
}

func restoreStubRegistrations(t *testing.T) {
	t.Helper()
	adapter.RegisterPlanBuilder(stubadapter.Format, stubadapter.Builder{})
	registry.RegisterWriter(importpkg.ProfileRecord, stubwriter.Writer{})
	registry.RegisterWriter(importpkg.ProfileInitdata, stubwriter.Writer{})
	registry.RegisterWriter(importpkg.ProfileTerminology, stubwriter.Writer{})
}

func TestRun_PlanBuilderNotFound(t *testing.T) {
	spec := testRecordSpec(importpkg.Options{})
	spec.Source.Format = "missing-format"

	_, err := runner.Run(context.Background(), testRuntimeScope(t), spec)
	if !errors.Is(err, importpkg.ErrPlanBuilderNotFound) {
		t.Fatalf("Run() error = %v, want ErrPlanBuilderNotFound", err)
	}
}

func TestRun_BuildFailure(t *testing.T) {
	adapter.RegisterPlanBuilder("fail-build", failingBuilder{})

	spec := testRecordSpec(importpkg.Options{})
	spec.Source.Format = "fail-build"

	_, err := runner.Run(context.Background(), testRuntimeScope(t), spec)
	if err == nil || err.Error() != "build failed" {
		t.Fatalf("Run() error = %v, want build failed", err)
	}
}

func TestRun_WriterNotRegistered(t *testing.T) {
	registry.ResetWritersForTest()
	t.Cleanup(func() { restoreStubRegistrations(t) })

	adapter.ResetPlanBuildersForTest()
	t.Cleanup(func() {
		adapter.ResetPlanBuildersForTest()
		restoreStubRegistrations(t)
	})
	adapter.RegisterPlanBuilder("fmt", emptyPlanBuilder{})

	spec := testRecordSpec(importpkg.Options{StubUnitCount: 1})
	spec.Source.Format = "fmt"

	_, err := runner.Run(context.Background(), testRuntimeScope(t), spec)
	if !errors.Is(err, importpkg.ErrWriterNotRegistered) {
		t.Fatalf("Run() error = %v, want ErrWriterNotRegistered", err)
	}
}

type emptyPlanBuilder struct{}

func (emptyPlanBuilder) Build(context.Context, importpkg.Spec) (plan.Plan, error) {
	return plan.Plan{}, nil
}
