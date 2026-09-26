// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  availableChartTypes,
  barAdapter,
  lineAdapter,
  pieAdapter,
  resolveChartAdapter,
} from './chart/chartTypeAdapter';
import {
  chartSpecToPieRows,
  chartSpecToXyRows,
  normalizeSeriesToPercent,
  resolveGroupedXyClickTarget,
  resolveLineClickCategory,
  resolveStackedXyClickTarget,
  sortChartCategories,
} from './chartViewHelpers';

test('toSeriesOut coerces non-finite values to 0', () => {
  const spec = barAdapter.build({
    categories: ['A', 'B'],
    seriesMatrix: [{ name: 'X', data: [Number.POSITIVE_INFINITY, Number.NaN] }],
    metricLabel: 'Count',
    stacked: false,
  });
  expect(spec.series[0]!.values).toEqual([0, 0]);
});

test('resolveChartAdapter returns bar/line/pie and rejects unknown', () => {
  expect(resolveChartAdapter('bar')?.id).toBe('bar');
  expect(resolveChartAdapter('line')?.id).toBe('line');
  expect(resolveChartAdapter('pie')?.id).toBe('pie');
  expect(resolveChartAdapter('radar')).toBeUndefined();
});

test('availableChartTypes filters pie by groupDepth or multi-series', () => {
  expect(
    availableChartTypes({
      groupDepth: 0,
      stacked: true,
      seriesCount: 1,
      metricAlias: 'count',
    }),
  ).toEqual(['bar', 'line']);
  expect(
    availableChartTypes({
      groupDepth: 0,
      stacked: true,
      seriesCount: 2,
      metricAlias: 'count',
    }),
  ).toEqual(['bar', 'line', 'pie']);
  expect(
    availableChartTypes({
      groupDepth: 1,
      stacked: true,
      seriesCount: 2,
      metricAlias: 'count',
    }),
  ).toEqual(['bar', 'line', 'pie']);
});

test('barAdapter builds ChartConfig with choy token colors', () => {
  const spec = barAdapter.build({
    categories: ['A', 'B'],
    seriesMatrix: [
      { name: 'Desktop', data: [1, 2] },
      { name: 'Mobile', data: [3, 4] },
    ],
    metricLabel: 'Count',
    stacked: true,
  });
  expect(spec.kind).toBe('bar');
  expect(spec.stacked).toBe(true);
  expect(spec.config.desktop_0?.color).toBe('var(--choy-chart-1)');
  expect(spec.config.mobile_1?.color).toBe('var(--choy-chart-2)');
  expect(spec.series.map(s => s.key)).toEqual(['desktop_0', 'mobile_1']);
});

test('lineAdapter honors custom palette and empty series names', () => {
  const spec = lineAdapter.build({
    categories: ['A'],
    seriesMatrix: [
      { name: '', data: [1] },
      { name: '!!!', data: [2] },
    ],
    metricLabel: 'Count',
    stacked: false,
    palette: ['var(--choy-chart-3)', 'var(--choy-chart-4)'],
  });
  expect(spec.kind).toBe('line');
  expect(spec.series[0]!.key).toBe('s0_0');
  expect(spec.series[0]!.name).toBe('Series 1');
  expect(spec.series[1]!.key).toBe('s1_1');
  expect(spec.config.s0_0?.color).toBe('var(--choy-chart-3)');
});

test('seriesKey avoids reserved row keys and slug collisions', () => {
  const spec = barAdapter.build({
    categories: ['A'],
    seriesMatrix: [
      { name: 'Category', data: [1] },
      { name: 'Index', data: [2] },
      { name: 'Direct Sales', data: [3] },
      { name: 'direct-sales', data: [4] },
    ],
    metricLabel: 'Count',
    stacked: true,
  });
  expect(spec.series.map(s => s.key)).toEqual([
    's_category_0',
    's_index_1',
    'direct_sales_2',
    'direct_sales_3',
  ]);
  const rows = chartSpecToXyRows(spec);
  expect(rows[0]!.category).toBe('A');
  expect(rows[0]!.index).toBe(0);
  expect(rows[0]!.s_category_0).toBe(1);
  expect(rows[0]!.direct_sales_2).toBe(3);
  expect(rows[0]!.direct_sales_3).toBe(4);
});

test('pieAdapter collapses multi-series into slices', () => {
  const spec = pieAdapter.build({
    categories: ['A', 'B'],
    seriesMatrix: [
      { name: 'Desktop', data: [10, 20] },
      { name: 'Mobile', data: [5, 5] },
    ],
    metricLabel: 'Count',
    stacked: true,
  });
  expect(spec.kind).toBe('pie');
  expect(spec.slices).toEqual([
    { key: 'desktop_0', name: 'Desktop', value: 30 },
    { key: 'mobile_1', name: 'Mobile', value: 10 },
  ]);
});

test('normalizeSeriesToPercent and sortChartCategories', () => {
  const categories = ['A', 'B'];
  const series = [
    { name: 'X', data: [25, 10] },
    { name: 'Y', data: [75, 40] },
  ];
  expect(normalizeSeriesToPercent(categories, series)).toEqual([
    { name: 'X', data: [25, 20] },
    { name: 'Y', data: [75, 80] },
  ]);
  expect(
    normalizeSeriesToPercent(['Z'], [{ name: 'X', data: [0] }, { name: 'Y', data: [0] }]),
  ).toEqual([
    { name: 'X', data: [0] },
    { name: 'Y', data: [0] },
  ]);
  expect(
    normalizeSeriesToPercent(
      ['N'],
      [
        { name: 'X', data: [10] },
        { name: 'Y', data: [-5] },
      ],
    ),
  ).toEqual([
    { name: 'X', data: [100] },
    { name: 'Y', data: [0] },
  ]);
  expect(
    normalizeSeriesToPercent(
      ['T'],
      [
        { name: 'A', data: [1] },
        { name: 'B', data: [1] },
        { name: 'C', data: [1] },
      ],
    ),
  ).toEqual([
    { name: 'A', data: [33.33] },
    { name: 'B', data: [33.33] },
    { name: 'C', data: [33.33] },
  ]);
  const none = sortChartCategories(categories, series, 'none');
  expect(none.categories).toEqual(['A', 'B']);
  const desc = sortChartCategories(categories, series, 'desc');
  expect(desc.categories).toEqual(['A', 'B']);
  expect(desc.seriesMatrix[0]!.data).toEqual([25, 10]);
  const asc = sortChartCategories(categories, series, 'asc');
  expect(asc.categories).toEqual(['B', 'A']);
  expect(asc.seriesMatrix[0]!.data).toEqual([10, 25]);
});

test('chartSpecToXyRows / chartSpecToPieRows flatten for Unovis', () => {
  const spec = barAdapter.build({
    categories: ['Jan', 'Feb'],
    seriesMatrix: [{ name: 'Desktop', data: [1, 2] }],
    metricLabel: 'Count',
    stacked: false,
  });
  expect(chartSpecToXyRows(spec)).toEqual([
    { category: 'Jan', index: 0, desktop_0: 1 },
    { category: 'Feb', index: 1, desktop_0: 2 },
  ]);
  expect(
    chartSpecToXyRows({
      ...spec,
      series: [{ key: 'desktop_0', name: 'Desktop', values: [1] }],
    }),
  ).toEqual([
    { category: 'Jan', index: 0, desktop_0: 1 },
    { category: 'Feb', index: 1, desktop_0: 0 },
  ]);
  const pie = pieAdapter.build({
    categories: ['Chrome', 'Safari'],
    seriesMatrix: [{ name: 'Visitors', data: [100, 50] }],
    metricLabel: 'Visitors',
    stacked: false,
  });
  expect(chartSpecToPieRows({ ...pie, slices: undefined })).toEqual([]);
  expect(chartSpecToPieRows(pie).map(r => r.name)).toEqual(['Chrome', 'Safari']);
  expect(chartSpecToPieRows(pie)[0]!.color).toBe('var(--choy-chart-1)');
  // Missing slice color falls back when config entry has no color.
  const noColor = {
    ...pie,
    config: { ...pie.config, [pie.slices![0]!.key]: { label: 'Chrome' } },
  };
  expect(chartSpecToPieRows(noColor)[0]!.color).toBe('var(--choy-chart-1)');
});

test('pieAdapter drops non-positive slices', () => {
  const spec = pieAdapter.build({
    categories: ['A', 'B', 'C'],
    seriesMatrix: [{ name: 'Visitors', data: [10, 0, -5] }],
    metricLabel: 'Visitors',
    stacked: false,
  });
  expect(spec.slices).toEqual([{ key: 'a_0', name: 'A', value: 10 }]);
  expect(Object.keys(spec.config)).toEqual(['a_0']);
});

test('resolveGroupedXyClickTarget maps flat element index', () => {
  expect(resolveGroupedXyClickTarget({ index: 1 }, 3, 2)).toEqual({
    categoryIdx: 1,
    seriesIdx: 1,
  });
  expect(resolveGroupedXyClickTarget(undefined, 3, 2)).toEqual({
    categoryIdx: 1,
    seriesIdx: 1,
  });
  expect(resolveGroupedXyClickTarget({ index: 0 }, undefined, 1)).toEqual({
    categoryIdx: 0,
    seriesIdx: 0,
  });
  // Flat index implies another category — do not invent a series.
  expect(resolveGroupedXyClickTarget({ index: 0 }, 3, 2)).toEqual({
    categoryIdx: 0,
    seriesIdx: undefined,
  });
});

test('resolveStackedXyClickTarget prefers row index over event index', () => {
  expect(resolveStackedXyClickTarget({ index: 2, stackIndex: 9 }, 1, 0)).toEqual({
    categoryIdx: 2,
    seriesIdx: 0,
  });
  expect(resolveStackedXyClickTarget({ index: '2' }, undefined, 1)).toEqual({
    categoryIdx: 2,
    seriesIdx: 1,
  });
  expect(resolveStackedXyClickTarget(undefined, 3, 1)).toEqual({
    categoryIdx: 3,
    seriesIdx: 1,
  });
  // Row present but index unusable → fall through to null (covers final return).
  expect(resolveStackedXyClickTarget({ category: 'A' }, undefined, 0)).toEqual({
    categoryIdx: null,
    seriesIdx: 0,
  });
  expect(resolveStackedXyClickTarget({ index: '' }, undefined, 1)).toEqual({
    categoryIdx: null,
    seriesIdx: 1,
  });
  expect(resolveStackedXyClickTarget({ index: 'nope' }, undefined, 1)).toEqual({
    categoryIdx: null,
    seriesIdx: 1,
  });
  expect(resolveStackedXyClickTarget({ index: Number.NaN }, undefined, 1)).toEqual({
    categoryIdx: null,
    seriesIdx: 1,
  });
});

test('resolveLineClickCategory maps relative x to nearest category', () => {
  expect(resolveLineClickCategory(0.5, 0)).toBe(0);
  expect(resolveLineClickCategory(0.5, 1)).toBe(0);
  expect(resolveLineClickCategory(-0.2, 4)).toBe(0);
  expect(resolveLineClickCategory(1.2, 4)).toBe(3);
  expect(resolveLineClickCategory(0.5, 3)).toBe(1);
});
