// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package registry

import (
	"context"

	"github.com/choysum-dev/choysum/internal/export/plan"
	"github.com/choysum-dev/choysum/pkg/scope"
)

// Result is the Reader output before Sink serialization.
type Result struct {
	UnitCount int
	Messages  []Message
	// Outcomes holds per-unit aggregates when Total > 0; otherwise the runner derives stats from Messages.
	Outcomes Outcomes
}

// Outcomes aggregates unit-level export results from a reader.
type Outcomes struct {
	Total   int
	Ok      int
	Error   int
	Skip    int
	Warning int
}

// HasOutcomes reports whether the reader supplied authoritative outcome counts.
func (r Result) HasOutcomes() bool {
	return r.Outcomes.Total > 0
}

// Message mirrors import report lines without importing importpkg into the interface surface.
type Message struct {
	Type      string
	Row       int
	Field     string
	Code      string
	Text      string
	RecordRef string
}

// Reader reads export units for one profile.
type Reader interface {
	Read(ctx context.Context, runtimeScope scope.Scope, p plan.Plan) (Result, error)
}
