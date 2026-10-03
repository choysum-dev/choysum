// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, h, nextTick } from 'vue';
import { createPinia, setActivePinia } from 'pinia';
import { createI18n } from 'vue-i18n';
import * as VueRouter from 'vue-router';
import { createMenuPlugin } from '@/core/web/menu';
import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import { useMenuStore } from '../stores/menuStore';
import ChoySidebarNav from '../components/layout/ChoySidebarNav.vue';
import { SidebarProvider } from '../components/vendor/ui/sidebar/index';

const createFeStubRouter = (VueRouter as any).createFeStubRouter;

async function mountSidebar(opts?: { activeChild?: boolean }) {
  const menuPlugin = createMenuPlugin();
  menuPlugin.manager.addMenu({
    id: 'app',
    title: 'App',
    children: [
      {
        id: 'group',
        title: 'Group',
        children: [
          { id: 'child-a', title: 'Child A', path: '/a' },
          { id: 'child-b', title: 'Child B', path: '/b' },
        ],
      },
      { id: 'leaf', title: 'Leaf', path: '/leaf' },
      {
        id: '',
        title: 'NoId Group',
        children: [{ id: 'noid-child', title: 'NoId Child', path: '/noid' }],
      },
    ],
  } as any);

  const pinia = createPinia();
  setActivePinia(pinia);
  const routePath = opts?.activeChild ? '/a' : '/leaf';
  const { router } = createFeStubRouter({
    route: {
      path: routePath,
      fullPath: routePath,
      meta: {},
    },
  });
  const i18n = createI18n({ legacy: false, locale: 'en', messages: { en: {} } });

  const Host = defineComponent({
    name: 'UseMenuSidebarHost',
    setup() {
      const store = useMenuStore();
      const activeId = opts?.activeChild ? 'child-a' : 'leaf';
      const active = store.getMenu(activeId);
      if (active) store.setActiveMenu(active);
      return () =>
        h(SidebarProvider, null, {
          default: () => h(ChoySidebarNav, { useDefaultIcon: true }),
        });
    },
  });

  const mounted = mountApp(Host as any, {
    plugins: [menuPlugin, pinia, router, i18n],
  });
  await flushPromises();
  await nextTick();
  return mounted;
}

describe('ChoySidebarNav expansion', () => {
  test('manually expands and collapses inactive groups', async () => {
    const mounted = await mountSidebar();
    const groupBtn = Array.from(
      mounted.el.querySelectorAll('[data-testid=choy-sidebar-nav-group]'),
    ).find((b) => (b.textContent || '').includes('Group')) as HTMLButtonElement | undefined;
    expect(groupBtn).toBeTruthy();
    expect(groupBtn!.getAttribute('aria-expanded')).toBe('false');
    expect(mounted.text()).not.toContain('Child A');

    groupBtn!.click();
    groupBtn!.click();
    await flushPromises();
    await nextTick();
    expect(groupBtn!.getAttribute('aria-expanded')).toBe('true');
    expect(mounted.text()).toContain('Child A');

    groupBtn!.click();
    await flushPromises();
    await nextTick();
    expect(groupBtn!.getAttribute('aria-expanded')).toBe('false');
    expect(mounted.text()).not.toContain('Child A');
    mounted.unmount();
  });

  test('keeps ancestry-expanded groups open for the active menu', async () => {
    const mounted = await mountSidebar({ activeChild: true });
    const groupBtn = Array.from(
      mounted.el.querySelectorAll('[data-testid=choy-sidebar-nav-group]'),
    ).find((b) => (b.textContent || '').includes('Group')) as HTMLButtonElement | undefined;
    expect(groupBtn).toBeTruthy();
    expect(groupBtn!.getAttribute('aria-expanded')).toBe('true');
    expect(mounted.text()).toContain('Child A');

    groupBtn!.click();
    await flushPromises();
    await nextTick();
    expect(groupBtn!.getAttribute('aria-expanded')).toBe('true');
    mounted.unmount();
  });

  test('open/close with empty menu id is a no-op', async () => {
    const mounted = await mountSidebar();
    const noIdBtn = Array.from(
      mounted.el.querySelectorAll('[data-testid=choy-sidebar-nav-group]'),
    ).find((b) => (b.textContent || '').includes('NoId Group')) as HTMLButtonElement | undefined;
    expect(noIdBtn).toBeTruthy();
    expect(noIdBtn!.getAttribute('aria-expanded')).toBe('false');
    noIdBtn!.click();
    await flushPromises();
    await nextTick();
    expect(noIdBtn!.getAttribute('aria-expanded')).toBe('false');
    mounted.unmount();
  });

  test('ancestry-expanded empty-id group still no-ops on toggle', async () => {
    const menuPlugin = createMenuPlugin();
    menuPlugin.manager.addMenu({
      id: 'app',
      title: 'App',
      children: [
        {
          id: '',
          title: 'NoId Group',
          children: [{ id: 'noid-child', title: 'NoId Child', path: '/noid' }],
        },
      ],
    } as any);
    const pinia = createPinia();
    setActivePinia(pinia);
    const { router } = createFeStubRouter({
      route: { path: '/noid', fullPath: '/noid', meta: {} },
    });
    const i18n = createI18n({ legacy: false, locale: 'en', messages: { en: {} } });
    const Host = defineComponent({
      setup() {
        const store = useMenuStore();
        const active = store.getMenu('noid-child');
        if (active) store.setActiveMenu(active);
        return () =>
          h(SidebarProvider, null, {
            default: () => h(ChoySidebarNav, { useDefaultIcon: true }),
          });
      },
    });
    const mounted = mountApp(Host as any, {
      plugins: [menuPlugin, pinia, router, i18n],
    });
    await flushPromises();
    await nextTick();
    const noIdBtn = Array.from(
      mounted.el.querySelectorAll('[data-testid=choy-sidebar-nav-group]'),
    ).find((b) => (b.textContent || '').includes('NoId Group')) as HTMLButtonElement | undefined;
    expect(noIdBtn).toBeTruthy();
    expect(noIdBtn!.getAttribute('aria-expanded')).toBe('true');
    noIdBtn!.click();
    await flushPromises();
    await nextTick();
    expect(noIdBtn!.getAttribute('aria-expanded')).toBe('true');
    mounted.unmount();
  });

  test('navigates when a leaf item is clicked', async () => {
    const mounted = await mountSidebar();
    const leaf = Array.from(mounted.el.querySelectorAll('[data-testid=choy-sidebar-nav-leaf]')).find(
      (b) => (b.textContent || '').includes('Leaf'),
    ) as HTMLButtonElement | undefined;
    expect(leaf).toBeTruthy();
    leaf!.click();
    await flushPromises();
    mounted.unmount();
  });
});
