// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package stub

import (
	"context"
	"fmt"

	"github.com/choysum-dev/choysum/internal/export/plan"
	"github.com/choysum-dev/choysum/internal/export/registry"
	exportpkg "github.com/choysum-dev/choysum/pkg/export"
	"github.com/choysum-dev/choysum/pkg/scope"
)

// Reader is a no-op export stub that counts units and can inject a failure.
type Reader struct{}

// Read implements registry.Reader.
func (Reader) Read(ctx context.Context, runtimeScope scope.Scope, p plan.Plan) (registry.Result, error) {
	_ = ctx
	_ = runtimeScope

	count := p.StubUnitCount
	if count < 0 {
		count = 0
	}
	result := registry.Result{
		UnitCount: count,
		Outcomes: registry.Outcomes{
			Total: count,
			Ok:    count,
		},
	}

	failAt := p.StubFailUnitIndex
	if failAt > 0 {
		if failAt > count {
			failAt = count
			if failAt == 0 {
				failAt = 1
			}
		}
		msg := registry.Message{
			Type: "error",
			Row:  failAt,
			Code: "constraint",
			Text: fmt.Sprintf("stub unit %d failed", failAt),
		}
		result.Messages = append(result.Messages, msg)
		ok := count - 1
		if ok < 0 {
			ok = 0
		}
		result.Outcomes = registry.Outcomes{
			Total: count,
			Ok:    ok,
			Error: 1,
		}
		return result, &exportpkg.Error{
			Code: msg.Code,
			Text: msg.Text,
			Row:  msg.Row,
		}
	}
	return result, nil
}
