// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi, beforeEach } from 'vitest';
import { applyPermissionToMenus } from '@/auth/web/menu/applyPermissionToMenus';
import { canRoute, hasAction, type PermissionState } from '@/auth/web/permission';
import { authMenus } from '../menu/menus';
import { menus as webMenus } from '@/web/web/menu/menus';

let mockAuthStore: any;

function clone<T>(value: T): T {
  return JSON.parse(JSON.stringify(value));
}

vi.mock('../stores/auth', () => {
  return {
    useAuthStore: () => mockAuthStore,
  };
});

describe('permission flow integration', () => {
  beforeEach(() => {
    mockAuthStore = {
      isAuthenticated: true,
      loadPermissionState: vi.fn(async () => undefined),
      permissionState: null,
      identity: {
        metadata: {
          activeCompanyId: 'c1',
          enabledCompanyIds: ['c1'],
        },
      },
    };
  });

  it('keeps route guard, menu filtering and action checks consistent', async () => {
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

    mockAuthStore.permissionState = state;

    const { permissionGuard } = await import('./guard');

    const allowed = await permissionGuard(
      {
        path: '/auth/allowed',
        fullPath: '/auth/allowed',
        meta: { requiresAuth: true, resourceId: 'auth.route.allowed' },
      } as any,
      {} as any
    );

    expect(allowed).toBe(true);

    const denied = await permissionGuard(
      {
        path: '/auth/denied',
        fullPath: '/auth/denied',
        meta: { requiresAuth: true, resourceId: 'auth.route.denied' },
      } as any,
      {} as any
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

  it('respects enabled scope when active company differs', async () => {
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

    mockAuthStore.identity.metadata = {
      activeCompanyId: 'c1',
      enabledCompanyIds: ['c1', 'c2'],
    };
    mockAuthStore.permissionState = state;

    const { permissionGuard } = await import('./guard');

    const result = await permissionGuard(
      {
        path: '/auth/c2',
        fullPath: '/auth/c2',
        meta: { requiresAuth: true, resourceId: 'auth.route.c2_only' },
      } as any,
      {} as any
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

  it('smoke: permission whitelist drives route guard and action visibility together', async () => {
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

    mockAuthStore.permissionState = state;

    const { permissionGuard } = await import('./guard');

    const allowed = await permissionGuard(
      {
        path: '/auth/tokens',
        fullPath: '/auth/tokens',
        meta: { requiresAuth: true, resourceId: 'auth.route.token_list' },
      } as any,
      {} as any
    );
    expect(allowed).toBe(true);

    const denied = await permissionGuard(
      {
        path: '/auth/tokens/kanban',
        fullPath: '/auth/tokens/kanban',
        meta: { requiresAuth: true, resourceId: 'auth.route.token_kanban' },
      } as any,
      {} as any
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

  it('keeps home visible in menu projection for ordinary users', async () => {
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

    mockAuthStore.permissionState = state;

    const { permissionGuard } = await import('./guard');

    const result = await permissionGuard(
      {
        path: '/home',
        fullPath: '/home',
        meta: { requiresAuth: true, resourceId: 'web.route.home' },
      } as any,
      {} as any
    );

    expect(result).toBe(true);
    expect(canRoute('web.route.home', state, { activeCompanyId: 'c1', enabledCompanyIds: ['c1'] })).toBe(true);

    const menus = [...clone(webMenus as any), ...clone(authMenus as any)] as any[];
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
});
