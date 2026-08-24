// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package adapter_test

import (
	"context"
	"errors"
	"testing"

	"github.com/choysum-dev/choysum/internal/import/adapter"
	"github.com/choysum-dev/choysum/internal/import/plan"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
)

type stubBuilder struct{}

func (stubBuilder) Build(context.Context, importpkg.Spec) (plan.Plan, error) {
	return plan.Plan{}, nil
}

func TestPlanBuilderFor(t *testing.T) {
	t.Cleanup(adapter.ResetPlanBuildersForTest)

	_, err := adapter.PlanBuilderFor("missing")
	if !errors.Is(err, importpkg.ErrPlanBuilderNotFound) {
		t.Fatalf("PlanBuilderFor() error = %v, want ErrPlanBuilderNotFound", err)
	}

	adapter.RegisterPlanBuilder("fmt", stubBuilder{})
	b, err := adapter.PlanBuilderFor("fmt")
	if err != nil || b == nil {
		t.Fatalf("PlanBuilderFor() = %v, %v", b, err)
	}

	adapter.RegisterPlanBuilder("fmt", nil)
	_, err = adapter.PlanBuilderFor("fmt")
	if !errors.Is(err, importpkg.ErrPlanBuilderNotFound) {
		t.Fatalf("PlanBuilderFor(nil) error = %v, want ErrPlanBuilderNotFound", err)
	}
}
