// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { ChoyChartSeries, ChoyChartSort, ChoyChartSpec } from './chart/chartTypeAdapter';

export type ChoyChartMetricOption = {
  alias: string;
  label: string;
};

export type ChoyChartItemClickPayload = {
  chartType: string;
  category?: string;
  seriesName?: string;
  value?: number;
  categoryIndex?: number;
  seriesIndex?: number;
  path?: string[];
};

/**
 * Normalizes each category column to percentages that sum to 100 (or 0).
 */
export function normalizeSeriesToPercent(
  categories: string[],
  seriesMatrix: ChoyChartSeries[],
): ChoyChartSeries[] {
  const totals = categories.map((_, idx) =>
    seriesMatrix.reduce((sum, s) => sum + (Number(s.data[idx]) || 0), 0),
  );
  return seriesMatrix.map(s => ({
    name: s.name,
    data: (s.data || []).map((v, idx) => {
      const total = totals[idx] || 0;
      if (!total) return 0;
      return (100 * (Number(v) || 0)) / total;
    }),
  }));
}

/**
 * Sorts categories (and aligned series values) by Y total ascending/descending.
 */
export function sortChartCategories(
  categories: string[],
  seriesMatrix: ChoyChartSeries[],
  sort: ChoyChartSort,
): { categories: string[]; seriesMatrix: ChoyChartSeries[] } {
  if (sort === 'none' || categories.length === 0) {
    return {
      categories: categories.slice(),
      seriesMatrix: seriesMatrix.map(s => ({ name: s.name, data: (s.data || []).slice() })),
    };
  }
  const totals = categories.map((_, idx) =>
    seriesMatrix.reduce((sum, s) => sum + (Number(s.data[idx]) || 0), 0),
  );
  const order = categories
    .map((_, idx) => idx)
    .sort((a, b) => {
      const diff = (totals[a] || 0) - (totals[b] || 0);
      return sort === 'asc' ? diff : -diff;
    });
  return {
    categories: order.map(i => categories[i]!),
    seriesMatrix: seriesMatrix.map(s => ({
      name: s.name,
      data: order.map(i => Number(s.data[i]) || 0),
    })),
  };
}

/**
 * Flattens a ChoyChartSpec into Unovis XY row objects (`category` + series keys).
 */
export function chartSpecToXyRows(
  spec: ChoyChartSpec,
): Array<Record<string, string | number>> {
  return spec.categories.map((category, idx) => {
    const row: Record<string, string | number> = { category, index: idx };
    for (const s of spec.series) {
      row[s.key] = s.values[idx] ?? 0;
    }
    return row;
  });
}

/**
 * Builds pie rows for VisDonut (`name`, `value`, `color`, `key`).
 */
export function chartSpecToPieRows(
  spec: ChoyChartSpec,
): Array<{ key: string; name: string; value: number; color: string }> {
  const slices = spec.slices || [];
  return slices.map(sl => ({
    key: sl.key,
    name: sl.name,
    value: sl.value,
    color: spec.config[sl.key]?.color || 'var(--choy-chart-1)',
  }));
}

function rowCategoryIndex(
  row: Record<string, string | number> | undefined,
): number | null {
  if (!row) return null;
  const rawIndex = row.index;
  if (typeof rawIndex === 'number' && Number.isFinite(rawIndex)) {
    return Math.round(rawIndex);
  }
  if (
    typeof rawIndex === 'string' &&
    rawIndex !== '' &&
    Number.isFinite(Number(rawIndex))
  ) {
    return Math.round(Number(rawIndex));
  }
  return null;
}

/**
 * Maps a GroupedBar Unovis click (row + flat element index) to category/series.
 */
export function resolveGroupedXyClickTarget(
  row: Record<string, string | number> | undefined,
  flatIndex: number | undefined,
  seriesCount: number,
): { categoryIdx: number | null; seriesIdx: number | undefined } {
  let categoryIdx = rowCategoryIndex(row);
  let seriesIdx: number | undefined;
  if (
    typeof flatIndex === 'number' &&
    Number.isFinite(flatIndex) &&
    seriesCount > 0
  ) {
    const flat = Math.round(flatIndex);
    seriesIdx = ((flat % seriesCount) + seriesCount) % seriesCount;
    if (categoryIdx == null) {
      categoryIdx = Math.floor(flat / seriesCount);
    }
  }
  if (seriesCount === 1) seriesIdx = 0;
  return { categoryIdx, seriesIdx };
}

/**
 * Maps a StackedBar Unovis click (mapped datum with stackIndex + category index).
 */
export function resolveStackedXyClickTarget(
  row: Record<string, string | number> | undefined,
  categoryIdxFromEvent: number | undefined,
  stackIndex: number | undefined,
): { categoryIdx: number | null; seriesIdx: number | undefined } {
  const categoryIdx =
    typeof categoryIdxFromEvent === 'number' && Number.isFinite(categoryIdxFromEvent)
      ? Math.round(categoryIdxFromEvent)
      : rowCategoryIndex(row);
  const seriesIdx =
    typeof stackIndex === 'number' && Number.isFinite(stackIndex)
      ? Math.round(stackIndex)
      : undefined;
  return { categoryIdx, seriesIdx };
}
