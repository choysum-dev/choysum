// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { RouteLocationNormalized } from 'vue-router';
import { useAuthStore } from '../stores/auth';
import { canRoute } from '@/auth/web/permission';
import { pickFirstAllowedSoftLandPath } from './soft_land_nav';

/** Optional store injection for FE unit tests (default: Pinia useAuthStore). */
export type AuthGuardDeps = {
  getAuthStore?: () => ReturnType<typeof useAuthStore>;
};

function resolveAuthStore(deps?: AuthGuardDeps) {
  return (deps?.getAuthStore ?? useAuthStore)();
}

/**
 * Redirect unauthenticated users to the login page.
 */
export async function authGuard(
  to: RouteLocationNormalized,
  _from: RouteLocationNormalized,
  // Default keeps Function.length at 2 so a direct beforeEach(authGuard) registration
  // is not treated as a legacy next-callback guard by vue-router.
  deps: AuthGuardDeps = {}
) {
  if (to.meta.requiresAuth === false || to.meta.isAuthPage) {
    return true;
  }

  const authStore = resolveAuthStore(deps);

  // Ensure auth initialization, including refresh-token recovery, finishes before checking state.
  try {
    await authStore.ensureAuthReady();
  } catch (e) {
    // Initialization failures fall through to the unauthenticated redirect path.
  }

  if (!authStore.isAuthenticated) {
    return { path: '/login', query: { redirect: to.fullPath }, replace: true };
  }
  return true;
}

/**
 * Redirect users to the permission error page when the route resource is not allowed.
 */
export async function permissionGuard(
  to: RouteLocationNormalized,
  _from: RouteLocationNormalized,
  deps: AuthGuardDeps = {}
) {
  // Error pages bypass the permission guard to avoid redirect loops.
  if (String(to.path || '').startsWith('/error/')) {
    return true;
  }

  const resourceId = String((to.meta as any)?.resourceId || '').trim();

  if (!resourceId || to.meta.requiresAuth === false) {
    return true;
  }

  const authStore = resolveAuthStore(deps);

  // Let the auth guard handle unauthenticated navigation.
  if (!authStore.isAuthenticated) {
    return true;
  }

  try {
    // Refresh the client-side permission snapshot before evaluating the route.
    await authStore.loadPermissionState(false);
  } catch {
    // Keep fail-closed semantics when the permission snapshot cannot be refreshed.
  }

  const meta = (authStore.identity as any)?.metadata as any;
  const ctx = {
    activeCompanyId: meta?.activeCompanyId,
    enabledCompanyIds: meta?.enabledCompanyIds,
  };

  const ok = canRoute(resourceId, authStore.permissionState, ctx);
  if (!ok) {
    if (to.path === '/' || to.path === '/home') {
      const fallbackPath = pickFirstAllowedSoftLandPath(canRoute, authStore.permissionState, ctx);
      if (fallbackPath && fallbackPath !== to.path) {
        return { path: fallbackPath, replace: true };
      }
    }

    return {
      path: '/error/403',
      query: {
        reason: 'permission',
        message: 'PermissionDenied',
        from: to.fullPath,
      },
      replace: true,
    };
  }

  return true;
}
