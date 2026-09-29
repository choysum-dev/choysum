// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { GroupRow } from '@/web/web/query/types';
import type { ChoyChartSeries } from './chart/chartTypeAdapter';

/**
 * Map top-level group rows from a chart controller snapshot into chart categories
 * and a single metric series (count or named aggregate alias).
 */
export function groupRowsToChartSeries(
  rows: ReadonlyArray<GroupRow | { kind?: string; depth?: number; label?: string; count?: number; metrics?: Record<string, unknown> }>,
  opts: { metricAlias?: string; metricLabel?: string; groupDepth?: number } = {},
): { categories: string[]; seriesMatrix: ChoyChartSeries[] } {
  const depth = opts.groupDepth ?? 0;
  const alias = String(opts.metricAlias || 'count').trim() || 'count';
  const label = opts.metricLabel || alias;
  const groups = rows.filter(
    (r) => (r as GroupRow).kind === 'group' && Number((r as GroupRow).depth ?? 0) === depth,
  );
  const categories: string[] = [];
  const data: number[] = [];
  for (const g of groups) {
    const row = g as GroupRow;
    categories.push(String(row.label ?? row.key ?? ''));
    if (alias === 'count') {
      const n = Number(row.count ?? row.metrics?.count ?? 0);
      data.push(Number.isFinite(n) ? n : 0);
    } else {
      const n = Number(row.metrics?.[alias] ?? 0);
      data.push(Number.isFinite(n) ? n : 0);
    }
  }
  return {
    categories,
    seriesMatrix: categories.length ? [{ name: label, data }] : [],
  };
}
