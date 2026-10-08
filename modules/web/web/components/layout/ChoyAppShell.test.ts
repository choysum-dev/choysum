// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { h } from 'vue';
import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import ChoyAppShell from './ChoyAppShell.vue';

describe('ChoyAppShell', () => {
  beforeEach(async () => {
    const { pinViewportWidth } = await import('../../stores/layoutStore/pinViewport');
    const { createPinia, setActivePinia } = await import('pinia');
    pinViewportWidth(1280);
    setActivePinia(createPinia());
    const { resetInstalledMenu } = await import('@/core/web/menu');
    resetInstalledMenu();
  });

  test('defaults to header and sidebar and renders router-view', async () => {
    const mounted = mountApp(ChoyAppShell as any, {
      stubs: {
        'router-view': { setup: () => () => h('div', { 'data-test': 'router-view' }) },
      },
    });
    await flushPromises();
    expect(mounted.q('[data-testid=choy-shell]')?.getAttribute('data-shell-mode')).toBe('app');
    expect(mounted.q('[data-testid=choy-shell-header]')).not.toBeNull();
    expect(mounted.q('[data-testid=choy-shell-header]')?.className || '').toContain('sticky');
    expect(mounted.q('.choy-shell')?.className || '').toContain('overflow-hidden');
    expect(mounted.q('[data-testid=choy-shell-canvas]')?.className || '').toContain('overflow-y-auto');
    expect(mounted.q('[data-testid=choy-shell-header-sep]')).not.toBeNull();
    expect(mounted.q('[data-anchor="choy.shell.header-actions"]')).not.toBeNull();
    expect(mounted.q('[data-testid=choy-shell-nav-split]')).not.toBeNull();
    expect(mounted.q('[data-testid=choy-shell-nav-pane]')).not.toBeNull();
    expect(
      mounted.q('[data-testid=choy-shell-nav-pane-scroll]')?.className || '',
    ).toContain('overflow-auto');
    expect(mounted.q('[data-testid=choy-shell-app-rail]')).not.toBeNull();
    expect(mounted.q('[data-testid=choy-shell-aside]')).not.toBeNull();
    expect(mounted.q('[data-testid=choy-shell-footer]')).not.toBeNull();
    expect(mounted.q('[data-testid=choy-shell-footer]')?.className || '').not.toContain('border-t');
    expect(mounted.q('[data-testid=choy-app-footer]')?.textContent || '').toContain('Powered by Choysum');
    expect(mounted.q('[data-testid=choy-shell-menu-trigger]')).not.toBeNull();
    expect(mounted.q('[data-testid=choy-shell-brand]')).not.toBeNull();
    expect(mounted.q('[data-testid=choy-shell-command-trigger]')).not.toBeNull();
    expect(mounted.q('[data-test=router-view]')).not.toBeNull();
    mounted.unmount();
  });

  test('renders aside and footer chrome when enabled', async () => {
    const mounted = mountApp(ChoyAppShell as any, {
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
    expect(mounted.q('[data-anchor="choy.shell.header-actions"]')).not.toBeNull();
    expect(mounted.q('[data-test=header-action]')?.textContent).toBe('A');
    expect(mounted.q('[data-test=nav-link]')?.textContent).toBe('Home');
    expect(mounted.q('[data-test=footer-note]')?.textContent).toBe('Foot');
    mounted.unmount();
  });

  test('renders built-in aside chrome when showSidebar has no aside slot', async () => {
    const mounted = mountApp(ChoyAppShell as any, {
      props: { showHeader: true, showSidebar: true, showFooter: false },
      stubs: {
        'router-view': { setup: () => () => h('div', { 'data-test': 'router-view' }) },
      },
    });
    await flushPromises();
    expect(mounted.q('[data-testid=choy-shell-app-rail]')).not.toBeNull();
    expect(mounted.q('[data-testid=choy-shell-aside]')).not.toBeNull();
    mounted.unmount();
  });

  test('hides nav rail when showSidebar is false', async () => {
    const mounted = mountApp(ChoyAppShell as any, {
      props: { showHeader: true, showSidebar: false, showFooter: true },
      stubs: {
        'router-view': { setup: () => () => h('div', { 'data-test': 'router-view' }) },
      },
    });
    await flushPromises();
    expect(mounted.q('[data-testid=choy-shell]')?.getAttribute('data-shell-mode')).toBe('app');
    expect(mounted.q('[data-testid=choy-shell-app-rail]')).toBeNull();
    expect(mounted.q('[data-testid=choy-app-footer]')?.textContent || '').toContain('Powered by Choysum');
    mounted.unmount();
  });

  test('hides header chrome when showHeader is false', async () => {
    const mounted = mountApp(ChoyAppShell as any, {
      props: { showHeader: false, showSidebar: true, showFooter: false },
      stubs: {
        'router-view': { setup: () => () => h('div', { 'data-test': 'router-view' }) },
      },
    });
    await flushPromises();
    expect(mounted.q('[data-testid=choy-shell-header]')).toBeNull();
    expect(mounted.q('[data-test=router-view]')).not.toBeNull();
    mounted.unmount();
  });

  test('default slot overrides router-view fallback', async () => {
    const mounted = mountApp(ChoyAppShell as any, {
      props: { showHeader: false, showSidebar: true },
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
    const mounted = mountApp(ChoyAppShell as any, {
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

    const mounted = mountApp(ChoyAppShell as any, {
      plugins: [menuPlugin, pinia, router, i18n],
      stubs: {
        'router-view': { setup: () => () => h('div', { 'data-test': 'router-view' }) },
      },
    });
    await flushPromises();
    expect(mounted.q('[data-testid=choy-shell-app-rail]')).not.toBeNull();
    expect(mounted.q('[data-testid=choy-shell-aside]')).not.toBeNull();
    expect(mounted.text()).toContain('Shell App');
    mounted.unmount();
  });
});
