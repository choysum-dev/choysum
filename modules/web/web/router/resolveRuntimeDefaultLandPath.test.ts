// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createPinia, setActivePinia } from 'pinia';
import { createMenuPlugin } from '@/core/web/menu';
import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import { defineComponent, h } from 'vue';
import { MODULE_BOARD_PATH } from './resolveDefaultLandPath';
import { resolveRuntimeDefaultLandPath } from './resolveRuntimeDefaultLandPath';

describe('resolveRuntimeDefaultLandPath', () => {
  test('falls back to Module Board without menu injection', () => {
    setActivePinia(createPinia());
    expect(resolveRuntimeDefaultLandPath()).toBe(MODULE_BOARD_PATH);
  });

  test('picks first menu leaf and skips route-denied paths via canNavigate', async () => {
    const createFeStubRouter = (await import('vue-router') as any).createFeStubRouter;
    const menuPlugin = createMenuPlugin();
    menuPlugin.manager.addMenu({
      id: 'denied.menu',
      title: 'Denied',
      path: '/denied',
      order: 1,
    } as any);
    menuPlugin.manager.addMenu({
      id: 'ok.menu',
      title: 'Ok',
      path: '/ok',
      order: 2,
    } as any);

    const pinia = createPinia();
    setActivePinia(pinia);
    const { router } = createFeStubRouter({
      route: { path: '/', fullPath: '/', meta: {} },
    });
    // FE stub resolve ignores a routes table; map path → resourceId for canRoute.
    const resourceByPath: Record<string, string> = {
      '/denied': 'route.denied',
      '/ok': 'route.ok',
    };
    router.resolve = (to: unknown) => {
      const path = typeof to === 'string' ? to : String((to as any)?.path || '/');
      return {
        path,
        fullPath: path,
        href: path,
        name: undefined,
        query: {},
        params: {},
        matched: [{ path }],
        meta: { resourceId: resourceByPath[path] },
      };
    };

    const { useAuthStore } = await import('@/auth/web/stores/auth');
    const auth = useAuthStore(pinia);
    (auth as any).isAuthenticated = true;
    (auth as any).permissionState = {
      permStateVersion: 1,
      byCompany: {
        '*': { ui: { routes: ['route.ok'], menus: ['*'], actions: [] } },
      },
    };
    (auth as any).identity = { metadata: { activeCompanyId: 'c1', enabledCompanyIds: ['c1'] } };

    const Host = defineComponent({
      setup() {
        const path = resolveRuntimeDefaultLandPath();
        return () => h('div', { 'data-path': path });
      },
    });

    const mounted = mountApp(Host as any, {
      plugins: [menuPlugin, pinia, router],
    });
    await flushPromises();
    expect(mounted.q('[data-path]')?.getAttribute('data-path')).toBe('/ok');
    mounted.unmount();
  });

  test('skips leaves when router.resolve throws and continues to next leaf', async () => {
    const createFeStubRouter = (await import('vue-router') as any).createFeStubRouter;
    const menuPlugin = createMenuPlugin();
    menuPlugin.manager.addMenu({
      id: 'boom.menu',
      title: 'Boom',
      path: '/boom',
      order: 1,
    } as any);
    menuPlugin.manager.addMenu({
      id: 'ok.menu',
      title: 'Ok',
      path: '/ok',
      order: 2,
    } as any);

    const pinia = createPinia();
    setActivePinia(pinia);
    const { router } = createFeStubRouter({
      route: { path: '/', fullPath: '/', meta: {} },
    });
    router.resolve = (to: unknown) => {
      const path = typeof to === 'string' ? to : String((to as any)?.path || '/');
      if (path === '/boom') throw new Error('resolve failed');
      return {
        path,
        fullPath: path,
        href: path,
        name: undefined,
        query: {},
        params: {},
        matched: [{ path }],
        meta: { resourceId: 'route.ok' },
      };
    };

    const { useAuthStore } = await import('@/auth/web/stores/auth');
    const auth = useAuthStore(pinia);
    (auth as any).isAuthenticated = true;
    (auth as any).permissionState = {
      permStateVersion: 1,
      byCompany: { '*': { ui: { routes: ['*'], menus: ['*'], actions: [] } } },
    };
    (auth as any).identity = { metadata: { activeCompanyId: 'c1', enabledCompanyIds: ['c1'] } };

    const Host = defineComponent({
      setup() {
        const path = resolveRuntimeDefaultLandPath();
        return () => h('div', { 'data-path': path });
      },
    });

    const mounted = mountApp(Host as any, {
      plugins: [menuPlugin, pinia, router],
    });
    await flushPromises();
    expect(mounted.q('[data-path]')?.getAttribute('data-path')).toBe('/ok');
    mounted.unmount();
  });

  test('falls back to Module Board when menu store cannot init without router', async () => {
    const menuPlugin = createMenuPlugin();
    menuPlugin.manager.addMenu({
      id: 'only.menu',
      title: 'Only',
      path: '/only',
      order: 1,
    } as any);
    const pinia = createPinia();
    setActivePinia(pinia);

    const Host = defineComponent({
      setup() {
        const path = resolveRuntimeDefaultLandPath();
        return () => h('div', { 'data-path': path });
      },
    });

    const mounted = mountApp(Host as any, {
      plugins: [menuPlugin, pinia],
    });
    await flushPromises();
    // useMenuStore setup calls useRoute(); without a router plugin it throws and
    // resolveRuntimeDefaultLandPath fails closed to Module Board.
    expect(mounted.q('[data-path]')?.getAttribute('data-path')).toBe(MODULE_BOARD_PATH);
    mounted.unmount();
  });
});
