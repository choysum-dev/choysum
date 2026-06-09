// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi, beforeEach } from 'vitest';

let mockAuthStore: any;
const canRouteMock = vi.fn<(resourceId: string, permissionState: any, ctx: any) => boolean>(() => true);

vi.mock('../stores/auth', () => {
  return {
    useAuthStore: () => mockAuthStore,
  };
});

vi.mock('@/auth/web/permission', () => {
  return {
    canRoute: canRouteMock,
  };
});

describe('route guards', () => {
  beforeEach(() => {
    canRouteMock.mockReset();
    canRouteMock.mockReturnValue(true);
  });

  it('authGuard redirects unauthenticated users to login', async () => {
    mockAuthStore = {
      ensureAuthReady: vi.fn(async () => undefined),
      isAuthenticated: false,
    };

    const { authGuard } = await import('./guard');

    const result = await authGuard(
      {
        path: '/auth/users',
        fullPath: '/auth/users?page=1',
        meta: { requiresAuth: true },
      } as any,
      {} as any
    );

    expect(mockAuthStore.ensureAuthReady).toHaveBeenCalledTimes(1);
    expect(result).toEqual({
      path: '/login',
      query: { redirect: '/auth/users?page=1' },
      replace: true,
    });
  });

  it('permissionGuard redirects to 403 when resource is not allowed', async () => {
    mockAuthStore = {
      isAuthenticated: true,
      loadPermissionState: vi.fn(async () => undefined),
      permissionState: { byCompany: {} },
      identity: { metadata: { activeCompanyId: 'c1', enabledCompanyIds: ['c1'] } },
    };
    canRouteMock.mockReturnValue(false);

    const { permissionGuard } = await import('./guard');

    const result = await permissionGuard(
      {
        path: '/auth/users',
        fullPath: '/auth/users',
        meta: { requiresAuth: true, resourceId: 'auth.route.user_list' },
      } as any,
      {} as any
    );

    expect(mockAuthStore.loadPermissionState).toHaveBeenCalledWith(false);
    expect(canRouteMock).toHaveBeenCalledWith('auth.route.user_list', mockAuthStore.permissionState, { activeCompanyId: 'c1', enabledCompanyIds: ['c1'] });
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

  it('permissionGuard allows route when no resource id is declared', async () => {
    mockAuthStore = {
      isAuthenticated: true,
      loadPermissionState: vi.fn(async () => undefined),
      permissionState: { byCompany: {} },
      identity: { metadata: { activeCompanyId: 'c1', enabledCompanyIds: ['c1'] } },
    };

    const { permissionGuard } = await import('./guard');

    const result = await permissionGuard(
      {
        path: '/public/help',
        fullPath: '/public/help',
        meta: { requiresAuth: true },
      } as any,
      {} as any
    );

    expect(result).toBe(true);
    expect(canRouteMock).not.toHaveBeenCalled();
  });

  it('permissionGuard bypasses /error/** routes to avoid redirect loop', async () => {
    mockAuthStore = {
      isAuthenticated: false,
      loadPermissionState: vi.fn(async () => undefined),
      permissionState: null,
      identity: { metadata: { activeCompanyId: 'c1', enabledCompanyIds: ['c1'] } },
    };

    const { permissionGuard } = await import('./guard');

    const result = await permissionGuard(
      {
        path: '/error/403',
        fullPath: '/error/403?from=/auth/users',
        meta: { requiresAuth: true, resourceId: 'auth.route.user_list' },
      } as any,
      {} as any
    );

    expect(result).toBe(true);
    expect(mockAuthStore.loadPermissionState).not.toHaveBeenCalled();
    expect(canRouteMock).not.toHaveBeenCalled();
  });

  it('permissionGuard bypasses public route when requiresAuth=false', async () => {
    mockAuthStore = {
      isAuthenticated: true,
      loadPermissionState: vi.fn(async () => undefined),
      permissionState: { byCompany: {} },
      identity: { metadata: { activeCompanyId: 'c1', enabledCompanyIds: ['c1'] } },
    };

    const { permissionGuard } = await import('./guard');

    const result = await permissionGuard(
      {
        path: '/login',
        fullPath: '/login',
        meta: { requiresAuth: false, resourceId: 'auth.route.login' },
      } as any,
      {} as any
    );

    expect(result).toBe(true);
    expect(mockAuthStore.loadPermissionState).not.toHaveBeenCalled();
    expect(canRouteMock).not.toHaveBeenCalled();
  });

  it('permissionGuard delegates unauthenticated case to authGuard path', async () => {
    mockAuthStore = {
      isAuthenticated: false,
      loadPermissionState: vi.fn(async () => undefined),
      permissionState: { byCompany: {} },
      identity: { metadata: { activeCompanyId: 'c1', enabledCompanyIds: ['c1'] } },
    };

    const { permissionGuard } = await import('./guard');

    const result = await permissionGuard(
      {
        path: '/auth/users',
        fullPath: '/auth/users',
        meta: { requiresAuth: true, resourceId: 'auth.route.user_list' },
      } as any,
      {} as any
    );

    expect(result).toBe(true);
    expect(mockAuthStore.loadPermissionState).not.toHaveBeenCalled();
    expect(canRouteMock).not.toHaveBeenCalled();
  });

  it('permissionGuard soft-lands denied /home to first allowed app route', async () => {
    mockAuthStore = {
      isAuthenticated: true,
      loadPermissionState: vi.fn(async () => undefined),
      permissionState: { byCompany: {} },
      identity: { metadata: { activeCompanyId: 'c1', enabledCompanyIds: ['c1'] } },
    };

    canRouteMock.mockImplementation((resourceId: string) => resourceId === 'auth.route.user_list');

    const { permissionGuard } = await import('./guard');

    const result = await permissionGuard(
      {
        path: '/home',
        fullPath: '/home',
        meta: { requiresAuth: true, resourceId: 'web.route.home' },
      } as any,
      {} as any
    );

    expect(result).toEqual({ path: '/auth/users', replace: true });
  });

  it('permissionGuard soft-landing keeps deterministic order under same permission set', async () => {
    mockAuthStore = {
      isAuthenticated: true,
      loadPermissionState: vi.fn(async () => undefined),
      permissionState: { byCompany: {} },
      identity: { metadata: { activeCompanyId: 'c1', enabledCompanyIds: ['c1'] } },
    };

    canRouteMock.mockImplementation((resourceId: string) => {
      return resourceId === 'auth.route.user_create' || resourceId === 'auth.route.role_list' || resourceId === 'auth.route.token_list';
    });

    const { permissionGuard } = await import('./guard');

    const result = await permissionGuard(
      {
        path: '/',
        fullPath: '/',
        meta: { requiresAuth: true, resourceId: 'web.route.home' },
      } as any,
      {} as any
    );

    // role_list and token_list both routeSequence=10, so parent menu sequence decides (30 < 50).
    // user_create has routeSequence=30 and should never win over the two list pages.
    expect(result).toEqual({ path: '/auth/roles', replace: true });
  });
});
