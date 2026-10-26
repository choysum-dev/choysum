// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { h } from 'vue';
import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import ChoyGuestHeader from './ChoyGuestHeader.vue';

describe('ChoyGuestHeader', () => {
  test('renders desktop Home link, mobile nav, and header tools', async () => {
    const homes: number[] = [];
    const mounted = mountApp(ChoyGuestHeader as any, {
      props: {
        goHome: () => {
          homes.push(1);
        },
        homeHref: '/',
      },
    });
    await flushPromises();
    expect(mounted.q('[data-testid=choy-shell-header]')?.className || '').toContain(
      'choy-guest-header',
    );
    expect(
      (mounted.q('[data-testid=choy-shell-header]') as HTMLElement | null)?.style?.height || '',
    ).toContain('--choy-layout-chrome-height');
    const home = mounted.q('[data-testid=choy-guest-nav-home]') as HTMLElement | null;
    expect(home).not.toBeNull();
    expect(home!.getAttribute('href')).toBe('/');
    expect((home!.textContent || '').trim()).toBe('Home');
    expect(mounted.q('[data-testid=choy-guest-mobile-nav]')).not.toBeNull();
    expect(mounted.q('[data-testid=choy-shell-header-tools]')).not.toBeNull();
    expect(mounted.q('[data-testid=choy-shell-header-sep]')).toBeNull();
    mounted.unmount();
  });

  test('plain click calls goHome; modifier click keeps native navigation', async () => {
    const homes: number[] = [];
    const mounted = mountApp(ChoyGuestHeader as any, {
      props: {
        goHome: () => {
          homes.push(1);
        },
        homeHref: '/land',
      },
      slots: {
        'header-actions': () => h('button', { 'data-testid': 'guest-extra' }, 'Extra'),
      },
    });
    await flushPromises();
    expect(mounted.q('[data-testid=choy-shell-header-sep]')).not.toBeNull();
    expect(mounted.q('[data-testid=guest-extra]')).not.toBeNull();

    const home = mounted.q('[data-testid=choy-guest-nav-home]') as HTMLElement;
    const ctrlClick = new Event('click', { bubbles: true, cancelable: true }) as MouseEvent;
    Object.assign(ctrlClick, {
      button: 0,
      ctrlKey: true,
      metaKey: false,
      shiftKey: false,
      altKey: false,
    });
    home.dispatchEvent(ctrlClick);
    await flushPromises();
    expect(homes).toEqual([]);
    expect(ctrlClick.defaultPrevented).toBe(false);

    const plainClick = new Event('click', { bubbles: true, cancelable: true }) as MouseEvent;
    Object.assign(plainClick, {
      button: 0,
      ctrlKey: false,
      metaKey: false,
      shiftKey: false,
      altKey: false,
    });
    home.dispatchEvent(plainClick);
    await flushPromises();
    expect(homes).toEqual([1]);
    expect(plainClick.defaultPrevented).toBe(true);
    mounted.unmount();
  });
});
