// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

// Chart type adapter abstraction for OChartView
// Each adapter decides if it supports current data context and builds ECharts option.

import type { EChartsOption } from 'echarts';
// Register the shared ECharts components used by chart adapters.
import { use } from 'echarts/core';
import { BarChart, LineChart, PieChart } from 'echarts/charts';
import { TitleComponent, TooltipComponent, LegendComponent, GridComponent } from 'echarts/components';
import { SVGRenderer } from 'echarts/renderers';

let __echartsRegistered = false;

/**
 * Registers the ECharts modules required by the supported chart adapters.
 */
export function ensureEChartsRegistered() {
  if (!__echartsRegistered) {
    use([BarChart, LineChart, PieChart, TitleComponent, TooltipComponent, LegendComponent, GridComponent, SVGRenderer]);
    __echartsRegistered = true;
  }
}

export interface ChartBuildContext {
  categories: string[];
  seriesMatrix: Array<{ name: string; data: number[] }>;
  metricLabel: string;
  stacked: boolean;
  palette?: string[];

  // Percent-stacked mode uses values already normalized to 0-100.
  percent?: boolean;
}

export interface ChartSupportContext {
  groupDepth: number;
  stacked: boolean;
  seriesCount: number;
  metricAlias: string;
}

export interface IChartTypeAdapter {
  id: string;
  supports(ctx: ChartSupportContext): boolean;
  buildOption(data: ChartBuildContext): EChartsOption;
}

function baseColors(palette?: string[]): string[] | undefined {
  return palette && palette.length ? palette : undefined; // rely on ECharts default if undefined
}

export const barAdapter: IChartTypeAdapter = {
  id: 'bar',
  supports: _ctx => true, // bar supports all contexts
  buildOption({ categories, seriesMatrix, metricLabel, stacked, palette, percent }): EChartsOption {
    const colors = palette && palette.length ? palette : ['#5470c6', '#91cc75', '#fac858', '#ee6666', '#73c0de', '#3ba272', '#fc8452', '#9a60b4', '#ea7ccc'];
    const showTotal = stacked && seriesMatrix.length > 1;
    const totals = showTotal ? categories.map((_, idx) => seriesMatrix.reduce((sum, s) => sum + Number(s.data[idx] ?? 0), 0)) : [];
    return {
      color: colors,
      tooltip: {
        trigger: 'item',
        formatter: param => {
          const p: any = Array.isArray(param) ? param[0] : param;
          const name = String(p.name ?? p.axisValue ?? '');
          const val = Number(p.data?.value ?? p.data ?? 0);
          const valueStr = percent ? `${val}%` : `${val}`;
          const lines = [`${name}`, `${p.marker}${p.seriesName}: ${valueStr}`];
          if (showTotal) {
            lines.push(percent ? `Total: 100%` : `Total: ${totals[p.dataIndex ?? 0] ?? 0}`);
          }
          return lines.join('<br/>');
        },
      },
      legend: { show: true, top: 4, right: 8 },
      grid: { left: 24, right: 16, top: 32, bottom: 24 },
      xAxis: { type: 'category', data: categories },
      yAxis: percent
        ? {
            type: 'value',
            max: 100,
            axisLabel: { formatter: '{value}%' },
          }
        : { type: 'value' },
      series: seriesMatrix.map((s, idx) => ({
        name: s.name,
        type: 'bar',
        data: percent ? s.data.map(v => ({ value: v })) : s.data,
        stack: stacked && seriesMatrix.length > 1 ? 'total' : undefined,
        barMaxWidth: 48,
        emphasis: { focus: 'series' },
        itemStyle: { color: colors[idx % colors.length] },
      })),
    } as EChartsOption;
  },
};

export const lineAdapter: IChartTypeAdapter = {
  id: 'line',
  supports: _ctx => true,
  buildOption({ categories, seriesMatrix, metricLabel, stacked, palette, percent }): EChartsOption {
    const colors = palette && palette.length ? palette : ['#5470c6', '#91cc75', '#fac858', '#ee6666', '#73c0de', '#3ba272', '#fc8452', '#9a60b4', '#ea7ccc'];
    const showTotal = stacked && seriesMatrix.length > 1;
    const totals = showTotal ? categories.map((_, idx) => seriesMatrix.reduce((sum, s) => sum + Number(s.data[idx] ?? 0), 0)) : [];
    return {
      color: colors,
      tooltip: {
        trigger: 'item',
        formatter: param => {
          const p: any = Array.isArray(param) ? param[0] : param;
          const name = String(p.name ?? p.axisValue ?? '');
          const val = Number(p.data?.value ?? p.data ?? 0);
          const valueStr = percent ? `${val}%` : `${val}`;
          const lines = [`${name}`, `${p.marker}${p.seriesName}: ${valueStr}`];
          if (showTotal) {
            lines.push(percent ? `Total: 100%` : `Total: ${totals[p.dataIndex ?? 0] ?? 0}`);
          }
          return lines.join('<br/>');
        },
      },
      legend: { show: true, top: 4, right: 8 },
      grid: { left: 24, right: 16, top: 32, bottom: 24 },
      xAxis: { type: 'category', data: categories },
      yAxis: percent ? { type: 'value', max: 100, axisLabel: { formatter: '{value}%' } } : { type: 'value' },
      series: seriesMatrix.map((s, idx) => ({
        name: s.name,
        type: 'line',
        data: percent ? s.data.map(v => ({ value: v })) : s.data,
        smooth: true,
        stack: stacked && seriesMatrix.length > 1 ? 'total' : undefined,
        showSymbol: true,
        symbolSize: 8,
        lineStyle: { width: 2, color: colors[idx % colors.length] },
        itemStyle: { color: colors[idx % colors.length] },
        emphasis: { focus: 'series' },
        areaStyle: stacked && seriesMatrix.length > 1 ? { opacity: percent ? 0.4 : 0.25 } : undefined,
      })),
    } as EChartsOption;
  },
};

export const pieAdapter: IChartTypeAdapter = {
  id: 'pie',
  // Pie charts accept any group depth and collapse multi-series data in buildOption.
  supports: ctx => ctx.groupDepth >= 1,
  buildOption({ categories, seriesMatrix, metricLabel, palette, percent, stacked }): EChartsOption {
    const colors = palette && palette.length ? palette : ['#5470c6', '#91cc75', '#fac858', '#ee6666', '#73c0de', '#3ba272', '#fc8452', '#9a60b4', '#ea7ccc'];

    // Single-series data stays at the category level; multi-series data is collapsed by series name.
    let slices: { name: string; value: number }[] = [];
    if (seriesMatrix.length <= 1) {
      const dataArr = (seriesMatrix[0]?.data || []).slice();
      slices = categories.map((c, i) => ({ name: c, value: dataArr[i] ?? 0 }));
    } else {
      slices = seriesMatrix.map(s => ({ name: s.name, value: s.data.reduce((a, b) => a + Number(b ?? 0), 0) }));
    }
    const grandTotal = slices.reduce((a, b) => a + b.value, 0);
    const showTotal = !!stacked;
    return {
      color: colors,

      // Hide grid and axes so merged bar/line configuration does not leak into pie charts.
      xAxis: { show: false },
      yAxis: { show: false },
      grid: { top: 0, bottom: 0, left: 0, right: 0 },
      tooltip: {
        trigger: 'item',
        formatter: params => {
          const val = Number((params as any).value ?? 0);
          if (!grandTotal) return `${(params as any).name}: ${val}`;
          const pct = ((val / grandTotal) * 100).toFixed(2);
          const lines = [`${(params as any).name}`, `${metricLabel}: ${val} (${pct}%)`];
          if (showTotal) lines.push(`Total: ${grandTotal}`);
          return lines.join('<br/>');
        },
      },
      legend: { show: true, top: 4, right: 8 },
      series: [
        {
          name: metricLabel,
          type: 'pie',
          radius: ['30%', '70%'],
          center: ['50%', '50%'],
          label: { formatter: '{b}' },
          data: slices.map((sl, i) => ({
            name: sl.name,
            value: sl.value,
            itemStyle: { color: colors[i % colors.length] },
          })),
          emphasis: { itemStyle: { shadowBlur: 8, shadowOffsetX: 0, shadowColor: 'rgba(0,0,0,0.3)' } },
          tooltip: {
            formatter: params => {
              const val = Number((params as any).value ?? 0);
              if (!grandTotal) return `${metricLabel}: ${val}`;
              const pct = ((val / grandTotal) * 100).toFixed(2);
              const lines = [`${(params as any).name}`, `${metricLabel}: ${val} (${pct}%)`];
              if (showTotal) lines.push(`Total: ${grandTotal}`);
              return lines.join('<br/>');
            },
          },
        },
      ],
    } as EChartsOption;
  },
};

export const chartTypeRegistry: Record<string, IChartTypeAdapter> = {
  bar: barAdapter,
  line: lineAdapter,
  pie: pieAdapter,
};

/**
 * Resolves a chart adapter by chart type.
 */
export function resolveChartAdapter(type: string): IChartTypeAdapter | undefined {
  return chartTypeRegistry[type];
}
