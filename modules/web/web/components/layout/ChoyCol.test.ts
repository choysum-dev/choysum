// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { h } from 'vue';
import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import ChoyCol from './ChoyCol.vue';
import ChoyGrid from './ChoyGrid.vue';

function mountInGrid(
  colProps: Record<string, unknown> = {},
  gridProps: Record<string, unknown> = {},
) {
  return mountApp(ChoyGrid as any, {
    props: gridProps,
    slots: {
      default: () =>
        h(ChoyCol as any, { ...colProps, 'data-test': 'col' }, () => 'cell'),
    },
  });
}

describe('ChoyCol', () => {
  test('sets 421 CSS vars when span is omitted under a responsive grid', async () => {
    const mounted = mountInGrid();
    await flushPromises();
    const col = mounted.q('[data-anchor="choy.col"]') as HTMLElement | null;
    expect(col).not.toBeNull();
    expect(col!.style.getPropertyValue('--choy-col-span-base').trim()).toBe('12');
    expect(col!.style.getPropertyValue('--choy-col-span-md').trim()).toBe('6');
    expect(col!.style.getPropertyValue('--choy-col-span-lg').trim()).toBe('3');
    mounted.unmount();
  });

  test('scales numeric span under responsive grid', async () => {
    const mounted = mountInGrid({ span: 3 });
    await flushPromises();
    const col = mounted.q('[data-anchor="choy.col"]') as HTMLElement;
    expect(col.style.getPropertyValue('--choy-col-span-lg').trim()).toBe('3');
    expect(col.style.getPropertyValue('--choy-col-span-md').trim()).toBe('6');
    expect(col.style.getPropertyValue('--choy-col-span-base').trim()).toBe('12');
    mounted.unmount();
  });

  test('keeps numeric span fixed when grid responsive is false', async () => {
    const mounted = mountInGrid({ span: 3 }, { responsive: false });
    await flushPromises();
    const col = mounted.q('[data-anchor="choy.col"]') as HTMLElement;
    expect(col.style.getPropertyValue('--choy-col-span-base').trim()).toBe('3');
    expect(col.style.getPropertyValue('--choy-col-span-md').trim()).toBe('3');
    expect(col.style.getPropertyValue('--choy-col-span-lg').trim()).toBe('3');
    mounted.unmount();
  });

  test('honors object span bands', async () => {
    const mounted = mountInGrid({ span: { base: 12, md: 6, lg: 3 } });
    await flushPromises();
    const col = mounted.q('[data-anchor="choy.col"]') as HTMLElement;
    expect(col.style.getPropertyValue('--choy-col-span-base').trim()).toBe('12');
    expect(col.style.getPropertyValue('--choy-col-span-md').trim()).toBe('6');
    expect(col.style.getPropertyValue('--choy-col-span-lg').trim()).toBe('3');
    mounted.unmount();
  });
});
