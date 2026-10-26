// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { PermissionState } from '@/auth/web/permission';
import { asyncFnRecorder } from '@/web/web/__tests__/mountApp';
import { authGuard, permissionGuard, type AuthGuardDeps } from './guard';

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

test('authGuard allows authenticated users on public error routes', async () => {
  const ensureAuthReady = asyncFnRecorder();
  const mockAuthStore = {
    ensureAuthReady,
    isAuthenticated: true,
  };

  const result = await authGuard(
    {
      path: '/error/403',
      fullPath: '/error/403?from=/auth/users',
      meta: { requiresAuth: false },
    } as any,
    {} as any,
    depsFor(mockAuthStore)
  );

  expect(result).toBe(true);
  expect(ensureAuthReady.calls.length).toBe(0);
});

test('authGuard redirects unauthenticated users to login', async () => {
  const ensureAuthReady = asyncFnRecorder();
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
  const loadPermissionState = asyncFnRecorder();
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
  const loadPermissionState = asyncFnRecorder();
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
  const loadPermissionState = asyncFnRecorder();
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
  const loadPermissionState = asyncFnRecorder();
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
  const loadPermissionState = asyncFnRecorder();
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

test('permissionGuard sends denied routes to Error (no /home soft-land)', async () => {
  const mockAuthStore = {
    isAuthenticated: true,
    loadPermissionState: asyncFnRecorder(),
    permissionState: routesState(['auth.route.user_list']),
    identity: { metadata: { activeCompanyId: 'c1', enabledCompanyIds: ['c1'] } },
  };

  const result = await permissionGuard(
    {
      path: '/meta/modules',
      fullPath: '/meta/modules',
      meta: { requiresAuth: true, resourceId: 'meta.route.module_board' },
    } as any,
    {} as any,
    depsFor(mockAuthStore)
  );

  expect(result).toEqual({
    path: '/error/403',
    query: {
      reason: 'permission',
      message: 'PermissionDenied',
      from: '/meta/modules',
    },
    replace: true,
  });
});

test('permissionGuard denied create-only grant still goes to Error without soft-land', async () => {
  const mockAuthStore = {
    isAuthenticated: true,
    loadPermissionState: asyncFnRecorder(),
    permissionState: routesState(['auth.route.field_rule_create']),
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
