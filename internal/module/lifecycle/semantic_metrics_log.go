// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"context"
	"log/slog"
	"time"

	logutil "github.com/choysum-dev/choysum/internal/logger"
	"github.com/choysum-dev/choysum/internal/parser/backendtsparser"
)

func resetSemanticMetricsForOp() {
	backendtsparser.ResetSemanticMetrics()
}

// logSemanticMetricsSummary emits a DEBUG snapshot of semantic program activity for
// the just-finished lifecycle op, then resets counters for the next op.
func logSemanticMetricsSummary(logger *slog.Logger, opid string) {
	snap := backendtsparser.SnapshotAndResetSemanticMetrics()
	if logger == nil {
		return
	}
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

// phaseEndInfoMinDuration is the wall-time floor for PhaseEnd logs at Info.
const phaseEndInfoMinDuration = time.Second

// logFinalizingPhaseEnd logs aggregate PhaseEnd duration; Info when >=1s
// (demoted under QuietProgress), else Debug.
func logFinalizingPhaseEnd(ctx context.Context, logger *slog.Logger, phaseEndDuration time.Duration) {
	if logger == nil {
		return
	}
	attrs := []any{"step", moduleStepPhaseEnd, "duration_ms", phaseEndDuration.Milliseconds()}
	if phaseEndDuration >= phaseEndInfoMinDuration {
		logger.Log(ctx, logutil.ProgressBoundaryLogLevel(ctx), "module operation finalizing phase end completed", attrs...)
		return
	}
	logger.Debug("module operation finalizing phase end completed", attrs...)
}

// logFinalizingPhaseEndModule logs one module's PhaseEnd script run; Info when >=1s
// (demoted under QuietProgress), else Debug.
func logFinalizingPhaseEndModule(ctx context.Context, logger *slog.Logger, phaseModule string, duration time.Duration) {
	if logger == nil {
		return
	}
	attrs := []any{
		"step", moduleStepPhaseEnd,
		"phase_module", phaseModule,
		"duration_ms", duration.Milliseconds(),
	}
	msg := "module operation finalizing phase end module completed"
	if duration >= phaseEndInfoMinDuration {
		logger.Log(ctx, logutil.ProgressBoundaryLogLevel(ctx), msg, attrs...)
		return
	}
	logger.Debug(msg, attrs...)
}
