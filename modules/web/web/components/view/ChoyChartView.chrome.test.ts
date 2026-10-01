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

    const trays = mounted.qa('[data-testid=choy-action-tray]');
    expect(trays.length).toBe(2);
    expect(trays[0]?.getAttribute('aria-label')).toBe('System actions');
    expect(trays[1]?.getAttribute('aria-label')).toBe('User actions');

    const controls = mounted.q('[data-region=chart-controls]');
    expect(controls).not.toBeNull();
    expect(controls?.className || '').toContain('min-h-[var(--choy-control-height)]');

    const metricSelect = mounted.q('select.choy-input') as HTMLSelectElement | null;
    expect(metricSelect).not.toBeNull();
    expect(String(metricSelect?.className || '')).toContain('h-control');
    expect(metricSelect?.getAttribute('aria-label')).toBe('Metric selection');
    expect(mounted.text()).toContain('Metric');
    expect(mounted.text()).toContain('Stack');
    expect(mounted.text()).toContain('New');
    expect(mounted.text()).toContain('Refresh');
    expect(mounted.q('[data-test=user-act]')).not.toBeNull();

    const exposed = mounted.root as {
      controller: { vm: { loading: boolean; error: unknown } };
    };
    exposed.controller.vm.error = new Error('chart failed');
    await nextTick();
    expect(mounted.text()).toContain('chart failed');
    expect(mounted.text()).toContain('Retry');

    mounted.unmount();
  });
});
