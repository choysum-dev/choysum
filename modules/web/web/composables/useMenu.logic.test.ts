// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, h, nextTick } from 'vue';
import { createPinia, setActivePinia } from 'pinia';
import * as VueRouter from 'vue-router';
import { createMenuPlugin } from '@/core/web/menu';
import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import { useMenuStore } from '../stores/menuStore';
import { useMenu } from './useMenu';

const createFeStubRouter = (VueRouter as any).createFeStubRouter;

describe('useMenu logic', () => {
  test('covers navigateTo branches, expand state, and store proxies', async () => {
    const menuPlugin = createMenuPlugin();
    menuPlugin.manager.addMenu({
      id: 'app',
      title: 'App',
      children: [
        { id: 'leaf', title: 'Leaf', path: '/leaf' },
        {
          id: 'group',
          title: 'Group',
          children: [{ id: 'child', title: 'Child', path: '/child' }],
        },
        {
          id: 'dead',
          title: 'Dead',
          children: [{ id: 'dead-child', title: 'Dead Child' }],
        },
        {
          id: 'ext-blank',
          title: 'Ext Blank',
          path: 'https://example.com/a',
          externalLink: true,
          openMode: 'window',
        },
        {
          id: 'ext-parent',
          title: 'Ext Parent',
          path: 'https://example.com/b',
          externalLink: true,
          openMode: 'parent',
        },
        {
          id: 'ext-top',
          title: 'Ext Top',
          path: 'https://example.com/c',
          externalLink: true,
          openMode: 'top',
        },
        {
          id: 'ext-self',
          title: 'Ext Self',
          path: 'https://example.com/d',
          externalLink: true,
        },
      ],
    } as any);

    const pinia = createPinia();
    setActivePinia(pinia);
    const { router } = createFeStubRouter({
      route: { path: '/leaf', fullPath: '/leaf', meta: {} },
    });

    const prevOpen = window.open;
    const opened: string[] = [];
    window.open = ((_url?: string | URL, target?: string) => {
      opened.push(String(target || ''));
      return null;
    }) as typeof window.open;

    let api!: ReturnType<typeof useMenu>;
    const Host = defineComponent({
      setup() {
        api = useMenu();
        return () => h('div', { 'data-testid': 'use-menu-host' });
      },
    });
    const mounted = mountApp(Host as any, {
      plugins: [menuPlugin, pinia, router],
    });
    await flushPromises();
    await nextTick();

    expect(await api.navigateTo('missing')).toBe(false);
    expect(await api.navigateTo('dead')).toBe(false);
    expect(await api.navigateTo('leaf')).toBe(true);
    expect(await api.navigateTo('ext-blank')).toBe(true);
    expect(await api.navigateTo('ext-parent')).toBe(true);
    expect(await api.navigateTo('ext-top')).toBe(true);
    expect(await api.navigateTo('ext-self')).toBe(true);
    expect(opened).toEqual(['_blank', '_parent', '_top', '_self']);

    const store = useMenuStore();
    // Clear ancestry so manual expand/collapse is not forced open by activeMenu.
    const leaf = store.getMenu('leaf');
    expect(leaf).toBeTruthy();
    store.setActiveMenu(leaf!);

    expect(api.isExpanded('group')).toBe(false);
    api.openSubMenu('group');
    api.openSubMenu('group');
    expect(api.isExpanded('group')).toBe(true);
    api.closeSubMenu('group');
    api.closeSubMenu('group');
    expect(api.isExpanded('group')).toBe(false);
    api.openSubMenu('');
    api.closeSubMenu('');

    const child = store.getMenu('child');
    expect(child).toBeTruthy();
    store.setActiveMenu(child!);
    expect(api.isExpanded('group')).toBe(true);
    expect(api.isExpanded('missing-group')).toBe(false);

    const push = router.push.bind(router);
    router.push = (async () => {
      throw new Error('push failed');
    }) as typeof router.push;
    expect(await api.navigateTo('child')).toBe(false);
    router.push = push;

    expect(await api.navigateTo(leaf!)).toBe(true);
    expect(api.hasMenu('leaf')).toBe(true);
    expect(api.getMenu('leaf')?.id).toBe('leaf');
    expect(api.getMenuByPath('/leaf')?.id).toBe('leaf');
    expect(api.getMenuChildren('group').length).toBe(1);
    expect(api.getMenuParent('child')?.id).toBe('group');
    expect(api.getMenus().length).toBeGreaterThan(0);
    api.setActiveMenu(leaf!);

    window.open = prevOpen;
    mounted.unmount();
  });

  test('record detail path fallback keeps a stable activeMenuId', async () => {
    const menuPlugin = createMenuPlugin();
    menuPlugin.manager.addMenu({
      id: 'auth',
      title: 'Access Control',
      children: [{ id: 'users', title: 'User List', path: '/auth/users' }],
    } as any);
    const pinia = createPinia();
    setActivePinia(pinia);
    const { router } = createFeStubRouter({
      route: {
        name: 'UserDetail',
        path: '/auth/users/u1',
        fullPath: '/auth/users/u1',
        params: { id: 'u1' },
        meta: {},
      },
    });
    const Host = defineComponent({
      setup() {
        const store = useMenuStore();
        return () =>
          h('div', {
            'data-active-menu': store.activeMenu?.id || '',
            'data-active-app': store.activeApp?.id || '',
          });
      },
    });
    const mounted = mountApp(Host as any, {
      plugins: [menuPlugin, pinia, router],
    });
    await flushPromises();
    await nextTick();
    const store = useMenuStore();
    expect(store.activeMenu?.id).toBe('users');
    expect(store.activeMenuId).toBe('users');
    expect(store.activeApp?.id).toBe('auth');
    mounted.unmount();
  });

  test('binds the route after the store was created without injection', async () => {
    const menuPlugin = createMenuPlugin();
    menuPlugin.manager.addMenu({
      id: 'auth',
      title: 'Access Control',
      children: [{ id: 'users', title: 'User List', path: '/auth/users' }],
    } as any);
    const pinia = createPinia();
    setActivePinia(pinia);
    const early = useMenuStore();
    expect(early.activeApp).toBeNull();
    const { router } = createFeStubRouter({
      route: {
        name: 'UserList',
        path: '/auth/users',
        fullPath: '/auth/users',
        params: {},
        meta: {},
      },
    });
    const Host = defineComponent({
      setup() {
        const api = useMenu();
        return () =>
          h('div', {
            'data-app': api.activeApp.value?.id || '',
            'data-menu': api.activeMenu.value?.id || '',
          });
      },
    });
    const mounted = mountApp(Host as any, { plugins: [menuPlugin, pinia, router] });
    await flushPromises();
    await nextTick();
    expect(mounted.q('[data-app]')?.getAttribute('data-app')).toBe('auth');
    expect(mounted.q('[data-menu]')?.getAttribute('data-menu')).toBe('users');
    const store = useMenuStore();
    store.bindRouteFromInjection();
    store.bindRouteFromInjection();
    mounted.unmount();
  });

  test('keeps a manual selection while the menu manager is missing', async () => {
    const pinia = createPinia();
    setActivePinia(pinia);
    const store = useMenuStore();
    store.setActiveMenu({ id: 'keep', title: 'Keep', path: '/keep' } as any);
    expect(store.activeMenu).toBeNull();
    expect(store.activeMenuId).toBe('keep');
    store.bindRouteFromInjection();
    store.setActiveMenu(null);
    expect(store.activeMenuId).toBeNull();
  });

  test('path fallback matches /web-prefixed routes after late bind', async () => {
    const menuPlugin = createMenuPlugin();
    menuPlugin.manager.addMenu({
      id: 'auth',
      title: 'Access Control',
      children: [{ id: 'users', title: 'User List', path: '/auth/users' }],
    } as any);
    const pinia = createPinia();
    setActivePinia(pinia);
    const early = useMenuStore();
    expect(early.activeMenu).toBeNull();
    const { router } = createFeStubRouter({
      route: {
        name: 'UserList',
        path: '/web/auth/users',
        fullPath: '/web/auth/users',
        params: {},
        meta: {},
      },
    });
    const Host = defineComponent({
      setup() {
        const store = useMenuStore();
        store.bindRouteFromInjection();
        return () => h('div', { 'data-menu': store.activeMenu?.id || '' });
      },
    });
    const mounted = mountApp(Host as any, { plugins: [menuPlugin, pinia, router] });
    await flushPromises();
    await nextTick();
    expect(mounted.q('[data-menu]')?.getAttribute('data-menu')).toBe('users');
    mounted.unmount();
  });

  test('activeApp is null when __parent hops exceed the cycle cap', async () => {
    const menuPlugin = createMenuPlugin();
    menuPlugin.manager.addMenu({ id: 'a', title: 'A', path: '/a' } as any);
    menuPlugin.manager.addMenu({ id: 'b', title: 'B', path: '/b' } as any);
    const a = menuPlugin.manager.getMenu('a') as any;
    const b = menuPlugin.manager.getMenu('b') as any;
    a.__parent = b;
    b.__parent = a;
    const pinia = createPinia();
    setActivePinia(pinia);
    const { router } = createFeStubRouter({
      route: { name: 'Loop', path: '/a', fullPath: '/a', params: {}, meta: {} },
    });
    const Host = defineComponent({
      setup() {
        const store = useMenuStore();
        store.setActiveMenu(a);
        return () =>
          h('div', {
            'data-app': store.activeApp?.id || '',
            'data-menu': store.activeMenu?.id || '',
          });
      },
    });
    const mounted = mountApp(Host as any, { plugins: [menuPlugin, pinia, router] });
    await flushPromises();
    expect(mounted.q('[data-app]')?.getAttribute('data-app')).toBe('');
    expect(mounted.q('[data-menu]')?.getAttribute('data-menu')).toBe('a');
    const store = useMenuStore();
    store.setActiveMenu(store.activeMenu);
    await flushPromises();
    mounted.unmount();
  });

  test('isExpanded stops walking a cyclic __parent chain', async () => {
    const menuPlugin = createMenuPlugin();
    const pinia = createPinia();
    setActivePinia(pinia);
    const { router } = createFeStubRouter({
      route: { name: 'Loop', path: '/loop', fullPath: '/loop', params: {}, meta: {} },
    });
    const a = { id: 'a', title: 'A', path: '/a' } as any;
    const b = { id: 'b', title: 'B', path: '/b' } as any;
    a.__parent = b;
    b.__parent = a;
    const Host = defineComponent({
      setup() {
        const api = useMenu();
        api.setActiveMenu(a);
        return () => h('div', { 'data-expanded': api.isExpanded('never') ? '1' : '0' });
      },
    });
    const mounted = mountApp(Host as any, { plugins: [menuPlugin, pinia, router] });
    await flushPromises();
    expect(mounted.q('[data-expanded]')?.getAttribute('data-expanded')).toBe('0');
    mounted.unmount();
  });
});
