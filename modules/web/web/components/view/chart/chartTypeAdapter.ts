// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  CHOY_CHART_FALLBACK_COLOR,
  type ChartConfig,
} from '../../vendor/ui/chart/chartTypes';

/** Default series colors bound to kit chart tokens (no hardcoded hex). */
export const CHOY_CHART_DEFAULT_PALETTE = [
  CHOY_CHART_FALLBACK_COLOR,
  'var(--choy-chart-2)',
  'var(--choy-chart-3)',
  'var(--choy-chart-4)',
  'var(--choy-chart-5)',
] as const;

export type ChoyChartKind = 'bar' | 'line' | 'pie';

export type ChoyChartSort = 'none' | 'asc' | 'desc';

export type ChoyChartSeries = {
  name: string;
  data: number[];
};

export type ChartBuildContext = {
  categories: string[];
  seriesMatrix: ChoyChartSeries[];
  metricLabel: string;
  stacked: boolean;
  palette?: string[];
  /** Percent-stacked: values already normalized to 0–100. */
  percent?: boolean;
};

export type ChartSupportContext = {
  groupDepth: number;
  stacked: boolean;
  seriesCount: number;
  metricAlias: string;
};

export type ChoyChartSlice = {
  key: string;
  name: string;
  value: number;
  /** Original categories/series index before non-positive slices are dropped. */
  categoryIndex?: number;
};

export type ChoyChartSeriesOut = {
  key: string;
  name: string;
  values: number[];
};

/**
 * Unovis / ChartConfig render spec. Adapters must not produce EChartsOption.
 */
export type ChoyChartSpec = {
  kind: ChoyChartKind;
  config: ChartConfig;
  categories: string[];
  series: ChoyChartSeriesOut[];
  stacked: boolean;
  percent: boolean;
  metricLabel: string;
  /** Pie: collapsed slices (optional for XY charts). */
  slices?: ChoyChartSlice[];
};

export interface IChartTypeAdapter {
  id: ChoyChartKind;
  supports(ctx: ChartSupportContext): boolean;
  build(data: ChartBuildContext): ChoyChartSpec;
}

function resolvePalette(palette?: string[]): string[] {
  return palette && palette.length ? palette : [...CHOY_CHART_DEFAULT_PALETTE];
}

/** Reserved XY row fields; series keys must not overwrite them. */
const RESERVED_SERIES_KEYS = new Set(['category', 'index']);

function seriesKey(name: string, index: number): string {
  const slug = String(name || '')
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '_')
    .replace(/^_|_$/g, '');
  let base = slug || `s${index}`;
  if (RESERVED_SERIES_KEYS.has(base)) {
    base = `s_${base}`;
  }
  // Append index so colliding slugs (e.g. "Direct Sales" / "direct-sales") stay unique.
  return `${base}_${index}`;
}

function buildConfig(
  seriesMatrix: ChoyChartSeries[],
  colors: string[],
): ChartConfig {
  const config: ChartConfig = {};
  seriesMatrix.forEach((s, idx) => {
    const key = seriesKey(s.name, idx);
    config[key] = {
      label: s.name || `Series ${idx + 1}`,
      color: colors[idx % colors.length],
    };
  });
  return config;
}

function toFiniteNumber(v: unknown): number {
  const n = Number(v);
  return Number.isFinite(n) ? n : 0;
}

function toSeriesOut(seriesMatrix: ChoyChartSeries[]): ChoyChartSeriesOut[] {
  return seriesMatrix.map((s, idx) => ({
    key: seriesKey(s.name, idx),
    name: s.name || `Series ${idx + 1}`,
    values: (s.data || []).map(toFiniteNumber),
  }));
}

export const barAdapter: IChartTypeAdapter = {
  id: 'bar',
  supports: () => true,
  build(data) {
    const colors = resolvePalette(data.palette);
    return {
      kind: 'bar',
      config: buildConfig(data.seriesMatrix, colors),
      categories: data.categories.slice(),
      series: toSeriesOut(data.seriesMatrix),
      stacked: !!data.stacked && data.seriesMatrix.length > 1,
      percent: !!data.percent,
      metricLabel: data.metricLabel,
    };
  },
};

export const lineAdapter: IChartTypeAdapter = {
  id: 'line',
  supports: () => true,
  build(data) {
    const colors = resolvePalette(data.palette);
    return {
      kind: 'line',
      config: buildConfig(data.seriesMatrix, colors),
      categories: data.categories.slice(),
      series: toSeriesOut(data.seriesMatrix),
      stacked: !!data.stacked && data.seriesMatrix.length > 1,
      percent: !!data.percent,
      metricLabel: data.metricLabel,
    };
  },
};

export const pieAdapter: IChartTypeAdapter = {
  id: 'pie',
  // Pie needs at least one grouping dimension (categories or multi-series).
  supports: ctx => ctx.groupDepth >= 1 || ctx.seriesCount > 1,
  build(data) {
    const colors = resolvePalette(data.palette);
    let slices: ChoyChartSlice[] = [];
    if (data.seriesMatrix.length <= 1) {
      const values = data.seriesMatrix[0]?.data || [];
      slices = data.categories.map((name, i) => ({
        key: seriesKey(name, i),
        name,
        value: toFiniteNumber(values[i]),
        categoryIndex: i,
      }));
    } else {
      slices = data.seriesMatrix.map((s, i) => ({
        key: seriesKey(s.name, i),
        name: s.name || `Series ${i + 1}`,
        value: (s.data || []).reduce((a, b) => a + toFiniteNumber(b), 0),
        categoryIndex: i,
      }));
    }
    // Donut arcs need positive values; drop empty/negative slices so the ring
    // stays valid and palette colors stay dense.
    slices = slices.filter(sl => sl.value > 0);
    const config: ChartConfig = {};
    slices.forEach((sl, idx) => {
      config[sl.key] = {
        label: sl.name,
        color: colors[idx % colors.length],
      };
    });
    return {
      kind: 'pie',
      config,
      categories: data.categories.slice(),
      series: toSeriesOut(data.seriesMatrix),
      stacked: !!data.stacked,
      percent: !!data.percent,
      metricLabel: data.metricLabel,
      slices,
    };
  },
};

export const chartTypeRegistry: Record<ChoyChartKind, IChartTypeAdapter> = {
  bar: barAdapter,
  line: lineAdapter,
  pie: pieAdapter,
};

/**
 * Resolves a chart type adapter by id.
 */
export function resolveChartAdapter(type: string): IChartTypeAdapter | undefined {
  if (!Object.prototype.hasOwnProperty.call(chartTypeRegistry, type)) {
    return undefined;
  }
  return chartTypeRegistry[type as ChoyChartKind];
}

/**
 * Filters registry ids that support the current data context.
 */
export function availableChartTypes(ctx: ChartSupportContext): ChoyChartKind[] {
  return (Object.keys(chartTypeRegistry) as ChoyChartKind[]).filter(id =>
    chartTypeRegistry[id].supports(ctx),
  );
}
