// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { RouteLocationNormalized, NavigationGuardNext } from 'vue-router';
import { useAuthStore } from '../stores/auth';
import { canRoute } from '@/auth/web/permission';
import { appRoutes } from './routes';
import { authMenus } from '../menu/menus';

const DEFAULT_SEQUENCE = Number.POSITIVE_INFINITY;

/**
 * Normalize a route path into an absolute application path.
 */
function normalizeRoutePath(path: unknown): string {
  const raw = String(path || '').trim();
  if (!raw) return '';
  return raw.startsWith('/') ? raw : `/${raw}`;
}

/**
 * Check whether a route path can be used as a navigation target.
 */
function isNavigablePath(path: string): boolean {
  if (!path) return false;
  if (path.startsWith('/error/')) return false;
  if (path.includes(':')) return false;
  return true;
}

/**
 * Normalize a route or menu sequence value.
 */
function asSequence(value: unknown): number {
  const n = Number(value);
  return Number.isFinite(n) ? n : DEFAULT_SEQUENCE;
}

type MenuPathEntry = {
  path: string;
  sequence: number;
};

/**
 * Build a flattened list of menu paths and their ordering metadata.
 */
function buildMenuPathEntries(): MenuPathEntry[] {
  const out: MenuPathEntry[] = [];

  const walk = (items: any[]) => {
    for (const item of items || []) {
      const path = normalizeRoutePath(item?.path);
      if (path && !String(item?.externalLink || '').trim()) {
        out.push({ path, sequence: asSequence(item?.order) });
      }
      const children = Array.isArray(item?.children) ? item.children : [];
      if (children.length > 0) walk(children);
    }
  };

  walk(authMenus as any[]);
  return out;
}

const menuPathEntries = buildMenuPathEntries();

/**
 * Check whether one route path is covered by a menu path entry.
 */
function matchesMenuPath(routePath: string, menuPath: string): boolean {
  if (!routePath || !menuPath) return false;
  if (routePath === menuPath) return true;
  if (!routePath.startsWith(menuPath)) return false;
  const next = routePath.charAt(menuPath.length);
  return next === '/' || next === '';
}

/**
 * Resolve the most specific menu ordering for a route path.
 */
function findMenuSequenceForRoutePath(routePath: string): number {
  let bestLength = -1;
  let bestSequence = DEFAULT_SEQUENCE;
  let ambiguous = false;

  for (const entry of menuPathEntries) {
    if (!matchesMenuPath(routePath, entry.path)) continue;
    const length = entry.path.length;
    if (length > bestLength) {
      bestLength = length;
      bestSequence = entry.sequence;
      ambiguous = false;
      continue;
    }
    if (length == bestLength && entry.sequence !== bestSequence) {
      ambiguous = true;
    }
  }

  if (ambiguous) return DEFAULT_SEQUENCE;
  return bestSequence;
}

/**
 * Pick the first route path that is allowed by the current permission snapshot.
 */
function pickFirstAllowedRoutePath(state: any, ctx: { activeCompanyId?: string; enabledCompanyIds?: string[] }): string {
  const candidates: Array<{
    path: string;
    resourceId: string;
    routeSequence: number;
    menuSequence: number;
  }> = [];

  for (const route of appRoutes) {
    const resourceId = String((route as any)?.meta?.resourceId || '').trim();
    const routePath = normalizeRoutePath((route as any)?.path);
    if (!resourceId || !isNavigablePath(routePath)) continue;
    if (!canRoute(resourceId, state, ctx)) continue;

    const routeSequence = asSequence((route as any)?.meta?.routeSequence ?? (route as any)?.meta?.sequence);
    const menuSequence = findMenuSequenceForRoutePath(routePath);

    candidates.push({
      path: routePath,
      resourceId,
      routeSequence,
      menuSequence,
    });
  }

  candidates.sort((a, b) => {
    if (a.routeSequence !== b.routeSequence) return a.routeSequence - b.routeSequence;
    if (a.menuSequence !== b.menuSequence) return a.menuSequence - b.menuSequence;
    const idCmp = a.resourceId.localeCompare(b.resourceId);
    if (idCmp !== 0) return idCmp;
    return a.path.localeCompare(b.path);
  });

  return candidates[0]?.path || '';
}

/**
 * Redirect unauthenticated users to the login page.
 */
export async function authGuard(to: RouteLocationNormalized, from: RouteLocationNormalized, next: NavigationGuardNext) {
  if (to.meta.requiresAuth === false || to.meta.isAuthPage) {
    return next();
  }

  const authStore = useAuthStore();

  // Ensure auth initialization, including refresh-token recovery, finishes before checking state.
  try {
    await authStore.ensureAuthReady();
  } catch (e) {
    // Initialization failures fall through to the unauthenticated redirect path.
  }

  if (!authStore.isAuthenticated) {
    return next({ path: '/login', query: { redirect: to.fullPath }, replace: true });
  }
  next();
}

/**
 * Redirect users to the permission error page when the route resource is not allowed.
 */
export async function permissionGuard(to: RouteLocationNormalized, from: RouteLocationNormalized, next: NavigationGuardNext) {
  // Error pages bypass the permission guard to avoid redirect loops.
  if (String(to.path || '').startsWith('/error/')) {
    return next();
  }

  const resourceId = String((to.meta as any)?.resourceId || '').trim();

  if (!resourceId || to.meta.requiresAuth === false) {
    return next();
  }

  const authStore = useAuthStore();

  // Let the auth guard handle unauthenticated navigation.
  if (!authStore.isAuthenticated) {
    return next();
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
      const fallbackPath = pickFirstAllowedRoutePath(authStore.permissionState, ctx);
      if (fallbackPath && fallbackPath !== to.path) {
        return next({ path: fallbackPath, replace: true });
      }
    }

    return next({
      path: '/error/403',
      query: {
        reason: 'permission',
        message: 'PermissionDenied',
        from: to.fullPath,
      },
      replace: true,
    });
  }

  next();
}
