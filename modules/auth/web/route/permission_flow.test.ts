// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { applyPermissionToMenus } from '@/auth/web/menu/applyPermissionToMenus';
import { canRoute, hasAction, type PermissionState } from '@/auth/web/permission';
import { permissionGuard, type AuthGuardDeps } from './guard';

function clone<T>(value: T): T {
  return JSON.parse(JSON.stringify(value));
}

function fnRecorder(): { calls: unknown[][] } & ((...args: unknown[]) => Promise<undefined>) {
  const rec = Object.assign(
    async (...args: unknown[]) => {
      rec.calls.push(args);
      return undefined;
    },
    { calls: [] as unknown[][] }
  );
  return rec;
}

function depsFor(store: any): AuthGuardDeps {
  return { getAuthStore: () => store };
}

function makeStore(permissionState: PermissionState | null, identityMeta?: Record<string, unknown>) {
  return {
    isAuthenticated: true,
    loadPermissionState: fnRecorder(),
    permissionState,
    identity: {
      metadata: {
        activeCompanyId: 'c1',
        enabledCompanyIds: ['c1'],
        ...(identityMeta || {}),
      },
    },
  };
}

test('permission flow: keeps route guard, menu filtering and action checks consistent', async () => {
  const state: PermissionState = {
    permStateVersion: 1,
    byCompany: {
      '*': {
        ui: {
          routes: ['auth.route.allowed'],
          menus: ['auth.menu.root', 'auth.menu.allowed'],
          actions: ['auth.action.allowed'],
        },
      },
    },
  };

  const store = makeStore(state);
  const d = depsFor(store);

  const allowed = await permissionGuard(
    {
      path: '/auth/allowed',
      fullPath: '/auth/allowed',
      meta: { requiresAuth: true, resourceId: 'auth.route.allowed' },
    } as any,
    {} as any,
    d
  );
  expect(allowed).toBe(true);

  const denied = await permissionGuard(
    {
      path: '/auth/denied',
      fullPath: '/auth/denied',
      meta: { requiresAuth: true, resourceId: 'auth.route.denied' },
    } as any,
    {} as any,
    d
  );
  expect(denied).toEqual({
    path: '/error/403',
    query: {
      reason: 'permission',
      message: 'PermissionDenied',
      from: '/auth/denied',
    },
    replace: true,
  });

  const menus: any[] = [
    {
      id: 'auth.menu.root',
      title: 'Root',
      children: [
        { id: 'auth.menu.allowed', title: 'Allowed', path: '/auth/allowed' },
        { id: 'auth.menu.denied', title: 'Denied', path: '/auth/denied' },
      ],
    },
  ];

  applyPermissionToMenus(menus as any, state, {
    activeCompanyId: 'c1',
    enabledCompanyIds: ['c1'],
  });

  expect(Boolean(menus[0].hidden)).toBe(false);
  expect(Boolean(menus[0].children[0].hidden)).toBe(false);
  expect(Boolean(menus[0].children[1].hidden)).toBe(true);

  expect(hasAction('auth.action.allowed', state, { activeCompanyId: 'c1', enabledCompanyIds: ['c1'] })).toBe(true);
  expect(hasAction('auth.action.denied', state, { activeCompanyId: 'c1', enabledCompanyIds: ['c1'] })).toBe(false);
});

test('permission flow: respects enabled scope when active company differs', async () => {
  const state: PermissionState = {
    permStateVersion: 1,
    byCompany: {
      '*': { ui: { routes: [], menus: ['auth.menu.root'], actions: [] } },
      c1: { ui: { routes: [], menus: [], actions: [] } },
      c2: {
        ui: {
          routes: ['auth.route.c2_only'],
          menus: ['auth.menu.c2_only'],
          actions: ['auth.action.c2_only'],
        },
      },
    },
  };

  const store = makeStore(state, {
    activeCompanyId: 'c1',
    enabledCompanyIds: ['c1', 'c2'],
  });

  const result = await permissionGuard(
    {
      path: '/auth/c2',
      fullPath: '/auth/c2',
      meta: { requiresAuth: true, resourceId: 'auth.route.c2_only' },
    } as any,
    {} as any,
    depsFor(store)
  );
  expect(result).toBe(true);

  const menus: any[] = [
    {
      id: 'auth.menu.root',
      title: 'Root',
      children: [{ id: 'auth.menu.c2_only', title: 'C2', path: '/auth/c2' }],
    },
  ];

  applyPermissionToMenus(menus as any, state, {
    activeCompanyId: 'c1',
    enabledCompanyIds: ['c1', 'c2'],
  });

  expect(Boolean(menus[0].hidden)).toBe(false);
  expect(Boolean(menus[0].children[0].hidden)).toBe(false);

  expect(hasAction('auth.action.c2_only', state, { activeCompanyId: 'c1', enabledCompanyIds: ['c1', 'c2'] })).toBe(true);
  expect(hasAction('auth.action.c2_only', state, { activeCompanyId: 'c1', enabledCompanyIds: ['c1', 'c2'] }, 'active')).toBe(false);
});

test('permission flow: smoke whitelist drives route guard and action visibility together', async () => {
  const state: PermissionState = {
    permStateVersion: 1,
    byCompany: {
      '*': {
        ui: {
          routes: ['auth.route.token_list'],
          menus: ['auth.menu.root', 'auth.menu.token_list'],
          actions: ['auth.action.token_edit'],
        },
      },
    },
  };

  const store = makeStore(state);
  const d = depsFor(store);

  const allowed = await permissionGuard(
    {
      path: '/auth/tokens',
      fullPath: '/auth/tokens',
      meta: { requiresAuth: true, resourceId: 'auth.route.token_list' },
    } as any,
    {} as any,
    d
  );
  expect(allowed).toBe(true);

  const denied = await permissionGuard(
    {
      path: '/auth/tokens/kanban',
      fullPath: '/auth/tokens/kanban',
      meta: { requiresAuth: true, resourceId: 'auth.route.token_kanban' },
    } as any,
    {} as any,
    d
  );
  expect(denied).toEqual({
    path: '/error/403',
    query: {
      reason: 'permission',
      message: 'PermissionDenied',
      from: '/auth/tokens/kanban',
    },
    replace: true,
  });

  expect(canRoute('auth.route.token_list', state, { activeCompanyId: 'c1', enabledCompanyIds: ['c1'] })).toBe(true);
  expect(canRoute('auth.route.token_kanban', state, { activeCompanyId: 'c1', enabledCompanyIds: ['c1'] })).toBe(false);
  expect(hasAction('auth.action.token_edit', state, { activeCompanyId: 'c1', enabledCompanyIds: ['c1'] })).toBe(true);
  expect(hasAction('auth.action.token_delete', state, { activeCompanyId: 'c1', enabledCompanyIds: ['c1'] })).toBe(false);
});

test('permission flow: keeps home visible in menu projection for ordinary users', async () => {
  const state: PermissionState = {
    permStateVersion: 1,
    byCompany: {
      '*': {
        ui: {
          routes: ['web.route.home'],
          menus: ['web.menu.home'],
          actions: [],
        },
      },
    },
  };

  const store = makeStore(state);

  const result = await permissionGuard(
    {
      path: '/home',
      fullPath: '/home',
      meta: { requiresAuth: true, resourceId: 'web.route.home' },
    } as any,
    {} as any,
    depsFor(store)
  );

  expect(result).toBe(true);
  expect(canRoute('web.route.home', state, { activeCompanyId: 'c1', enabledCompanyIds: ['c1'] })).toBe(true);

  const menus = clone([
    { id: 'web.menu.home', title: 'Home', path: '/home' },
    { id: 'auth.menu.root', title: 'Access Control', children: [{ id: 'auth.menu.user_list', title: 'Users' }] },
  ]) as any[];
  applyPermissionToMenus(menus as any, state, {
    activeCompanyId: 'c1',
    enabledCompanyIds: ['c1'],
  });

  const homeMenu = menus.find(menu => menu.id === 'web.menu.home');
  const authRoot = menus.find(menu => menu.id === 'auth.menu.root');

  expect(homeMenu).toBeTruthy();
  expect(Boolean(homeMenu.hidden)).toBe(false);
  expect(Boolean(homeMenu.disabled)).toBe(false);

  expect(authRoot).toBeTruthy();
  expect(Boolean(authRoot.hidden)).toBe(true);
});
