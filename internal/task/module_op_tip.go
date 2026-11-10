// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package task

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/choysum-dev/choysum/pkg/bus"
)

const (
	metaModuleOpMethodPrefix = "meta.MetaModuleOperation/Execute"
	metaModuleOpTipModel     = "meta.MetaModuleOperation"
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

func moduleOpPayloadString(v any) string {
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(s)
}

func moduleOpIDFromReqMap(req map[string]any) string {
	if id := moduleOpPayloadString(req["Id"]); id != "" {
		return id
	}
	return moduleOpPayloadString(req["id"])
}

func moduleOpOperationID(job *Job) string {
	if job == nil || len(job.PayloadJson) == 0 {
		return ""
	}
	var payload map[string]any
	if err := json.Unmarshal(job.PayloadJson, &payload); err != nil || payload == nil {
		return ""
	}
	if req, ok := payload["req"].(map[string]any); ok {
		if id := moduleOpIDFromReqMap(req); id != "" {
			return id
		}
	}
	// Some stores may keep req as a JSON string instead of an object.
	if raw, ok := payload["req"].(string); ok && strings.TrimSpace(raw) != "" {
		var req map[string]any
		if err := json.Unmarshal([]byte(raw), &req); err == nil {
			if id := moduleOpIDFromReqMap(req); id != "" {
				return id
			}
		}
	}
	return moduleOpIDFromReqMap(payload)
}

// publishModuleOpChanged emits a best-effort tip after Meta module-op status
// has been persisted. Publish failures are ignored and never roll back the job.
func (d *Dispatcher) publishModuleOpChanged(job *Job, source string) {
	if d == nil || d.events == nil || !isMetaModuleOpJob(job) {
		return
	}
	jobID := strings.TrimSpace(job.Id)
	operationID := moduleOpOperationID(job)
	if operationID == "" {
		operationID = jobID
	}
	if operationID == "" {
		return
	}
	src := strings.TrimSpace(source)
	if src == "" {
		src = "task.Dispatcher"
	}
	payload := map[string]any{
		"model":       metaModuleOpTipModel,
		"resId":       operationID,
		"operationId": operationID,
		"userId":      moduleOpOperatorUserID(job),
	}
	if jobID != "" {
		payload["jobId"] = jobID
	}
	_ = d.events.Publish(context.Background(), bus.Event{
		Topic:   bus.TopicMetaModuleOpChanged,
		Source:  src,
		At:      time.Now().UTC(),
		Payload: payload,
	})
}
