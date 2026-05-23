// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package task

import (
	"context"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type metricStore struct {
	once  sync.Once
	meter metric.Meter

	queuedAge              metric.Int64Histogram
	dispatchLatency        metric.Int64Histogram
	execDuration           metric.Int64Histogram
	inflight               metric.Int64Histogram
	dispatchResult         metric.Int64Counter
	executeResult          metric.Int64Counter
	retryAfter             metric.Int64Histogram
	retryAfterAdjust       metric.Int64Counter
	gcDeleted              metric.Int64Counter
	wakeupEvents           metric.Int64Counter
	pollFallbackHits       metric.Int64Counter
	scheduleTriggered      metric.Int64Counter
	scheduleNextRun        metric.Int64Histogram
	scheduleEnqueueSkipped metric.Int64Counter
}

var store metricStore

func initMetrics() {
	store.once.Do(func() {
		store.meter = otel.Meter("choysum.task")
		store.queuedAge, _ = store.meter.Int64Histogram("task_jobs_queued_age")
		store.dispatchLatency, _ = store.meter.Int64Histogram("task_dispatch_latency")
		store.execDuration, _ = store.meter.Int64Histogram("task_execute_duration")
		store.inflight, _ = store.meter.Int64Histogram("task_inflight")
		store.dispatchResult, _ = store.meter.Int64Counter("task_dispatch_result_total")
		store.executeResult, _ = store.meter.Int64Counter("task_execute_result_total")
		store.retryAfter, _ = store.meter.Int64Histogram("task_retry_after")
		store.retryAfterAdjust, _ = store.meter.Int64Counter("task_retry_after_adjusted_total")
		store.gcDeleted, _ = store.meter.Int64Counter("task_gc_deleted_total")
		store.wakeupEvents, _ = store.meter.Int64Counter("task_wakeup_events_total")
		store.pollFallbackHits, _ = store.meter.Int64Counter("task_poll_fallback_hits_total")
		store.scheduleTriggered, _ = store.meter.Int64Counter("task_schedule_triggered_total")
		store.scheduleNextRun, _ = store.meter.Int64Histogram("task_schedule_next_run")
		store.scheduleEnqueueSkipped, _ = store.meter.Int64Counter("task_schedule_enqueue_skipped_total")
	})
}

func RecordMetric(name string, fields map[string]any) {
	initMetrics()
	attrs := toAttrs(fields)
	now := time.Now()
	ctx := context.Background()
	switch name {
	case "task_jobs_queued_age":
		if v, ok := toInt64(fields["age_ms"]); ok {
			store.queuedAge.Record(ctx, v, metric.WithAttributes(attrs...))
		}
	case "task_dispatch_latency":
		if v, ok := toInt64(fields["latency_ms"]); ok {
			store.dispatchLatency.Record(ctx, v, metric.WithAttributes(attrs...))
		}
	case "task_execute_duration":
		if v, ok := toInt64(fields["duration_ms"]); ok {
			store.execDuration.Record(ctx, v, metric.WithAttributes(attrs...))
		}
	case "task_inflight":
		if v, ok := toInt64(fields["count"]); ok {
			store.inflight.Record(ctx, v, metric.WithAttributes(attrs...))
		}
	case "task_dispatch_result":
		store.dispatchResult.Add(ctx, 1, metric.WithAttributes(attrs...))
	case "task_execute_result":
		store.executeResult.Add(ctx, 1, metric.WithAttributes(attrs...))
	case "task_retry_after":
		if v, ok := toInt64(fields["retry_after_ms"]); ok {
			store.retryAfter.Record(ctx, v, metric.WithAttributes(attrs...))
		}
	case "task_retry_after_adjusted":
		store.retryAfterAdjust.Add(ctx, 1, metric.WithAttributes(attrs...))
	case "task_gc_deleted":
		if v, ok := toInt64(fields["count"]); ok {
			store.gcDeleted.Add(ctx, v, metric.WithAttributes(attrs...))
		}
	case "task_wakeup_events":
		store.wakeupEvents.Add(ctx, 1, metric.WithAttributes(attrs...))
	case "task_poll_fallback_hits":
		if v, ok := toInt64(fields["count"]); ok {
			store.pollFallbackHits.Add(ctx, v, metric.WithAttributes(attrs...))
			return
		}
		store.pollFallbackHits.Add(ctx, 1, metric.WithAttributes(attrs...))
	case "task_schedule_triggered":
		store.scheduleTriggered.Add(ctx, 1, metric.WithAttributes(attrs...))
	case "task_schedule_next_run":
		if t, ok := fields["next_run_at"].(time.Time); ok {
			delta := t.Sub(now).Milliseconds()
			if delta < 0 {
				delta = 0
			}
			store.scheduleNextRun.Record(ctx, delta, metric.WithAttributes(attrs...))
		}
	case "task_schedule_enqueue_skipped":
		store.scheduleEnqueueSkipped.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
}

func toAttrs(fields map[string]any) []attribute.KeyValue {
	attrs := make([]attribute.KeyValue, 0, len(fields))
	for k, v := range fields {
		switch tv := v.(type) {
		case string:
			attrs = append(attrs, attribute.String(k, tv))
		case bool:
			attrs = append(attrs, attribute.Bool(k, tv))
		case int:
			attrs = append(attrs, attribute.Int(k, tv))
		case int64:
			attrs = append(attrs, attribute.Int64(k, tv))
		case float64:
			attrs = append(attrs, attribute.Float64(k, tv))
		}
	}
	return attrs
}

func toInt64(v any) (int64, bool) {
	switch tv := v.(type) {
	case int64:
		return tv, true
	case int:
		return int64(tv), true
	case float64:
		return int64(tv), true
	case float32:
		return int64(tv), true
	default:
		return 0, false
	}
}
