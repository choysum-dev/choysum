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
  // Largest-remainder to 2 dp so each column sums to exactly 100 (or 0).
  const columns = categories.map((_, idx) => {
    const n = seriesMatrix.length;
    if (n === 0) return [];
    const positives = seriesMatrix.map(s => positive(s.data?.[idx]));
    // Scale by max before summing so large finite values cannot overflow total.
    let max = 0;
    for (const v of positives) {
      if (v > max) max = v;
    }
    if (!max) return seriesMatrix.map(() => 0);
    const scaled = positives.map(v => v / max);
    const scaledTotal = scaled.reduce((a, b) => a + b, 0);
    if (!scaledTotal) return seriesMatrix.map(() => 0);
    // Divide before the 10000 factor to avoid numerator overflow.
    const raw = scaled.map(v => 10000 * (v / scaledTotal));
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
 * `order[sortedIdx]` is the original category index before sorting.
 */
export function sortChartCategories(
  categories: string[],
  seriesMatrix: ChoyChartSeries[],
  sort: ChoyChartSort,
): { categories: string[]; seriesMatrix: ChoyChartSeries[]; order: number[] } {
  if (sort === 'none' || categories.length === 0) {
    return {
      categories: categories.slice(),
      seriesMatrix: seriesMatrix.map(s => ({
        name: s.name,
        data: (s.data || []).map(toFiniteNumber),
      })),
      order: categories.map((_, idx) => idx),
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
    order,
  };
}

const KNOWN_TEMPORAL_GRANS = new Set(['year', 'quarter', 'month', 'week', 'day']);

/**
 * True when the first groupby dimension is a temporal granularity
 * (`CreatedAt:month` or `{ field, granularity: 'month' }`).
 */
export function chartCategoryAxisIsTemporal(groups: unknown): boolean {
  const list = Array.isArray(groups) ? groups : groups != null ? [groups] : [];
  const first = list[0];
  if (first == null || first === '') return false;
  if (typeof first === 'string') {
    const idx = first.indexOf(':');
    if (idx <= 0) return false;
    const gran = first.slice(idx + 1).trim().toLowerCase();
    return KNOWN_TEMPORAL_GRANS.has(gran);
  }
  if (typeof first === 'object') {
    const g = first as { field?: unknown; granularity?: unknown };
    const gran = String(g.granularity || '').trim().toLowerCase();
    if (KNOWN_TEMPORAL_GRANS.has(gran)) return true;
    const field = String(g.field || '');
    const idx = field.indexOf(':');
    if (idx > 0) {
      const maybe = field.slice(idx + 1).trim().toLowerCase();
      return KNOWN_TEMPORAL_GRANS.has(maybe);
    }
  }
  return false;
}

/**
 * Parse a chart category label/key into a sortable epoch ms.
 * Supports ORM display labels (YYYY, YYYY-Qn, YYYY-MM, GGGG-Www, YYYY-MM-DD)
 * and ISO / Date-parseable keys.
 */
export function parseChartCategoryTime(raw: string): number | undefined {
  const s = String(raw || '').trim();
  if (!s) return undefined;
  // Composite group keys use `|`; sort by the first segment (category axis).
  const head = s.includes('|') ? s.slice(0, s.indexOf('|')) : s;

  let m = /^(\d{4})$/.exec(head);
  if (m) return Date.UTC(Number(m[1]), 0, 1);

  m = /^(\d{4})-Q([1-4])$/i.exec(head);
  if (m) return Date.UTC(Number(m[1]), (Number(m[2]) - 1) * 3, 1);

  m = /^(\d{4})-(\d{2})$/.exec(head);
  if (m) {
    const month = Number(m[2]);
    if (month >= 1 && month <= 12) return Date.UTC(Number(m[1]), month - 1, 1);
  }

  m = /^(\d{4})-W(\d{2})$/i.exec(head);
  if (m) {
    // ISO week: Thursday of week 1 is in the ISO year; approximate via UTC.
    const year = Number(m[1]);
    const week = Number(m[2]);
    if (week >= 1 && week <= 53) {
      const jan4 = new Date(Date.UTC(year, 0, 4));
      const day = jan4.getUTCDay() || 7;
      const monday = new Date(jan4);
      monday.setUTCDate(jan4.getUTCDate() - day + 1 + (week - 1) * 7);
      return monday.getTime();
    }
  }

  m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(head);
  if (m) {
    const month = Number(m[2]);
    const day = Number(m[3]);
    if (month >= 1 && month <= 12 && day >= 1 && day <= 31) {
      return Date.UTC(Number(m[1]), month - 1, day);
    }
  }

  const t = Date.parse(head);
  return Number.isFinite(t) ? t : undefined;
}

function reorderChartSeries(
  categories: string[],
  seriesMatrix: ChoyChartSeries[],
  order: number[],
): { categories: string[]; seriesMatrix: ChoyChartSeries[]; order: number[] } {
  return {
    categories: order.map(i => categories[i]!),
    seriesMatrix: seriesMatrix.map(s => ({
      name: s.name,
      data: order.map(i => toFiniteNumber(s.data?.[i])),
    })),
    order,
  };
}

/**
 * Ascending time order for category labels/keys. Unparsed labels keep relative
 * order after parsed ones (stable).
 */
export function sortChartCategoriesByTime(
  categories: string[],
  seriesMatrix: ChoyChartSeries[],
): { categories: string[]; seriesMatrix: ChoyChartSeries[]; order: number[] } {
  if (categories.length === 0) {
    return { categories: [], seriesMatrix: [], order: [] };
  }
  const times = categories.map(c => parseChartCategoryTime(c));
  const order = categories
    .map((_, idx) => idx)
    .sort((a, b) => {
      const ta = times[a];
      const tb = times[b];
      if (ta != null && tb != null) return ta - tb || a - b;
      if (ta != null) return -1;
      if (tb != null) return 1;
      return a - b;
    });
  return reorderChartSeries(categories, seriesMatrix, order);
}

/**
 * Running total along the category axis for each series (Odoo Cumulative).
 */
export function accumulateChartSeries(
  categories: string[],
  seriesMatrix: ChoyChartSeries[],
): ChoyChartSeries[] {
  return seriesMatrix.map(s => {
    let run = 0;
    return {
      name: s.name,
      data: categories.map((_, i) => {
        run += toFiniteNumber(s.data?.[i]);
        return run;
      }),
    };
  });
}

/**
 * Y domain for stacked charts. Unovis sizes the axis from each series max, which
 * clips the stack total; this uses the per-category sum plus a small headroom.
 */
export function resolveStackedYDomain(
  rows: ReadonlyArray<Record<string, string | number>>,
  seriesKeys: readonly string[],
  opts?: { percent?: boolean },
): [number, number] {
  if (opts?.percent) return [0, 100];
  let max = 0;
  for (const row of rows) {
    let sum = 0;
    for (const key of seriesKeys) {
      const n = Number(row[key] ?? 0);
      if (Number.isFinite(n) && n > 0) sum += n;
    }
    if (sum > max) max = sum;
  }
  if (max <= 0) return [0, 1];
  const padded = max * 1.08;
  const mag = 10 ** Math.floor(Math.log10(padded));
  const step = mag >= 10 ? mag / 2 : mag;
  const nice = Math.ceil(padded / step) * step;
  return [0, nice];
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

/** Category index carried on an XY row (`index` from chartSpecToXyRows). */
export function rowCategoryIndex(
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
 * Equal-width category columns along a padded bar domain (band centers).
 * For flush line domains prefer {@link resolveLineClickCategory} (nearest vertex).
 */
export function resolveLineHoverCategory(rel: number, count: number): number {
  if (count <= 1) return 0;
  if (!Number.isFinite(rel)) return 0;
  const clamped = Math.min(0.999999, Math.max(0, rel));
  return Math.min(count - 1, Math.floor(clamped * count));
}

/**
 * Hover band for a flush line domain (`[0, n-1]`): Voronoi column around vertex `i`.
 * Coordinates are relative to the plot host (same space as the band overlay).
 */
export function resolveLineFlushHoverBand(input: {
  categoryIdx: number;
  categoryCount: number;
  plotLeft: number;
  plotTop: number;
  plotWidth: number;
  plotHeight: number;
  hostLeft: number;
  hostTop: number;
}): { left: number; top: number; width: number; height: number } {
  const n = Math.max(1, Math.floor(Number(input.categoryCount) || 0));
  const idx = Math.min(n - 1, Math.max(0, Math.floor(Number(input.categoryIdx) || 0)));
  const plotW = Math.max(0, Number(input.plotWidth) || 0);
  const plotH = Math.max(0, Number(input.plotHeight) || 0);
  let leftFrac = 0;
  let rightFrac = 1;
  if (n > 1) {
    const center = idx / (n - 1);
    const prev = idx === 0 ? 0 : (idx - 1) / (n - 1);
    const next = idx === n - 1 ? 1 : (idx + 1) / (n - 1);
    leftFrac = idx === 0 ? 0 : (prev + center) / 2;
    rightFrac = idx === n - 1 ? 1 : (center + next) / 2;
  }
  return {
    left: Number(input.plotLeft) - Number(input.hostLeft) + leftFrac * plotW,
    top: Number(input.plotTop) - Number(input.hostTop),
    width: Math.max(1, (rightFrac - leftFrac) * plotW),
    height: plotH,
  };
}

/**
 * Resolve line hover category + nearest series from plot-relative pointer fractions.
 * Uses nearest-vertex X mapping for flush line domains.
 */
export function resolveLinePlotHoverTarget(input: {
  relX: number;
  relY: number;
  categoryCount: number;
  seriesValuesAtCategory: (categoryIdx: number) => number[];
  yDomain: [number, number];
}): { categoryIdx: number; seriesIdx: number } {
  const nCat = Math.max(0, Math.floor(Number(input.categoryCount) || 0));
  const categoryIdx = resolveLineClickCategory(input.relX, nCat);
  const values = input.seriesValuesAtCategory(categoryIdx);
  const seriesIdx =
    values.length <= 1 ? 0 : resolveNearestLineSeriesByY(values, input.relY, input.yDomain);
  return { categoryIdx, seriesIdx };
}

/**
 * Prefer the Unovis line/scatter geometry for the plot box; fall back to the SVG.
 */
export function resolveXyPlotClientRect(svg: Element): {
  left: number;
  top: number;
  width: number;
  height: number;
  right: number;
  bottom: number;
} {
  const plotEl =
    svg.querySelector('.line')?.closest('g') ||
    svg.querySelector('.point')?.closest('g') ||
    svg;
  const rect = plotEl.getBoundingClientRect();
  return {
    left: rect.left,
    top: rect.top,
    width: rect.width,
    height: rect.height,
    right: rect.right,
    bottom: rect.bottom,
  };
}

/**
 * Picks the series whose value at the category is closest to the pointer's
 * data-Y (relY: 0 = plot top, 1 = plot bottom).
 */
export function resolveNearestLineSeriesByY(
  seriesValuesAtCategory: number[],
  relY: number,
  yDomain: [number, number],
): number {
  const n = seriesValuesAtCategory.length;
  if (n <= 1) return 0;
  const y0 = Number(yDomain[0]);
  const y1 = Number(yDomain[1]);
  const lo = Number.isFinite(y0) ? y0 : 0;
  const hi = Number.isFinite(y1) && y1 > lo ? y1 : lo + 1;
  const clamped = Math.min(1, Math.max(0, Number(relY) || 0));
  const dataY = lo + (hi - lo) * (1 - clamped);
  let best = 0;
  let bestDist = Infinity;
  for (let i = 0; i < n; i++) {
    const v = Number(seriesValuesAtCategory[i]);
    const value = Number.isFinite(v) ? v : 0;
    const dist = Math.abs(value - dataY);
    if (dist < bestDist) {
      bestDist = dist;
      best = i;
    }
  }
  return best;
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

/** Default Unovis plot min height (px) when the chart body has not sized yet. */
export const CHOY_CHART_MIN_PLOT_HEIGHT = 280;

/** Space reserved below the plot for ChartLegendContent (px). */
export const CHOY_CHART_LEGEND_RESERVE = 40;

/**
 * Soft cap on bar / group width (px). High enough that few categories stay
 * chunky across a full-width plot; many categories still shrink by step size.
 */
export const CHOY_CHART_BAR_MAX_WIDTH = 280;

/** Stacked / grouped bar padding + max-width for a category count (Odoo-like density). */
export type ChartBarLayout = {
  barMaxWidth: number;
  barPadding: number;
  groupPadding: number;
};

/**
 * Wider bars / less intra-band padding when there are few categories so the
 * plot does not look like two thin sticks in a sea of white space.
 */
export function resolveChartBarLayout(categoryCount: number): ChartBarLayout {
  const n = Math.max(0, Math.floor(Number(categoryCount) || 0));
  if (n <= 2) {
    return { barMaxWidth: 360, barPadding: 0.08, groupPadding: 0.12 };
  }
  if (n <= 4) {
    return { barMaxWidth: 300, barPadding: 0.12, groupPadding: 0.16 };
  }
  if (n <= 8) {
    return { barMaxWidth: CHOY_CHART_BAR_MAX_WIDTH, barPadding: 0.16, groupPadding: 0.2 };
  }
  return { barMaxWidth: CHOY_CHART_BAR_MAX_WIDTH, barPadding: 0.18, groupPadding: 0.2 };
}

/** Line stroke width (px). */
export const CHOY_CHART_LINE_WIDTH = 2.5;

/** Resting scatter marker diameter (px) at line vertices (hover uses a halo). */
export const CHOY_CHART_LINE_POINT_SIZE = 8;

/**
 * Extra px beyond the Unovis plot clip so line vertex markers at x=0 / x=n-1
 * stay fully visible while the domain stays flush (Odoo-like).
 */
export const CHOY_CHART_LINE_CLIP_EXTEND = Math.ceil(CHOY_CHART_LINE_POINT_SIZE / 2) + 4;

/** Area fill under line series (0–1); keep light so the stroke stays primary. */
export const CHOY_CHART_LINE_AREA_OPACITY = 0.2;

/** X-domain strategy for integer category indices. */
export type ChartXDomainKind = 'bar' | 'line';

/**
 * X domain for integer category indices.
 * - bar: half-step outer pad so categories sit in band centers
 * - line: first index at 0 / last at n-1 (Odoo-style, flush to the axes)
 * Returns undefined when there are no categories.
 */
export function resolveChartXDomain(
  categoryCount: number,
  kind: ChartXDomainKind = 'bar',
): [number, number] | undefined {
  const n = Math.floor(Number(categoryCount) || 0);
  if (n <= 0) return undefined;
  if (kind === 'line') {
    if (n === 1) return [-0.5, 0.5];
    return [0, n - 1];
  }
  return [-0.5, n - 0.5];
}

/**
 * Plot height for VisXYContainer / VisSingleContainer: fill the chart body
 * minus legend reserve, never below minHeight.
 */
export function resolveChartPlotHeight(
  bodyHeight: number,
  opts?: { minHeight?: number; legendReserve?: number },
): number {
  const minHeight = opts?.minHeight ?? CHOY_CHART_MIN_PLOT_HEIGHT;
  const legendReserve = opts?.legendReserve ?? CHOY_CHART_LEGEND_RESERVE;
  const usable = Math.floor(Number(bodyHeight) || 0) - legendReserve;
  return Math.max(minHeight, usable);
}

/** One legend-colored value row inside a chart hover tooltip. */
export type ChartTooltipRow = {
  color: string;
  label: string;
  valueText: string;
};

/** Input for {@link formatChartTooltipHtml} (metric title + value rows). */
export type ChartTooltipHtmlInput = {
  title: string;
  rows: ChartTooltipRow[];
};

/**
 * Escapes text for insertion into Unovis / floating tooltip HTML
 * (including single quotes used in attribute-like payloads).
 */
export function escapeChartTooltipText(raw: string): string {
  return String(raw ?? '')
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;');
}

/** Minimal XY chart tip input (category title + one or more series rows). */
export type ChartXyTooltipSpec = {
  categories: string[];
  series: Array<{ key: string; name: string; values: number[] }>;
  config: Record<string, { color?: string } | undefined>;
  metricLabel?: string;
  percent?: boolean;
  palette?: readonly string[];
};

/**
 * Odoo-like XY tip HTML: title is the category; multi-series lists every series
 * at that category. Returns null when the category/series set is empty.
 */
export function buildChartXyTooltipHtml(
  current: ChartXyTooltipSpec,
  categoryIdx: number | null,
): string | null {
  if (categoryIdx == null) return null;
  const category = current.categories[categoryIdx];
  if (category == null) return null;
  const seriesList = current.series;
  if (!seriesList.length) return null;
  const palette = current.palette?.length ? current.palette : [CHOY_CHART_FALLBACK_COLOR];
  const multi = seriesList.length > 1;
  if (!multi) {
    const series = seriesList[0]!;
    const color = current.config[series.key]?.color || palette[0] || CHOY_CHART_FALLBACK_COLOR;
    const value = Number(series.values[categoryIdx]) || 0;
    return formatChartTooltipHtml({
      title: category,
      rows: [
        {
          color,
          label: current.metricLabel || series.name,
          valueText: formatChartTooltipValue(value, { percent: !!current.percent }),
        },
      ],
    });
  }
  const rows = seriesList.map((series, idx) => {
    const color =
      current.config[series.key]?.color ||
      palette[idx % palette.length] ||
      CHOY_CHART_FALLBACK_COLOR;
    const value = Number(series.values[categoryIdx]) || 0;
    return {
      color,
      label: series.name,
      valueText: formatChartTooltipValue(value, { percent: !!current.percent }),
    };
  });
  return formatChartTooltipHtml({ title: category, rows });
}

/**
 * Odoo-like compact axis / tip numbers: `1.23k`, `4.50M`; smaller values stay plain.
 */
export function formatChartCompactNumber(n: number): string {
  const value = Number.isFinite(n) ? n : 0;
  const sign = value < 0 ? '-' : '';
  const abs = Math.abs(value);
  if (abs >= 1_000_000) return `${sign}${(abs / 1_000_000).toFixed(2)}M`;
  if (abs >= 1_000) return `${sign}${(abs / 1_000).toFixed(2)}k`;
  if (Number.isInteger(abs)) return `${sign}${abs}`;
  return `${sign}${abs.toFixed(2)}`;
}

/**
 * Formats a chart measure for tooltips; percent stacks append `%`.
 * Non-percent values use {@link formatChartCompactNumber} (Odoo-like `k`/`M`).
 * Falls back to `String(n)` when formatting fails.
 */
export function formatChartTooltipValue(
  n: number,
  opts?: { percent?: boolean; locale?: string; compact?: boolean },
): string {
  const value = Number.isFinite(n) ? n : 0;
  if (opts?.percent) {
    let formatted: string;
    try {
      const NumberFormatCtor = (
        globalThis as { Intl?: { NumberFormat?: new (locales?: string) => { format: (v: number) => string } } }
      ).Intl?.NumberFormat;
      formatted =
        typeof NumberFormatCtor === 'function'
          ? new NumberFormatCtor(opts?.locale).format(value)
          : String(value);
    } catch {
      formatted = String(value);
    }
    return `${formatted}%`;
  }
  const useCompact = opts?.compact !== false;
  if (useCompact) return formatChartCompactNumber(value);
  try {
    const NumberFormatCtor = (
      globalThis as { Intl?: { NumberFormat?: new (locales?: string) => { format: (v: number) => string } } }
    ).Intl?.NumberFormat;
    if (typeof NumberFormatCtor === 'function') {
      return new NumberFormatCtor(opts?.locale).format(value);
    }
  } catch {
    /* fall through */
  }
  return String(value);
}

/**
 * Share of a pie slice as `xx.xx%` (of the positive total; empty pie → `0.00%`).
 */
export function formatChartTooltipPiePercent(value: number, total: number): string {
  const v = Number.isFinite(value) ? Math.max(0, value) : 0;
  const t = Number.isFinite(total) ? total : 0;
  if (!(t > 0)) return '0.00%';
  return `${((v / t) * 100).toFixed(2)}%`;
}

/**
 * Places a floating chart tooltip near the cursor while keeping it inside bounds.
 * `hostWidth`/`hostHeight` are the clamp box (plot host or the viewport for a
 * `position: fixed` tip teleported to `body`). Prefers above-right; flips
 * left/below when the preferred corner would overflow.
 */
export function resolveChartFloatingTooltipPosition(input: {
  cursorX: number;
  cursorY: number;
  hostWidth: number;
  hostHeight: number;
  tipWidth: number;
  tipHeight: number;
  gap?: number;
}): { x: number; y: number } {
  const gap = Number.isFinite(input.gap as number) ? Number(input.gap) : 12;
  const hostW = Math.max(0, Number(input.hostWidth) || 0);
  const hostH = Math.max(0, Number(input.hostHeight) || 0);
  const tipW = Math.max(0, Number(input.tipWidth) || 0);
  const tipH = Math.max(0, Number(input.tipHeight) || 0);
  const cx = Number(input.cursorX) || 0;
  const cy = Number(input.cursorY) || 0;

  let x = cx + gap;
  if (x + tipW > hostW - gap) {
    x = cx - gap - tipW;
  }
  x = Math.max(gap, Math.min(x, Math.max(gap, hostW - tipW - gap)));

  let y = cy - gap - tipH;
  if (y < gap) {
    y = cy + gap;
  }
  y = Math.max(gap, Math.min(y, Math.max(gap, hostH - tipH - gap)));

  return { x, y };
}

/** Viewport size for fixed floating tips (SSR-safe fallback). */
export function resolveChartTooltipViewportSize(): { width: number; height: number } {
  if (typeof globalThis !== 'undefined') {
    const w = (globalThis as { innerWidth?: number }).innerWidth;
    const h = (globalThis as { innerHeight?: number }).innerHeight;
    if (typeof w === 'number' && w > 0 && typeof h === 'number' && h > 0) {
      return { width: w, height: h };
    }
  }
  return { width: 1024, height: 768 };
}

/**
 * Builds compact HTML for Unovis VisTooltip (metric title + swatch/label/value rows).
 */
export function formatChartTooltipHtml(input: ChartTooltipHtmlInput): string {
  const title = escapeChartTooltipText(input.title ?? '');
  const rows = Array.isArray(input.rows) ? input.rows : [];
  const rowHtml = rows
    .map((row) => {
      const color = escapeChartTooltipText(row.color || CHOY_CHART_FALLBACK_COLOR);
      const label = escapeChartTooltipText(row.label ?? '');
      const valueText = escapeChartTooltipText(row.valueText ?? '');
      return (
        `<div class="choy-chart-tooltip__row">` +
        `<span class="choy-chart-tooltip__swatch" style="background-color:${color}"></span>` +
        `<span class="choy-chart-tooltip__label">${label}</span>` +
        `<span class="choy-chart-tooltip__value">${valueText}</span>` +
        `</div>`
      );
    })
    .join('');
  return (
    `<div class="choy-chart-tooltip__panel">` +
    `<div class="choy-chart-tooltip__title">${title}</div>` +
    rowHtml +
    `</div>`
  );
}
