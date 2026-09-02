// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package task

import (
	"context"
	"strings"
	"time"

	"github.com/choysum-dev/choysum/pkg/bus"
)

const (
	metaModuleOpMethodPrefix = "meta.MetaModule/Execute"
	metaModuleOpTipModel     = "task.Job"
)

func isMetaModuleOpJob(job *Job) bool {
	if job == nil {
		return false
	}
	return strings.HasPrefix(strings.TrimSpace(job.FullMethod), metaModuleOpMethodPrefix)
}

func moduleOpOperatorUserID(job *Job) string {
	if job == nil {
		return ""
	}
	if userID := strings.TrimSpace(job.TriggeredByUserId); userID != "" {
		return userID
	}
	return strings.TrimSpace(job.SchedulerUserId)
}

// publishModuleOpChanged emits a best-effort tip after Meta module-op status
// has been persisted. Publish failures are ignored and never roll back the job.
func (d *Dispatcher) publishModuleOpChanged(job *Job, source string) {
	if d == nil || d.events == nil || !isMetaModuleOpJob(job) {
		return
	}
	jobID := strings.TrimSpace(job.Id)
	if jobID == "" {
		return
	}
	src := strings.TrimSpace(source)
	if src == "" {
		src = "task.Dispatcher"
	}
	_ = d.events.Publish(context.Background(), bus.Event{
		Topic:  bus.TopicMetaModuleOpChanged,
		Source: src,
		At:     time.Now().UTC(),
		Payload: map[string]any{
			"model":  metaModuleOpTipModel,
			"resId":  jobID,
			"jobId":  jobID,
			"userId": moduleOpOperatorUserID(job),
		},
	})
}
