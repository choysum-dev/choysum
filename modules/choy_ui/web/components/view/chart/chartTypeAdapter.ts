// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { ChartConfig } from '../../vendor/ui/chart/chartTypes';

/** Default series colors bound to kit chart tokens (no hardcoded hex). */
export const CHOY_CHART_DEFAULT_PALETTE = [
  'var(--choy-chart-1)',
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

function seriesKey(name: string, index: number): string {
  const slug = String(name || '')
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '_')
    .replace(/^_|_$/g, '');
  return slug || `s${index}`;
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

function toSeriesOut(seriesMatrix: ChoyChartSeries[]): ChoyChartSeriesOut[] {
  return seriesMatrix.map((s, idx) => ({
    key: seriesKey(s.name, idx),
    name: s.name || `Series ${idx + 1}`,
    values: (s.data || []).map(v => Number(v) || 0),
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
  supports: ctx => ctx.groupDepth >= 1,
  build(data) {
    const colors = resolvePalette(data.palette);
    let slices: ChoyChartSlice[] = [];
    if (data.seriesMatrix.length <= 1) {
      const values = data.seriesMatrix[0]?.data || [];
      slices = data.categories.map((name, i) => ({
        key: seriesKey(name, i),
        name,
        value: Number(values[i]) || 0,
      }));
    } else {
      slices = data.seriesMatrix.map((s, i) => ({
        key: seriesKey(s.name, i),
        name: s.name || `Series ${i + 1}`,
        value: (s.data || []).reduce((a, b) => a + (Number(b) || 0), 0),
      }));
    }
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
  if (type === 'bar' || type === 'line' || type === 'pie') {
    return chartTypeRegistry[type];
  }
  return undefined;
}

/**
 * Filters registry ids that support the current data context.
 */
export function availableChartTypes(ctx: ChartSupportContext): ChoyChartKind[] {
  return (Object.keys(chartTypeRegistry) as ChoyChartKind[]).filter(id =>
    chartTypeRegistry[id].supports(ctx),
  );
}
