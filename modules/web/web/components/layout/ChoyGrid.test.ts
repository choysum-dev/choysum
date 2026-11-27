// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { h } from 'vue';
import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import ChoyGrid from './ChoyGrid.vue';

describe('ChoyGrid', () => {
  test('defaults to twelve tracks and the medium gap class', async () => {
    const mounted = mountApp(ChoyGrid as any, {
      slots: {
        default: () => h('div', { 'data-test': 'cell' }, 'c'),
      },
    });
    await flushPromises();
    const root = mounted.q('[data-anchor="choy.grid"]') as HTMLElement | null;
    expect(root).not.toBeNull();
    expect(root!.className).toContain('gap-2');
    expect(root!.style.gridTemplateColumns).toContain('repeat(12');
    expect(root!.getAttribute('data-responsive')).toBe('true');
    expect(mounted.q('[data-test=cell]')?.textContent).toBe('c');
    mounted.unmount();
  });

  test('honors responsive=false', async () => {
    const mounted = mountApp(ChoyGrid as any, { props: { responsive: false } });
    await flushPromises();
    const root = mounted.q('[data-anchor="choy.grid"]') as HTMLElement | null;
    expect(root!.getAttribute('data-responsive')).toBe('false');
    mounted.unmount();
  });

  test('applies explicit gap variants', async () => {
    for (const [gap, cls] of [
      ['none', 'gap-0'],
      ['sm', 'gap-2'],
      ['lg', 'gap-6'],
    ] as const) {
      const mounted = mountApp(ChoyGrid as any, { props: { gap, cols: 4 } });
      await flushPromises();
      const root = mounted.q('[data-anchor="choy.grid"]') as HTMLElement | null;
      expect(root!.className).toContain(cls);
      expect(root!.style.gridTemplateColumns).toContain('repeat(4');
      mounted.unmount();
    }
  });
});
