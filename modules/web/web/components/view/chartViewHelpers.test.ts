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
  accumulateChartSeries,
  CHOY_CHART_MIN_PLOT_HEIGHT,
  chartCategoryAxisIsTemporal,
  chartSpecToPieRows,
  chartSpecToXyRows,
  formatChartTooltipHtml,
  formatChartTooltipPiePercent,
  formatChartTooltipValue,
  resolveChartFloatingTooltipPosition,
  normalizeSeriesToPercent,
  parseChartCategoryTime,
  resolveChartPlotHeight,
  resolveChartXDomain,
  resolveStackedYDomain,
  resolveGroupedXyClickTarget,
  resolveClickClientX,
  resolveLineClickCategory,
  resolveLineHoverCategory,
  resolveNearestLineSeriesByY,
  resolveStackedXyClickTarget,
  sortChartCategories,
  sortChartCategoriesByTime,
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
  // Own-property guard: Object.prototype members must not resolve as adapters.
  expect(resolveChartAdapter('toString')).toBeUndefined();
  expect(resolveChartAdapter('constructor')).toBeUndefined();
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
    { key: 'desktop_0', name: 'Desktop', value: 30, categoryIndex: 0 },
    { key: 'mobile_1', name: 'Mobile', value: 10, categoryIndex: 1 },
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
    { name: 'A', data: [33.34] },
    { name: 'B', data: [33.33] },
    { name: 'C', data: [33.33] },
  ]);
  const none = sortChartCategories(categories, series, 'none');
  expect(none.categories).toEqual(['A', 'B']);
  expect(none.order).toEqual([0, 1]);
  const desc = sortChartCategories(categories, series, 'desc');
  expect(desc.categories).toEqual(['A', 'B']);
  expect(desc.order).toEqual([0, 1]);
  expect(desc.seriesMatrix[0]!.data).toEqual([25, 10]);
  const asc = sortChartCategories(categories, series, 'asc');
  expect(asc.categories).toEqual(['B', 'A']);
  expect(asc.order).toEqual([1, 0]);
  expect(asc.seriesMatrix[0]!.data).toEqual([10, 25]);
});

test('sort uses raw totals before percent normalization', () => {
  const sorted = sortChartCategories(
    ['A', 'B'],
    [
      { name: 'X', data: [90, 10] },
      { name: 'Y', data: [0, 10] },
    ],
    'desc',
  );
  expect(sorted.categories).toEqual(['A', 'B']);
  expect(normalizeSeriesToPercent(sorted.categories, sorted.seriesMatrix)).toEqual([
    { name: 'X', data: [100, 50] },
    { name: 'Y', data: [0, 50] },
  ]);
});

test('normalizeSeriesToPercent pads short series to categories length', () => {
  expect(
    normalizeSeriesToPercent(
      ['A', 'B', 'C'],
      [{ name: 'X', data: [10] }],
    ),
  ).toEqual([{ name: 'X', data: [100, 0, 0] }]);
});

test('normalizeSeriesToPercent handles large finite values without overflow', () => {
  expect(
    normalizeSeriesToPercent(
      ['T'],
      [
        { name: 'A', data: [1e305] },
        { name: 'B', data: [1e305] },
      ],
    ),
  ).toEqual([
    { name: 'A', data: [50] },
    { name: 'B', data: [50] },
  ]);
});

test('sortChartCategories none coerces non-finite values', () => {
  const sorted = sortChartCategories(
    ['A'],
    [{ name: 'X', data: [Number.POSITIVE_INFINITY, Number.NaN] }],
    'none',
  );
  expect(sorted.seriesMatrix[0]!.data).toEqual([0, 0]);
  expect(sorted.order).toEqual([0]);
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

test('pieAdapter preserves original categoryIndex after filtering', () => {
  const spec = pieAdapter.build({
    categories: ['A', 'B', 'C'],
    seriesMatrix: [{ name: 'Visitors', data: [0, 40, 60] }],
    metricLabel: 'Visitors',
    stacked: false,
  });
  expect(spec.slices?.map(s => s.categoryIndex)).toEqual([1, 2]);
  expect(spec.slices?.map(s => s.name)).toEqual(['B', 'C']);
});

test('pieAdapter drops non-positive slices', () => {
  const spec = pieAdapter.build({
    categories: ['A', 'B', 'C'],
    seriesMatrix: [{ name: 'Visitors', data: [10, 0, -5] }],
    metricLabel: 'Visitors',
    stacked: false,
  });
  expect(spec.slices).toEqual([
    { key: 'a_0', name: 'A', value: 10, categoryIndex: 0 },
  ]);
  expect(Object.keys(spec.config)).toEqual(['a_0']);
});

test('normalize and sort tolerate missing series data', () => {
  const categories = ['A', 'B'];
  const series = [
    { name: 'X', data: [10] },
    { name: 'Y' } as { name: string; data?: number[] },
  ];
  expect(
    normalizeSeriesToPercent(categories, series as { name: string; data: number[] }[]),
  ).toEqual([
    { name: 'X', data: [100, 0] },
    { name: 'Y', data: [0, 0] },
  ]);
  expect(
    sortChartCategories(
      categories,
      series as { name: string; data: number[] }[],
      'desc',
    ).categories,
  ).toEqual(['A', 'B']);
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
  expect(resolveStackedXyClickTarget({ index: '   ' }, undefined, 1)).toEqual({
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
  expect(resolveLineClickCategory(Number.NaN, 4)).toBe(0);
});

test('resolveLineHoverCategory uses equal-width category bands', () => {
  expect(resolveLineHoverCategory(0.1, 2)).toBe(0);
  expect(resolveLineHoverCategory(0.6, 2)).toBe(1);
  expect(resolveLineHoverCategory(0.2, 3)).toBe(0);
  expect(resolveLineHoverCategory(0.5, 3)).toBe(1);
  expect(resolveLineHoverCategory(0.9, 3)).toBe(2);
  expect(resolveLineHoverCategory(1, 3)).toBe(2);
});

test('resolveNearestLineSeriesByY picks closest series to pointer data-Y', () => {
  expect(resolveNearestLineSeriesByY([10, 200], 0.1, [0, 250])).toBe(1);
  expect(resolveNearestLineSeriesByY([10, 200], 0.95, [0, 250])).toBe(0);
  expect(resolveNearestLineSeriesByY([42], 0.5, [0, 100])).toBe(0);
});

test('resolveClickClientX prefers changedTouches then clientX', () => {
  expect(resolveClickClientX({ clientX: 42 })).toBe(42);
  expect(resolveClickClientX({ clientX: Number.NaN })).toBeUndefined();
  expect(resolveClickClientX({ changedTouches: [{ clientX: 17 }] })).toBe(17);
  expect(
    resolveClickClientX({ clientX: 99, changedTouches: [{ clientX: 17 }] }),
  ).toBe(17);
  // Non-finite touch X falls through to event.clientX.
  expect(
    resolveClickClientX({
      clientX: 55,
      changedTouches: [{ clientX: Number.NaN }],
    }),
  ).toBe(55);
  expect(resolveClickClientX({ changedTouches: [] })).toBeUndefined();
  expect(resolveClickClientX({})).toBeUndefined();
});

test('sortChartCategories coerces non-finite values to 0', () => {
  const sorted = sortChartCategories(
    ['A', 'B'],
    [
      { name: 'X', data: [Number.POSITIVE_INFINITY, 1] },
      { name: 'Y', data: [0, 2] },
    ],
    'asc',
  );
  expect(sorted.categories).toEqual(['A', 'B']);
  expect(sorted.seriesMatrix[0]!.data).toEqual([0, 1]);
});

test('resolveChartPlotHeight fills body minus legend reserve and respects min', () => {
  expect(resolveChartPlotHeight(0)).toBe(CHOY_CHART_MIN_PLOT_HEIGHT);
  expect(resolveChartPlotHeight(200)).toBe(CHOY_CHART_MIN_PLOT_HEIGHT);
  expect(resolveChartPlotHeight(400)).toBe(360);
  expect(resolveChartPlotHeight(400, { minHeight: 100, legendReserve: 50 })).toBe(350);
});

test('resolveChartXDomain pads integer indices so first/last categories are inset', () => {
  expect(resolveChartXDomain(0)).toBeUndefined();
  expect(resolveChartXDomain(-1)).toBeUndefined();
  expect(resolveChartXDomain(1)).toEqual([-0.5, 0.5]);
  expect(resolveChartXDomain(2)).toEqual([-0.5, 1.5]);
  expect(resolveChartXDomain(4)).toEqual([-0.5, 3.5]);
});

test('resolveChartXDomain for line starts at index 0 like Odoo', () => {
  expect(resolveChartXDomain(1, 'line')).toEqual([-0.5, 0.5]);
  expect(resolveChartXDomain(2, 'line')).toEqual([0, 1]);
  expect(resolveChartXDomain(4, 'line')).toEqual([0, 3]);
});

test('chartCategoryAxisIsTemporal detects first groupby granularity', () => {
  expect(chartCategoryAxisIsTemporal([])).toBe(false);
  expect(chartCategoryAxisIsTemporal(['TokenType'])).toBe(false);
  expect(chartCategoryAxisIsTemporal(['CreatedAt:month'])).toBe(true);
  expect(chartCategoryAxisIsTemporal([{ field: 'CreatedAt', granularity: 'day' }])).toBe(true);
  expect(chartCategoryAxisIsTemporal([{ field: 'TokenType' }, { field: 'CreatedAt', granularity: 'week' }])).toBe(
    false,
  );
  expect(chartCategoryAxisIsTemporal(['CreatedAt:month', 'Revoked'])).toBe(true);
});

test('parseChartCategoryTime understands ORM temporal labels', () => {
  expect(parseChartCategoryTime('2025')).toBe(Date.UTC(2025, 0, 1));
  expect(parseChartCategoryTime('2025-Q4')).toBe(Date.UTC(2025, 9, 1));
  expect(parseChartCategoryTime('2025-11')).toBe(Date.UTC(2025, 10, 1));
  expect(parseChartCategoryTime('2025-11-06')).toBe(Date.UTC(2025, 10, 6));
  expect(typeof parseChartCategoryTime('2025-W01')).toBe('number');
  expect(parseChartCategoryTime('not-a-date')).toBeUndefined();
});

test('sortChartCategoriesByTime orders ascending and reorders series', () => {
  const sorted = sortChartCategoriesByTime(
    ['2025-Q4', '2025-Q1', '2025-Q2'],
    [
      { name: 'A', data: [40, 10, 20] },
      { name: 'B', data: [4, 1, 2] },
    ],
  );
  expect(sorted.categories).toEqual(['2025-Q1', '2025-Q2', '2025-Q4']);
  expect(sorted.seriesMatrix[0]!.data).toEqual([10, 20, 40]);
  expect(sorted.seriesMatrix[1]!.data).toEqual([1, 2, 4]);
});

test('accumulateChartSeries builds running totals along categories', () => {
  expect(
    accumulateChartSeries(
      ['a', 'b', 'c'],
      [
        { name: 'X', data: [1, 2, 3] },
        { name: 'Y', data: [10, 0, 5] },
      ],
    ),
  ).toEqual([
    { name: 'X', data: [1, 3, 6] },
    { name: 'Y', data: [10, 10, 15] },
  ]);
});

test('resolveStackedYDomain uses column sums so stacks are not clipped', () => {
  expect(
    resolveStackedYDomain(
      [
        { a: 140, b: 20 },
        { a: 100, b: 50 },
      ],
      ['a', 'b'],
    )[1],
  ).toBeGreaterThanOrEqual(160);
  expect(resolveStackedYDomain([], ['a'])).toEqual([0, 1]);
  expect(resolveStackedYDomain([{ a: 40, b: 60 }], ['a', 'b'], { percent: true })).toEqual([
    0,
    100,
  ]);
});

test('formatChartTooltipValue uses locale digits and percent suffix', () => {
  const plain = formatChartTooltipValue(1234.5, { locale: 'en-US' });
  // QuickJS FE may lack Intl.NumberFormat; accept ungrouped digits there.
  expect(plain === '1,234.5' || plain === '1234.5').toBe(true);
  expect(formatChartTooltipValue(12.5, { locale: 'en-US', percent: true })).toMatch(/12\.5%$/);
  expect(formatChartTooltipValue(Number.NaN, { locale: 'en-US' })).toBe('0');
});

test('formatChartTooltipPiePercent is share of positive total', () => {
  expect(formatChartTooltipPiePercent(25, 100)).toBe('25.00%');
  expect(formatChartTooltipPiePercent(1, 3)).toBe('33.33%');
  expect(formatChartTooltipPiePercent(10, 0)).toBe('0.00%');
  expect(formatChartTooltipPiePercent(-5, 100)).toBe('0.00%');
});

test('resolveChartFloatingTooltipPosition flips when near the right/top edges', () => {
  expect(
    resolveChartFloatingTooltipPosition({
      cursorX: 20,
      cursorY: 80,
      hostWidth: 400,
      hostHeight: 300,
      tipWidth: 140,
      tipHeight: 56,
      gap: 12,
    }),
  ).toEqual({ x: 32, y: 12 });
  const nearRight = resolveChartFloatingTooltipPosition({
    cursorX: 380,
    cursorY: 80,
    hostWidth: 400,
    hostHeight: 300,
    tipWidth: 140,
    tipHeight: 56,
    gap: 12,
  });
  expect(nearRight.x).toBeLessThan(380 - 140);
  expect(nearRight.x + 140).toBeLessThanOrEqual(400 - 12);
});

test('formatChartTooltipHtml renders title, swatch, label, value and escapes text', () => {
  const html = formatChartTooltipHtml({
    title: 'Count <x>',
    rows: [
      {
        color: '#ff0000',
        label: 'Series & "A"',
        valueText: '12 (50.00%)',
      },
    ],
  });
  expect(html).toContain('choy-chart-tooltip__title');
  expect(html).toContain('Count &lt;x&gt;');
  expect(html).toContain('background-color:#ff0000');
  expect(html).toContain('Series &amp; &quot;A&quot;');
  expect(html).toContain('12 (50.00%)');
});
