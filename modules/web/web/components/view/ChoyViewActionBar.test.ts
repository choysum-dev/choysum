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
    expect(bar).not.toBeNull();
    expect(bar!.getAttribute('data-layout')).toBe('thirds');
    const leading = mounted.q('[data-slot=view-action-bar-leading]') as HTMLElement | null;
    expect(leading).not.toBeNull();
    expect(leading!.querySelector('[data-testid=mid]')).toBeNull();
    expect(mounted.q('[data-testid=mid]')?.textContent).toBe('M');
    // Leading is the first toolbar child so center stays in column 2.
    const first = bar!.children[0] as HTMLElement | undefined;
    expect(first).toBeTruthy();
    expect(first!.getAttribute('data-slot')).toBe('view-action-bar-leading');
    expect(first).toBe(leading!);
    mounted.unmount();
  });

  test('thirds layout pads an empty center column when #center is omitted', async () => {
    const mounted = mountApp(ChoyViewActionBar as any, {
      props: { layout: 'thirds' },
      slots: {
        'system-actions': () => h('button', { 'data-testid': 'sys' }, 'S'),
        'header-right': () => h('button', { 'data-testid': 'right' }, 'R'),
      },
    });
    await flushPromises();
    expect(mounted.q('[data-slot=view-action-bar-center]')).not.toBeNull();
    expect(mounted.q('[data-testid=sys]')?.textContent).toBe('S');
    expect(mounted.q('[data-testid=right]')?.textContent).toBe('R');
    mounted.unmount();

    const split = mountApp(ChoyViewActionBar as any, {
      props: { layout: 'split' },
      slots: {
        'system-actions': () => h('button', { 'data-testid': 'sys' }, 'S'),
      },
    });
    await flushPromises();
    expect(split.q('[data-slot=view-action-bar-center]')).toBeNull();
    split.unmount();
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

  test('identity (title + ⋯) precedes actions cluster', async () => {
    const mounted = mountApp(ChoyViewActionBar as any, {
      props: { layout: 'thirds' },
      slots: {
        identity: () => [
          h('h1', { 'data-testid': 'title' }, 'Partners'),
          h('button', { 'data-testid': 'more' }, '⋯'),
        ],
        'system-actions': () => h('button', { 'data-testid': 'sys' }, 'New'),
      },
    });
    await flushPromises();
    const leading = mounted.q('[data-slot=view-action-bar-leading]') as HTMLElement | null;
    expect(leading).not.toBeNull();
    expect(mounted.q('[data-slot=view-action-bar-identity]')).not.toBeNull();
    expect(mounted.q('[data-slot=view-action-bar-actions]')).not.toBeNull();
    expect(mounted.q('[data-testid=title]')?.textContent).toBe('Partners');
    expect(mounted.q('[data-testid=sys]')?.textContent).toBe('New');
    expect(mounted.q('[data-testid=more]')?.textContent).toBe('⋯');
    // Leading text order: title → ⋯ (identity) → New (actions).
    const text = (leading!.textContent || '').replace(/\s+/g, ' ');
    const titleIdx = text.indexOf('Partners');
    const moreIdx = text.indexOf('⋯');
    const newIdx = text.indexOf('New');
    expect(titleIdx).toBeGreaterThanOrEqual(0);
    expect(moreIdx).toBeGreaterThan(titleIdx);
    expect(newIdx).toBeGreaterThan(moreIdx);
    mounted.unmount();
  });
});
