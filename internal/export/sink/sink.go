// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package sink

import (
	"context"

	"github.com/choysum-dev/choysum/internal/export/plan"
	"github.com/choysum-dev/choysum/internal/export/registry"
	"github.com/choysum-dev/choysum/pkg/scope"
)

// Sink serializes Reader results (for example CSV or PO).
type Sink interface {
	Write(ctx context.Context, runtimeScope scope.Scope, p plan.Plan, result registry.Result) error
}
