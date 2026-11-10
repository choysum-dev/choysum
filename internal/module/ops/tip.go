// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package ops

import (
	"context"
	"database/sql"
	"strings"
	"time"

	modmeta "github.com/choysum-dev/choysum/internal/module/meta"
	"github.com/choysum-dev/choysum/pkg/bus"
)

const tipSource = "internal/module/ops"

// publishOperationChanged emits a best-effort tip keyed by operation Id.
// Publish failures never roll back the operation row.
func (o *Ops) publishOperationChanged(ctx context.Context, row *modmeta.ModuleOperation) {
	if o == nil || o.Events == nil || row == nil {
		return
	}
	opID := ""
	if row.Id.Valid {
		opID = strings.TrimSpace(row.Id.String)
	}
	if opID == "" {
		return
	}
	payload := map[string]any{
		"operationId": opID,
		"model":       "meta.ModuleOperation",
		"resId":       opID,
		"userId":      strings.TrimSpace(row.OperatorUserId),
	}
	if row.JobId.Valid && strings.TrimSpace(row.JobId.String) != "" {
		payload["jobId"] = strings.TrimSpace(row.JobId.String)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	_ = o.Events.Publish(ctx, bus.Event{
		Topic:   bus.TopicMetaModuleOpChanged,
		Source:  tipSource,
		At:      time.Now().UTC(),
		Payload: payload,
	})
}

func nullString(s string) sql.NullString {
	s = strings.TrimSpace(s)
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}
