// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import { h } from 'vue';
import ChoyLayout from './ChoyLayout.vue';

describe('ChoyLayout mount', () => {
  test('renders overlay drawer semantics and dismisses via backdrop', async () => {
    let dismissed = 0;
    const mounted = mountApp(ChoyLayout as any, {
      props: {
        showHeader: true,
        showAside: true,
        asideOverlay: true,
        asideCollapsed: false,
        asideAriaLabel: 'Main navigation',
      },
      slots: {
        header: () => h('div', { 'data-test': 'hdr' }, 'H'),
        aside: () => h('div', { 'data-test': 'aside-body' }, 'Nav'),
        'aside-foot': () => h('div', { 'data-test': 'aside-foot' }, 'Attr'),
        default: () => h('div', { 'data-test': 'main' }, 'Canvas'),
      },
      on: {
        onAsideDismiss: () => {
          dismissed += 1;
        },
      },
    });
    await flushPromises();
    const aside = mounted.q('[data-testid=choy-layout-aside]') as HTMLElement | null;
    expect(aside).not.toBeNull();
    expect(aside?.getAttribute('role')).toBe('dialog');
    expect(aside?.getAttribute('aria-modal')).toBe('true');
    expect(aside?.getAttribute('aria-label')).toBe('Main navigation');
    const backdrop = mounted.q('[data-testid=choy-layout-aside-backdrop]') as HTMLElement | null;
    expect(backdrop).not.toBeNull();
    backdrop!.click();
    await flushPromises();
    expect(dismissed).toBe(1);
    mounted.unmount();
  });

  test('applies collapsed rail width token', async () => {
    const mounted = mountApp(ChoyLayout as any, {
      props: {
        showAside: true,
        asideCollapsed: true,
        asideOverlay: false,
      },
      slots: {
        aside: () => h('div', 'Nav'),
        default: () => h('div', 'Main'),
      },
    });
    await flushPromises();
    const aside = mounted.q('[data-testid=choy-layout-aside]') as HTMLElement | null;
    expect(aside?.style.width || '').toContain('collapsed');
    mounted.unmount();
  });
});
