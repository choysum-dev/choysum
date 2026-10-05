// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Runtime wrapper around {@link resolveDefaultLandPath} for router redirects
 * and brand click. Reads the live menu tree when Pinia/menu injection is ready;
 * skips leaves whose matched route resource fails `canRoute` (menu grants and
 * route grants are separate buckets); falls back to Module Board when stores
 * are unavailable (early boot / unit mounts).
 */
import { inject } from 'vue';
import { useRouter } from 'vue-router';
import { useAuthStore } from '@/auth/web/stores/auth';
import { canRoute } from '@/auth/web/permission';
import { MenuSymbol, getInstalledMenu, type MenuItem } from '@/core/web/menu';
import { useMenuStore } from '../stores/menuStore';
import { resolveDefaultLandPath, MODULE_BOARD_PATH } from './resolveDefaultLandPath';

type LandRouter = { resolve: (to: string) => { meta?: unknown } };

/**
 * Vue's `useRouter()` injects without throwing when there is no currentInstance
 * (sync Root/CatchAll redirect). A dangling `undefined.resolve` then fail-closes
 * every leaf and lands on Module Board.
 */
function resolveLandRouter(): LandRouter | undefined {
  try {
    const router = useRouter() as LandRouter | undefined;
    if (!router || typeof router.resolve !== 'function') return undefined;
    return router;
  } catch {
    return undefined;
  }
}

/** True when the snapshot has at least one route grant (including `*`). */
function hasRouteGrantSnapshot(state: unknown): boolean {
  if (!state || typeof state !== 'object') return false;
  const byCompany = (state as { byCompany?: unknown }).byCompany;
  if (!byCompany || typeof byCompany !== 'object') return false;
  for (const pack of Object.values(byCompany as Record<string, { ui?: { routes?: unknown } }>)) {
    const routes = pack?.ui?.routes;
    if (Array.isArray(routes) && routes.length > 0) return true;
  }
  return false;
}

function buildRuntimeCanNavigate(): ((path: string) => boolean) | undefined {
  try {
    const auth = useAuthStore();
    const snapshot = auth.permissionState;
    // Empty persist `{ permStateVersion: 0, byCompany: {} }` is truthy but
    // fail-closes canRoute; wait until a real grant set exists.
    if (!hasRouteGrantSnapshot(snapshot)) return undefined;
    const router = resolveLandRouter();
    if (!router) return undefined;
    const meta = (auth.identity as any)?.metadata as any;
    const ctx = {
      activeCompanyId: meta?.activeCompanyId,
      enabledCompanyIds: meta?.enabledCompanyIds,
    };
    return (path: string) => {
      try {
        const resolved = router.resolve(path);
        const resourceId = String((resolved.meta as any)?.resourceId || '').trim();
        // No resource id → treat as reachable (layout / public); authGuard still applies.
        if (!resourceId) return true;
        return canRoute(resourceId, snapshot, ctx);
      } catch {
        // Unverifiable path: skip so DFS can fall through to the next leaf / Module Board.
        return false;
      }
    };
  } catch {
    return undefined;
  }
}

/**
 * Land DFS uses declared visibility, not permission-projected `hidden`.
 * Root redirect is sync and often runs before the permission snapshot is ready.
 */
function withDeclaredVisibility(item: MenuItem): MenuItem {
  const meta = item.meta as { __permBaseHidden?: boolean } | undefined;
  const hidden = meta && meta.__permBaseHidden !== undefined ? !!meta.__permBaseHidden : !!item.hidden;
  return {
    ...item,
    hidden,
    children: item.children?.map(withDeclaredVisibility),
  };
}

/**
 * Reads the live menu tree without requiring a component inject context.
 */
function readRuntimeMenus(): MenuItem[] {
  try {
    const fromStore = useMenuStore().getMenus?.() ?? [];
    if (fromStore.length) return fromStore;
  } catch {
    // Store setup may throw before router/pinia are ready.
  }
  try {
    const injected = inject(MenuSymbol, null) as { getMenus?: () => MenuItem[] } | null;
    const fromInject = injected?.getMenus?.() ?? [];
    if (fromInject.length) return fromInject;
  } catch {
    // Router redirect callbacks have no currentInstance.
  }
  return getInstalledMenu()?.getMenus?.() ?? [];
}

export function resolveRuntimeDefaultLandPath(): string {
  try {
    const raw = readRuntimeMenus();
    let snapshotReady = false;
    try {
      snapshotReady = hasRouteGrantSnapshot(useAuthStore().permissionState);
    } catch {
      snapshotReady = false;
    }
    // Restore declared visibility only before a grant snapshot exists; afterwards
    // keep permission-projected `hidden` so land matches the sidebar.
    const menus = snapshotReady ? raw : raw.map(withDeclaredVisibility);
    return resolveDefaultLandPath({
      menus,
      canNavigate: buildRuntimeCanNavigate(),
    });
  } catch {
    return MODULE_BOARD_PATH;
  }
}
