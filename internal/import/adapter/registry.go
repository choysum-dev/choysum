// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package adapter

import (
	"context"
	"sync"

	"github.com/choysum-dev/choysum/internal/import/plan"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
)

// PlanBuilder parses a spec source into a plan.
type PlanBuilder interface {
	Build(ctx context.Context, spec importpkg.Spec) (plan.Plan, error)
}

var (
	builderMu sync.RWMutex
	builders  = make(map[string]PlanBuilder)
)

// RegisterPlanBuilder registers a builder for a source format string.
func RegisterPlanBuilder(format string, b PlanBuilder) {
	builderMu.Lock()
	defer builderMu.Unlock()
	builders[format] = b
}

// PlanBuilderFor returns the builder for format.
func PlanBuilderFor(format string) (PlanBuilder, error) {
	builderMu.RLock()
	b, ok := builders[format]
	builderMu.RUnlock()
	if !ok || b == nil {
		return nil, importpkg.ErrPlanBuilderNotFound
	}
	return b, nil
}

// ResetPlanBuildersForTest clears registered builders. Tests only.
func ResetPlanBuildersForTest() {
	builderMu.Lock()
	builders = make(map[string]PlanBuilder)
	builderMu.Unlock()
}
