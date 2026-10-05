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
          default: () => h(ChoySidebarNav),
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
            default: () => h(ChoySidebarNav),
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
    ) as HTMLAnchorElement | undefined;
    expect(leaf).toBeTruthy();
    expect(leaf!.tagName).toBe('A');
    expect(leaf!.getAttribute('href')).toBe('/leaf');
    leaf!.click();
    await flushPromises();
    mounted.unmount();
  });

  test('labels app roots and marks the current leaf', async () => {
    const mounted = await mountSidebar();
    const labels = Array.from(
      mounted.el.querySelectorAll('[data-testid=choy-sidebar-nav-group-label]'),
    ).map((el) => (el.textContent || '').trim());
    expect(labels).toEqual(['App']);

    const leaf = Array.from(mounted.el.querySelectorAll('[data-testid=choy-sidebar-nav-leaf]')).find(
      (b) => (b.textContent || '').includes('Leaf'),
    ) as HTMLButtonElement | undefined;
    expect(leaf?.getAttribute('data-active')).toBe('true');
    expect(leaf?.getAttribute('aria-current')).toBe('page');

    const other = Array.from(mounted.el.querySelectorAll('[data-testid=choy-sidebar-nav-leaf]')).find(
      (b) => (b.textContent || '').includes('Child A'),
    );
    expect(other).toBeUndefined();
    mounted.unmount();
  });

  test('renders every app root, not only the active app', async () => {
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
    const i18n = createI18n({ legacy: false, locale: 'en', messages: { en: {} } });
    const Host = defineComponent({
      setup() {
        const store = useMenuStore();
        const active = store.getMenu('meta-board');
        if (active) store.setActiveMenu(active);
        return () =>
          h(SidebarProvider, null, {
            default: () => h(ChoySidebarNav),
          });
      },
    });
    const mounted = mountApp(Host as any, {
      plugins: [menuPlugin, pinia, router, i18n],
    });
    await flushPromises();
    await nextTick();
    const labels = Array.from(
      mounted.el.querySelectorAll('[data-testid=choy-sidebar-nav-group-label]'),
    ).map((el) => (el.textContent || '').trim());
    expect(labels).toEqual(['Module Management', 'Master Data']);
    expect(mounted.text()).toContain('Company');
    const activeLeaf = Array.from(
      mounted.el.querySelectorAll('[data-testid=choy-sidebar-nav-leaf]'),
    ).find((b) => (b.textContent || '').includes('Module Board')) as HTMLButtonElement | undefined;
    expect(activeLeaf?.getAttribute('data-active')).toBe('true');
    mounted.unmount();
  });

  test('renders text-only leaves when menus declare no icon', async () => {
    const mounted = await mountSidebar();
    const leaf = Array.from(mounted.el.querySelectorAll('[data-testid=choy-sidebar-nav-leaf]')).find(
      (b) => (b.textContent || '').includes('Leaf'),
    ) as HTMLButtonElement | undefined;
    expect(leaf?.querySelector('svg')).toBeNull();
    expect(leaf?.querySelector('[aria-hidden=true]')).toBeNull();
    mounted.unmount();
  });

  test('renders empty state when no menus exist', async () => {
    const menuPlugin = createMenuPlugin();
    const pinia = createPinia();
    setActivePinia(pinia);
    const { router } = createFeStubRouter({
      route: { path: '/', fullPath: '/', meta: {} },
    });
    const i18n = createI18n({ legacy: false, locale: 'en', messages: { en: {} } });
    const Host = defineComponent({
      setup() {
        return () =>
          h(SidebarProvider, null, {
            default: () => h(ChoySidebarNav),
          });
      },
    });
    const mounted = mountApp(Host as any, {
      plugins: [menuPlugin, pinia, router, i18n],
    });
    await flushPromises();
    await nextTick();
    expect(mounted.q('[data-testid=choy-sidebar-nav-empty]')).not.toBeNull();
    mounted.unmount();
  });

  test('hides hidden items and recurses past three levels', async () => {
    const menuPlugin = createMenuPlugin();
    menuPlugin.manager.addMenu({
      id: 'app',
      title: 'App',
      children: [
        {
          id: 'l1',
          title: 'L1',
          children: [
            {
              id: 'l2',
              title: 'L2',
              children: [
                {
                  id: 'l3',
                  title: 'L3',
                  children: [{ id: 'l4', title: 'L4 Leaf', path: '/l4' }],
                },
              ],
            },
          ],
        },
        { id: 'hidden-leaf', title: 'Hidden', path: '/h', hidden: true },
      ],
    } as any);
    const pinia = createPinia();
    setActivePinia(pinia);
    const { router } = createFeStubRouter({
      route: { path: '/l4', fullPath: '/l4', meta: {} },
    });
    const i18n = createI18n({ legacy: false, locale: 'en', messages: { en: {} } });
    const Host = defineComponent({
      setup() {
        const store = useMenuStore();
        const active = store.getMenu('l4');
        if (active) store.setActiveMenu(active);
        return () =>
          h(SidebarProvider, null, {
            default: () => h(ChoySidebarNav),
          });
      },
    });
    const mounted = mountApp(Host as any, {
      plugins: [menuPlugin, pinia, router, i18n],
    });
    await flushPromises();
    await nextTick();
    expect(mounted.text()).not.toContain('Hidden');
    expect(mounted.text()).toContain('L4 Leaf');
    const l4 = Array.from(mounted.el.querySelectorAll('[data-testid=choy-sidebar-nav-leaf]')).find(
      (b) => (b.textContent || '').includes('L4 Leaf'),
    ) as HTMLButtonElement | undefined;
    expect(l4).toBeTruthy();
    l4!.click();
    await flushPromises();
    mounted.unmount();
  });

  test('tolerates missing menu store / router context', async () => {
    setActivePinia(undefined as any);
    const mounted = mountApp(ChoySidebarNav as any);
    await flushPromises();
    expect(mounted.q('[data-testid=choy-sidebar-nav-empty]')).not.toBeNull();
    mounted.unmount();
  });
});
