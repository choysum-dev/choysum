// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { RouteLocationNormalized } from 'vue-router';
import { useAuthStore } from '../stores/auth';
import { canRoute } from '@/auth/web/permission';

/** Optional store injection for FE unit tests (default: Pinia useAuthStore). */
export type AuthGuardDeps = {
  getAuthStore?: () => ReturnType<typeof useAuthStore>;
};

function resolveAuthStore(deps?: AuthGuardDeps) {
  return (deps?.getAuthStore ?? useAuthStore)();
}

function isSetupRoute(to: RouteLocationNormalized): boolean {
  if (to.name === 'setup') return true;
  const resourceId = String((to.meta as { resourceId?: unknown } | null | undefined)?.resourceId ?? '').trim();
  if (resourceId === 'auth.route.setup') return true;
  const path = String(to.path || '').replace(/\/+$/, '') || '/';
  return path === '/setup' || path === '/web/setup';
}

function isLogoutOrErrorRoute(to: RouteLocationNormalized): boolean {
  if (to.name === 'logout') return true;
  const path = String(to.path || '');
  if (path === '/logout' || path === '/web/logout' || path.startsWith('/error/')) return true;
  const resourceId = String((to.meta as { resourceId?: unknown } | null | undefined)?.resourceId ?? '').trim();
  return resourceId === 'auth.route.logout';
}

/** Guest auth shell entries that participate in SetupPending ↔ Ready routing. */
function isSetupShellRoute(to: RouteLocationNormalized): boolean {
  if (isSetupRoute(to)) return true;
  if (to.name === 'login' || to.name === 'register') return true;
  const resourceId = String((to.meta as { resourceId?: unknown } | null | undefined)?.resourceId ?? '').trim();
  if (resourceId === 'auth.route.login' || resourceId === 'auth.route.register') return true;
  const path = (String(to.path || '').replace(/\/+$/, '') || '/');
  return (
    path === '/' ||
    path === '/login' ||
    path === '/register' ||
    path === '/web/login' ||
    path === '/web/register'
  );
}

/**
 * Route SetupPending guests to /setup and keep Ready guests off /setup.
 * Only guest shell routes are redirected; protected routes still go through
 * authGuard → /login, which then lands on /setup when pending.
 */
export async function setupGuard(
  to: RouteLocationNormalized,
  _from: RouteLocationNormalized,
  deps: AuthGuardDeps = {}
) {
  if (isLogoutOrErrorRoute(to) || !isSetupShellRoute(to)) {
    return true;
  }

  const authStore = resolveAuthStore(deps);
  if (typeof authStore.ensureSetupStateReady !== 'function') {
    return true;
  }

  let complete: boolean;
  try {
    complete = await authStore.ensureSetupStateReady();
  } catch {
    // Probe failures must not block the shell; host gate still protects Login.
    return true;
  }

  if (!complete) {
    if (!isSetupRoute(to)) {
      return { name: 'setup', replace: true };
    }
    return true;
  }

  if (isSetupRoute(to)) {
    return { name: 'login', replace: true };
  }
  return true;
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
  if (to.meta.requiresAuth === false) {
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
    // Default land uses resolveDefaultLandPath (menu DFS → Module Board). Denied
    // board / business routes go to Error — no /home soft-land special case.
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
