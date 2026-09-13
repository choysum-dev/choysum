// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package backendtsparser

import (
	"testing"
	"time"
)

func TestSemanticMetrics_RecordSnapshotAndReset(t *testing.T) {
	ResetSemanticMetrics()
	t.Cleanup(ResetSemanticMetrics)

	recordSemanticCacheHit()
	recordSemanticCacheHit()
	recordSemanticBuild(1500 * time.Millisecond)

	snap := SnapshotSemanticMetrics()
	if snap.CacheHitCount != 2 {
		t.Fatalf("CacheHitCount=%d want 2", snap.CacheHitCount)
	}
	if snap.BuildCount != 1 {
		t.Fatalf("BuildCount=%d want 1", snap.BuildCount)
	}
	if snap.BuildMs < 1500 {
		t.Fatalf("BuildMs=%d want >=1500", snap.BuildMs)
	}

	resetSnap := SnapshotAndResetSemanticMetrics()
	if resetSnap.BuildCount != 1 || resetSnap.CacheHitCount != 2 {
		t.Fatalf("reset snap=%+v", resetSnap)
	}
	empty := SnapshotSemanticMetrics()
	if empty.BuildCount != 0 || empty.CacheHitCount != 0 || empty.BuildMs != 0 {
		t.Fatalf("after reset got %+v", empty)
	}
}

func TestSemanticMetrics_NegativeBuildElapsedClamped(t *testing.T) {
	ResetSemanticMetrics()
	t.Cleanup(ResetSemanticMetrics)

	recordSemanticBuild(-50 * time.Millisecond)
	snap := SnapshotSemanticMetrics()
	if snap.BuildCount != 1 {
		t.Fatalf("BuildCount=%d want 1", snap.BuildCount)
	}
	if snap.BuildMs != 0 {
		t.Fatalf("BuildMs=%d want 0 for negative elapsed", snap.BuildMs)
	}
}

func TestSemanticMetrics_TestHelpers(t *testing.T) {
	ResetSemanticMetrics()
	t.Cleanup(ResetSemanticMetrics)

	RecordSemanticCacheHitForTest()
	RecordSemanticBuildForTest(5 * time.Millisecond)
	snap := SnapshotSemanticMetrics()
	if snap.CacheHitCount != 1 || snap.BuildCount != 1 {
		t.Fatalf("test helpers snap=%+v", snap)
	}
}
