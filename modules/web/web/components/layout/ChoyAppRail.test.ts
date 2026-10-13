// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, h, nextTick } from 'vue';
import { createPinia, setActivePinia } from 'pinia';
import { createI18n } from 'vue-i18n';
import * as VueRouter from 'vue-router';
import { createMenuPlugin } from '@/core/web/menu';
import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import { useMenuStore } from '../../stores/menuStore';
import { useLayoutStore } from '../../stores/layoutStore';
import { pinViewportWidth } from '../../stores/layoutStore/pinViewport';
import ChoyAppRail from './ChoyAppRail.vue';
import { SidebarProvider } from '../vendor/ui/sidebar/index';

const createFeStubRouter = (VueRouter as any).createFeStubRouter;

function mountRail(opts: {
  menus: Array<Record<string, unknown>>;
  path?: string;
  collapsed?: boolean;
  withRouter?: boolean;
  withSidebar?: boolean;
  activeLeafId?: string;
}) {
  const menuPlugin = createMenuPlugin();
  for (const menu of opts.menus) {
    menuPlugin.manager.addMenu(menu as any);
  }
  const pinia = createPinia();
  setActivePinia(pinia);
  pinViewportWidth(opts.collapsed ? 900 : 1280);
  const layout = useLayoutStore();
  layout.sidebarMode = opts.collapsed ? 'collapsed' : 'expanded';

  const plugins: any[] = [menuPlugin, pinia];
  let pushes: unknown[] = [];
  if (opts.withRouter !== false) {
    const path = opts.path || '/meta/modules';
    const { router } = createFeStubRouter({
      route: { path, fullPath: path, meta: {} },
    });
    const originalPush = router.push?.bind(router);
    router.push = (to: unknown) => {
      pushes.push(to);
      return originalPush ? originalPush(to) : Promise.resolve();
    };
    plugins.push(router);
  }
  plugins.push(createI18n({ legacy: false, locale: 'en', messages: { en: {} } }));

  const Host = defineComponent({
    setup() {
      const store = useMenuStore();
      if (opts.activeLeafId) {
        const active = store.getMenu(opts.activeLeafId);
        if (active) store.setActiveMenu(active);
      }
      const rail = () => h(ChoyAppRail);
      if (opts.withSidebar === false) return rail;
      return () => h(SidebarProvider, null, { default: rail });
    },
  });
  const mounted = mountApp(Host as any, { plugins });
  return { mounted, pushes, layout };
}

describe('ChoyAppRail', () => {
  test('lists every visible app root and lands the first leaf on click when expanded', async () => {
    const { mounted, pushes } = mountRail({
      menus: [
        {
          id: 'meta',
          title: 'Module Management',
          children: [{ id: 'meta-board', title: 'Module Board', path: '/meta/modules' }],
        },
        {
          id: 'base',
          title: 'Master Data',
          children: [{ id: 'base-company', title: 'Company', path: '/base/companies' }],
        },
      ],
      activeLeafId: 'meta-board',
      collapsed: false,
    });
    await flushPromises();
    await nextTick();
    const items = Array.from(
      mounted.el.querySelectorAll('[data-testid=choy-shell-app-rail-item]'),
    ) as HTMLButtonElement[];
    expect(items.map((el) => (el.querySelector('.sr-only')?.textContent || '').trim())).toEqual([
      'Module Management',
      'Master Data',
    ]);
    expect(items[0]?.getAttribute('data-active')).toBe('true');
    const tiles = Array.from(
      mounted.el.querySelectorAll('[data-testid=choy-shell-app-rail-tile]'),
    ) as HTMLElement[];
    expect(tiles[0]?.className || '').toContain('bg-sidebar-primary');
    expect(tiles[1]?.className || '').not.toContain('bg-sidebar-primary');
    items[1]!.click();
    await flushPromises();
    expect(pushes).toContain('/base/companies');
    mounted.unmount();
  });

  test('collapsed click does not navigate the app root; flyout lists leaves and branch subs', async () => {
    const menuPlugin = createMenuPlugin();
    menuPlugin.manager.addMenu({
      id: 'base',
      title: 'Master Data',
      children: [
        { id: 'loose', title: 'Company', path: '/base/companies' },
        {
          id: 'units',
          title: 'Units',
          children: [{ id: 'unit', title: 'Unit', path: '/base/units' }],
        },
      ],
    } as any);
    const pinia = createPinia();
    setActivePinia(pinia);
    pinViewportWidth(900);
    useLayoutStore().sidebarMode = 'collapsed';
    const { router } = createFeStubRouter({
      route: { path: '/base/companies', fullPath: '/base/companies', meta: {} },
    });
    const pushes: unknown[] = [];
    const originalPush = router.push?.bind(router);
    router.push = (to: unknown) => {
      pushes.push(to);
      return originalPush ? originalPush(to) : Promise.resolve();
    };
    const i18n = createI18n({ legacy: false, locale: 'en', messages: { en: {} } });
    const Host = defineComponent({
      setup() {
        return () =>
          h(SidebarProvider, null, {
            default: () => h(ChoyAppRail),
          });
      },
    });
    const mounted = mountApp(Host as any, {
      plugins: [menuPlugin, pinia, router, i18n],
    });
    await flushPromises();
    await nextTick();
    const item = mounted.q('[data-testid=choy-shell-app-rail-item]') as HTMLButtonElement;
    item.click();
    await flushPromises();
    expect(pushes).toEqual([]);

    // Reka stubs do not emit open; drive the controlled flyout like a successful trigger.
    const railMounted = mountApp(ChoyAppRail as any, {
      plugins: [menuPlugin, pinia, router, i18n],
    });
    await flushPromises();
    await nextTick();
    const ss = railMounted.setupState();
    ss.setFlyoutOpen({ id: 'base', title: 'Master Data' }, true);
    await nextTick();
    expect(railMounted.q('[data-testid=choy-shell-app-rail-flyout]')).not.toBeNull();
    expect(railMounted.qa('[data-testid=choy-shell-app-rail-flyout-leaf]').length).toBeGreaterThan(0);
    expect(
      railMounted.q('[data-testid=choy-shell-app-rail-flyout-sub]')?.textContent?.trim(),
    ).toBe('Units');
    expect(railMounted.q('[data-testid=choy-shell-app-rail-flyout-group]')).toBeNull();
    mounted.unmount();
    railMounted.unmount();
  });

  test('collapsed flyout ignores dismiss while hover intent is active', async () => {
    const menuPlugin = createMenuPlugin();
    menuPlugin.manager.addMenu({
      id: 'base',
      title: 'Master Data',
      children: [{ id: 'loose', title: 'Company', path: '/base/companies' }],
    } as any);
    const pinia = createPinia();
    setActivePinia(pinia);
    pinViewportWidth(900);
    useLayoutStore().sidebarMode = 'collapsed';
    const { router } = createFeStubRouter({
      route: { path: '/', fullPath: '/', meta: {} },
    });
    const i18n = createI18n({ legacy: false, locale: 'en', messages: { en: {} } });
    const mounted = mountApp(ChoyAppRail as any, {
      plugins: [menuPlugin, pinia, router, i18n],
    });
    await flushPromises();
    await nextTick();
    const ss = mounted.setupState();
    const app = { id: 'base', title: 'Master Data' };
    ss.onFlyoutHoverEnter(app);
    await new Promise((r) => setTimeout(r, 120));
    expect(ss.openFlyoutKey).toBe('base');
    ss.setFlyoutOpen(app, false);
    expect(ss.openFlyoutKey).toBe('base');
    ss.onFlyoutHoverLeave(app);
    await new Promise((r) => setTimeout(r, 220));
    expect(ss.openFlyoutKey).toBeNull();
    mounted.unmount();
  });

  test('collapsed flyout stays open while pointer crosses into nested submenu', async () => {
    const menuPlugin = createMenuPlugin();
    menuPlugin.manager.addMenu({
      id: 'base',
      title: 'Master Data',
      children: [
        {
          id: 'units',
          title: 'Units',
          children: [{ id: 'unit', title: 'Unit', path: '/base/units' }],
        },
      ],
    } as any);
    const pinia = createPinia();
    setActivePinia(pinia);
    pinViewportWidth(900);
    useLayoutStore().sidebarMode = 'collapsed';
    const { router } = createFeStubRouter({
      route: { path: '/', fullPath: '/', meta: {} },
    });
    const i18n = createI18n({ legacy: false, locale: 'en', messages: { en: {} } });
    const mounted = mountApp(ChoyAppRail as any, {
      plugins: [menuPlugin, pinia, router, i18n],
    });
    await flushPromises();
    await nextTick();
    const ss = mounted.setupState();
    const app = { id: 'base', title: 'Master Data' };
    ss.onFlyoutHoverEnter(app);
    await new Promise((r) => setTimeout(r, 120));
    expect(ss.openFlyoutKey).toBe('base');
    // Leave L1 content toward portaled L2; Reka often dismisses in that gap.
    ss.onFlyoutHoverLeave(app);
    ss.setFlyoutOpen(app, false);
    expect(ss.openFlyoutKey).toBe('base');
    // Nested SubContent re-arms hover intent before the close grace ends.
    ss.onFlyoutHoverEnter(app);
    await new Promise((r) => setTimeout(r, 250));
    expect(ss.openFlyoutKey).toBe('base');
    ss.onFlyoutHoverLeave(app);
    await new Promise((r) => setTimeout(r, 220));
    expect(ss.openFlyoutKey).toBeNull();
    mounted.unmount();
  });

  test('collapsed flyout leaf select navigates', async () => {
    const menuPlugin = createMenuPlugin();
    menuPlugin.manager.addMenu({
      id: 'base',
      title: 'Master Data',
      children: [{ id: 'loose', title: 'Company', path: '/base/companies' }],
    } as any);
    const pinia = createPinia();
    setActivePinia(pinia);
    pinViewportWidth(900);
    useLayoutStore().sidebarMode = 'collapsed';
    const { router } = createFeStubRouter({
      route: { path: '/', fullPath: '/', meta: {} },
    });
    const pushes: unknown[] = [];
    const originalPush = router.push?.bind(router);
    router.push = (to: unknown) => {
      pushes.push(to);
      return originalPush ? originalPush(to) : Promise.resolve();
    };
    const i18n = createI18n({ legacy: false, locale: 'en', messages: { en: {} } });
    const mounted = mountApp(ChoyAppRail as any, {
      plugins: [menuPlugin, pinia, router, i18n],
    });
    await flushPromises();
    await nextTick();
    const ss = mounted.setupState();
    ss.setFlyoutOpen({ id: 'base', title: 'Master Data' }, true);
    await nextTick();
    ss.onFlyoutLeaf({ id: 'loose', title: 'Company', path: '/base/companies' });
    await flushPromises();
    expect(pushes).toContain('/base/companies');
    expect(ss.openFlyoutKey).toBeNull();
    mounted.unmount();
  });

  test('renders a first-letter fallback when the app root has no icon', async () => {
    const { mounted } = mountRail({
      menus: [
        {
          id: 'custom',
          title: '公司管理',
          children: [{ id: 'custom-leaf', title: 'Leaf', path: '/custom' }],
        },
      ],
      path: '/custom',
      collapsed: false,
    });
    await flushPromises();
    await nextTick();
    expect(mounted.q('[data-testid=choy-shell-app-rail-fallback]')?.textContent?.trim()).toBe('公');
    expect(mounted.q('[data-testid=choy-shell-app-rail-item]')?.querySelector('svg')).toBeNull();
    expect(mounted.q('[data-testid=choy-shell-app-rail-tooltip]')).toBeNull();
    mounted.unmount();
  });

  test('opens a tooltip portal when expanded and skips apps without a key', async () => {
    const menuPlugin = createMenuPlugin();
    menuPlugin.manager.addMenu({
      id: 'custom',
      title: '公司管理',
      children: [{ id: 'custom-leaf', title: 'Leaf', path: '/custom' }],
    } as any);
    const pinia = createPinia();
    setActivePinia(pinia);
    pinViewportWidth(1280);
    useLayoutStore().sidebarMode = 'expanded';
    const { router } = createFeStubRouter({
      route: { path: '/custom', fullPath: '/custom', meta: {} },
    });
    const i18n = createI18n({ legacy: false, locale: 'en', messages: { en: {} } });
    const mounted = mountApp(ChoyAppRail as any, {
      plugins: [menuPlugin, pinia, router, i18n],
    });
    await flushPromises();
    await nextTick();
    const ss = mounted.setupState();
    ss.onTooltipOpen({ id: '', path: '', title: '' }, true);
    ss.onTooltipOpen({ id: 'custom', title: '公司管理' }, true);
    await nextTick();
    expect(mounted.q('[data-testid=choy-shell-app-rail-tooltip]')).not.toBeNull();
    ss.onTooltipOpen({ id: 'custom', title: '公司管理' }, false);
    await nextTick();
    expect(mounted.q('[data-testid=choy-shell-app-rail-tooltip]')).toBeNull();
    mounted.unmount();
  });

  test('lists apps when the menu store is present without a router', async () => {
    const menuPlugin = createMenuPlugin();
    menuPlugin.manager.addMenu({
      id: 'custom',
      title: '公司管理',
      children: [{ id: 'custom-leaf', title: 'Leaf', path: '/custom' }],
    } as any);
    const pinia = createPinia();
    setActivePinia(pinia);
    pinViewportWidth(1280);
    useLayoutStore().sidebarMode = 'expanded';
    const i18n = createI18n({ legacy: false, locale: 'en', messages: { en: {} } });
    const Host = defineComponent({
      setup() {
        const store = useMenuStore();
        return () =>
          h(SidebarProvider, null, {
            default: () =>
              h('div', [
                h('span', {
                  'data-testid': 'menu-count',
                  'data-count': String(store.getMenus().length),
                }),
                h(ChoyAppRail),
              ]),
          });
      },
    });
    const mounted = mountApp(Host as any, {
      plugins: [menuPlugin, pinia, i18n],
    });
    await flushPromises();
    await nextTick();
    expect(mounted.q('[data-testid=menu-count]')?.getAttribute('data-count')).toBe('1');
    expect(mounted.qa('[data-testid=choy-shell-app-rail-item]').length).toBe(1);
    mounted.unmount();
  });

  test('renders without menu, i18n, or sidebar context', async () => {
    setActivePinia(undefined as any);
    const mounted = mountApp(ChoyAppRail as any);
    await flushPromises();
    expect(mounted.q('[data-testid=choy-shell-app-rail]')).not.toBeNull();
    expect(mounted.qa('[data-testid=choy-shell-app-rail-item]').length).toBe(0);
    mounted.unmount();
  });

  test('collapsed flyout closes without hover timers and clears pending open timers on expand', async () => {
    const menuPlugin = createMenuPlugin();
    menuPlugin.manager.addMenu({
      id: 'base',
      title: 'Master Data',
      children: [{ id: 'loose', title: 'Company', path: '/base/companies' }],
    } as any);
    const pinia = createPinia();
    setActivePinia(pinia);
    pinViewportWidth(900);
    const layout = useLayoutStore();
    layout.sidebarMode = 'collapsed';
    const { router } = createFeStubRouter({
      route: { path: '/', fullPath: '/', meta: {} },
    });
    const i18n = createI18n({ legacy: false, locale: 'en', messages: { en: {} } });
    const mounted = mountApp(ChoyAppRail as any, {
      plugins: [menuPlugin, pinia, router, i18n],
    });
    await flushPromises();
    await nextTick();
    const ss = mounted.setupState();
    const app = { id: 'base', title: 'Master Data' };
    ss.setFlyoutOpen(app, true);
    expect(ss.openFlyoutKey).toBe('base');
    ss.setFlyoutOpen(app, false);
    expect(ss.openFlyoutKey).toBeNull();

    ss.onFlyoutHoverEnter(app);
    // Expand while the open timer is still pending so clearAllHoverTimers runs with work.
    layout.sidebarMode = 'expanded';
    pinViewportWidth(1280);
    await flushPromises();
    await nextTick();
    expect(ss.openFlyoutKey).toBeNull();
    mounted.unmount();
  });

  test('collapsed flyout switches apps and clears timers on expand', async () => {
    const menuPlugin = createMenuPlugin();
    menuPlugin.manager.addMenu({
      id: 'base',
      title: 'Master Data',
      children: [{ id: 'loose', title: 'Company', path: '/base/companies' }],
    } as any);
    menuPlugin.manager.addMenu({
      id: 'auth',
      title: 'Auth',
      children: [{ id: 'users', title: 'Users', path: '/auth/users' }],
    } as any);
    const pinia = createPinia();
    setActivePinia(pinia);
    pinViewportWidth(900);
    const layout = useLayoutStore();
    layout.sidebarMode = 'collapsed';
    const { router } = createFeStubRouter({
      route: { path: '/', fullPath: '/', meta: {} },
    });
    const i18n = createI18n({ legacy: false, locale: 'en', messages: { en: {} } });
    const mounted = mountApp(ChoyAppRail as any, {
      plugins: [menuPlugin, pinia, router, i18n],
    });
    await flushPromises();
    await nextTick();
    const ss = mounted.setupState();
    const base = { id: 'base', title: 'Master Data' };
    const auth = { id: 'auth', title: 'Auth' };
    ss.onFlyoutHoverEnter(base);
    await new Promise((r) => setTimeout(r, 120));
    expect(ss.openFlyoutKey).toBe('base');
    ss.onFlyoutHoverEnter(auth);
    await new Promise((r) => setTimeout(r, 120));
    expect(ss.openFlyoutKey).toBe('auth');
    expect(ss.hoverIntent.base || 0).toBe(0);

    ss.onFlyoutHoverLeave(auth);
    // Closing path with an active close timer should keep the flyout until grace ends.
    ss.setFlyoutOpen(auth, false);
    expect(ss.openFlyoutKey).toBe('auth');
    await new Promise((r) => setTimeout(r, 220));
    expect(ss.openFlyoutKey).toBeNull();

    ss.onFlyoutHoverEnter(base);
    await new Promise((r) => setTimeout(r, 120));
    expect(ss.openFlyoutKey).toBe('base');
    layout.sidebarMode = 'expanded';
    pinViewportWidth(1280);
    await flushPromises();
    await nextTick();
    await new Promise((r) => setTimeout(r, 20));
    expect(ss.openFlyoutKey).toBeNull();
    mounted.unmount();
  });
});
