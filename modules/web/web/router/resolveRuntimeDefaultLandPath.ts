// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Runtime wrapper around {@link resolveDefaultLandPath} for router redirects
 * and brand click. Reads the live menu tree when Pinia/menu injection is ready;
 * skips leaves whose matched route resource fails `canRoute` (menu grants and
 * route grants are separate buckets); falls back to Module Board when stores
 * are unavailable (early boot / unit mounts).
 */
import { useRouter } from 'vue-router';
import { useAuthStore } from '@/auth/web/stores/auth';
import { canRoute } from '@/auth/web/permission';
import { useMenuStore } from '../stores/menuStore';
import { resolveDefaultLandPath, MODULE_BOARD_PATH } from './resolveDefaultLandPath';

function buildRuntimeCanNavigate(): ((path: string) => boolean) | undefined {
  try {
    const auth = useAuthStore();
    const router = useRouter();
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
        return canRoute(resourceId, auth.permissionState, ctx);
      } catch {
        return true;
      }
    };
  } catch {
    return undefined;
  }
}

export function resolveRuntimeDefaultLandPath(): string {
  try {
    const menus = useMenuStore().getMenus?.() ?? [];
    return resolveDefaultLandPath({
      menus,
      canNavigate: buildRuntimeCanNavigate(),
    });
  } catch {
    return MODULE_BOARD_PATH;
  }
}
