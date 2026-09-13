// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/choysum-dev/choysum/internal/parser/backendtsparser"
)

func TestResetSemanticMetricsForOpClearsCounters(t *testing.T) {
	backendtsparser.ResetSemanticMetrics()
	t.Cleanup(backendtsparser.ResetSemanticMetrics)

	backendtsparser.RecordSemanticBuildForTest(10 * time.Millisecond)
	backendtsparser.RecordSemanticCacheHitForTest()
	resetSemanticMetricsForOp()

	snap := backendtsparser.SnapshotSemanticMetrics()
	if snap.BuildCount != 0 || snap.CacheHitCount != 0 || snap.BuildMs != 0 {
		t.Fatalf("after reset got %+v", snap)
	}
}

func TestLogSemanticMetricsSummary(t *testing.T) {
	backendtsparser.ResetSemanticMetrics()
	t.Cleanup(backendtsparser.ResetSemanticMetrics)

	t.Run("nil logger", func(t *testing.T) {
		backendtsparser.RecordSemanticCacheHitForTest()
		logSemanticMetricsSummary(nil, "op-1")
		snap := backendtsparser.SnapshotSemanticMetrics()
		if snap.CacheHitCount != 0 {
			t.Fatalf("nil logger still resets counters, got %+v", snap)
		}
	})

	t.Run("empty snapshot skips log", func(t *testing.T) {
		var buf bytes.Buffer
		logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
		logSemanticMetricsSummary(logger, "op-empty")
		if buf.Len() != 0 {
			t.Fatalf("expected no log for empty metrics, got %q", buf.String())
		}
	})

	t.Run("emits with opid", func(t *testing.T) {
		backendtsparser.ResetSemanticMetrics()
		backendtsparser.RecordSemanticBuildForTest(25 * time.Millisecond)
		backendtsparser.RecordSemanticCacheHitForTest()

		var buf bytes.Buffer
		logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
		logSemanticMetricsSummary(logger, "op-123")

		logs := buf.String()
		for _, want := range []string{
			`"msg":"lifecycle semantic program metrics"`,
			`"opid":"op-123"`,
			`"semantic_build_count":1`,
			`"semantic_cache_hit_count":1`,
			`"semantic_build_ms":`,
		} {
			if !strings.Contains(logs, want) {
				t.Fatalf("expected %q in %q", want, logs)
			}
		}
		empty := backendtsparser.SnapshotSemanticMetrics()
		if empty.BuildCount != 0 || empty.CacheHitCount != 0 {
			t.Fatalf("summary must reset counters, got %+v", empty)
		}
	})

	t.Run("emits without opid", func(t *testing.T) {
		backendtsparser.ResetSemanticMetrics()
		backendtsparser.RecordSemanticCacheHitForTest()

		var buf bytes.Buffer
		logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
		logSemanticMetricsSummary(logger, "")

		logs := buf.String()
		if !strings.Contains(logs, `"semantic_cache_hit_count":1`) {
			t.Fatalf("expected cache hit in %q", logs)
		}
		if strings.Contains(logs, `"opid"`) {
			t.Fatalf("did not expect opid in %q", logs)
		}
	})
}

func TestLogFinalizingPhaseEnd(t *testing.T) {
	t.Run("nil logger", func(t *testing.T) {
		logFinalizingPhaseEnd(nil, time.Second)
	})

	t.Run("info when at least one second", func(t *testing.T) {
		var buf bytes.Buffer
		logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
		logFinalizingPhaseEnd(logger, time.Second)

		logs := buf.String()
		if !strings.Contains(logs, `"level":"INFO"`) {
			t.Fatalf("expected INFO, got %q", logs)
		}
		for _, want := range []string{
			`"msg":"module operation finalizing phase end completed"`,
			`"step":"phase_end"`,
			`"duration_ms":1000`,
		} {
			if !strings.Contains(logs, want) {
				t.Fatalf("expected %q in %q", want, logs)
			}
		}
	})

	t.Run("debug when under one second", func(t *testing.T) {
		var buf bytes.Buffer
		logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
		logFinalizingPhaseEnd(logger, 250*time.Millisecond)

		logs := buf.String()
		if !strings.Contains(logs, `"level":"DEBUG"`) {
			t.Fatalf("expected DEBUG, got %q", logs)
		}
		if !strings.Contains(logs, `"step":"phase_end"`) {
			t.Fatalf("expected phase_end step in %q", logs)
		}
	})
}
