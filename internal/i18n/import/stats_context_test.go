// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package i18nimport

import (
	"context"
	"testing"
)

func TestAccumulateImportStats_sumsIntoContextSink(t *testing.T) {
	t.Parallel()
	acc := &ImportStats{}
	ctx := ContextWithStatsAccumulator(context.Background(), acc)
	AccumulateImportStats(ctx, &ImportStats{
		Upserted:        2,
		SkippedOverride: 1,
		RejectedNoCtxt:  3,
		SkippedObsolete: 4,
		PurgedRetired:   5,
		Lang:            "zh_CN",
	})
	AccumulateImportStats(ctx, &ImportStats{Upserted: 1, Lang: "en_US"})
	if acc.Upserted != 3 || acc.SkippedOverride != 1 || acc.RejectedNoCtxt != 3 {
		t.Fatalf("unexpected acc: %+v", acc)
	}
	if acc.Lang != "zh_CN" {
		t.Fatalf("Lang = %q, want first non-empty", acc.Lang)
	}
}

func TestAccumulateImportStats_noSinkNoPanic(t *testing.T) {
	t.Parallel()
	AccumulateImportStats(nil, &ImportStats{Upserted: 1})
	AccumulateImportStats(context.Background(), &ImportStats{Upserted: 1})
	AccumulateImportStats(ContextWithStatsAccumulator(context.Background(), nil), &ImportStats{Upserted: 1})
}

func TestPropagateStatsAccumulator_copiesSink(t *testing.T) {
	t.Parallel()
	acc := &ImportStats{}
	src := ContextWithStatsAccumulator(context.Background(), acc)
	dst := PropagateStatsAccumulator(context.Background(), src)
	AccumulateImportStats(dst, &ImportStats{Upserted: 2, Lang: "zh_CN"})
	if acc.Upserted != 2 || acc.Lang != "zh_CN" {
		t.Fatalf("acc=%+v", *acc)
	}
	if PropagateStatsAccumulator(context.Background(), nil) == nil {
		t.Fatal("nil src should return dst")
	}
	if PropagateStatsAccumulator(context.Background(), context.Background()) == nil {
		t.Fatal("no sink should return dst")
	}
}
