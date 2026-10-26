// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { h, nextTick } from 'vue';
import { flushPromises, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import Popover from '../vendor/ui/popover/Popover.vue';
import PopoverContent from '../vendor/ui/popover/PopoverContent.vue';
import PopoverTrigger from '../vendor/ui/popover/PopoverTrigger.vue';
import ChoyGuestMobileNav from './ChoyGuestMobileNav.vue';

describe('ChoyGuestMobileNav', () => {
  beforeEach(() => {
    stubSfc(Popover as any, {
      name: 'Popover',
      props: {
        open: { type: Boolean, default: false },
      },
      emits: ['update:open'],
      setup(props: any, { slots }: any) {
        return () =>
          h(
            'div',
            { 'data-testid': 'guest-mobile-popover', 'data-open': props.open ? 'true' : 'false' },
            slots.default?.(),
          );
      },
    });
    stubSfc(PopoverTrigger as any, {
      name: 'PopoverTrigger',
      setup(_: any, { slots }: any) {
        return () => h('div', { class: 'popover-trigger' }, slots.default?.());
      },
    });
    stubSfc(PopoverContent as any, {
      name: 'PopoverContent',
      setup(_: any, { slots, attrs }: any) {
        return () =>
          h(
            'div',
            {
              class: 'popover-content',
              'data-testid': 'guest-mobile-panel',
              style: attrs.style as any,
            },
            slots.default?.(),
          );
      },
    });
  });

  afterEach(() => {
    restoreSfc(Popover as any);
    restoreSfc(PopoverTrigger as any);
    restoreSfc(PopoverContent as any);
  });

  test('renders menu trigger and item links from props', async () => {
    const selects: string[] = [];
    const mounted = mountApp(ChoyGuestMobileNav as any, {
      props: {
        class: 'flex lg:hidden',
        items: [
          {
            id: 'home',
            label: 'Home',
            href: '/',
            onSelect: () => selects.push('home'),
          },
          {
            id: 'docs',
            label: 'Docs',
            href: '/docs',
            onSelect: () => selects.push('docs'),
          },
        ],
      },
    });
    await flushPromises();
    const trigger = mounted.q('[data-testid=choy-guest-mobile-nav]') as HTMLButtonElement;
    expect(trigger).not.toBeNull();
    expect(trigger.getAttribute('aria-label') || '').toContain('Menu');
    expect(trigger.getAttribute('aria-expanded')).toBe('false');
    expect(trigger.className || '').toContain('lg:hidden');
    expect((mounted.q('[data-testid=choy-guest-mobile-nav-home]')?.textContent || '').trim()).toBe(
      'Home',
    );
    expect(mounted.q('[data-testid=choy-guest-mobile-nav-home]')?.getAttribute('href')).toBe('/');
    expect((mounted.q('[data-testid=choy-guest-mobile-nav-docs]')?.textContent || '').trim()).toBe(
      'Docs',
    );
    const panel = mounted.q('[data-testid=guest-mobile-panel]') as HTMLElement | null;
    expect(panel).not.toBeNull();
    expect(panel!.style.width || String(panel!.getAttribute('style') || '')).toContain(
      'reka-popper-available-width',
    );
    mounted.unmount();
  });

  test('plain item click selects and closes; modifier click does not', async () => {
    const selects: string[] = [];
    const mounted = mountApp(ChoyGuestMobileNav as any, {
      props: {
        items: [
          {
            id: 'home',
            label: 'Home',
            href: '/',
            onSelect: () => selects.push('home'),
          },
        ],
      },
    });
    await flushPromises();
    const trigger = mounted.q('[data-testid=choy-guest-mobile-nav]') as HTMLButtonElement;
    expect(trigger.getAttribute('aria-expanded')).toBe('false');

    // setupState unwraps refs; assign the boolean to open the panel.
    const state = mounted.setupState() as { open?: boolean } | undefined;
    expect(state).toBeTruthy();
    expect(typeof state!.open).toBe('boolean');
    state!.open = true;
    await nextTick();
    expect(trigger.getAttribute('aria-expanded')).toBe('true');
    expect(mounted.q('[data-testid=guest-mobile-popover]')?.getAttribute('data-open')).toBe('true');

    const home = mounted.q('[data-testid=choy-guest-mobile-nav-home]') as HTMLElement;
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
    expect(selects).toEqual([]);
    expect(ctrlClick.defaultPrevented).toBe(false);
    expect(state!.open).toBe(true);

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
    expect(selects).toEqual(['home']);
    expect(plainClick.defaultPrevented).toBe(true);
    expect(state!.open).toBe(false);
    expect(trigger.getAttribute('aria-expanded')).toBe('false');
    mounted.unmount();
  });
});
