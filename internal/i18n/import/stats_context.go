// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package i18nimport

import "context"

type statsAccumulatorKey struct{}

// ContextWithStatsAccumulator attaches an ImportStats sink for packaged PO writes.
// Callers (e.g. hotreload) sum Upserted / SkippedOverride / RejectedNoCtxt across files.
func ContextWithStatsAccumulator(ctx context.Context, stats *ImportStats) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if stats == nil {
		return ctx
	}
	return context.WithValue(ctx, statsAccumulatorKey{}, stats)
}

// AccumulateImportStats merges one PO file's ImportStats into a context sink when present.
func AccumulateImportStats(ctx context.Context, stats *ImportStats) {
	if ctx == nil || stats == nil {
		return
	}
	acc, ok := ctx.Value(statsAccumulatorKey{}).(*ImportStats)
	if !ok || acc == nil {
		return
	}
	acc.Upserted += stats.Upserted
	acc.SkippedOverride += stats.SkippedOverride
	acc.RejectedNoCtxt += stats.RejectedNoCtxt
	acc.SkippedObsolete += stats.SkippedObsolete
	acc.PurgedRetired += stats.PurgedRetired
	if acc.Lang == "" {
		acc.Lang = stats.Lang
	}
}
