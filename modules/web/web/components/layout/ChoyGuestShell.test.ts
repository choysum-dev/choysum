// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, h, onActivated, ref } from 'vue';
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
    expect(mounted.q('.choy-guest-header')?.className || '').toContain('border-border/40');
    expect(mounted.q('.choy-guest-header')?.className || '').toContain('backdrop-blur');
    expect(mounted.q('[data-testid=choy-shell-header-sep]')).toBeNull();
    expect(mounted.q('[data-testid=choy-shell-footer]')?.className || '').toContain('border-t');
    expect(mounted.q('[data-testid=choy-shell-app-rail]')).toBeNull();
    expect(mounted.q('[data-testid=choy-shell-aside]')).toBeNull();
    expect(mounted.q('[data-testid=choy-shell-menu-trigger]')).toBeNull();
    expect((mounted.q('[data-testid=choy-shell-brand]')?.textContent || '').trim()).toBe('Choysum');
    expect(mounted.q('[data-testid=choy-guest-nav-home]')).not.toBeNull();
    expect(mounted.q('[data-testid=choy-app-footer]')?.textContent || '').toContain('Powered by Choysum');
    expect(mounted.q('[data-testid=choy-app-footer]')?.className || '').toContain('leading-loose');
    expect(mounted.q('[data-test=router-view]')).not.toBeNull();
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

  test('honors route.meta.keepAlive when rendering the matched view', async () => {
    const Page = defineComponent({
      name: 'DemoPage',
      setup: () => () => h('div', { 'data-test': 'page' }, 'ok'),
    });

    function mountWithMeta(keepAlive: boolean) {
      return mountApp(ChoyGuestShell as any, {
        props: { showHeader: false },
        stubs: {
          'router-view': {
            setup: (_props: any, { slots }: any) => {
              return () =>
                slots.default?.({
                  Component: Page,
                  route: { meta: { keepAlive }, name: 'Demo', path: '/demo', fullPath: '/demo' },
                });
            },
          },
        },
      });
    }

    const cached = mountWithMeta(true);
    await flushPromises();
    expect(cached.q('[data-test=page]')?.textContent).toBe('ok');
    cached.unmount();

    const plain = mountWithMeta(false);
    await flushPromises();
    expect(plain.q('[data-test=page]')?.textContent).toBe('ok');
    plain.unmount();
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
                route: { meta: { keepAlive: false }, name: 'GuestLayout', path: '/', fullPath: '/' },
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

  test('treats missing route.meta as a non-keepAlive view', async () => {
    const Page = defineComponent({
      name: 'NoMetaPage',
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
                route: { name: 'Bare', path: '/bare', fullPath: '/bare' },
              });
          },
        },
      },
    });
    await flushPromises();
    expect(mounted.q('[data-test=page]')?.textContent).toBe('ok');
    mounted.unmount();
  });

  test('remounts keepAlive views when path params change under the same name', async () => {
    let mountCount = 0;
    const CachedPage = defineComponent({
      name: 'ParamCachedPage',
      setup() {
        mountCount += 1;
        return () => h('div', { 'data-test': 'cached' }, `m${mountCount}`);
      },
    });

    const current = ref({
      Component: CachedPage as any,
      route: {
        meta: { keepAlive: true },
        name: 'Record',
        path: '/records/1',
        fullPath: '/records/1',
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
    expect(mounted.q('[data-test=cached]')?.textContent).toBe('m1');

    current.value = {
      Component: CachedPage as any,
      route: {
        meta: { keepAlive: true },
        name: 'Record',
        path: '/records/2',
        fullPath: '/records/2',
      },
    };
    await flushPromises();
    expect(mounted.q('[data-test=cached]')?.textContent).toBe('m2');
    expect(mountCount).toBe(2);
    mounted.unmount();
  });

  test('reuses keepAlive cache when only the query string changes', async () => {
    let mountCount = 0;
    const CachedPage = defineComponent({
      name: 'QueryCachedPage',
      setup() {
        mountCount += 1;
        return () => h('div', { 'data-test': 'cached' }, `m${mountCount}`);
      },
    });

    const current = ref({
      Component: CachedPage as any,
      route: {
        meta: { keepAlive: true },
        name: 'Modules',
        path: '/meta/modules',
        fullPath: '/meta/modules?tab=a',
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
    expect(mounted.q('[data-test=cached]')?.textContent).toBe('m1');

    current.value = {
      Component: CachedPage as any,
      route: {
        meta: { keepAlive: true },
        name: 'Modules',
        path: '/meta/modules',
        fullPath: '/meta/modules?tab=b',
      },
    };
    await flushPromises();
    expect(mounted.q('[data-test=cached]')?.textContent).toBe('m1');
    expect(mountCount).toBe(1);
    mounted.unmount();
  });

  test('keeps cached views alive across a non-keepAlive navigation', async () => {
    let mountCount = 0;
    let activateCount = 0;
    const CachedPage = defineComponent({
      name: 'CachedPage',
      setup() {
        mountCount += 1;
        onActivated(() => {
          activateCount += 1;
        });
        return () => h('div', { 'data-test': 'cached' }, `m${mountCount}`);
      },
    });
    const PlainPage = defineComponent({
      name: 'PlainPage',
      setup: () => () => h('div', { 'data-test': 'plain' }, 'plain'),
    });

    const current = ref({
      Component: CachedPage as any,
      route: { meta: { keepAlive: true }, name: 'Cached', path: '/cached', fullPath: '/cached' },
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
    expect(mounted.q('[data-test=cached]')?.textContent).toBe('m1');
    expect(mountCount).toBe(1);
    const activatesAfterFirst = activateCount;

    current.value = {
      Component: PlainPage as any,
      route: { meta: { keepAlive: false }, name: 'Plain', path: '/plain', fullPath: '/plain' },
    };
    await flushPromises();
    expect(mounted.q('[data-test=plain]')?.textContent).toBe('plain');

    current.value = {
      Component: CachedPage as any,
      route: { meta: { keepAlive: true }, name: 'Cached', path: '/cached', fullPath: '/cached' },
    };
    await flushPromises();
    expect(mounted.q('[data-test=cached]')?.textContent).toBe('m1');
    expect(mountCount).toBe(1);
    expect(activateCount).toBeGreaterThan(activatesAfterFirst);
    mounted.unmount();
  });

  test('brand link navigates home when a router is installed', async () => {
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
    const brand = mounted.q('[data-testid=choy-shell-brand]') as HTMLElement | null;
    expect(brand).not.toBeNull();
    brand!.click();
    await flushPromises();
    expect(pushes).toContain('/meta/modules');
    mounted.unmount();
  });

  test('Home nav link navigates to the land path when a router is installed', async () => {
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
    home!.click();
    await flushPromises();
    expect(pushes).toContain('/meta/modules');
    mounted.unmount();
  });

  test('uses i18n when the plugin is installed', async () => {
    const { createI18n } = await import('vue-i18n');
    const sourceMessages = (await import('../../i18n/source')).default;
    const i18n = createI18n({
      legacy: false,
      locale: 'en',
      messages: { en: sourceMessages as any },
    });
    const mounted = mountApp(ChoyGuestShell as any, {
      plugins: [i18n],
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
});
