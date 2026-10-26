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
        'user-actions': () => h('button', { type: 'button', 'data-test': 'user-act' }, 'Extra'),
      },
    });
    await flushPromises();

    const bar = mounted.q('[data-slot=view-action-bar]') as HTMLElement | null;
    expect(bar).not.toBeNull();
    expect(bar?.getAttribute('data-layout')).toBe('thirds');
    expect(String(bar?.className || '')).toContain('border-b-0');

    const trays = mounted.qa('[data-testid=choy-action-tray]');
    expect(trays.length).toBe(2);
    expect(trays[0]?.getAttribute('aria-label')).toBe('System actions');
    expect(trays[1]?.getAttribute('aria-label')).toBe('User actions');

    const controls = mounted.q('[data-region=chart-controls]');
    expect(controls).not.toBeNull();
    expect(controls?.className || '').toContain('min-h-control');

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
      controller: { vm: { loading: boolean; error: unknown } };
    };
    exposed.controller.vm.error = new Error('chart failed');
    await nextTick();
    expect(mounted.text()).toContain('chart failed');
    expect(mounted.text()).toContain('Retry');

    exposed.controller.vm.error = null;
    exposed.controller.vm.loading = true;
    await nextTick();
    expect(mounted.q('[data-testid=choy-spinner]')).not.toBeNull();
    expect(mounted.q('[data-testid=choy-empty]')).toBeNull();

    exposed.controller.vm.loading = false;
    await nextTick();
    expect(mounted.q('[data-testid=choy-empty]')).not.toBeNull();

    mounted.unmount();
  });

  test('omits empty user-actions tray and keeps thirds center without search', async () => {
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
});
