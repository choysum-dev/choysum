// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { GroupRow } from '@/web/web/query/types';
import type { ChoyChartSeries } from './chart/chartTypeAdapter';
import { parseChartCategoryTime } from './chartViewHelpers';

type ChartGroupLike =
  | GroupRow
  | {
      kind?: string;
      depth?: number;
      key?: string;
      label?: string;
      count?: number;
      metrics?: Record<string, unknown>;
      children?: ChartGroupLike[];
    };

function isGroupRow(row: ChartGroupLike): boolean {
  return (row as GroupRow).kind === 'group';
}

function metricValue(row: ChartGroupLike, alias: string): number {
  if (alias === 'count') {
    const n = Number(
      (row as GroupRow).count ?? (row as GroupRow).metrics?.count ?? 0,
    );
    return Number.isFinite(n) ? n : 0;
  }
  const n = Number((row as GroupRow).metrics?.[alias] ?? 0);
  return Number.isFinite(n) ? n : 0;
}

function groupLabel(row: ChartGroupLike): string {
  return String((row as GroupRow).label ?? (row as GroupRow).key ?? '');
}

function childGroups(row: ChartGroupLike): ChartGroupLike[] {
  const kids = (row as GroupRow).children;
  if (!Array.isArray(kids)) return [];
  return kids.filter((c) => isGroupRow(c));
}

/** Prefer raw group key (ISO bucket), then display label, for temporal ordering. */
function groupTimeSortValue(row: ChartGroupLike): number | undefined {
  const key = String((row as GroupRow).key ?? '');
  const fromKey = parseChartCategoryTime(key);
  if (fromKey != null) return fromKey;
  return parseChartCategoryTime(groupLabel(row));
}

/**
 * When most category groups carry a parseable time key/label, sort ascending
 * so the X axis follows the timeline even if the snapshot order is shuffled.
 */
function orderParentsForChart(parents: ChartGroupLike[]): ChartGroupLike[] {
  if (parents.length <= 1) return parents.slice();
  const timed = parents.map((p, idx) => ({ idx, t: groupTimeSortValue(p) }));
  const parsed = timed.filter((x) => x.t != null).length;
  if (parsed * 2 < parents.length) return parents.slice();
  return timed
    .slice()
    .sort((a, b) => {
      if (a.t != null && b.t != null) return a.t - b.t || a.idx - b.idx;
      if (a.t != null) return -1;
      if (b.t != null) return 1;
      return a.idx - b.idx;
    })
    .map((x) => parents[x.idx]!);
}

/**
 * Map group rows from a chart controller snapshot into chart categories and
 * series. With nested children on category groups, each child label becomes a
 * series (Odoo-style second groupby). Otherwise a single metric series is used.
 */
export function groupRowsToChartSeries(
  rows: ReadonlyArray<ChartGroupLike>,
  opts: { metricAlias?: string; metricLabel?: string; groupDepth?: number } = {},
): { categories: string[]; seriesMatrix: ChoyChartSeries[] } {
  const depth = opts.groupDepth ?? 0;
  const alias = String(opts.metricAlias || 'count').trim() || 'count';
  const label = opts.metricLabel || alias;
  const parents = orderParentsForChart(
    rows.filter(
      (r) => isGroupRow(r) && Number((r as GroupRow).depth ?? 0) === depth,
    ),
  );

  const nested = parents.some((p) => childGroups(p).length > 0);
  if (nested) {
    const seriesNames: string[] = [];
    const seen = new Set<string>();
    for (const parent of parents) {
      for (const child of childGroups(parent)) {
        const name = groupLabel(child);
        if (seen.has(name)) continue;
        seen.add(name);
        seriesNames.push(name);
      }
    }
    const categories = parents.map((p) => groupLabel(p));
    const seriesMatrix: ChoyChartSeries[] = seriesNames.map((name) => ({
      name,
      data: parents.map((parent) => {
        const child = childGroups(parent).find((c) => groupLabel(c) === name);
        return child ? metricValue(child, alias) : 0;
      }),
    }));
    return {
      categories,
      seriesMatrix: categories.length && seriesMatrix.length ? seriesMatrix : [],
    };
  }

  const categories: string[] = [];
  const data: number[] = [];
  for (const g of parents) {
    categories.push(groupLabel(g));
    data.push(metricValue(g, alias));
  }
  return {
    categories,
    seriesMatrix: categories.length ? [{ name: label, data }] : [],
  };
}
