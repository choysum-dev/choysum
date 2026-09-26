// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { ChoyChartSeries, ChoyChartSort, ChoyChartSpec } from './chart/chartTypeAdapter';
import { CHOY_CHART_FALLBACK_COLOR } from '../vendor/ui/chart/chartTypes';

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
 * Coerces unknown values to a finite number (Infinity/NaN → 0).
 */
function toFiniteNumber(v: unknown): number {
  const n = Number(v);
  return Number.isFinite(n) ? n : 0;
}

/**
 * Normalizes each category column to percentages that sum to 100 (or 0).
 */
export function normalizeSeriesToPercent(
  categories: string[],
  seriesMatrix: ChoyChartSeries[],
): ChoyChartSeries[] {
  // Negative values cannot form a 0–100 percent stack; clamp them out of totals
  // (same product rule as pieAdapter dropping non-positive slices).
  const positive = (v: unknown): number => Math.max(0, toFiniteNumber(v));
  const totals = categories.map((_, idx) =>
    seriesMatrix.reduce((sum, s) => sum + positive(s.data?.[idx]), 0),
  );
  // Largest-remainder to 2 dp so each column sums to exactly 100 (or 0).
  const columns = categories.map((_, idx) => {
    const total = totals[idx] || 0;
    const n = seriesMatrix.length;
    if (!total || n === 0) return seriesMatrix.map(() => 0);
    const raw = seriesMatrix.map(s => (10000 * positive(s.data?.[idx])) / total);
    const hundredths = raw.map(v => Math.floor(v));
    const order = raw
      .map((v, si) => ({ si, frac: v - (hundredths[si] ?? 0) }))
      .sort((a, b) => b.frac - a.frac || a.si - b.si);
    let remainder = 10000 - hundredths.reduce((a, b) => a + b, 0);
    for (let i = 0; i < remainder && i < order.length; i++) {
      const si = order[i]!.si;
      hundredths[si] = (hundredths[si] ?? 0) + 1;
    }
    return hundredths.map(v => v / 100);
  });
  return seriesMatrix.map((s, si) => ({
    name: s.name,
    data: categories.map((_, cIdx) => columns[cIdx]![si] ?? 0),
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
    seriesMatrix.reduce((sum, s) => sum + toFiniteNumber(s.data?.[idx]), 0),
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
      data: order.map(i => toFiniteNumber(s.data?.[i])),
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
    color: spec.config[sl.key]?.color || CHOY_CHART_FALLBACK_COLOR,
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
  if (typeof rawIndex === 'string' && rawIndex.trim() !== '') {
    const parsed = Number(rawIndex);
    if (Number.isFinite(parsed)) return Math.round(parsed);
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
    const derivedSeries = ((flat % seriesCount) + seriesCount) % seriesCount;
    const derivedCategory = Math.floor(flat / seriesCount);
    if (categoryIdx == null) {
      // No row index available: the flat element index determines both.
      categoryIdx = derivedCategory;
      seriesIdx = derivedSeries;
    } else if (derivedCategory === categoryIdx) {
      seriesIdx = derivedSeries;
    }
    // Otherwise the event index disagrees with the datum: leave seriesIdx
    // undefined rather than attributing a series from another category.
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
  // Prefer the row's own index (always set by chartSpecToXyRows); the event
  // index is only a fallback when the datum did not carry one.
  const categoryIdx =
    rowCategoryIndex(row) ??
    (typeof categoryIdxFromEvent === 'number' && Number.isFinite(categoryIdxFromEvent)
      ? Math.round(categoryIdxFromEvent)
      : null);
  const seriesIdx =
    typeof stackIndex === 'number' && Number.isFinite(stackIndex)
      ? Math.round(stackIndex)
      : undefined;
  return { categoryIdx, seriesIdx };
}

/**
 * Maps a relative pointer position (0–1) along the plot width to the nearest
 * category index; pure so the line-click math stays unit-testable.
 */
export function resolveLineClickCategory(rel: number, count: number): number {
  if (count <= 1) return 0;
  if (!Number.isFinite(rel)) return 0;
  const clamped = Math.min(1, Math.max(0, rel));
  return Math.round(clamped * (count - 1));
}

/**
 * Client X for Unovis click events. Touch uses `changedTouches`; mouse/pointer
 * use `clientX` on the event itself.
 */
export function resolveClickClientX(event: {
  clientX?: number;
  changedTouches?: ArrayLike<{ clientX?: number }>;
}): number | undefined {
  const touches = event.changedTouches;
  if (touches && touches.length > 0) {
    const x = touches[0]?.clientX;
    if (typeof x === 'number' && Number.isFinite(x)) return x;
  }
  const x = event.clientX;
  return typeof x === 'number' && Number.isFinite(x) ? x : undefined;
}
