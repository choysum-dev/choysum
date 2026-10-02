// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, h, onActivated, ref } from 'vue';
import { flushPromises, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import ChoyWebShell from './ChoyWebShell.vue';
import ChoyLayout from './ChoyLayout.vue';

describe('ChoyWebShell', () => {
  beforeEach(async () => {
    const { pinViewportWidth } = await import('../../stores/layoutStore/pinViewport');
    const { createPinia, setActivePinia } = await import('pinia');
    pinViewportWidth(1280);
    setActivePinia(createPinia());
    stubSfc(ChoyLayout, {
      props: {
        showHeader: { type: Boolean, default: undefined },
        showAside: { type: Boolean, default: undefined },
        showFooter: { type: Boolean, default: undefined },
        asideOverlay: { type: Boolean, default: false },
        asideCollapsed: { type: Boolean, default: false },
        class: null,
      },
      setup: ((props: any, { slots }: any) => {
        return () =>
          h(
            'div',
            {
              'data-test': 'choy-layout',
              'data-header': String(props.showHeader),
              'data-aside': String(props.showAside),
              'data-footer': String(props.showFooter),
              'data-overlay': String(!!props.asideOverlay),
              'data-collapsed': String(!!props.asideCollapsed),
            },
            [
              slots.header ? h('div', { 'data-test': 'slot-header' }, slots.header()) : null,
              slots.aside ? h('div', { 'data-test': 'slot-aside' }, slots.aside()) : null,
              slots['aside-foot']
                ? h('div', { 'data-test': 'slot-aside-foot' }, slots['aside-foot']())
                : null,
              slots.default?.(),
              slots.footer ? h('div', { 'data-test': 'slot-footer' }, slots.footer()) : null,
            ],
          );
      }) as any,
    });
  });

  afterEach(() => {
    restoreSfc(ChoyLayout);
  });

  test('defaults to header and sidebar and renders router-view', async () => {
    const mounted = mountApp(ChoyWebShell as any, {
      stubs: {
        'router-view': { setup: () => () => h('div', { 'data-test': 'router-view' }) },
      },
    });
    await flushPromises();
    const root = mounted.q('[data-test=choy-layout]');
    expect(root?.getAttribute('data-header')).toBe('true');
    expect(root?.getAttribute('data-aside')).toBe('true');
    expect(root?.getAttribute('data-footer')).toBe('false');
    expect(mounted.q('[data-test=slot-header]')?.textContent).toContain('Choysum');
    expect(mounted.q('[data-test=slot-aside]')).not.toBeNull();
    expect(mounted.q('[data-test=slot-aside-foot]')).not.toBeNull();
    expect(mounted.q('[data-testid=choy-shell-attrib]')).not.toBeNull();
    expect(mounted.q('[data-test=slot-footer]')).toBeNull();
    expect(mounted.q('[data-test=router-view]')).not.toBeNull();
    mounted.unmount();
  });

  test('renders aside and footer chrome when enabled', async () => {
    const mounted = mountApp(ChoyWebShell as any, {
      props: { showHeader: true, showSidebar: true, showFooter: true },
      slots: {
        'header-actions': () => h('button', { 'data-test': 'header-action' }, 'A'),
        aside: () => h('a', { 'data-test': 'nav-link' }, 'Home'),
        footer: () => h('span', { 'data-test': 'footer-note' }, 'Foot'),
      },
      stubs: {
        'router-view': { setup: () => () => h('div', { 'data-test': 'router-view' }) },
      },
    });
    await flushPromises();
    const root = mounted.q('[data-test=choy-layout]');
    expect(root?.getAttribute('data-aside')).toBe('true');
    expect(root?.getAttribute('data-footer')).toBe('true');
    expect(mounted.q('[data-test=header-action]')?.textContent).toBe('A');
    expect(mounted.q('[data-test=nav-link]')?.textContent).toBe('Home');
    expect(mounted.q('[data-test=footer-note]')?.textContent).toBe('Foot');
    mounted.unmount();
  });

  test('renders built-in aside chrome when showSidebar has no aside slot', async () => {
    const mounted = mountApp(ChoyWebShell as any, {
      props: { showHeader: true, showSidebar: true, showFooter: false },
      stubs: {
        'router-view': { setup: () => () => h('div', { 'data-test': 'router-view' }) },
      },
    });
    await flushPromises();
    const root = mounted.q('[data-test=choy-layout]');
    expect(root?.getAttribute('data-aside')).toBe('true');
    expect(mounted.q('[data-test=slot-aside]')).not.toBeNull();
    mounted.unmount();
  });

  test('skips empty footer chrome when showFooter lacks footer slot', async () => {
    const mounted = mountApp(ChoyWebShell as any, {
      props: { showHeader: true, showSidebar: false, showFooter: true },
      stubs: {
        'router-view': { setup: () => () => h('div', { 'data-test': 'router-view' }) },
      },
    });
    await flushPromises();
    const root = mounted.q('[data-test=choy-layout]');
    expect(root?.getAttribute('data-footer')).toBe('false');
    expect(mounted.q('[data-test=slot-footer]')).toBeNull();
    mounted.unmount();
  });

  test('hides header chrome when showHeader is false', async () => {
    const mounted = mountApp(ChoyWebShell as any, {
      props: { showHeader: false, showSidebar: false, showFooter: false },
      stubs: {
        'router-view': { setup: () => () => h('div', { 'data-test': 'router-view' }) },
      },
    });
    await flushPromises();
    expect(mounted.q('[data-test=slot-header]')).toBeNull();
    expect(mounted.q('[data-test=router-view]')).not.toBeNull();
    mounted.unmount();
  });

  test('default slot overrides router-view fallback', async () => {
    const mounted = mountApp(ChoyWebShell as any, {
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
      return mountApp(ChoyWebShell as any, {
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
    const mounted = mountApp(ChoyWebShell as any, {
      props: { showHeader: false },
      stubs: {
        'router-view': {
          setup: (_props: any, { slots }: any) => {
            return () =>
              slots.default?.({
                Component: undefined,
                route: { meta: { keepAlive: false }, name: 'AppLayout', path: '/', fullPath: '/' },
              });
          },
        },
      },
    });
    await flushPromises();
    expect(mounted.q('[data-test=choy-layout]')).not.toBeNull();
    expect(mounted.q('[data-test=page]')).toBeNull();
    expect(mounted.q('[data-test=cached]')).toBeNull();
    mounted.unmount();
  });

  test('treats missing route.meta as a non-keepAlive view', async () => {
    const Page = defineComponent({
      name: 'NoMetaPage',
      setup: () => () => h('div', { 'data-test': 'page' }, 'ok'),
    });
    const mounted = mountApp(ChoyWebShell as any, {
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

    const mounted = mountApp(ChoyWebShell as any, {
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

    const mounted = mountApp(ChoyWebShell as any, {
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

    const mounted = mountApp(ChoyWebShell as any, {
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

  test('hides aside chrome when route.meta.isAuthPage is set', async () => {
    const createFeStubRouter = (await import('vue-router') as any).createFeStubRouter;
    const { router } = createFeStubRouter({
      route: { path: '/login', fullPath: '/login', meta: { isAuthPage: true } },
    });
    const mounted = mountApp(ChoyWebShell as any, {
      plugins: [router],
      stubs: {
        'router-view': { setup: () => () => h('div', { 'data-test': 'router-view' }) },
      },
    });
    await flushPromises();
    const root = mounted.q('[data-test=choy-layout]');
    expect(root?.getAttribute('data-aside')).toBe('false');
    expect(mounted.q('[data-test=slot-aside]')).toBeNull();
    expect(mounted.q('[data-test=slot-header]')?.textContent).toContain('Choysum');
    mounted.unmount();
  });

  test('brand link navigates home when a router is installed', async () => {
    const createFeStubRouter = (await import('vue-router') as any).createFeStubRouter;
    const { router } = createFeStubRouter({
      route: { path: '/other', fullPath: '/other', meta: {} },
    });
    const pushes: unknown[] = [];
    const originalPush = router.push?.bind(router);
    router.push = (to: unknown) => {
      pushes.push(to);
      return originalPush ? originalPush(to) : Promise.resolve();
    };
    const mounted = mountApp(ChoyWebShell as any, {
      plugins: [router],
      stubs: {
        'router-view': { setup: () => () => h('div', { 'data-test': 'router-view' }) },
      },
    });
    await flushPromises();
    const brand = mounted.q('.choy-shell__brand') as HTMLElement | null;
    expect(brand).not.toBeNull();
    brand!.click();
    await flushPromises();
    expect(pushes).toContain('/meta/modules');
    mounted.unmount();
  });

  test('wires built-in sidebar menu when pinia and menu plugin are installed', async () => {
    const createFeStubRouter = (await import('vue-router') as any).createFeStubRouter;
    const { createPinia, setActivePinia } = await import('pinia');
    const { createMenuPlugin } = await import('@/core/web/menu');
    const { createI18n } = await import('vue-i18n');

    const menuPlugin = createMenuPlugin();
    menuPlugin.manager.addMenu({
      id: 'shell-app',
      title: 'Shell App',
      path: '/shell-app',
    } as any);
    const pinia = createPinia();
    setActivePinia(pinia);
    const { router } = createFeStubRouter({
      route: { path: '/meta/modules', fullPath: '/meta/modules', meta: {} },
    });
    const i18n = createI18n({ legacy: false, locale: 'en', messages: { en: {} } });

    const mounted = mountApp(ChoyWebShell as any, {
      plugins: [menuPlugin, pinia, router, i18n],
      stubs: {
        'router-view': { setup: () => () => h('div', { 'data-test': 'router-view' }) },
      },
    });
    await flushPromises();
    expect(mounted.q('[data-test=slot-aside]')).not.toBeNull();
    expect(mounted.text()).toContain('Shell App');
    mounted.unmount();
  });
});
