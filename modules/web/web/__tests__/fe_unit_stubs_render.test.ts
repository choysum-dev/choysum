// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { h } from 'vue';
import { ToggleGroupItem } from 'reka-ui';
import { VisXYContainer } from '@unovis/vue';

import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';

describe('FE unit stubs render paths', () => {
  test('reka-ui ToggleGroupItem stub renders a button', async () => {
    const { unmount, q } = mountApp(ToggleGroupItem as any, {
      props: { value: 'And' },
      slots: { default: () => 'AND' },
    });
    await flushPromises();
    const btn = q('[data-reka-stub="ToggleGroupItem"]') as HTMLElement | null;
    expect(btn).toBeTruthy();
    expect(btn?.getAttribute('data-value')).toBe('And');
    expect(btn?.textContent || '').toContain('AND');
    unmount();
  });

  test('unovis/vue VisXYContainer stub renders a host div', async () => {
    const { unmount, q } = mountApp(VisXYContainer as any, {
      slots: { default: () => h('span', { class: 'child' }, 'chart') },
    });
    await flushPromises();
    const host = q('[data-stub="VisXYContainer"]') as HTMLElement | null;
    expect(host).toBeTruthy();
    expect(q('.child')?.textContent || '').toContain('chart');
    unmount();
  });
});
