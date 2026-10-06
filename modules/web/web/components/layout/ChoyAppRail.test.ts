// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, h, nextTick } from 'vue';
import { createPinia, setActivePinia } from 'pinia';
import { createI18n } from 'vue-i18n';
import * as VueRouter from 'vue-router';
import { createMenuPlugin } from '@/core/web/menu';
import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import { useMenuStore } from '../../stores/menuStore';
import ChoyAppRail from './ChoyAppRail.vue';
import { SidebarProvider } from '../vendor/ui/sidebar/index';

const createFeStubRouter = (VueRouter as any).createFeStubRouter;

describe('ChoyAppRail', () => {
  test('lists every visible app root and lands the first leaf on click', async () => {
    const menuPlugin = createMenuPlugin();
    menuPlugin.manager.addMenu({
      id: 'meta',
      title: 'Module Management',
      children: [{ id: 'meta-board', title: 'Module Board', path: '/meta/modules' }],
    } as any);
    menuPlugin.manager.addMenu({
      id: 'base',
      title: 'Master Data',
      children: [{ id: 'base-company', title: 'Company', path: '/base/companies' }],
    } as any);
    const pinia = createPinia();
    setActivePinia(pinia);
    const { router } = createFeStubRouter({
      route: { path: '/meta/modules', fullPath: '/meta/modules', meta: {} },
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
        const store = useMenuStore();
        const active = store.getMenu('meta-board');
        if (active) store.setActiveMenu(active);
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

  test('renders a first-letter fallback when the app root has no icon', async () => {
    const menuPlugin = createMenuPlugin();
    menuPlugin.manager.addMenu({
      id: 'custom',
      title: '公司管理',
      children: [{ id: 'custom-leaf', title: 'Leaf', path: '/custom' }],
    } as any);
    const pinia = createPinia();
    setActivePinia(pinia);
    const { router } = createFeStubRouter({
      route: { path: '/custom', fullPath: '/custom', meta: {} },
    });
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
    expect(mounted.q('[data-testid=choy-shell-app-rail-fallback]')?.textContent?.trim()).toBe('公');
    expect(mounted.q('[data-testid=choy-shell-app-rail-item]')?.querySelector('svg')).toBeNull();
    expect(mounted.q('[data-testid=choy-shell-app-rail-tooltip]')).toBeNull();
    mounted.unmount();
  });

  test('opens a tooltip portal and skips apps without a key', async () => {
    const menuPlugin = createMenuPlugin();
    menuPlugin.manager.addMenu({
      id: 'custom',
      title: '公司管理',
      children: [{ id: 'custom-leaf', title: 'Leaf', path: '/custom' }],
    } as any);
    const pinia = createPinia();
    setActivePinia(pinia);
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

  test('renders without menu, i18n, or sidebar context', async () => {
    setActivePinia(undefined as any);
    const mounted = mountApp(ChoyAppRail as any);
    await flushPromises();
    expect(mounted.q('[data-testid=choy-shell-app-rail]')).not.toBeNull();
    expect(mounted.qa('[data-testid=choy-shell-app-rail-item]').length).toBe(0);
    mounted.unmount();
  });
});
