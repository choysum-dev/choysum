// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { h } from 'vue';
import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import ChoyViewContainer from './ChoyViewContainer.vue';

describe('ChoyViewContainer', () => {
  test('renders header/body/footer regions and default gap-3', async () => {
    const mounted = mountApp(ChoyViewContainer as any, {
      slots: {
        header: () => h('div', { 'data-testid': 'hdr' }, 'H'),
        default: () => h('div', { 'data-testid': 'body' }, 'B'),
        footer: () => h('div', { 'data-testid': 'ftr' }, 'F'),
      },
    });
    await flushPromises();
    const root = mounted.q('[data-slot=view-container]') as HTMLElement | null;
    expect(root).not.toBeNull();
    expect(root?.className || '').toContain('gap-3');
    const header = mounted.q('[data-slot=view-header]') as HTMLElement | null;
    expect(header?.querySelector('[data-testid=hdr]')?.textContent).toBe('H');
    expect(mounted.q('[data-slot=view-body]')?.getAttribute('data-anchor')).toBe('choy.view.sheet');
    expect(mounted.q('[data-testid=body]')?.textContent).toBe('B');
    const footer = mounted.q('[data-slot=view-footer]') as HTMLElement | null;
    expect(footer?.querySelector('[data-testid=ftr]')?.textContent).toBe('F');
    mounted.unmount();
  });

  test('skips default gap when class already sets a full gap utility', async () => {
    const withGap = mountApp(ChoyViewContainer as any, {
      props: { class: 'gap-6' },
      slots: { default: () => h('div') },
    });
    await flushPromises();
    expect(withGap.q('[data-slot=view-container]')?.className || '').toContain('gap-6');
    expect(withGap.q('[data-slot=view-container]')?.className || '').not.toContain('gap-3');
    withGap.unmount();

    const axisOnly = mountApp(ChoyViewContainer as any, {
      props: { class: 'gap-x-4' },
      slots: { default: () => h('div') },
    });
    await flushPromises();
    // Axis-specific gap must not suppress the container default.
    expect(axisOnly.q('[data-slot=view-container]')?.className || '').toContain('gap-3');
    axisOnly.unmount();
  });

  test('omits header/footer wrappers when those slots are empty', async () => {
    const mounted = mountApp(ChoyViewContainer as any, {
      slots: { default: () => h('div', { 'data-testid': 'body' }, 'only') },
    });
    await flushPromises();
    expect(mounted.q('[data-slot=view-header]')).toBeNull();
    expect(mounted.q('[data-slot=view-footer]')).toBeNull();
    expect(mounted.q('[data-testid=body]')?.textContent).toBe('only');
    mounted.unmount();
  });
});
