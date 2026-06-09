// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed } from 'vue';
import { useAuthStore } from '@/auth/web/stores/auth';
import { canRoute as canRouteRaw, canMenu as canMenuRaw, hasAction as hasActionRaw } from '@/auth/web/permission';

/**
 * Expose permission helpers bound to the current auth state.
 */
export function usePermission() {
  const authStore = useAuthStore();

  const ctx = computed(() => {
    const meta = (authStore.identity as any)?.metadata as any;
    return {
      activeCompanyId: meta?.activeCompanyId,
      enabledCompanyIds: meta?.enabledCompanyIds,
    };
  });

  /**
   * Check whether the current user can access a route resource.
   */
  const canRoute = (resourceId: string | undefined): boolean => canRouteRaw(resourceId, authStore.permissionState, ctx.value);

  /**
   * Check whether the current user can see a menu resource.
   */
  const canMenu = (resourceId: string | undefined): boolean => canMenuRaw(resourceId, authStore.permissionState, ctx.value);

  /**
   * Check whether the current user has an action resource.
   */
  const hasAction = (resourceId: string | undefined): boolean => hasActionRaw(resourceId, authStore.permissionState, ctx.value);

  return {
    ctx,
    permissionState: authStore.permissionState,
    canRoute,
    canMenu,
    hasAction,
  };
}
