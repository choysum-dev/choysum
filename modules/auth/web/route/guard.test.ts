// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { PermissionState } from '@/auth/web/permission';
import { authGuard, permissionGuard, type AuthGuardDeps } from './guard';

type CallRecorder = { calls: unknown[][] };

function fnRecorder(): CallRecorder & ((...args: unknown[]) => Promise<undefined>) {
  const rec: CallRecorder & ((...args: unknown[]) => Promise<undefined>) = Object.assign(
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

function routesState(routes: string[]): PermissionState {
  return {
    permStateVersion: 1,
    byCompany: {
      '*': { ui: { routes, menus: [], actions: [] } },
    },
  };
}

test('authGuard redirects unauthenticated users to login', async () => {
  const ensureAuthReady = fnRecorder();
  const mockAuthStore = {
    ensureAuthReady,
    isAuthenticated: false,
  };

  const result = await authGuard(
    {
      path: '/auth/users',
      fullPath: '/auth/users?page=1',
      meta: { requiresAuth: true },
    } as any,
    {} as any,
    depsFor(mockAuthStore)
  );

  expect(ensureAuthReady.calls.length).toBe(1);
  expect(result).toEqual({
    path: '/login',
    query: { redirect: '/auth/users?page=1' },
    replace: true,
  });
});

test('permissionGuard redirects to 403 when resource is not allowed', async () => {
  const loadPermissionState = fnRecorder();
  const mockAuthStore = {
    isAuthenticated: true,
    loadPermissionState,
    permissionState: routesState([]),
    identity: { metadata: { activeCompanyId: 'c1', enabledCompanyIds: ['c1'] } },
  };

  const result = await permissionGuard(
    {
      path: '/auth/users',
      fullPath: '/auth/users',
      meta: { requiresAuth: true, resourceId: 'auth.route.user_list' },
    } as any,
    {} as any,
    depsFor(mockAuthStore)
  );

  expect(loadPermissionState.calls).toEqual([[false]]);
  expect(result).toEqual({
    path: '/error/403',
    query: {
      reason: 'permission',
      message: 'PermissionDenied',
      from: '/auth/users',
    },
    replace: true,
  });
});

test('permissionGuard allows route when no resource id is declared', async () => {
  const loadPermissionState = fnRecorder();
  const mockAuthStore = {
    isAuthenticated: true,
    loadPermissionState,
    permissionState: routesState([]),
    identity: { metadata: { activeCompanyId: 'c1', enabledCompanyIds: ['c1'] } },
  };

  const result = await permissionGuard(
    {
      path: '/public/help',
      fullPath: '/public/help',
      meta: { requiresAuth: true },
    } as any,
    {} as any,
    depsFor(mockAuthStore)
  );

  expect(result).toBe(true);
  expect(loadPermissionState.calls.length).toBe(0);
});

test('permissionGuard bypasses /error/** routes to avoid redirect loop', async () => {
  const loadPermissionState = fnRecorder();
  const mockAuthStore = {
    isAuthenticated: false,
    loadPermissionState,
    permissionState: null,
    identity: { metadata: { activeCompanyId: 'c1', enabledCompanyIds: ['c1'] } },
  };

  const result = await permissionGuard(
    {
      path: '/error/403',
      fullPath: '/error/403?from=/auth/users',
      meta: { requiresAuth: true, resourceId: 'auth.route.user_list' },
    } as any,
    {} as any,
    depsFor(mockAuthStore)
  );

  expect(result).toBe(true);
  expect(loadPermissionState.calls.length).toBe(0);
});

test('permissionGuard bypasses public route when requiresAuth=false', async () => {
  const loadPermissionState = fnRecorder();
  const mockAuthStore = {
    isAuthenticated: true,
    loadPermissionState,
    permissionState: routesState([]),
    identity: { metadata: { activeCompanyId: 'c1', enabledCompanyIds: ['c1'] } },
  };

  const result = await permissionGuard(
    {
      path: '/login',
      fullPath: '/login',
      meta: { requiresAuth: false, resourceId: 'auth.route.login' },
    } as any,
    {} as any,
    depsFor(mockAuthStore)
  );

  expect(result).toBe(true);
  expect(loadPermissionState.calls.length).toBe(0);
});

test('permissionGuard delegates unauthenticated case to authGuard path', async () => {
  const loadPermissionState = fnRecorder();
  const mockAuthStore = {
    isAuthenticated: false,
    loadPermissionState,
    permissionState: routesState([]),
    identity: { metadata: { activeCompanyId: 'c1', enabledCompanyIds: ['c1'] } },
  };

  const result = await permissionGuard(
    {
      path: '/auth/users',
      fullPath: '/auth/users',
      meta: { requiresAuth: true, resourceId: 'auth.route.user_list' },
    } as any,
    {} as any,
    depsFor(mockAuthStore)
  );

  expect(result).toBe(true);
  expect(loadPermissionState.calls.length).toBe(0);
});

test('permissionGuard soft-lands denied /home to first allowed app route', async () => {
  const mockAuthStore = {
    isAuthenticated: true,
    loadPermissionState: fnRecorder(),
    permissionState: routesState(['auth.route.user_list']),
    identity: { metadata: { activeCompanyId: 'c1', enabledCompanyIds: ['c1'] } },
  };

  const result = await permissionGuard(
    {
      path: '/home',
      fullPath: '/home',
      meta: { requiresAuth: true, resourceId: 'web.route.home' },
    } as any,
    {} as any,
    depsFor(mockAuthStore)
  );

  expect(result).toEqual({ path: '/auth/users', replace: true });
});

test('permissionGuard soft-landing keeps deterministic order under same permission set', async () => {
  const mockAuthStore = {
    isAuthenticated: true,
    loadPermissionState: fnRecorder(),
    permissionState: routesState(['auth.route.user_create', 'auth.route.role_list', 'auth.route.token_list']),
    identity: { metadata: { activeCompanyId: 'c1', enabledCompanyIds: ['c1'] } },
  };

  const result = await permissionGuard(
    {
      path: '/',
      fullPath: '/',
      meta: { requiresAuth: true, resourceId: 'web.route.home' },
    } as any,
    {} as any,
    depsFor(mockAuthStore)
  );

  // role_list and token_list both routeSequence=10, so parent menu sequence decides (30 < 50).
  // user_create has routeSequence=30 and should never win over the two list pages.
  expect(result).toEqual({ path: '/auth/roles', replace: true });
});
