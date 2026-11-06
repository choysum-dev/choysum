// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package ops

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// Apply validates blockers and baseRevision, then moves the operation to queued.
func (o *Ops) Apply(ctx context.Context, req ApplyRequest) error {
	if o == nil {
		return errors.New("ops: nil host")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	id := strings.TrimSpace(req.Id)
	if id == "" {
		return errors.New("ops: operation id is required")
	}
	row, err := o.loadOperation(ctx, id)
	if err != nil {
		return err
	}
	if row.Status != StatusDraft {
		return fmt.Errorf("ops: operation %q status is %q, want draft", id, row.Status)
	}
	doc, err := unmarshalPlanDocument(row.PlanJson)
	if err != nil {
		return fmt.Errorf("ops: decode plan_json: %w", err)
	}
	if len(doc.Blockers) > 0 {
		return fmt.Errorf("ops: operation %q has blockers", id)
	}
	if row.Action != ActionSync && row.Action != ActionLoadDemo {
		current := computeBaseRevision(ctx, o, doc.ModuleOrder)
		if doc.BaseRevision == "" || current != doc.BaseRevision {
			return fmt.Errorf("ops: baseRevision mismatch for operation %q", id)
		}
	}
	row.Status = StatusQueued
	row.Phase = PhaseQueued
	return o.saveOperation(ctx, row)
}
