// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import { h, nextTick } from 'vue';
import ChoyChartView from './ChoyChartView.vue';

function makeStore() {
  return {
    storeId: 'web.chart.chrome.test',
    state: { queryState: {} },
  } as any;
}

describe('ChoyChartView chrome', () => {
  test('renders ActionTray chrome, i18n controls, and Retry on error', async () => {
    const store = makeStore();
    const mounted = mountApp(ChoyChartView as any, {
      props: {
        store,
        autoBootstrap: false,
        showCreate: true,
        metrics: [{ alias: 'count', label: 'Count' }],
      },
      slots: {
        'contextual-actions': () => h('button', { type: 'button', 'data-test': 'user-act' }, 'Extra'),
      },
    });
    await flushPromises();

    const bar = mounted.q('[data-slot=view-action-bar]') as HTMLElement | null;
    expect(bar).not.toBeNull();
    expect(bar?.getAttribute('data-layout')).toBe('thirds');
    expect(String(bar?.className || '')).toContain('border-b-0');

    const trays = mounted.qa('[data-testid=choy-action-tray]');
    expect(trays.length).toBe(1);
    expect(trays[0]?.getAttribute('aria-label')).toBe('System actions');
    expect(mounted.q('[data-testid=chart-contextual-actions]')).not.toBeNull();
    expect(mounted.q('[data-test=user-act]')).not.toBeNull();

    const controls = mounted.q('[data-region=chart-controls]');
    expect(controls).not.toBeNull();
    expect(controls?.className || '').toContain('min-h-control');
    // Vertical rhythm is view-header gap; controls must not add their own py.
    expect(` ${controls?.className || ''} `).not.toContain(' pb-3 ');

    const chartRoot = mounted.q('[data-anchor="choy.chart-view"]') as HTMLElement | null;
    expect(chartRoot!.className).toContain('gap-2');
    expect(chartRoot!.className.split(/\s+/)).not.toContain('gap-5');

    const host = mounted.q('[data-slot=view-body-host]') as HTMLElement | null;
    expect(host!.getAttribute('data-sheet-pad')).toBe('board');
    expect(host!.className).toContain('p-[var(--choy-layout-section-gap)]');
    expect(host!.className).not.toContain('p-[var(--choy-view-sheet-pad)]');

    const metricTrigger = mounted.q('[data-testid=choy-chart-metric]') as HTMLElement | null;
    expect(metricTrigger).not.toBeNull();
    expect(metricTrigger?.getAttribute('aria-label')).toBe('Metric selection');
    // Empty / single-series: stack chrome is hidden (same as pie).
    expect(mounted.q('[data-testid=choy-chart-stack]')).toBeNull();
    expect(mounted.text()).toContain('New');
    expect(mounted.q('[data-testid=choy-chart-type-bar]')).not.toBeNull();
    expect(mounted.q('[data-testid=choy-chart-type-group]')).not.toBeNull();
    expect(mounted.q('[data-test=user-act]')).not.toBeNull();

    expect(mounted.q('[data-testid=page-io-menu-trigger]')).not.toBeNull();
    mounted.click('[data-testid=page-io-menu-trigger]');
    await flushPromises();
    expect(mounted.q('[data-testid=chart-overflow-refresh]')).not.toBeNull();
    expect(mounted.text()).toContain('Refresh');

    const body = mounted.q('[data-region=chart-body]') as HTMLElement | null;
    expect(body).not.toBeNull();
    expect(String(body?.className || '')).toContain('flex-1');

    const exposed = mounted.root as {
      controller: { vm: { loading: boolean; error: unknown }; apply: () => Promise<unknown> };
      onRetry: () => Promise<void>;
    };
    const ctrlForError = exposed.controller.vm as {
      loading: boolean;
      error: unknown;
      hasAppliedOnce?: boolean;
    };
    ctrlForError.error = new Error('chart failed');
    ctrlForError.hasAppliedOnce = true;
    await nextTick();
    // BusyHost surface error: generic title + Error.message detail (not raw-only overlay).
    expect(mounted.q('[data-testid=choy-view-surface-error]')).not.toBeNull();
    expect(mounted.text()).toContain('Failed to load');
    expect(mounted.text()).toContain('chart failed');
    expect(mounted.text()).toContain('Retry');
    // prepError is null here → onRetry falls through to onRefresh.
    let applyCalls = 0;
    exposed.controller.apply = async () => {
      applyCalls += 1;
      return [];
    };
    await exposed.onRetry();
    await flushPromises();
    expect(applyCalls).toBe(1);

    const ctrlVm = exposed.controller.vm as {
      loading: boolean;
      error: unknown;
      hasAppliedOnce?: boolean;
    };
    // Concurrent Retry while busy must be ignored.
    ctrlVm.loading = true;
    await nextTick();
    await exposed.onRetry();
    expect(applyCalls).toBe(1);

    ctrlVm.error = null;
    ctrlVm.hasAppliedOnce = false;
    ctrlVm.loading = true;
    await nextTick();
    // Empty chart first frame → block Loader2; no-frame busy paints overlay immediately.
    expect(mounted.q('[data-slot=view-body-host]')?.getAttribute('aria-busy')).toBe('true');
    expect(mounted.q('[data-slot=view-busy-overlay]')?.getAttribute('data-mode')).toBe('block');
    expect(mounted.q('[data-testid=choy-empty]')).toBeNull();

    ctrlVm.loading = false;
    await nextTick();
    // Still gated until first apply settles.
    expect(mounted.q('[data-testid=choy-empty]')).toBeNull();

    ctrlVm.hasAppliedOnce = true;
    await nextTick();
    const empty = mounted.q('[data-testid=choy-empty]') as HTMLElement | null;
    expect(empty).not.toBeNull();
    expect(empty!.className).toContain('items-center');
    expect(empty!.className).toContain('justify-center');
    expect(mounted.q('[data-region=chart-empty]')).not.toBeNull();

    // Soft-fail Cancel dismisses prep/controller errors without requiring Retry.
    ctrlVm.error = new Error('soft fail');
    const chartState = mounted.setupState() as {
      onSurfaceCancel: () => void;
      prepError: unknown;
    };
    (chartState as any).prepError = new Error('prep');
    chartState.onSurfaceCancel();
    await nextTick();
    expect(ctrlVm.error).toBeNull();
    expect((chartState as any).prepError == null).toBe(true);

    mounted.unmount();
  });

  test('omits empty contextual center and keeps thirds layout without search', async () => {
    const store = makeStore();
    const mounted = mountApp(ChoyChartView as any, {
      props: {
        store,
        autoBootstrap: false,
        showCreate: false,
        metrics: [{ alias: 'count', label: 'Count' }],
      },
    });
    await flushPromises();

    const trays = mounted.qa('[data-testid=choy-action-tray]');
    expect(trays.length).toBe(1);
    expect(trays[0]?.getAttribute('aria-label')).toBe('System actions');
    expect(mounted.q('[data-slot=view-action-bar]')?.getAttribute('data-layout')).toBe('thirds');

    mounted.unmount();
  });

  test('pie uses a capped centered square host; controls stay justify-between', async () => {
    const store = {
      storeId: 'web.chart.chrome.pie',
      state: { queryState: { appliedGroups: ['IsActive'] } },
    } as any;
    const mounted = mountApp(ChoyChartView as any, {
      props: {
        store,
        autoBootstrap: false,
        chartType: 'pie',
        metrics: [{ alias: 'count', label: 'Count' }],
      },
    });
    await flushPromises();

    const controls = mounted.q('[data-region=chart-controls]') as HTMLElement | null;
    expect(controls?.className || '').toContain('justify-between');
    expect(mounted.q('[data-testid=choy-chart-stack]')).toBeNull();
    expect(mounted.q('[data-testid=choy-chart-sort-group]')).toBeNull();
    expect(mounted.q('[data-testid=choy-chart-type-group]')).not.toBeNull();

    const exposed = mounted.root as {
      syncPlotHeight: () => void;
      pieSide: { value: number } | number;
      rebuildFromSnapshot: () => void;
      controller: { vm: Record<string, unknown> };
    };
    exposed.controller.vm.result = {
      kind: 'group',
      rows: [
        { kind: 'group', depth: 0, key: 'true', label: 'true', count: 10 },
        { kind: 'group', depth: 0, key: 'false', label: 'false', count: 2 },
      ],
    };
    exposed.controller.vm.hasAppliedOnce = true;
    exposed.rebuildFromSnapshot();
    await nextTick();

    const frame = mounted.q('[data-region=chart-pie-frame]') as HTMLElement | null;
    expect(frame).not.toBeNull();
    expect(frame!.className).toContain('items-center');
    expect(frame!.className).toContain('justify-center');
    const plot = mounted.q('[data-region=chart-pie-plot]') as HTMLElement | null;
    expect(plot).not.toBeNull();
    // Diameter cap math is unit-tested in resolvePiePlotSize; here assert the
    // square host is wired from pieSide (min side until the body is measured).
    const side =
      typeof exposed.pieSide === 'number' ? exposed.pieSide : Number(exposed.pieSide?.value);
    expect(side).toBeGreaterThanOrEqual(280);
    expect(side).toBeLessThanOrEqual(480);
    expect(plot!.style.width).toBe(`${side}px`);
    expect(plot!.style.height).toBe(`${side}px`);

    mounted.unmount();
  });
});
