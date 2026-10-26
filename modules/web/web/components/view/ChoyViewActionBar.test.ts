// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { h } from 'vue';
import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import ChoyViewActionBar from './ChoyViewActionBar.vue';

describe('ChoyViewActionBar', () => {
  test('split layout hosts system/user trays, center, and header-right', async () => {
    const mounted = mountApp(ChoyViewActionBar as any, {
      props: {
        layout: 'split',
        systemActionsLabel: 'Sys',
        userActionsLabel: 'Usr',
      },
      slots: {
        'system-actions': () => h('button', { 'data-testid': 'sys' }, 'S'),
        'user-actions': () => h('button', { 'data-testid': 'usr' }, 'U'),
        center: () => h('input', { 'data-testid': 'center' }),
        'header-right': () => h('button', { 'data-testid': 'right' }, 'R'),
      },
    });
    await flushPromises();
    expect(mounted.q('[data-testid=choy-view-action-bar-host]')).not.toBeNull();
    const bar = mounted.q('[data-slot=view-action-bar]') as HTMLElement | null;
    expect(bar?.getAttribute('data-layout')).toBe('split');
    expect(mounted.q('[data-slot=view-action-bar-leading]')).not.toBeNull();
    expect(mounted.q('[data-testid=sys]')?.textContent).toBe('S');
    expect(mounted.q('[data-testid=usr]')?.textContent).toBe('U');
    expect(mounted.q('[data-testid=center]')).not.toBeNull();
    expect(mounted.q('[data-testid=right]')?.textContent).toBe('R');
    expect(mounted.q('[data-slot=search-fit-row]')).not.toBeNull();
    mounted.unmount();
  });

  test('thirds layout keeps the leading column even without action slots', async () => {
    const mounted = mountApp(ChoyViewActionBar as any, {
      props: { layout: 'thirds' },
      slots: {
        center: () => h('span', { 'data-testid': 'mid' }, 'M'),
      },
    });
    await flushPromises();
    const bar = mounted.q('[data-slot=view-action-bar]') as HTMLElement | null;
    expect(bar?.getAttribute('data-layout')).toBe('thirds');
    const leading = mounted.q('[data-slot=view-action-bar-leading]') as HTMLElement | null;
    expect(leading).not.toBeNull();
    expect(leading?.querySelector('[data-testid=mid]')).toBeNull();
    expect(mounted.q('[data-testid=mid]')?.textContent).toBe('M');
    // Leading is the first toolbar child so center stays in column 2.
    const kids = Array.from(bar?.children || []) as HTMLElement[];
    expect(kids[0]?.getAttribute('data-slot')).toBe('view-action-bar-leading');
    expect(kids[0]).toBe(leading);
    mounted.unmount();
  });

  test('border-b-0 class drops the hairline padding air', async () => {
    const mounted = mountApp(ChoyViewActionBar as any, {
      props: { class: 'border-b-0' },
    });
    await flushPromises();
    const bar = mounted.q('[data-slot=view-action-bar]') as HTMLElement | null;
    expect(bar?.className || '').toContain('border-b-0');
    // Hairline air (pb-2) is only added when the bottom border is present.
    expect(` ${bar?.className || ''} `).not.toContain(' pb-2 ');
    mounted.unmount();
  });
});
