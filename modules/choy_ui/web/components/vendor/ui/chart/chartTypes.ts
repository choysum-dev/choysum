// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/** shadcn-vue ChartConfig: series key → label + CSS color (prefer --choy-chart-*). */
export type ChartConfigItem = {
  label?: string;
  color?: string;
};

export type ChartConfig = Record<string, ChartConfigItem>;

/** Default fill when a series/slice has no color in ChartConfig. */
export const CHOY_CHART_FALLBACK_COLOR = 'var(--choy-chart-1)';
