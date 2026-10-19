// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import { nextTick } from 'vue';
import ChoyChartView from './ChoyChartView.vue';

function makeStore(groups: unknown[] = ['CreatedAt:month']) {
  return {
    storeId: 'web.chart.coverage.test',
    state: { queryState: { appliedGroups: groups, keyword: 'tok' } },
  } as any;
}

const nestedGroupResult = {
  kind: 'group',
  rows: [
    {
      kind: 'group',
      depth: 0,
      key: '2024-01',
      label: '2024-01',
      children: [
        { kind: 'group', depth: 1, key: 'open', label: 'Open', count: 10 },
        { kind: 'group', depth: 1, key: 'done', label: 'Done', count: 20 },
      ],
    },
    {
      kind: 'group',
      depth: 0,
      key: '2024-02',
      label: '2024-02',
      children: [
        { kind: 'group', depth: 1, key: 'open', label: 'Open', count: 5 },
        { kind: 'group', depth: 1, key: 'done', label: 'Done', count: 25 },
      ],
    },
  ],
};

const flatGroupResult = {
  kind: 'group',
  rows: [
    { kind: 'group', depth: 0, key: 'a', label: 'Alpha', count: 3 },
    { kind: 'group', depth: 0, key: 'b', label: 'Beta', count: 7 },
  ],
};

type ChartExpose = {
  controller: {
    vm: { result: unknown; loading: boolean; error: unknown };
    apply: (opts?: unknown) => Promise<void>;
  };
  rebuildFromSnapshot: () => void;
  bootstrap: () => Promise<void>;
  selectChartType: (t: string) => void;
  selectSort: (s: string) => void;
  selectMetric: (a: string) => void;
  toggleStacked: () => void;
  toggleCumulative: () => void;
  onMetricSelect: (v: unknown) => void;
  onSearch: (q: unknown) => void;
  onRefresh: () => Promise<void>;
  initMetricOptions: () => void;
  syncPlotHeight: () => void;
  xAccessor: (d: { index?: number }) => number;
  xTickFormat: (v: number) => string;
  yTickFormat: (v: number) => string;
  onStackedBarClick: (d: unknown, e: unknown, idx?: number) => void;
  onGroupedBarClick: (d: unknown, e: unknown, idx?: number) => void;
  onPieSegmentClick: (d: unknown) => void;
  onStackedBarTooltip: (d: unknown, i?: number, nodes?: Element[]) => string | null;
  onGroupedBarTooltip: (d: unknown, i?: number, nodes?: Element[]) => string | null;
  onPieSegmentTooltip: (d: unknown, i?: number, nodes?: Element[]) => string | null;
};

describe('ChoyChartView coverage', () => {
  test('bar tooltips clicks and metric sort switches', async () => {
    let loadErr: unknown;
    let searchErr: unknown;
    const mounted = mountApp(ChoyChartView as any, {
      reactiveProps: true,
      props: {
        store: makeStore(),
        autoBootstrap: false,
        chartType: 'bar',
        stacked: true,
        cumulative: false,
        sort: 'none',
        metricAlias: 'count',
        metrics: [
          { alias: 'count', label: 'Count' },
          { alias: 'amount_sum', label: 'Amount' },
        ],
        formatLabel: (s: string) => `L:${s}`,
        defaultGroups: { field: 'TokenType' } as any,
        onLoadError: (e: unknown) => {
          loadErr = e;
        },
        onSearchError: (e: unknown) => {
          searchErr = e;
        },
      },
    });
    await flushPromises();
    const ex = mounted.root as ChartExpose & {
      linePlotYDomain: () => [number, number];
      normalizeDefaultGroups: () => unknown;
      applyQuery: (opts?: unknown) => Promise<void>;
    };
    ex.controller.apply = async () => {
      throw new Error('apply failed');
    };
    ex.controller.vm.result = nestedGroupResult;
    ex.rebuildFromSnapshot();
    await nextTick();
    expect(mounted.text()).toContain('L:Open');
    expect(ex.normalizeDefaultGroups()).toBeTruthy();
    ex.linePlotYDomain();
    expect(ex.xTickFormat(0)).toContain('L:');
    ex.controller.vm.error = 'plain';
    await nextTick();
    ex.controller.vm.error = null;
    // Successful metric change hits assign + emit.
    ex.selectMetric('amount_sum');
    await nextTick();
    ex.selectMetric('count');
    await nextTick();

    (mounted.props as any).stacked = false;
    await nextTick();
    (mounted.props as any).cumulative = true;
    await nextTick();
    (mounted.props as any).sort = 'asc';
    await nextTick();
    (mounted.props as any).metricAlias = 'amount_sum';
    await nextTick();
    (mounted.props as any).metricAlias = 'missing';
    await nextTick();
    (mounted.props as any).chartType = 'line';
    await nextTick();
    (mounted.props as any).chartType = 'bar';
    await nextTick();
    (mounted.props as any).metrics = [{ alias: 'only', label: 'Only' }];
    await nextTick();

    ex.selectMetric('only');
    ex.selectMetric('only'); // same alias no-op
    ex.onMetricSelect(null);
    ex.onMetricSelect('');
    ex.onMetricSelect('only');
    // Unknown alias while metrics list is non-empty.
    (mounted.props as any).metrics = [{ alias: 'only', label: 'Only' }];
    await nextTick();
    ex.selectMetric('nope');
    // Empty metrics list allows any alias through the known-list guard.
    (mounted.props as any).metrics = [];
    await nextTick();
    ex.selectMetric('custom');
    (mounted.props as any).metrics = [{ alias: 'only', label: 'Only' }];
    await nextTick();
    ex.xTickFormat(0.3);
    ex.xTickFormat(-1);
    ex.xTickFormat(99);
    ex.selectSort('desc');
    ex.selectSort('none');
    ex.toggleStacked();
    await nextTick();
    // Narrow types so availableTypes watch remaps pie → bar.
    (mounted.props as any).chartType = 'pie';
    await nextTick();
    (mounted.props as any).chartTypes = ['bar', 'line'];
    await nextTick();
    (mounted.props as any).chartTypes = ['bar', 'line', 'pie'];
    await nextTick();

    const mark = document.createElement('div');
    ex.onStackedBarClick({ index: 0, stackIndex: 1 }, {}, 0);
    ex.onStackedBarTooltip({ datum: { index: 0 }, stackIndex: 0, index: 0 }, 0, [mark]);
    ex.onStackedBarTooltip(undefined);
    ex.toggleStacked();
    await nextTick();
    ex.onGroupedBarClick({ index: 0 }, {}, 1);
    ex.onGroupedBarTooltip({ index: 0 }, 1, [mark]);
    ex.onGroupedBarTooltip(undefined);

    await ex.onRefresh();
    expect(loadErr).toBeTruthy();
    ex.onSearch({ keyword: 'x', appliedFilters: [], appliedGroups: [] });
    await flushPromises();
    expect(searchErr).toBeTruthy();

    ex.controller.apply = async () => {
      ex.controller.vm.result = nestedGroupResult;
    };
    await ex.applyQuery({ keyword: 'kw', appliedGroups: ['TokenType'] });
    await nextTick();

    // Cleared snapshot → tooltip handlers hit null-spec branches.
    ex.controller.vm.result = { kind: 'list', rows: [] };
    ex.rebuildFromSnapshot();
    await nextTick();
    const mark2 = document.createElement('div');
    ex.onStackedBarTooltip({ index: 0 }, 0, [mark2]);
    ex.onGroupedBarTooltip({ index: 0 }, 0, [mark2]);
    ex.onPieSegmentTooltip({ index: 0 }, 0, [mark2]);

    ex.syncPlotHeight();
    ex.xAccessor({ index: 1 });
    ex.yTickFormat(12);
    ex.xTickFormat(0);
    ex.xTickFormat(Number.NaN);
    ex.initMetricOptions();
    await ex.bootstrap().catch(() => undefined);

    mounted.unmount();
  });
  test('empty chartTypes keeps raw chartType in watch', async () => {
    const mounted = mountApp(ChoyChartView as any, {
      reactiveProps: true,
      props: {
        store: makeStore(['TokenType']),
        autoBootstrap: false,
        chartTypes: [],
        chartType: 'bar',
        metrics: [{ alias: 'count', label: 'Count' }],
      },
    });
    await flushPromises();
    const st = mounted.setupState() as {
      localChartType?: { value: string } | string;
      availableTypes?: { value: string[] } | string[];
    };
    const types = () => {
      const t = st.availableTypes;
      return t && typeof t === 'object' && 'value' in t ? t.value : (t as string[]);
    };
    const localType = () => {
      const t = st.localChartType;
      return t && typeof t === 'object' && 'value' in t ? t.value : (t as string);
    };
    expect(types()?.length ?? 0).toBe(0);
    (mounted.props as any).chartType = 'line';
    await nextTick();
    expect(localType()).toBe('line');

    (mounted.props as any).chartTypes = ['bar', 'line'];
    await nextTick();
    (mounted.props as any).chartType = 'line';
    await nextTick();
    expect(localType()).toBe('line');
    // Narrowing allowed types remaps an unsupported current type via availableTypes watch.
    (mounted.props as any).chartTypes = ['bar'];
    await nextTick();
    expect(localType()).toBe('bar');
    mounted.unmount();
  });

  test('autoBootstrap catches load errors', async () => {
    const errors: unknown[] = [];
    const mounted = mountApp(ChoyChartView as any, {
      props: {
        store: makeStore(['TokenType']),
        autoBootstrap: true,
        metrics: [],
        onLoadError: (e: unknown) => errors.push(e),
        beforeBootstrap: async () => {
          throw new Error('before boom');
        },
      },
    });
    await flushPromises();
    expect(errors.length).toBeGreaterThan(0);
    mounted.unmount();
  });

  test('autoBootstrap ResizeObserver syncs plot height', async () => {
    const PrevRO = (globalThis as any).ResizeObserver;
    const resizeCallbacks: Array<() => void> = [];
    (globalThis as any).ResizeObserver = class {
      constructor(cb: (entries?: unknown, observer?: unknown) => void) {
        resizeCallbacks.push(() => cb([], undefined));
      }
      observe() {
        const last = resizeCallbacks[resizeCallbacks.length - 1];
        if (last) last();
      }
      disconnect() {}
    };
    try {
      const mounted = mountApp(ChoyChartView as any, {
        props: {
          store: makeStore(['TokenType']),
          autoBootstrap: true,
          metrics: [{ alias: 'count', label: 'Count' }],
        },
      });
      await flushPromises();
      for (const run of resizeCallbacks) run();
      const ex = mounted.root as ChartExpose;
      ex.controller.vm.result = flatGroupResult;
      ex.rebuildFromSnapshot();
      await nextTick();
      mounted.unmount();
    } finally {
      (globalThis as any).ResizeObserver = PrevRO;
    }
  });

  test('percent stackMode normalizes series', async () => {
    const mounted = mountApp(ChoyChartView as any, {
      props: {
        store: makeStore(['TokenType']),
        autoBootstrap: false,
        chartType: 'bar',
        stacked: true,
        stackMode: 'percent',
        sort: 'asc',
        metrics: [{ alias: 'count', label: 'Count' }],
      },
    });
    await flushPromises();
    const ex = mounted.root as ChartExpose;
    ex.controller.vm.result = nestedGroupResult;
    ex.rebuildFromSnapshot();
    await nextTick();
    expect(ex.yTickFormat(50)).toContain('%');
    ex.selectSort('desc');
    ex.selectChartType('line');
    // Percent stays bar-only; line stacked uses absolute column-sum domain.
    expect(ex.yTickFormat(50)).not.toContain('%');
    const lineStackedDomain = (
      ex as ChartExpose & { linePlotYDomain: () => [number, number] }
    ).linePlotYDomain();
    // Nested Open+Done totals 30/category; single-series max is 25.
    expect(lineStackedDomain[1]).toBeGreaterThan(25);
    // Line keeps Stack control (Odoo); toggle off → unstacked series max.
    ex.toggleStacked();
    await nextTick();
    const lineUnstackedDomain = (
      ex as ChartExpose & { linePlotYDomain: () => [number, number] }
    ).linePlotYDomain();
    expect(lineUnstackedDomain[1]).toBeGreaterThan(0);
    // Fixture has multi-series column sums > max series, so unstacked max must drop.
    expect(lineUnstackedDomain[1]).toBeLessThan(lineStackedDomain[1]);
    ex.toggleCumulative();
    await nextTick();
    mounted.unmount();
  });

  test('line stacked dual series uses column-sum Y domain (Token-like)', async () => {
    const mounted = mountApp(ChoyChartView as any, {
      props: {
        store: makeStore(['TokenType', 'Revoked']),
        autoBootstrap: false,
        chartType: 'line',
        stacked: true,
        metrics: [{ alias: 'count', label: 'Count' }],
      },
    });
    await flushPromises();
    const ex = mounted.root as ChartExpose & { linePlotYDomain: () => [number, number] };
    ex.controller.vm.result = nestedGroupResult;
    ex.rebuildFromSnapshot();
    await nextTick();
    // Nested Open+Done: Jan 30 / Feb 30 → stacked domain above a single-series max (25).
    const domain = ex.linePlotYDomain();
    expect(domain[1]).toBeGreaterThan(25);
    expect(mounted.q('[data-testid=choy-chart-stack]')).not.toBeNull();
    // Unovis does not invoke y accessors in jsdom; exercise stacked tops for patch coverage.
    const setup = mounted.setupState() as Record<string, any>;
    const accessors = (setup.linePlotYAccessors?.value ?? setup.linePlotYAccessors) as Array<
      (d: Record<string, string | number>) => number
    >;
    const rows = (setup.xyRows?.value ?? setup.xyRows) as Array<Record<string, string | number>>;
    expect(accessors.length).toBeGreaterThan(1);
    expect(rows.length).toBeGreaterThan(0);
    const tops = accessors.map((fn) => fn(rows[0]!));
    expect(tops[0]).toBeGreaterThan(0);
    expect(tops[1]).toBeGreaterThan(tops[0]!);
    ex.toggleStacked();
    await nextTick();
    expect(ex.linePlotYDomain()[1]).toBeLessThanOrEqual(domain[1]);
    mounted.unmount();
  });

  test('pie tooltips and clicks cover segment handlers', async () => {
    const mounted = mountApp(ChoyChartView as any, {
      props: {
        store: makeStore(['TokenType', 'Revoked']),
        autoBootstrap: false,
        chartType: 'pie',
        metrics: [{ alias: 'count', label: 'Count' }],
      },
    });
    await flushPromises();
    const ex = mounted.root as ChartExpose;
    ex.controller.vm.result = nestedGroupResult;
    ex.rebuildFromSnapshot();
    await nextTick();

    const setup = mounted.setupState() as Record<string, any>;
    const pieSpec = setup.spec?.value ?? setup.spec;
    const slices = pieSpec?.slices || [];
    expect(slices.length > 0).toBe(true);

    const mark = document.createElement('div');
    for (let i = 0; i < slices.length; i++) {
      const sl = slices[i]!;
      expect(
        ex.onPieSegmentTooltip(
          { data: { key: sl.key, name: sl.name, value: sl.value }, index: i },
          i,
          [mark],
        ),
      ).toBeTruthy();
      ex.onPieSegmentTooltip({ name: sl.name }, i, [mark]);
      ex.onPieSegmentClick({ data: { key: sl.key, name: sl.name }, index: i });
      ex.onPieSegmentClick({ name: sl.name });
      ex.onPieSegmentClick({ index: i });
    }
    ex.onPieSegmentTooltip({}, 0, [mark]);
    ex.onPieSegmentTooltip({ index: 99 }, 99, [mark]);
    ex.onPieSegmentClick({ index: 99 });
    ex.onPieSegmentClick({ data: { key: 'missing' }, name: slices[0]!.name });

    mounted.unmount();
  });
});
