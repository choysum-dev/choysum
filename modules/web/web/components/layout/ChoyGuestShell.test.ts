// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, h, ref } from 'vue';
import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import ChoyGuestShell from './ChoyGuestShell.vue';

describe('ChoyGuestShell', () => {
  beforeEach(async () => {
    const { pinViewportWidth } = await import('../../stores/layoutStore/pinViewport');
    const { createPinia, setActivePinia } = await import('pinia');
    pinViewportWidth(1280);
    setActivePinia(createPinia());
  });

  test('renders guest canvas chrome without the app rail', async () => {
    const mounted = mountApp(ChoyGuestShell as any, {
      stubs: {
        'router-view': { setup: () => () => h('div', { 'data-test': 'router-view' }) },
      },
    });
    await flushPromises();
    expect(mounted.q('[data-testid=choy-shell]')?.getAttribute('data-shell-mode')).toBe('guest');
    expect(mounted.q('.choy-shell')?.className || '').toContain('choy-shell--guest');
    expect(mounted.q('[data-testid=choy-shell-header]')).not.toBeNull();
    expect(mounted.q('.choy-guest-header')?.className || '').toContain('bg-background/95');
    expect(mounted.q('.choy-guest-header')?.className || '').toContain('backdrop-blur');
    expect(mounted.q('.choy-guest-header')?.className || '').not.toContain('border-b');
    expect(mounted.q('[data-testid=choy-shell-footer]')?.className || '').not.toContain('border-t');
    expect(mounted.q('[data-testid=choy-shell-scroll]')?.className || '').toContain('choy-shell__main');
    expect(mounted.q('[data-testid=choy-shell-scroll]')?.className || '').toContain('overflow-hidden');
    expect(mounted.q('[data-testid=choy-shell-canvas]')?.className || '').toContain('overflow-y-auto');
    expect(mounted.q('[data-testid=choy-shell-content]')).not.toBeNull();
    expect(mounted.q('.choy-shell__scroll-inner')).toBeNull();
    expect(mounted.q('[data-testid=choy-shell-app-rail]')).toBeNull();
    expect(mounted.q('[data-testid=choy-shell-aside]')).toBeNull();
    expect(mounted.q('[data-testid=choy-shell-menu-trigger]')).toBeNull();
    expect(mounted.q('[data-testid=choy-shell-brand]')).toBeNull();
    expect(mounted.q('[data-testid=choy-guest-nav-home]')).not.toBeNull();
    expect(mounted.q('[data-testid=choy-guest-mobile-nav]')).not.toBeNull();
    expect(mounted.q('[data-testid=choy-guest-mobile-nav]')?.className || '').toContain('lg:hidden');
    expect(mounted.q('[data-slot=navigation-menu]')?.className || '').toContain('hidden');
    expect(mounted.q('[data-slot=navigation-menu]')?.className || '').toContain('lg:flex');
    expect(mounted.q('[data-testid=choy-app-footer]')?.textContent || '').toContain('Powered by Choysum');
    expect(mounted.q('[data-testid=choy-app-footer-logo]')).not.toBeNull();
    expect(mounted.q('[data-testid=choy-app-footer]')?.className || '').toContain('leading-snug');
    expect(mounted.q('[data-testid=choy-shell-header-tools]')).not.toBeNull();
    expect(mounted.q('[data-testid=choy-shell-theme]')).not.toBeNull();
    expect(
      (mounted.q('.choy-guest-header') as HTMLElement | null)?.style?.height || '',
    ).toContain('--choy-layout-chrome-height');
    expect(
      (mounted.q('[data-testid=choy-shell-footer]') as HTMLElement | null)?.style?.height || '',
    ).toContain('--choy-layout-footer-height');
    expect(mounted.q('[data-test=router-view]')).not.toBeNull();
    // Header, canvas, and footer are siblings on the guest frame.
    const frame = mounted.q('[data-testid=choy-shell-scroll]');
    const footer = mounted.q('[data-testid=choy-shell-footer]');
    const canvas = mounted.q('[data-testid=choy-shell-canvas]');
    expect(footer?.parentElement === frame).toBe(true);
    expect(canvas?.parentElement === frame).toBe(true);
    expect(mounted.q('[data-testid=choy-shell-header]')?.parentElement === frame).toBe(true);
    mounted.unmount();
  });

  test('hides header chrome when showHeader is false', async () => {
    const mounted = mountApp(ChoyGuestShell as any, {
      props: { showHeader: false, showFooter: false },
      stubs: {
        'router-view': { setup: () => () => h('div', { 'data-test': 'router-view' }) },
      },
    });
    await flushPromises();
    expect(mounted.q('[data-testid=choy-shell-header]')).toBeNull();
    expect(mounted.q('[data-test=router-view]')).not.toBeNull();
    mounted.unmount();
  });

  test('hides footer when showFooter is false', async () => {
    const mounted = mountApp(ChoyGuestShell as any, {
      props: { showHeader: true, showFooter: false },
      stubs: {
        'router-view': { setup: () => () => h('div', { 'data-test': 'router-view' }) },
      },
    });
    await flushPromises();
    expect(mounted.q('[data-testid=choy-shell-header]')).not.toBeNull();
    expect(mounted.q('[data-testid=choy-shell-footer]')).toBeNull();
    mounted.unmount();
  });

  test('default slot overrides router-view fallback', async () => {
    const mounted = mountApp(ChoyGuestShell as any, {
      props: { showHeader: false },
      slots: {
        default: () => h('div', { 'data-test': 'custom-body' }, 'Custom'),
      },
      stubs: {
        'router-view': { setup: () => () => h('div', { 'data-test': 'router-view' }) },
      },
    });
    await flushPromises();
    expect(mounted.q('[data-test=custom-body]')?.textContent).toBe('Custom');
    expect(mounted.q('[data-test=router-view]')).toBeNull();
    mounted.unmount();
  });

  test('renders the matched view when Component is present', async () => {
    const Page = defineComponent({
      name: 'DemoPage',
      setup: () => () => h('div', { 'data-test': 'page' }, 'ok'),
    });
    const mounted = mountApp(ChoyGuestShell as any, {
      props: { showHeader: false },
      stubs: {
        'router-view': {
          setup: (_props: any, { slots }: any) => {
            return () =>
              slots.default?.({
                Component: Page,
                route: { meta: {}, name: 'Demo', path: '/demo', fullPath: '/demo' },
              });
          },
        },
      },
    });
    await flushPromises();
    expect(mounted.q('[data-test=page]')?.textContent).toBe('ok');
    mounted.unmount();
  });

  test('skips rendering when router-view has no matched Component', async () => {
    const mounted = mountApp(ChoyGuestShell as any, {
      props: { showHeader: false },
      stubs: {
        'router-view': {
          setup: (_props: any, { slots }: any) => {
            return () =>
              slots.default?.({
                Component: undefined,
                route: { meta: {}, name: 'GuestLayout', path: '/', fullPath: '/' },
              });
          },
        },
      },
    });
    await flushPromises();
    expect(mounted.q('[data-testid=choy-shell]')).not.toBeNull();
    expect(mounted.q('[data-test=page]')).toBeNull();
    mounted.unmount();
  });

  test('remounts the view when fullPath changes', async () => {
    let mountCount = 0;
    const Page = defineComponent({
      name: 'GuestPage',
      setup() {
        mountCount += 1;
        return () => h('div', { 'data-test': 'page' }, `m${mountCount}`);
      },
    });

    const current = ref({
      Component: Page as any,
      route: {
        meta: {},
        name: 'Login',
        path: '/login',
        fullPath: '/login',
      },
    });

    const mounted = mountApp(ChoyGuestShell as any, {
      props: { showHeader: false },
      stubs: {
        'router-view': {
          setup: (_props: any, { slots }: any) => {
            return () => slots.default?.(current.value);
          },
        },
      },
    });
    await flushPromises();
    expect(mounted.q('[data-test=page]')?.textContent).toBe('m1');

    current.value = {
      Component: Page as any,
      route: {
        meta: {},
        name: 'Login',
        path: '/login',
        fullPath: '/login?next=/web',
      },
    };
    await flushPromises();
    expect(mounted.q('[data-test=page]')?.textContent).toBe('m2');
    expect(mountCount).toBe(2);
    mounted.unmount();
  });

  test('ignores meta.keepAlive and remounts across guest navigations', async () => {
    let mountCount = 0;
    const Page = defineComponent({
      name: 'KeepAliveFlagPage',
      setup() {
        mountCount += 1;
        return () => h('div', { 'data-test': 'page' }, `m${mountCount}`);
      },
    });
    const Other = defineComponent({
      name: 'OtherGuestPage',
      setup: () => () => h('div', { 'data-test': 'other' }, 'other'),
    });

    const current = ref({
      Component: Page as any,
      route: {
        meta: { keepAlive: true },
        name: 'Login',
        path: '/login',
        fullPath: '/login',
      },
    });
    const mounted = mountApp(ChoyGuestShell as any, {
      props: { showHeader: false },
      stubs: {
        'router-view': {
          setup: (_props: any, { slots }: any) => {
            return () => slots.default?.(current.value);
          },
        },
      },
    });
    await flushPromises();
    expect(mounted.q('[data-test=page]')?.textContent).toBe('m1');

    current.value = {
      Component: Other as any,
      route: {
        meta: { keepAlive: false },
        name: 'Register',
        path: '/register',
        fullPath: '/register',
      },
    };
    await flushPromises();
    expect(mounted.q('[data-test=other]')?.textContent).toBe('other');

    current.value = {
      Component: Page as any,
      route: {
        meta: { keepAlive: true },
        name: 'Login',
        path: '/login',
        fullPath: '/login',
      },
    };
    await flushPromises();
    expect(mounted.q('[data-test=page]')?.textContent).toBe('m2');
    expect(mountCount).toBe(2);
    mounted.unmount();
  });


  test('Home nav link navigates to / when a router is installed', async () => {
    const createFeStubRouter = (await import('vue-router') as any).createFeStubRouter;
    const { router } = createFeStubRouter({
      route: { path: '/login', fullPath: '/login', meta: {} },
    });
    const pushes: unknown[] = [];
    const originalPush = router.push?.bind(router);
    router.push = (to: unknown) => {
      pushes.push(to);
      return originalPush ? originalPush(to) : Promise.resolve();
    };
    const mounted = mountApp(ChoyGuestShell as any, {
      plugins: [router],
      stubs: {
        'router-view': { setup: () => () => h('div', { 'data-test': 'router-view' }) },
      },
    });
    await flushPromises();
    const home = mounted.q('[data-testid=choy-guest-nav-home]') as HTMLElement | null;
    expect(home).not.toBeNull();
    expect(home!.getAttribute('href')).toBe('/');
    const ctrlClick = new Event('click', { bubbles: true, cancelable: true }) as MouseEvent;
    Object.assign(ctrlClick, { button: 0, ctrlKey: true, metaKey: false, shiftKey: false, altKey: false });
    home!.dispatchEvent(ctrlClick);
    await flushPromises();
    expect(pushes).not.toContain('/');
    home!.click();
    await flushPromises();
    expect(pushes).toContain('/');
    mounted.unmount();
  });

  test('Home nav falls back to location.assign without a router', async () => {
    const assigns: string[] = [];
    const original = globalThis.location;
    Object.defineProperty(globalThis, 'location', {
      configurable: true,
      value: {
        assign: (url: string) => {
          assigns.push(url);
        },
      },
    });
    try {
      const mounted = mountApp(ChoyGuestShell as any, {
        stubs: {
          'router-view': { setup: () => () => h('div', { 'data-test': 'router-view' }) },
        },
      });
      await flushPromises();
      const home = mounted.q('[data-testid=choy-guest-nav-home]') as HTMLElement | null;
      expect(home).not.toBeNull();
      home!.click();
      await flushPromises();
      expect(assigns).toContain('/');
      mounted.unmount();
    } finally {
      Object.defineProperty(globalThis, 'location', {
        configurable: true,
        value: original,
      });
    }
  });

  test('renders Home nav label from translate fallback', async () => {
    const mounted = mountApp(ChoyGuestShell as any, {
      stubs: {
        'router-view': { setup: () => () => h('div', { 'data-test': 'router-view' }) },
      },
    });
    await flushPromises();
    expect(mounted.q('[data-testid=choy-shell]')?.getAttribute('data-shell-mode')).toBe('guest');
    expect(mounted.q('[data-testid=choy-shell-header]')).not.toBeNull();
    expect((mounted.q('[data-testid=choy-guest-nav-home]')?.textContent || '').trim()).toBe('Home');
    mounted.unmount();
  });

  test('omits Home nav on /web/setup while keeping header tools', async () => {
    const original = globalThis.location;
    Object.defineProperty(globalThis, 'location', {
      configurable: true,
      value: { ...(original as object), pathname: '/web/setup' },
    });
    try {
      const mounted = mountApp(ChoyGuestShell as any, {
        stubs: {
          'router-view': { setup: () => () => h('div', { 'data-test': 'router-view' }) },
        },
      });
      await flushPromises();
      expect(mounted.q('[data-testid=choy-guest-nav-home]')).toBeNull();
      expect(mounted.q('[data-testid=choy-guest-mobile-nav]')).toBeNull();
      expect(mounted.q('[data-testid=choy-shell-header-tools]')).not.toBeNull();
      mounted.unmount();
    } finally {
      Object.defineProperty(globalThis, 'location', {
        configurable: true,
        value: original,
      });
    }
  });

  test('mobile nav Home link navigates to / when a router is installed', async () => {
    const createFeStubRouter = (await import('vue-router') as any).createFeStubRouter;
    const { router } = createFeStubRouter({
      route: { path: '/login', fullPath: '/login', meta: {} },
    });
    const pushes: unknown[] = [];
    const originalPush = router.push?.bind(router);
    router.push = (to: unknown) => {
      pushes.push(to);
      return originalPush ? originalPush(to) : Promise.resolve();
    };
    const mounted = mountApp(ChoyGuestShell as any, {
      plugins: [router],
      stubs: {
        'router-view': { setup: () => () => h('div', { 'data-test': 'router-view' }) },
      },
    });
    await flushPromises();
    const home = mounted.q('[data-testid=choy-guest-mobile-nav-home]') as HTMLElement | null;
    expect(home).not.toBeNull();
    expect((home?.textContent || '').trim()).toBe('Home');
    expect(home!.getAttribute('href')).toBe('/');
    home!.click();
    await flushPromises();
    expect(pushes).toContain('/');
    mounted.unmount();
  });

  test('isUnmodifiedPrimaryClick accepts plain left clicks only', async () => {
    const { isUnmodifiedPrimaryClick } = await import('./guestNavActivate');
    expect(isUnmodifiedPrimaryClick({ button: 0 } as MouseEvent)).toBe(true);
    expect(isUnmodifiedPrimaryClick({} as MouseEvent)).toBe(true);
    expect(isUnmodifiedPrimaryClick({ button: 0, metaKey: true } as MouseEvent)).toBe(false);
    expect(isUnmodifiedPrimaryClick({ button: 0, ctrlKey: true } as MouseEvent)).toBe(false);
    expect(isUnmodifiedPrimaryClick({ button: 0, shiftKey: true } as MouseEvent)).toBe(false);
    expect(isUnmodifiedPrimaryClick({ button: 0, altKey: true } as MouseEvent)).toBe(false);
    expect(isUnmodifiedPrimaryClick({ button: 1 } as MouseEvent)).toBe(false);
  });
});

