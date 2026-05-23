// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package task

import (
	"testing"
	"time"
)

func TestToAttrsIncludesSupportedTypes(t *testing.T) {
	attrs := toAttrs(map[string]any{
		"name":   "choysum",
		"ok":     true,
		"count":  3,
		"total":  int64(5),
		"ratio":  1.5,
		"ignore": []string{"x"},
	})

	if len(attrs) != 5 {
		t.Fatalf("attrs len = %d, want 5", len(attrs))
	}
}

func TestToInt64AcceptsNumericTypes(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   any
		want int64
		ok   bool
	}{
		{name: "int64", in: int64(9), want: 9, ok: true},
		{name: "int", in: int(7), want: 7, ok: true},
		{name: "float64", in: float64(5.9), want: 5, ok: true},
		{name: "float32", in: float32(4.9), want: 4, ok: true},
		{name: "string", in: "4", want: 0, ok: false},
	} {
		got, ok := toInt64(tc.in)
		if ok != tc.ok || got != tc.want {
			t.Fatalf("toInt64(%s) = (%d, %v), want (%d, %v)", tc.name, got, ok, tc.want, tc.ok)
		}
	}
}

func TestRecordMetricAcceptsKnownNames(t *testing.T) {
	RecordMetric("task_jobs_queued_age", map[string]any{"age_ms": int64(10), "target_app": "auth"})
	RecordMetric("task_dispatch_latency", map[string]any{"latency_ms": int(5)})
	RecordMetric("task_execute_duration", map[string]any{"duration_ms": float64(8)})
	RecordMetric("task_inflight", map[string]any{"count": 2})
	RecordMetric("task_dispatch_result", map[string]any{"status": "ok"})
	RecordMetric("task_execute_result", map[string]any{"status": "ok"})
	RecordMetric("task_retry_after", map[string]any{"retry_after_ms": int64(50)})
	RecordMetric("task_retry_after_adjusted", map[string]any{"reason": "cap"})
	RecordMetric("task_gc_deleted", map[string]any{"count": int64(3)})
	RecordMetric("task_wakeup_events", map[string]any{"source": "enqueue"})
	RecordMetric("task_poll_fallback_hits", map[string]any{"count": 2})
	RecordMetric("task_schedule_triggered", map[string]any{"schedule_id": "s1"})
	RecordMetric("task_schedule_next_run", map[string]any{"next_run_at": time.Now().UTC().Add(time.Minute)})
	RecordMetric("task_schedule_enqueue_skipped", map[string]any{"reason": "rate_limit"})
	RecordMetric("unknown_metric", map[string]any{"ignored": true})
}
