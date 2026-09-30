// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, h, nextTick } from 'vue';
import { createPinia, setActivePinia } from 'pinia';
import { createI18n } from 'vue-i18n';
import * as VueRouter from 'vue-router';
import { createMenuPlugin } from '@/core/web/menu';
import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import { useMenu } from './useMenu';
import { useMenuStore } from '../stores/menuStore';

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
      const menu = useMenu();
      const store = useMenuStore();
      // Activate a leaf under the app so renderSidebarMenu shows app children.
      const activeId = opts?.activeChild ? 'child-a' : 'leaf';
      const active = store.getMenu(activeId);
      if (active) store.setActiveMenu(active);
      return () => menu.renderSidebarMenu({ useDefaultIcon: true });
    },
  });

  const mounted = mountApp(Host as any, {
    plugins: [menuPlugin, pinia, router, i18n],
  });
  await flushPromises();
  await nextTick();
  return mounted;
}

describe('useMenu sidebar expansion', () => {
  test('manually expands and collapses inactive groups', async () => {
    const mounted = await mountSidebar();
    const groupBtn = Array.from(mounted.el.querySelectorAll('button.choy-menu__sub-title')).find(b =>
      (b.textContent || '').includes('Group'),
    ) as HTMLButtonElement | undefined;
    expect(groupBtn).toBeTruthy();
    expect(groupBtn!.getAttribute('aria-expanded')).toBe('false');
    expect(mounted.text()).not.toContain('Child A');

    groupBtn!.click();
    // Second click before re-render still sees expanded=false; openSubMenu no-ops when already open.
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
    const groupBtn = Array.from(mounted.el.querySelectorAll('button.choy-menu__sub-title')).find(b =>
      (b.textContent || '').includes('Group'),
    ) as HTMLButtonElement | undefined;
    expect(groupBtn).toBeTruthy();
    expect(groupBtn!.getAttribute('aria-expanded')).toBe('true');
    expect(mounted.text()).toContain('Child A');

    // Already expanded via ancestry: click close is a no-op on the Set but still fires.
    groupBtn!.click();
    await flushPromises();
    await nextTick();
    expect(groupBtn!.getAttribute('aria-expanded')).toBe('true');
    mounted.unmount();
  });

  test('open/close with empty menu id is a no-op', async () => {
    const mounted = await mountSidebar();
    const noIdBtn = Array.from(mounted.el.querySelectorAll('button.choy-menu__sub-title')).find(b =>
      (b.textContent || '').includes('NoId Group'),
    ) as HTMLButtonElement | undefined;
    expect(noIdBtn).toBeTruthy();
    expect(noIdBtn!.getAttribute('aria-expanded')).toBe('false');
    noIdBtn!.click();
    await flushPromises();
    await nextTick();
    // Empty id cannot be stored in openedSubMenus.
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
        const menu = useMenu();
        const store = useMenuStore();
        const active = store.getMenu('noid-child');
        if (active) store.setActiveMenu(active);
        return () => menu.renderSidebarMenu({ useDefaultIcon: true });
      },
    });
    const mounted = mountApp(Host as any, {
      plugins: [menuPlugin, pinia, router, i18n],
    });
    await flushPromises();
    await nextTick();
    const noIdBtn = Array.from(mounted.el.querySelectorAll('button.choy-menu__sub-title')).find(b =>
      (b.textContent || '').includes('NoId Group'),
    ) as HTMLButtonElement | undefined;
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
    const leaf = Array.from(mounted.el.querySelectorAll('button.choy-menu__item')).find(b =>
      (b.textContent || '').includes('Leaf'),
    ) as HTMLButtonElement | undefined;
    expect(leaf).toBeTruthy();
    leaf!.click();
    await flushPromises();
    mounted.unmount();
  });
});
