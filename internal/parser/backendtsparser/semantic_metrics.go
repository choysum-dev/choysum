// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package backendtsparser

import (
	"sync/atomic"
	"time"
)

// SemanticMetricsSnapshot is a point-in-time view of semantic program cache/build
// activity since the last reset. Used by lifecycle ops for DEBUG summaries.
type SemanticMetricsSnapshot struct {
	BuildCount    int64
	CacheHitCount int64
	BuildMs       int64
}

var (
	semanticBuildCount    atomic.Int64
	semanticCacheHitCount atomic.Int64
	semanticBuildNanos    atomic.Int64
)

func recordSemanticCacheHit() {
	semanticCacheHitCount.Add(1)
}

func recordSemanticBuild(elapsed time.Duration) {
	if elapsed < 0 {
		elapsed = 0
	}
	semanticBuildCount.Add(1)
	semanticBuildNanos.Add(elapsed.Nanoseconds())
}

// SnapshotSemanticMetrics returns current counters without resetting them.
func SnapshotSemanticMetrics() SemanticMetricsSnapshot {
	return SemanticMetricsSnapshot{
		BuildCount:    semanticBuildCount.Load(),
		CacheHitCount: semanticCacheHitCount.Load(),
		BuildMs:       semanticBuildNanos.Load() / int64(time.Millisecond),
	}
}

// ResetSemanticMetrics clears semantic program counters.
func ResetSemanticMetrics() {
	semanticBuildCount.Store(0)
	semanticCacheHitCount.Store(0)
	semanticBuildNanos.Store(0)
}

// SnapshotAndResetSemanticMetrics returns current counters and clears them.
func SnapshotAndResetSemanticMetrics() SemanticMetricsSnapshot {
	snap := SnapshotSemanticMetrics()
	ResetSemanticMetrics()
	return snap
}
