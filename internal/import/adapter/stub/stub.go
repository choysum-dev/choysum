// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package stub

import (
	"context"

	"github.com/choysum-dev/choysum/internal/import/adapter"
	"github.com/choysum-dev/choysum/internal/import/plan"
	planstub "github.com/choysum-dev/choysum/internal/import/plan/stub"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
)

const Format = "stub"

func init() {
	adapter.RegisterPlanBuilder(Format, Builder{})
}

// Builder produces stub units from Spec.Options.StubUnitCount.
type Builder struct{}

// Build implements adapter.PlanBuilder.
func (Builder) Build(ctx context.Context, spec importpkg.Spec) (plan.Plan, error) {
	_ = ctx
	count := spec.Options.StubUnitCount
	if count <= 0 {
		count = 1
	}
	failAt := spec.Options.StubFailUnitIndex

	units := make([]plan.Unit, 0, count)
	for i := 1; i <= count; i++ {
		units = append(units, planstub.Unit{Index: i, Fail: failAt > 0 && i == failAt})
	}
	return plan.Plan{Units: units}, nil
}
