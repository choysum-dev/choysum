// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"log/slog"

	"github.com/choysum-dev/choysum/internal/parser/backendtsparser"
)

func resetSemanticMetricsForOp() {
	backendtsparser.ResetSemanticMetrics()
}

// logSemanticMetricsSummary emits a DEBUG snapshot of semantic program activity for
// the just-finished lifecycle op, then resets counters for the next op.
func logSemanticMetricsSummary(logger *slog.Logger, opid string) {
	if logger == nil {
		return
	}
	snap := backendtsparser.SnapshotAndResetSemanticMetrics()
	if snap.BuildCount == 0 && snap.CacheHitCount == 0 {
		return
	}
	attrs := []any{
		"semantic_build_count", snap.BuildCount,
		"semantic_cache_hit_count", snap.CacheHitCount,
		"semantic_build_ms", snap.BuildMs,
	}
	if opid != "" {
		attrs = append([]any{"opid", opid}, attrs...)
	}
	logger.Debug("lifecycle semantic program metrics", attrs...)
}
